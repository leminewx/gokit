package ini

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.configs.ini: "

var (
	// ErrReadFailed 表示读取配置文件失败
	ErrReadFailed = errors.New(_ERROR_PREFIX + "failed to read config file")
	// ErrWriteFailed 表示写入配置文件失败
	ErrWriteFailed = errors.New(_ERROR_PREFIX + "failed to write config file")
	// ErrParseFailed 表示配置内容格式非法
	ErrParseFailed = errors.New(_ERROR_PREFIX + "failed to parse config content")
	// ErrKeyNotFound 表示指定的节或键不存在
	ErrKeyNotFound = errors.New(_ERROR_PREFIX + "key not found")
	// ErrInvalidStruct 表示传入的值既不是结构体也不是结构体指针
	ErrInvalidStruct = errors.New(_ERROR_PREFIX + "value must be a struct or pointer to struct")
	// ErrInvalidTarget 表示反序列化目标不是非空结构体指针
	ErrInvalidTarget = errors.New(_ERROR_PREFIX + "invalid unmarshal target")
	// ErrConvertFailed 表示字符串转换为字段类型失败
	ErrConvertFailed = errors.New(_ERROR_PREFIX + "failed to convert value")
	// ErrUnsupportedType 表示字段类型不受支持
	ErrUnsupportedType = errors.New(_ERROR_PREFIX + "unsupported field type")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
