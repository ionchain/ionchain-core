# JSON-RPC API
## Getting Started

[JSON](http://json.org/)是一种轻量级的数据交换格式。它可以表示数字，字符串，有序的值序列以及名称/值对的集合。

[JSON-RPC](http://www.jsonrpc.org/specification)是一种无状态，轻量级的远程过程调用（RPC）协议。这个规范主要定义了几个数据结构和围绕它们处理的规则。它是传输不可知的，因为这些概念可以在同一个进程中，在套接字上，在HTTP上，或在许多不同的消息传递环境中使用。它使用JSON（[RFC 4627](http://www.ietf.org/rfc/rfc4627.txt)）作为数据格式。

## JSON-RPC Support

* JSON-RPC 2.0
* 通讯方式：HTTP、IPC

## JSON-RPC API 示例环境

API说明中为每个API的使用列举了示例，这些示例可以使用Firebug、[postman](https://www.getpostman.com/apps)、curl等http接口测试工具，将Request内容POST到ionc客户端JSON-RPC监听端口，Result内容即为JSON-RPC API执行结果的返回。

支持单条请求和批量请求处理。
单条请求指请求中只有一个method方法。单条请求示例：

```json
//Request
{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":73}
```
```json
//Result
{
  "id":73,
  "jsonrpc": "2.0",
  "result": "0x430e23400"
}
```

批量请求可以将多个method方法打包，在一次请求中发送。批量请求示例：

```json
//Request
[{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5", "latest"],"id":1}, {"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":2}]
```
```json
//Result
[
    {
        "jsonrpc": "2.0",
        "id": 1,
        "result": "0x50e6f3a0"
    },
    {
        "jsonrpc": "2.0",
        "id": 2,
        "result": "0x29f9dd48c"
    }
]
```

## JSON-RPC API Reference

<!-- toc -->

### admin

#### admin_peers
获取记账节点信息列表

* Parameters    
  * 没有
* Returns
  * `Array` - 节点信息
* Example

```json
//Request
{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":73}
```
```json
//Result
{
  "id":73,
  "jsonrpc": "2.0",
  "result": []
}
```


#### admin_addPeer
添加新的远程节点

* Parameters    
  * url  节点的enode加上IP

```json
{"params":["enode://a979fb575495b8d6db44f75017d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303"]}
```

* Returns    
  * `boolean`：true为添加成功，false为添加失败
* Example

```json
 // Request
 {"jsonrpc":"2.0","method":"admin_addPeer","params":["enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb496de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303"],"id":"67"}
```
```json
 // Result
 {
     "jsonrpc": "2.0",
     "id": 67,
     "result": true
 }

```

#### admin_datadir
获取当前节点用于存储数据库的绝对路径

* Parameters    
  * 没有
* Returns     
  * 存储数据库的绝对路径

* Example
*
```json
//Request
{"jsonrpc":"2.0","method":"admin_datadir","params":[],"id":73}
```
```json
 //Result
{
    "jsonrpc": "2.0",
    "id": 73,
    "result": "/home/ionc/db"
}
```


#### admin_nodeInfo
获取节点信息

* Parameters    
  * 没有
* Returns      
  * 节点信息
* Example

```json
//Request
{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":67}
```
```json
//Result
{
    "jsonrpc": "2.0",
    "id": 67,
    "result": {
        "id": "d94b7b8196930c2d1151f4d6ed9d5514838bdd162203c54e4ac6e7e02db26ce4825f32507f839f96e1937f5752fadbaadc4d138bc017317a2a06780572790b75",
        "name": "ionc/v1.0.0-stable-74064c91/linux-amd64/go1.12",
        "enode": "enode://d94b7b8196930c2d1151f4d6ed9d5514838bdd162203c54e4ac6e7e02db26ce4825f32507f839f96e1937f572fadbaadc4d138bc017317a2a06780572790b75@[::]:30303",
        "ip": "::",
        "ports": {
            "discovery": 30303,
            "listener": 30303
        },
        "listenAddr": "[::]:30303",
        "protocols": {
            "ionc": {
                "difficulty": 1,
                "genesis": "0x17c084a3ccda852443baecbd282ed63eb4489e1c73117edb1a5b85c43aa5457b",
                "head": "0x17c084a3ccda852443baecbd282ed63eb4489e1c73117edb1a5b85c43aa5457b"
            }
        }
    }
}
```

#### admin_startRPC
启动HTTP RPC服务

* Parameters  
  * `host`：主机
  * `port`：端口
  * `cors`：跨资源共享
  * `apis`：apis
  * `vhosts`：虚拟主机
* Returns  
  * `Boolean`：true为启动成功，false为启动失败
* Example

 ```json
// Request
{"jsonrpc":"2.0","method":"admin_startRPC","params":["localhost",8545,"","",""],"id":67}
```
```json
 // Result
 {
     "jsonrpc": "2.0",
     "id": 67,
     "result": true
 }
 ```

#### admin_stopRPC
关闭当前打开的HTTP RPC端点

* Parameters  
  * 没有
* Returns  
  * `Boolean`：true为关闭成功，false为关闭失败
* Example

 ```json
// Request
{"jsonrpc":"2.0","method":"admin_stopRPC","params":[],"id":67}
```
```json
// Result
{
     "jsonrpc": "2.0",
     "id": 67,
     "result": true
}
```

#### admin_startWS
启动WebSocket RPC服务

* Parameters  
  * `host`：主机
  * `port`：端口
  * `cors`：跨资源共享
  * `apis`：apis
* Returns  
  * `Boolean`：true为启动成功，false为启动失败
* Example

```json
// Request
{"jsonrpc":"2.0","method":"admin_startWS","params":["localhost",8546,"",""],"id":67}
```
```json
// Result
{
     "jsonrpc": "2.0",
     "id": 67,
     "result": true
}
```

#### admin_stopWS
关闭WebSocket RPC服务

* Parameters    
  * 没有
* Returns   
  * `Boolean`：true为关闭成功，false为关闭失败

```json
// Request
{"jsonrpc":"2.0","method":"admin_stopWS","params":[],"id":67}
```
```json
// Result
{
  "jsonrpc":"2.0",
  "id":67,
  "result":true
}
```   

### eth

#### eth_getBalance
返回给定地址的帐户的余额

* Parameters  
  * `DATA`- 账户地址
  * `QUANTITY|TAG`- 整数块号或字符串"latest"，"earliest"或者"pending"

```json
{"params": [
   "0x8bae247e1c2543454585d627ae4c6d67543602b5",
   "latest"
  ]}
```

* Returns  
  * `QUANTITY` - u中当前余额的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5", "latest"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x20000000000"
}
```

#### eth_getStorageAt
返回给定地址的存储位置的值

* Parameters  
  * `DATA`，20字节 - 地址
  * `QUANTITY` - 存储位置的整数
  * `QUANTITY|TAG` - 整数块号或字符串"latest"，"earliest"或者"pending"

```json
{"params": [
   "0x8bae247e1c2543454585d627ae4c6d67543602b5",
   "0x0", // storage position at 0
   "0x62" // state at block number 2
]}
```

* Returns  
  * `DATA` - 这个存储位置的值
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5", "0x0", "0x62"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x000000000000000000000000000000000000262e"
}
```

#### eth_blockNumber
返回最近的块的高度

* Parameters  
  * 没有
* Returns  
  * `QUANTITY` - 客户端所在的当前块号的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":83}
```
```json
// Result
{
  "id":83,
  "jsonrpc": "2.0",
  "result": "0x64" // 100
}
```

#### eth_getBlockByNumber
通过块编号返回有关块的信息

* Parameters  
  * `QUANTITY|TAG` - 块号的整数，或字符串"earliest"，"latest"或者"pending"如默认块参数中所示
  * `Boolean` - 如果true它返回完整的事务对象，则false只有事务的哈希值

```json
{"params": [
   "0x64", // 100
   true
]}
```

* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x64", true],"id":1}
```
```json
// Result
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "baseTarget": "0x2b487aaef8",
        "blockSignature": "0xac609458dc050956152a64012bd8323c186da23241bdfd263fade654405173a020be1376d660a6de96f1098d8f17ddb56e3224a086df72718d9cc5ccad16970d01",
        "difficulty": "0x17a87ea6",
        "extraData": "0xd68301000084696f6e6386676f312e3132856c696e7578",
        "gasLimit": "0x96ae380",
        "gasUsed": "0x0",
        "generationSignature": "0x6dd52fe2a521981b882ce89aa4255310f0f907ec8a094a949d4344dc08d28a32",
        "hash": "0xdbc06e7e21eed04831e8630278d354b4ad75d242e4920f9fec0af4499c7949be",
        "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "miner": "0x6d3056c5aaf3180f94ee44ec23bbc53a9fa59f66",
        "number": "0x64",
        "parentHash": "0xb69f610c72a3f0ce7a18951cf74eeb734744054376284b78069f5ec2da3f5c21",
        "receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
        "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
        "size": "0x258",
        "stateRoot": "0x4bc7eaca524a9ec1a1ab5ec51af92b1f2814f77b0f0dc987bfc087c58c2b27c0",
        "timestamp": "0x5d0a4e7a",
        "totalDifficulty": "0x16a053a830",
        "transactions": [],
        "transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
        "uncles": []
    }
}

```

#### eth_getCoinbase
返回客户的coinbase地址(奖励地址)
* Parameters  
  * 没有
* Returns  
  * `DATA` - 当前coinbase地址
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getCoinbase","params":[],"id":64}
```
```json
// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "xxx"
}
```


#### eth_getBlockByHash
通过散列返回有关块的信息

* Parameters  
  * `DATA`，32字节 - 块的散列
  * `Boolean` - 如果true它返回完整的事务对象，则false只有事务的哈希值

```json
{"params": [
   "0xdbc06e7e21eed04831e8630278d354b4ad75d242e4920f9fec0af4499c7949be",
   true
]}
```

* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0xdbc06e7e21eed04831e8630278d354b4ad75d242e4920f9fec0af4499c7949be", true],"id":1}
```
```json
// Result
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "baseTarget": "0x2b487aaef8",
        "blockSignature": "0xac609458dc050956152a64012bd8323c186da23241bdfd263fade654405173a020be1376d660a6de96f1098d8f17ddb56e3224a086df72718d9cc5ccad16970d01",
        "difficulty": "0x17a87ea6",
        "extraData": "0xd68301000084696f6e6386676f312e3132856c696e7578",
        "gasLimit": "0x96ae380",
        "gasUsed": "0x0",
        "generationSignature": "0x6dd52fe2a521981b882ce89aa4255310f0f907ec8a094a949d4344dc08d28a32",
        "hash": "0xdbc06e7e21eed04831e8630278d354b4ad75d242e4920f9fec0af4499c7949be",
        "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "miner": "0x6d3056c5aaf3180f94ee44ec23bbc53a9fa59f66",
        "number": "0x64",
        "parentHash": "0xb69f610c72a3f0ce7a18951cf74eeb734744054376284b78069f5ec2da3f5c21",
        "receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
        "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
        "size": "0x258",
        "stateRoot": "0x4bc7eaca524a9ec1a1ab5ec51af92b1f2814f77b0f0dc987bfc087c58c2b27c0",
        "timestamp": "0x5d0a4e7a",
        "totalDifficulty": "0x16a053a830",
        "transactions": [],
        "transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
        "uncles": []
    }
}
```

#### eth_call
立即执行新的消息调用，而不在块链上创建事务
* Parameters  
  * `Object` - 事务调用对象
    * `from`：DATA - （可选）交易的发送地址
    * `to`：DATA - 事务处理的地址
    * `gas`：QUANTITY - （可选）交易消耗燃料数量，eth_call消耗零gas，但这个参数可能需要一些执行
    * `gasPrice`：QUANTITY - （可选）交易消耗单位gas价格
    * `value`：QUANTITY - （可选）发送的值与此事务的整数
    * `data`：DATA - （可选）合同的编译代码
  * `QUANTITY|TAG`- 整数块号或字符串"latest"，"earliest"或者"pending"
* Returns  
  * `DATA` - 执行合同的回报价值
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_call","params":[{"from":"0x8bae247e1c2543454585d627ae4c6d67543602b5","to":"0x31bf7b9f55f155f4ae512e30ac65c590dfad0ca6","gas":"0x76c0","gasPrice":"0x9184e72a000","value":"0x9184e72a","data":"0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}, "latest"],"id":67}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x1d055281899007cbe6865a48d0a79239dac8e486"
}
```

#### eth_estimateGas
生成并返回交易完成所需的Gas估算值
* Parameters  
  请参阅eth_call参数，期望所有属性都是可选的
* Returns  
  * `QUANTITY` - 使用的gas量
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{"from":"0x8bae247e1c2543454585d627ae4c6d67543602b5","to":"0x31bf7b9f55f155f4ae512e30ac65c590dfad0ca6","gas":"0x76c0","gasPrice":"0x9184e72a000","value":"0x9184e72a","data":"0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x4000"
}
```

