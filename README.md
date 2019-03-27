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

### 以编程的方式与`IONC`节点交互

作为一个开发人员想通过自己的程序与`ionchain`网络进行交互，而不是通过`JavaScript console`的方式，为了满足这种需求，`ionc`有一个内置`JSON-RPC API`，这些API通过`HTTP`、`WebSockets`和`IPC`方式暴露出去。其中`IPC`端口是默认开启的，`HTTP`和`WS`端口需要手动开启，同时为了安全方面的考虑，这两个端口只会暴露部分API。

基于HTTP的JSON-RPC API 选项：

  * `--rpc` 开启 HTTP-RPC 服务
  * `--rpcaddr` HTTP-RPC 服务监听地址 (默认: "localhost")
  * `--rpcport` HTTP-RPC 服务监听端口 (默认: 8545)
  * `--rpcapi` 通过HTTP暴露出可用的API
  * `--rpccorsdomain` 逗号分隔的一系列域，通过这些域接收跨域请求

基于WebSocket的 JSON-RPC API选项:


  * `--ws` 开启 WS-RPC 服务
  * `--wsaddr` WS-RPC 服务监听地址(默认: "localhost")
  * `--wsport` WS-RPC 服务监听端口 (默认: 8546)
  * `--wsapi` 通过WS-PRC暴露出可用的API

基于IPC的JSON-RPC AP选项


  * `--ipcdisable` 禁用 IPC-RPC 服务
  * `--ipcapi` 通过IPC-PRC暴露出可用的API

**注意：在使用http/ws接口之前，你需要了解相关的安全知识，在公网上，黑客会利用节点在公网上暴露的接口进行破坏式的攻击**

### 创建一个私有链

创建一个自己的私有链会有一点复杂，因为你需要手动修改很多官方创建文件的配置。


#### 定义私有链创世块

首先，为你的私有网络创建一个创始状态，这个创始状态需要你的私有网络中的所有节点都知晓，并达成共识。`genesis.json`以JSON格式组成：

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

以上关于保证金合约是如何创建、编译的将在另外一个项目中做详细说明，我们建议你修改`nonce`值为一个随机数，这样可以防止未知的远程节点连接到你的网络中。如果你需要给某些账户预设一些资金，可以使用修改`alloc`值：
```json
"alloc": {
  "0x0000000000000000000000000000000000000001": {"balance": "111111111"},
  "0x0000000000000000000000000000000000000002": {"balance": "222222222"}
}
```
当`genesis.json`文件创建完成时，你需要在所有的`ionc`节点执行初始化操作。

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

