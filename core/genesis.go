// Copyright 2014 The go-ionchain Authors
// This file is part of the go-ionchain library.
//
// The go-ionchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ionchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ionchain library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ionchain/ionchain-core/common"
	"github.com/ionchain/ionchain-core/common/hexutil"
	"github.com/ionchain/ionchain-core/common/math"
	"github.com/ionchain/ionchain-core/core/state"
	"github.com/ionchain/ionchain-core/core/types"
	"github.com/ionchain/ionchain-core/ioncdb"
	"github.com/ionchain/ionchain-core/log"
	"github.com/ionchain/ionchain-core/params"
	"github.com/ionchain/ionchain-core/rlp"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`

	// 新增字段
	BaseTarget          *big.Int       `json:baseTarget              gencodec:"required"` // baseTarget
	Coinbase            common.Address `json:"coinbase"`
	BlockSignature      []byte         `json:blockSignature          gencodec:"required"` // 区块签名信息
	GenerationSignature []byte         `json:generationSignature     gencodec:"required"` // 生成签名信息
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database already contains an incompatible genesis block (have %x, new %x)", e.Stored[:8], e.New[:8])
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db ioncdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}

	// Just commit the new block if there is no stored genesis block.
	stored := GetCanonicalHash(db, 0) // 从本地取出创世块
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		return genesis.Config, block.Hash(), err
	}

	// Check whether the genesis block is already written.
	if genesis != nil {
		block, _ := genesis.ToBlock()
		hash := block.Hash()
		if hash != stored {
			return genesis.Config, block.Hash(), &GenesisMismatchError{stored, hash}
		}
	}

	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	storedcfg, err := GetChainConfig(db, stored)
	if err != nil {
		if err == ErrChainConfigNotFound {
			// This case happens if a genesis write was interrupted.
			log.Warn("Found genesis block without chain config")
			err = WriteChainConfig(db, stored, newcfg)
		}
		return newcfg, stored, err
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}

	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := GetBlockNumber(db, GetHeadHeaderHash(db))
	if height == missingNumber {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, height)
	if compatErr != nil && height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	return newcfg, stored, WriteChainConfig(db, stored, newcfg)
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.TestnetGenesisHash:
		return params.TestnetChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock creates the block and state of a genesis specification.
func (g *Genesis) ToBlock() (*types.Block, *state.StateDB) {
	db, _ := ioncdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:              new(big.Int).SetUint64(g.Number),
		Time:                new(big.Int).SetUint64(g.Timestamp),
		ParentHash:          g.ParentHash,
		Extra:               g.ExtraData,
		GasLimit:            new(big.Int).SetUint64(g.GasLimit),
		GasUsed:             new(big.Int).SetUint64(g.GasUsed),
		Difficulty:          g.Difficulty,
		Coinbase:            g.Coinbase,
		Root:                root,
		BaseTarget:          g.BaseTarget,
		BlockSignature:      g.BlockSignature,
		GenerationSignature: g.GenerationSignature,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}
	return types.NewBlock(head, nil, nil,nil), statedb
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db ioncdb.Database) (*types.Block, error) {
	block, statedb := g.ToBlock()
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	if _, err := statedb.CommitTo(db, false); err != nil {
		return nil, fmt.Errorf("cannot write state: %v", err)
	}
	if err := WriteTd(db, block.Hash(), block.NumberU64(), g.Difficulty); err != nil {
		return nil, err
	}
	if err := WriteBlock(db, block); err != nil {
		return nil, err
	}
	if err := WriteBlockReceipts(db, block.Hash(), block.NumberU64(), nil); err != nil {
		return nil, err
	}
	if err := WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
		return nil, err
	}
	if err := WriteHeadBlockHash(db, block.Hash()); err != nil {
		return nil, err
	}
	if err := WriteHeadHeaderHash(db, block.Hash()); err != nil {
		return nil, err
	}
	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	return block, WriteChainConfig(db, block.Hash(), config)
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db ioncdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db ioncdb.Database, addr common.Address, balance *big.Int) *types.Block {
	g := Genesis{Alloc: GenesisAlloc{addr: {Balance: balance}}}
	g.BaseTarget = big.NewInt(153722867280912930)

	decodeByte, _ := hex.DecodeString("e3f22583ddb856060f8c54886420b1797f952975cda55156911369b7a557d1cf")
	g.GenerationSignature = decodeByte
	return g.MustCommit(db)
}

// DefaultGenesisBlock returns the ionchain main net genesis block.
func DefaultGenesisBlock() *Genesis {
	file := strings.NewReader(`
		{
		  "config": {
			"chainId": 1,
			"homesteadBlock": 0,
			"eip155Block": 0,
			"eip158Block": 0
		  },
		  "alloc": {
			"0xeb680f30715f347d4eb5cd03ac5eced297ac5046": {
			  "balance": "0x52b7d2dcc80cd2e4000000"
			},
			"0x0000000000000000000000000000000000000100": {
			  "code": "0x60806040526004361061008d576000357c010000000000000000000000000000000000000000000000000000000090048063be38ffd81161006b578063be38ffd814610167578063cbd8877e1461019a578063d0e30db0146101af578063e18128e9146101b75761008d565b806327e235e3146100925780632e1a7d4d146100d757806365476ea314610115575b600080fd5b34801561009e57600080fd5b506100c5600480360360208110156100b557600080fd5b5035600160a060020a03166101ea565b60408051918252519081900360200190f35b3480156100e357600080fd5b50610101600480360360208110156100fa57600080fd5b50356101fc565b604080519115158252519081900360200190f35b34801561012157600080fd5b5061014e6004803603604081101561013857600080fd5b50600160a060020a0381351690602001356106a1565b6040805192835260208301919091528051918290030190f35b34801561017357600080fd5b506100c56004803603602081101561018a57600080fd5b5035600160a060020a03166106dc565b3480156101a657600080fd5b506100c56106ee565b6101016106f4565b3480156101c357600080fd5b506100c5600480360360208110156101da57600080fd5b5035600160a060020a031661084d565b60016020526000908152604090205481565b3360009081526001602052604081205461021c908363ffffffff61094316565b33600090815260016020908152604080832093909355600290529081205483911015610323573360009081526002602052604090205481116102ef5733600090815260026020526040902054610278908263ffffffff61094316565b336000908152600260208181526040808420859055600382529283902083518085019094529181529282529181016102ae610955565b905281546001818101845560009384526020808520845160029485029091019081559381015193909101929092553383529052604081208190559050610323565b3360009081526002602052604090205461031090829063ffffffff61094316565b3360009081526002602052604081205590505b600061032d610955565b905060005b33600090815260036020526040902054811080156103505750600083115b156104f557600080543382526003602052604090912080546103989291908490811061037857fe5b90600052602060002090600202016001015461095990919063ffffffff16565b8211156104ed573360009081526003602052604090208054849190839081106103bd57fe5b600091825260209091206002909102015410610479573360009081526003602052604090208054610411918591849081106103f457fe5b60009182526020909120600290910201549063ffffffff61094316565b33600090815260036020526040902080548390811061042c57fe5b60009182526020808320600290920290910192909255338152600390915260408120805491945083918390811061045f57fe5b9060005260206000209060020201600101819055506104ed565b33600090815260036020526040902080546104b791908390811061049957fe5b6000918252602090912060029091020154849063ffffffff61094316565b336000908152600360205260409020805491945090829081106104d657fe5b600091825260208220600290910201818155600101555b600101610332565b5060005b33600090815260036020526040902054811080156105175750600083115b15610660576000805433825260036020526040909120805461053f9291908490811061037857fe5b82116106585733600090815260036020526040902080548491908390811061056357fe5b60009182526020909120600290910201541061060257336000908152600360205260409020805461059a918591849081106103f457fe5b3360009081526003602052604090208054839081106105b557fe5b6000918252602080832060029092029091019290925533815260039091526040812080549194508391839081106105e857fe5b906000526020600020906002020160010181905550610658565b336000908152600360205260409020805461062291908390811061049957fe5b3360009081526003602052604090208054919450908290811061064157fe5b600091825260208220600290910201818155600101555b6001016104f9565b50811561066957fe5b604051339085156108fc029086906000818181858888f19350505050158015610696573d6000803e3d6000fd5b506001949350505050565b6003602052816000526040600020818154811015156106bc57fe5b600091825260209091206002909102018054600190910154909250905082565b60026020526000908152604090205481565b60005481565b33600090815260016020526040812054610714903463ffffffff61095916565b33600090815260016020526040812091909155805b336000908152600360205260409020548110156107e95733600090815260036020526040902080548290811061075b57fe5b906000526020600020906002020160000154600014156107e15733600090815260036020526040902080543491908390811061079357fe5b60009182526020909120600290910201556107ac610955565b3360009081526003602052604090208054839081106107c757fe5b906000526020600020906002020160010181905550600191505b600101610729565b50801515610845573360009081526003602090815260409182902082518084019093523483529190810161081b610955565b90528154600181810184556000938452602093849020835160029093020191825592909101519101555b600191505090565b600160a060020a0381166000908152600260205260408120548110156108885750600160a060020a0381166000908152600260205260409020545b6000610892610955565b905060005b600160a060020a03841660009081526003602052604090205481101561093c5760008054600160a060020a03861682526003602052604090912080546108e39291908490811061037857fe5b82111561093457600160a060020a0384166000908152600360205260409020805461093191908390811061091357fe5b6000918252602090912060029091020154849063ffffffff61095916565b92505b600101610897565b5050919050565b60008282111561094f57fe5b50900390565b4390565b60008282018381101561096857fe5b939250505056fea165627a7a72305820ae5d51b2643e27587e3ceee1912f7df775ff568253ad138897d09cfffe8bae1f0029",
			  "storage": {
				"0x0000000000000000000000000000000000000000000000000000000000000000": "0x0a",
				"0x33d4e30ad2c3b9f507062560fe978acc29929f1ee5c2c33abe6d050171fd8c93": "0x152d02c7e14af6800000",
				"0xf0bc51b6429a737673d08c93b1250adb286af2441c7e8b05b63ae4d1c62f5309": "0x152d02c7e14af6800000",
				"0x1a651ba38e9ef28f337203b6d5855ab359f361c01ead47fa34af5a8ad411c8e5": "0x152d02c7e14af6800000",
				"0xe51a734a12431380f5a1925e3a000a14d63b1b295b70e5255071b0056c828b87": "0x152d02c7e14af6800000",
				"0x23e2f55fca9f62cfbb86338b12a9f0d98f14e64ac5ee21492e96926327f31019": "0x152d02c7e14af6800000",
				"0xeb453466d2384525758334977e4d724cf41dc7f2333a161d20e300e10c0f1911": "0x152d02c7e14af6800000"
			  },
			  "balance": "0xd9a7c07f349d4ac7640000"
			}
		  },
		  "coinbase": "0x0000000000000000000000000000000000000000",
		  "difficulty": "0x01",
		  "extraData": "0x777573686f756865",
		  "gasLimit": "0x096ae380",
		  "nonce": "0x0000000000000001",
		  "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		  "timestamp": "0x00",
		  "baseTarget": "0x01dd37f5698c",
		  "blockSignature": "0x00",
		  "generationSignature": "0x00"
		}