#### eth_getBlockTransactionCountByNumber
从与给定块号匹配的块中返回块中的事务数
* Parameters  
  * `QUANTITY|TAG`- 块号的整数，或字符串"earliest"，"latest"或者"pending"如默认块参数中所示

```json
{"params": [
   "0x17"
]}
```

* Returns  
  * `QUANTITY` - 此区块中的交易数量的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByNumber","params":["0x17"],"id":1}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":1,
    "result":"0x20"
}
```

#### eth_getBlockTransactionCountByHash
从匹配给定块散列的块中返回块中的事务数
* Parameters  
  * `DATA`，32字节 - 块的散列

```json
{"params": [
   "0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"
]}
```

* Returns  
  * `QUANTITY` - 此区块中的交易数量的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByHash","params":["0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x20"
}
```

#### eth_getTransactionByBlockNumberAndIndex
通过块号和交易指标位置返回有关交易的信息
* Parameters  
  * `QUANTITY|TAG`- 块号或字符串"earliest"，"latest"或者"pending"如默认块参数中所示
  * `QUANTITY` - 交易指标位置

```json
{"params": [
   "0x62",
   "0x0"
]}
```
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getTransactionByBlockNumberAndIndex","params":["0x62", "0x0"],"id":1}
```
```json
// Result
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "blockHash": "0xa395f8d6cae3aba2f99086aabaa24a22b3bed0df3b846eda8fff258239fcc101",
        "blockNumber": "0x1746e3",
        "from": "0x8bae247e1c2543454585d627ae4c6d67543602b5",
        "gas": "0x61a80",
        "gasPrice": "0x4b91f08c",
        "hash": "0x08ab8a4662875c81336dfe66d40cb857fae8478a0ac7ff334d81186d668a1b6b",
        "input": "0x",
        "nonce": "0x5",
        "to": "0x31bf7b9f55f155f4ae512e30ac65c590dfad0ca6",
        "transactionIndex": "0x0",
        "value": "0xde0b6b3a7640000",
        "v": "0x1b",
        "r": "0xa5f27fdd1fe7df8d6bfc9a0a4a720171dbdd33438ed4e87877df3c0a97c6e733",
        "s": "0x15920b5840af35fbd4dc810622cbd78fe911ff77bda19c761b2f9849fff33839"
    }
}
```

#### eth_getTransactionByBlockHashAndIndex
通过块散列和事务索引位置返回有关事务的信息
* Parameters  
  * `DATA1`，32字节 - 块的散列
  * `QUANTITY` - 交易指标头寸的整数

```json
{"params": [
   "0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c",
   "0x0" // 0
]}
```

* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getTransactionByBlockHashAndIndex","params":["0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c", "0x0"],"id":1}
```

