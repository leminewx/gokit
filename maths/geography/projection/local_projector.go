package projection

import (
	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// localProjector 将任意投影器输出的坐标转换到本地平面（以 center 为原点，保持原投影方向）
type localProjector struct {
	baseProjector Projector
	centerGeo     geography.LLAPoint
	centerProj    Point
}

// NewLocalProjector 创建一个本地平面投影器
//   - base: 投影器（从地理坐标投影到投影坐标）
//   - center: 投影原点经纬度
func NewLocalProjector(base Projector, center geography.LLAPoint) Projector {
	centerProj, _ := base.Project(center)
	return &localProjector{
		baseProjector: base,
		centerGeo:     center,
		centerProj:    centerProj,
	}
}

// Project 投影
func (own *localProjector) Project(point geography.LLAPoint) (Point, error) {
	proj, err := own.baseProjector.Project(point)
	if err != nil {
		return Point{}, err
	}

	easting, northing := proj[0], proj[1]
	originEasting, originNorthing := own.centerProj[0], own.centerProj[1]
	return Point{easting - originEasting, northing - originNorthing}, nil
}

// Unproject 反投影
func (own *localProjector) Unproject(point Point) (geography.LLAPoint, error) {
	easting, northing := point.GetValues()
	originEasting, originNorthing := own.centerProj.GetValues()
	return own.baseProjector.Unproject(Point{easting + originEasting, northing + originNorthing})
}

// GetCode 获取投影坐标系代码
func (own *localProjector) GetCode() Code {
	return own.baseProjector.GetCode()
}

// GetDatum 获取反投影坐标系基准面
func (own *localProjector) GetDatum() datum.Datum {
	return own.baseProjector.GetDatum()
}

// ValidateLatitude 检查纬度是否有效
func (own *localProjector) ValidateLatitude(lat float64) error {
	return own.baseProjector.ValidateLatitude(lat)
}
