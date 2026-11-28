package projection

import (
	"fmt"
)

// Point 投影坐标点
//   - easting：东向坐标（索引 0）
//   - northing：北向坐标（索引 1）
type Point [2]float64

// CalculateCross 计算两个坐标的叉积
func (own Point) CalculateCross(p Point) float64 {
	return own[0]*p[1] - own[1]*p[0]
}

// GetValues 获取东向与北向坐标值
func (own Point) GetValues() (easting, northing float64) {
	return own[0], own[1]
}

// IsEqualTo 判断两个坐标是否相同
func (own Point) IsEqualTo(p Point) bool {
	return own[0] == p[0] && own[1] == p[1]
}

// String 获取坐标的字符串表示，例如 "(113.123456, 22.654321)"
func (own Point) String() string {
	return fmt.Sprintf("(%.6f, %.6f)", own[0], own[1])
}
