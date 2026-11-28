package ini

import (
	"reflect"
	"testing"
	"time"
)

func TestUnmarshalExtendedTypes(t *testing.T) {
	type options struct {
		Timeout time.Duration `ini:"timeout"`
		Retries *int          `ini:"retries"`
		Ratio   float32       `ini:"ratio"`
		Tags    []string      `ini:"tags"`
		Start   time.Time     `ini:"start"`
	}

	cfg, err := LoadFromString(`
timeout = 1m30s
retries = 3
ratio = 0.5
tags =
start = 2026-11-22T10:00:00Z
`)
	if err != nil {
		t.Fatal(err)
	}

	var app options
	if err := cfg.Unmarshal(&app); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if app.Timeout != 90*time.Second {
		t.Fatalf("Timeout = %v, want 1m30s", app.Timeout)
	}
	if app.Retries == nil || *app.Retries != 3 {
		t.Fatalf("Retries = %v, want 3", app.Retries)
	}
	if app.Ratio != 0.5 {
		t.Fatalf("Ratio = %v, want 0.5", app.Ratio)
	}
	if len(app.Tags) != 0 {
		t.Fatalf("Tags = %v, want empty", app.Tags)
	}
	if app.Start.IsZero() {
		t.Fatal("Start should be parsed via TextUnmarshaler")
	}
}

// 测试 Unmarshal（INI → 结构体）
func TestUnmarshal(t *testing.T) {
	ini, err := LoadFromINI("../../harness/testdata/test.ini")
	if err != nil {
		t.Fatalf("LoadFromINI error: %v", err)
	}

	var app TestConfig
	if err := ini.Unmarshal(&app); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// 验证值
	if app.Stream.CameraID != "c001" {
		t.Errorf("Stream.CameraID = %s, want c001", app.Stream.CameraID)
	}
	if app.Stream.DropInterval != 2 {
		t.Errorf("Stream.DropInterval = %d, want 2", app.Stream.DropInterval)
	}
	if app.Stream.ExposureTime != false {
		t.Errorf("Stream.ExposureTime = %v, want false", app.Stream.ExposureTime)
	}
	if app.Detect.Type != 1 {
		t.Errorf("Detect.Type = %d, want 1", app.Detect.Type)
	}
	expectedConf := []float64{0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3}
	if !reflect.DeepEqual(app.Detect.Conf, expectedConf) {
		t.Errorf("Detect.Conf = %v, want %v", app.Detect.Conf, expectedConf)
	}
	if app.Calib.Path != "./config/217c001.json" {
		t.Errorf("Calib.Path = %s, want ./config/217c001.json", app.Calib.Path)
	}
	if app.Vroll.RegisterID != "replace-me" {
		t.Errorf("Vroll.RegisterID = %s, want replace-me", app.Vroll.RegisterID)
	}
	// 全局节在样例中不存在，期望零值
	if app.GlobalKey != "" {
		t.Errorf("GlobalKey = %s, want empty", app.GlobalKey)
	}
}

// 测试嵌套结构体（自动识别为节）
func TestNestedStruct(t *testing.T) {
	// 测试 Unmarshal 嵌套
	ini := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}
	ini.SetValue("db", "host", "localhost")
	ini.SetValue("db", "port", "3306")

	type DB struct {
		Host string `ini:"host"`
		Port int    `ini:"port"`
	}
	type App struct {
		Database DB `ini:"db"`
	}
	var app App
	if err := ini.Unmarshal(&app); err != nil {
		t.Fatal(err)
	}
	if app.Database.Host != "localhost" {
		t.Errorf("Database.Host = %s, want localhost", app.Database.Host)
	}
	if app.Database.Port != 3306 {
		t.Errorf("Database.Port = %d, want 3306", app.Database.Port)
	}

	// 测试 Marshal 嵌套
	app2 := App{
		Database: DB{Host: "127.0.0.1", Port: 5432},
	}
	ini2, err := Marshal(app2)
	if err != nil {
		t.Fatal(err)
	}
	val, _ := ini2.GetValue("db", "host")
	if val != "127.0.0.1" {
		t.Errorf("db.host = %s, want 127.0.0.1", val)
	}
	val, _ = ini2.GetValue("db", "port")
	if val != "5432" {
		t.Errorf("db.port = %s, want 5432", val)
	}
}
