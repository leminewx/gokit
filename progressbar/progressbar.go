// Package progressbar 提供一个用于终端的、单行动画式进度条（Progressbar），
// 适用于文件上传/下载、批量任务、安装升级等需要展示实时进度的命令行场景。
//
// 设计目标：
//   - 零第三方依赖（终端尺寸探测除外：内部使用 golang.org/x/sys）；
//   - 并发安全：所有共享状态由 sync.RWMutex 保护，可在多 goroutine 下并发 Add；
//   - 低开销：渲染默认按 100ms 节流，避免高频更新刷爆终端；热路径复用
//     strings.Builder 对象池，避免每帧重复分配；
//   - 自适应宽度：按实际输出目标探测终端宽度，自动调整进度条长度
//     （探测按 500ms 节流；输出非终端时直接使用默认宽度）；
//   - 可配置输出：默认写 os.Stdout，可通过 SetWriter 重定向（测试/日志文件）。
//
// 渲染效果（单行刷新，含指示图标、进度条、百分比、进度、速度、剩余时间）：
//
//	⣷ [████████████··········]  48.15%  |  512.00 MiB/1.00 GiB  |  12.30 MiB/s  |  42s
//
// 典型用法：
//
//	bar := progressbar.New(totalBytes)
//	bar.SetType(progressbar.TYPE_BYTES)
//	bar.SetPrefix("下载：")
//	bar.SetStyle(progressbar.StyleSquarePoint)
//
//	for {
//	    n, err := readChunk()
//	    if err != nil { // 出错时手动收尾，保证光标恢复
//	        bar.Finish()
//	        break
//	    }
//	    bar.Add(int64(n))
//	    if bar.IsFinished() {
//	        break
//	    }
//	}
//
// 主要 API：
//   - New(total)                               创建进度条（total 应 > 0）
//   - Add(increment)                           累加进度（increment <= 0 会被忽略）
//   - Update(current)                          直接设置当前进度（仅接受递增，回退会被忽略）
//   - GetCount() / GetTotal() / GetProgress()   查询当前进度 / 总量 / 进度
//   - IsFinished()                             是否已结束
//   - Finish()                                 手动结束（异常中断时用于恢复光标并收尾）
//
// 可选配置（均无返回值，逐条调用即可）：
//   - SetPrefix(string)         左侧前缀文本（支持中文，宽度按显示列计算）
//   - SetStyle(Style)           进度条样式，见下方 Style 常量
//   - SetType(Type)             TYPE_BYTES（按字节单位）或 TYPE_NUMBER（按数量）
//   - SetSpinners([]rune)       自定义指示图标集（空切片会被忽略）
//   - SetWriter(io.Writer)      输出目标，默认 os.Stdout
//   - SetFinish(func)           完成时的收尾文案生成器
//
// 内置样式（Style）与类型（Type）：
//
//	StyleSquarePoint  ■·     TYPE_BYTES   按字节展示（B/KiB/MiB/GiB…，1024 进制）
//	StyleSquareSpace  ■␣     TYPE_NUMBER  按数量展示（123/1000）
//	StyleBarPoint     █·
//	StyleBarSpace     █␣
//	StyleRhombus       ▰▱
//
// 注意：
//   - 渲染含 ANSI 转义序列（\033[2K\r 清行、\033[?25l/h 隐藏/显示光标），
//     仅适用于支持 ANSI 的终端；写入普通文件时建议配合 SetWriter 并按需
//     关闭光标控制；
//   - total <= 0 时百分比按 0 处理，不会产生 NaN；
//   - Finish 与自动完成均保证只执行一次，重复调用不会产生额外输出。
package progressbar

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitee.com/leminewx/gokit/speedometer"
	"gitee.com/leminewx/gokit/stringx"
)

