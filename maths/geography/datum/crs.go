package datum

/*
+-----------+-----------------------------------+-------------------------------+
| 编码		| EPSG:4326							| EPSG:4490						|
+-----------+-----------------------------------+-------------------------------+
| 坐标类型	| 地理坐标 (WGS84)					| 地理坐标 (CGCS2000)			|
+-----------+-----------------------------------+-------------------------------+
| 单位		| 度 (°)							| 度 (°)						|
+-----------+-----------------------------------+-------------------------------+
| 核心特点	| 基于GPS使用的标准椭球体，用于经纬度定位	| 中国国家大地坐标系，当前由官方主导	|
+-----------+-----------------------------------+-------------------------------+
| 应用场景	| 原始GPS数据存储、全球地图展示、GeoJSON	| 国内数据存储基准					|
+-----------+-----------------------------------+-------------------------------+
*/

// CRS 坐标系类型（高8位）
type CRS uint16

const (
	// CRS_WGS84 国际 1984 GPS 全球标准坐标系（EPSG:4326）
	CRS_WGS84 CRS = 0x01 << 8
	// CRS_NAD83 北美 1983 基准面坐标系（EPSG:4269）
	CRS_NAD83 CRS = 0x02 << 8
	// CRS_NAD27 北美 1927 基准面坐标系（EPSG:4267）
	CRS_NAD27 CRS = 0x03 << 8
	// CRS_CGCS2000 中国 2000 大地坐标系（EPSG:4490）
	CRS_CGCS2000 CRS = 0x04 << 8
	// CRS_XIAN_1980 西安 1980 坐标系（EPSG:4610）
	CRS_XIAN_1980 CRS = 0x05 << 8
	// CRS_BEIJING_1954 北京 1954 坐标系（EPSG:4214）
	CRS_BEIJING_1954 CRS = 0x06 << 8
)

// GetCode 获取坐标系编码
func (own CRS) GetCode() string {
	switch own {
	case CRS_WGS84:
		return "EPSG:4326"
	case CRS_NAD27:
		return "EPSG:4267"
	case CRS_NAD83:
		return "EPSG:4269"
	case CRS_CGCS2000:
		return "EPSG:4490"
	case CRS_XIAN_1980:
		return "EPSG:4610"
	case CRS_BEIJING_1954:
		return "EPSG:4214"
	default:
		return "EPSG:UNKNOWN"
	}
}

// IsValid 检查坐标系是否有效
func (own CRS) IsValid() bool {
	switch own {
	case CRS_WGS84, CRS_NAD27, CRS_NAD83, CRS_CGCS2000, CRS_XIAN_1980, CRS_BEIJING_1954:
		return true
	default:
		return false
	}
}

// String 返回坐标系名称（与 EPSG 注册名一致）。
func (own CRS) String() string {
	switch own {
	case CRS_WGS84:
		return "WGS84"
	case CRS_NAD27:
		return "NAD27"
	case CRS_NAD83:
		return "NAD83"
	case CRS_CGCS2000:
		return "CGCS2000"
	case CRS_XIAN_1980:
		return "XIAN1980"
	case CRS_BEIJING_1954:
		return "BEIJING1954"
	default:
		return "UNKNOWN"
	}
}
