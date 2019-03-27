##  ionchain-core


本项目是ionchain协议golang版本的实现


## 源码编译

在编译之前，你现需要安装golang（版本大于等于1.10）和`C`编译器

`clone`项目到你指定的目录中：

```
git clone https://github.com/ionchain/ionchain-core
```

使用以下命令编译`ionc`

```
cd ionchain-core

make ionc
```

或者可以通过以下命令编译其他平台的`ionc`版本（`Linux`，`windows`）

```
make all
```

### 在ionchain主网上运行全节点

用户在`ionchain`是最多的使用场景就是：创建账号，转移资产，部署、调用合约。为了满足这个特定的场景，可以使用快速同步方法来启动网络，执行以下命令:

```
$ ionc console
```

上面这个命令将产生以下两个操作:

 * 在快速同步模式下，启动`ionc`节点，在快速同步模式下，节点将会下载所有的状态数据，而不是执行所有`ionchain`网络上的所有交易.
 * 开启一个内置的`javascript console`控制台，在控制台中用户可以与`ionchain`网络进行交互。


#### 使用Docker快速启动节点

启动`ionchain`网络最快速的方式就是在本地启动一个`Docker`：

```
docker run -d --name ionchain-node -v /Users/alice/ionchain:/root \
           -p 8545:8545 -p 30303:30303 \
           ionchain/go-ionchain
```

`docker`会在`/Users/alice/ionchain`本地目录中映射一个持久的`volume`用来存储区块，同时也会映射默认端口。如果你想从其他容器或主机通过`RPC`方式访问运行的节点，需要加上`--rpcaddr 0.0.0.0`参数。默认情况下，`ionc`绑定的本地接口与`RPC`端点是不能从外部访问的。

### Programmatically interfacing Geth nodes

As a developer, sooner rather than later you'll want to start interacting with Geth and the Ethereum
network via your own programs and not manually through the console. To aid this, Geth has built-in
support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC) and
[Geth specific APIs](https://github.com/ethereum/go-ethereum/wiki/Management-APIs)). These can be
exposed via HTTP, WebSockets and IPC (UNIX sockets on UNIX based platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by Geth, whereas the HTTP
and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons.
These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpcport` HTTP-RPC server listening port (default: 8545)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: "localhost")
  * `--wsport` WS-RPC server listening port (default: 8546)
  * `--wsapi` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect
via HTTP, WS or IPC to a Geth node configured with the above flags and you'll need to speak [JSON-RPC](https://www.jsonrpc.org/specification)
on all transports. You can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based transport before
doing so! Hackers on the internet are actively trying to subvert Ethereum nodes with exposed APIs!
Further, all browser tabs can access locally running web servers, so malicious web pages could try to
subvert locally available APIs!**

### 创建一个私有链

Maintaining your own private network is more involved as a lot of configurations taken for granted in
the official networks need to be manually set up.

#### 定义私有链创世块

First, you'll need to create the genesis state of your networks, which all nodes need to be aware of
and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):

```json
{
		  "config": {
			"chainId":
		  },
		  "alloc": {},
			"0x0000000000000000000000000000000000000100": {
			  "code": "编译后的保证金合约二进制代码",
			  "storage": {
				"0x0000000000000000000000000000000000000000000000000000000000000000": "0x0a",
				"0x33d4e30ad2c3b9f507062560fe978acc29929f1ee5c2c33abe6d050171fd8c93": "0x0de0b6b3a7640000",
				"0xe0811e07d38b83ef44191e63c263ef79eeed21f1260fd00fef00a37495c1accc": "0xd9a7c07f349d4ac7640000"
			  },
			  "balance": ""
			}
		  },
		  "coinbase": "0x0000000000000000000000000000000000000000",
		  "difficulty": "0x01",
		  "extraData": "0x777573686f756865",
		  "gasLimit": "0x47e7c4",
		  "nonce": "0x0000000000000001",
		  "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		  "timestamp": "0x00",
		  "baseTarget": "0x1bc4fd6588",
		  "blockSignature": "0x00",
		  "generationSignature": "0x00"
		}
```

The above fields should be fine for most purposes, although we'd recommend changing the `nonce` to
some random value so you prevent unknown remote nodes from being able to connect to you. If you'd
like to pre-fund some accounts for easier testing, you can populate the `alloc` field with account
configs:

```json
"alloc": {
  "0x0000000000000000000000000000000000000001": {"balance": "111111111"},
  "0x0000000000000000000000000000000000000002": {"balance": "222222222"}
}
```

With the genesis state defined in the above JSON file, you'll need to initialize **every** Geth node
with it prior to starting it up to ensure all blockchain parameters are correctly set:

```
$ ionc init path/to/genesis.json
```

#### 创建bootnode节点

With all nodes that you want to run initialized to the desired genesis state, you'll need to start a
bootstrap node that others can use to find each other in your network and/or over the internet. The
clean way is to configure and run a dedicated bootnode:

```
$ bootnode --genkey=boot.key
$ bootnode --nodekey=boot.key
```

With the bootnode online, it will display an [`enode` URL](https://github.com/ethereum/wiki/wiki/enode-url-format)
that other nodes can use to connect to it and exchange peer information. Make sure to replace the
displayed IP address information (most probably `[::]`) with your externally accessible IP to get the
actual `enode` URL.

*Note: You could also use a full-fledged Geth node as a bootnode, but it's the less recommended way.*

#### 启动节点

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>` to ensure
it's indeed reachable), start every subsequent Geth node pointed to the bootnode for peer discovery
via the `--bootnodes` flag. It will probably also be desirable to keep the data directory of your
private network separated, so do also specify a custom `--datadir` flag.

```
$ ionc --datadir=path/to/custom/data/folder --bootnodes=<bootnode-enode-url-from-above>
```

*Note: Since your network will be completely cut off from the main and test networks, you'll also
need to configure a miner to process transactions and create new blocks for you.*

#### 运行私有链miner

Mining on the public Ethereum network is a complex task as it's only feasible using GPUs, requiring
an OpenCL or CUDA enabled `ethminer` instance. For information on such a setup, please consult the
[EtherMining subreddit](https://www.reddit.com/r/EtherMining/) and the [Genoil miner](https://github.com/Genoil/cpp-ethereum)
repository.

In a private network setting, however a single CPU miner instance is more than enough for practical
purposes as it can produce a stable stream of blocks at the correct intervals without needing heavy
resources (consider running on a single thread, no need for multiple ones either). To start a Geth
instance for mining, run it with all your usual flags, extended by:

```
$ ionc <usual-flags> --mine --minerthreads=1 --etherbase=0x0000000000000000000000000000000000000000
```

Which will start mining blocks and transactions on a single CPU thread, crediting all proceedings to
the account specified by `--etherbase`. You can further tune the mining by changing the default gas
limit blocks converge to (`--targetgaslimit`) and the price transactions are accepted at (`--gasprice`).

