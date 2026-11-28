package rsax

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
)

// _MIN_KEY_BITS RSA 密钥允许的最小位数
//   - 1024 位已不满足现代安全强度（NIST SP 800-57 建议至少 2048 位），
//     仍可解析既有的 1024 位密钥，但不再生成新的弱密钥
const _MIN_KEY_BITS = 2048

// _DEFAULT_KEY_BITS RSA 密钥推荐的默认位数
const _DEFAULT_KEY_BITS = 2048

// GenerateDefaultRSAKeyPair 使用推荐的 2048 位生成 RSA 公私钥对
func GenerateDefaultRSAKeyPair() (*rsa.PublicKey, *rsa.PrivateKey, error) {
	return GenerateRSAKeyPair(_DEFAULT_KEY_BITS)
}

// GenerateRSAKeyPair generates a pair of the RSA public and private keys.
//   - bits：recommend using multiples of 1024，且不得小于 2048.
func GenerateRSAKeyPair(bits int) (*rsa.PublicKey, *rsa.PrivateKey, error) {
	if bits < _MIN_KEY_BITS {
		return nil, nil, wrapError(ErrInvalidKeyBits, "%d < %d", bits, _MIN_KEY_BITS)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, wrapError(ErrGenerateKeyPair, "%w", err)
	}

	return &privateKey.PublicKey, privateKey, nil
}

// ---------------------------------------------------------------------------------------------------------------------
// operations of RSA public key.
// ---------------------------------------------------------------------------------------------------------------------

// SavePKCS1PublicKeyToFile saves the PKCS1 RSA public key to the file (PEM coding).
func SavePKCS1PublicKeyToFile(publicKey *rsa.PublicKey, filename string) error {
	if publicKey == nil {
		return wrapError(ErrNilKey, "public key")
	}

	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY", // PKCS#1 special use
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create(filename)
	if err != nil {
		return wrapError(ErrCreateFile, "%q: %w", filename, err)
	}
	defer func() { _ = publicKeyFile.Close() }()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return wrapError(ErrEncodePEM, "pkcs1 public key: %w", err)
	}

	return nil
}

// SavePKIXPublicKeyToFile saves the PKIX RSA public key to the file (PEM coding).
func SavePKIXPublicKeyToFile(publicKey *rsa.PublicKey, filename string) error {
	if publicKey == nil {
		return wrapError(ErrNilKey, "public key")
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return wrapError(ErrMarshalKey, "pkix public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY", // PKIX standard
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create(filename)
	if err != nil {
		return wrapError(ErrCreateFile, "%q: %w", filename, err)
	}
	defer func() { _ = publicKeyFile.Close() }()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return wrapError(ErrEncodePEM, "pkix public key: %w", err)
	}

	return nil
}

// LoadPKCS1PublicKeyFromFile loads the PKCS#1 RSA public key from the file (PEM coding).
func LoadPKCS1PublicKeyFromFile(filename string) (*rsa.PublicKey, error) {
	publicKeyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFile, "%q: %w", filename, err)
	}

	block, _ := pem.Decode(publicKeyFile)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, wrapError(ErrDecodePEM, "pkcs1 public key")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs1 public key: %w", err)
	}

	return publicKey, nil
}

// LoadPKIXPublicKeyFromFile loads the PKIX RSA public key from the file (PEM coding).
func LoadPKIXPublicKeyFromFile(filename string) (*rsa.PublicKey, error) {
	publicKeyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFile, "%q: %w", filename, err)
	}

	block, _ := pem.Decode(publicKeyFile)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, wrapError(ErrDecodePEM, "pkix public key")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkix public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "public key")
	}

	return publicKey, nil
}

// ReadPKCS1PublicKeyToPEMString reads the PKCS#1 RSA public key to a PEM string (DER coding).
func ReadPKCS1PublicKeyToPEMString(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", wrapError(ErrNilKey, "public key")
	}

	// marshal the public to the DER bytes.
	derBytes := x509.MarshalPKCS1PublicKey(publicKey)
	pemBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY", // PKCS#1 special use
		Bytes: derBytes,
	}

	// encodes to a string.
	pemBytes := pem.EncodeToMemory(pemBlock)
	return string(pemBytes), nil
}

// ReadPKIXPublicKeyToPEMString reads the PKIX RSA public key to a PEM string (DER coding).
func ReadPKIXPublicKeyToPEMString(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", wrapError(ErrNilKey, "public key")
	}

	// marshal the public to the DER bytes.
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", wrapError(ErrMarshalKey, "pkix public key: %w", err)
	}

	// codes to a DER blick.
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY", // PKIX standard
		Bytes: derBytes,
	}

	// encodes to a string.
	pemBytes := pem.EncodeToMemory(pemBlock)
	return string(pemBytes), nil
}

// ReadPKCS1PublicKeyToBase64String reads the PKCS#1 RSA public key to a base64 string (DER coding).
func ReadPKCS1PublicKeyToBase64String(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", wrapError(ErrNilKey, "public key")
	}

	derBytes := x509.MarshalPKCS1PublicKey(publicKey)
	return base64.StdEncoding.EncodeToString(derBytes), nil
}

// ReadPKIXPublicKeyToBase64String reads the PKIX RSA public key to a base64 string (DER coding).
func ReadPKIXPublicKeyToBase64String(publicKey *rsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", wrapError(ErrNilKey, "public key")
	}

	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", wrapError(ErrMarshalKey, "pkix public key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(derBytes), nil
}