#### eth_getTransactionCount
返回从地址发送的交易数
* Parameters  
  * `DATA`，20字节 - 地址
  * `QUANTITY|TAG`- 整数块号或字符串"latest"，"earliest"或者"pending"，请参阅默认块参数

```json
{"params": [
   "0x8bae247e1c2543454585d627ae4c6d67543602b5",
   "latest" // state at the latest block
]}
```

* Returns  
  * `QUANTITY` - 从这个地址发送的交易数量的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5","latest"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x4d0"
}
```

#### eth_getTransactionByHash
根据交易hash查询交易信息
* Parameters  
  * `DATA` 32 Bytes - 交易hash

```json
{"params": [
   "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
]}
```

* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getTransactionByHash","params":["0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"],"id":1}
```
```json
// Result
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "blockHash": "0xa395f8d6cae3aba2f99086aabaa24a22b3bed0df3b846eda8fff258239fcc101",
        "blockNumber": "0x1746e3",
        "from": "0x8bae247e1c2543454585d627ae4c6d67543602b5",
        "gas": "0x61a80",
        "gasPrice": "0x4b91f08c",
        "hash": "0x08ab8a4662875c81336dfe66d40cb857fae8478a0ac7ff334d81186d668a1b6b",
        "input": "0x",
        "nonce": "0x5",
        "to": "0x31bf7b9f55f155f4ae512e30ac65c590dfad0ca6",
        "transactionIndex": "0x0",
        "value": "0xde0b6b3a7640000",
        "v": "0x1b",
        "r": "0xa5f27fdd1fe7df8d6bfc9a0a4a720171dbdd33438ed4e87877df3c0a97c6e733",
        "s": "0x15920b5840af35fbd4dc810622cbd78fe911ff77bda19c761b2f9849fff33839"
    }
}
```
```json
// Result(pending)
{
    "jsonrpc":"2.0",
    "id":1,
    "result":{
        "blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
        "blockNumber":null,
        "transactionIndex":"0x0",
        "hash":"0x7db7883bb23a31deb9f01b5e6fb28363b1aee1b9b6797ea8b5706be170a1187c",
        "from":"PUYWgYXiMWARdS1cQ4pHQTj3jGs6yCTui1",
        "accountNonce":"0x4b0",
        "to":"PGidCVbYGnFsm188Wyq5ywGxCF3vKJvCU3",
        "gasPrice":"0x200b20",
        "gasLimit":"0x1b7740",
        "amount":"0x15f290be2080369",
        "payload":"0x614144",
        "expire":"0x5b557935",
        "extra":"0x41424346",
        "signature":"0x394188274b23883843993923982392839283484738347bc766887878866667888a888789328387323322232328498943849384944879434940304030483849394384"
    }
}

```

