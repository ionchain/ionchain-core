使用Gradle构建java工程

# 依赖：

```
// 引入 libs 下面的jar包
compile fileTree(dir: 'libs', include: ['*.jar'])

// 钱包账户工具类中使用到
compile group: 'command.fasterxml.jackson.core', name: 'jackson-core', version: '2.9.9'
compile group: 'command.fasterxml.jackson.core', name: 'jackson-databind', version: '2.9.9'

//生成助记词
compile group: 'org.bouncycastle', name: 'bcprov-jdk15on', version: '1.62'
compile 'org.bitcoinj:bitcoinj-core:0.14.7'

// 控制台日志 ,可不加
compile 'ch.qos.logback:logback-core:1.2.3'
```
# 完整的构建脚本

```groovy
plugins {
    id 'java'
}

group 'IONCShool'
version '1.0-SNAPSHOT'

sourceCompatibility = 1.8

repositories {
    // 先从阿里云去下载相关依赖
    maven {
        url { 'http://maven.aliyun.com/nexus/content/groups/public/' }
    }
    google()
    jcenter()
    mavenCentral()
//    maven {
//        url "https://oss.sonatype.org/content/repositories/snapshots"
//    }
}

dependencies {
    // 引入 libs 下面的jar包
    compile fileTree(dir: 'libs', include: ['*.jar'])

    // 钱包账户工具类中使用到
    compile group: 'command.fasterxml.jackson.core', name: 'jackson-core', version: '2.9.9'
    compile group: 'command.fasterxml.jackson.core', name: 'jackson-databind', version: '2.9.9'

    //生成助记词
    compile group: 'org.bouncycastle', name: 'bcprov-jdk15on', version: '1.62'
    compile 'org.bitcoinj:bitcoinj-core:0.14.7'

    // 控制台日志 ,可不加
    compile 'ch.qos.logback:logback-core:1.2.3'
    compile 'ch.qos.logback:logback-classic:1.2.3'
}
```
# 代码片段

## 生成助记词

```
//生成助记词
    // https://mvnrepository.com/artifact/org.bitcoinj/bitcoinj-core
//    compile group: 'org.bitcoinj', name: 'bitcoinj-core', version: '0.15.2'
    compile 'org.bitcoinj:bitcoinj-core:0.14.7'
```
```java
/**
 * 生成助记词
 *
 * @param walletName 钱包的名字
 * @param password   钱包密码
 * @return 助记词字符串
 */
public String generateMnemonic(String walletName, String password) {
    byte[] initialEntropy = new byte[16];
    secureRandom.nextBytes(initialEntropy);//产生一个随机数
    return MnemonicUtils.generateMnemonic(initialEntropy);
}
```
## 生成账户信息:公私钥、地址

```java
/**
 *
 * @param walletName   钱包的名字
 * @param mnemonicCode 助记词
 * @param password     密码
 */
public WalletBean importWalletByMnemonicCode(String walletName, String mnemonicCode, String password) {
    try {
        String[] pathArray = ETH_TYPE.split("/");
        String passphrase = "";
        long creationTimeSeconds = System.currentTimeMillis() / 1000;
        DeterministicSeed ds;
        ds = new DeterministicSeed(mnemonicCode, null, passphrase, creationTimeSeconds);
        //种子
        byte[] seedBytes = ds.getSeedBytes();
        if (seedBytes == null) {
            log.error("失败");
            return null;
        }
        DeterministicKey dkKey = HDKeyDerivation.createMasterPrivateKey(seedBytes);
        for (int i = 1; i < pathArray.length; i++) {
            ChildNumber childNumber;
            if (pathArray[i].endsWith("'")) {
                int number = Integer.parseInt(pathArray[i].substring(0,
                        pathArray[i].length() - 1));
                childNumber = new ChildNumber(number, true);
            } else {
                int number = Integer.parseInt(pathArray[i]);
                childNumber = new ChildNumber(number, false);
            }
            dkKey = HDKeyDerivation.deriveChildKey(dkKey, childNumber);
        }
        ECKeyPair ecKeyPair = ECKeyPair.create(dkKey.getPrivKeyBytes());
        WalletBean walletBean = new WalletBean();
//            WalletFile walletFile = Wallet.create(password, ecKeyPair, 1024, 1); // WalletUtils. .generateNewWalletFile();
        String privateKey = ecKeyPair.getPrivateKey().toString(16);
        String publicKey = ecKeyPair.getPublicKey().toString(16);
        walletBean.setPrivateKey(privateKey);
        walletBean.setPublic_key(publicKey);
        log.info("私钥： " + privateKey);
        String keystore = WalletUtils.generateWalletFile(password, ecKeyPair, new File(keystoreDir), false);
        keystore = keystoreDir + "/" + keystore;
        walletBean.setKeystore(keystore);
        log.info("钱包keystore： " + keystore);
        walletBean.setName(walletName);
//            String addr1 = walletFile.getAddress();
        String addr2 = Keys.getAddress(ecKeyPair);
        String walletAddress = toChecksumAddress(addr2);
        walletBean.setAddress(toChecksumAddress(walletAddress));//设置钱包地址
        log.info("钱包地址： " + walletAddress);
        walletBean.setPassword(password);
        log.info("助记词 === " + mnemonicCode);
        walletBean.setMnemonic(mnemonicCode);
        log.info(walletBean.toString());
        return walletBean;
    } catch (CipherException | IOException | UnreadableWalletException e) {
        log.error(e.getMessage());
        return null;
    }
}
```
## 地址合法性检验

```java
/**
 * 地址合法性校验
 *
 * @param address IONC Wallet Account 地址
 * @return 是否合法
 */
public static boolean isIONCValidAddress(String address) {
    if (isEmpty(address) || !address.startsWith("0x"))
        return false;
    return isValidAddress(address);
}

private static boolean isEmpty(String input) {
    return input == null || input.isEmpty();
}

public static boolean isValidAddress(String input) {
    String cleanInput = Numeric.cleanHexPrefix(input);

    try {
        Numeric.toBigIntNoPrefix(cleanInput);
    } catch (NumberFormatException e) {
        return false;
    }

    return cleanInput.length() == 40;
}
```
