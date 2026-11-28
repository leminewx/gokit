package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPrettyHandler(t *testing.T) {
	handler := NewPrettyHandler(os.Stdout, &Options{
		Level:     slog.LevelDebug,
		WithColor: false,
	})

	logger := slog.New(handler)
	logger.Debug("调试信息")
	logger.Info("服务启动成功", "host", "127.0.0.1", "port", 8080)
	logger.Warn("磁盘空间不足", "used", 0.85)
	logger.Error("账号或密码错误")

	handler = NewPrettyHandler(os.Stdout, &Options{
		Level:       slog.LevelDebug,
		WithColor:   true,
		AttrConnect: " => ",
	})

	logger = slog.New(handler)
	logger.Debug("调试信息")
	logger.Info("服务启动成功", "host", "127.0.0.1", "port", 8080)
	logger.Warn("磁盘空间不足", "used", 0.85)
	logger.Error("账号或密码错误")

	handler = NewPrettyHandler(os.Stdout, &Options{
		Level:       slog.LevelDebug,
		WithColor:   false,
		WithSource:  true,
		AttrConnect: " && ",
	})

	// WithSource 必须配合 logger.NewLogger 才能正确解析源码位置：
	// go test 下 testing.tRunner 会吞掉 TestXxx 帧，故在独立 goroutine 内调用
	// （栈：goexit → 本闭包 → Info → log，无 tRunner），源码即可定位到本测试文件。
	srcLogger := NewLogger(handler)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		srcLogger.Debug("调试信息")
		srcLogger.Info("服务启动成功", "host", "127.0.0.1", "port", 8080)
		srcLogger.Warn("磁盘空间不足", "used", 0.85)
		srcLogger.Error("账号或密码错误")
	}()
	wg.Wait()

	handler = NewPrettyHandler(os.Stdout, &Options{
		Level:       slog.LevelDebug,
		WithColor:   true,
		WithSource:  true,
		AttrConnect: " >> ",
	})

	srcLogger = NewLogger(handler)
	wg.Add(1)
	go func() {
		defer wg.Done()
		srcLogger.Debug("调试信息")
		srcLogger.Info("服务启动成功", "host", "127.0.0.1", "port", 8080)
		srcLogger.Warn("磁盘空间不足", "used", 0.85)
		srcLogger.Error("账号或密码错误")
	}()
	wg.Wait()
}

func TestPrettyHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	base := NewPrettyHandler(&buf, &Options{WithColor: false})
	// 模拟 logger.With("service", "api", "version", "v1")
	h := base.WithAttrs([]slog.Attr{
		slog.String("service", "api"),
		slog.String("version", "v1"),
	})
	logger := slog.New(h)
	logger.Info("started", "port", 8080)

	out := buf.String()
	for _, want := range []string{"service=api", "version=v1", "port=8080"} {
		if !strings.Contains(out, want) {
			t.Errorf("WithAttrs lost field %q\n got: %s", want, out)
		}
	}
}

func TestPrettyHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	base := NewPrettyHandler(&buf, &Options{WithColor: false})
	h := base.WithGroup("req").WithAttrs([]slog.Attr{slog.String("id", "abc")})
	logger := slog.New(h)
	logger.Info("handled", "path", "/health")

	out := buf.String()
	if !strings.Contains(out, "req.id=abc") {
		t.Errorf("WithGroup prefix missing: got %s", out)
	}
	if !strings.Contains(out, "req.path=/health") {
		t.Errorf("WithGroup nested attr prefix missing: got %s", out)
	}
}

// TestPrettyHandler_ReplaceAttrNoDeadlock 验证在 ReplaceAttr 内用同一 logger 打日志不会死锁
//   - 回归点：Handle 曾在持锁期间调用 ReplaceAttr，sync.Mutex 不可重入
func TestPrettyHandler_ReplaceAttrNoDeadlock(t *testing.T) {
	var buf bytes.Buffer

	var logger *slog.Logger
	var nested bool
	handler := NewPrettyHandler(&buf, &Options{
		WithColor: false,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if !nested && attr.Key == "secret" {
				nested = true
				logger.Info("nested") // 回调内再次打日志
				return slog.String(attr.Key, "***")
			}

			return attr
		},
	})
	logger = slog.New(handler)

	done := make(chan struct{})
	go func() {
		defer close(done)
		logger.Info("outer", "secret", "p@ss")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReplaceAttr 内再次打日志导致死锁")
	}

	out := buf.String()
	if !strings.Contains(out, "nested") || !strings.Contains(out, "outer") {
		t.Fatalf("两条日志都应输出, got %q", out)
	}
	if !strings.Contains(out, "secret=***") || strings.Contains(out, "p@ss") {
		t.Fatalf("ReplaceAttr 未生效, got %q", out)
	}
}

