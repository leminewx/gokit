package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSplitSections(t *testing.T) {
	t.Parallel()

	// 模拟远端脚本输出：前导噪声 + CRLF 换行 + 末尾无换行
	output := "warning: something\r\n" +
		"###gokit:hostname\r\nhost-a\r\n" +
		"###gokit:kernel\r\nLinux 5.15.148-tegra aarch64\r\n" +
		"###gokit:cpuinfo\r\n8\r\n" +
		"###gokit:empty\r\n"

	sections := splitSections(output)
	if len(sections) != 4 {
		t.Fatalf("分段数 = %d, want 4：%v", len(sections), sections)
	}

	cases := map[string]string{
		"hostname": "host-a\n",
		"kernel":   "Linux 5.15.148-tegra aarch64\n",
		"cpuinfo":  "8\n",
		"empty":    "\n", // 空分段仅包含标记行后的换行
	}

	for name, want := range cases {
		if got := sections[name]; got != want {
			t.Errorf("分段 %q = %q, want %q", name, got, want)
		}
	}

	if _, ok := sections["warning"]; ok {
		t.Error("标记行之前的内容不应被当作分段")
	}
}

func TestParseCPUCores(t *testing.T) {
	t.Parallel()

	cases := []struct {
		output string
		want   int
	}{
		{"96\n", 96},
		{"8", 8},
		{"0\n", 0},
		{"", 0},
		{"abc\n", 0},
		{"-1\n", 0},
	}

	for _, tc := range cases {
		if got := parseCPUCores(tc.output); got != tc.want {
			t.Errorf("parseCPUCores(%q) = %d, want %d", tc.output, got, tc.want)
		}
	}
}

func TestCollectorScript(t *testing.T) {
	t.Parallel()

	collector := NewCollector(nil, WithSampleInterval(3*time.Second))
	script := collector.script()

	// 各段标记必须齐全，否则 Collect 无法切分结果
	for _, name := range []string{"hostname", "kernel", "uptime", "loadavg", "cpuinfo", "stat1", "stat2", "meminfo", "netdev", "df"} {
		if !strings.Contains(script, "'"+_SECTION_PREFIX+name+"'") {
			t.Errorf("采集脚本缺少分段 %q：\n%s", name, script)
		}
	}

	// 远端 locale 会翻译 df/top 表头，脚本必须固定为 C
	if !strings.Contains(script, "export LC_ALL=C") {
		t.Errorf("采集脚本应固定 LC_ALL=C：\n%s", script)
	}

	if !strings.Contains(script, "sleep 3\n") {
		t.Errorf("采样间隔 3s 未写入脚本：\n%s", script)
	}
	if strings.Count(script, "grep '^cpu' /proc/stat") != 2 {
		t.Errorf("需要两次 /proc/stat 采样：\n%s", script)
	}
	if !strings.Contains(script, "df -P -k") {
		t.Errorf("磁盘应使用 POSIX 输出避免折行：\n%s", script)
	}

	// BusyBox 的 sleep 不支持小数秒，小于 1 秒的间隔必须抬升为 1 秒
	fast := NewCollector(nil, WithSampleInterval(100*time.Millisecond))
	if !strings.Contains(fast.script(), "sleep 1\n") {
		t.Errorf("小于 1 秒的采样间隔应抬升为 1 秒：\n%s", fast.script())
	}

	// 带小数的间隔向上取整，保证实际采样时长不短于配置值
	fractional := NewCollector(nil, WithSampleInterval(1500*time.Millisecond))
	if !strings.Contains(fractional.script(), "sleep 2\n") {
		t.Errorf("1.5s 采样间隔应向上取整为 2 秒：\n%s", fractional.script())
	}
}

func TestSampleSeconds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		interval time.Duration
		want     int
	}{
		{2 * time.Second, 2},
		{1500 * time.Millisecond, 2},
		{1001 * time.Millisecond, 2},
		{100 * time.Millisecond, 1},
		{0, 1},
		{-time.Second, 1},
	}

	for _, tc := range cases {
		if got := (&Collector{interval: tc.interval}).sampleSeconds(); got != tc.want {
			t.Errorf("sampleSeconds(%v) = %d, want %d", tc.interval, got, tc.want)
		}
	}
}

func TestNewCollectorDefaults(t *testing.T) {
	t.Parallel()

	collector := NewCollector(nil)
	if collector.interval != _DEFAULT_SAMPLE_INTERVAL {
		t.Errorf("默认采样间隔 = %v, want %v", collector.interval, _DEFAULT_SAMPLE_INTERVAL)
	}
	if collector.timeout != _DEFAULT_COLLECT_TIMEOUT {
		t.Errorf("默认超时 = %v, want %v", collector.timeout, _DEFAULT_COLLECT_TIMEOUT)
	}

	// 非法选项值不应覆盖默认值
	collector = NewCollector(nil, WithSampleInterval(0), WithCollectTimeout(-time.Second))
	if collector.interval != _DEFAULT_SAMPLE_INTERVAL || collector.timeout != _DEFAULT_COLLECT_TIMEOUT {
		t.Errorf("非法选项覆盖了默认值：interval=%v timeout=%v", collector.interval, collector.timeout)
	}
}

func TestCollectWithoutClient(t *testing.T) {
	t.Parallel()

	for _, collector := range []*Collector{NewCollector(nil), nil} {
		if _, err := collector.Collect(context.Background()); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("Collect() 错误 = %v, want ErrInvalidArgument", err)
		}
	}
}
