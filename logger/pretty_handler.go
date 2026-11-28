package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"gitee.com/leminewx/gokit/color"
	"gitee.com/leminewx/gokit/stringx"
)

const _DEFAULT_TIME_FORMAT = "2006-01-02 15:04:05.000"

// 编译期断言：Enabled/Handle/WithAttrs/WithGroup 是 slog.Handler 钉死的方法名
var _ slog.Handler = (*prettyHandler)(nil)

// presetAttr 记录预置属性及其生效时的 group 路径
//
// 属性在 WithAttrs 时绑定当前 group，之后再用 WithGroup 不应改变已有属性的前缀，
// 这与 log/slog 的 Handler 契约一致。
type presetAttr struct {
	groups []string
	attr   slog.Attr
}

// prettyHandler 是一个美观的文本格式 Handler
type prettyHandler struct {
	writer io.Writer // 输出目标
	opts   *Options  // 配置选项

	// 预格式化属性，由 WithAttrs/WithGroup 产生
	attrs  []presetAttr
	groups []string // 当前 group 路径

	lock *sync.Mutex // 与 WithAttrs/WithGroup 派生的 Handler 共享，保证同一 writer 上的并发写入不交错
}

// Options 定义配置
type Options struct {
	Level       slog.Leveler                                    // 日志级别
	TimeFormat  string                                          // 时间格式
	WithColor   bool                                            // 是否启用颜色
	WithSource  bool                                            // 是否添加源位置
	AttrSplit   string                                          // 属性分隔符
	AttrConnect string                                          // 属性连接符
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr // 属性替换
}

// NewPrettyHandler 创建新的 Handler
//   - opts 会被复制一份，填充默认值时不会修改调用方传入的 Options
func NewPrettyHandler(w io.Writer, opts *Options) slog.Handler {
	if w == nil {
		w = os.Stdout
	}

	effective := Options{}
	if opts != nil {
		effective = *opts
	}

	// 填充默认值，避免在 Handle 中修改共享 opts
	if effective.Level == nil {
		effective.Level = slog.LevelInfo
	}

	if effective.TimeFormat == "" {
		effective.TimeFormat = _DEFAULT_TIME_FORMAT
	}

	if effective.AttrConnect == "" {
		effective.AttrConnect = " => "
	}

	if effective.AttrSplit == "" {
		effective.AttrSplit = " "
	}

	return &prettyHandler{writer: w, opts: &effective, lock: &sync.Mutex{}}
}

// Enabled 判断是否处理该级别的日志
func (own *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= own.opts.Level.Level()
}

