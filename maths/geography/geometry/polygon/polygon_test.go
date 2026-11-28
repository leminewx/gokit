package polygon

import (
	"fmt"
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// ============================================================================
// NewLLAPolygon 创建测试
// ============================================================================
func TestNewLLAPolygon_Basic(t *testing.T) {
	poly, err := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poly.GetVertexNumber() != 4 {
		t.Errorf("vertex number = %d, want 4", poly.GetVertexNumber())
	}
	if poly.GetEdgeNumber() != 4 {
		t.Errorf("edge number = %d, want 4", poly.GetEdgeNumber())
	}
	if poly.GetPerimeter() <= 0 {
		t.Errorf("perimeter = %f, want > 0", poly.GetPerimeter())
	}
	if poly.GetDatum() != datum.WGS84 {
		t.Errorf("datum = %v, want WGS84", poly.GetDatum())
	}
}

func TestNewLLAPolygon_TooFewPoints(t *testing.T) {
	_, err := NewLLAPolygon([]geography.LLAPoint{
		lla(116, 39, 0),
		lla(116, 40, 0),
	}, geography.CalculateHaversineDistance)
	if err == nil {
		t.Error("expected error for < 3 points")
	}
}

func TestNewLLAPolygon_Deduplicate(t *testing.T) {
	// 重复相邻点
	vts := []geography.LLAPoint{
		lla(116.397, 39.908, 0),
		lla(116.397, 39.908, 0), // 重复
		lla(116.398, 39.908, 0),
		lla(116.398, 39.909, 0),
		lla(116.397, 39.909, 0),
		lla(116.397, 39.909, 0), // 重复
	}

	poly, err := NewLLAPolygon(vts, geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poly.GetVertexNumber() != 4 {
		t.Errorf("vertex number = %d, want 4 (after dedup)", poly.GetVertexNumber())
	}
}

func TestNewLLAPolygon_DifferentDatum(t *testing.T) {
	vts := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 116, 39, 0),
		geography.NewLLAPoint(datum.CGCS2000, 116, 40, 0),
		geography.NewLLAPoint(datum.WGS84, 117, 39, 0),
	}

	_, err := NewLLAPolygon(vts, geography.CalculateHaversineDistance)
	if err == nil {
		t.Error("expected error for different datums")
	}
}

