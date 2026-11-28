package rsax

import (
	"crypto"
	"errors"
	"testing"
)

func TestSignAndVerifyPKCS1(t *testing.T) {
	publicKey, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	data := []byte("gokit-rsax-pkcs1")
	signature, err := SignWithPrivateKey(privateKey, crypto.SHA256, data)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	if len(signature) != privateKey.Size() {
		t.Fatalf("签名长度 = %d, want %d", len(signature), privateKey.Size())
	}

	if err := VerifyWithPublicKey(publicKey, crypto.SHA256, data, signature); err != nil {
		t.Fatalf("验签失败: %v", err)
	}

	// 篡改数据后验签必须失败，且可被 errors.Is 判定
	if err := VerifyWithPublicKey(publicKey, crypto.SHA256, []byte("tampered"), signature); !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("篡改数据后应返回 ErrVerifyFailed, got %v", err)
	}
}

func TestSignAndVerifyPSS(t *testing.T) {
	publicKey, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	data := []byte("gokit-rsax-pss")
	signature, err := SignPSSWithPrivateKey(privateKey, crypto.SHA256, data)
	if err != nil {
		t.Fatalf("PSS 签名失败: %v", err)
	}

	if err := VerifyPSSWithPublicKey(publicKey, crypto.SHA256, data, signature); err != nil {
		t.Fatalf("PSS 验签失败: %v", err)
	}

	signature[0] ^= 0xff
	if err := VerifyPSSWithPublicKey(publicKey, crypto.SHA256, data, signature); !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("篡改签名后应返回 ErrVerifyFailed, got %v", err)
	}
}

func TestSignErrors(t *testing.T) {
	_, privateKey, err := GenerateDefaultRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	if _, err := SignWithPrivateKey(nil, crypto.SHA256, nil); !errors.Is(err, ErrNilKey) {
		t.Fatalf("期望 ErrNilKey, got %v", err)
	}

	// 未注册的哈希算法不可用
	if _, err := SignWithPrivateKey(privateKey, crypto.Hash(0), []byte("data")); !errors.Is(err, ErrUnsupportedHash) {
		t.Fatalf("期望 ErrUnsupportedHash, got %v", err)
	}

	if err := VerifyWithPublicKey(nil, crypto.SHA256, nil, nil); !errors.Is(err, ErrNilKey) {
		t.Fatalf("期望 ErrNilKey, got %v", err)
	}
}
