package datum

// Datum 基准面 (包含坐标系和椭球体)
type Datum uint16

const (
	// WGS84 WGS84 坐标系 + WGS84 椭球体
	WGS84 = Datum(uint16(CRS_WGS84) | uint16(ELLIPSOID_WGS84))
	// CGCS2000 CGCS2000 坐标系 + CGCS2000 椭球体
	CGCS2000 = Datum(uint16(CRS_CGCS2000) | uint16(ELLIPSOID_CGCS2000))
)

// GetCRS 获取基准面的地理坐标系
func (own Datum) GetCRS() CRS {
	return CRS(own & 0xFF00)
}

// GetEllipsoid 获取基准面的椭球体模型
func (own Datum) GetEllipsoid() Ellipsoid {
	return Ellipsoid(own & 0x00FF)
}