// Handle 是核心的日志处理与格式化方法
func (own *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	// 源码定位优先使用 slog.Record 自带的 PC（由 NewLogger 在用户调用边界抓取，已正确跨越 testing.tRunner 等反射边界，零额外栈遍历）；
	// 仅当用户直接用 slog.New(h) 且未开启 AddSource（r.PC==0）时，回退到自行栈遍历。
	var srcLine int
	var srcFile string
	if own.opts.WithSource {
		if r.PC != 0 {
			if fn := runtime.FuncForPC(r.PC); fn != nil {
				srcFile, srcLine = fn.FileLine(r.PC)
			}
		} else {
			srcFile, srcLine, _ = callerFrame()
		}
	}

	// 快照共享状态（构造后均不可变，无需持锁）
	groups := own.groups
	withColor := own.opts.WithColor
	timeFormat := own.opts.TimeFormat
	attrSplit := own.opts.AttrSplit
	attrConnect := own.opts.AttrConnect
	replaceAttr := own.opts.ReplaceAttr

	// ReplaceAttr 是用户回调，必须在锁外执行：回调内若用同一个 logger 再打日志，
	// sync.Mutex 不可重入会造成死锁。无 ReplaceAttr 时走零分配的直写快路径。
	var replaced []slog.Attr
	if replaceAttr != nil {
		replaced = make([]slog.Attr, 0, len(own.attrs)+r.NumAttrs())
		for _, pa := range own.attrs {
			replaced = append(replaced, applyReplace(replaceAttr, pa.groups, pa.attr))
		}
		r.Attrs(func(a slog.Attr) bool {
			replaced = append(replaced, applyReplace(replaceAttr, groups, a))
			return true
		})
	}

	// 写出与归还 builder 必须在同一把锁内完成：writer 本身未必并发安全（如 bytes.Buffer），
	// 而 stringx.PutBuilder 会 Reset 并放回池中，若先归还再写，其他 goroutine 复用的
	// builder 会覆盖同一块底层数组。
	own.lock.Lock()
	defer own.lock.Unlock()

	builder := stringx.GetBuilder()
	// defer 保证 writer panic 时锁与池内 builder 仍被正确释放，避免 logger 永久僵死
	defer stringx.PutBuilder(builder)

	// 1. 日志级别（可选上色，但级别文字始终输出）
	if withColor {
		colorizeLevel(builder, r.Level) // 根据级别上色
	} else {
		builder.WriteString(getLevelString(r.Level))
	}
	builder.WriteString(" - ")

	// 2. 时间
	if !r.Time.IsZero() {
		builder.WriteString(r.Time.Format(timeFormat))
	}

	// 3. 源代码位置 (可选，已在锁外抓取)
	if own.opts.WithSource && srcFile != "" {
		builder.WriteString(" - ")
		// filepath.Base 等价：取最后一个路径分隔符之后的部分
		if i := strings.LastIndexByte(srcFile, '/'); i >= 0 {
			builder.WriteString(srcFile[i+1:])
		} else if i := strings.LastIndexByte(srcFile, '\\'); i >= 0 {
			builder.WriteString(srcFile[i+1:])
		} else {
			builder.WriteString(srcFile)
		}

		builder.WriteByte(':')
		writeInt(builder, srcLine)
	}
	builder.WriteString(" - ")

	// 4. 日志消息
	builder.WriteString(r.Message)

	// 5. 属性 (key=value)，先输出预置属性（WithAttrs），再输出本次属性，边遍历边写入 builder，避免中间切片分配
	wroteAttr := false
	// writeAttr 写出单个属性
	//   - LogValuer 先 Resolve，否则 KindLogValuer 的 String() 返回空串
	//   - group 按 key.child=value 递归展开，同样因为 KindGroup 的 String() 返回空串
	var writeAttr func(a slog.Attr)
	writeAttr = func(a slog.Attr) {
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return
		}

		if a.Value.Kind() == slog.KindGroup {
			for _, sub := range a.Value.Group() {
				if a.Key == "" { // 匿名单组：直接内联子属性
					writeAttr(sub)
					continue
				}
				writeAttr(slog.Attr{Key: a.Key + "." + sub.Key, Value: sub.Value})
			}
			return
		}

		if !wroteAttr {
			builder.WriteString(attrConnect)
			wroteAttr = true
		} else {
			builder.WriteString(attrSplit)
		}

		builder.WriteString(a.Key)
		builder.WriteByte('=')
		builder.WriteString(a.Value.String())
	}

	// 预置属性
	if replaceAttr != nil {
		for _, a := range replaced {
			writeAttr(a)
		}
	} else {
		for _, pa := range own.attrs {
			writeAttr(applyReplace(nil, pa.groups, pa.attr))
		}
		r.Attrs(func(a slog.Attr) bool {
			writeAttr(applyReplace(nil, groups, a))
			return true
		})
	}

	builder.WriteString("\n")

	// 通过 unsafe 直接取 builder 底层 buf 传给 Write，避免 builder.String() 复制 + []byte 转换两次分配。
	buf := builderStringBytes(builder)
	_, err := own.writer.Write(buf)
	return err
}

// WithAttrs 返回带有预设属性的新 Handler
func (own *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return own
	}

	newAttrs := make([]presetAttr, len(own.attrs), len(own.attrs)+len(attrs))
	copy(newAttrs, own.attrs)
	for _, a := range attrs {
		newAttrs = append(newAttrs, presetAttr{groups: own.groups, attr: a})
	}

	return &prettyHandler{
		writer: own.writer,
		opts:   own.opts,
		attrs:  newAttrs,
		groups: own.groups,
		lock:   own.lock,
	}
}

// WithGroup 返回带有分组的新 Handler
func (own *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return own
	}

	newGroups := make([]string, len(own.groups)+1)
	copy(newGroups, own.groups)
	newGroups[len(own.groups)] = name

	return &prettyHandler{
		writer: own.writer,
		opts:   own.opts,
		attrs:  own.attrs,
		groups: newGroups,
		lock:   own.lock,
	}
}

// callerFrame 抓取用户代码的源文件与行号
//
// slog 不会替我们填充 r.PC（只有 slog.New 时开启 AddSource 才会），因此此处
// 自行抓取调用栈。为了不拖慢热路径，做了三重优化：
//  1. 仅在 WithSource 开启时调用（opt-in，绝大多数日志不付出代价）；
//  2. 在 Handle 持锁之前调用，避免栈遍历占用 sync.Mutex；
//  3. 优先用固定 skip 直取用户帧（零堆分配，栈上数组），仅当失效才回退全栈扫描。
//
// 调用深度（自本函数向上）在标准库中高度稳定：
//
//	skip=0  runtime.Callers
//	skip=1  callerFrame
//	skip=2  prettyHandler.Handle
//	skip=3  log/slog.(*Logger).log
//	skip=4  log/slog.(*Logger).Info / Warn / ...
//	skip=5  用户调用方  ← 目标
func callerFrame() (file string, line int, ok bool) {
	// 方案 A：固定 skip 直取用户帧（零堆分配，绝大多数情况命中）。
	// 自 slog.Info 起向上多探几步，凡命中已知内部包则继续上跳。
	const baseSkip = 4
	const maxSteps = 8
	var pc [1]uintptr
	for step := 0; step < maxSteps; step++ {
		n := runtime.Callers(baseSkip+step, pc[:])
		if n == 0 {
			break
		}

		fn := runtime.FuncForPC(pc[0])
		if fn == nil {
			break
		}

		name := fn.Name()
		if isInternalFrame(name) {
			continue
		}

		file, line = fn.FileLine(pc[0])
		return file, line, true
	}

	// 方案 B：全栈扫描兜底（仅在固定 skip 失效时走，正确优先于性能）。
	return callerFrameScan()
}

