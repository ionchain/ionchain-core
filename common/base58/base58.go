// Copyright 2016 The go-ionchain Authors
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
package base58

import (
	"fmt"
	"math/big"
)

var bigRadix = big.NewInt(58)
var bigZero = big.NewInt(0)


// Decode decodes a modified base58 string to a byte slice.
func Decode(b string) ([]byte, error) {
	answer := big.NewInt(0)
	j := big.NewInt(1)

	scratch := new(big.Int)
	for i := len(b) - 1; i >= 0; i-- {
		tmp := b58[b[i]]
		if tmp == 255 {
			return nil, fmt.Errorf("invalid character `%v`", string(b[i]))
		}
		scratch.SetInt64(int64(tmp))
		scratch.Mul(j, scratch)
		answer.Add(answer, scratch)
		j.Mul(j, bigRadix)
	}

	tmpval := answer.Bytes()

	var numZeros int
	for numZeros = 0; numZeros < len(b); numZeros++ {
		if b[numZeros] != alphabetIdx0 {
			break
		}
	}
	flen := numZeros + len(tmpval)
	val := make([]byte, flen)
	copy(val[numZeros:], tmpval)

	return val, nil
}

// Encode encodes a byte slice to a modified base58 string.
func Encode(b []byte) string {
	x := new(big.Int)
	x.SetBytes(b)

	answer := make([]byte, 0, len(b)*136/100)
	for x.Cmp(bigZero) > 0 {
		mod := new(big.Int)
		x.DivMod(x, bigRadix, mod)
		answer = append(answer, alphabet[mod.Int64()])
	}

	// leading zero bytes
	for _, i := range b {
		if i != 0 {
			break
		}
		answer = append(answer, alphabetIdx0)
	}

	// reverse
	alen := len(answer)
	for i := 0; i < alen/2; i++ {
		answer[i], answer[alen-1-i] = answer[alen-1-i], answer[i]
	}

	return string(answer)
}
