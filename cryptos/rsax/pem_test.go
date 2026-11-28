package rsax

import (
	"errors"
	"testing"
)

func TestParseKeyFromPEM(t *testing.T) {
	publicKey, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// PKCS#1 公钥 PEM
	pkcs1PublicPEM, err := ReadPKCS1PublicKeyToPEMString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	parsedPublic, err := ParsePublicKey([]byte(pkcs1PublicPEM))
	if err != nil {
		t.Fatalf("解析 PKCS#1 公钥失败: %v", err)
	}
	if parsedPublic.N.Cmp(publicKey.N) != 0 {
		t.Fatal("PKCS#1 公钥解析结果不一致")
	}

	// PKIX 公钥 PEM
	pkixPublicPEM, err := ReadPKIXPublicKeyToPEMString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePublicKey([]byte(pkixPublicPEM)); err != nil {
		t.Fatalf("解析 PKIX 公钥失败: %v", err)
	}

	// PKCS#1 私钥 PEM
	pkcs1PrivatePEM, err := ReadPKCS1PrivateKeyToPEMString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	parsedPrivate, err := ParsePrivateKey([]byte(pkcs1PrivatePEM))
	if err != nil {
		t.Fatalf("解析 PKCS#1 私钥失败: %v", err)
	}
	if parsedPrivate.D.Cmp(privateKey.D) != 0 {
		t.Fatal("PKCS#1 私钥解析结果不一致")
	}

	// PKCS#8 私钥 PEM
	pkcs8PrivatePEM, err := ReadPKCS8PrivateKeyToPEMString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePrivateKey([]byte(pkcs8PrivatePEM)); err != nil {
		t.Fatalf("解析 PKCS#8 私钥失败: %v", err)
	}
}

func TestParseKeyFromBase64(t *testing.T) {
	publicKey, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	publicBase64, err := ReadPKIXPublicKeyToBase64String(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	parsedPublic, err := ParsePublicKeyFromBase64(publicBase64)
	if err != nil {
		t.Fatalf("解析 Base64 公钥失败: %v", err)
	}
	if parsedPublic.N.Cmp(publicKey.N) != 0 {
		t.Fatal("Base64 公钥解析结果不一致")
	}

	privateBase64, err := ReadPKCS8PrivateKeyToBase64String(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	parsedPrivate, err := ParsePrivateKeyFromBase64(privateBase64)
	if err != nil {
		t.Fatalf("解析 Base64 私钥失败: %v", err)
	}
	if parsedPrivate.D.Cmp(privateKey.D) != 0 {
		t.Fatal("Base64 私钥解析结果不一致")
	}

	if _, err := ParsePublicKeyFromBase64("not-base64!"); !errors.Is(err, ErrDecodeBase64) {
		t.Fatalf("期望 ErrDecodeBase64, got %v", err)
	}
}

func TestParseInvalidKey(t *testing.T) {
	if _, err := ParsePublicKey([]byte("definitely not a key")); err == nil {
		t.Fatal("非法公钥输入应返回错误")
	}
	if _, err := ParsePrivateKey([]byte("definitely not a key")); err == nil {
		t.Fatal("非法私钥输入应返回错误")
	}

	publicKey, _, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// 用公钥 PEM 解析私钥应失败
	publicPEM, err := ReadPKIXPublicKeyToPEMString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePrivateKey([]byte(publicPEM)); !errors.Is(err, ErrDecodePEM) {
		t.Fatalf("期望 ErrDecodePEM, got %v", err)
	}
}
