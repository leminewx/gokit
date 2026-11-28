package rsax

import (
	"crypto"
	"crypto/rsa"
	"encoding/base64"
)

// EncryptOAEPToBase64WithPublicKey 使用 OAEP 加密并返回标准 Base64 字符串
func EncryptOAEPToBase64WithPublicKey(publicKey *rsa.PublicKey, plaintext []byte) (string, error) {
	ciphertext, err := EncryptOAEPWithPublicKey(publicKey, plaintext)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptOAEPFromBase64WithPrivateKey 解密 Base64 字符串
func DecryptOAEPFromBase64WithPrivateKey(privateKey *rsa.PrivateKey, ciphertext string) ([]byte, error) {
	encrypted, err := decodeBase64(ciphertext)
	if err != nil {
		return nil, err
	}

	return DecryptOAEPWithPrivateKey(privateKey, encrypted)
}

// GetMaxOAEPPlaintextSize 返回 OAEP 单次加密允许的最大明文长度（字节）
//   - 超过该长度的数据应先分块，或改用对称加密加 RSA 封装密钥
func GetMaxOAEPPlaintextSize(publicKey *rsa.PublicKey, hash crypto.Hash) (int, error) {
	if publicKey == nil {
		return 0, wrapError(ErrNilKey, "public key")
	}

	if !hash.Available() {
		return 0, wrapError(ErrUnsupportedHash, "%v", hash)
	}

	size := publicKey.Size() - 2*hash.Size() - 2
	if size <= 0 {
		return 0, wrapError(ErrInvalidKeyBits, "key too small for %v", hash)
	}

	return size, nil
}
