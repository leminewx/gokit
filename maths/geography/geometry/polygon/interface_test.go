package polygon

import (
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

func TestIsPointInPolygonByRay_Inside(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	centroid := poly.GetProjectedCentroid()

	if !IsPointInPolygonByRay(poly, centroid) {
		t.Errorf("centroid %v should be inside polygon", centroid)
	}
}

func TestIsPointInPolygonByRay_Outside(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	farPoint, _ := newMercatorProjector().Project(lla(120, 40, 0))

	if IsPointInPolygonByRay(poly, farPoint) {
		t.Errorf("far point %v should be outside polygon", farPoint)
	}
}

func TestIsPointInPolygonByRay_OnEdge(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	vts := poly.GetProjectedVertexes()

	// 取底边中点
	mid := projection.Point{
		(vts[0][0] + vts[1][0]) / 2,
		(vts[0][1] + vts[1][1]) / 2,
	}

	// onEdge 默认为 false，边上点不应判定为内部
	if IsPointInPolygonByRay(poly, mid) {
		t.Error("point on edge should not be inside when onEdge is default (false)")
	}

	// onEdge=true 时边上点应判定为内部
	if !IsPointInPolygonByRay(poly, mid, true) {
		t.Error("point on edge should be inside when onEdge=true")
	}
}

func TestIsPointInPolygonByRay_OnVertex(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	vts := poly.GetProjectedVertexes()

	// 测试第一个顶点
	vertex := vts[0]

	if IsPointInPolygonByRay(poly, vertex) {
		t.Error("vertex should not be inside when onEdge is default (false)")
	}
	if !IsPointInPolygonByRay(poly, vertex, true) {
		t.Error("vertex should be inside when onEdge=true")
	}
}

func TestIsPointInPolygonByRay_Triangle(t *testing.T) {
	poly := makeProjectedTrianglePoly(t)
	centroid := poly.GetProjectedCentroid()

	if !IsPointInPolygonByRay(poly, centroid) {
		t.Errorf("centroid %v should be inside triangle", centroid)
	}
}

func TestIsPointInPolygonByRay_WithoutProject(t *testing.T) {
	poly, err := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("create polygon: %v", err)
	}

	// 未投影时，GetProjectedVertexes 返回 nil，点不应该被判定为在内部
	if IsPointInPolygonByRay(poly, projection.Point{0, 0}) {
		t.Error("point should not be inside unprojected polygon")
	}
}

func TestIsPointInPolygonByRay_UnprojectedPolygonReturnsNil(t *testing.T) {
	poly, err := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("create polygon: %v", err)
	}

	// 未投影的多边形调用射线法时，IsOnPolyline 会收到 nil 切片
	// 射线法循环不会执行（num=0），应返回 false
	if IsPointInPolygonByRay(poly, projection.Point{0, 0}) {
		t.Error("unprojected polygon should not contain any point via ray casting")
	}
}

// TestIsPointInPolygonByRay_TableDriven 表驱动测试，覆盖多种位置关系
func TestIsPointInPolygonByRay_TableDriven(t *testing.T) {
	poly := makeProjectedSquarePoly(t)

	tests := []struct {
		name   string
		point  projection.Point
		onEdge bool
		want   bool
	}{
		{name: "center", point: poly.GetProjectedCentroid(), onEdge: false, want: true},
		{name: "far outside", point: projection.Point{999999, 999999}, onEdge: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPointInPolygonByRay(poly, tt.point, tt.onEdge)
			if got != tt.want {
				t.Errorf("IsPointInPolygonByRay(%v, onEdge=%v) = %v, want %v",
					tt.point, tt.onEdge, got, tt.want)
			}
		})
	}
}

// TestIsPointInPolygonByRay_ConvexPolygon 凸多边形多方向射线测试
func TestIsPointInPolygonByRay_ConvexPolygon(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	centroid := poly.GetProjectedCentroid()
	cx, cy := centroid.GetValues()

	// 在质心周围采样若干点，应全部在内部（偏移量很小）
	offsets := []struct{ dx, dy float64 }{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	}
	for _, off := range offsets {
		pt := projection.Point{cx + off.dx, cy + off.dy}
		if !IsPointInPolygonByRay(poly, pt) {
			t.Errorf("point near centroid %v should be inside", pt)
		}
	}
}

// TestIsPointInPolygonByRay_OnBoundaryWithEdgeFlag 边界点 onEdge 标志组合测试
func TestIsPointInPolygonByRay_OnBoundaryWithEdgeFlag(t *testing.T) {
	poly := makeProjectedSquarePoly(t)
	vts := poly.GetProjectedVertexes()

	// 构造一个位于边上的测试点
	edgePoint := projection.Point{
		(vts[0][0] + vts[1][0]) / 2,
		(vts[0][1] + vts[1][1]) / 2,
	}

	// onEdge 为多种输入形式
	if IsPointInPolygonByRay(poly, edgePoint) {
		t.Error("edgePoint should return false when onEdge is not provided")
	}
	if !IsPointInPolygonByRay(poly, edgePoint, true) {
		t.Error("edgePoint should return true when onEdge=true")
	}
	if IsPointInPolygonByRay(poly, edgePoint, false) {
		t.Error("edgePoint should return false when onEdge=false")
	}
}
