package datum

import "math"

// Ellipsoid 椭球体类型（低8位）
type Ellipsoid uint16

const (
	// ELLIPSOID_WGS84 WGS 1984 椭球体（EPSG:7030）
	ELLIPSOID_WGS84 Ellipsoid = 0x01
	// ELLIPSOID_GRS80 GRS 1980 椭球体（EPSG:7019）
	ELLIPSOID_GRS80 Ellipsoid = 0x02
	// ELLIPSOID_IAG1975 IAG 1975 椭球体（EPSG:7049）
	ELLIPSOID_IAG1975 Ellipsoid = 0x03
	// ELLIPSOID_CGCS2000 CGCS 2000 椭球体（实际使用 GRS80 椭球体）
	ELLIPSOID_CGCS2000 Ellipsoid = 0x04
	// ELLIPSOID_KRASOVSKY Krasovsky 1940 椭球体（EPSG:7024）
	ELLIPSOID_KRASOVSKY Ellipsoid = 0x05
)

// GetFlattening 获取椭球体扁率
func (own Ellipsoid) GetFlattening() float64 {
	switch own {
	case ELLIPSOID_WGS84:
		return 1 / 298.257223563
	case ELLIPSOID_GRS80, ELLIPSOID_CGCS2000:
		return 1 / 298.257222101
	case ELLIPSOID_IAG1975:
		return 1 / 298.257
	case ELLIPSOID_KRASOVSKY:
		return 1 / 298.3
	default:
		return 0.0
	}
}

// GetMajorRadius 获取椭球体的长半径（赤道半径）
func (own Ellipsoid) GetMajorRadius() float64 {
	switch own {
	case ELLIPSOID_WGS84, ELLIPSOID_GRS80, ELLIPSOID_CGCS2000:
		return 6378137.0
	case ELLIPSOID_IAG1975:
		return 6378140.0
	case ELLIPSOID_KRASOVSKY:
		return 6378245.0
	default:
		return 0.0
	}
}

// GetMinorRadius 获取椭球体的短半径（极半径）
func (own Ellipsoid) GetMinorRadius() float64 {
	return own.GetMajorRadius() * (1 - own.GetFlattening())
}

// GetFirstEccentricitySquared 获取第一偏心率平方
func (own Ellipsoid) GetFirstEccentricitySquared() float64 {
	switch own {
	case ELLIPSOID_WGS84:
		return 0.00669437999014
	case ELLIPSOID_CGCS2000, ELLIPSOID_GRS80:
		return 0.00669438002290
	case ELLIPSOID_KRASOVSKY:
		return 0.00669342162297
	case ELLIPSOID_IAG1975:
		return 0.006694384999
	default:
		return 0.0
	}
}

// GetSecondEccentricitySquared 获取第二偏心率平方
func (own Ellipsoid) GetSecondEccentricitySquared() float64 {
	switch own {
	case ELLIPSOID_WGS84:
		return 0.00673949674228
	case ELLIPSOID_CGCS2000, ELLIPSOID_GRS80:
		return 0.00673949677548
	case ELLIPSOID_KRASOVSKY:
		return 0.00673852541468
	case ELLIPSOID_IAG1975:
		return 0.006739501819
	default:
		return 0.0
	}
}

// CalculatePrimeVerticalRadius 计算卯酉圈曲率半径
func (own Ellipsoid) CalculatePrimeVerticalRadius(lat float64) float64 {
	latRad := lat * math.Pi / 180.0
	sinLat := math.Sin(latRad)
	return own.GetMajorRadius() / math.Sqrt(1-own.GetFirstEccentricitySquared()*sinLat*sinLat)
}

// CalculateMeridianArc 计算子午线弧长（从赤道到纬度 lat 弧度）
//   - 使用复合 Simpson 数值积分，精度优于 1e-6 米
func (own Ellipsoid) CalculateMeridianArc(lat float64) float64 {
	// 系数计算 (基于 e^2 的幂)
	e2 := own.GetFirstEccentricitySquared()
	e4 := e2 * e2
	e6 := e4 * e2
	e8 := e6 * e2

	// 子午线弧长公式: X = a(1-e^2) * (A0*B - A2*sin2B + A4*sin4B - A6*sin6B + A8*sin8B)
	m0 := own.GetMajorRadius() * (1 - e2)

	A0 := 1.0 + 3.0/4.0*e2 + 45.0/64.0*e4 + 175.0/256.0*e6 + 11025.0/16384.0*e8
	A2 := 3.0/4.0*e2 + 15.0/16.0*e4 + 525.0/512.0*e6 + 2205.0/2048.0*e8
	A4 := 15.0/64.0*e4 + 105.0/256.0*e6 + 2205.0/4096.0*e8
	A6 := 35.0/512.0*e6 + 315.0/2048.0*e8
	A8 := 315.0 / 16384.0 * e8

	return m0 * (A0*lat - A2*math.Sin(2*lat) + A4*math.Sin(4*lat) - A6*math.Sin(6*lat) + A8*math.Sin(8*lat))
}

// GetCode 获取椭球体代码
func (own Ellipsoid) GetCode() string {
	switch own {
	case ELLIPSOID_WGS84:
		return "EPSG:7030"
	case ELLIPSOID_GRS80:
		return "EPSG:7019"
	case ELLIPSOID_IAG1975:
		return "EPSG:7049"
	case ELLIPSOID_CGCS2000:
		return "EPSG:7019"
	case ELLIPSOID_KRASOVSKY:
		return "EPSG:7024"
	default:
		return "UNKNOWN"
	}
}

// IsValid 检查椭球体是否有效
func (own Ellipsoid) IsValid() bool {
	switch own {
	case ELLIPSOID_WGS84, ELLIPSOID_GRS80, ELLIPSOID_IAG1975, ELLIPSOID_CGCS2000, ELLIPSOID_KRASOVSKY:
		return true
	default:
		return false
	}
}

func (own Ellipsoid) String() string {
	switch own {
	case ELLIPSOID_WGS84:
		return "WGS84"
	case ELLIPSOID_GRS80:
		return "GRS80"
	case ELLIPSOID_IAG1975:
		return "IAG1975"
	case ELLIPSOID_CGCS2000:
		return "CGCS2000"
	case ELLIPSOID_KRASOVSKY:
		return "KRASOVSKY1980"
	default:
		return "UNKNOWN"
	}
}