// TestPrettyHandler_ReplaceAttrReceivesGroups 验证 ReplaceAttr 能收到当前分组路径
//   - 回归点：applyReplace 曾固定传 nil，自定义替换器无法按分组过滤
func TestPrettyHandler_ReplaceAttrReceivesGroups(t *testing.T) {
	var buf bytes.Buffer

	var seen [][]string
	handler := NewPrettyHandler(&buf, &Options{
		WithColor: false,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			seen = append(seen, append([]string(nil), groups...))
			return attr
		},
	})

	logger := slog.New(handler.WithGroup("req"))
	logger.Info("handled", "id", 1)

	if len(seen) != 1 || len(seen[0]) != 1 || seen[0][0] != "req" {
		t.Fatalf("ReplaceAttr groups = %v, want [[req]]", seen)
	}
	if out := buf.String(); !strings.Contains(out, "req.id=1") {
		t.Fatalf("分组前缀缺失: %q", out)
	}
}

// TestPrettyHandler_PresetAttrKeepsOwnGroup 验证 WithGroup 不会给之前添加的预置属性加前缀
func TestPrettyHandler_PresetAttrKeepsOwnGroup(t *testing.T) {
	var buf bytes.Buffer
	base := NewPrettyHandler(&buf, &Options{WithColor: false})

	// slog.New().With() → WithGroup() 后，先前的属性应保持原样
	logger := slog.New(base.WithAttrs([]slog.Attr{slog.String("a", "1")}).WithGroup("g"))
	logger.Info("m", "b", 2)

	out := buf.String()
	if strings.Contains(out, "g.a=1") {
		t.Fatalf("预置属性不应被后续 WithGroup 加工: %q", out)
	}
	if !strings.Contains(out, "a=1") || !strings.Contains(out, "g.b=2") {
		t.Fatalf("属性输出不正确: %q", out)
	}
}

// TestNewPrettyHandlerDoesNotMutateOptions 验证构造时不会修改调用方的 Options
func TestNewPrettyHandlerDoesNotMutateOptions(t *testing.T) {
	opts := &Options{WithColor: false}
	_ = NewPrettyHandler(io.Discard, opts)

	if opts.Level != nil || opts.TimeFormat != "" || opts.AttrSplit != "" || opts.AttrConnect != "" {
		t.Fatalf("NewPrettyHandler 修改了调用方 Options: %+v", opts)
	}
}

func TestPrettyHandler_LevelFilter(t *testing.T) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{Level: slog.LevelWarn, WithColor: false})
	logger := slog.New(h)
	logger.Debug("debug skipped")
	logger.Info("info skipped")
	logger.Warn("warn shown")
	logger.Error("error shown")

	out := buf.String()
	if strings.Contains(out, "debug skipped") || strings.Contains(out, "info skipped") {
		t.Errorf("level filter not applied: %s", out)
	}
	if !strings.Contains(out, "warn shown") || !strings.Contains(out, "error shown") {
		t.Errorf("warn/error should appear: %s", out)
	}
}

func TestPrettyHandler_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false})
	logger := slog.New(h)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Info("msg", "i", n)
		}(i)
	}
	wg.Wait()
	// 无 panic / 无数据竞争（race 检测可进一步验证）
}

