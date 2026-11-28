package stringx

import (
	"bytes"
	"regexp"
	"strings"
)

// ansiEscapePattern 匹配 ANSI 转义序列（颜色、光标控制等）
var ansiEscapePattern = regexp.MustCompile(`\x1B\[[0-?]*[ -/]*[@-~]`)

// StripANSIString 剪除字符串中的 ANSI 转义序列
//   - 不含 ESC 时直接返回原字符串，避免正则匹配开销
func StripANSIString(s string) string {
	if s == "" || !strings.ContainsRune(s, '\x1B') {
		return s
	}

	return ansiEscapePattern.ReplaceAllString(s, "")
}

// StripANSIBytes 剪除字节序列中的 ANSI 转义序列
//   - 入参为 nil 时返回 nil；不含 ESC 时直接返回原切片
func StripANSIBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	if !bytes.ContainsRune(data, '\x1B') {
		return data
	}

	return ansiEscapePattern.ReplaceAll(data, nil)
}
