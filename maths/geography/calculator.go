package geography

import (
	"math"
)

const (
	BASIC_EPSILON         = 1e-7  // 通用容差精度
	CONVERGENCE_THRESHOLD = 1e-10 // 收敛精度阈值
)

// *********************************************************************************************************************
// 距离计算器
//    CalculateVincentyDistance 通过 Vincenty 反算公式计算两点间的椭球面距离（m）、起始方位角（°）、终点方位角（°）
//    CalculateHaversineDistance 通过 Haversine 公式计算两个坐标之间的球面近似距离（m）、起始方位角（°）、终点方位角（°）
// *********************************************************************************************************************

// # DistanceCalculator 距离计算器
type DistanceCalculator func(point1, point2 LLAPoint) (distance, startAzimuth, endAzimuth float64, err error)

// CalculateVincentyDistance 通过 Vincenty 反算公式计算两点间的椭球面距离（m）、起始方位角（°）、终点方位角（°）
//   - 测试：距离误差小于 0.05%，起始方位角误差小于 0.01%，终点方位角误差小于 0.01%
func CalculateVincentyDistance(point1, point2 LLAPoint) (distance, startAzimuth, endAzimuth float64, err error) {
	if point1.GetDatum() != point2.GetDatum() {
		return 0, 0, 0, ErrDatumNotMatched
	}
	if point1.IsEqualTo(point2) {
		return 0, 0, 0, nil
	}

	ell := point1.GetDatum().GetEllipsoid()
	f := ell.GetFlattening()
	a := ell.GetMajorRadius()
	b := ell.GetMinorRadius()

	lon1, lat1, _ := point1.GetRadianValues()
	lon2, lat2, _ := point2.GetRadianValues()

	U1 := math.Atan((1 - f) * math.Tan(lat1))
	U2 := math.Atan((1 - f) * math.Tan(lat2))

	// 提前计算并保存与 U1、U2 相关的三角函数值（循环中不需要重复计算）
	sinU1 := math.Sin(U1)
	cosU1 := math.Cos(U1)
	sinU2 := math.Sin(U2)
	cosU2 := math.Cos(U2)

	L := lon2 - lon1
	lambda := L
	lambdaPrev := 2 * math.Pi

	// 声明循环内需要修改的变量，以便循环后还能使用
	const _maxIterations = 200 // 最大迭代次数
	var sinLambda, cosLambda, sinSigma, cosSigma, sigma, sinAlpha, cosSqAlpha, cos2SigmaM, C float64
	for i := 0; i < _maxIterations; i++ {
		sinLambda = math.Sin(lambda)
		cosLambda = math.Cos(lambda)

		// 使用循环外已计算好的 sinU1, cosU1, sinU2, cosU2
		sinSigma = math.Sqrt(math.Pow(cosU2*sinLambda, 2) + math.Pow(cosU1*sinU2-sinU1*cosU2*cosLambda, 2))
		if sinSigma == 0 {
			return 0, 0, 0, nil
		}

		cosSigma = sinU1*sinU2 + cosU1*cosU2*cosLambda
		sigma = math.Atan2(sinSigma, cosSigma)
		sinAlpha = cosU1 * cosU2 * sinLambda / sinSigma
		cosSqAlpha = 1 - sinAlpha*sinAlpha
		if cosSqAlpha == 0 {
			cos2SigmaM = 0
		} else {
			cos2SigmaM = cosSigma - 2*sinU1*sinU2/cosSqAlpha
		}

		C = f / 16 * cosSqAlpha * (4 + f*(4-3*cosSqAlpha))
		lambdaPrev = lambda
		lambda = L + (1-C)*f*sinAlpha*(sigma+C*sinSigma*(cos2SigmaM+C*cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)))
		if math.Abs(lambda-lambdaPrev) < CONVERGENCE_THRESHOLD {
			break
		}

		if i == _maxIterations-1 {
			return 0, 0, 0, ErrVincentyNotConverged
		}
	}

	// 计算距离（使用循环最后一次迭代得到的 sinSigma, cosSigma, sigma, cos2SigmaM 等）
	uSq := cosSqAlpha * (a*a - b*b) / (b * b)
	A := 1 + uSq/16384*(4096+uSq*(-768+uSq*(320-175*uSq)))
	B := uSq / 1024 * (256 + uSq*(-128+uSq*(74-47*uSq)))
	deltaSigma := B * sinSigma * (cos2SigmaM + B/4*(cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)-B/6*cos2SigmaM*(-3+4*sinSigma*sinSigma)*(-3+4*cos2SigmaM*cos2SigmaM)))
	distance = b * A * (sigma - deltaSigma)

	// 起始方位角（使用最后一次迭代的 sinLambda, cosLambda）
	az1Rad := math.Atan2(cosU2*sinLambda, cosU1*sinU2-sinU1*cosU2*cosLambda)
	startAzimuth = math.Mod(az1Rad*180/math.Pi+360, 360)

	// 终点方位角
	az2Rad := math.Atan2(cosU1*sinLambda, -sinU1*cosU2+cosU1*sinU2*cosLambda)
	endAzimuth = math.Mod(az2Rad*180/math.Pi+360, 360)

	return
}

