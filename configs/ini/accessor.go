package ini

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetString 返回字符串值，键不存在时返回空字符串
func (own *INI) GetString(section, key string) string {
	value, _ := own.GetValue(section, key)
	return value
}

// GetInt 返回 int 值
func (own *INI) GetInt(section, key string) (int, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return 0, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return n, nil
}

// GetInt64 返回 int64 值
func (own *INI) GetInt64(section, key string) (int64, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return 0, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return n, nil
}

// GetUint64 返回 uint64 值
func (own *INI) GetUint64(section, key string) (uint64, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return 0, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	n, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return n, nil
}

// GetFloat64 返回 float64 值
func (own *INI) GetFloat64(section, key string) (float64, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return 0, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return n, nil
}

// GetBool 返回 bool 值
func (own *INI) GetBool(section, key string) (bool, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return false, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	b, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return b, nil
}

// GetDuration 返回时长值
//   - 支持 Go 时长格式（如 30s、1m30s），纯数字按纳秒解析
func (own *INI) GetDuration(section, key string) (time.Duration, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return 0, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	duration, err := parseDuration(value)
	if err != nil {
		return 0, wrapError(ErrConvertFailed, "%q.%q: %w", section, key, err)
	}

	return duration, nil
}

// GetStrings 返回字符串切片
//   - 优先按 JSON 数组解析，其次按逗号分隔；空值返回空切片
func (own *INI) GetStrings(section, key string) ([]string, error) {
	value, ok := own.GetValue(section, key)
	if !ok {
		return nil, wrapError(ErrKeyNotFound, "%q.%q", section, key)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, nil
	}

	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		var values []string
		if err := json.Unmarshal([]byte(value), &values); err == nil {
			return values, nil
		}
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, unquote(part))
	}

	return values, nil
}

// HasSection 判断节是否存在
func (own *INI) HasSection(name string) bool {
	_, ok := own.SectionMap[name]
	return ok
}

// HasKey 判断节中是否存在指定键
func (own *INI) HasKey(section, key string) bool {
	sec, ok := own.SectionMap[section]
	if !ok {
		return false
	}

	_, ok = sec.Keys[key]
	return ok
}

// GetSectionNames 返回所有节名，全局节以空字符串表示，顺序与写入顺序一致
func (own *INI) GetSectionNames() []string {
	names := make([]string, 0, len(own.Sections))
	for _, section := range own.Sections {
		names = append(names, section.Name)
	}

	return names
}

// GetKeys 返回节内所有键，顺序与写入顺序一致
func (own *INI) GetKeys(section string) []string {
	sec, ok := own.SectionMap[section]
	if !ok {
		return nil
	}

	return append([]string(nil), sec.Order...)
}

// GetSortedSectionNames 返回排序后的节名快照，便于稳定输出日志与比较
func (own *INI) GetSortedSectionNames() []string {
	names := own.GetSectionNames()
	sort.Strings(names)

	return names
}
