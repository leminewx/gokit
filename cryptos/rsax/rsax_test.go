package rsax

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 测试数据统一存放在仓库根的 harness/testdata，禁止写进包目录；
// 键文件由用例自己创建并在结束时清理，避免污染工作区。
const _TESTDATA_DIR = "../../harness/testdata"

// testdataFile 返回 harness/testdata 下的临时测试文件路径，并在用例结束时删除
func testdataFile(t *testing.T, name string) string {
	t.Helper()

	if err := os.MkdirAll(_TESTDATA_DIR, 0o755); err != nil {
		t.Fatalf("创建测试数据目录失败: %v", err)
	}

	path := filepath.Join(_TESTDATA_DIR, name)
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}

func TestGenerateKeyPair(t *testing.T) {
	publicKey, privateKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// PKCS#1 公钥/私钥落盘到 harness/testdata
	if err := SavePKCS1PublicKeyToFile(publicKey, testdataFile(t, "rsax_public_key.pem")); err != nil {
		t.Fatalf("保存公钥失败: %v", err)
	}
	if err := SavePKCS1PrivateKeyToFile(privateKey, testdataFile(t, "rsax_private_key.pem")); err != nil {
		t.Fatalf("保存私钥失败: %v", err)
	}
}

func TestCrypto(t *testing.T) {
	publicKey, privateKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 落盘后再加载，验证 保存 → 加载 → 加密 → 解密 全链路
	publicKeyPath := testdataFile(t, "rsax_crypto_public.pem")
	privateKeyPath := testdataFile(t, "rsax_crypto_private.pem")
	if err := SavePKCS1PublicKeyToFile(publicKey, publicKeyPath); err != nil {
		t.Fatalf("保存公钥失败: %v", err)
	}
	if err := SavePKCS1PrivateKeyToFile(privateKey, privateKeyPath); err != nil {
		t.Fatalf("保存私钥失败: %v", err)
	}

	loadedPublicKey, err := LoadPKCS1PublicKeyFromFile(publicKeyPath)
	if err != nil {
		t.Fatalf("加载公钥失败: %v", err)
	}
	loadedPrivateKey, err := LoadPKCS1PrivateKeyFromFile(privateKeyPath)
	if err != nil {
		t.Fatalf("加载私钥失败: %v", err)
	}

	plaintext := []byte("123456789")
	ciphertext, err := EncryptOAEPWithPublicKey(loadedPublicKey, plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("密文不应与明文相同")
	}

	decrypted, err := DecryptOAEPWithPrivateKey(loadedPrivateKey, ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密结果 = %q, want %q", decrypted, plaintext)
	}
}