// Progressbar 定义进度条的数据结构
type (
	Progressbar struct {
		total      int64 // 进度总量
		current    int64 // 当前进度量
		isFinished bool  // 是否完成

		writer     io.Writer // 输出目标，默认 os.Stdout
		hideCursor bool      // 隐藏光标
		autoResize bool      // 自动调整进度条的宽度
		termWidth  int       // 终端宽度

		// 布局缓存：total/prefix/type 未变化时无需重算宽度
		layoutDirty     bool      // 布局是否需要重新计算
		layoutType      Type      // 上次布局所依据的类型
		layoutPrefix    string    // 上次布局所依据的前缀
		prefixWidth     int       // 前缀的显示宽度（中文按 2 列计）
		totalWidth      int       // 进度区+速度区的预留宽度（仅依赖 total 与 barType）
		termChecked     time.Time // 上次查询终端宽度的时间
		barMaxWidth     int       // 进度条的最大宽度
		processMaxWidth int       // 进度的最大宽度
		speedMaxWidth   int       // 速度的最大宽度

		startTime     time.Time     // 进度开始时间
		lastRender    time.Time     // 最近渲染时间
		renderTimeout time.Duration // 单次渲染超时时间

		barType      Type   // 进度条类型
		barStyle     Style  // 进度条样式
		barPrefix    string // 进度条自定义前缀
		spinnerIndex int    // 上一个指示图标索引
		spinners     []rune // 进度加载的指示图标

		speedometer speedometer.Speedometer // 计速器

		callAfterFinish func(prefix string, cost time.Duration) string // 进度加载完成后调用

		lock sync.RWMutex
	}

	// Type 进度条类型
	Type int8

	// Style 进度条样式，[0] 为已完成字符、[1] 为未完成字符
	Style [2]rune
)

const (
	TYPE_BYTES  = iota // 数据类型
	TYPE_NUMBER        // 数量类型
)

// 内置进度条样式（数组不能定义为常量，故使用包级变量）
var (
	StyleSquarePoint Style = [2]rune{'■', '·'}
	StyleSquareSpace Style = [2]rune{'■', ' '}
	StyleBarPoint    Style = [2]rune{'█', '·'}
	StyleBarSpace    Style = [2]rune{'█', ' '}
	StyleRhombus     Style = [2]rune{'▰', '▱'}
)

const (
	_DEFAULT_TERMINAL_SIZE = 80 // 默认终端宽度（无法获取终端尺寸时使用）

	_BAR_MIN_SIZE = 10  // 进度条最小长度
	_BAR_MAX_SIZE = 100 // 进度条最大长度

	// 进度条右侧固定内容的预留宽度：
	// " ["" ] " + 百分比(6)+"% " + " | " + 进度 + " | " + 速度 + " | " + 剩余时间
	_BAR_RIGHT_MARGIN = 30

	// 终端宽度的最小刷新间隔：终端尺寸变化不频繁，无需每帧查询
	_TERMINAL_REFRESH_INTERVAL = 500 * time.Millisecond
)

var (
	defaultStyle    = StyleBarPoint
	defaultSpinners = []rune{'⣷', '⣯', '⣟', '⡿', '⢿', '⣻', '⣽', '⣾'}
)

// New 新建一个进度条
//   - total 为进度总量，必须 > 0，否则渲染时无法计算百分比（内部按 0 处理并直接结束）
func New(total int64) *Progressbar {
	bar := &Progressbar{
		total:         total,
		writer:        os.Stdout,
		startTime:     time.Now(),
		renderTimeout: 100 * time.Millisecond,
		autoResize:    true,
		hideCursor:    true,
		barStyle:      defaultStyle,
		spinners:      defaultSpinners,
		speedometer:   speedometer.New(3 * time.Second),
		callAfterFinish: func(prefix string, cost time.Duration) string {
			return fmt.Sprintf("\033[2K\r%s 🎉🎉🎉  \033[32mCompleted! Cost: %s\033[0m\n", prefix, cost)
		},
	}

	// 隐藏光标
	if bar.hideCursor {
		fmt.Fprint(bar.writer, "\033[?25l")
	}

	// 初始化进度条最大宽度
	bar.initBarMaxWidth()
	return bar
}

// SetPrefix 设置进度条的前缀（显示在进度条左侧）
//   - prefix ⣾ [■■■■■■■■■■·······]
func (own *Progressbar) SetPrefix(prefix string) {
	own.lock.Lock()
	own.barPrefix = prefix
	own.layoutDirty = true // 前缀变化影响布局，需重新计算宽度
	own.lock.Unlock()
}

