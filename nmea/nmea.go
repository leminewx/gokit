// Package nmea 解析与格式化 NMEA 0183 协议中的定位语句（GGA / RMC），
// 并提供度分坐标到十进制度的转换。
package nmea

import "math"

// ConvertRawToDecimal 将 NMEA 度分格式的坐标转换为十进制度数
//   - raw: DDMM.MMMMMM（纬度，如：4006.97231763）或 DDDMM.MMMMMM（经度，如：11632.90263794）
func ConvertRawToDecimal(raw float64) float64 {
	degrees := math.Floor(raw / 100) // 提取整数部分（度）
	minutes := raw - degrees*100     // 提取分（包括小数部分）
	return degrees + minutes/60      // 转换为度
}
