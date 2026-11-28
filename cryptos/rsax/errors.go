package rsax

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.cryptos.rsax: "

var (
	// ErrInvalidKeyBits 表示密钥位数小于允许的最小值
	ErrInvalidKeyBits = errors.New(_ERROR_PREFIX + "invalid key size")
	// ErrNilKey 表示传入的密钥为空
	ErrNilKey = errors.New(_ERROR_PREFIX + "key is nil")
	// ErrGenerateKeyPair 表示生成密钥对失败
	ErrGenerateKeyPair = errors.New(_ERROR_PREFIX + "failed to generate key pair")
	// ErrCreateFile 表示创建文件失败
	ErrCreateFile = errors.New(_ERROR_PREFIX + "failed to create file")
	// ErrReadFile 表示读取文件失败
	ErrReadFile = errors.New(_ERROR_PREFIX + "failed to read file")
	// ErrEncodePEM 表示 PEM 编码失败
	ErrEncodePEM = errors.New(_ERROR_PREFIX + "failed to encode PEM")
	// ErrDecodePEM 表示 PEM 解码失败或块类型不匹配
	ErrDecodePEM = errors.New(_ERROR_PREFIX + "failed to decode PEM")
	// ErrMarshalKey 表示密钥序列化失败
	ErrMarshalKey = errors.New(_ERROR_PREFIX + "failed to marshal key")
	// ErrParseKey 表示密钥解析失败
	ErrParseKey = errors.New(_ERROR_PREFIX + "failed to parse key")
	// ErrNotRSAKey 表示解析结果不是 RSA 密钥
	ErrNotRSAKey = errors.New(_ERROR_PREFIX + "key is not an RSA key")
	// ErrEncrypt 表示加密失败
	ErrEncrypt = errors.New(_ERROR_PREFIX + "failed to encrypt")
	// ErrDecrypt 表示解密失败
	ErrDecrypt = errors.New(_ERROR_PREFIX + "failed to decrypt")
	// ErrSignFailed 表示签名失败
	ErrSignFailed = errors.New(_ERROR_PREFIX + "failed to sign")
	// ErrVerifyFailed 表示签名验证失败
	ErrVerifyFailed = errors.New(_ERROR_PREFIX + "failed to verify signature")
	// ErrUnsupportedHash 表示哈希算法不可用
	ErrUnsupportedHash = errors.New(_ERROR_PREFIX + "unsupported hash function")
	// ErrDecodeBase64 表示 Base64 解码失败
	ErrDecodeBase64 = errors.New(_ERROR_PREFIX + "failed to decode base64")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