func TestNewLLAPolygon_Triangle(t *testing.T) {
	poly, err := NewLLAPolygon(makeTriangleVertexes(), geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poly.GetVertexNumber() != 3 {
		t.Errorf("vertex number = %d, want 3", poly.GetVertexNumber())
	}
	if poly.GetEdgeNumber() != 3 {
		t.Errorf("edge number = %d, want 3", poly.GetEdgeNumber())
	}
}

// ============================================================================
// Project 投影测试
// ============================================================================
func TestLLAPolygon_Project(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	projector := newMercatorProjector()
	if err := poly.Project(projector); err != nil {
		t.Fatalf("project error: %v", err)
	}

	// 投影后面积应为正
	if poly.GetProjectedArea() <= 0 {
		t.Errorf("area = %f, want > 0", poly.GetProjectedArea())
	}

	// 投影周长应为正
	if poly.GetProjectedPerimeter() <= 0 {
		t.Errorf("projected perimeter = %f, want > 0", poly.GetProjectedPerimeter())
	}

	// 投影顶点数量应与原始顶点相同
	if len(poly.GetProjectedVertexes()) != poly.GetVertexNumber() {
		t.Errorf("projected vertices = %d, want %d", len(poly.GetProjectedVertexes()), poly.GetVertexNumber())
	}

	// 质心不应为零向量
	centroid := poly.GetProjectedCentroid()
	cx, cy := centroid.GetValues()
	if cx == 0 && cy == 0 {
		t.Error("centroid should not be zero")
	}
}

func TestLLAPolygon_AreaUnchangedOnReprojection(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	projector := newMercatorProjector()
	poly.Project(projector)
	area1 := poly.GetProjectedArea()
	centroid1 := poly.GetProjectedCentroid()

	// 再次投影
	poly.Project(projector)
	area2 := poly.GetProjectedArea()
	centroid2 := poly.GetProjectedCentroid()

	if math.Abs(area1-area2) > geography.BASIC_EPSILON*1000 {
		t.Errorf("area changed after reprojection: %f -> %f", area1, area2)
	}

	cx1, cy1 := centroid1.GetValues()
	cx2, cy2 := centroid2.GetValues()
	if math.Abs(cx1-cx2) > geography.BASIC_EPSILON || math.Abs(cy1-cy2) > geography.BASIC_EPSILON {
		t.Errorf("centroid changed after reprojection: (%f,%f) -> (%f,%f)", cx1, cy1, cx2, cy2)
	}
}

func TestLLAPolygon_GetProjectedPerimeterWithoutProject(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	if pp := poly.GetProjectedPerimeter(); pp != 0 {
		t.Errorf("projected perimeter = %f, want 0 before projection", pp)
	}
}

// ============================================================================
// GetVertexes / GetVertexAt / GetVertexNumber 测试
// ============================================================================
func TestLLAPolygon_GetVertexes(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	vts := poly.GetVertexes()

	if len(vts) != 4 {
		t.Errorf("len = %d, want 4", len(vts))
	}

	// 应为副本
	vts[0] = geography.LLAPoint{}
	if pvts := poly.GetVertexes(); pvts[0].IsEqualTo(geography.LLAPoint{}) {
		t.Error("GetVertexes should return a copy")
	}
}

func TestLLAPolygon_GetVertexAt(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	// 有效索引
	v, ok := poly.GetVertexAt(0)
	if !ok {
		t.Error("GetVertexAt(0) should be ok")
	}
	lon, lat, _ := v.GetValues()
	if math.Abs(lon-116.397) > geography.BASIC_EPSILON || math.Abs(lat-39.908) > geography.BASIC_EPSILON {
		t.Errorf("vertex[0] = (%f,%f), want (116.397,39.908)", lon, lat)
	}

	// 越界
	_, ok = poly.GetVertexAt(-1)
	if ok {
		t.Error("GetVertexAt(-1) should not be ok")
	}
	_, ok = poly.GetVertexAt(4)
	if ok {
		t.Error("GetVertexAt(4) should not be ok")
	}
}

// ============================================================================
// GetEdges / GetEdgeAt / GetEdgeNumber 测试
// ============================================================================
func TestLLAPolygon_GetEdges(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	edges := poly.GetEdges()

	if len(edges) != 4 {
		t.Errorf("edges count = %d, want 4", len(edges))
	}

	// 闭合边：最后一条边应从顶点[3] 回到顶点[0]
	lastEnd, _ := edges[3].GetEndPoint()
	firstStart, _ := edges[0].GetStartPoint()
	if !lastEnd.IsEqualTo(firstStart) {
		t.Error("closing edge: last edge end != first edge start")
	}

	// 应为副本
	edges[0] = nil
	if poly.GetEdges()[0] == nil {
		t.Error("GetEdges should return a copy")
	}
}

func TestLLAPolygon_GetEdgeAt(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	e, ok := poly.GetEdgeAt(0)
	if !ok {
		t.Error("GetEdgeAt(0) should be ok")
	}
	if e == nil {
		t.Error("edge should not be nil")
	}

	_, ok = poly.GetEdgeAt(-1)
	if ok {
		t.Error("GetEdgeAt(-1) should not be ok")
	}
	_, ok = poly.GetEdgeAt(4)
	if ok {
		t.Error("GetEdgeAt(4) should not be ok")
	}
}

// ============================================================================
// ContainPoint 包含测试
// ============================================================================
func TestLLAPolygon_ContainPoint_Inside(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	poly.Project(newMercatorProjector())

	// 投影中心点（应在多边形内部）
	centroid := poly.GetProjectedCentroid()
	if !poly.ContainPoint(centroid) {
		t.Error("centroid should be inside the polygon")
	}
}

func TestLLAPolygon_ContainPoint_Outside(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	poly.Project(newMercatorProjector())

	// 远处的点应该在外面
	farPoint, _ := newMercatorProjector().Project(lla(120, 40, 0))
	if poly.ContainPoint(farPoint) {
		t.Error("far point should be outside")
	}
}

func TestLLAPolygon_ContainPoint_OnEdge(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	poly.Project(newMercatorProjector())

	// 边上中点
	vts := poly.GetProjectedVertexes()
	midX := (vts[0][0] + vts[1][0]) / 2
	midY := (vts[0][1] + vts[1][1]) / 2
	edgeMid := projection.Point{midX, midY}

	// 默认不包含边界
	if poly.ContainPoint(edgeMid) {
		t.Error("edge point should not be inside by default")
	}

	// onEdge=true 时包含边界
	if !poly.ContainPoint(edgeMid, true) {
		t.Error("edge point should be inside when onEdge=true")
	}
}

func TestLLAPolygon_ContainPoint_WithoutProject(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	// 未投影
	if poly.ContainPoint(projection.Point{0, 0}) {
		t.Error("should return false without projection")
	}
}

// ============================================================================
// ContainPolygon 包含多边形测试
// ============================================================================
func TestLLAPolygon_ContainPolygon_FullyContained(t *testing.T) {
	// 外层大多边形
	outer, _ := NewLLAPolygon([]geography.LLAPoint{
		lla(116.395, 39.906, 0),
		lla(116.400, 39.906, 0),
		lla(116.400, 39.911, 0),
		lla(116.395, 39.911, 0),
	}, geography.CalculateHaversineDistance)

	// 内层小多边形
	inner, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	projector := newMercatorProjector()
	outer.Project(projector)
	inner.Project(projector)

	ratio := outer.ContainPolygon(inner)
	if ratio < 0.99 {
		t.Errorf("containment ratio = %f, want 1.0", ratio)
	}
}

func TestLLAPolygon_ContainPolygon_NotContained(t *testing.T) {
	outer, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	inner, _ := NewLLAPolygon([]geography.LLAPoint{
		lla(120, 40, 0),
		lla(120.001, 40, 0),
		lla(120.001, 40.001, 0),
	}, geography.CalculateHaversineDistance)

	projector := newMercatorProjector()
	outer.Project(projector)
	inner.Project(projector)

	ratio := outer.ContainPolygon(inner)
	if ratio > geography.BASIC_EPSILON {
		t.Errorf("containment ratio = %f, want 0", ratio)
	}
}

func TestLLAPolygon_ContainPolygon_PartiallyContained(t *testing.T) {
	outer, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	// 部分在内部、部分在外部的多边形
	inner, _ := NewLLAPolygon([]geography.LLAPoint{
		lla(116.3975, 39.9085, 0),
		lla(116.3985, 39.9085, 0),
		lla(116.3985, 39.9095, 0),
	}, geography.CalculateHaversineDistance)

	projector := newMercatorProjector()
	outer.Project(projector)
	inner.Project(projector)

	ratio := outer.ContainPolygon(inner)
	if ratio <= 0 || ratio >= 1 {
		t.Errorf("containment ratio = %f, want between 0 and 1", ratio)
	}
}

func TestLLAPolygon_ContainPolygon_WithoutProject(t *testing.T) {
	outer, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	inner, _ := NewLLAPolygon(makeTriangleVertexes(), geography.CalculateHaversineDistance)

	ratio := outer.ContainPolygon(inner)
	if ratio != 0 {
		t.Errorf("containment ratio = %f, want 0 without projection", ratio)
	}
}

// ============================================================================
// GetProjectedVertexes 投影顶点测试
// ============================================================================
func TestLLAPolygon_GetProjectedVertexes(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	// 未投影时应返回 nil
	if vts := poly.GetProjectedVertexes(); vts != nil {
		t.Error("GetProjectedVertexes should return nil before projection")
	}

	poly.Project(newMercatorProjector())

	vts := poly.GetProjectedVertexes()
	if len(vts) != 4 {
		t.Errorf("projected vertex count = %d, want 4", len(vts))
	}

	// 应为副本
	vts[0] = projection.Point{0, 0}
	if pvts := poly.GetProjectedVertexes(); pvts[0][0] == 0 {
		t.Error("GetProjectedVertexes should return a copy")
	}
}

// ============================================================================
// IsPointInPolygonByRay 射线法独立函数测试
// ============================================================================
// makeProjectedSquarePoly 创建一个已投影的正方形多边形供射线法测试使用
func makeProjectedSquarePoly(t *testing.T) Polygon {
	t.Helper()
	poly, err := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("create polygon: %v", err)
	}
	if err := poly.Project(newMercatorProjector()); err != nil {
		t.Fatalf("project polygon: %v", err)
	}
	return poly
}