#### eth_getRawTransactionByHash
通过给定的hash查询指定的交易
* Parameters  
  * `DATA1`，32字节 - 块的散列

```json
{"params": [
   "0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"
]}
```

* Returns  
  * `DATA` - 交易信息，RLP编码

* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getRawTransactionByHash","params":["0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0xa7878c767b327d73483734787284738b9892320990490344738648654658746a54"
}
```


#### eth_getTransactionReceipt
根据交易hash查询交易收据
* Parameters  
  * `DATA`，32 Bytes - 交易hash

```json
{"params": [
  "0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"
]}
```

* Returns  
Object - 交易收据对象, 或者没有为null:
    * `transactionIndex`：QUANTITY- 整数交易指标头寸日志是从中创建的；null当其挂起的日志
    * `transactionHash`：DATA，32字节 - 创建此日志的事务的散列值；null当其挂起的日志
    * `blockHash`：DATA，32字节 - 这个日志所在的块的哈希null；null当其挂起的日志
    * `blockNumber`：QUANTITY- null当其挂起时，该日志所在的块号；null当其挂起的日志
    * `from`：DATA - 交易的发送地址
    * `to`：DATA - （创建新合同时可选）交易指向的地址
    * `cumulativeGasUsed`: QUANTITY - 交易使用的gas总量.
    * `gasUsed`: QUANTITY - 交易的使用的gas数量.
    * `contractAddress`: DATA, 20 Bytes - 如果是合约创建的，此字段是合约地址, 否则是null.
    * `logsBloom`: DATA, 256 Bytes - Bloom filter
可能存在的字段：
    * `logs`: Array - 交易日志对象数组.
    * `root`: 所有stateObject对象的RLP Hash值
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0xcd22242e195acf9a3677ceb68472c807f80c78931770283ab40bd728b219236c"],"id":1}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":1,
    "result":{
        "blockHash":"0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b",
        "blockNumber":"0x451",
        "contractAddress":"0xb60e8dd61c5d32be8058bb8eb970870f07233155",
        "cumulativeGasUsed":"0x33bc",
        "from":"0xdae19174969a7404e222c24b6726e4d089c12768",
        "gasUsed":"0x4dc",
        "logs":[],
        "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "to":"0x5929a871f57a1c5f7e4ea304cae92dacd1c1556b",
        "transactionHash":"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
        "transactionIndex":"0x1"
    }
}
```

