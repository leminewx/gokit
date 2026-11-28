package geography

import (
	"fmt"
	"math"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// # ENUPoint 东北天坐标
//
// 所有分量均为相对于参考原点的偏移量，而非绝对经纬度或海拔
//   - 0：东向，east（米）
//   - 1：北向，north（米）
//   - 2：天向，up（米）
type ENUPoint struct {
	values [3]float64  // 坐标值
	datum  datum.Datum // 基准面
}

// NewENUPoint 创建一个东北天坐标点
//   - dat：基准面
//   - values：坐标值 (0-east, 1-north, 2-up)
func NewENUPoint(dat datum.Datum, values ...float64) ENUPoint {
	p := ENUPoint{datum: dat}
	switch len(values) {
	case 1:
		p.values[0] = values[0]
	case 2:
		p.values[0] = values[0]
		p.values[1] = values[1]
	case 3:
		p.values[0] = values[0]
		p.values[1] = values[1]
		p.values[2] = values[2]
	}

	return p
}

// NewENUPoints 创建多个东北天坐标点
//   - dat：基准面
//   - points：坐标点值的列表 (坐标值：0-east, 1-north, 2-up)
func NewENUPoints(dat datum.Datum, points ...[3]float64) []ENUPoint {
	enuPoints := make([]ENUPoint, len(points))
	for i, point := range points {
		enuPoints[i] = ENUPoint{point, dat}
	}

	return enuPoints
}

// GetDatum 获取基准面
func (own ENUPoint) GetDatum() datum.Datum {
	return own.datum
}

// GetValues 获取坐标
func (own ENUPoint) GetValues() (east, north, up float64) {
	return own.values[0], own.values[1], own.values[2]
}

// IsEqualTo 判断两个坐标是否相同
func (own ENUPoint) IsEqualTo(p ENUPoint) bool {
	east, north, up := p.GetValues()
	return math.Abs(own.values[0]-east) < BASIC_EPSILON && math.Abs(own.values[1]-north) < BASIC_EPSILON && math.Abs(own.values[2]-up) < BASIC_EPSILON && own.datum == p.GetDatum()
}

// String 获取字符串格式的坐标信息
func (own ENUPoint) String() string {
	return fmt.Sprintf(`{"east":%f,"north":%f,"up":%f}`, own.values[0], own.values[1], own.values[2])
}
