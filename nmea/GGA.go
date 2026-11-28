package nmea

import (
	"fmt"
	"strconv"
	"strings"

	"gitee.com/leminewx/gokit"
	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// GGA 表示 NMEA 0183 协议中的全球定位系统定位数据
type GGA struct {
	System          System  `json:"system"`           // 系统（GP/GA/GL/GN）
	UTCTimeStr      string  `json:"utc_time_str"`     // UTC 时间（HHMMSS.sss）
	LatitudeRaw     float64 `json:"latitude_raw"`     // 纬度（度分格式）
	LatitudeDir     string  `json:"latitude_dir"`     // 纬度方向（N/S）
	LongitudeRaw    float64 `json:"longitude_raw"`    // 经度（度分格式）
	LongitudeDir    string  `json:"longitude_dir"`    // 经度方向（E/W）
	Quality         Quality `json:"quality"`          // 质量
	Satellites      int     `json:"satellites"`       // 使用卫星数量（00~24+）
	HDOP            float64 `json:"hdop"`             // HDOP（水平精度因子，数值越小精度越高）
	Altitude        float64 `json:"altitude"`         // 海拔高度（正射高度）
	AltitudeUnit    string  `json:"altitude_unit"`    // 海拔高度单位
	GeoidSeparation float64 `json:"geoid_separation"` // 大地水准面分离（WGS84椭球面与平均海平面的差值）
	GeoidUnit       string  `json:"geoid_unit"`       // 分离值单位
	DiffDataAge     float64 `json:"diff_data_age"`    // 差分GPS数据期限（从最近一次差分信号更新以来的秒数）
	DiffStationID   string  `json:"diff_station_id"`  // 差分参考站ID（差分参考基站编号）
	Checksum        string  `json:"checksum"`         // 校验和（以 * 开头，后跟两位十六进制数）
}

// ParseGGA 解析 NMEA 0183 协议的 GGA 数据
func ParseGGA(sentence string) (gga *GGA, err error) {
	if !strings.HasPrefix(sentence, "$") || !strings.Contains(sentence, string(TYPE_GGA)) {
		return nil, ErrNotGGASentence
	}

	// 去除末尾的换行符并按逗号分割
	// $GGA,071654.66,4006.97231404,N,11632.90258655,E,4,37,0.61,35.091,M,-7.863,M,1.4,4070*51
	parts := strings.Split(strings.TrimRight(sentence, "\r\n"), ",")
	if len(parts) != 15 {
		return nil, ErrGGACountMismatch
	}

	lasts := strings.Split(parts[14], "*")
	if len(lasts) != 2 {
		return nil, ErrMissingChecksum
	}
	gga = &GGA{
		System:        System(strings.Split(parts[0], string(TYPE_GGA))[0][1:]),
		UTCTimeStr:    parts[1],
		LatitudeDir:   parts[3],
		LongitudeDir:  parts[5],
		AltitudeUnit:  parts[10],
		GeoidUnit:     parts[12],
		DiffStationID: lasts[0],
		Checksum:      lasts[1],
	}

	if gga.LatitudeRaw, err = strconv.ParseFloat(parts[2], 64); err != nil {
		return nil, err
	}
	if gga.LongitudeRaw, err = strconv.ParseFloat(parts[4], 64); err != nil {
		return nil, err
	}
	if quality, err := strconv.Atoi(parts[6]); err != nil {
		return nil, err
	} else {
		gga.Quality = Quality(quality)
	}
	if gga.Satellites, err = strconv.Atoi(parts[7]); err != nil {
		return nil, err
	}
	if gga.HDOP, err = strconv.ParseFloat(parts[8], 64); err != nil {
		return nil, err
	}
	if gga.Altitude, err = strconv.ParseFloat(parts[9], 64); err != nil {
		return nil, err
	}
	if gga.GeoidSeparation, err = strconv.ParseFloat(parts[11], 64); err != nil {
		return nil, err
	}
	if gga.DiffDataAge, err = strconv.ParseFloat(parts[13], 64); err != nil {
		return nil, err
	}

	return
}

// GetLLAPoint 获取经纬度坐标消息
//   - dat: 地理基准面，默认使用 WGS84
func (own *GGA) GetLLAPoint(dat ...datum.Datum) geography.LLAPoint {
	return geography.NewLLAPoint(gokit.GetValueWithDefault(datum.WGS84, dat...), ConvertRawToDecimal(own.LatitudeRaw), ConvertRawToDecimal(own.LongitudeRaw))
}

func (own *GGA) String() string {
	// $GPGGA,071654.66,4006.97231404,N,11632.90258655,E,4,37,0.610,35.091,M,-7.863,M,1.400,4070*51
	// 小数字段固定 3 位、卫星数补足 2 位，使 String 的输出能被 ParseGGA 原样还原
	return fmt.Sprintf("$%sGGA,%s,%.8f,%s,%.8f,%s,%d,%02d,%.3f,%.3f,%s,%.3f,%s,%.3f,%s*%s", own.System, own.UTCTimeStr,
		own.LatitudeRaw, own.LatitudeDir, own.LongitudeRaw, own.LongitudeDir, own.Quality, own.Satellites,
		own.HDOP, own.Altitude, own.AltitudeUnit, own.GeoidSeparation,
		own.GeoidUnit, own.DiffDataAge, own.DiffStationID, own.Checksum)
}
