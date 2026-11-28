package line

import (
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// ============================================================================
// FindNearestPointOnPolyline 测试
// ============================================================================
func TestFindNearestPointOnPolyline(t *testing.T) {
	p := newTestPolylineProjected(t)
	proj := projection.NewWebMercatorProjector(datum.WGS84)

	// 线段中点附近
	midLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0, 0)
	midProj, err := proj.Project(midLLA)
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}

	dist, idx := FindNearestPointOnPolyline(p, midProj)
	if dist < 0 {
		t.Errorf("distance should be non-negative, got %f", dist)
	}
	if idx < 0 || idx >= 3 {
		t.Errorf("segment index out of range: %d", idx)
	}

	// 在线上时距离应接近 0
	if dist > geography.BASIC_EPSILON*10 {
		t.Errorf("distance for on-line point should be near 0, got %f", dist)
	}
}
