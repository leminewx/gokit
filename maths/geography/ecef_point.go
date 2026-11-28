package geography

import (
	"fmt"
	"math"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// # ECEFPoint 地心地固坐标
//
// 坐标系原点位于地球质心，Z轴指向协议地极（CTP），X轴指向本初子午线与赤道交点，Y轴与X、Z轴构成右手系
//   - 0：地心直角 X 坐标（米）
//   - 1：地心直角 Y 坐标（米）
//   - 2：地心直角 Z 坐标（米）
type ECEFPoint struct {
	values [3]float64  // 坐标值
	datum  datum.Datum // 基准面
}

// NewECEFPoint 创建一个地心地固坐标点
//   - dat：基准面
//   - values：坐标值 (0-X, 1-Y, 2-Z)
func NewECEFPoint(dat datum.Datum, values ...float64) ECEFPoint {
	point := ECEFPoint{datum: dat}
	switch len(values) {
	case 1:
		point.values[0] = values[0]
	case 2:
		point.values[0] = values[0]
		point.values[1] = values[1]
	case 3:
		point.values[0] = values[0]
		point.values[1] = values[1]
		point.values[2] = values[2]
	}

	return point
}

// NewECEFPoints 创建多个地心地固坐标点
//   - dat：基准面
//   - points：坐标点值的列表 (坐标值：0-X, 1-Y, 2-Z)
func NewECEFPoints(dat datum.Datum, points ...[3]float64) []ECEFPoint {
	ecefPoints := make([]ECEFPoint, len(points))
	for i, point := range points {
		ecefPoints[i] = ECEFPoint{point, dat}
	}

	return ecefPoints
}

// GetDatum 获取基准面
func (own ECEFPoint) GetDatum() datum.Datum {
	return own.datum
}

// GetValues 获取坐标
func (own ECEFPoint) GetValues() (x, y, z float64) {
	return own.values[0], own.values[1], own.values[2]
}

// IsEqualTo 判断两个坐标是否相同
func (own ECEFPoint) IsEqualTo(p ECEFPoint) bool {
	x, y, z := p.GetValues()
	return math.Abs(own.values[0]-x) < BASIC_EPSILON && math.Abs(own.values[1]-y) < BASIC_EPSILON && math.Abs(own.values[2]-z) < BASIC_EPSILON && own.datum == p.GetDatum()
}

// String 获取字符串格式的坐标信息
func (own ECEFPoint) String() string {
	return fmt.Sprintf(`{"x":%f,"y":%f,"z":%f}`, own.values[0], own.values[1], own.values[2])
}

// ConvertToENUPoint 转换为东北天坐标
func (own ECEFPoint) ConvertToENUPoint(refLLA LLAPoint) ENUPoint {
	// 参考点的经纬度转弧度（GetRadianValues 返回顺序为 longitude、latitude、altitude）
	refLonRad, refLatRad, _ := refLLA.GetRadianValues()
	// 参考点的地心地固坐标
	refECEF := refLLA.ConvertToECEFPoint()
	refECEFX, refECEFY, refECEFZ := refECEF.GetValues()

	// 差值向量
	dx := own.values[0] - refECEFX
	dy := own.values[1] - refECEFY
	dz := own.values[2] - refECEFZ

	// 旋转矩阵 (ECEF -> ENU)
	e := -math.Sin(refLonRad)*dx + math.Cos(refLonRad)*dy
	n := -math.Sin(refLatRad)*math.Cos(refLonRad)*dx - math.Sin(refLatRad)*math.Sin(refLonRad)*dy + math.Cos(refLatRad)*dz
	u := math.Cos(refLatRad)*math.Cos(refLonRad)*dx + math.Cos(refLatRad)*math.Sin(refLonRad)*dy + math.Sin(refLatRad)*dz

	return NewENUPoint(own.GetDatum(), e, n, u)
}
