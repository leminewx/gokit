package rsax

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"errors"
	"math/big"
	"testing"
)

func TestEncryptBase64RoundTrip(t *testing.T) {
	publicKey, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	plaintext := []byte("gokit-rsax-base64")
	ciphertext, err := EncryptOAEPToBase64WithPublicKey(publicKey, plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("密文为空")
	}

	decrypted, err := DecryptOAEPFromBase64WithPrivateKey(privateKey, ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("解密结果 = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptBase64RejectsInvalidInput(t *testing.T) {
	_, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	if _, err := DecryptOAEPFromBase64WithPrivateKey(privateKey, "%%%"); !errors.Is(err, ErrDecodeBase64) {
		t.Fatalf("期望 ErrDecodeBase64, got %v", err)
	}
}

func TestGetMaxOAEPPlaintextSize(t *testing.T) {
	publicKey, _, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	size, err := GetMaxOAEPPlaintextSize(publicKey, crypto.SHA256)
	if err != nil {
		t.Fatalf("计算最大明文长度失败: %v", err)
	}

	want := publicKey.Size() - 2*crypto.SHA256.Size() - 2
	if size != want {
		t.Fatalf("最大明文长度 = %d, want %d", size, want)
	}
}

func TestGetMaxOAEPPlaintextSizeNilKey(t *testing.T) {
	if _, err := GetMaxOAEPPlaintextSize(nil, crypto.SHA256); !errors.Is(err, ErrNilKey) {
		t.Fatalf("期望 ErrNilKey, got %v", err)
	}
}

func TestGetMaxOAEPPlaintextSizeUnsupportedHash(t *testing.T) {
	publicKey, _, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	if _, err := GetMaxOAEPPlaintextSize(publicKey, crypto.Hash(0)); !errors.Is(err, ErrUnsupportedHash) {
		t.Fatalf("期望 ErrUnsupportedHash, got %v", err)
	}
}

func TestGetMaxOAEPPlaintextSizeKeyTooSmall(t *testing.T) {
	// 64 字节模数不足以容纳 SHA-256 的 OAEP 填充
	smallKey := &rsa.PublicKey{N: new(big.Int).Lsh(big.NewInt(1), 511), E: 65537}

	if _, err := GetMaxOAEPPlaintextSize(smallKey, crypto.SHA256); !errors.Is(err, ErrInvalidKeyBits) {
		t.Fatalf("期望 ErrInvalidKeyBits, got %v", err)
	}
}