// TestKeyFormatRoundTrip 覆盖 PKCS#8 私钥与 PKIX 公钥的保存/加载往返
func TestKeyFormatRoundTrip(t *testing.T) {
	publicKey, privateKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	pkixPublicPath := testdataFile(t, "rsax_pkix_public.pem")
	pkcs8PrivatePath := testdataFile(t, "rsax_pkcs8_private.pem")
	if err := SavePKIXPublicKeyToFile(publicKey, pkixPublicPath); err != nil {
		t.Fatalf("保存 PKIX 公钥失败: %v", err)
	}
	if err := SavePKCS8PrivateKeyToFile(privateKey, pkcs8PrivatePath); err != nil {
		t.Fatalf("保存 PKCS#8 私钥失败: %v", err)
	}

	loadedPublicKey, err := LoadPKIXPublicKeyFromFile(pkixPublicPath)
	if err != nil {
		t.Fatalf("加载 PKIX 公钥失败: %v", err)
	}
	loadedPrivateKey, err := LoadPKCS8PrivateKeyFromFile(pkcs8PrivatePath)
	if err != nil {
		t.Fatalf("加载 PKCS#8 私钥失败: %v", err)
	}

	plaintext := []byte("gokit-rsax")
	ciphertext, err := EncryptOAEPWithPublicKey(loadedPublicKey, plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	decrypted, err := DecryptOAEPWithPrivateKey(loadedPrivateKey, ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密结果 = %q, want %q", decrypted, plaintext)
	}
}

func TestReadPublicKeyPEMString(t *testing.T) {
	publicKey, _, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// PKIX（BEGIN PUBLIC KEY）形式
	pemString, err := ReadPKIXPublicKeyToPEMString(publicKey)
	if err != nil {
		t.Fatalf("导出 PKIX PEM 失败: %v", err)
	}
	if !strings.HasPrefix(pemString, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("PKIX PEM 头不正确: %q", pemString[:min(len(pemString), 40)])
	}

	// PKCS#1（BEGIN RSA PUBLIC KEY）形式
	pkcs1PEM, err := ReadPKCS1PublicKeyToPEMString(publicKey)
	if err != nil {
		t.Fatalf("导出 PKCS#1 PEM 失败: %v", err)
	}
	if !strings.HasPrefix(pkcs1PEM, "-----BEGIN RSA PUBLIC KEY-----") {
		t.Errorf("PKCS#1 PEM 头不正确: %q", pkcs1PEM[:min(len(pkcs1PEM), 40)])
	}

	// Base64 形式不应包含 PEM 头尾
	base64String, err := ReadPKIXPublicKeyToBase64String(publicKey)
	if err != nil {
		t.Fatalf("导出 PKIX Base64 失败: %v", err)
	}
	if base64String == "" || strings.Contains(base64String, "BEGIN") {
		t.Errorf("Base64 输出不合法: %q", base64String)
	}
}

// TestGenerateRSAKeyPairRejectsWeakBits 验证不再生成 1024 位等弱密钥
func TestGenerateRSAKeyPairRejectsWeakBits(t *testing.T) {
	t.Parallel()

	for _, bits := range []int{512, 1024, 1025, 2047} {
		if _, _, err := GenerateRSAKeyPair(bits); !errors.Is(err, ErrInvalidKeyBits) {
			t.Errorf("GenerateRSAKeyPair(%d) = %v, want ErrInvalidKeyBits", bits, err)
		}
	}

	if _, _, err := GenerateRSAKeyPair(2048); err != nil {
		t.Fatalf("GenerateRSAKeyPair(2048) 失败: %v", err)
	}
}

// TestSavePrivateKeyFileMode 验证私钥文件以 0600 权限落盘
func TestSavePrivateKeyFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不支持 POSIX 文件权限位")
	}

	_, privateKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	for name, save := range map[string]func(string) error{
		"pkcs1": func(path string) error { return SavePKCS1PrivateKeyToFile(privateKey, path) },
		"pkcs8": func(path string) error { return SavePKCS8PrivateKeyToFile(privateKey, path) },
	} {
		path := testdataFile(t, "rsax_perm_"+name+".pem")

		// 先以宽松权限创建同名文件，验证已存在文件也会被收紧
		if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
			t.Fatalf("准备旧文件失败: %v", err)
		}
		if err := save(path); err != nil {
			t.Fatalf("保存私钥失败: %v", err)
		}

		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("获取文件信息失败: %v", err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("私钥文件权限 = %04o, want 0600", perm)
		}
	}
}

// TestSavePrivateKeyTruncatesExistingFile 验证覆盖已存在私钥文件时先截断再写入
//   - 旧实现先用 O_TRUNC 打开，若随后 Chmod 失败会先毁掉旧私钥；
//     现在改为先收紧权限、再 Truncate，本用例锁定“不残留旧内容”这一行为
func TestSavePrivateKeyTruncatesExistingFile(t *testing.T) {
	t.Parallel()

	_, privateKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 旧内容远长于 PEM，未截断时一定会有剩余字节
	stale := bytes.Repeat([]byte("STALE-KEY"), 4096)

	for name, tc := range map[string]struct {
		save func(string) error
		load func(string) error
	}{
		"pkcs1": {
			save: func(path string) error { return SavePKCS1PrivateKeyToFile(privateKey, path) },
			load: func(path string) error { _, err := LoadPKCS1PrivateKeyFromFile(path); return err },
		},
		"pkcs8": {
			save: func(path string) error { return SavePKCS8PrivateKeyToFile(privateKey, path) },
			load: func(path string) error { _, err := LoadPKCS8PrivateKeyFromFile(path); return err },
		},
	} {
		path := testdataFile(t, "rsax_truncate_"+name+".pem")
		if err := os.WriteFile(path, stale, 0o600); err != nil {
			t.Fatalf("%s: 准备旧文件失败: %v", name, err)
		}
		if err := tc.save(path); err != nil {
			t.Fatalf("%s: 保存私钥失败: %v", name, err)
		}

		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: 获取文件信息失败: %v", name, err)
		}
		if fi.Size() >= int64(len(stale)) {
			t.Errorf("%s: 文件大小 = %d, 旧内容未被截断", name, fi.Size())
		}
		if err := tc.load(path); err != nil {
			t.Errorf("%s: 覆盖后重新加载失败: %v", name, err)
		}
	}
}
