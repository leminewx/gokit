// Package stringx 提供文本与字符串处理工具：命名风格转换、ANSI 转义序列剪除、字节数与整型切片格式化。
package stringx

import (
	"fmt"
	"strings"
)

// ConvertToSnakeCase 将驼峰命名转换为蛇形（下划线）命名
//   - HelloWorld → hello_world
//   - HTTPServer → http_server（连续大写按一个单词处理）
//   - UserID → user_id
//   - 空字符串返回空字符串；已含下划线或空格的字符保持原样
func ConvertToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	var builder strings.Builder
	builder.Grow(len(s) + len(s)/4)

	for i, r := range runes {
		if r < 'A' || r > 'Z' {
			builder.WriteRune(r)
			continue
		}

		// 需要插入下划线的场景：前一个字符为小写/数字，或前一个大写但后一个是小写（单词首字母）
		if i > 0 {
			prev := runes[i-1]
			switch {
			case prev >= 'a' && prev <= 'z', prev >= '0' && prev <= '9':
				builder.WriteByte('_')
			case prev >= 'A' && prev <= 'Z' && i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z':
				builder.WriteByte('_')
			}
		}

		builder.WriteRune(r + ('a' - 'A'))
	}

	return builder.String()
}

// HumanizeBytes 将字节数转换为人类可读的格式（例如：512 B、1.00 KiB、1.23 MiB）
//   - 以 1024 为进制，单位使用 IEC 前缀（KiB/MiB/GiB/TiB/PiB），保留两位小数
//   - 小于 1024 字节（含负数）按字节原值输出，不做单位换算
func HumanizeBytes(bytes int64) string {
	switch {
	case bytes < 1<<10:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1<<20:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/(1<<10))
	case bytes < 1<<30:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/(1<<20))
	case bytes < 1<<40:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/(1<<30))
	case bytes < 1<<50:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/(1<<40))
	default:
		return fmt.Sprintf("%.2f PiB", float64(bytes)/(1<<50))
	}
}