// makeProjectedTrianglePoly 创建一个已投影的三角形多边形
func makeProjectedTrianglePoly(t *testing.T) Polygon {
	t.Helper()
	poly, err := NewLLAPolygon(makeTriangleVertexes(), geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("create polygon: %v", err)
	}
	if err := poly.Project(newMercatorProjector()); err != nil {
		t.Fatalf("project polygon: %v", err)
	}
	return poly
}

// ============================================================================
// 边界与极端情况
// ============================================================================
func TestLLAPolygon_String(t *testing.T) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	s := poly.String()
	if s == "" {
		t.Error("String should not be empty")
	}
	if s[0] != '[' || s[len(s)-1] != ']' {
		t.Errorf("String format wrong: %s", s)
	}
}

func TestLLAPolygon_ZeroAreaDegenerate(t *testing.T) {
	// 三点共线应退化
	vts := []geography.LLAPoint{
		lla(116.397, 39.908, 0),
		lla(116.398, 39.908, 0),
		lla(116.399, 39.908, 0),
	}

	poly, err := NewLLAPolygon(vts, geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := poly.Project(newMercatorProjector()); err != nil {
		t.Fatalf("project error: %v", err)
	}

	// 退化三角形面积应接近 0
	if poly.GetProjectedArea() > 1 {
		t.Errorf("degenerate triangle area = %f, want ~0", poly.GetProjectedArea())
	}
}

// ============================================================================
// Vincenty 与 Haversine 一致性
// ============================================================================
func TestLLAPolygon_VincentyAndHaversinePerimetersClose(t *testing.T) {
	vts := makeSquareVertexes()

	polyV, _ := NewLLAPolygon(vts, geography.CalculateVincentyDistance)
	polyH, _ := NewLLAPolygon(vts, geography.CalculateHaversineDistance)

	pV := polyV.GetPerimeter()
	pH := polyH.GetPerimeter()
	t.Logf("Vincenty perimeter: %f, Haversine perimeter: %f", pV, pH)

	diff := math.Abs(pV-pH) / pV
	if diff > 0.005 { // 0.5% 以内
		t.Errorf("Vincenty=%f, Haversine=%f, relative diff=%f too large", pV, pH, diff)
	}
}

// ============================================================================
// 表驱动：多边形创建与属性
// ============================================================================
func TestNewLLAPolygon_VertexOrder(t *testing.T) {
	vts := makeSquareVertexes()
	poly, _ := NewLLAPolygon(vts, geography.CalculateHaversineDistance)

	// 验证顶点顺序
	for i := 0; i < 4; i++ {
		v, ok := poly.GetVertexAt(i)
		if !ok {
			t.Fatalf("vertex %d not found", i)
		}
		lon, lat, _ := v.GetValues()
		expLon, expLat, _ := vts[i].GetValues()
		if math.Abs(lon-expLon) > geography.BASIC_EPSILON || math.Abs(lat-expLat) > geography.BASIC_EPSILON {
			t.Errorf("vertex[%d] = (%f,%f), want (%f,%f)", i, lon, lat, expLon, expLat)
		}
	}
}

// ============================================================================
// Interface compliance
// ============================================================================
func TestLLAPolygon_ImplementsPolygon(t *testing.T) {
	var _ Polygon = (*LLAPolygon)(nil)
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)

	// 确保所有方法均可调用
	_ = poly.GetProjectedArea()
	_ = poly.GetProjectedCentroid()
	_ = poly.GetPerimeter()
	_ = poly.GetProjectedPerimeter()
	_ = poly.GetEdges()
	_, _ = poly.GetEdgeAt(0)
	_ = poly.GetEdgeNumber()
	_ = poly.GetVertexes()
	_, _ = poly.GetVertexAt(0)
	_ = poly.GetVertexNumber()
	_ = poly.GetProjectedVertexes()
	_ = poly.GetDatum()
	_ = poly.ContainPoint(projection.Point{})
	_ = poly.ContainPolygon(poly)
	_ = poly.String()
	_ = poly.Project(newMercatorProjector())
}

