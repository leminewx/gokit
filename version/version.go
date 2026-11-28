// Package version 提供版本号的解析、比较与「版本区间表达式」匹配。
//
// 同一版本号在比较时可能承担两类职责，但比较规则必须一致：
//
//   - 选择实现：按版本挑选配置/协议/接口实现；
//   - 声明适配：用区间表达式声明「某组件适配哪些版本」。
//
// 两者各写一套比较实现时，对「段数不同」「带发布后缀」的处理极易分叉，
// 最终出现同一版本在不同流程中得出不同结论。本包把规则收敛为一处：
//
//   - 先归一化（去空白、转小写、去 v 前缀）；实际版本额外抽取版本核心，
//     发布后缀只用于标识构建批次，不参与比较
//     （app_V7.0.0.0_20260401 == v7.0.0.0）；
//   - 按 . 分段逐段比较，段数不同时缺失段视为数字 0（因此 v6.4 == v6.4.0）；
//   - 两段都是纯数字时按数值比较（v6.10 > v6.9）；
//   - 否则按字典序比较（仅用于非标准版本串，如 1a.2）。
//
// 区间表达式支持以下写法（详见 expr.go 的包内说明）：
//
//	=v3.1.2           精确匹配（= 可省略）
//	=v3.x             前缀通配（等价于 v3 整个系列）
//	<v3.x             小于 v3 系列（x 取下界）
//	≤v3.1.2           小于等于（也接受 <=）
//	>v6.0.x           大于 v6.0 系列（x 取上界）
//	≥v6.4.0 <v8.0.0   行内 AND（空格 / 逗号分隔）
//	多行               行间 OR（任意一行命中即通过）
//	* 或留空           不限
//
// 本包只依赖标准库，不引入任何第三方依赖。
package version

import (
	"regexp"
	"strconv"
	"strings"
)

// versionCore 版本核心串抽取：app_V7.0.0.0_20260401 -> v7.0.0.0
var versionCore = regexp.MustCompile(`(?i)v?\d+(?:\.\d+)+`)

// Extract 从带前后缀的版本串里抽取版本核心（小写、保留 v 前缀）。
// 未匹配到版本核心时返回空串（调用方自行决定回退行为）。
func Extract(raw string) string {
	return strings.ToLower(versionCore.FindString(strings.TrimSpace(raw)))
}

// Normalize 归一化版本写法：去空白、转小写、去 v 前缀（不抽取核心）。
func Normalize(raw string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "v")
}

// NormalizeActual 归一化「实际版本」：先从可能带前后缀的原始串里抽取版本核心，再归一化。
//
// 实际版本常形如 app_V7.0.0.0_20260401，直接按段拆分会让首段变成
// "app_v7" 参与比较，得出错误结论，因此实际值必须先抽取核心。
// 只对实际值抽取、不对表达式抽取：表达式里的 x 通配不能被截断。
func NormalizeActual(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	if match := Extract(raw); match != "" {
		raw = match
	}
	return Normalize(raw)
}

// SplitSegments 按 . 拆分已归一化的版本串。
func SplitSegments(ver string) []string {
	return strings.Split(Normalize(ver), ".")
}

// Compare 比较两个版本号：left 更小返回 -1，相等返回 0，更大返回 1。
//
// 入参可以是带前后缀的原始串（如 app_V7.0.0.0_20260401），内部按实际版本处理。
func Compare(left, right string) int {
	a := SplitSegments(NormalizeActual(left))
	b := SplitSegments(NormalizeActual(right))

	total := len(a)
	if len(b) > total {
		total = len(b)
	}

	for index := 0; index < total; index++ {
		var av, bv string
		if index < len(a) {
			av = a[index]
		}
		if index < len(b) {
			bv = b[index]
		}
		if result := compareSegment(av, bv); result != 0 {
			return result
		}
	}
	return 0
}

// IsEqual 判断两个版本号是否相等（语义与 Compare == 0 一致）。
func IsEqual(left, right string) bool {
	return Compare(left, right) == 0
}

// compareSegment 单段比较：兼容缺失段（空串按数字 0 处理）；
// 同为纯数字按数值，否则按字典序（非标准版本串兜底）。
func compareSegment(left, right string) int {
	if left == right {
		return 0
	}

	switch {
	case IsDigits(left) && IsDigits(right):
		return compareNumeric(left, right)
	case left == "" && IsDigits(right):
		// 缺失段等价于 0（v6.4 == v6.4.0）
		return compareNumeric("0", right)
	case right == "" && IsDigits(left):
		return compareNumeric(left, "0")
	}

	// 含非数字段（如 0_shiyan）或一侧为空：按字典序
	return strings.Compare(left, right)
}

// compareNumeric 纯数字段按数值比较（段内位数可能很长，先比长度再比字典序即可）。
func compareNumeric(left, right string) int {
	left = strings.TrimLeft(left, "0")
	right = strings.TrimLeft(right, "0")
	if left == "" {
		left = "0"
	}
	if right == "" {
		right = "0"
	}

	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	}
	return strings.Compare(left, right)
}

// IsDigits 报告字符串是否为纯数字版本段（空串不算）。
func IsDigits(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ParseInt 解析纯数字版本段；非数字返回 0 与 false。
func ParseInt(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