#### eth_sendRawTransaction
创建新的消息调用事务或为签名的事务创建合同
* Parameters  
  * Object - 交易Transaction对象的 RLP 编码

```json
{"params": ["0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"]}
```

* Returns  
  * `DATA` 32字节 - 交易散列，或者如果交易不可用，则为零散列
* Example

```json
// Request
{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"],"id":1}
```
```json
// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x67d5ab9277df504a436b1068697a444d30228584094632f10ab7ba5213a4eccc"
}
```

### txpool

#### txpool_content
列出当前等待下一个块中包含的所有交易的确切细节，以及计划在未来执行的交易
* Parameters  
  * 没有

* Returns  
  * `Array` - 交易细节信息
* Example

```json
// Request
{"jsonrpc":"2.0","method":"txpool_content","params":[],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":{
        "pending":{

        },
        "queued":{

        }
    }
}
```

#### txpool_status
查询当前等待下一个块中包含的交易数量，以及计划在未来执行的交易数量
* Parameters  
  * 没有

* Returns  
  * `number` - 交易数量
* Example

```json
// Request
{"jsonrpc":"2.0","method":"txpool_status","params":[],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":{
        "pending":"0x0",
        "queued":"0x0"
    }
}
```

#### txpool_inspect
可以查询以列出当前等待下一个块中包含的所有交易的文本摘要，以及计划在未来执行的交易的文本摘要
* Parameters  
  * 没有