// ============================================================================
// Benchmark
// ============================================================================
func BenchmarkNewLLAPolygon(b *testing.B) {
	vts := makeSquareVertexes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewLLAPolygon(vts, geography.CalculateHaversineDistance)
	}
}

func BenchmarkLLAPolygon_Project(b *testing.B) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	projector := newMercatorProjector()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poly.Project(projector)
	}
}

func BenchmarkLLAPolygon_ContainPoint(b *testing.B) {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	poly.Project(newMercatorProjector())
	centroid := poly.GetProjectedCentroid()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poly.ContainPoint(centroid)
	}
}

// ============================================================================
// Example
// ============================================================================
func ExampleNewLLAPolygon() {
	vts := []geography.LLAPoint{
		lla(116.397, 39.908, 0),
		lla(116.398, 39.908, 0),
		lla(116.398, 39.909, 0),
		lla(116.397, 39.909, 0),
	}

	poly, _ := NewLLAPolygon(vts, geography.CalculateHaversineDistance)
	fmt.Printf("Vertices: %d, Edges: %d, Perimeter: %.0f m\n",
		poly.GetVertexNumber(), poly.GetEdgeNumber(), poly.GetPerimeter())

	// Output:
	// Vertices: 4, Edges: 4, Perimeter: 393 m
}

