package color

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestWrapStringWithANSI(t *testing.T) {
	forceColorOutput(t)

	tests := []struct {
		name  string
		msg   string
		codes []Code
		want  string // 精确匹配，但注意颜色码可能顺序变化，我们检查关键部分
	}{
		{
			name:  "no codes",
			msg:   "plain",
			codes: nil,
			want:  "plain",
		},
		{
			name:  "single style",
			msg:   "bold",
			codes: []Code{STYLE_BOLD},
			want:  "\x1b[1mbold\x1b[0m",
		},
		{
			name:  "fg only",
			msg:   "red",
			codes: []Code{FG_RED},
			want:  "\x1b[31mred\x1b[0m",
		},
		{
			name:  "fg + bg + style",
			msg:   "combo",
			codes: []Code{FG_GREEN, BG_BLACK, STYLE_UNDERLINE},
			want:  "\x1b[32;40;4mcombo\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapStringWithANSI(tt.msg, tt.codes...)
			if got != tt.want {
				t.Errorf("wrapStringWithANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapFormatStringWithANSI(t *testing.T) {
	forceColorOutput(t)

	got := wrapFormatStringWithANSI("Hello %s", []any{"World"}, FG_CYAN, STYLE_BOLD)
	expectedPrefix := "\x1b[36;1mHello World\x1b[0m"
	if got != expectedPrefix {
		t.Errorf("wrapFormatStringWithANSI() = %q, want %q", got, expectedPrefix)
	}

	// 测试无 codes
	gotPlain := wrapFormatStringWithANSI("Hi %d", []any{42})
	if gotPlain != "Hi 42" {
		t.Errorf("wrapFormatStringWithANSI (no codes) = %q, want %q", gotPlain, "Hi 42")
	}
}

func TestBuildANSI(t *testing.T) {
	var b strings.Builder
	codes := []Code{STYLE_BOLD, FG_RED, BG_WHITE}
	appendANSIBuilder(&b, codes...)
	expected := "\x1b[1;31;47m"
	if b.String() != expected {
		t.Errorf("appendANSIBuilder() = %q, want %q", b.String(), expected)
	}

	// 真彩色
	var br strings.Builder
	rgb := []Code{NewRGB(255, 0, 0), NewBgRGB(0, 128, 255)}
	appendANSIBuilder(&br, rgb...)
	wantRGB := "\x1b[38;2;255;0;0;48;2;0;128;255m"
	if br.String() != wantRGB {
		t.Errorf("appendANSIBuilder(rgb) = %q, want %q", br.String(), wantRGB)
	}
}

// 测试 style 类型的方法
func TestStyleMethods(t *testing.T) {
	forceColorOutput(t)

	s := New(STYLE_BOLD, FG_GREEN)

	// Sprint
	result := s.Sprint("test")
	if !isANSIWrapped(result) {
		t.Errorf("Sprint did not wrap with ANSI: %q", result)
	}
	if !strings.Contains(result, "test") {
		t.Errorf("Sprint missing message")
	}

	// Sprintf
	resultf := s.Sprintf("Number %d", 42)
	if !isANSIWrapped(resultf) || !strings.Contains(resultf, "Number 42") {
		t.Errorf("Sprintf failed: %q", resultf)
	}

	// Sprintln
	resultln := s.Sprintln("line")
	if !strings.HasSuffix(resultln, "\n") {
		t.Errorf("Sprintln missing newline")
	}
	trimmed := strings.TrimSuffix(resultln, "\n")
	if !isANSIWrapped(trimmed) {
		t.Errorf("Sprintln content not wrapped")
	}

	// Print/Printf/Println 测试输出到 stdout 不易捕获，可重定向 defaultWriter 测试
	// 这里简化为检查不 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Print/Printf/Println panicked: %v", r)
		}
	}()
	s.Print("print")
	s.Printf("print %s", "format")
	s.Println("println")
}

func TestNoColor(t *testing.T) {
	// 保存并恢复全局开关
	old := IsNoColor()
	defer SetNoColor(old)

	SetNoColor(true)
	if !IsNoColor() {
		t.Fatal("SetNoColor(true) 未生效")
	}

	s := New(STYLE_BOLD, FG_RED)
	if got := s.Sprint("plain"); got != "plain" {
		t.Errorf("NoColor Sprint = %q, want %q", got, "plain")
	}
	if got := s.Sprintf("n=%d", 1); got != "n=1" {
		t.Errorf("NoColor Sprintf = %q, want %q", got, "n=1")
	}
	if got := s.Sprintln("x"); got != "x\n" {
		t.Errorf("NoColor Sprintln = %q, want %q", got, "x\n")
	}

	// 关闭纯文本模式后恢复带色输出
	SetNoColor(false)
	if got := s.Sprint("c"); !isANSIWrapped(got) {
		t.Errorf("after disabling NoColor Sprint not wrapped: %q", got)
	}
}

func TestSetNoColorConcurrent(t *testing.T) {
	forceColorOutput(t)

	const goroutines = 32
	done := make(chan struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 200; j++ {
				SetNoColor(j%2 == 0)
				_ = IsNoColor()
				_ = wrapStringWithANSI("concurrent", STYLE_BOLD)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

func TestSetOutput(t *testing.T) {
	forceColorOutput(t)

	var buf strings.Builder
	old := outputWriter()
	defer SetOutput(old)

	SetOutput(&buf)
	New(FG_RED).Print("hello")
	if got := buf.String(); got != "\x1b[31mhello\x1b[0m" {
		t.Errorf("Print to redirected writer = %q", got)
	}

	// 传入 nil 不应覆盖现有输出流
	SetOutput(nil)
	if got := outputWriter(); got != io.Writer(&buf) {
		t.Errorf("SetOutput(nil) 覆盖了现有输出流: %T", got)
	}
}

func TestRGB(t *testing.T) {
	forceColorOutput(t)

	// 真彩色前景
	s := New(NewRGB(255, 0, 0))
	got := s.Sprint("red")
	want := "\x1b[38;2;255;0;0mred\x1b[0m"
	if got != want {
		t.Errorf("RGB fg Sprint = %q, want %q", got, want)
	}

	// 真彩色背景
	bg := New(NewBgRGB(0, 128, 255))
	gotBg := bg.Sprint("blue")
	wantBg := "\x1b[48;2;0;128;255mblue\x1b[0m"
	if gotBg != wantBg {
		t.Errorf("BgRGB Sprint = %q, want %q", gotBg, wantBg)
	}
}

func TestBrightColors(t *testing.T) {
	forceColorOutput(t)

	s := New(FG_BRIGHT_GREEN, BG_BRIGHT_BLACK)
	got := s.Sprint("hi")
	want := "\x1b[92;100mhi\x1b[0m"
	if got != want {
		t.Errorf("bright Sprint = %q, want %q", got, want)
	}
}

func TestFprintln(t *testing.T) {
	forceColorOutput(t)

	var buf strings.Builder
	s := New(STYLE_BOLD, FG_RED)
	s.Fprintln(&buf, "hello")
	want := "\x1b[1;31mhello\x1b[0m\n"
	if buf.String() != want {
		t.Errorf("Fprintln = %q, want %q", buf.String(), want)
	}
}

// 测试并发安全（sync.Pool 使用）
func TestConcurrentUse(t *testing.T) {
	const goroutines = 100
	const iterations = 100
	done := make(chan bool)
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_ = wrapStringWithANSI("concurrent", STYLE_BOLD, FG_RED)
				_ = wrapFormatStringWithANSI("num %d", []any{j}, STYLE_UNDERLINE)
			}
			done <- true
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	// 若未 panic 则通过
}

func BenchmarkWrapStringWithANSI(b *testing.B) {
	msg := "benchmark message"
	codes := []Code{STYLE_BOLD, FG_RED, BG_WHITE, STYLE_UNDERLINE}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrapStringWithANSI(msg, codes...)
	}
}

func BenchmarkWrapFormatStringWithANSI(b *testing.B) {
	format := "Value: %d, Str: %s"
	args := []any{42, "test"}
	codes := []Code{STYLE_ITALIC, FG_CYAN}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrapFormatStringWithANSI(format, args, codes...)
	}
}

// 真彩色路径（RGB 展开更长，验证性能）
func BenchmarkWrapStringWithANSI_RGB(b *testing.B) {
	msg := "benchmark message"
	codes := []Code{NewRGB(255, 100, 50), NewBgRGB(10, 20, 30), STYLE_BOLD}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrapStringWithANSI(msg, codes...)
	}
}

// 真实调用路径（经 style 方法）
func BenchmarkStyle_Sprint(b *testing.B) {
	s := New(STYLE_BOLD, FG_RED, BG_WHITE, STYLE_UNDERLINE)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Sprint("benchmark message")
	}
}

