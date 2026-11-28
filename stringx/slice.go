package stringx

import (
	"strconv"
	"strings"
	"unsafe"

	"gitee.com/leminewx/gokit"
)

// JoinInts 使用分隔符拼接有符号整型切片
func JoinInts[T gokit.Int](items []T, sep string) string {
	return joinWith(items, sep, func(builder *strings.Builder, item T) {
		builder.WriteString(strconv.FormatInt(int64(item), 10))
	})
}

// JoinUints 使用分隔符拼接无符号整型切片
func JoinUints[T gokit.Uint](items []T, sep string) string {
	return joinWith(items, sep, func(builder *strings.Builder, item T) {
		builder.WriteString(strconv.FormatUint(uint64(item), 10))
	})
}

// joinWith 按分隔符拼接任意切片，format 负责把单个元素写入 builder
func joinWith[T any](items []T, sep string, format func(builder *strings.Builder, item T)) string {
	if len(items) == 0 {
		return ""
	}

	var builder strings.Builder
	format(&builder, items[0])
	for _, item := range items[1:] {
		builder.WriteString(sep)
		format(&builder, item)
	}

	return builder.String()
}

// ParseNumbers 将十进制整数字符串切片解析为 T 类型的整型切片
//   - T 仅支持有符号/无符号整型，并按 T 的位宽做溢出校验
//   - 任一元素解析失败时返回 ErrInvalidNumber，并通过错误链保留 strconv 的原始错误
func ParseNumbers[T gokit.Int | gokit.Uint](items []string) ([]T, error) {
	values := make([]T, 0, len(items))
	for i, item := range items {
		value, err := parseNumber[T](strings.TrimSpace(item))
		if err != nil {
			return nil, wrapError(ErrInvalidNumber, "index %d (%q): %w", i, item, err)
		}

		values = append(values, value)
	}

	return values, nil
}

// parseNumber 按 T 的符号属性与位宽解析单个十进制整数字符串
func parseNumber[T gokit.Int | gokit.Uint](text string) (T, error) {
	var zero T
	bitSize := int(unsafe.Sizeof(zero)) * 8

	switch any(zero).(type) {
	case int, int8, int16, int32, int64:
		value, err := strconv.ParseInt(text, 10, bitSize)
		if err != nil {
			return zero, err
		}

		return T(value), nil
	default:
		value, err := strconv.ParseUint(text, 10, bitSize)
		if err != nil {
			return zero, err
		}

		return T(value), nil
	}
}
