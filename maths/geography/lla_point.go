package geography

import (
	"fmt"
	"math"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// # LLAPoint 经纬高坐标
//   - 0：经度，longitude（°）
//   - 1：纬度，latitude（°）
//   - 2：海拔，altitude（米）
type LLAPoint struct {
	values [3]float64  // 坐标值
	datum  datum.Datum // 基准面
}

// NewLLAPoint 创建一个经纬高坐标点
//   - dat：基准面
//   - values：坐标值 (0-longitude, 1-latitude, 2-altitude)
func NewLLAPoint(dat datum.Datum, values ...float64) LLAPoint {
	p := LLAPoint{datum: dat}
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

// NewLLAPoints 创建多个地理坐标点
//   - dat：基准面
//   - points (坐标值：0-longitude, 1-latitude, 2-altitude)
func NewLLAPoints(dat datum.Datum, points ...[3]float64) []LLAPoint {
	llaPoints := make([]LLAPoint, len(points))
	for i, point := range points {
		llaPoints[i] = LLAPoint{point, dat}
	}

	return llaPoints
}

// GetDatum 获取基准面
func (own LLAPoint) GetDatum() datum.Datum {
	return own.datum
}

// GetValues 获取坐标
func (own LLAPoint) GetValues() (longitude, latitude, altitude float64) {
	return own.values[0], own.values[1], own.values[2]
}

// GetRadianValues 获取弧度坐标
func (own LLAPoint) GetRadianValues() (longitude, latitude, altitude float64) {
	return ConvertToRadian(own.values[0]), ConvertToRadian(own.values[1]), own.values[2]
}

// IsEqualTo 判断两个坐标是否相同
func (own LLAPoint) IsEqualTo(p LLAPoint) bool {
	lon, lat, alt := p.GetValues()
	return math.Abs(own.values[0]-lon) < BASIC_EPSILON && math.Abs(own.values[1]-lat) < BASIC_EPSILON && math.Abs(own.values[2]-alt) < BASIC_EPSILON && own.datum == p.GetDatum()
}

// String 获取字符串格式的坐标信息
func (own LLAPoint) String() string {
	return fmt.Sprintf(`{"lon":%f,"lat":%f,"alt":%f}`, own.values[0], own.values[1], own.values[2])
}

// ConvertToENUPoint 转换为东北天坐标
func (own LLAPoint) ConvertToENUPoint(refLLA LLAPoint) ENUPoint {
	ecef := own.ConvertToECEFPoint()
	return ecef.ConvertToENUPoint(refLLA)
}

// ConvertToECEFPoint 转换为地心地固坐标
func (own LLAPoint) ConvertToECEFPoint() ECEFPoint {
	// 注意：GetRadianValues 的返回顺序为 (longitude, latitude, altitude)
	lonRad, latRad, _ := own.GetRadianValues()
	sinLat := math.Sin(latRad)
	cosLat := math.Cos(latRad)
	sinLon := math.Sin(lonRad)
	cosLon := math.Cos(lonRad)

	// 卯酉圈曲率半径
	_, lat, alt := own.GetValues()
	N := own.datum.GetEllipsoid().CalculatePrimeVerticalRadius(lat)

	x := (N + alt) * cosLat * cosLon
	y := (N + alt) * cosLat * sinLon
	z := (N*(1-own.datum.GetEllipsoid().GetFirstEccentricitySquared()) + alt) * sinLat

	return NewECEFPoint(own.datum, x, y, z)
}
