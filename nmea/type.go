package nmea

// Type 语句类型（NMEA 0183 语句名，如 GGA、RMC）。
type Type string

const (
	// TYPE_GGA 定位信息语句（时间、经纬度、定位质量、卫星数等）
	TYPE_GGA Type = "GGA"
	// TYPE_RMC 推荐最小定位信息语句（时间、经纬度、速度、航向、日期等）
	TYPE_RMC Type = "RMC"
)

// System 卫星系统（语句中的 talker ID 前缀）。
type System string

const (
	// SYSTEM_GPS 美国 GPS 系统（talker ID：GP）
	SYSTEM_GPS System = "GP"
	// SYSTEM_GALILEO 欧盟 Galileo 系统（talker ID：GA）
	SYSTEM_GALILEO System = "GA"
	// SYSTEM_GLONASS 俄罗斯 GLONASS 系统（talker ID：GL）
	SYSTEM_GLONASS System = "GL"
	// SYSTEM_MULTI_GNSS 多系统联合定位（talker ID：GN）
	SYSTEM_MULTI_GNSS System = "GN"
)

// Quality GGA 语句中的定位质量指示（0~6）。
type Quality int

const (
	// QUALITY_INVALID 定位无效
	QUALITY_INVALID Quality = iota
	// QUALITY_GPS GPS 单点定位（SPS）
	QUALITY_GPS
	// QUALITY_DGPS 差分 GPS 定位（DGPS）
	QUALITY_DGPS
	// QUALITY_PPS PPS 固定解
	QUALITY_PPS
	// QUALITY_RTK_FIXED RTK 固定解
	QUALITY_RTK_FIXED
	// QUALITY_RTK_FLOAT RTK 浮点解
	QUALITY_RTK_FLOAT
	// QUALITY_ESTIMATED 估算定位（航位推算）
	QUALITY_ESTIMATED
)

// Mode GGA 语句中的定位模式指示（单字符）。
type Mode string

const (
	// MODE_AUTONOMOUS 自主定位（Autonomous）
	MODE_AUTONOMOUS Mode = "A"
	// MODE_DIFFERENTIAL 差分定位（Differential）
	MODE_DIFFERENTIAL Mode = "D"
	// MODE_ESTIMATED 估算定位（Estimated）
	MODE_ESTIMATED Mode = "E"
	// MODE_RTK_FLOAT RTK 浮点解（Float RTK）
	MODE_RTK_FLOAT Mode = "F"
	// MODE_MANUAL 手动输入模式（Manual input）
	MODE_MANUAL Mode = "M"
	// MODE_NO_FIX 定位无效（No fix）
	MODE_NO_FIX Mode = "N"
	// MODE_PRECISE 精密定位（Precise）
	MODE_PRECISE Mode = "P"
	// MODE_RTK RTK 实时动态（Real time kinematic）
	MODE_RTK Mode = "R"
	// MODE_SIMULATOR 模拟模式（Simulator）
	MODE_SIMULATOR Mode = "S"
)
