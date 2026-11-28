package version

import "testing"

// TestCompare 统一比较口径：v 前缀、大小写、段数差异、发布后缀都不影响结论。
func TestCompare(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left, right string
		want        int
	}{
		{"v7.0.0.0", "v6.4.0", 1},
		{"V7.0.0.0", "v7.0.0.0", 0},
		{"v6.0", "v6.0.1", -1},
		{"v6.0.10", "v6.0.9", 1},
		// 缺失段等价于 0：v6.4 与 v6.4.0.0 是同一版本
		{"v6.4.0.0", "v6.4.0", 0},
		{"v6.4", "v6.4.0.0", 0},
		// 前导零按数值比较
		{"v6.04", "v6.4", 0},
		// 带发布后缀的完整串按实际版本比较
		{"app_V7.0.0.0_20260401", "v6.4.0.0", 1},
		{"app_V6.4.0.0_20260101", "app_V6.4.0.0_20261231", 0},
	}
	for _, item := range cases {
		if got := Compare(item.left, item.right); got != item.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", item.left, item.right, got, item.want)
		}
	}
}

// TestCompareIgnoresBuildSuffix 发布后缀只标识构建批次，不参与比较。
func TestCompareIgnoresBuildSuffix(t *testing.T) {
	t.Parallel()

	if got := Compare("v6.4.0.0_shiyan", "v6.4.0.0"); got != 0 {
		t.Errorf("发布后缀不应影响比较：Compare = %d, want 0", got)
	}
	if got := Compare("v6.4.0.0", "v6.4.0.0_shiyan"); got != 0 {
		t.Errorf("发布后缀不应影响比较：Compare = %d, want 0", got)
	}
}

// TestIsEqual 与 Compare == 0 语义一致。
func TestIsEqual(t *testing.T) {
	t.Parallel()

	if !IsEqual("v6.4.0.0", "V6.4.0") {
		t.Error("IsEqual(v6.4.0.0, V6.4.0) 应为 true")
	}
	if IsEqual("v6.4.0.0", "v6.4.0.1") {
		t.Error("IsEqual(v6.4.0.0, v6.4.0.1) 应为 false")
	}
}

// TestNormalizeActual 只对实际值抽取版本核心，前后缀不参与比较。
func TestNormalizeActual(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"app_V7.0.0.0_20260401": "7.0.0.0",
		"  V6.4.0.0  ":          "6.4.0.0",
		"v6.4":                  "6.4",
		"":                      "",
	}
	for raw, want := range cases {
		if got := NormalizeActual(raw); got != want {
			t.Errorf("NormalizeActual(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestExtract 抽取失败返回空串，调用方需自行回退。
func TestExtract(t *testing.T) {
	t.Parallel()

	if got := Extract("app_V7.0.0.0_20260401"); got != "v7.0.0.0" {
		t.Errorf("Extract = %q, want v7.0.0.0", got)
	}
	if got := Extract("no-version-here"); got != "" {
		t.Errorf("Extract 未匹配应返回空串，got %q", got)
	}
}

// TestSplitSegments 拆分前先归一化（去空白、转小写、去 v 前缀）。
func TestSplitSegments(t *testing.T) {
	t.Parallel()

	got := SplitSegments(" V7.0.0 ")
	want := []string{"7", "0", "0"}
	if len(got) != len(want) {
		t.Fatalf("SplitSegments = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("SplitSegments = %v, want %v", got, want)
		}
	}
}

// TestParseIntAndIsDigits 版本段解析的边界：空串与前导零。
func TestParseIntAndIsDigits(t *testing.T) {
	t.Parallel()

	if !IsDigits("007") {
		t.Error("IsDigits(007) 应为 true")
	}
	if IsDigits("") || IsDigits("7a") {
		t.Error("IsDigits 对空串与非纯数字应返回 false")
	}
	if n, ok := ParseInt("10"); !ok || n != 10 {
		t.Errorf("ParseInt(10) = (%d, %v), want (10, true)", n, ok)
	}
	if _, ok := ParseInt("1a"); ok {
		t.Error("ParseInt(1a) 应返回 false")
	}
}
