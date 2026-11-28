package geography

import (
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// TestLLAPoint_ToECEFPoint_Order 验证经纬度顺序没有被交换
func TestLLAPoint_ToECEFPoint_Order(t *testing.T) {
	const tolerance = 1e-2 // 米

	cases := []struct {
		name     string
		lon, lat float64
		wantX    float64
		wantY    float64
		wantZ    float64
	}{
		{"赤道与本初子午线", 0, 0, 6378137, 0, 0},
		{"赤道与东经90度", 90, 0, 0, 6378137, 0},
		{"北极点", 0, 90, 0, 0, 6356752.314245},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			xyz := NewLLAPoint(datum.WGS84, c.lon, c.lat, 0).ConvertToECEFPoint()
			x, y, z := xyz.GetValues()

			if math.Abs(x-c.wantX) > tolerance || math.Abs(y-c.wantY) > tolerance || math.Abs(z-c.wantZ) > tolerance {
				t.Fatalf("expected (%.3f, %.3f, %.3f), got (%.3f, %.3f, %.3f)", c.wantX, c.wantY, c.wantZ, x, y, z)
			}
		})
	}
}