* Returns  

* Example

```json
// Request
{"jsonrpc":"2.0","method":"txpool_inspect","params":[],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":{
        "pending":{},
        "queued":{}
    }
}
```


### personal

#### personal_listWallets
返回钱包列表

* Parameters  
  * 没有
* Returns  
  * `Array` - 钱包信息列表  
  Object信息如下：   
    * `QUANTITY` - 钱包索引  
    * `DATA` - 钱包信息  

* Example

```json
// Request
{"jsonrpc":"2.0","method":"personal_listWallets","params":[],"id":67}
```
```json
// Result
{
  "jsonrpc": "2.0",
  "id": 67,
  "result": {
    "0x0": {
      "scryptN": 262144,
      "scryptP": 1,
      "store": "",
      "version": "1.0"
    }
  }
}
```

#### personal_listAccounts
返回所有钱包账户地址列表

* Parameters  
  * 没有
* Returns  
  * `Array` - 客户账户地址列表
* Example

```json
// Request
{"jsonrpc":"2.0","method":"personal_listAccounts","params":[],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":[
        "0x8bae247e1c2543454585d627ae4c6d67543602b5",
    ]
}
```


#### personal_newAccount
创建默认钱包新用户

* Parameters  
  * `DATA` - 创建帐户时设置的账户密码   
  * `DATA` - 创建帐户时设置的安全密码

* Returns  
  * `DATA`- 创建新账户经Base58编码后的地址  

* Example  

