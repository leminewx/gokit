# cryptos 安全加解密相关 API
导入前缀：`gitee.com/leminewx/gokit/cryptos/...`

## cryptos/rsax

RSA 密钥生成、PEM/Base64/DER 编解码、OAEP 加解密与 PKCS#1 v1.5 / PSS 签名。
所有函数只处理 `*rsa.PublicKey` / `*rsa.PrivateKey`，不做全局状态缓存。

### 密钥生成与加解密

| 函数 | 说明 |
|------|------|
| `GenerateRSAKeyPair(bits int) (*rsa.PublicKey, *rsa.PrivateKey, error)` | 生成密钥对；`bits < 2048` 返回 `ErrInvalidKeyBits`（已有 1024 位密钥仍可解析） |
| `GenerateDefaultRSAKeyPair() (*rsa.PublicKey, *rsa.PrivateKey, error)` | 使用 2048 位生成 |
| `EncryptOAEPWithPublicKey(publicKey, plaintext) ([]byte, error)` | 公钥 OAEP（SHA-256）加密 |
| `DecryptOAEPWithPrivateKey(privateKey, ciphertext) ([]byte, error)` | 私钥 OAEP 解密 |
| `EncryptOAEPToBase64WithPublicKey(publicKey, plaintext) (string, error)` | OAEP 加密后输出 Base64 |
| `DecryptOAEPFromBase64WithPrivateKey(privateKey, ciphertext string) ([]byte, error)` | 解析 Base64 并解密 |
| `GetMaxOAEPPlaintextSize(publicKey, hash crypto.Hash) (int, error)` | 可加密的最大明文长度（`Size - 2*hash.Size - 2`） |

### 签名与验签

```go
sig, err := rsax.SignWithPrivateKey(pri, crypto.SHA256, []byte("payload"))
err = rsax.VerifyWithPublicKey(pub, crypto.SHA256, []byte("payload"), sig)
```

| 函数 | 说明 |
|------|------|
| `SignWithPrivateKey(privateKey, hash, data) ([]byte, error)` | PKCS#1 v1.5 签名 |
| `VerifyWithPublicKey(publicKey, hash, data, signature) error` | PKCS#1 v1.5 验签，失败返回 `ErrVerifyFailed` |
| `SignPSSWithPrivateKey(privateKey, hash, data) ([]byte, error)` | PSS 签名，盐长等于摘要长度 |
| `VerifyPSSWithPublicKey(publicKey, hash, data, signature) error` | PSS 验签 |

`hash` 需为已链接的哈希实现（如 `crypto.SHA256`），不可用时返回 `ErrUnsupportedHash`。

### 通用 PEM / DER 解析

| 函数 | 说明 |
|------|------|
| `ParsePublicKey(data []byte) (*rsa.PublicKey, error)` | PEM（`RSA PUBLIC KEY` / `PUBLIC KEY` / `CERTIFICATE`）或 DER（PKCS#1 → PKIX 自动尝试） |
| `ParsePrivateKey(data []byte) (*rsa.PrivateKey, error)` | PEM / DER 的 PKCS#1 或 PKCS#8，解析后做 `Validate` |
| `ParsePublicKeyFromBase64(encoded string)` / `ParsePrivateKeyFromBase64(encoded string)` | 先做 Base64 解码再按上述规则解析 |
| `ReadPKCS1PrivateKeyToPEMString` / `ReadPKCS8PrivateKeyToPEMString` | 私钥 → PEM 字符串 |
| `ReadPKCS1PrivateKeyToBase64String` / `ReadPKCS8PrivateKeyToBase64String` | 私钥 → Base64 字符串 |

加密私钥（`ENCRYPTED PRIVATE KEY`）暂不支持，会返回明确的错误。

### 公钥格式

