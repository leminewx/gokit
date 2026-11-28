package projection

import (
	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// Projector 投影器
type Projector interface {
	// Project 投影
	Project(geography.LLAPoint) (Point, error)
	// Unproject 反投影
	Unproject(Point) (geography.LLAPoint, error)

	// GetCode 获取投影坐标系代码
	GetCode() Code
	// GetDatum 获取反投影坐标系基准面
	GetDatum() datum.Datum
	// ValidateLatitude 检查纬度是否有效
	ValidateLatitude(lat float64) error
}

// 编译期断言：所有投影器实现 Projector 接口
var (
	_ Projector = (*webMercatorProjector)(nil)
	_ Projector = (*lambertCylindricalEqualAreaProjector)(nil)
	_ Projector = (*localProjector)(nil)
)