func ExampleLLAPolygon_ContainPoint() {
	poly, _ := NewLLAPolygon(makeSquareVertexes(), geography.CalculateHaversineDistance)
	poly.Project(newMercatorProjector())

	centroid := poly.GetProjectedCentroid()
	fmt.Println(poly.ContainPoint(centroid))

	farPoint, _ := newMercatorProjector().Project(lla(120, 40, 0))
	fmt.Println(poly.ContainPoint(farPoint))

	// Output:
	// true
	// false
}

// ============================================================================
// 辅助构造
// ============================================================================
// 创建测试用的 WGS84 经纬高坐标点
func lla(lon, lat, alt float64) geography.LLAPoint {
	return geography.NewLLAPoint(datum.WGS84, lon, lat, alt)
}

// 创建一组顶点构成一个简单正方形（约 100m × 100m 在北京附近）
func makeSquareVertexes() []geography.LLAPoint {
	return []geography.LLAPoint{
		lla(116.397, 39.908, 0), // 左下
		lla(116.398, 39.908, 0), // 右下
		lla(116.398, 39.909, 0), // 右上
		lla(116.397, 39.909, 0), // 左上
	}
}

// 创建三角形顶点
func makeTriangleVertexes() []geography.LLAPoint {
	return []geography.LLAPoint{
		lla(116.397, 39.908, 0),
		lla(116.399, 39.908, 0),
		lla(116.398, 39.910, 0),
	}
}

// 新建投影器
func newMercatorProjector() projection.Projector {
	return projection.NewWebMercatorProjector(datum.WGS84)
}
