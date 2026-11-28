package ini

import (
	"encoding/json"
	"sort"
)

// MarshalJSON 实现 json.Marshaler 接口，将 INI 转为 JSON
//   - 输出格式：{"global":{"key":"value",...},"sectionName":{"key":"value",...}}
func (own *INI) MarshalJSON() ([]byte, error) {
	values := make(map[string]map[string]string, len(own.Sections))
	for _, section := range own.Sections {
		name := section.Name
		if name == "" {
			name = "global"
		}

		values[name] = section.Keys
	}

	return json.Marshal(values)
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，从 JSON 填充 INI
//   - 输入格式与 MarshalJSON 一致
//   - JSON 对象无序，解析后节名与节内键名均按字典序排列，保证结果确定性
func (own *INI) UnmarshalJSON(data []byte) error {
	var raw map[string]map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	own.Sections = nil
	own.SectionMap = make(map[string]*Section)

	// 全局节固定排在第一位
	if globalKeys, ok := raw["global"]; ok {
		section := newJSONSection("", globalKeys)
		own.Sections = append(own.Sections, section)
		own.SectionMap[""] = section
	}

	names := make([]string, 0, len(raw))
	for name := range raw {
		if name == "global" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		section := newJSONSection(name, raw[name])
		own.Sections = append(own.Sections, section)
		own.SectionMap[name] = section
	}

	return nil
}

// newJSONSection 由 JSON 键值构造节，键按字典序排列
func newJSONSection(name string, keys map[string]string) *Section {
	order := make([]string, 0, len(keys))
	for key := range keys {
		order = append(order, key)
	}
	sort.Strings(order)

	return &Section{Name: name, Keys: keys, Order: order}
}
