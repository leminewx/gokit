package command

import (
	"strconv"
	"strings"
)

// ParseDf 解析 df 输出（推荐 `df -P -k`，单位为 1K 块）
//   - 兼容 GNU coreutils、BusyBox 与 ToyBox 的表头（第 5 列为 `Use%` 或 `Capacity`）
//   - BusyBox 在文件系统名过长时会把一条记录折成两行（先名字、后数值），这里定位
//     `Use%` 列后向左取 3 列数值、向右取挂载点，折行会被自动拼接
//   - 挂载点允许包含空格（如 `/mnt/My Disk`）
//   - 只支持 `-k` 输出的固定列，`df -h` 的人类可读单位无法还原精确字节数
func ParseDf(output string) ([]FilesystemStats, error) {
	var (
		entries []FilesystemStats
		pending []string // 折行时暂存文件系统名
		inTable bool
	)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !inTable {
			if strings.Contains(line, "Filesystem") && (strings.Contains(line, "Mounted") || strings.Contains(line, "Mount")) {
				inTable = true
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			pending = append(pending, line)
			continue
		}

		// 定位 Use% 列：其前 3 列必须是数值，后面依次是挂载点（允许包含空格）
		pctIdx := -1
		for idx := 3; idx < len(fields)-1; idx++ {
			token := fields[idx]
			if !strings.HasSuffix(token, "%") && token != "-" {
				continue
			}
			if _, ok := parseUintField(fields[idx-1]); !ok {
				continue
			}
			if _, ok := parseUintField(fields[idx-2]); !ok {
				continue
			}
			if _, ok := parseUintField(fields[idx-3]); !ok {
				continue
			}
			pctIdx = idx
			break
		}
		if pctIdx < 0 {
			pending = append(pending, line)
			continue
		}

		totalKB, _ := parseUintField(fields[pctIdx-3])
		usedKB, _ := parseUintField(fields[pctIdx-2])
		freeKB, _ := parseUintField(fields[pctIdx-1])
		usedPct, _ := parsePercentField(fields[pctIdx])

		name := strings.Join(fields[:pctIdx-3], " ")
		if name == "" {
			// 文件系统名在上一行（BusyBox 折行）
			name = strings.Join(pending, " ")
		} else if len(pending) > 0 {
			name = strings.Join(append(pending, name), " ")
		}
		pending = nil

		mountedOn := strings.Join(fields[pctIdx+1:], " ")
		if name == "" || mountedOn == "" {
			continue
		}

		entries = append(entries, FilesystemStats{
			Filesystem:  name,
			MountedOn:   mountedOn,
			TotalBytes:  totalKB << 10,
			UsedBytes:   usedKB << 10,
			FreeBytes:   freeKB << 10,
			UsedPercent: usedPct,
		})
	}

	if !inTable {
		return nil, wrapError(ErrParse, "df: table header not found")
	}

	return entries, nil
}

// parseUintField 解析十进制无符号整数
func parseUintField(s string) (uint64, bool) {
	// df 在部分平台会把超大容量显示为 0 或以 '-' 表示不可用
	if s == "-" {
		return 0, true
	}

	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil
}

// parsePercentField 解析 "93%" 形式的百分比
func parsePercentField(s string) (float64, bool) {
	if s == "-" {
		return 0, true
	}

	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	return v, err == nil
}