`)
	genesis := new(Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		fmt.Printf("invalid genesis file: %v", err)
	}
	return genesis
}

// DefaultTestnetGenesisBlock returns the Ropsten network genesis block.
func DefaultTestnetGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.TestnetChainConfig,
		ExtraData:  hexutil.MustDecode("0x3535353535353535353535353535353535353535353535353535353535353535"),
		GasLimit:   16777216,
		Difficulty: big.NewInt(1048576),
		Alloc:      decodePrealloc(testnetAllocData),
	}
}

// DefaultRinkebyGenesisBlock returns the Rinkeby network genesis block.
func DefaultRinkebyGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RinkebyChainConfig,
		Timestamp:  1492009146,
		ExtraData:  hexutil.MustDecode("0x52657370656374206d7920617574686f7269746168207e452e436172746d616e42eb768f2244c8811c63729a21a3569731535f067ffc57839b00206d1ad20c69a1981b489f772031b279182d99e65703f0076e4812653aab85fca0f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(rinkebyAllocData),
	}
}

// DeveloperGenesisBlock returns the 'ionc --dev' genesis block. Note, this must
// be seeded with the
func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique.Period = period

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, 65)...),
		GasLimit:   6283185,
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			faucet: {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func decodePrealloc(data string) GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
	}
	return ga
}
