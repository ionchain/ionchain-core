```java
private static String node = "http://localhost:8545";
private static Web3j web3j = Web3j.build(new HttpService(node));
private static String keystoreDir = "./keystore";

```

```java
/**
     * @param from              转出地址
     * @param to                转入地址
     * @param password          交易密码
     * @param filename          ks文件路径
     * @param transactionAmount 转账金额
     */
    public void testTransaction(String from, String to, String password, String filename, double transactionAmount) {

        try {
            //获取nonce
            EthGetTransactionCount ethGetTransactionCount;
            ethGetTransactionCount = web3j.ethGetTransactionCount(from, DefaultBlockParameterName.PENDING).send();
            BigInteger nonce = ethGetTransactionCount.getTransactionCount();
            //BigInteger nonce = BigInteger.valueOf(1);
            //gasLimit
            BigInteger gasLimit = Convert.toWei("21000", Convert.Unit.WEI).toBigInteger();  //最低 21000

            //获取gasPrice
            final BigDecimal mGasPriceScaleGWei = BigDecimal.valueOf(0.1); //gWei
            BigInteger gasPrice = BigDecimal.valueOf(40).multiply(mGasPriceScaleGWei).toBigInteger();
            to = to.toLowerCase();

            //转账金额
            BigInteger value = Convert.toWei(BigDecimal.valueOf(transactionAmount), Convert.Unit.ETHER).toBigInteger();

            //构造离线交易
            RawTransaction rawTransaction = RawTransaction.createEtherTransaction(nonce, gasPrice, gasLimit, to, value);
            //签名
            Credentials credentials = loadWalletFile(password, filename);
            byte[] signedMessage = TransactionEncoder.signMessage(rawTransaction, credentials);
            String signedData = Numeric.toHexString(signedMessage);

            //发送交易
            EthSendTransaction ethSendTransaction = web3j.ethSendRawTransaction(signedData).send();
            //hash
            String hash = ethSendTransaction.getTransactionHash();
            System.out.println("signedRawTransaction: " + signedData);
            System.out.println("hash:" + hash);
        } catch (Exception e) {
            System.out.println("error = " + e.getLocalizedMessage());
        }

    }


    /**
     * 通过 钱包密码和keystore文件导入
     */
    private Credentials loadWalletFile(String password, String walletFileName) throws Exception {
        String src = keystoreDir + "/" + walletFileName;
        return WalletUtils.loadCredentials(password, src);
    }
    ```
