package rsax

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
)

// ParsePublicKey 解析 RSA 公钥，自动识别 PEM 与 DER 两种输入
//   - PEM 块类型支持 "RSA PUBLIC KEY"（PKCS#1）、"PUBLIC KEY"（PKIX）与 "CERTIFICATE"
//   - DER 输入按 PKCS#1 / PKIX 依次尝试
func ParsePublicKey(data []byte) (*rsa.PublicKey, error) {
	block, err := decodePEMBlock(data)
	if err != nil {
		// 不是 PEM 时按 DER 处理
		return parseDERPublicKey(data)
	}

	switch block.Type {
	case "RSA PUBLIC KEY":
		return parsePKCS1PublicKeyDER(block.Bytes)
	case "PUBLIC KEY":
		return parsePKIXPublicKeyDER(block.Bytes)
	case "CERTIFICATE":
		return parseCertificatePublicKey(block.Bytes)
	default:
		return nil, wrapError(ErrDecodePEM, "public key: unexpected block type %q", block.Type)
	}
}

// ParsePrivateKey 解析 RSA 私钥，自动识别 PEM 与 DER 两种输入
//   - PEM 块类型支持 "RSA PRIVATE KEY"（PKCS#1）与 "PRIVATE KEY"（PKCS#8）
//   - 解析后会校验私钥参数，避免使用损坏的密钥
func ParsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, err := decodePEMBlock(data)
	if err != nil {
		return parseDERPrivateKey(data)
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return parsePKCS1PrivateKeyDER(block.Bytes)
	case "PRIVATE KEY":
		return parsePKCS8PrivateKeyDER(block.Bytes)
	case "ENCRYPTED PRIVATE KEY":
		return nil, wrapError(ErrDecodePEM, "private key: encrypted pem is not supported")
	default:
		return nil, wrapError(ErrDecodePEM, "private key: unexpected block type %q", block.Type)
	}
}

// ParsePublicKeyFromBase64 解析 Base64 编码的 DER 公钥
func ParsePublicKeyFromBase64(encoded string) (*rsa.PublicKey, error) {
	der, err := decodeBase64(encoded)
	if err != nil {
		return nil, err
	}

	return parseDERPublicKey(der)
}

// ParsePrivateKeyFromBase64 解析 Base64 编码的 DER 私钥
func ParsePrivateKeyFromBase64(encoded string) (*rsa.PrivateKey, error) {
	der, err := decodeBase64(encoded)
	if err != nil {
		return nil, err
	}

	return parseDERPrivateKey(der)
}

// ReadPKCS1PrivateKeyToPEMString 将私钥导出为 PKCS#1 PEM 字符串
func ReadPKCS1PrivateKeyToPEMString(privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", wrapError(ErrNilKey, "private key")
	}

	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})), nil
}

// ReadPKCS8PrivateKeyToPEMString 将私钥导出为 PKCS#8 PEM 字符串
func ReadPKCS8PrivateKeyToPEMString(privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", wrapError(ErrNilKey, "private key")
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", wrapError(ErrMarshalKey, "pkcs8 private key: %w", err)
	}

	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	})), nil
}

// ReadPKCS1PrivateKeyToBase64String 将 PKCS#1 私钥导出为 Base64 字符串（DER 编码）
func ReadPKCS1PrivateKeyToBase64String(privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", wrapError(ErrNilKey, "private key")
	}

	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(privateKey)), nil
}

// ReadPKCS8PrivateKeyToBase64String 将 PKCS#8 私钥导出为 Base64 字符串（DER 编码）
func ReadPKCS8PrivateKeyToBase64String(privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", wrapError(ErrNilKey, "private key")
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", wrapError(ErrMarshalKey, "pkcs8 private key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(derBytes), nil
}

// decodePEMBlock 解码首个 PEM 块
func decodePEMBlock(data []byte) (*pem.Block, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, wrapError(ErrDecodePEM, "no pem block found")
	}

	return block, nil
}

// decodeBase64 解码标准 Base64（容忍首尾空白）
func decodeBase64(encoded string) ([]byte, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, wrapError(ErrDecodeBase64, "%w", err)
	}

	return der, nil
}

// parseDERPublicKey 依次尝试 PKCS#1 与 PKIX 解析公钥
func parseDERPublicKey(der []byte) (*rsa.PublicKey, error) {
	publicKey, err := parsePKCS1PublicKeyDER(der)
	if err == nil {
		return publicKey, nil
	}

	return parsePKIXPublicKeyDER(der)
}

// parseDERPrivateKey 依次尝试 PKCS#1 与 PKCS#8 解析私钥
func parseDERPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	privateKey, err := parsePKCS1PrivateKeyDER(der)
	if err == nil {
		return privateKey, nil
	}

	return parsePKCS8PrivateKeyDER(der)
}

func parsePKCS1PublicKeyDER(der []byte) (*rsa.PublicKey, error) {
	publicKey, err := x509.ParsePKCS1PublicKey(der)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs1 public key: %w", err)
	}

	return publicKey, nil
}

func parsePKIXPublicKeyDER(der []byte) (*rsa.PublicKey, error) {
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkix public key: %w", err)
	}

	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "public key")
	}

	return publicKey, nil
}

func parseCertificatePublicKey(der []byte) (*rsa.PublicKey, error) {
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, wrapError(ErrParseKey, "certificate: %w", err)
	}

	publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "certificate public key")
	}

	return publicKey, nil
}

func parsePKCS1PrivateKeyDER(der []byte) (*rsa.PrivateKey, error) {
	privateKey, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs1 private key: %w", err)
	}

	if err := privateKey.Validate(); err != nil {
		return nil, wrapError(ErrParseKey, "pkcs1 private key: %w", err)
	}

	return privateKey, nil
}

func parsePKCS8PrivateKeyDER(der []byte) (*rsa.PrivateKey, error) {
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, wrapError(ErrParseKey, "pkcs8 private key: %w", err)
	}

	privateKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, wrapError(ErrNotRSAKey, "private key")
	}

	if err := privateKey.Validate(); err != nil {
		return nil, wrapError(ErrParseKey, "pkcs8 private key: %w", err)
	}

	return privateKey, nil
}
