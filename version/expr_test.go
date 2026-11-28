package version

import (
	"errors"
	"testing"
)

// TestExprMatchComparators 四种比较写法（含 x 通配）的边界判定
func TestExprMatchComparators(t *testing.T) {
	t.Parallel()

	cases := []struct {
		expr   string
		actual string
		want   bool
	}{
		// <v3.x：小于 v3 整个系列 → v2.x 满足，v3.x 一律不满足
		{"<v3.x", "v2.9.0", true},
		{"<v3.x", "v3.0.0", false},
		{"<v3.x", "v3.5.1", false},

		// <=v3.1.2：普通小于等于
		{"<=v3.1.2", "v3.1.2", true},
		{"<=v3.1.2", "v3.1.3", false},

		// >v6.0.x：大于 v6.0 整个系列 → v6.1.0 起才满足
		{">v6.0.x", "v6.0.0", false},
		{">v6.0.x", "v6.0.5", false},
		{">v6.0.x", "v6.1.0", true},

		// >=v6.0.0.x：大于等于 v6.0.0 系列下界
		{">=v6.0.0.x", "v5.9.9", false},
		{">=v6.0.0.x", "v6.0.0.0", true},
		{">=v6.0.0.x", "v6.0.0.5", true},

		// 前缀通配（无操作符）= 整个系列
		{"v3.x", "v3.0.0", true},
		{"v3.x", "v3.9.9", true},
		{"v3.x", "v4.0.0", false},

		// 精确
		{"v7.0.0.0", "v7.0.0.0", true},
		{"v7.0.0.0", "V7.0.0.0", true},
		{"v7.0.0.0", "v7.0.0.1", false},

		// 多条件 AND（空格 / 逗号两种分隔）
		{">=v6.4.0 <v8.0.0", "v7.0.0.0", true},
		{">=v6.4.0 <v8.0.0", "v8.0.0.0", false},
		{">=v6.4.0 <v8.0.0", "v6.3.9", false},
		{">=v6.4.0,<v8.0.0", "v6.4.0", true},

		// 操作符与版本号之间有空格
		{">= v6.4.0", "v6.4.0", true},
	}
	for _, item := range cases {
		got, err := Match(item.expr, item.actual)
		if err != nil {
			t.Errorf("Match(%q, %q) 返回错误: %v", item.expr, item.actual, err)
			continue
		}
		if got != item.want {
			t.Errorf("Match(%q, %q) = %v, want %v", item.expr, item.actual, got, item.want)
		}
	}
}

// TestMatchUnrestricted 未声明适配范围时保持历史行为（不限制）
func TestMatchUnrestricted(t *testing.T) {
	t.Parallel()

	for _, expr := range []string{"", "  ", "*", "any", "ALL"} {
		got, err := Match(expr, "v1.2.3")
		if err != nil || !got {
			t.Errorf("Match(%q, v1.2.3) = (%v, %v), want (true, nil)", expr, got, err)
		}
		if !IsUnlimited(expr) {
			t.Errorf("IsUnlimited(%q) = false, want true", expr)
		}
	}
}

// TestMatchEmptyActual 拿不到实际版本时必须报错，不能静默放行
func TestMatchEmptyActual(t *testing.T) {
	t.Parallel()

	got, err := Match(">=v6.4.0", "")
	if !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("Match 空版本应返回 ErrEmptyVersion，实际 got=%v err=%v", got, err)
	}
	if got {
		t.Errorf("Match 空版本不应返回 true")
	}
}

// TestValidateRejects 非法表达式在录入阶段就要被拦住
func TestValidateRejects(t *testing.T) {
	t.Parallel()

	bad := []string{
		">=",               // 缺版本号
		"v3..1",            // 空段
		"v3.x.1",           // 通配符不在末段
		"v3.x.x",           // 通配符出现多次
		"abc",              // 不是版本号
		"<",                // 缺版本号
		">=v6.4.0 >",       // 第二个条件缺版本号
		"v7.0.0.0",         // 缺比较符（录入规范要求首字符为 =、≥、≤、>、<）
		"v3.x",             // 缺比较符
		">=v6.4.0\nv8.0.0", // 多区间中第二行缺比较符
	}
	for _, expr := range bad {
		err := Validate(expr)
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error", expr)
			continue
		}
		if !errors.Is(err, ErrInvalidVersionExpr) && !errors.Is(err, ErrInvalidVersion) {
			t.Errorf("Validate(%q) 错误未携带哨兵: %v", expr, err)
		}
	}
}

// TestValidateAccepts 合法表达式（含未声明、Unicode 比较符、多区间）
func TestValidateAccepts(t *testing.T) {
	t.Parallel()

	good := []string{
		"", "*", "=v7.0.0.0", "<v3.x", "≤v3.1.2", ">v6.0.x", "≥v6.0.0.x",
		"=v3.x", ">=v6.4.0 <v8.0.0", ">=v6.4.0,<v8.0.0",
		"≥v6.4.0 <v8.0.0",
		"≥v6.4.0 <v8.0.0\n≥v9.0.0 <v10.0.0",
		"≥v6.4.0 <v8.0.0\r\n\r\n≥v9.0.0 <v10.0.0",
		"＞v6.0.0", "＜＝v3.1.2", "＝v7.0.0.0",
	}
	for _, expr := range good {
		if err := Validate(expr); err != nil {
			t.Errorf("Validate(%q) 返回错误: %v", expr, err)
		}
	}
}