// EncryptOAEPWithPublicKey encrypts the RSA public key with OAEP by SHA-256.
func EncryptOAEPWithPublicKey(publicKey *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, wrapError(ErrNilKey, "public key")
	}

	encryptBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, plaintext, nil)
	if err != nil {
		return nil, wrapError(ErrEncrypt, "oaep: %w", err)
	}

	return encryptBytes, nil
}

// ---------------------------------------------------------------------------------------------------------------------
// operations of RSA private key.
// ---------------------------------------------------------------------------------------------------------------------

// SavePKCS1PrivateKeyToFile saves the PKCS#1 RSA private key to the file (PEM coding).
//   - 文件以 0600 权限创建，避免同一主机上的其他用户读取私钥（Windows 由 ACL 决定）
func SavePKCS1PrivateKeyToFile(privateKey *rsa.PrivateKey, filename string) error {
	if privateKey == nil {
		return wrapError(ErrNilKey, "private key")
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY", // PKCS#1 special using
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return wrapError(ErrCreateFile, "%q: %w", filename, err)
	}
	defer func() { _ = privateKeyFile.Close() }()

	// 已存在的文件不会因 O_CREATE 恢复权限，必须显式收紧
	//   - 不能先用 O_TRUNC 打开：open 即截断，若随后 Chmod 失败（如文件属主不是当前用户、
	//     NFS/FAT 不支持权限位），旧私钥会被销毁且新私钥未写入，造成不可恢复的数据丢失
	//   - 因此按“先收紧权限、再截断写入”的顺序处理
	if err := privateKeyFile.Chmod(0o600); err != nil {
		return wrapError(ErrCreateFile, "%q: chmod 0600: %w", filename, err)
	}
	if err := privateKeyFile.Truncate(0); err != nil {
		return wrapError(ErrCreateFile, "%q: truncate: %w", filename, err)
	}

	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return wrapError(ErrEncodePEM, "pkcs1 private key: %w", err)
	}

	return nil
}

// SavePKCS8PrivateKeyToFile saves the PKCS#8 RSA private key to the file (PEM coding).
//   - 文件以 0600 权限创建，避免同一主机上的其他用户读取私钥（Windows 由 ACL 决定）
func SavePKCS8PrivateKeyToFile(privateKey *rsa.PrivateKey, filename string) error {
	if privateKey == nil {
		return wrapError(ErrNilKey, "private key")
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return wrapError(ErrMarshalKey, "pkcs8 private key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY", // PKCS#8 standard
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return wrapError(ErrCreateFile, "%q: %w", filename, err)
	}
	defer func() { _ = privateKeyFile.Close() }()

	// 已存在的文件不会因 O_CREATE 恢复权限，必须显式收紧
	//   - 不能先用 O_TRUNC 打开：open 即截断，若随后 Chmod 失败（如文件属主不是当前用户、
	//     NFS/FAT 不支持权限位），旧私钥会被销毁且新私钥未写入，造成不可恢复的数据丢失
	//   - 因此按“先收紧权限、再截断写入”的顺序处理
	if err := privateKeyFile.Chmod(0o600); err != nil {
		return wrapError(ErrCreateFile, "%q: chmod 0600: %w", filename, err)
	}
	if err := privateKeyFile.Truncate(0); err != nil {
		return wrapError(ErrCreateFile, "%q: truncate: %w", filename, err)
	}

	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return wrapError(ErrEncodePEM, "pkcs8 private key: %w", err)
	}

	return nil
}

// LoadPKCS1PrivateKeyFromFile loads the PKCS#1 RSA private key from the file (PEM coding).
func LoadPKCS1PrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
	privateKeyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFile, "%q: %w", filename, err)
	}

	block, _ := pem.Decode(privateKeyFile)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, wrapError(ErrDecodePEM, "pkcs1 private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs1 private key: %w", err)
	}

	return privateKey, nil
}

// LoadPKCS8PrivateKeyFromFile loads the PKCS#8 RSA private key from the file (PEM coding).
func LoadPKCS8PrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
	privateKeyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFile, "%q: %w", filename, err)
	}

	block, _ := pem.Decode(privateKeyFile)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, wrapError(ErrDecodePEM, "pkcs8 private key")
	}

	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs8 private key: %w", err)
	}

	privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "private key")
	}

	return privateKey, nil
}

// LoadPKIXPrivateKeyFromFile loads the PKIX (PKCS#8) RSA private key from the file (PEM coding).
//   - Note: a PKIX private key is encoded as a PKCS#8 "PRIVATE KEY" block.
func LoadPKIXPrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {
	privateKeyFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFile, "%q: %w", filename, err)
	}

	block, _ := pem.Decode(privateKeyFile)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, wrapError(ErrDecodePEM, "pkix private key")
	}

	// PKIX 私钥实际以 PKCS#8 格式编码
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkix private key: %w", err)
	}

	privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "private key")
	}

	return privateKey, nil
}

// DecryptOAEPWithPrivateKey decrypts the RSA private key with OAEP by SHA-256.
func DecryptOAEPWithPrivateKey(privateKey *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, wrapError(ErrNilKey, "private key")
	}

	decryptBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, wrapError(ErrDecrypt, "oaep: %w", err)
	}

	return decryptBytes, nil
}
