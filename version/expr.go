package version

import (
	"fmt"
	"strings"
	"unicode"
)

// 本文件实现「版本区间表达式」：用一段文本声明「适配哪些版本」，而不是枚举版本号。
//
// 单点取值表达不了「v6.4 到 v8.0 都适配」这类事实，逐个列举又要求每次发版
// 都改声明，因此改用区间表达式：
//
//	=v3.1.2           精确匹配（= 可省略）
//	=v3.x             前缀通配（等价于 v3 整个系列）
//	<v3.x             小于 v3 系列（x 取下界，即 < v3.0.0）
//	≤v3.1.2           小于等于（也接受 <=）
//	>v6.0.x           大于 v6.0 系列（x 取上界，即 >= v6.1.0）
//	≥v6.0.0.x         大于等于 v6.0.0 系列（x 取下界，也接受 >=）
//	≥v6.4.0 <v8.0.0   同一行多个条件用空格 / 逗号分隔，全部满足才通过（行内 AND）
//
// 多行写法表示「多个区间取并集」，任意一行满足即通过（行间 OR）：
//
//	≥v6.4.0 <v8.0.0
//	≥v9.0.0 <v10.0.0
//
//	* 或留空          不限
//
// 录入时（Validate）要求每个条件都以 =、≥、≤、>、< 之一开头，比较符统一
// 归一化为 ≥/≤ 后落盘（NormalizeExpr）；匹配时（Match）放宽为兼容历史值，
// 允许 v4.0.0.0 这类不带比较符的旧写法，避免升级后老数据全部失效。
//
// x 的展开规则（这条最容易写错，实现按「操作符方向」决定取上界还是下界）：
//
//	op 为 <  ：取该系列下界，保证系列内任何版本都不满足
//	op 为 <= ：取该系列上界，保证系列内任何版本都满足
//	op 为 >  ：取该系列上界，保证系列内任何版本都不满足
//	op 为 >= ：取该系列下界，保证系列内任何版本都满足
//	op 为 = / 无：视为前缀区间 [下界, 上界]，即整个系列

// _WILDCARD_PLACEHOLDER x 展开为上界时使用的段值（版本段为纯数字，足够大即可）
const _WILDCARD_PLACEHOLDER = "999999"

// term 单个比较条件（已把 x 展开为具体边界）
type term struct {
	Raw string // 原始写法（错误文案用）
	Op  string // "" / "=" / "<" / "<=" / ">" / ">="
	Ver string // 规范化后的版本写法（保留 x 通配，回显用）
	Lo  string // 区间下界（规范化后的版本，如 v3.0.0）
	Hi  string // 区间上界（无通配时与 Lo 相同）
}

// operatorReplacer 把 ≥/≤ 与全角比较符归一为 ASCII 操作符（录入侧允许直接复制粘贴）
var operatorReplacer = strings.NewReplacer(
	"＞＝", ">=", "＜＝", "<=",
	"≥", ">=", "≤", "<=",
	"＞", ">", "＜", "<", "＝", "=",
)

// IsUnlimited 报告表达式是否表示「不限制」（空串 / * / any / all）
func IsUnlimited(expr string) bool {
	switch strings.ToLower(strings.TrimSpace(expr)) {
	case "", "*", "any", "all":
		return true
	}
	return false
}

// Validate 按录入规范校验表达式（严格模式）。
//
// 上传/保存声明时会先调用它：把写错的表达式拦在录入阶段，
// 而不是等到执行时才判定适配关系无法判定。
// 严格模式要求每个条件都以 =、≥、≤、>、< 之一开头，多区间用换行分隔。
func Validate(expr string) error {
	if IsUnlimited(expr) {
		return nil
	}
	_, err := parseGroups(expr, true)
	return err
}

