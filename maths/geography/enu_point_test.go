package geography

import (
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

// TestECEFPoint_ToENUPoint_Order 验证 ENU 转换的参考点经纬度顺序
func TestECEFPoint_ToENUPoint_Order(t *testing.T) {
	ref := NewLLAPoint(datum.WGS84, 0, 0, 0)
	enu := NewLLAPoint(datum.WGS84, 0, 0, 100).ConvertToECEFPoint().ConvertToENUPoint(ref)
	east, north, up := enu.GetValues()

	if math.Abs(east) > 1e-6 || math.Abs(north) > 1e-6 || math.Abs(up-100) > 1e-6 {
		t.Fatalf("expected (0, 0, 100), got (%.6f, %.6f, %.6f)", east, north, up)
	}
}
