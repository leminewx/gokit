// Package gokit 提供一组互相独立、可单独引入的 Go 基础工具包：文本、时间、文件、版本、
// 缓存、日志、颜色、网络、调度、数据结构与算法工具等。
//
// 各子包只依赖标准库与 gokit 内部的最小公共包（如 timex、stringx），不引入任何第三方
// 依赖（标准库增强包内部除外：nets/smtpx、nets/sshx、cryptos/password、nets/serial
// 使用 golang.org/x/crypto 与 golang.org/x/sys）。
package gokit

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// VERSION 是 gokit 的语义化版本号，格式为 MAJOR.MINOR.PATCH，与 Git 标签一致。
//
//   - MAJOR：不兼容的 API 变更
//   - MINOR：向后兼容的功能新增
//   - PATCH：向后兼容的问题修复
//
// 当前 v1.1.0 在 v1.0.0 首发能力之上新增 maths/percentile 泛型输入入口。
const VERSION = "1.1.0"

type (
	// Int 定义有符号整型的数据类型
	Int interface {
		int | int8 | int16 | int32 | int64
	}

	// Uint 定义无符号整型的数据类型
	Uint interface {
		uint | uint8 | uint16 | uint32 | uint64
	}

	// Float 定义浮点型的数据类型
	Float interface {
		float32 | float64
	}

	// Complex 定义复数的数据类型
	Complex interface {
		complex64 | complex128
	}

	// Number 定义数值型的数据类型（包括有符号整型、无符号整型、浮点型）
	Number interface {
		Int | Uint | Float
	}
)

// ListenQuitSignal 监听退出信号
func ListenQuitSignal(ctx context.Context, done func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	select {
	case <-ctx.Done():
	case <-sigCtx.Done():
	}

	if done != nil {
		return done()
	}

	return nil
}

// GetValueWithDefault 尝试返回有效值，若不存在则返回默认值
//   - flt: 默认值
func GetValueWithDefault[T any](flt T, values ...T) T {
	if len(values) == 0 {
		return flt
	}

	return values[0]
}

// CreateEmptyFile 创建一个空的文件
//   - 若中间路径不存在，则自动创建中间路径
func CreateEmptyFile(filename string) error {
	// 创建文件所在的所有父目录（如果不存在）
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 创建目标文件
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	return file.Close()
}

// IsDir 判断是否是目录
func IsDir(filename string) (bool, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		return true, nil
	}

	return false, nil
}