| 函数 | 说明 |
|------|------|
| `SavePKCS1PublicKeyToFile(publicKey, filename) error` | 导出 PKCS#1 PEM |
| `SavePKIXPublicKeyToFile(publicKey, filename) error` | 导出 PKIX（`PUBLIC KEY`）PEM |
| `LoadPKCS1PublicKeyFromFile(filename) (*rsa.PublicKey, error)` | 读取 PKCS#1 |
| `LoadPKIXPublicKeyFromFile(filename) (*rsa.PublicKey, error)` | 读取 PKIX |
| `ReadPKCS1PublicKeyToPEMString(publicKey) (string, error)` | PKCS#1 → PEM 字符串 |
| `ReadPKCS1PublicKeyToBase64String(publicKey) (string, error)` | PKCS#1 → Base64 |
| `ReadPKIXPublicKeyToPEMString(publicKey) (string, error)` | PKIX → PEM 字符串 |
| `ReadPKIXPublicKeyToBase64String(publicKey) (string, error)` | PKIX → Base64 |

### 私钥格式

| 函数 | 说明 |
|------|------|
| `SavePKCS1PrivateKeyToFile(privateKey, filename) error` | 导出 PKCS#1（`RSA PRIVATE KEY`） |
| `SavePKCS8PrivateKeyToFile(privateKey, filename) error` | 导出 PKCS#8（`PRIVATE KEY`） |
| `LoadPKCS1PrivateKeyFromFile(filename) (*rsa.PrivateKey, error)` | 读取 PKCS#1 |
| `LoadPKCS8PrivateKeyFromFile(filename) (*rsa.PrivateKey, error)` | 读取 PKCS#8 |
| `LoadPKIXPrivateKeyFromFile(filename) (*rsa.PrivateKey, error)` | 读取 PKIX 私钥 |

私钥落盘函数以 `0600` 权限创建（已存在的文件也会被收紧），避免同一主机上的其他用户读取；
Windows 上权限由 ACL 决定。

### 示例

```go
pub, pri, err := rsax.GenerateRSAKeyPair(2048)
if err != nil {
    return err
}

cipher, err := rsax.EncryptOAEPWithPublicKey(pub, []byte("secret"))
plain, err := rsax.DecryptOAEPWithPrivateKey(pri, cipher)
```

### 测试与密钥文件约定

- 测试用例把密钥写到 `harness/testdata/`（`const _TESTDATA_DIR = "../../harness/testdata"`），
  并通过 `t.Cleanup` 删除，**不得**在 `cryptos/rsax/` 包目录生成 `*.pem`。
- 仓库不提交任何真实私钥；需要长期密钥时应由部署流程注入或使用密钥管理服务。

## cryptos/password

基于 bcrypt 的密码摘要与校验：盐值与代价因子编码在摘要内，无需单独存储盐。

```go
hashed, err := password.Hash([]byte(plaintext))       // 默认 cost = 10
hashed, err := password.HashWithCost(plaintext, 12)   // 自定义 cost∈[4,31]

err = password.Verify(hashed, []byte(plaintext))      // 不匹配 → ErrPasswordMismatch
ok := password.Match(hashed, []byte(plaintext))       // 布尔封装

if password.IsRehashNeeded(hashed, 12) {              // 登录成功后逐步提升 cost
    hashed, _ = password.HashWithCost([]byte(plaintext), 12)
}
```

| 符号 | 说明 |
|------|------|
| `Hash(plain []byte) ([]byte, error)` | 默认代价因子（`DEFAULT_COST` = 10）摘要 |
| `HashWithCost(plain []byte, cost int) ([]byte, error)` | 指定代价因子；越界返回 `ErrInvalidCost`；密码超过 72 字节返回 `bcrypt.ErrPasswordTooLong`（不截断） |
| `Verify(hashed, plain []byte) error` | 匹配返回 `nil`；不匹配 `ErrPasswordMismatch`；非 bcrypt 摘要 `ErrInvalidHash` |
| `Match(hashed, plain []byte) bool` | `Verify` 的布尔封装 |
| `IsRehashNeeded(hashed []byte, cost int) bool` | 摘要代价因子与 `cost` 不一致（或摘要非法）时返回 `true` |
| 常量 | `DEFAULT_COST` / `MIN_COST`(4) / `MAX_COST`(31) |

注意：bcrypt 只取密码前 72 字节，超长输入直接报错而不是静默截断；密码长度与复杂度校验由调用方自行负责。

