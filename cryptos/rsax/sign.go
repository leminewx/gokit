package rsax

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

// SignWithPrivateKey 使用 PKCS#1 v1.5 方案对 data 签名
//   - hash 需为可用的哈希算法（如 crypto.SHA256）；调用方需保证其对应包已导入
func SignWithPrivateKey(privateKey *rsa.PrivateKey, hash crypto.Hash, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, wrapError(ErrNilKey, "private key")
	}

	digest, err := digestOf(hash, data)
	if err != nil {
		return nil, err
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, hash, digest)
	if err != nil {
		return nil, wrapError(ErrSignFailed, "pkcs1v15: %w", err)
	}

	return signature, nil
}

// VerifyWithPublicKey 验证 PKCS#1 v1.5 签名
//   - 签名不匹配时返回包装了 ErrVerifyFailed 的错误
func VerifyWithPublicKey(publicKey *rsa.PublicKey, hash crypto.Hash, data, signature []byte) error {
	if publicKey == nil {
		return wrapError(ErrNilKey, "public key")
	}

	digest, err := digestOf(hash, data)
	if err != nil {
		return err
	}

	if err := rsa.VerifyPKCS1v15(publicKey, hash, digest, signature); err != nil {
		return wrapError(ErrVerifyFailed, "pkcs1v15: %w", err)
	}

	return nil
}

// SignPSSWithPrivateKey 使用 PSS 方案对 data 签名
//   - 盐长度取哈希输出长度（PSSSaltLengthEqualsHash），与多数实现可互操作
func SignPSSWithPrivateKey(privateKey *rsa.PrivateKey, hash crypto.Hash, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, wrapError(ErrNilKey, "private key")
	}

	digest, err := digestOf(hash, data)
	if err != nil {
		return nil, err
	}

	signature, err := rsa.SignPSS(rand.Reader, privateKey, hash, digest, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       hash,
	})
	if err != nil {
		return nil, wrapError(ErrSignFailed, "pss: %w", err)
	}

	return signature, nil
}

// VerifyPSSWithPublicKey 验证 PSS 签名
//   - 签名不匹配时返回包装了 ErrVerifyFailed 的错误
func VerifyPSSWithPublicKey(publicKey *rsa.PublicKey, hash crypto.Hash, data, signature []byte) error {
	if publicKey == nil {
		return wrapError(ErrNilKey, "public key")
	}

	digest, err := digestOf(hash, data)
	if err != nil {
		return err
	}

	if err := rsa.VerifyPSS(publicKey, hash, digest, signature, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       hash,
	}); err != nil {
		return wrapError(ErrVerifyFailed, "pss: %w", err)
	}

	return nil
}

// digestOf 计算 data 的摘要，并校验哈希算法可用
func digestOf(hash crypto.Hash, data []byte) ([]byte, error) {
	if !hash.Available() {
		return nil, wrapError(ErrUnsupportedHash, "%v", hash)
	}

	hasher := hash.New()
	hasher.Write(data)

	return hasher.Sum(nil), nil
}
