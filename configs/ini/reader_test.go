package ini

import (
	"testing"
	"time"
)

func TestLoadFromString(t *testing.T) {
	content := `
# 全局配置
name = "demo"   # 行内注释
url = http://example.com/a#b ; 未以空白分隔的 # 保留

[cache]
ttl = 30s
size=1024
tags = [ "a", "b" ]
`
	cfg, err := LoadFromString(content)
	if err != nil {
		t.Fatalf("LoadFromString error: %v", err)
	}

	if got := cfg.GetString("", "name"); got != "demo" {
		t.Fatalf("name = %q, want demo", got)
	}
	if got := cfg.GetString("", "url"); got != "http://example.com/a#b" {
		t.Fatalf("url = %q, want anchor preserved", got)
	}
	if got := cfg.GetString("cache", "size"); got != "1024" {
		t.Fatalf("cache.size = %q, want 1024", got)
	}

	ttl, err := cfg.GetDuration("cache", "ttl")
	if err != nil || ttl != 30*time.Second {
		t.Fatalf("cache.ttl = %v %v, want 30s", ttl, err)
	}

	tags, err := cfg.GetStrings("cache", "tags")
	if err != nil {
		t.Fatalf("GetStrings error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("cache.tags = %v, want [a b]", tags)
	}
}

// 测试从 INI 文件加载
// 测试数据统一存放在仓库根的 harness/testdata，不散落在各包或根目录
func TestLoadFromINI(t *testing.T) {
	ini, err := LoadFromINI("../../harness/testdata/test.ini")
	if err != nil {
		t.Fatalf("LoadFromINI error: %v", err)
	}

	// 验证节数量（8 个命名节）
	expectedSections := 8 // 包含全局节（空字符串）
	if len(ini.Sections) != expectedSections {
		t.Errorf("expected %d sections, got %d", expectedSections, len(ini.Sections))
	}

	// 验证关键值
	tests := []struct {
		section, key, expected string
	}{
		{"stream", "camera_id", "c001"},
		{"stream", "input_stream_addr", "rtsp://user:password@192.0.2.15:8554/217"},
		{"detect", "type", "1"},
		{"detect", "conf", "[0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3]"},
		{"event", "event", "false"},
		{"calib", "path", "./config/217c001.json"},
		{"vroll", "registerid", "replace-me"},
	}
	for _, tt := range tests {
		val, ok := ini.GetValue(tt.section, tt.key)
		if !ok {
			t.Errorf("key %s.%s not found", tt.section, tt.key)
		}
		if val != tt.expected {
			t.Errorf("key %s.%s = %q, want %q", tt.section, tt.key, val, tt.expected)
		}
	}
}
