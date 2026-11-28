package datum

import (
	"testing"
)

func TestCRS(t *testing.T) {
	t.Log(WGS84.GetEllipsoid())
	t.Log(CGCS2000.GetEllipsoid())
}
