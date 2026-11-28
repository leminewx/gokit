package ini

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// LoadFromINI 从 ini 文件读取配置
func LoadFromINI(filename string) (*INI, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, wrapError(ErrReadFailed, "%q: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	cfg, err := LoadFromReader(file)
	if err != nil {
		return nil, wrapError(ErrReadFailed, "%q: %w", filename, err)
	}

	return cfg, nil
}

// LoadFromString 从字符串解析配置，便于测试与内嵌配置
func LoadFromString(content string) (*INI, error) {
	return LoadFromReader(strings.NewReader(content))
}

// LoadFromReader 从 reader 解析配置
//   - 空行与 # / ; 开头的整行注释会被忽略
//   - 值后仅当 # / ; 前有空白时按行内注释剥离
//   - 键值中的成对引号会被去掉
func LoadFromReader(reader io.Reader) (*INI, error) {
	cfg := &INI{
		Sections:   []*Section{},
		SectionMap: make(map[string]*Section),
	}

	scanner := bufio.NewScanner(reader)
	var currentSection *Section // 当前正在处理的节
	for scanner.Scan() {
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}

		// 处理节
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			// 空节名视为非法，只跳过该行
			if name == "" {
				continue
			}

			if section, exists := cfg.SectionMap[name]; exists {
				// 节已存在，复用
				currentSection = section
			} else {
				currentSection = &Section{
					Name:  name,
					Keys:  make(map[string]string),
					Order: []string{},
				}

				cfg.Sections = append(cfg.Sections, currentSection)
				cfg.SectionMap[name] = currentSection
			}

			continue
		}

		// 处理键值对
		equalIdx := strings.Index(line, "=")
		if equalIdx == -1 {
			continue // 无效行，跳过
		}

		key := strings.TrimSpace(line[:equalIdx])
		if key == "" {
			continue
		}
		value := unquote(line[equalIdx+1:])

		// 如果还没有进入任何节，则自动使用全局节（空字符串）
		if currentSection == nil {
			globalName := ""
			if section, exists := cfg.SectionMap[globalName]; exists {
				currentSection = section
			} else {
				currentSection = &Section{
					Name:  globalName,
					Keys:  make(map[string]string),
					Order: []string{},
				}

				// 全局节应放在最前面
				cfg.Sections = append([]*Section{currentSection}, cfg.Sections...)
				cfg.SectionMap[globalName] = currentSection
			}
		}

		// 设置键值对，保持键的顺序
		if _, exists := currentSection.Keys[key]; !exists {
			currentSection.Order = append(currentSection.Order, key)
		}
		currentSection.Keys[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, wrapError(ErrReadFailed, "%w", err)
	}

	return cfg, nil
}

// stripInlineComment 去掉行内注释
//   - 仅当 # 或 ; 前面是空白且不在引号内时视为注释起点
func stripInlineComment(line string) string {
	var quote rune
	for idx, char := range line {
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '"' || char == '\'':
			quote = char
		case char == '#' || char == ';':
			if idx == 0 {
				return ""
			}
			if prev := line[idx-1]; prev == ' ' || prev == '\t' {
				return strings.TrimSpace(line[:idx])
			}
		}
	}

	return line
}
