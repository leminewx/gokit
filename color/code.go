// Package color 提供轻量、零依赖的 ANSI 终端着色与文本样式能力。
//
// 设计目标：
//   - 不依赖任何第三方库，仅使用 Go 标准库；
//   - API 风格与标准库 fmt 保持一致（Print / Printf / Println / Sprint / Sprintf / Sprintln
//     以及对应的 Fprint / Fprintf / Fprintln）；
//   - 调用时按需组合"样式 + 前景色 + 背景色"代码，自动包裹 ANSI 转义序列并在末尾重置，
//     不会污染后续输出。
//
// 快速上手：
//
//	color.New(color.FG_GREEN, color.STYLE_BOLD).Println("成功")
//	color.New(color.FG_RED).Printf("错误码: %d\n", 500)
//
// 颜色与样式：
//   - 基础 8 色前景（FG_BLACK ... FG_WHITE）与背景（BG_BLACK ... BG_WHITE）；
//   - 亮色前景（FG_BRIGHT_BLACK ... FG_BRIGHT_WHITE）与亮色背景
//     （BG_BRIGHT_BLACK ... BG_BRIGHT_WHITE），覆盖现代终端的 16 色；
//   - 24 位真彩色：使用 color.NewRGB(r, g, b) 与 color.NewBgRGB(r, g, b) 生成任意
//     前景/背景色，自动展开为 "\x1b[38;2;r;g;bm" / "\x1b[48;2;r;g;bm"；
//   - 文本样式：加粗、淡色、斜体、下划线、闪烁、反显、隐藏、删除线
//     （STYLE_BOLD / STYLE_DIM / STYLE_ITALIC / STYLE_UNDERLINE / STYLE_BLINK /
//     STYLE_REVERSE / STYLE_CONCEAL / STYLE_STRIKETHROUGH）。
//
// 全局开关与输出控制：
//   - 全局变量 NoColor：设为 true 时所有带色输出降级为纯文本，
//     适用于日志文件、非 TTY 环境、CI 等场景，避免写入不可读的转义序列；
//   - 函数 SetOutput(w)：修改默认输出流（默认 os.Stdout），便于重定向或测试。
//
// 性能说明：
//   - 内部复用 sync.Pool 中的 strings.Builder，减少短生命周期分配；
//   - 在 NoColor 模式或空代码情况下直接返回原字符串，零额外开销。
//
// 本包与 gokit 其它模块（如 logger）协同工作：logger 会读取 color.NoColor
// 来决定是否输出彩色日志级别。
package color

import "strconv"

// Code 颜色和样式的枚举类型
type Code int

// 文本样式代码（加粗、淡色、斜体等）。
const (
	STYLE_BOLD          Code = iota + 1 // 加粗
	STYLE_DIM                           // 淡色
	STYLE_ITALIC                        // 斜体
	STYLE_UNDERLINE                     // 下划线
	STYLE_BLINK                         // 闪烁
	STYLE_FAST_BLINK                    // 快速闪烁
	STYLE_REVERSE                       // 反转
	STYLE_CONCEAL                       // 隐藏文本
	STYLE_STRIKETHROUGH                 // 删除线
)

// 前景色（基础 8 色）。
const (
	FG_BLACK   Code = iota + 30 // 黑
	FG_RED                      // 红
	FG_GREEN                    // 绿
	FG_YELLOW                   // 黄
	FG_BLUE                     // 蓝
	FG_MAGENTA                  // 紫
	FG_CYAN                     // 青
	FG_WHITE                    // 白
)

// 亮前景色（90-97）。
const (
	FG_BRIGHT_BLACK   Code = iota + 90 // 亮黑（灰）
	FG_BRIGHT_RED                      // 亮红
	FG_BRIGHT_GREEN                    // 亮绿
	FG_BRIGHT_YELLOW                   // 亮黄
	FG_BRIGHT_BLUE                     // 亮蓝
	FG_BRIGHT_MAGENTA                  // 亮紫
	FG_BRIGHT_CYAN                     // 亮青
	FG_BRIGHT_WHITE                    // 亮白
)

// 背景色（基础 8 色）。
const (
	BG_BLACK   Code = iota + 40 // 黑
	BG_RED                      // 红
	BG_GREEN                    // 绿
	BG_YELLOW                   // 黄
	BG_BLUE                     // 蓝
	BG_MAGENTA                  // 紫
	BG_CYAN                     // 青
	BG_WHITE                    // 白
)

// 亮背景色（100-107）。
const (
	BG_BRIGHT_BLACK   Code = iota + 100 // 亮黑（灰）
	BG_BRIGHT_RED                       // 亮红
	BG_BRIGHT_GREEN                     // 亮绿
	BG_BRIGHT_YELLOW                    // 亮黄
	BG_BRIGHT_BLUE                      // 亮蓝
	BG_BRIGHT_MAGENTA                   // 亮紫
	BG_BRIGHT_CYAN                      // 亮青
	BG_BRIGHT_WHITE                     // 亮白
)

const (
	_FLAG_RGB_FOREGROUND Code = 0x1000000 // 真彩色标志位（前景）
	_FLAG_RGB_BACKGROUND Code = 0x2000000 // 真彩色标志位（背景）
)

// NewRGB 生成 24 位真彩色前景色代码（38;2;r;g;b）
func NewRGB(r, g, b uint8) Code {
	return _FLAG_RGB_FOREGROUND | Code(r)<<16 | Code(g)<<8 | Code(b)
}

// NewBgRGB 生成 24 位真彩色背景色代码（48;2;r;g;b）
func NewBgRGB(r, g, b uint8) Code {
	return _FLAG_RGB_BACKGROUND | Code(r)<<16 | Code(g)<<8 | Code(b)
}

// isRGB 判断是否为真彩色代码
func (own Code) isRGB() (r, g, b uint8, fg bool, ok bool) {
	switch {
	case own&_FLAG_RGB_FOREGROUND != 0 && own&_FLAG_RGB_BACKGROUND == 0:
		v := own &^ _FLAG_RGB_FOREGROUND
		return uint8(v >> 16), uint8(v >> 8), uint8(v), true, true
	case own&_FLAG_RGB_BACKGROUND != 0:
		v := own &^ _FLAG_RGB_BACKGROUND
		return uint8(v >> 16), uint8(v >> 8), uint8(v), false, true
	}

	return 0, 0, 0, false, false
}

// String 返回代码对应的 ANSI 数字字符串（供调试）
func (own Code) String() string {
	return strconv.Itoa(int(own))
}
