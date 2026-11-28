package nmea

import (
	"math"
	"testing"
)

func TestConvertRawToDecimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  float64
		want float64
	}{
		// 40°06.97231763′ → 40 + 6.97231763/60
		{"纬度", 4006.97231763, 40 + 6.97231763/60},
		// 116°32.90263794′ → 116 + 32.90263794/60
		{"经度", 11632.90263794, 116 + 32.90263794/60},
		{"整度", 4000, 40},
		{"零值", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ConvertRawToDecimal(tt.raw); math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("ConvertRawToDecimal(%v) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// TestGGA_RoundTripCanonical 验证 GGA 的 String 输出可被解析器原样还原
func TestGGA_RoundTripCanonical(t *testing.T) {
	cases := []string{
		// 常规：卫星数两位、小数字段三位
		"$GPGGA,071654.66,4006.97231404,N,11632.90258655,E,4,37,0.610,35.091,M,-7.863,M,1.400,4070*51",
		// 卫星数不足两位时需要补零
		"$GPGGA,071654.66,4006.97231404,N,11632.90258655,E,1,07,0.900,35.091,M,-7.863,M,1.400,4070*51",
	}

	for _, sentence := range cases {
		gga, err := ParseGGA(sentence)
		if err != nil {
			t.Fatalf("parse %q failed: %v", sentence, err)
		}
		if got := gga.String(); got != sentence {
			t.Fatalf("String() = %q, want %q", got, sentence)
		}

		// 二次往返应保持幂等
		again, err := ParseGGA(gga.String())
		if err != nil {
			t.Fatalf("re-failed to parse output: %v", err)
		}
		if got := again.String(); got != sentence {
			t.Fatalf("second round trip = %q, want %q", got, sentence)
		}
	}
}

// TestRMC_RoundTripCanonical 验证 RMC 的 String 输出可被解析器原样还原
func TestRMC_RoundTripCanonical(t *testing.T) {
	cases := []string{
		// 无磁偏角
		"$GPRMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,,,D*50",
		// 西磁偏角：解析器曾把磁偏角误写进航向字段
		"$GPRMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,1.500,W,D*50",
	}

	for _, sentence := range cases {
		rmc, err := ParseRMC(sentence)
		if err != nil {
			t.Fatalf("parse %q failed: %v", sentence, err)
		}
		if got := rmc.String(); got != sentence {
			t.Fatalf("String() = %q, want %q", got, sentence)
		}
	}
}

// TestRMC_ParseMagVariation 验证磁偏角写入 MagVariation 而不是覆盖航向
func TestRMC_ParseMagVariation(t *testing.T) {
	rmc, err := ParseRMC("$GPRMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,1.500,W,D*50")
	if err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if math.Abs(rmc.GroundHeading-264.8) > 1e-9 {
		t.Fatalf("GroundHeading = %v, want 264.8", rmc.GroundHeading)
	}
	if math.Abs(rmc.MagVariation-1.5) > 1e-9 {
		t.Fatalf("MagVariation = %v, want 1.5", rmc.MagVariation)
	}
	if rmc.MagDir != "W" {
		t.Fatalf("MagDir = %q, want W", rmc.MagDir)
	}
}
