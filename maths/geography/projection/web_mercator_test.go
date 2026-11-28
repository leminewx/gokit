package projection

import (
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestEPSG3857(t *testing.T) {
	point1 := geography.NewLLAPoint(datum.WGS84, 116.5488492356011, 40.11609330383517)
	point2 := geography.NewLLAPoint(datum.WGS84, 116.54885612111968, 40.11610172030012)
	t.Log(geography.CalculateVincentyDistance(point1, point2))

	projector := NewWebMercatorProjector(datum.WGS84)
	p1, _ := projector.Project(point1)
	p2, _ := projector.Project(point2)
	t.Log(CalculateEuclidDistance(p1, p2))
}