// TestMatchMultiRange 多行表示多个区间取并集（行间 OR），行内仍为 AND
func TestMatchMultiRange(t *testing.T) {
	t.Parallel()

	expr := "≥v6.4.0 <v8.0.0\n≥v9.0.0 <v10.0.0"
	cases := []struct {
		actual string
		want   bool
	}{
		{"v7.0.0.0", true},   // 命中第一个区间
		{"v9.5.0.0", true},   // 命中第二个区间
		{"v8.5.0.0", false},  // 两个区间之间（缺口）
		{"v6.3.0.0", false},  // 低于所有区间
		{"v10.0.0.0", false}, // 高于所有区间
	}
	for _, item := range cases {
		got, err := Match(expr, item.actual)
		if err != nil {
			t.Errorf("Match(%q, %q) 返回错误: %v", expr, item.actual, err)
			continue
		}
		if got != item.want {
			t.Errorf("Match(%q, %q) = %v, want %v", expr, item.actual, got, item.want)
		}
	}
}

// TestMatchUnicodeOperators ≥/≤ 与全角比较符和 ASCII 写法等价
func TestMatchUnicodeOperators(t *testing.T) {
	t.Parallel()

	cases := []struct {
		expr   string
		actual string
		want   bool
	}{
		{"≥v6.4.0", "v6.4.0", true},
		{"≥v6.4.0", "v6.3.9", false},
		{"≤v3.1.2", "v3.1.2", true},
		{"≤v3.1.2", "v3.1.3", false},
		{"＞v6.0.0", "v6.0.1", true},
		{"＜＝v3.1.2", "v3.1.2", true},
		{"＝v7.0.0.0", "v7.0.0.0", true},
	}
	for _, item := range cases {
		got, err := Match(item.expr, item.actual)
		if err != nil {
			t.Errorf("Match(%q, %q) 返回错误: %v", item.expr, item.actual, err)
			continue
		}
		if got != item.want {
			t.Errorf("Match(%q, %q) = %v, want %v", item.expr, item.actual, got, item.want)
		}
	}
}

// TestNormalizeExpr 落盘/回显写法：比较符统一 ≥/≤，多区间换行分隔
func TestNormalizeExpr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		expr string
		want string
	}{
		{"", ""},
		{"*", ""},
		{">=v4.0.0.0", "≥v4.0.0.0"},
		{"≥v4.0.0.0", "≥v4.0.0.0"},
		{" <= v3.1.2 ", "≤v3.1.2"},
		{"v4.0.0.0", "=v4.0.0.0"},
		{"=V4.0.0.0", "=v4.0.0.0"},
		{">=v6.4.0,<v8.0.0", "≥v6.4.0 <v8.0.0"},
		{"≥v6.4.0 <v8.0.0\r\n\r\n≥v9.0.0 <v10.0.0", "≥v6.4.0 <v8.0.0\n≥v9.0.0 <v10.0.0"},
	}
	for _, item := range cases {
		if got := NormalizeExpr(item.expr); got != item.want {
			t.Errorf("NormalizeExpr(%q) = %q, want %q", item.expr, got, item.want)
		}
	}
}

// TestExplain 规范化描述（供接口回显）
func TestExplain(t *testing.T) {
	t.Parallel()

	if got := Explain(""); got != "不限" {
		t.Errorf("Explain(\"\") = %q", got)
	}
	if got := Explain("<v3.x"); got != "< [v3.0, v3.999999]" {
		t.Errorf("Explain(<v3.x) = %q", got)
	}
	if got := Explain("v3.x"); got != "= [v3.0, v3.999999]" {
		t.Errorf("Explain(v3.x) = %q", got)
	}
	if got := Explain("≥v6.4.0\n≥v9.0.0"); got != "≥ v6.4.0 或 ≥ v9.0.0" {
		t.Errorf("Explain(多区间) = %q", got)
	}
}

// TestMatchDecoratedActual 实际版本常带前后缀（如 app_V7.0.0.0_20260401），
// 必须先抽取版本核心再比较；否则首段会被当成字符串参与字典序比较，结论全错。
func TestMatchDecoratedActual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		expr   string
		actual string
		want   bool
	}{
		{">=v6.4.0 <v8.0.0", "app_V7.0.0.0_20260401", true},
		{">=v6.4.0", "app_V6.3.0.0_20260401", false},
		{"<v3.x", "app_V2.9.0.0_20260401", true},
		{"v7.x", "app_V7.0.0.0_20260401", true},
		{"v7.0.0.0", "app_V7.0.0.0_20260401", true},
		{"v7.0.0.0", "V7.0.0.0", true},
	}
	for _, item := range cases {
		got, err := Match(item.expr, item.actual)
		if err != nil {
			t.Errorf("Match(%q, %q) 返回错误: %v", item.expr, item.actual, err)
			continue
		}
		if got != item.want {
			t.Errorf("Match(%q, %q) = %v, want %v", item.expr, item.actual, got, item.want)
		}
	}
}
