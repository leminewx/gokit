package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// newTestLogger 构造将 JSON 记录写入 buf 的 Logger
func newTestLogger(buf *bytes.Buffer, level slog.Level) *Logger {
	return NewLogger(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}))
}

func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("解析日志记录失败: %v, 原始内容: %s", err, buf.String())
	}
	return record
}

func TestLoggerLevels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelDebug)
	log.Log(slog.LevelInfo, "hello", "k", "v")

	record := decodeRecord(t, &buf)
	if record["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", record["msg"])
	}
	if record["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", record["level"])
	}
	if record["k"] != "v" {
		t.Errorf("属性 k = %v, want v", record["k"])
	}
}

func TestLoggerAllLevels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelDebug)
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("日志条数 = %d, want 4", len(lines))
	}

	wantLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("解析第 %d 条日志失败: %v", i, err)
		}
		if record["level"] != wantLevels[i] {
			t.Errorf("第 %d 条日志级别 = %v, want %s", i, record["level"], wantLevels[i])
		}
	}
}

func TestLoggerLevelFilter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelWarn)
	log.Debug("skip")
	log.Info("skip")
	if buf.Len() != 0 {
		t.Fatalf("低于最低级别的日志未被过滤: %s", buf.String())
	}

	log.Warn("keep")
	if !strings.Contains(buf.String(), "keep") {
		t.Errorf("WARN 日志未输出: %s", buf.String())
	}
}

func TestLoggerSourceLocation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	newTestLogger(&buf, slog.LevelInfo).Info("with-source")

	record := decodeRecord(t, &buf)
	source, ok := record["source"].(map[string]any)
	if !ok {
		t.Fatalf("日志记录缺少 source 字段: %s", buf.String())
	}

	file, _ := source["file"].(string)
	if !strings.HasSuffix(file, "logger_test.go") {
		t.Fatalf("source = %v, 原始记录 = %s", source, buf.String())
	}
}

func TestLoggerWithAndGroup(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelInfo).With("service", "demo").WithGroup("req")
	log.Info("m", "id", 1)

	record := decodeRecord(t, &buf)
	if record["service"] != "demo" {
		t.Errorf("预设属性 service = %v, want demo", record["service"])
	}

	group, ok := record["req"].(map[string]any)
	if !ok {
		t.Fatalf("分组字段 req 缺失: %s", buf.String())
	}
	if group["id"] != float64(1) {
		t.Errorf("分组属性 req.id = %v, want 1", group["id"])
	}
}

func TestLoggerGetHandler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelInfo)
	if log.GetHandler() == nil {
		t.Fatal("GetHandler 返回 nil")
	}
}