// NormalizeExpr 把表达式规范化为落盘/回显写法：
// 比较符统一为 ≥/≤，单行多条件用空格分隔，多区间用换行分隔。
// 表达式非法时原样返回（仅去首尾空白），由调用方决定如何报错。
func NormalizeExpr(expr string) string {
	if IsUnlimited(expr) {
		return ""
	}
	groups, err := parseGroups(expr, false)
	if err != nil {
		return strings.TrimSpace(expr)
	}
	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		parts := make([]string, 0, len(group))
		for _, one := range group {
			parts = append(parts, one.display())
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

// Match 判断 actual 是否满足表达式。
//
// 表达式为空（或 * / any）时恒为 true：历史数据没有声明适配范围，
// 保持「未声明即不限制」的既有语义，避免升级后老数据全部失效。
// actual 为空（调用方没能拿到实际版本）时返回 ErrEmptyVersion，
// 不做静默放行——放行等于假装校验过，比不校验更危险。
func Match(expr, actual string) (bool, error) {
	if IsUnlimited(expr) {
		return true, nil
	}
	actual = NormalizeActual(actual)
	if actual == "" {
		return false, wrapError(ErrEmptyVersion, "actual version is empty, cannot match expression %q", strings.TrimSpace(expr))
	}

	groups, err := parseGroups(expr, false)
	if err != nil {
		return false, err
	}
	// 行间 OR：任一区间命中即通过；行内 AND：组内条件全部满足才命中
	for _, group := range groups {
		if matchGroup(group, actual) {
			return true, nil
		}
	}
	return false, nil
}

// Explain 返回表达式的规范化描述（供前端/接口回显，如 "v3.x -> [v3.0, v3.999999]"）
func Explain(expr string) string {
	if IsUnlimited(expr) {
		return "不限"
	}
	groups, err := parseGroups(expr, false)
	if err != nil {
		return "非法表达式: " + err.Error()
	}
	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		parts := make([]string, 0, len(group))
		for _, one := range group {
			op := displayOperator(one.Op)
			if one.Lo == one.Hi {
				parts = append(parts, fmt.Sprintf("%s %s", op, one.Lo))
				continue
			}
			parts = append(parts, fmt.Sprintf("%s [%s, %s]", op, one.Lo, one.Hi))
		}
		lines = append(lines, strings.Join(parts, " 且 "))
	}
	return strings.Join(lines, " 或 ")
}

// displayOperator 把内部 ASCII 操作符转成展示用的 ≥/≤
func displayOperator(op string) string {
	switch op {
	case ">=":
		return "≥"
	case "<=":
		return "≤"
	case "":
		return "="
	}
	return op
}

// match 单个条件判定
func (own term) match(actual string) bool {
	switch own.Op {
	case "<":
		return Compare(actual, own.Lo) < 0
	case "<=":
		return Compare(actual, own.Hi) <= 0
	case ">":
		return Compare(actual, own.Hi) > 0
	case ">=":
		return Compare(actual, own.Lo) >= 0
	default: // 精确 / 前缀通配
		return Compare(actual, own.Lo) >= 0 && Compare(actual, own.Hi) <= 0
	}
}

// parseGroups 把表达式拆成「行间 OR、行内 AND」的两级结构。
//
// strict 为 true 时按录入规范校验：每个条件必须以 =、≥、≤、>、< 之一开头；
// strict 为 false 时保持历史兼容，允许 v4.0.0.0 这类不带比较符的旧写法。
func parseGroups(expr string, strict bool) ([][]term, error) {
	lines := splitGroups(normalizeOperators(expr))
	if len(lines) == 0 {
		return nil, wrapError(ErrInvalidVersionExpr, "expression %q contains no condition", expr)
	}

	groups := make([][]term, 0, len(lines))
	for _, line := range lines {
		fields := splitLineTerms(line)
		if len(fields) == 0 {
			continue
		}
		group := make([]term, 0, len(fields))
		for _, field := range fields {
			if strict && !hasLeadingOperator(field) {
				return nil, wrapError(ErrInvalidVersionExpr, "condition %q must start with =, >=, <=, > or < (separate ranges with newlines)", field)
			}
			one, err := parseTerm(field)
			if err != nil {
				return nil, err
			}
			group = append(group, one)
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	if len(groups) == 0 {
		return nil, wrapError(ErrInvalidVersionExpr, "expression %q contains no condition", expr)
	}
	return groups, nil
}

// matchGroup 组内条件全部满足才算命中（AND）
func matchGroup(group []term, actual string) bool {
	for _, one := range group {
		if !one.match(actual) {
			return false
		}
	}
	return true
}

// normalizeOperators 把 ≥/≤ 与全角比较符转为 ASCII，供后续解析
func normalizeOperators(expr string) string {
	return operatorReplacer.Replace(expr)
}

// splitGroups 按换行拆分为多个区间（兼容 \r\n，忽略空行）
func splitGroups(expr string) []string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(expr, "\r\n", "\n"), "\r", "\n")
	raw := strings.Split(normalized, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// splitLineTerms 行内按空白 / 逗号 / 分号拆分，并吸收「操作符与版本号之间」的空格
// （允许 ">= v6.4.0" 这种写法）
func splitLineTerms(line string) []string {
	// 先把比较符与版本号之间的空白去掉，避免被当成两个条件
	compact := strings.NewReplacer(
		">= ", ">=", "<= ", "<=", "== ", "==", "> ", ">", "< ", "<", "= ", "=",
	).Replace(strings.TrimSpace(line))

	raw := strings.FieldsFunc(compact, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	})
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// hasLeadingOperator 判断条件是否以比较符开头（严格校验用）
func hasLeadingOperator(field string) bool {
	for _, candidate := range []string{">=", "<=", "==", ">", "<", "="} {
		if strings.HasPrefix(field, candidate) {
			return true
		}
	}
	return false
}

// parseTerm 解析单个条件，并把 x 展开成具体区间边界
func parseTerm(field string) (term, error) {
	op := ""
	rest := strings.TrimSpace(field)
	// 顺序重要：先匹配两字符操作符，避免 ">=" 被当成 ">"
	for _, candidate := range []string{">=", "<=", "==", ">", "<", "="} {
		if strings.HasPrefix(rest, candidate) {
			op = candidate
			rest = strings.TrimSpace(rest[len(candidate):])
			break
		}
	}
	if op == "==" {
		op = "="
	}

	lo, hi, err := expandVersion(rest)
	if err != nil {
		return term{}, wrapError(ErrInvalidVersionExpr, "condition %q: %v", field, err)
	}

	return term{Raw: field, Op: op, Ver: normalizeVerText(rest), Lo: lo, Hi: hi}, nil
}

// display 返回规范化写法：比较符统一为 ≥/≤，无操作符（历史写法）补 = 前缀
func (own term) display() string {
	if own.Ver == "" {
		return own.Raw
	}
	switch own.Op {
	case ">=":
		return "≥" + own.Ver
	case "<=":
		return "≤" + own.Ver
	case ">", "<", "=":
		return own.Op + own.Ver
	default:
		return "=" + own.Ver
	}
}

// normalizeVerText 规范化版本写法用于回显：小写、去首尾空白、补 v 前缀
func normalizeVerText(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text == "" {
		return ""
	}
	if !strings.HasPrefix(text, "v") {
		text = "v" + text
	}
	return text
}

// expandVersion 把版本写法规范化为 [下界, 上界]：
// 无通配时上下界相同；含 x 时按 x 所在位置截断并补边界值。
func expandVersion(raw string) (string, string, error) {
	ver := Normalize(raw)
	if ver == "" {
		return "", "", wrapError(ErrInvalidVersion, "version is empty")
	}
	if strings.Contains(ver, "..") {
		return "", "", wrapError(ErrInvalidVersion, "empty segment in %q", raw)
	}

	segments := strings.Split(ver, ".")
	wildcard := -1
	for index, segment := range segments {
		switch {
		case segment == "":
			return "", "", wrapError(ErrInvalidVersion, "empty segment in %q", raw)
		case segment == "x":
			if wildcard >= 0 {
				return "", "", wrapError(ErrInvalidVersion, "wildcard x may appear only once: %q", raw)
			}
			wildcard = index
		case !IsDigits(segment):
			return "", "", wrapError(ErrInvalidVersion, "segment %q is neither numeric nor wildcard x: %q", segment, raw)
		}
	}
	if wildcard >= 0 && wildcard != len(segments)-1 {
		return "", "", wrapError(ErrInvalidVersion, "wildcard x must be the last segment: %q", raw)
	}
	// 裸写 x / 写成 "vx" 这类表达式没有可比较的版本主体（会被展开成不限），
	// 实质等同静默放行，宁可直接报错要求写成 v3.x 这类明确形式。
	if wildcard == 0 {
		return "", "", wrapError(ErrInvalidVersion, "wildcard x requires at least one numeric segment (e.g. v3.x): %q", raw)
	}
	if wildcard < 0 {
		// 非通配：上下界相同（精确值）
		normalized := "v" + strings.Join(segments, ".")
		return normalized, normalized, nil
	}

	prefix := segments[:wildcard]
	lower := append(append([]string{}, prefix...), "0")
	upper := append(append([]string{}, prefix...), _WILDCARD_PLACEHOLDER)
	return "v" + strings.Join(lower, "."), "v" + strings.Join(upper, "."), nil
}
