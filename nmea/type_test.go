package nmea

import "testing"

// TestTypeValues 固化语句类型的报文取值，避免重命名时误改协议字面量
func TestTypeValues(t *testing.T) {
	t.Parallel()

	cases := map[Type]string{
		TYPE_GGA: "GGA",
		TYPE_RMC: "RMC",
	}

	for got, want := range cases {
		if string(got) != want {
			t.Errorf("语句类型 = %q, want %q", got, want)
		}
	}
}

func TestSystemValues(t *testing.T) {
	t.Parallel()

	cases := map[System]string{
		SYSTEM_GPS:        "GP",
		SYSTEM_GALILEO:    "GA",
		SYSTEM_GLONASS:    "GL",
		SYSTEM_MULTI_GNSS: "GN",
	}

	for got, want := range cases {
		if string(got) != want {
			t.Errorf("卫星系统 = %q, want %q", got, want)
		}
	}
}

// TestQualityValues 固化定位质量的枚举序号，序号与 NMEA 0183 协议一致
func TestQualityValues(t *testing.T) {
	t.Parallel()

	cases := map[Quality]int{
		QUALITY_INVALID:   0,
		QUALITY_GPS:       1,
		QUALITY_DGPS:      2,
		QUALITY_PPS:       3,
		QUALITY_RTK_FIXED: 4,
		QUALITY_RTK_FLOAT: 5,
		QUALITY_ESTIMATED: 6,
	}

	for got, want := range cases {
		if int(got) != want {
			t.Errorf("定位质量 = %d, want %d", got, want)
		}
	}
}

// TestModeValues 固化定位模式的单字符取值
func TestModeValues(t *testing.T) {
	t.Parallel()

	cases := map[Mode]string{
		MODE_AUTONOMOUS:   "A",
		MODE_DIFFERENTIAL: "D",
		MODE_ESTIMATED:    "E",
		MODE_RTK_FLOAT:    "F",
		MODE_MANUAL:       "M",
		MODE_NO_FIX:       "N",
		MODE_PRECISE:      "P",
		MODE_RTK:          "R",
		MODE_SIMULATOR:    "S",
	}

	for got, want := range cases {
		if string(got) != want {
			t.Errorf("定位模式 = %q, want %q", got, want)
		}
	}
}
