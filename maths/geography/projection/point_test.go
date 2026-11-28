package projection

import (
	"testing"
)

// TestPointString 验证坐标点字符串格式
func TestPointString(t *testing.T) {
	t.Parallel()

	point := Point{1.5, -2.25}
	if got := point.String(); got != "(1.500000, -2.250000)" {
		t.Errorf("Point.String() = %q, want (1.500000, -2.250000)", got)
	}
}
