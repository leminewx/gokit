package ini

import (
	"testing"
)

// 测试 SetValue 和 GetValue
func TestSetAndGet(t *testing.T) {
	ini := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}

	// 设置新节和键
	ini.SetValue("stream", "camera_id", "c002")
	ini.SetValue("stream", "fps", "30")
	ini.SetValue("", "global_key", "global_val")

	val, ok := ini.GetValue("stream", "camera_id")
	if !ok || val != "c002" {
		t.Errorf("expected c002, got %s", val)
	}
	val, ok = ini.GetValue("stream", "fps")
	if !ok || val != "30" {
		t.Errorf("expected 30, got %s", val)
	}
	val, ok = ini.GetValue("", "global_key")
	if !ok || val != "global_val" {
		t.Errorf("expected global_val, got %s", val)
	}

	// 修改已存在的键
	ini.SetValue("stream", "camera_id", "c003")
	val, _ = ini.GetValue("stream", "camera_id")
	if val != "c003" {
		t.Errorf("after modify, expected c003, got %s", val)
	}
}

// 测试删除节和删除键
func TestRemove(t *testing.T) {
	ini := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}
	ini.SetValue("sec1", "k1", "v1")
	ini.SetValue("sec2", "k2", "v2")
	ini.SetValue("sec3", "k3", "v3")

	ini.RemoveSections("sec1", "sec3")
	if _, ok := ini.SectionMap["sec1"]; ok {
		t.Error("sec1 should be removed")
	}
	if _, ok := ini.SectionMap["sec3"]; ok {
		t.Error("sec3 should be removed")
	}
	if _, ok := ini.SectionMap["sec2"]; !ok {
		t.Error("sec2 should remain")
	}
	if len(ini.Sections) != 1 || ini.Sections[0].Name != "sec2" {
		t.Errorf("sections count wrong, got %d", len(ini.Sections))
	}

	// 删除键
	ini.SetValue("test", "a", "1")
	ini.SetValue("test", "b", "2")
	ini.SetValue("test", "c", "3")
	ini.RemoveKeys("test", "a", "c")
	sec := ini.SectionMap["test"]
	if _, ok := sec.Keys["a"]; ok {
		t.Error("key a should be removed")
	}
	if _, ok := sec.Keys["c"]; ok {
		t.Error("key c should be removed")
	}
	if _, ok := sec.Keys["b"]; !ok {
		t.Error("key b should remain")
	}
	if len(sec.Order) != 1 || sec.Order[0] != "b" {
		t.Errorf("order wrong, got %v", sec.Order)
	}
}

// 定义用于结构体映射的结构体
type TestStream struct {
	CameraID     string `ini:"camera_id"`
	InputStream  string `ini:"input_stream_addr"`
	Timefile     string `ini:"timefile"`
	DropInterval int    `ini:"drop_interval"`
	ExposureTime bool   `ini:"exposure_time"`
}

type TestDetect struct {
	Type int       `ini:"type"`
	Conf []float64 `ini:"conf"` // 数组作为字符串存储
}

type TestCalib struct {
	Type int    `ini:"type"`
	Path string `ini:"path"`
}

type TestVroll struct {
	IsRecord   bool   `ini:"is_record"`
	RegisterID string `ini:"registerid"`
}

type TestConfig struct {
	Stream    TestStream `ini:"stream"`
	Detect    TestDetect `ini:"detect"`
	Calib     TestCalib  `ini:"calib"`
	Vroll     TestVroll  `ini:"vroll"`
	GlobalKey string     `ini:"global_key"` // 期望在全局节中
}
