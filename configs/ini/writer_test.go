package ini

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteToStringRoundTrip(t *testing.T) {
	cfg, err := LoadFromString(`
name = demo

[cache]
ttl = 1m
`)
	if err != nil {
		t.Fatal(err)
	}

	text := cfg.String()
	if !strings.Contains(text, "name = demo") || !strings.Contains(text, "[cache]") {
		t.Fatalf("unexpected text output:\n%s", text)
	}

	reloaded, err := LoadFromString(text)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if reloaded.GetString("cache", "ttl") != "1m" {
		t.Fatalf("cache.ttl lost after round trip: %q", reloaded.GetString("cache", "ttl"))
	}

	written, err := cfg.WriteTo(&strings.Builder{})
	if err != nil || written != int64(len(text)) {
		t.Fatalf("WriteTo = %d %v, want %d", written, err, len(text))
	}
}

// 测试保存和重新加载
func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "out.ini")

	ini := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}
	ini.SetValue("", "global", "gval")
	ini.SetValue("sec", "k1", "v1")

	if err := ini.Save(tmpFile); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// 重新加载
	ini2, err := LoadFromINI(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromINI error: %v", err)
	}
	val, _ := ini2.GetValue("", "global")
	if val != "gval" {
		t.Errorf("global expected gval, got %s", val)
	}
	val, _ = ini2.GetValue("sec", "k1")
	if val != "v1" {
		t.Errorf("sec.k1 expected v1, got %s", val)
	}
}
