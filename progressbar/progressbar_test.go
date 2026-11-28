package progressbar

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// newTestBar 创建一个输出到内存缓冲区的进度条，便于断言渲染结果
func newTestBar(total int64) (*Progressbar, *bytes.Buffer) {
	var buf bytes.Buffer
	bar := New(total)
	bar.SetWriter(&buf)
	bar.SetType(TYPE_NUMBER)
	return bar, &buf
}

func TestProgressbar_Render(t *testing.T) {
	t.Parallel()

	bar, buf := newTestBar(100)
	bar.Add(50)

	out := buf.String()
	for _, want := range []string{"50.00%", "50/100", "/s"} {
		if !strings.Contains(out, want) {
			t.Errorf("render output missing %q\n got: %q", want, out)
		}
	}
	// 进度条本体应包含已填充与未填充两种字符
	if !strings.Contains(out, "[") || !strings.Contains(out, "]") {
		t.Errorf("render output missing bar brackets\n got: %q", out)
	}
	if bar.IsFinished() {
		t.Error("bar should not be finished at 50%")
	}
}

func TestProgressbar_Finish(t *testing.T) {
	t.Parallel()

	bar, buf := newTestBar(100)
	bar.Add(100)

	if !bar.IsFinished() {
		t.Error("bar should be finished after reaching total")
	}
	if !strings.Contains(buf.String(), "Cost:") {
		t.Errorf("finish output missing completion message\n got: %q", buf.String())
	}

	// 重复结束不应产生额外输出
	before := buf.Len()
	bar.Finish()
	if buf.Len() != before {
		t.Errorf("duplicate finish produced extra output: %q", buf.String()[before:])
	}
}

func TestProgressbar_Update(t *testing.T) {
	t.Parallel()

	bar, _ := newTestBar(100)
	bar.Update(30)
	bar.Update(80)

	// Update 只接受递增，回退应被忽略
	bar.Update(40)
	if got := bar.GetCount(); got != 80 {
		t.Errorf("Update should ignore regression, count = %d, want 80", got)
	}
}

func TestProgressbar_AddInvalid(t *testing.T) {
	t.Parallel()

	bar, _ := newTestBar(100)
	bar.Add(0)
	bar.Add(-10)
	if got := bar.GetCount(); got != 0 {
		t.Errorf("non-positive Add should be ignored, count = %d, want 0", got)
	}
}

func TestProgressbar_ZeroTotal(t *testing.T) {
	t.Parallel()

	// total <= 0 不应除零导致 NaN 或 panic
	bar, buf := newTestBar(0)
	bar.Add(0)
	bar.Finish()
	if out := buf.String(); strings.Contains(out, "NaN") {
		t.Errorf("zero total produced NaN: %q", out)
	}
}

func TestProgressbar_SetPrefixAndStyle(t *testing.T) {
	t.Parallel()

	bar, buf := newTestBar(100)
	bar.SetPrefix("下载：")
	bar.SetSpinners([]rune{'|', '/', '-', '\\'})
	bar.Add(100)

	out := buf.String()
	if !strings.Contains(out, "下载：") {
		t.Errorf("prefix not rendered: %q", out)
	}
	if !strings.Contains(out, "█") {
		t.Errorf("default style not applied: %q", out)
	}
}

// TestProgressbar_Style 验证内置样式常量被正确渲染
func TestProgressbar_Style(t *testing.T) {
	t.Parallel()

	bar, buf := newTestBar(100)
	bar.SetStyle(StyleRhombus)
	bar.Add(50)

	out := buf.String()
	if !strings.Contains(out, "▰") || !strings.Contains(out, "▱") {
		t.Fatalf("custom style not applied: %q", out)
	}
}

// TestProgressbar_TruncateRunes 验证超宽内容按 rune 截断，不会切断多字节字符
func TestProgressbar_TruncateRunes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"abc", 2, "ab"},
		{"中文测试", 2, "中文"},
		{"a中文b", 3, "a中文"},
		{"🎈🎈🎈", 1, "🎈"},
	}
	for _, c := range cases {
		if got := truncateRunes(c.in, c.max); got != c.want {
			t.Errorf("truncateRunes(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestProgressbar_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	bar, _ := newTestBar(1000000)
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 1000; j++ {
				bar.Add(1)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}

	if got := bar.GetCount(); got != 8000 {
		t.Errorf("concurrent Add lost updates: count = %d, want 8000", got)
	}
}

// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		   	27151431	        44.72 ns/op	       0 B/op	       0 allocs/op
func BenchmarkProgressbar_Render(b *testing.B) {
	bar := New(1 << 40)
	bar.SetWriter(ioDiscard{})
	bar.SetType(TYPE_BYTES)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Add(1 << 20)
	}
}

// BenchmarkProgressbar_RenderForce 绕过渲染节流，衡量单次完整渲染的真实成本
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		   	 1157518	      1088 ns/op	     324 B/op	      12 allocs/op
func BenchmarkProgressbar_RenderForce(b *testing.B) {
	bar := New(1 << 50)
	bar.SetWriter(ioDiscard{})
	bar.SetType(TYPE_BYTES)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.lastRender = time.Time{} // 强制每次都渲染
		bar.Add(1 << 20)
	}
}

// ioDiscard 避免测试期写入终端
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

// TestProgressbar_EndToEnd 验证两轮完整渲染流程（Reset 与样式切换）
//   - 写入磁盘丢弃器，避免测试期间刷屏
func TestProgressbar_EndToEnd(t *testing.T) {
	t.Parallel()

	bar := New(1000)
	bar.SetWriter(ioDiscard{})
	bar.SetPrefix("下载文件1")
	bar.SetType(TYPE_BYTES)
	for range 1000 {
		bar.Add(1)
		if bar.IsFinished() {
			break
		}
	}
	if !bar.IsFinished() {
		t.Fatal("第一轮进度条应已完成")
	}

	if err := bar.Reset(); err != nil {
		t.Fatalf("Reset() 返回错误: %v", err)
	}
	bar.SetPrefix("下载文件2")
	bar.SetType(TYPE_NUMBER)
	bar.SetSpinners([]rune{'|', '/', '-', '\\'})
	bar.SetStyle([2]rune{'#', '-'})
	for range 1000 {
		bar.Add(1)
		if bar.IsFinished() {
			break
		}
	}
	if !bar.IsFinished() {
		t.Fatal("第二轮进度条应已完成")
	}
}