func TestColor(t *testing.T) {
	// 创建一个粗体样式（用于标题）
	titleStyle := New(STYLE_BOLD)
	titleStyle.Println("=== 样式演示 ===")

	// 每个样式单独展示
	New(STYLE_BOLD).Println("粗体 (STYLE_BOLD)")
	New(STYLE_DIM).Println("淡色 (STYLE_DIM)")
	New(STYLE_ITALIC).Println("斜体 (STYLE_ITALIC)")
	New(STYLE_UNDERLINE).Println("下划线 (STYLE_UNDERLINE)")
	New(STYLE_BLINK).Println("闪烁 (STYLE_BLINK)")
	New(STYLE_FAST_BLINK).Println("快速闪烁 (STYLE_FAST_BLINK)")
	New(STYLE_REVERSE).Println("反色 (STYLE_REVERSE)")
	New(STYLE_CONCEAL).Println("隐藏 (STYLE_CONCEAL)") // 可能看不见文字
	New(STYLE_STRIKETHROUGH).Println("删除线 (STYLE_STRIKETHROUGH)")

	// 前景色演示
	titleStyle.Println("\n=== 前景色 ===")
	for _, c := range []Code{FG_BLACK, FG_RED, FG_GREEN, FG_YELLOW,
		FG_BLUE, FG_MAGENTA, FG_CYAN, FG_WHITE} {
		// 使用该前景色打印其代码
		New(c).Printf(" %d ", c)
	}

	fmt.Println()

	// 背景色演示（用白色前景显示）
	titleStyle.Println("\n=== 背景色 ===")
	for _, c := range []Code{BG_BLACK, BG_RED, BG_GREEN, BG_YELLOW,
		BG_BLUE, BG_MAGENTA, BG_CYAN, BG_WHITE} {
		New(c, FG_WHITE).Printf(" %d ", c)
	}

	fmt.Println()

	// 组合示例
	titleStyle.Println("\n=== 组合示例 ===")
	New(FG_RED, BG_WHITE, STYLE_BOLD).Println("红字白底粗体")
	New(FG_GREEN, BG_BLACK, STYLE_ITALIC).Println("绿字黑底斜体")
	New(FG_YELLOW, BG_BLUE, STYLE_UNDERLINE).Println("黄字蓝底下划线")
	New(FG_CYAN, BG_MAGENTA, STYLE_STRIKETHROUGH).Println("青字洋红底删除线")
	New(FG_WHITE, BG_RED, STYLE_REVERSE).Println("白字红底反色")
}

// forceColorOutput 关闭纯文本模式并在用例结束后恢复原状态
func forceColorOutput(t *testing.T) {
	t.Helper()

	old := IsNoColor()
	SetNoColor(false)
	t.Cleanup(func() { SetNoColor(old) })
}

// 辅助函数：检查字符串是否以 ANSI 开头并以重置结尾
func isANSIWrapped(s string) bool {
	return strings.HasPrefix(s, "\x1b[") && strings.HasSuffix(s, "\x1b[0m")
}
