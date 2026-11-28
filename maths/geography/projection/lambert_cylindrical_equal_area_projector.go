package projection

import (
	"math"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// lambertCylindricalEqualAreaProjector 等积圆柱投影器（EPSG:6933）
type lambertCylindricalEqualAreaProjector struct {
	datum datum.Datum
}

// NewLambertCylindricalEqualAreaProjector 新建一个等积圆柱投影器（EPSG:6933，EASE-Grid 2.0 全球版）
//   - 投影公式：x = R * λ, y = R * sin(φ)
//   - 赤道为标准纬线，面积无变形，角度和形状变形严重
func NewLambertCylindricalEqualAreaProjector(earth datum.Datum) Projector {
	return &lambertCylindricalEqualAreaProjector{datum: earth}
}

// Project 将地理坐标投影到 EPSG:6933 的投影坐标（东向，北向，单位：m）
//   - 注意：EPSG:6933 坐标投影的有效范围是 86°S ~ 86°N
func (own *lambertCylindricalEqualAreaProjector) Project(point geography.LLAPoint) (Point, error) {
	// 检查纬度范围
	_, lat, _ := point.GetValues()
	if err := own.ValidateLatitude(lat); err != nil {
		return Point{}, err
	}

	lonRad, latRad, _ := point.GetRadianValues()
	radius := point.GetDatum().GetEllipsoid().GetMajorRadius()

	// 应用公式：x = R * λ, y = R * sin(φ)
	east := radius * lonRad
	north := radius * math.Sin(latRad)

	return Point{east, north}, nil
}

// Unproject 将 EPSG:6933 投影坐标点反投影到地理坐标系（单位：°）
//   - 注意：EPSG:6933 坐标投影的有效范围是 86°S ~ 86°N
func (own *lambertCylindricalEqualAreaProjector) Unproject(point Point) (geography.LLAPoint, error) {
	radius := own.datum.GetEllipsoid().GetMajorRadius()
	easting, northing := point.GetValues()

	// 1. 检查北向值是否在有效范围内（因为 sin(lat) ∈ [-1,1]）
	const eps = 1e-9 // 允许的误差范围
	if math.Abs(northing) > radius+eps {
		return geography.LLAPoint{}, wrapNorthingError("EPSG:6933 northing out of range [-%f, %f]: %f", radius, radius, northing)
	}
	if northing > radius {
		northing = radius
	} else if northing < -radius {
		northing = -radius
	}

	// 2. 计算经度：lon = x / R
	lonRad := easting / radius
	lon := lonRad * 180.0 / math.Pi

	// 3. 计算纬度：lat = arcsin(y / R)
	latRad := math.Asin(northing / radius)
	lat := latRad * 180.0 / math.Pi

	// 4. 检查纬度范围
	if err := own.ValidateLatitude(lat); err != nil {
		return geography.LLAPoint{}, err
	}

	return geography.NewLLAPoint(own.datum, lon, lat), nil
}

// GetCode 获取投影坐标系代码
func (own *lambertCylindricalEqualAreaProjector) GetCode() Code {
	return EPSG_6933
}

// GetDatum 获取反投影坐标系基准面
func (own *lambertCylindricalEqualAreaProjector) GetDatum() datum.Datum {
	return own.datum
}

// ValidateLatitude 检查纬度是否有效
func (own *lambertCylindricalEqualAreaProjector) ValidateLatitude(lat float64) error {
	if lat < -86.0 || 86.0 < lat {
		return wrapLatitudeError("EPSG:6933 valid latitude out of range [-86°, 86°]: %f", lat)
	}

	return nil
}