// isInternalFrame 判断该帧是否属于 gokit/slog/runtime 等内部实现，
// 这些帧不应作为"用户源码位置"输出。注意：测试时 testing 包亦属内部，
// 但用户的 TestXxx 函数不含 "testing."，仍会被正确识别为用户帧。
func isInternalFrame(name string) bool {
	return strings.Contains(name, "log/slog.") ||
		strings.Contains(name, "(*prettyHandler)") ||
		strings.Contains(name, "NewPrettyHandler") ||
		strings.Contains(name, "(*Logger).") ||
		strings.Contains(name, "logger.callerFrame") ||
		strings.Contains(name, "runtime.") ||
		strings.Contains(name, "testing.") ||
		strings.Contains(name, "sync.") ||
		strings.Contains(name, "bytes.") ||
		strings.Contains(name, "bufio.") ||
		strings.Contains(name, "reflect.") ||
		// 包路径中的 internal 段：形如 internal/xxx、net/internal/xxx。
		// 注意不能写成 "internal."——包路径用 / 分隔，带点的形式永远不会命中
		strings.Contains(name, "internal/")
}

// callerFrameScan 全栈扫描的兜底实现（仅在固定 skip 失效时走，性能较差但正确）。
// 使用栈上固定数组承接 PC，并逐帧用 FuncForPC 解析，避免 runtime.CallersFrames
// 构造 frames 对象带来的堆分配。
func callerFrameScan() (file string, line int, ok bool) {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(0, pcs[:])
	if n == 0 {
		return "", 0, false
	}

	for i := 0; i < n; i++ {
		fn := runtime.FuncForPC(pcs[i])
		if fn == nil {
			continue
		}
		if isInternalFrame(fn.Name()) {
			continue
		}
		file, line = fn.FileLine(pcs[i])
		return file, line, true
	}

	return "", 0, false
}

const (
	_LEVEL_DEBUG = "DEBUG"
	_LEVEL_INFO  = " INFO"
	_LEVEL_WARN  = " WARN"
	_LEVEL_ERROR = "ERROR"
)

// 辅助函数：返回级别的文字表示（定宽 5 字符，便于对齐）
func getLevelString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return _LEVEL_DEBUG
	case slog.LevelInfo:
		return _LEVEL_INFO
	case slog.LevelWarn:
		return _LEVEL_WARN
	case slog.LevelError:
		return _LEVEL_ERROR
	default:
		return level.String()
	}
}

// 辅助函数：颜色化级别
func colorizeLevel(builder *strings.Builder, level slog.Level) {
	// 使用 ANSI 颜色代码
	switch level {
	case slog.LevelDebug:
		color.New(color.FG_BRIGHT_CYAN, color.STYLE_BOLD).Fprint(builder, _LEVEL_DEBUG)
	case slog.LevelInfo:
		color.New(color.FG_BRIGHT_BLUE, color.STYLE_BOLD).Fprint(builder, _LEVEL_INFO)
	case slog.LevelWarn:
		color.New(color.FG_BRIGHT_YELLOW, color.STYLE_BOLD).Fprint(builder, _LEVEL_WARN)
	case slog.LevelError:
		color.New(color.FG_BRIGHT_RED, color.STYLE_BOLD).Fprint(builder, _LEVEL_ERROR)
	default:
		builder.WriteString(level.String())
	}
}

// writeInt 将十进制整数写入 builder（手写，避免 strconv 的临时分配）
func writeInt(b *strings.Builder, n int) {
	if n == 0 {
		b.WriteByte('0')
		return
	}

	if n < 0 {
		b.WriteByte('-')
		n = -n
	}

	var tmp [12]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}

	b.Write(tmp[i:])
}

// builderStringBytes 零分配地取得 strings.Builder 的底层字节切片。
// 注意：返回的切片在 builder 被 Reset/Put 之前有效，调用方须在 Write 完成后才归还 builder。
func builderStringBytes(b *strings.Builder) []byte {
	return unsafe.Slice(unsafe.StringData(b.String()), b.Len())
}

// applyReplace 应用 ReplaceAttr 并加上分组前缀
//   - groups 为属性生效时的分组路径，原样传给 ReplaceAttr（符合 slog.Handler 契约）
func applyReplace(replaceAttr func(groups []string, attr slog.Attr) slog.Attr, groups []string, a slog.Attr) slog.Attr {
	if replaceAttr != nil {
		a = replaceAttr(groups, a)
	}

	if len(groups) > 0 {
		a.Key = strings.Join(groups, ".") + "." + a.Key
	}

	return a
}
