package ini

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type (
	// INI 表示整个配置
	INI struct {
		Sections   []*Section          // 节顺序（全局节在第一个）
		SectionMap map[string]*Section // 节名到节的映射（快速查找）
	}

	// Section 表示一个节
	Section struct {
		Name  string            // 节名称（空字符串表示全局节）
		Keys  map[string]string // 键值对
		Order []string          // 键的顺序（保持原顺序）
	}
)

// 编译期断言：INI 支持 JSON 序列化与文本写出
var (
	_ json.Marshaler   = (*INI)(nil)
	_ json.Unmarshaler = (*INI)(nil)
	_ io.WriterTo      = (*INI)(nil)
)

// LoadFromJSON 从 JSON 文件加载配置
//   - 输入格式与 MarshalJSON 一致：{"global":{...},"section":{...}}
func LoadFromJSON(filename string) (*INI, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, wrapError(ErrReadFailed, "%q: %w", filename, err)
	}

	cfg := &INI{}
	if err := cfg.UnmarshalJSON(data); err != nil {
		return nil, wrapError(ErrParseFailed, "%q: %w", filename, err)
	}

	return cfg, nil
}

// Marshal 从结构体 v 生成配置
//   - 标签规则与 Unmarshal 一致
//   - 所有可导出字段都会被导出，零值字段写入为零值字符串
func Marshal(v any) (*INI, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, wrapError(ErrInvalidStruct, "nil pointer")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, wrapError(ErrInvalidStruct, "%s", rv.Kind())
	}

	cfg := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}

	if err := cfg.marshalStruct(rv, ""); err != nil {
		return nil, err
	}

	// 确保全局节存在（即使空）
	if _, ok := cfg.SectionMap[""]; !ok {
		sec := &Section{Name: "", Keys: make(map[string]string), Order: []string{}}
		cfg.Sections = append([]*Section{sec}, cfg.Sections...)
		cfg.SectionMap[""] = sec
	}

	return cfg, nil
}

// marshalStruct 递归导出结构体字段，parentSection 为父节名（空表示全局）
func (own *INI) marshalStruct(rv reflect.Value, parentSection string) error {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		if !field.CanInterface() {
			continue
		}

		structFieldTag := rt.Field(i).Tag.Get("ini")
		if structFieldTag == "-" {
			continue
		}

		var section, key string
		if structFieldTag != "" {
			parts := strings.SplitN(structFieldTag, ".", 2)
			if len(parts) == 2 {
				section, key = parts[0], parts[1]
			} else if isSectionStruct(field) {
				// 结构体字段：这个词是节名
				section = parts[0]
			} else {
				// 基本类型：这个词是键名
				key = parts[0]
			}
		}

		if section == "" && parentSection != "" {
			section = parentSection
		}

		// 处理嵌套结构体
		if isSectionStruct(field) {
			sectionName := section
			if sectionName == "" {
				sectionName = rt.Field(i).Name
			}

			if err := own.marshalStruct(field, sectionName); err != nil {
				return err
			}

			continue
		}

		if key == "" {
			key = rt.Field(i).Name
		}

		// 基础类型（包括切片）：将值转为字符串
		val, err := stringify(field)
		if err != nil {
			return wrapError(ErrConvertFailed, "field %s: %w", rt.Field(i).Name, err)
		}

		own.SetValue(section, key, val)
	}

	return nil
}