// SetSpinners 设置进度条的指示图标集
//   - 默认指示图标集：['⣷', '⣯', '⣟', '⡿', '⢿', '⣻', '⣽', '⣾']
func (own *Progressbar) SetSpinners(spinners []rune) {
	own.lock.Lock()
	// 空切片会导致后续取模除零，故忽略非法输入；复制一份避免调用方后续修改造成竞态
	if len(spinners) > 0 {
		own.spinners = append([]rune(nil), spinners...)
	}
	own.lock.Unlock()
}

// SetStyle 设置进度条的样式
//   - 可使用内置常量（StyleBarPoint 等），或自定义 [已完成字符, 未完成字符]
func (own *Progressbar) SetStyle(style Style) {
	own.lock.Lock()
	own.barStyle = style
	own.lock.Unlock()
}

// SetType 设置进度条的类型
//   - 默认：TYPE_BYTES
func (own *Progressbar) SetType(typ Type) {
	own.lock.Lock()
	own.barType = typ
	own.layoutDirty = true // 类型变化影响进度/速度的显示宽度
	own.lock.Unlock()
}

// SetWriter 设置输出目标（默认 os.Stdout）
//   - 传入 io.Discard 可静默运行；非终端（如文件、管道）建议同时关闭 hideCursor
func (own *Progressbar) SetWriter(w io.Writer) {
	own.lock.Lock()
	if w != nil {
		own.writer = w
	}
	own.lock.Unlock()
}

// SetFinish 进度条完成后调用该函数
//   - 清空当前行使用：\033[2K
func (own *Progressbar) SetFinish(finish func(prefix string, cost time.Duration) string) {
	own.lock.Lock()
	own.callAfterFinish = finish
	own.lock.Unlock()
}

// GetCount 返回当前已完成的进度量
func (own *Progressbar) GetCount() int64 {
	own.lock.RLock()
	current := own.current
	own.lock.RUnlock()
	return current
}

// GetTotal 返回进度总量
func (own *Progressbar) GetTotal() int64 {
	own.lock.RLock()
	total := own.total
	own.lock.RUnlock()
	return total
}

// GetProgress 返回当前进度百分比（0.0 ~ 1.0）
//   - total <= 0 时返回 0，避免产生 NaN/Inf
func (own *Progressbar) GetProgress() float64 {
	own.lock.RLock()
	defer own.lock.RUnlock()

	if own.total <= 0 {
		return 0
	}

	return float64(own.current) / float64(own.total)
}

// IsFinished 检查进度是否完成
func (own *Progressbar) IsFinished() bool {
	own.lock.RLock()
	isOver := own.isFinished
	own.lock.RUnlock()
	return isOver
}

// Add 增加新完成的进度
//   - increment <= 0 时直接返回：既不污染计速器，也不会误触发完成渲染
func (own *Progressbar) Add(increment int64) {
	if increment <= 0 {
		return
	}

	own.lock.Lock()

	own.current += increment
	if own.current > own.total {
		own.current = own.total
	}

	own.speedometer.Add(int(increment))
	own.render()

	own.lock.Unlock()
}

// Update 更新当前的总进度
//   - 仅接受递增的值：回退会污染计速器与预估时间，故被忽略
func (own *Progressbar) Update(current int64) {
	own.lock.Lock()

	if current > own.total {
		current = own.total
	}
	if current <= own.current {
		own.lock.Unlock()
		return
	}

	own.speedometer.Add(int(current - own.current))
	own.current = current
	own.render()

	own.lock.Unlock()
}

// Finish 手动结束进度条（用于中断/异常场景），保证光标恢复与收尾输出只执行一次
func (own *Progressbar) Finish() {
	own.lock.Lock()
	if !own.isFinished {
		own.finish()
	}

	own.lock.Unlock()
}

