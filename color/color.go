package color

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"gitee.com/leminewx/gokit/stringx"
)

const (
	_ENV_NO_COLOR = "NO_COLOR" // 环境变量：非空时禁用颜色输出
	_ENV_TERM     = "TERM"     // 环境变量：终端类型
	_TERM_DUMB    = "dumb"     // TERM=dumb 视为不支持的终端
)

var (
	// writerMu 保护 defaultWriter 的并发读写
	writerMu sync.RWMutex
	// defaultWriter 默认的输出流
	defaultWriter io.Writer = os.Stdout
	// noColor 全局开关：为 true 时所有带色输出降级为纯文本（适用于日志文件、非 TTY 环境）
	noColor atomic.Bool
)

// init 按 NO_COLOR 与 TERM=dumb 自动禁用颜色输出
func init() {
	if os.Getenv(_ENV_NO_COLOR) != "" || os.Getenv(_ENV_TERM) == _TERM_DUMB {
		noColor.Store(true)
	}
}

// SetOutput 设置默认输出流，传入 nil 时忽略
func SetOutput(w io.Writer) {
	if w == nil {
		return
	}

	writerMu.Lock()
	defaultWriter = w
	writerMu.Unlock()
}

// outputWriter 读取当前默认输出流
func outputWriter() io.Writer {
	writerMu.RLock()
	defer writerMu.RUnlock()

	return defaultWriter
}

// SetNoColor 设置是否禁用颜色输出
//   - 为 true 时所有带色输出降级为纯文本，适用于日志文件与非 TTY 环境
func SetNoColor(disabled bool) {
	noColor.Store(disabled)
}

// IsNoColor 返回当前是否已禁用颜色输出
func IsNoColor() bool {
	return noColor.Load()
}

type style []Code

func New(codes ...Code) style {
	return style(codes)
}

// Print 打印带颜色和样式的文本，不自动换行
func (own style) Print(msg string) {
	fmt.Fprint(outputWriter(), wrapStringWithANSI(msg, own...))
}

// Printf 打印带颜色和样式的文本，不自动换行
func (own style) Printf(format string, v ...any) {
	fmt.Fprint(outputWriter(), wrapFormatStringWithANSI(format, v, own...))
}

// Println 打印带颜色和样式的文本，并自动换行
func (own style) Println(msg string) {
	fmt.Fprintln(outputWriter(), wrapStringWithANSI(msg, own...))
}

// Sprint 返回一个带颜色和样式的字符串
func (own style) Sprint(msg string) string {
	return wrapStringWithANSI(msg, own...)
}

// Sprintf 返回一个带颜色和样式的字符串
func (own style) Sprintf(format string, v ...any) string {
	return wrapFormatStringWithANSI(format, v, own...)
}

// Sprintln 返回一个带颜色和样式的字符串，并自动换行
func (own style) Sprintln(msg string) string {
	return wrapStringWithANSI(msg, own...) + "\n"
}

// Fprint 打印带颜色和样式的文本到指定的 io.Writer
func (own style) Fprint(w io.Writer, msg string) {
	fmt.Fprint(w, wrapStringWithANSI(msg, own...))
}

// Fprintf 打印带颜色和样式的文本到指定的 io.Writer
func (own style) Fprintf(w io.Writer, format string, v ...any) {
	fmt.Fprint(w, wrapFormatStringWithANSI(format, v, own...))
}

// Fprintln 打印带颜色和样式的文本到指定的 io.Writer，并自动换行
func (own style) Fprintln(w io.Writer, msg string) {
	fmt.Fprintln(w, wrapStringWithANSI(msg, own...))
}

// wrapStringWithANSI 用 ANSI 转义序列包裹消息
func wrapStringWithANSI(msg string, codes ...Code) string {
	// 纯文本模式或空代码：直接返回原文
	if IsNoColor() || len(codes) == 0 {
		return msg
	}

	builder := stringx.GetBuilder()
	appendANSIBuilder(builder, codes...)
	builder.WriteString(msg)
	builder.WriteString("\x1b[0m")
	result := builder.String()
	stringx.PutBuilder(builder)
	return result
}

// wrapFormatStringWithANSI 格式化字符串并包裹 ANSI 代码
func wrapFormatStringWithANSI(format string, v []any, codes ...Code) string {
	if IsNoColor() || len(codes) == 0 {
		return fmt.Sprintf(format, v...)
	}

	builder := stringx.GetBuilder()
	appendANSIBuilder(builder, codes...)
	fmt.Fprintf(builder, format, v...)
	builder.WriteString("\x1b[0m")
	result := builder.String()
	stringx.PutBuilder(builder)
	return result
}

// appendANSIBuilder 将 ANSI 转义前缀 \x1b[...m 写入 builder
func appendANSIBuilder(builder *strings.Builder, codes ...Code) {
	builder.WriteString("\x1b[")
	for i, code := range codes {
		if i > 0 {
			builder.WriteByte(';')
		}

		// 真彩色（24 位）展开为 38;2;r;g;b 或 48;2;r;g;b
		if r, g, b, fg, ok := code.isRGB(); ok {
			if fg {
				builder.WriteString("38;2;")
			} else {
				builder.WriteString("48;2;")
			}
			builder.WriteString(strconv.Itoa(int(r)))
			builder.WriteByte(';')
			builder.WriteString(strconv.Itoa(int(g)))
			builder.WriteByte(';')
			builder.WriteString(strconv.Itoa(int(b)))
			continue
		}

		builder.WriteString(strconv.Itoa(int(code)))
	}
	builder.WriteByte('m')
}