// CalculateHaversineDistance 通过 Haversine 公式计算两个坐标之间的球面近似距离（m）、起始方位角（°）、终点方位角（°）
//   - 测试：距离误差小于 0.2%，起始方位角误差小于 0.01%，终点方位角误差小于 0.01%
func CalculateHaversineDistance(point1, point2 LLAPoint) (distance, startAzimuth, endAzimuth float64, err error) {
	if point1.GetDatum() != point2.GetDatum() {
		return 0, 0, 0, ErrDatumNotMatched
	}
	if point1.IsEqualTo(point2) {
		return 0, 0, 0, nil
	}

	lon1, lat1, _ := point1.GetRadianValues()
	lon2, lat2, _ := point2.GetRadianValues()

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance = c * point1.GetDatum().GetEllipsoid().GetMajorRadius()

	// 起始方位角
	yStart := math.Sin(dLon) * math.Cos(lat2)
	xStart := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	startAzimuth = math.Mod(math.Atan2(yStart, xStart)*180/math.Pi+360, 360)

	// 终点方位角（正向）
	yEnd := math.Sin(dLon) * math.Cos(lat1)
	xEnd := math.Sin(lat2)*math.Cos(lat1)*math.Cos(dLon) - math.Cos(lat2)*math.Sin(lat1)
	endAzimuth = math.Mod(math.Atan2(yEnd, xEnd)*180/math.Pi+360, 360)

	return
}

// *********************************************************************************************************************
// 多边形周长计算器
//    CalculatePolygonVincentyPerimeter 基于 Vincenty 公式计算多边形的椭球面周长（m）
//    CalculatePolygonHaversinePerimeter 基于 Haversine 公式计算多边形的球面近似周长（m）
// *********************************************************************************************************************

// # PerimeterCalculator 周长计算器
type PerimeterCalculator func(points ...LLAPoint) (perimeter float64, err error)

// CalculatePolygonVincentyPerimeter 基于 Vincenty 公式计算多边形的椭球面周长（m）
//   - points：多边形顶点序列，至少需要3个坐标点（自动闭合)，支持任意简单多边形（不自交），顶点按顺序排列（顺时针或逆时针）
//   - 经过测试，误差小于 0.4%
func CalculatePolygonVincentyPerimeter(points ...LLAPoint) (float64, error) {
	return calculatePolygonPerimeter(CalculateVincentyDistance, points...)
}

// CalculatePolygonHaversinePerimeter 基于 Haversine 公式计算多边形的球面近似周长（m）
//   - points：多边形顶点序列，至少需要3个坐标点（自动闭合)，支持任意简单多边形（不自交），顶点按顺序排列（顺时针或逆时针）
func CalculatePolygonHaversinePerimeter(points ...LLAPoint) (float64, error) {
	return calculatePolygonPerimeter(CalculateHaversineDistance, points...)
}

// calculatePolygonPerimeter 计算多边形的周长
func calculatePolygonPerimeter(calc DistanceCalculator, points ...LLAPoint) (float64, error) {
	num := len(points)
	if num < 3 {
		return 0, ErrPolygonLessThan3Points
	}

	sum := 0.0
	for i := 0; i < num; i++ {
		dis, _, _, err := calc(points[i], points[(i+1)%num])
		if err != nil {
			return 0, err
		}

		sum += dis
	}

	return math.Abs(sum), nil
}

// **********************************************************************************************************************
// 辅助函数
//    Deduplicate 删除重复地理坐标点
//    ConvertToRadian 把角度值转换为弧度值
// **********************************************************************************************************************

// Deduplicate 删除重复地理坐标点
func Deduplicate[T Equalter[T]](points ...T) []T {
	if len(points) == 0 {
		return []T{}
	}

	filteredPoints := make([]T, 0, len(points)+1)
	for i, a := range points {
		if i == 0 {
			filteredPoints = append(filteredPoints, a)
			continue
		}

		var exist bool
		for _, b := range filteredPoints {
			if b.IsEqualTo(a) {
				exist = true
				break
			}
		}

		if !exist {
			filteredPoints = append(filteredPoints, a)
		}
	}

	return filteredPoints
}

// ConvertToRadian 把角度值转换为弧度值
func ConvertToRadian(value float64) float64 {
	return value * math.Pi / 180.0
}