// render 渲染（调用方必须持有写锁）
func (own *Progressbar) render() {
	if own.isFinished {
		return
	}

	// 渲染节流：未完成且距上次渲染不足 renderTimeout 时跳过，
	// 避免高频 Add 导致终端被刷爆（time.Now 是此处唯一不可避免的开销）
	now := time.Now()
	if now.Sub(own.lastRender) < own.renderTimeout && own.current < own.total {
		return
	}
	own.lastRender = now

	// 仅在需要时重算布局（total/prefix/type 变化或开启自适应宽度）
	if own.layoutDirty || own.autoResize {
		own.initBarMaxWidth()
	}

	// 计算进度百分比（total <= 0 视为已完成，避免除零/NaN）
	percent := float64(0)
	if own.total > 0 {
		percent = float64(own.current) / float64(own.total)
		if percent > 1 {
			percent = 1
		}
	}

	// 计算已用时间（精确到秒）
	elapsed := time.Since(own.startTime).Round(time.Second)
	// 计算预计总耗时与剩余时间（percent 极小时预估无意义，置 0）
	remaining := time.Duration(0)
	if percent > 0 {
		estimated := time.Duration(float64(elapsed) / percent).Round(time.Second)
		if estimated > elapsed {
			remaining = (estimated - elapsed).Round(time.Second)
		}
	}

	// 构建进度条
	builder := stringx.GetBuilder()
	builder.WriteString("\033[2K\r")
	builder.WriteString(own.barPrefix)
	builder.WriteByte(' ')
	if n := len(own.spinners); n > 0 {
		builder.WriteRune(own.spinners[own.spinnerIndex%n])
		own.spinnerIndex = (own.spinnerIndex + 1) % n
	}

	builder.WriteString(" [")
	filled := int(math.Round(float64(own.barMaxWidth) * percent))
	// 直接循环写入，避免 strings.Repeat 每帧产生两个临时字符串
	for i := 0; i < filled; i++ {
		builder.WriteRune(own.barStyle[0])
	}
	for i := filled; i < own.barMaxWidth; i++ {
		builder.WriteRune(own.barStyle[1])
	}
	builder.WriteString("] ")

	// 写入百分比（替代 fmt.Sprintf，避免格式化开销与分配）
	writePercent(builder, percent*100)
	builder.WriteString(" | ")
	own.calcProcess(builder)
	builder.WriteString(" | ")
	own.calcSpeed(builder)
	builder.WriteString(" | ")
	builder.WriteString(remaining.String())

	output := builder.String()
	builder.Reset()
	stringx.PutBuilder(builder)

	// 按 rune 截断超宽内容（避免中文/emoji 被按字节截断成乱码）
	output = truncateRunes(output, own.termWidth)

	// 先快照输出目标，再在锁外写 IO：避免与 SetWriter 并发读写同一字段
	writer := own.writer
	own.lock.Unlock()
	fmt.Fprint(writer, output)
	own.lock.Lock()

	if own.current >= own.total {
		own.finish()
	}
}

// writePercent 以 "%6.2f%% " 的格式写入百分比（零 fmt 开销、零额外分配）
func writePercent(builder *strings.Builder, percent float64) {
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	// 放大 100 倍并四舍五入，得到两位小数的整数表示（如 1234 -> 12.34）
	v := int(percent*100 + 0.5)
	intPart := v / 100
	fracPart := v % 100

	var buf [16]byte
	var tmp [8]byte

	// 整数部分
	i := len(tmp)
	if intPart == 0 {
		i--
		tmp[i] = '0'
	}
	for intPart > 0 {
		i--
		tmp[i] = byte('0' + intPart%10)
		intPart /= 10
	}

	// 右对齐到宽度 6
	pad := 6 - (len(tmp) - i)
	n := 0
	for p := 0; p < pad; p++ {
		buf[n] = ' '
		n++
	}

	copy(buf[n:], tmp[i:])
	n += len(tmp) - i
	buf[n] = '.'
	n++
	buf[n] = byte('0' + fracPart/10)
	n++
	buf[n] = byte('0' + fracPart%10)
	n++
	buf[n] = '%'
	n++
	buf[n] = ' '
	n++
	builder.Write(buf[:n])
}

// truncateRunes 按 rune（字符）而非字节截断字符串，避免多字节字符被切断
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}

	count := 0
	for i := range s {
		if count >= max {
			return s[:i]
		}
		count++
	}
	return s
}

// terminalFileOf 尝试将输出目标还原为 *os.File
//   - 仅当 writer 直接是 *os.File（或极简包装）时才能探测终端宽度；
//     其余情况（bytes.Buffer、io.Discard、网络流等）返回 nil，由调用方回落默认宽度
func terminalFileOf(w io.Writer) *os.File {
	if f, ok := w.(*os.File); ok {
		return f
	}
	return nil
}