```json
// Request
{"jsonrpc":"2.0","method":"personal_newAccount","params":["123456", "987654321"],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":"0x8bae247e1c2543454585d627ae4c6d67543602b5"
}
```

#### personal_unlockAccount
根据地址解锁钱包账户，若两个或两个以上钱包账户密码相同、地址相同时都解锁成功

* Parameters  
  * `DATA`- 账户地址
  * `passphrase`- 创建帐户时设置的账户密码
  * `duration`- 解锁时间
* Returns  
  * `DATA`- 处理结果，true为成功，false为失败
* Example

```json
// Request
{"jsonrpc":"2.0","method":"personal_unlockAccount","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5","123456", 0],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":true
}
```

#### personal_lockAccount
根据地址锁定钱包账户，若两个或两个以上钱包账户密码相同、地址相同时都锁定成功

* Parameters  
  * `DATA`- 账户地址

* Returns  
  * `DATA`- 处理结果，true为成功，false为失败

* Example

```json
// Request
{"jsonrpc":"2.0","method":"personal_lockAccount","params":["0x8bae247e1c2543454585d627ae4c6d67543602b5"],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":true
}
```

#### personal_sign
签名交易数据（仅需要账户密码）

* Parameters  
  注意：账户签名key的MAC 和 账户签名key的类型 至少有一不为空
  * `DATA`- 要签名数据
  * `DATA`- 账户地址
  * `passphrase`- 创建帐户时设置的账户密码
* Returns  
  * `DATA`- 签名后的数据
* Example  

```json
// Request
{"jsonrpc":"2.0","method":"personal_sign","params":["0x424344","0x6a0dF9E94a41fCd89d8236a8C03f9D678df5Acf9", "123456"],"id":67}
```
```json
// Result
{
    "jsonrpc":"2.0",
    "id":67,
    "result":"0x20ef64deef8b0b0108424bece022e264cd855d0f91d7e0e01dc07f7eea62e5da657ab02c51712405d6cdbb529b64f86e6ef63b314feeff8f704f3278107762281c"
}

```
### miner

#### miner_getHashrate
用来读取当前挖矿节点的每秒钟哈希值算出数量
* Parameters  
  * 无
```json
{"params": []}
```

* Returns  
  * `uint64`
* Example

```json
// Request
{"jsonrpc":"2.0", "method":"miner_getHashratee", "params":[],"id":73}
```
```json
// Result
{
  "id":73,
  "jsonrpc":"2.0",
  "result": 32
}
```


#### miner_start
启动挖矿
* Parameters  
  * `int`  启动线程数
```json
{"params": [1]}
```

* Returns  
  * `error` 正常启动无任何信息返回
* Example

```json
// Request
{"jsonrpc":"2.0", "method":"miner_start", "params":[1],"id":73}
```
```json
// Result
{
  "id":73,
  "jsonrpc":"2.0",
  "result": []
}
```


#### miner_stop
停止挖矿
* Parameters  
  * 无
```json
{"params": []}
```

* Returns  
  * `boolean` true表示停挖矿止成功 false表示停止挖矿失败
* Example

```json
// Request
{"jsonrpc":"2.0", "method":"miner_stop", "params":[],"id":73}
```
```json
// Result
{
  "id":73,
  "jsonrpc":"2.0",
  "result": true
}
```

### net

#### net_peerCount
返回当前连接到客户端的对端的数量
* Parameters  
  * 没有
* Returns  
  * `QUANTITY` - 连接对等体的数量的整数
* Example

```json
// Request
{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":74}
```
```json
// Result
{
  "id":74,
  "jsonrpc": "2.0",
  "result": "0xa" 
}
```

#### net_listening
true如果客户端正在主动侦听网络连接，则返回
* Parameters  
  * 没有
* Returns  
  * `Boolean`- true当听时，否则false
* Example

```json
// Request
{"jsonrpc":"2.0","method":"net_listening","params":[],"id":67}
```
```json
// Result
{
  "id":67,
  "jsonrpc":"2.0",
  "result":true
}
```