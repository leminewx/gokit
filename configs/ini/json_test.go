package ini

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestJSONOrderDeterministic(t *testing.T) {
	data := []byte(`{"zeta":{"b":"2","a":"1"},"alpha":{"y":"2","x":"1"},"global":{"k":"v"}}`)

	first := &INI{}
	if err := first.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}

	names := first.GetSectionNames()
	want := []string{"", "alpha", "zeta"}
	if len(names) != len(want) {
		t.Fatalf("GetSectionNames = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("GetSectionNames = %v, want %v", names, want)
		}
	}

	keys := first.GetKeys("zeta")
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("zeta keys = %v, want [a b]", keys)
	}

	// 重复解析结果保持一致
	second := &INI{}
	if err := second.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if got, wantText := second.String(), first.String(); got != wantText {
		t.Fatalf("non-deterministic output:\n%s\n---\n%s", got, wantText)
	}
}

// 测试 JSON 序列化/反序列化（通过 json.Marshal/Unmarshal）
func TestJSONMarshalUnmarshal(t *testing.T) {
	ini := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}
	ini.SetValue("", "global_key", "global_val")
	ini.SetValue("sec1", "key1", "val1")
	ini.SetValue("sec2", "key2", "val2")

	// 序列化
	data, err := json.Marshal(ini)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	// 反序列化到新对象
	ini2 := &INI{}
	if err := json.Unmarshal(data, ini2); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// 比较
	if len(ini2.Sections) != 3 { // global + sec1 + sec2
		t.Errorf("expected 3 sections, got %d", len(ini2.Sections))
	}
	val, _ := ini2.GetValue("", "global_key")
	if val != "global_val" {
		t.Errorf("expected global_val, got %s", val)
	}
	val, _ = ini2.GetValue("sec1", "key1")
	if val != "val1" {
		t.Errorf("expected val1, got %s", val)
	}
}

// 测试从 JSON 文件加载（LoadFromJSON）
func TestLoadFromJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")
	jsonContent := `{"global":{"key":"gval"},"db":{"host":"localhost","port":"3306"}}`
	if err := os.WriteFile(tmpFile, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	ini, err := LoadFromJSON(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromJSON error: %v", err)
	}
	val, _ := ini.GetValue("", "key")
	if val != "gval" {
		t.Errorf("global.key = %s, want gval", val)
	}
	val, _ = ini.GetValue("db", "host")
	if val != "localhost" {
		t.Errorf("db.host = %s, want localhost", val)
	}
	val, _ = ini.GetValue("db", "port")
	if val != "3306" {
		t.Errorf("db.port = %s, want 3306", val)
	}
}

// 测试 Marshal（结构体 → INI）
func TestMarshal(t *testing.T) {
	app := TestConfig{
		Stream: TestStream{
			CameraID:     "c999",
			InputStream:  "rtsp://test",
			Timefile:     "time.log",
			DropInterval: 5,
			ExposureTime: true,
		},
		Detect: TestDetect{
			Type: 2,
			Conf: []float64{1.0, 2.0},
		},
		Calib: TestCalib{
			Type: 3,
			Path: "/path/to/calib.json",
		},
		Vroll: TestVroll{
			IsRecord:   true,
			RegisterID: "abc123",
		},
		GlobalKey: "global_value",
	}

	ini, err := Marshal(app)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// 验证值
	val, _ := ini.GetValue("stream", "camera_id")
	if val != "c999" {
		t.Errorf("stream.camera_id = %s, want c999", val)
	}
	val, _ = ini.GetValue("stream", "drop_interval")
	if val != "5" {
		t.Errorf("stream.drop_interval = %s, want 5", val)
	}
	val, _ = ini.GetValue("detect", "conf")
	if reflect.DeepEqual(val, []float64{1.0, 2.0}) {
		t.Errorf("detect.conf = %s, want [1.0, 2.0]", val)
	}
	val, _ = ini.GetValue("", "global_key")
	if val != "global_value" {
		t.Errorf("global_key = %s, want global_value", val)
	}
}
