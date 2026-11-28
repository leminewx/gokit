// Package logger 提供一个基于标准库 log/slog 的、美观且可定制的文本日志
// Handler（PrettyHandler），用于替代 slog 默认的手写文本格式。
//
// 设计目标：
//   - 完全兼容 Go 标准库 log/slog：实现 slog.Handler 接口，可直接传入
//     slog.New(handler) 使用；
//   - 零第三方依赖，仅在内部复用 gokit 的 color 包；
//   - 线程安全：所有共享状态通过 sync.Mutex 保护，可在多 goroutine 下并发
//     写入；IO 写在锁外执行以降低锁竞争；
//   - 高性能：热路径使用栈上 strings.Builder（零对象池往返开销），属性边遍历
//     边写入避免中间切片分配，无 WithAttrs 时零预置属性复制。典型 INFO 日志
//     约 0.8µs/op、11 次分配（基准见 pretty_handler_test.go）。
//
// 快速上手：
//
//	h := logger.NewPrettyHandler(os.Stdout, &logger.Options{
//	    Level:     slog.LevelDebug,
//	    WithColor: true,
//	})
//	log := slog.New(h)
//	log.Info("服务启动", "host", "127.0.0.1", "port", 8080)
//
// 输出示例（带颜色）：
//
//	INFO  - 2026-08-28 22:31:44.416 - 服务启动 | host=127.0.0.1 port=8080
//
// 配置项（Options）：
//   - Level slog.Leveler：最低输出级别，低于该级别的日志被过滤；
//   - WithColor bool：是否对级别文字进行 ANSI 着色（DEBUG 青、INFO 蓝、
//     WARN 黄、ERROR 红）；
//   - WithSource bool：是否追加源文件与行号（file:line）；
//   - TimeFormat string：时间格式（默认 "2006-01-02 15:04:05.000"）；
//   - AttrSplit / AttrConnect string：多个属性之间的分隔符（默认空格）
//     与 key/value 之间的连接符（默认 "="）；
//   - ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr：
//     标准 slog 属性替换钩子，可用于脱敏或重命名。
//
// 与 slog 生态的契合：
//   - 支持 slog 的 With(key, value)（通过 WithAttrs 持久化属性）；
//   - 支持 WithGroup(name)（分组名作为属性 key 前缀，如 "req.id=abc"）；
//   - 支持运行时 Caller 定位（基于 r.PC，比硬编码 runtime.Caller 更准确）。
//
// 注意：
//   - WithColor 仅建议在 TTY/终端启用；写入文件或日志收集系统时建议关闭，
//     或同步将 gokit 的 color.NoColor 置为 true 以避免转义序列污染。
package logger

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// Logger 是对 slog.Handler 的轻量包装，相比直接用 slog.New(handler)，
// 它在用户调用边界（Info/Warn/...）自行捕获调用栈 PC，从而可靠地解析
// 源码位置（file:line），且能正确跨越 testing.tRunner、反射等边界——
// 这是 slog.New(handler) 默认不会填充 r.PC（除非开启 AddSource）所做不到的。
//
// 用法：
//
//	log := logger.NewLogger(logger.NewPrettyHandler(os.Stdout, &logger.Options{
//	    WithColor:  true,
//	    WithSource: true,
//	}))
//	log.Info("服务启动", "host", "127.0.0.1", "port", 8080)
type Logger struct {
	handler slog.Handler
}

// NewLogger 基于一个 slog.Handler 创建带源码定位能力的 Logger。
func NewLogger(handler slog.Handler) *Logger {
	return &Logger{handler: handler}
}

// log 是各级别方法的统一入口，负责在用户调用边界抓取 PC 并构造记录。
func (own *Logger) log(level slog.Level, msg string, args ...any) {
	if !own.handler.Enabled(context.Background(), level) {
		return
	}
	// 在用户调用边界抓取 PC。调用链为：runtime.Callers ← log ← 级别方法 ← 用户。
	// 级别方法均标记 //go:noinline，保证每一层都存在独立的物理栈帧，
	// 否则内联后物理 PC 会归属调用方，而 CallersFrames 又把它展开成
	// (*Logger).Info 逻辑帧，导致日志位置被错认成 logger.go。
	const baseSkip = 2
	const maxSteps = 6
	var pcs [1]uintptr
	var pc uintptr
	for step := 0; step < maxSteps; step++ {
		if runtime.Callers(baseSkip+step, pcs[:]) == 0 {
			break
		}
		fn := runtime.FuncForPC(pcs[0])
		if fn == nil {
			break
		}
		if isInternalFrame(fn.Name()) || strings.Contains(fn.Name(), "(*Logger).") {
			continue
		}
		pc = pcs[0]
		break
	}
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	_ = own.handler.Handle(context.Background(), r)
}

// Debug 记录 DEBUG 级别日志。
//
//go:noinline
func (own *Logger) Debug(msg string, args ...any) { own.log(slog.LevelDebug, msg, args...) }

// Info 记录 INFO 级别日志。
//
//go:noinline
func (own *Logger) Info(msg string, args ...any) { own.log(slog.LevelInfo, msg, args...) }

// Warn 记录 WARN 级别日志。
//
//go:noinline
func (own *Logger) Warn(msg string, args ...any) { own.log(slog.LevelWarn, msg, args...) }

// Error 记录 ERROR 级别日志。
//
//go:noinline
func (own *Logger) Error(msg string, args ...any) { own.log(slog.LevelError, msg, args...) }

// Log 记录指定级别的日志。
//
//go:noinline
func (own *Logger) Log(level slog.Level, msg string, args ...any) { own.log(level, msg, args...) }

// With 返回带有预设属性的 Logger。
func (own *Logger) With(args ...any) *Logger {
	h := own.handler
	if len(args) > 0 {
		attrs := argsToAttrs(args...)
		h = h.WithAttrs(attrs)
	}
	return &Logger{handler: h}
}

// WithGroup 返回带有分组的 Logger。
func (own *Logger) WithGroup(name string) *Logger {
	return &Logger{handler: own.handler.WithGroup(name)}
}

// GetHandler 返回底层 slog.Handler（便于与 slog 生态互操作）。
func (own *Logger) GetHandler() slog.Handler { return own.handler }

// argsToAttrs 将 key-value 形式的参数转换为 slog.Attr 列表，
// 复用 slog.Record.Add 的官方解析规则（成对为 key=value，奇数末尾补 nil）。
func argsToAttrs(args ...any) []slog.Attr {
	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0)
	r.Add(args...)
	attrs := make([]slog.Attr, 0, len(args))
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	return attrs
}
