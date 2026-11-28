package projection

import (
	"math"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// webMercatorProjector 伪墨卡托投影器（Web Mercator）
type webMercatorProjector struct {
	datum datum.Datum
}

// NewWebMercatorProjector 新建一个伪墨卡托投影器（Web Mercator，EPSG:3857）
//   - 基于球体的正轴等角圆柱投影，赤道为基准线
//   - 广泛用于 Web 地图（Google、OSM 等）
func NewWebMercatorProjector(earth datum.Datum) Projector {
	return &webMercatorProjector{datum: earth}
}

// Project 将地理坐标投影到 EPSG:3857 的投影坐标（东向，北向，单位：m）
//   - 注意：EPSG:3857 坐标投影的有效范围是 85.05112878°S ~ 85.05112878°N
func (own *webMercatorProjector) Project(point geography.LLAPoint) (Point, error) {
	// 检查纬度范围
	_, lat, _ := point.GetValues()
	if err := own.ValidateLatitude(lat); err != nil {
		return Point{}, err
	}

	lonRad, latRad, _ := point.GetRadianValues()
	radius := point.GetDatum().GetEllipsoid().GetMajorRadius()

	// 计算投影坐标
	east := radius * lonRad
	north := radius * math.Log(math.Tan(math.Pi/4+latRad/2))
	return Point{east, north}, nil
}

// Unproject 将 EPSG:3857 投影坐标点反投影到地理坐标系（单位：°）
//   - 注意：EPSG:3857 坐标投影的有效范围是 -85.05112878°S ~ 85.05112878°N
func (own *webMercatorProjector) Unproject(point Point) (geography.LLAPoint, error) {
	radius := own.datum.GetEllipsoid().GetMajorRadius()
	easting, northing := point.GetValues()

	// 1. 计算经度
	lonRad := easting / radius
	lon := lonRad * 180.0 / math.Pi

	// 2. 计算纬度
	latRad := math.Pi/2 - 2*math.Atan(math.Exp(-northing/radius))
	lat := latRad * 180.0 / math.Pi

	// 3. 检查纬度范围
	if err := own.ValidateLatitude(lat); err != nil {
		return geography.LLAPoint{}, err
	}

	return geography.NewLLAPoint(own.datum, lon, lat), nil
}

// GetCode 获取投影坐标系代码
func (own *webMercatorProjector) GetCode() Code {
	return EPSG_3857
}

// GetDatum 获取反投影坐标系基准面
func (own *webMercatorProjector) GetDatum() datum.Datum {
	return own.datum
}

// ValidateLatitude 检查纬度是否有效
func (own *webMercatorProjector) ValidateLatitude(lat float64) error {
	if lat < -85.05112878 || 85.05112878 < lat {
		return wrapLatitudeError("EPSG:3857 valid latitude out of range [-85.05112878, 85.05112878]: %f", lat)
	}

	return nil
}