func TestPrettyHandler_WithSource(t *testing.T) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false, WithSource: true})
	// 使用 logger.NewLogger：PC 在用户调用边界抓取并填入 r.PC，Handle 直接复用。
	// 注意：go test 下 testing.tRunner 会吞掉 TestXxx 帧，故放在独立 goroutine 内
	// 调用（goexit→本闭包→Info→log，无 tRunner），源码位置即可解析到本测试文件。
	logger := NewLogger(h)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("hello from test")
	}()
	wg.Wait()

	out := buf.String()
	// 应包含 "pretty_handler_test.go:" 且行号为正整数
	if !strings.Contains(out, "pretty_handler_test.go:") {
		t.Fatalf("WithSource did not print source file, got: %s", out)
	}
	// 提取冒号后的行号，确认非零
	idx := strings.Index(out, "pretty_handler_test.go:")
	rest := out[idx+len("pretty_handler_test.go:"):]
	numPart := rest[:strings.IndexByte(rest, ' ')]
	if numPart == "0" || numPart == "" {
		t.Fatalf("WithSource line number invalid: %q (out=%s)", numPart, out)
	}
}

func TestPrettyHandler_NoSource(t *testing.T) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false, WithSource: false})
	logger := slog.New(h)
	logger.Info("no source here")
	out := buf.String()
	if strings.Contains(out, ".go:") {
		t.Fatalf("WithSource=false should not print file:line, got: %s", out)
	}
}

// secret 测试用 LogValuer：输出时必须被解析，而不是输出空串
type secret string

func (s secret) LogValue() slog.Value { return slog.StringValue("<redacted>") }

// TestPrettyHandler_GroupAndLogValuerAttrs 验证 group 属性会展开、LogValuer 会被解析
//   - 回归点：两者在 slog.Value.String() 中均返回空串，曾被输出为 key=
func TestPrettyHandler_GroupAndLogValuerAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false})
	logger := slog.New(h)

	logger.Info("msg",
		slog.Group("db",
			slog.String("host", "127.0.0.1"),
			slog.Group("auth", slog.String("user", "root")),
		),
		"token", secret("p@ss"),
	)

	out := buf.String()
	for _, want := range []string{
		"db.host=127.0.0.1",
		"db.auth.user=root",
		"token=<redacted>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "p@ss") {
		t.Errorf("LogValuer 未被解析: %q", out)
	}
}

var (
	contextBG = context.Background()
	dummyTime = time.Date(2026, 8, 28, 22, 31, 44, 416000000, time.UTC)
)

// 无缓冲、无颜色的典型场景
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		 2046154	       568.1 ns/op	     384 B/op	      10 allocs/op
func BenchmarkHandle_Plain(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &Options{WithColor: false})
	r := slog.Record{}
	r.Message = "request handled"
	r.Level = slog.LevelInfo
	r.Time = dummyTime
	r.Add("path", "/api/users")
	r.Add("status", 200)
	r.Add("latency_ms", 12)

	b.ReportAllocs()
	for b.Loop() {
		_ = h.Handle(contextBG, r)
	}
}

// 带颜色的典型场景
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		 1783046	       726.7 ns/op	     320 B/op	      10 allocs/op
func BenchmarkHandle_Color(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &Options{WithColor: true})
	r := slog.Record{}
	r.Message = "request handled"
	r.Level = slog.LevelInfo
	r.Time = dummyTime
	r.Add("path", "/api/users")
	r.Add("status", 200)

	b.ReportAllocs()
	for b.Loop() {
		_ = h.Handle(contextBG, r)
	}
}

// 无 WithSource（最常见），确认 CallersFrames 不被触发
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		 1802554	       652.7 ns/op	     300 B/op	       6 allocs/op
func BenchmarkHandle_NoSource(b *testing.B) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false, WithSource: false})
	logger := slog.New(h)

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		logger.Info("msg", "k", i)
	}
}

// 带 WithSource，验证源码定位的开销（opt-in，仅此场景付出代价）。
// 使用 logger.NewLogger：PC 在用户调用边界抓取并填入 r.PC，Handle 直接复用，无需在 Handle 内再做栈遍历。
// cpu: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz		 1043757	      1176 ns/op	     472 B/op	       7 allocs/op
func BenchmarkHandle_WithSource(b *testing.B) {
	var buf bytes.Buffer
	h := NewPrettyHandler(&buf, &Options{WithColor: false, WithSource: true})
	logger := NewLogger(h)

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		logger.Info("msg", "k", i)
	}
}