// stringify 将字段值转为 ini 文本，切片以 JSON 数组表示
func stringify(field reflect.Value) (string, error) {
	if field.Kind() != reflect.Slice {
		return fmt.Sprint(field.Interface()), nil
	}

	if field.IsNil() {
		return "[]", nil
	}

	data, err := json.Marshal(field.Interface())
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetValue 获取指定节和键的值
//   - 若 section 为空字符串，则获取全局配置中的值
func (own *INI) GetValue(section, key string) (string, bool) {
	if sec, ok := own.SectionMap[section]; ok {
		val, ok := sec.Keys[key]
		return val, ok
	}

	return "", false
}

// SetValue 设置指定节和键的值，若节或键不存在则创建
//   - 若 section 为空字符串，则设置全局配置中的值
func (own *INI) SetValue(section, key, value string) {
	sec, ok := own.SectionMap[section]
	if !ok {
		sec = &Section{
			Name:  section,
			Keys:  make(map[string]string),
			Order: []string{},
		}
		if section == "" {
			// 全局节放在最前面
			own.Sections = append([]*Section{sec}, own.Sections...)
		} else {
			own.Sections = append(own.Sections, sec)
		}

		own.SectionMap[section] = sec
	}

	if _, exists := sec.Keys[key]; !exists {
		sec.Order = append(sec.Order, key)
	}

	sec.Keys[key] = value
}

// RemoveSections 删除指定节
func (own *INI) RemoveSections(sections ...string) {
	removed := make(map[string]bool, len(sections))
	for _, section := range sections {
		if _, ok := own.SectionMap[section]; ok {
			removed[section] = true
			delete(own.SectionMap, section)
		}
	}

	if len(removed) == 0 {
		return
	}

	kept := make([]*Section, 0, len(own.Sections))
	for _, sec := range own.Sections {
		if !removed[sec.Name] {
			kept = append(kept, sec)
		}
	}

	own.Sections = kept
}

// RemoveKeys 删除指定节中的键值对
func (own *INI) RemoveKeys(section string, keys ...string) {
	sec, ok := own.SectionMap[section]
	if !ok {
		return
	}

	removed := make(map[string]bool, len(keys))
	for _, key := range keys {
		removed[key] = true
		delete(sec.Keys, key)
	}

	order := make([]string, 0, len(sec.Order))
	for _, key := range sec.Order {
		if !removed[key] {
			order = append(order, key)
		}
	}

	sec.Order = order
}

// ----------------------------------------------------------------------
// 与结构体互转（基于反射）
// ----------------------------------------------------------------------

// Unmarshal 将配置解析到结构体 v（必须为非空指针）
//   - 支持标签 `ini:"key"` 或 `ini:"section.key"`（节名和键名用点分隔）
//   - 若只有键名，则默认使用全局节（空字符串）
//   - 如果字段是结构体，则自动将其视为一个节（节名为字段名或标签中的节名）
//   - 支持基础类型、切片、指针、time.Duration 与 encoding.TextUnmarshaler
func (own *INI) Unmarshal(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return wrapError(ErrInvalidTarget, "v must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return wrapError(ErrInvalidTarget, "v must point to a struct")
	}

	return own.unmarshalStruct(rv, "")
}

// unmarshalStruct 递归处理结构体，parentSection 为父节名（空表示全局）
func (own *INI) unmarshalStruct(rv reflect.Value, parentSection string) error {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		if !field.CanSet() {
			continue
		}

		structFieldTag := rt.Field(i).Tag.Get("ini")
		if structFieldTag == "-" {
			continue
		}

		var section, key string
		if structFieldTag != "" {
			parts := strings.SplitN(structFieldTag, ".", 2)
			if len(parts) == 2 {
				section, key = parts[0], parts[1]
			} else if isSectionStruct(field) {
				section = parts[0]
			} else {
				key = parts[0]
			}
		}

		if section == "" && parentSection != "" {
			section = parentSection
		}

		if isSectionStruct(field) {
			sectionName := section
			if sectionName == "" {
				sectionName = rt.Field(i).Name
			}

			if err := own.unmarshalStruct(field, sectionName); err != nil {
				return err
			}

			continue
		}

		if key == "" {
			key = rt.Field(i).Name
		}

		val, ok := own.GetValue(section, key)
		if !ok {
			continue
		}

		if err := setField(field, val); err != nil {
			return wrapError(ErrConvertFailed, "field %s: %w", rt.Field(i).Name, err)
		}
	}

	return nil
}