// initBarMaxWidth 计算进度条可用宽度
//   - 终端宽度通过系统调用获取（Windows 上为 ioctl），成本较高；终端尺寸变化
//     并不频繁，故按 _TERMINAL_REFRESH_INTERVAL 节流刷新，而非每帧查询。
func (own *Progressbar) initBarMaxWidth() {
	// 获取终端宽度（受刷新间隔保护，避免每帧执行一次系统调用）
	// 宽度取决于实际输出目标：重定向到文件/管道时不是终端，直接回落默认宽度
	now := time.Now()
	if own.termWidth == 0 || now.Sub(own.termChecked) >= _TERMINAL_REFRESH_INTERVAL {
		own.termWidth = own.getTerminalSize(terminalFileOf(own.writer))
		own.termChecked = now
	}

	// 进度/速度区宽度只与 total、barType 有关，二者不变时直接复用缓存，
	// 省去每帧的 Itoa / HumanizeBytes 格式化开销
	if own.layoutDirty || own.totalWidth == 0 ||
		own.layoutType != own.barType || own.layoutPrefix != own.barPrefix {
		// 宽度估算必须与 calcProcess / calcSpeed 的实际展示格式保持一致
		switch own.barType {
		case TYPE_BYTES:
			total := len(stringx.HumanizeBytes(own.total))
			own.processMaxWidth = total*2 + 1 // 1.00GB/1023.22TB
			own.speedMaxWidth = total + 3     // 1.00GB/s
		default:
			total := len(strconv.FormatInt(own.total, 10))
			own.processMaxWidth = total*2 + 1 // 123456789/123456789
			own.speedMaxWidth = total + 5     // 123456789.00/s
		}

		own.totalWidth = own.processMaxWidth + own.speedMaxWidth
		own.prefixWidth = 0

		// 前缀按显示宽度计算：中文等宽字符占 2 列，按字节算会导致进度条过长
		for range own.barPrefix {
			own.prefixWidth += 2
		}
	}
	own.barMaxWidth = own.termWidth - (own.prefixWidth + own.totalWidth + _BAR_RIGHT_MARGIN)

	// 设置进度条宽度
	if own.barMaxWidth < _BAR_MIN_SIZE {
		own.barMaxWidth = _BAR_MIN_SIZE
	} else if own.barMaxWidth > _BAR_MAX_SIZE {
		own.barMaxWidth = _BAR_MAX_SIZE
	}

	// 记录本次布局所依据的输入，并清除脏标记
	own.layoutType = own.barType
	own.layoutPrefix = own.barPrefix
	own.layoutDirty = false
}

// calcProcess 计算进度（当前量/总量）
func (own *Progressbar) calcProcess(builder *strings.Builder) {
	if own.barType == TYPE_NUMBER {
		builder.WriteString(strconv.FormatInt(own.current, 10))
		builder.WriteByte('/')
		builder.WriteString(strconv.FormatInt(own.total, 10))
		return
	}

	// TYPE_BYTES：按人类可读的字节单位展示
	builder.WriteString(stringx.HumanizeBytes(own.current))
	builder.WriteByte('/')
	builder.WriteString(stringx.HumanizeBytes(own.total))
}

// calcSpeed 计算速度
func (own *Progressbar) calcSpeed(builder *strings.Builder) {
	if own.barType == TYPE_NUMBER {
		builder.WriteString(strconv.FormatFloat(own.speedometer.GetSpeed(), 'f', 2, 64))
		builder.WriteString("/s")
		return
	}

	// TYPE_BYTES：按人类可读的字节单位展示
	builder.WriteString(stringx.HumanizeBytes(int64(own.speedometer.GetSpeed())))
	builder.WriteString("/s")
}

// finish 进度条完成（调用方必须持有写锁）
func (own *Progressbar) finish() {
	own.isFinished = true
	if own.hideCursor {
		fmt.Fprint(own.writer, "\033[?25h") // 若光标被隐藏，则完成时恢复显示
	}

	fmt.Fprintln(own.writer, own.callAfterFinish(own.barPrefix, time.Since(own.startTime).Round(time.Second)))
}

// Reset 重置进度条状态，但不重置总量和类型
func (own *Progressbar) Reset() error {
	own.lock.Lock()
	own.isFinished = false
	own.current = 0
	own.startTime = time.Now()
	own.speedometer = speedometer.New(3 * time.Second)
	own.lock.Unlock()
	return nil
}
