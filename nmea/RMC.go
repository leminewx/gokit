package nmea

import (
	"fmt"
	"strconv"
	"strings"

	"gitee.com/leminewx/gokit"
	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// RMC 表示 NMEA 0183 协议中的推荐最小定位信息
type RMC struct {
	System         System  `json:"system"`          // 系统（GP/GA/GL/GN）
	UTCTimeStr     string  `json:"utc_time_str"`    // UTC 时间（HHMMSS.sss）
	Status         string  `json:"status"`          // 定位状态（A/V）
	LatitudeRaw    float64 `json:"latitude_raw"`    // 纬度（度分格式）
	LatitudeDir    string  `json:"latitude_dir"`    // 纬度方向（N/S）
	LongitudeRaw   float64 `json:"longitude_raw"`   // 经度（度分格式）
	LongitudeDir   string  `json:"longitude_dir"`   // 经度方向（E/W）
	GroundVelocity float64 `json:"ground_velocity"` // 地面速率 (节)
	GroundHeading  float64 `json:"ground_heading"`  // 地面航向 (真北参考方向，单位：度)
	UTCDateStr     string  `json:"utc_date_str"`    // UTC 日期（DDMMYY）
	MagVariation   float64 `json:"mag_variation"`   // 磁偏角 (可选)
	MagDir         string  `json:"mag_dir"`         // 磁偏角方向 (E/W)
	Mode           Mode    `json:"mode"`            // 模式指示: A/D/E/F/M/N/P/R/S
	Checksum       string  `json:"checksum"`        // 校验和
}

// ParseRMC 解析 NMEA 0183 协议的 RMC 数据
func ParseRMC(sentence string) (rmc *RMC, err error) {
	if !strings.HasPrefix(sentence, "$") || !strings.Contains(sentence, string(TYPE_RMC)) {
		return nil, ErrNotRMCSentence
	}

	// 去除末尾的换行符并按逗号分割
	// $RMC,071654.65,A,4006.97231763,N,11632.90263794,E,014.252,264.8,291125,,,D*50
	parts := strings.Split(strings.TrimRight(sentence, "\r\n"), ",")
	if len(parts) != 13 {
		return nil, ErrRMCCountMismatch
	}

	lasts := strings.Split(parts[12], "*")
	if len(lasts) != 2 {
		return nil, ErrMissingChecksum
	}
	rmc = &RMC{
		System:       System(strings.Split(parts[0], string(TYPE_RMC))[0][1:]),
		UTCTimeStr:   parts[1],
		Status:       parts[2],
		LatitudeDir:  parts[4],
		LongitudeDir: parts[6],
		UTCDateStr:   parts[9],
		MagDir:       parts[11],
		Mode:         Mode(lasts[0]),
		Checksum:     lasts[1],
	}

	if rmc.LatitudeRaw, err = strconv.ParseFloat(parts[3], 64); err != nil {
		return nil, err
	}
	if rmc.LongitudeRaw, err = strconv.ParseFloat(parts[5], 64); err != nil {
		return nil, err
	}
	if rmc.GroundVelocity, err = strconv.ParseFloat(parts[7], 64); err != nil {
		return nil, err
	}
	if rmc.GroundHeading, err = strconv.ParseFloat(parts[8], 64); err != nil {
		return nil, err
	}
	if parts[10] != "" {
		if rmc.MagVariation, err = strconv.ParseFloat(parts[10], 64); err != nil {
			return nil, err
		}
	}

	return
}

// GetLLAPoint 获取经纬度坐标消息
//   - dat: 地理基准面，默认使用 WGS84
func (own *RMC) GetLLAPoint(dat ...datum.Datum) geography.LLAPoint {
	return geography.NewLLAPoint(gokit.GetValueWithDefault(datum.WGS84, dat...), ConvertRawToDecimal(own.LatitudeRaw), ConvertRawToDecimal(own.LongitudeRaw))
}

func (own *RMC) String() string {
	// $GPRMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,,,D*50
	// 小数字段固定 3 位，使 String 的输出能被 ParseRMC 原样还原
	if own.MagDir != "" || own.MagVariation != 0 {
		return fmt.Sprintf("$%sRMC,%s,%s,%.8f,%s,%.8f,%s,%.3f,%.3f,%s,%.3f,%s,%s*%s", own.System, own.UTCTimeStr,
			own.Status, own.LatitudeRaw, own.LatitudeDir, own.LongitudeRaw, own.LongitudeDir,
			own.GroundVelocity, own.GroundHeading, own.UTCDateStr, own.MagVariation,
			own.MagDir, own.Mode, own.Checksum)
	}

	return fmt.Sprintf("$%sRMC,%s,%s,%.8f,%s,%.8f,%s,%.3f,%.3f,%s,,,%s*%s", own.System, own.UTCTimeStr, own.Status,
		own.LatitudeRaw, own.LatitudeDir, own.LongitudeRaw, own.LongitudeDir,
		own.GroundVelocity, own.GroundHeading, own.UTCDateStr, own.Mode, own.Checksum)
}
