package line

import (
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// ============================================================================
// NewLLAPolyline 测试
// ============================================================================
func TestNewLLAPolyline_Basic(t *testing.T) {
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 116.3972, 39.9075, 0),
		geography.NewLLAPoint(datum.WGS84, 121.4737, 31.2304, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline failed: %v", err)
	}

	if p.GetPointNumber() != 2 {
		t.Errorf("expected 2 points, got %d", p.GetPointNumber())
	}
	if p.GetLength() <= 0 {
		t.Errorf("expected positive length, got %f", p.GetLength())
	}
}

func TestNewLLAPolyline_TooFewPoints(t *testing.T) {
	// 0 个点
	_, err := NewLLAPolyline(nil, geography.CalculateVincentyDistance)
	if err == nil {
		t.Error("expected error for nil points")
	}

	// 1 个点
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
	}
	_, err = NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err == nil {
		t.Error("expected error for single point")
	}
}

func TestNewLLAPolyline_Deduplicate(t *testing.T) {
	// 相邻重复点应被过滤
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0), // 重复
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline failed: %v", err)
	}
	if p.GetPointNumber() != 2 {
		t.Errorf("expected 2 points after dedup, got %d", p.GetPointNumber())
	}
}

func TestNewLLAPolyline_DifferentDatum(t *testing.T) {
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.CGCS2000, 1, 0, 0),
	}
	_, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err == nil {
		t.Error("expected datum mismatch error")
	}
}

func TestNewLLAPolyline_Haversine(t *testing.T) {
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateHaversineDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline with Haversine failed: %v", err)
	}
	if p.GetLength() <= 0 {
		t.Errorf("expected positive length, got %f", p.GetLength())
	}
}

// ============================================================================
// GetPoints / GetPointNumber 测试
// ============================================================================
func TestLLAPolyline_GetPoints(t *testing.T) {
	p := newTestPolyline(t)

	points := p.GetPoints()
	if len(points) != 4 {
		t.Fatalf("expected 4 points, got %d", len(points))
	}

	// 验证点顺序
	lon, lat, _ := points[0].GetValues()
	if lon != 0 || lat != 0 {
		t.Errorf("point[0] = (%f, %f), expected (0, 0)", lon, lat)
	}

	lon, lat, _ = points[3].GetValues()
	if lon != 0 || lat != 1 {
		t.Errorf("point[3] = (%f, %f), expected (0, 1)", lon, lat)
	}
}

func TestLLAPolyline_GetPointNumber(t *testing.T) {
	p := newTestPolyline(t)
	if n := p.GetPointNumber(); n != 4 {
		t.Errorf("expected 4, got %d", n)
	}
}

// ============================================================================
// GetDatum 测试
// ============================================================================
func TestLLAPolyline_GetDatum(t *testing.T) {
	p := newTestPolyline(t)
	if d := p.GetDatum(); d != datum.WGS84 {
		t.Errorf("expected WGS84, got %v", d)
	}
}

// ============================================================================
// Project 测试
// ============================================================================
func TestLLAPolyline_Project(t *testing.T) {
	p := newTestPolyline(t)
	proj := projection.NewWebMercatorProjector(datum.WGS84)

	// 未投影时投影长度应为 0
	if pl := p.GetProjectedLength(); pl != 0 {
		t.Errorf("projected length before projection should be 0, got %f", pl)
	}

	if err := p.Project(proj); err != nil {
		t.Fatalf("project failed: %v", err)
	}

	// 投影后投影长度应 > 0
	pl := p.GetProjectedLength()
	if pl <= 0 {
		t.Errorf("projected length after projection should be > 0, got %f", pl)
	}

	// 投影点数量应 = 点数
	projPoints := p.GetProjectedPoints()
	if len(projPoints) != 4 {
		t.Errorf("expected 4 projected points, got %d", len(projPoints))
	}
}

func TestLLAPolyline_Project_Lambert(t *testing.T) {
	p := newTestPolyline(t)
	proj := projection.NewLambertCylindricalEqualAreaProjector(datum.WGS84)

	if err := p.Project(proj); err != nil {
		t.Fatalf("lambert project failed: %v", err)
	}

	points := p.GetProjectedPoints()
	if len(points) != 4 {
		t.Errorf("expected 4 projected points, got %d", len(points))
	}
}

func TestLLAPolyline_GetProjectedPoints_NotProjected(t *testing.T) {
	p := newTestPolyline(t)
	points := p.GetProjectedPoints()
	if points != nil {
		t.Errorf("expected nil for not-projected polyline, got %v", points)
	}
}

// ============================================================================
// Insert 测试
// ============================================================================
func TestLLAPolyline_Insert_Append(t *testing.T) {
	p := newTestPolyline(t)
	oldLen := p.GetLength()

	newPoint := geography.NewLLAPoint(datum.WGS84, -1, 1, 0)
	err := p.Insert(newPoint)
	if err != nil {
		t.Fatalf("insert at end failed: %v", err)
	}

	if p.GetPointNumber() != 5 {
		t.Errorf("expected 5 points, got %d", p.GetPointNumber())
	}
	if p.GetLength() <= oldLen {
		t.Errorf("length should increase after append")
	}

	// 验证最后一个点
	lastPoint, ok := p.GetPointAt(4)
	if !ok {
		t.Fatal("failed to get last point")
	}
	if !lastPoint.IsEqualTo(newPoint) {
		t.Errorf("last point mismatch: got %v, expected %v", lastPoint, newPoint)
	}
}

func TestLLAPolyline_Insert_Prepend(t *testing.T) {
	p := newTestPolyline(t)
	oldLen := p.GetLength()

	newPoint := geography.NewLLAPoint(datum.WGS84, -1, 0, 0)
	err := p.Insert(newPoint, 0)
	if err != nil {
		t.Fatalf("insert at head failed: %v", err)
	}

	if p.GetPointNumber() != 5 {
		t.Errorf("expected 5 points, got %d", p.GetPointNumber())
	}
	if p.GetLength() <= oldLen {
		t.Errorf("length should increase after prepend")
	}

	// 验证第一个点
	firstPoint, ok := p.GetPointAt(0)
	if !ok {
		t.Fatal("failed to get first point")
	}
	if !firstPoint.IsEqualTo(newPoint) {
		t.Errorf("first point mismatch: got %v, expected %v", firstPoint, newPoint)
	}
}

func TestLLAPolyline_Insert_Middle(t *testing.T) {
	p := newTestPolyline(t)

	// 在索引 1 处插入（i.e., 原 P0→新点→P1→P2→P3）
	newPoint := geography.NewLLAPoint(datum.WGS84, 0.5, 0, 0)
	err := p.Insert(newPoint, 1)
	if err != nil {
		t.Fatalf("insert in middle failed: %v", err)
	}

	if p.GetPointNumber() != 5 {
		t.Errorf("expected 5 points, got %d", p.GetPointNumber())
	}

	// 验证插入点位置
	inserted, ok := p.GetPointAt(1)
	if !ok {
		t.Fatal("failed to get inserted point")
	}
	if !inserted.IsEqualTo(newPoint) {
		t.Errorf("inserted point mismatch: got %v, expected %v", inserted, newPoint)
	}

	// 验证插入后的点顺序: P0→新点→P1→P2→P3
	p0, _ := p.GetPointAt(0)
	p1, _ := p.GetPointAt(1)
	p2, _ := p.GetPointAt(2)
	p3, _ := p.GetPointAt(3)
	p4, _ := p.GetPointAt(4)

	exp0 := geography.NewLLAPoint(datum.WGS84, 0, 0, 0)
	exp2 := geography.NewLLAPoint(datum.WGS84, 1, 0, 0)
	exp3 := geography.NewLLAPoint(datum.WGS84, 1, 1, 0)
	exp4 := geography.NewLLAPoint(datum.WGS84, 0, 1, 0)

	if !p0.IsEqualTo(exp0) || !p1.IsEqualTo(newPoint) || !p2.IsEqualTo(exp2) ||
		!p3.IsEqualTo(exp3) || !p4.IsEqualTo(exp4) {
		t.Error("point order after middle insert is incorrect")
	}
}

func TestLLAPolyline_Insert_OutOfBounds(t *testing.T) {
	p := newTestPolyline(t)

	err := p.Insert(geography.NewLLAPoint(datum.WGS84, 0, 0, 0), 100)
	if err == nil {
		t.Error("expected error for out-of-bounds insert")
	}
}

func TestLLAPolyline_Insert_InvalidatesProjection(t *testing.T) {
	p := newTestPolylineProjected(t)

	if p.GetProjectedLength() <= 0 {
		t.Fatal("expected positive projected length before insert")
	}

	err := p.Insert(geography.NewLLAPoint(datum.WGS84, 0.5, 0.5, 0), 2)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// 插入后投影状态应被重置
	if p.GetProjectedLength() != 0 {
		t.Errorf("projected length should be 0 after insert, got %f", p.GetProjectedLength())
	}
	if pts := p.GetProjectedPoints(); pts != nil {
		t.Errorf("projected points should be nil after insert, got %v", pts)
	}
}

// ============================================================================
// Delete 测试
// ============================================================================
func TestLLAPolyline_Delete_First(t *testing.T) {
	p := newTestPolyline(t)
	// 点数: 4, 段数: 3, 原始长度: sum of 3 segments
	nBefore := p.GetPointNumber()
	lenBefore := p.GetLength()

	p.Delete(0)

	if p.GetPointNumber() != nBefore-1 {
		t.Errorf("expected %d points, got %d", nBefore-1, p.GetPointNumber())
	}
	if p.GetLength() >= lenBefore {
		t.Errorf("length should decrease after deleting first point")
	}

	// 新首点应该为原 P1(1,0)
	first, _ := p.GetPointAt(0)
	exp := geography.NewLLAPoint(datum.WGS84, 1, 0, 0)
	if !first.IsEqualTo(exp) {
		t.Errorf("new first point = %v, expected %v", first, exp)
	}
}

func TestLLAPolyline_Delete_Last(t *testing.T) {
	p := newTestPolyline(t)
	nBefore := p.GetPointNumber()
	lenBefore := p.GetLength()

	p.Delete(nBefore - 1)

	if p.GetPointNumber() != nBefore-1 {
		t.Errorf("expected %d points, got %d", nBefore-1, p.GetPointNumber())
	}
	if p.GetLength() >= lenBefore {
		t.Errorf("length should decrease after deleting last point")
	}

	// 新尾点应该为原 P2(1,1)
	last, _ := p.GetPointAt(p.GetPointNumber() - 1)
	exp := geography.NewLLAPoint(datum.WGS84, 1, 1, 0)
	if !last.IsEqualTo(exp) {
		t.Errorf("new last point = %v, expected %v", last, exp)
	}
}

func TestLLAPolyline_Delete_Middle(t *testing.T) {
	p := newTestPolyline(t)
	// 删除 P1(1,0)，应合并为 P0→P2，新点序列: P0(0,0)→P2(1,1)→P3(0,1)
	p.Delete(1)

	if p.GetPointNumber() != 3 {
		t.Errorf("expected 3 points, got %d", p.GetPointNumber())
	}

	// 验证新点序列
	p0, _ := p.GetPointAt(0)
	p1, _ := p.GetPointAt(1)
	p2, _ := p.GetPointAt(2)

	exp0 := geography.NewLLAPoint(datum.WGS84, 0, 0, 0)
	exp1 := geography.NewLLAPoint(datum.WGS84, 1, 1, 0)
	exp2 := geography.NewLLAPoint(datum.WGS84, 0, 1, 0)

	if !p0.IsEqualTo(exp0) || !p1.IsEqualTo(exp1) || !p2.IsEqualTo(exp2) {
		t.Error("point sequence after middle delete is incorrect")
	}

	// 长度应合理: 新段 P0→P2 的椭圆距离
	if p.GetLength() <= 0 {
		t.Errorf("length should be positive after delete, got %f", p.GetLength())
	}
}

func TestLLAPolyline_Delete_TooFewPoints(t *testing.T) {
	// 创建只有 2 个点的折线
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline failed: %v", err)
	}

	// 不应允许删除（会导致 < 2 点）
	p.Delete(0)
	if p.GetPointNumber() != 2 {
		t.Errorf("should still have 2 points after denied delete, got %d", p.GetPointNumber())
	}
}

func TestLLAPolyline_Delete_OutOfBounds(t *testing.T) {
	p := newTestPolyline(t)
	n := p.GetPointNumber()

	// 负索引
	p.Delete(-1)
	if p.GetPointNumber() != n {
		t.Errorf("point count changed after invalid delete")
	}

	// 越界索引
	p.Delete(n)
	if p.GetPointNumber() != n {
		t.Errorf("point count changed after invalid delete")
	}
}

func TestLLAPolyline_Delete_InvalidatesProjection(t *testing.T) {
	p := newTestPolylineProjected(t)

	if p.GetProjectedLength() <= 0 {
		t.Fatal("expected positive projected length before delete")
	}

	p.Delete(1)

	// 删除后投影状态应重置
	if p.GetProjectedLength() != 0 {
		t.Errorf("projected length should be 0 after delete, got %f", p.GetProjectedLength())
	}
}

// ============================================================================
// Insert + Delete 组合测试
// ============================================================================
func TestLLAPolyline_InsertDelete_RoundTrip(t *testing.T) {
	p := newTestPolyline(t)
	originalLength := p.GetLength()
	originalN := p.GetPointNumber()

	// 插入再删除同一个位置，应回到原始状态
	middlePoint := geography.NewLLAPoint(datum.WGS84, 0.5, 0.5, 0)
	if err := p.Insert(middlePoint, 2); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	p.Delete(2)

	if p.GetPointNumber() != originalN {
		t.Errorf("point count: expected %d, got %d", originalN, p.GetPointNumber())
	}

	// 长度应近似恢复（删除中间点时合并段会产生新段，由于大地线不是简单的加法减法，可能有微小误差）
	lengthDiff := math.Abs(p.GetLength() - originalLength)
	// 允许 0.1% 的精度损失
	if lengthDiff > originalLength*0.001 {
		t.Errorf("length not restored: original=%f, after=%f, diff=%f", originalLength, p.GetLength(), lengthDiff)
	}
}

// ============================================================================
// CalculateSideValue 测试
// ============================================================================
func TestLLAPolyline_CalculateSideValue_LeftPoint(t *testing.T) {
	p := newTestPolylineProjected(t)

	// 取折线中间上方（北）的点 — 应该在左侧（取决于多段线走向）
	proj := projection.NewWebMercatorProjector(datum.WGS84)
	leftLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0.6, 0) // P0_P1 中点偏北
	leftProj, err := proj.Project(leftLLA)
	if err != nil {
		t.Fatalf("project test point failed: %v", err)
	}

	result := p.CalculateSideValue(leftProj)
	// 对于从 (0,0)→(1,0)→(1,1)→(0,1) 的路径，(0.5, 0.6) 在所有段的上方
	if result == 0 {
		t.Error("expected non-zero (left/right), got 0")
	}
}

func TestLLAPolyline_CalculateSideValue_OnLine(t *testing.T) {
	p := newTestPolylineProjected(t)

	// 线段中点
	onPointLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0, 0)
	proj := projection.NewWebMercatorProjector(datum.WGS84)
	onPoint, err := proj.Project(onPointLLA)
	if err != nil {
		t.Fatalf("project test point failed: %v", err)
	}

	result := p.CalculateSideValue(onPoint)
	// 在线上的点应返回 0（容差范围内）
	if result != 0 {
		t.Errorf("point on line should return 0, got %f", result)
	}
}

func TestLLAPolyline_CalculateSideValue_NotProjected(t *testing.T) {
	p := newTestPolyline(t)

	result := p.CalculateSideValue(projection.Point{100, 100})
	if result != 0 {
		t.Errorf("not-projected polyline should return 0, got %f", result)
	}
}

// ============================================================================
// GetPointAt 测试
// ============================================================================
func TestLLAPolyline_GePointAt(t *testing.T) {
	p := newTestPolyline(t)

	// 第一个点
	pt, ok := p.GetPointAt(0)
	if !ok {
		t.Fatal("GetPointAt(0) failed")
	}
	lon, lat, _ := pt.GetValues()
	if lon != 0 || lat != 0 {
		t.Errorf("point[0] = (%f, %f), expected (0, 0)", lon, lat)
	}

	// 最后一个点
	pt, ok = p.GetPointAt(3)
	if !ok {
		t.Fatal("GetPointAt(3) failed")
	}
	lon, lat, _ = pt.GetValues()
	if lon != 0 || lat != 1 {
		t.Errorf("point[3] = (%f, %f), expected (0, 1)", lon, lat)
	}

	// 越界
	_, ok = p.GetPointAt(-1)
	if ok {
		t.Error("GetPointAt(-1) should return false")
	}
	_, ok = p.GetPointAt(4)
	if ok {
		t.Error("GetPointAt(4) should return false")
	}
}

// ============================================================================
// GetProjectedPointAt 测试
// ============================================================================
func TestLLAPolyline_GetProjectedPointAt(t *testing.T) {
	p := newTestPolylineProjected(t)

	pt, ok := p.GetProjectedPointAt(0)
	if !ok {
		t.Fatal("GetProjectedPointAt(0) failed")
	}
	x, y := pt.GetValues()
	if x == 0 && y == 0 {
		// WebMercator 下 (0,0) 确实投影到 (0,0)，合理
	} else {
		t.Logf("projected point[0] = (%f, %f)", x, y)
	}

	// 越界
	_, ok = p.GetProjectedPointAt(-1)
	if ok {
		t.Error("GetProjectedPointAt(-1) should return false")
	}
	_, ok = p.GetProjectedPointAt(4)
	if ok {
		t.Error("GetProjectedPointAt(4) should return false")
	}
}

// ============================================================================
// GetSegments / GetSegmentAt 测试
// ============================================================================
func TestLLAPolyline_GetSegments(t *testing.T) {
	p := newTestPolyline(t)
	segs := p.GetSegments()

	if len(segs) != 3 {
		t.Errorf("expected 3 segments, got %d", len(segs))
	}

	// 验证返回的是副本：修改返回值不应影响原 polyline
	segs[0] = nil
	segs2 := p.GetSegments()
	if segs2[0] == nil {
		t.Error("GetSegments should return a copy, but original was mutated")
	}
}

func TestLLAPolyline_GetSegmentAt(t *testing.T) {
	p := newTestPolyline(t)

	seg, ok := p.GetSegmentAt(0)
	if !ok || seg == nil {
		t.Fatal("GetSegmentAt(0) failed")
	}

	start, _ := seg.GetStartPoint()
	lon, lat, _ := start.GetValues()
	if lon != 0 || lat != 0 {
		t.Errorf("segment[0] start = (%f, %f), expected (0, 0)", lon, lat)
	}

	// 越界
	_, ok = p.GetSegmentAt(-1)
	if ok {
		t.Error("GetSegmentAt(-1) should return false")
	}
	_, ok = p.GetSegmentAt(3)
	if ok {
		t.Error("GetSegmentAt(3) should return false")
	}
}

// ============================================================================
// Start / End 点测试
// ============================================================================
func TestLLAPolyline_StartEndPoint(t *testing.T) {
	p := newTestPolyline(t)

	start, startAz := p.GetStartPoint()
	lon, lat, _ := start.GetValues()
	if lon != 0 || lat != 0 {
		t.Errorf("start point = (%f, %f), expected (0, 0)", lon, lat)
	}
	if startAz < 0 || startAz >= 360 {
		t.Errorf("start azimuth out of range: %f", startAz)
	}

	end, endAz := p.GetEndPoint()
	lon, lat, _ = end.GetValues()
	if lon != 0 || lat != 1 {
		t.Errorf("end point = (%f, %f), expected (0, 1)", lon, lat)
	}
	if endAz < 0 || endAz >= 360 {
		t.Errorf("end azimuth out of range: %f", endAz)
	}
}

func TestLLAPolyline_StartEndProjectedPoint(t *testing.T) {
	p := newTestPolylineProjected(t)

	start := p.GetStartProjectedPoint()
	sx, sy := start.GetValues()
	// WebMercator 下 (0,0) → (0,0)
	if math.Abs(sx) > 1e-9 || math.Abs(sy) > 1e-9 {
		t.Errorf("start projected point should be near (0,0), got (%f, %f)", sx, sy)
	}

	end := p.GetEndProjectedPoint()
	ex, ey := end.GetValues()
	// WebMercator 下 (0,1) → (0, ~111325)
	if math.Abs(ex) > 1e-9 || ey <= 0 {
		t.Errorf("end projected point x should be near 0, y > 0, got (%f, %f)", ex, ey)
	}
}

// ============================================================================
// String 测试
// ============================================================================
func TestLLAPolyline_String(t *testing.T) {
	p := newTestPolyline(t)
	s := p.String()
	if s == "" {
		t.Error("String() returned empty")
	}

	// 检查基本格式：应包含 JSON 标记和段分隔符
	if s[0] != '[' || s[len(s)-1] != ']' {
		t.Errorf("String() should be wrapped in [], got: %s", s)
	}
}

// ============================================================================
// 边界情况测试
// ============================================================================
func TestLLAPolyline_EmptySegments_EdgeMethods(t *testing.T) {
	// 创建一个有效但最简的折线
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline failed: %v", err)
	}

	// 所有基础方法应正常工作
	_ = p.GetLength()
	_ = p.GetDatum()
	_ = p.GetPoints()
	_ = p.GetPointNumber()
	_ = p.GetSegments()
	_ = p.String()

	start, _ := p.GetStartPoint()
	end, _ := p.GetEndPoint()
	if start.IsEqualTo(end) {
		t.Error("start and end should be different")
	}

	pt, ok := p.GetPointAt(0)
	if !ok {
		t.Error("GetPointAt(0) failed for 2-point polyline")
	}
	_ = pt
}

func TestLLAPolyline_CalculateSideValue_StraightLine(t *testing.T) {
	// 一条水平直线: (0,0) → (2,0)
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 2, 0, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLAPolyline failed: %v", err)
	}

	proj := projection.NewWebMercatorProjector(datum.WGS84)
	if err := p.Project(proj); err != nil {
		t.Fatalf("project failed: %v", err)
	}

	// 左侧（北）
	leftLLA := geography.NewLLAPoint(datum.WGS84, 1, 0.1, 0)
	leftProj, _ := proj.Project(leftLLA)
	result := p.CalculateSideValue(leftProj)
	// 对于多段线 CalculateSideValue，正数=左侧，负数=右侧
	if result <= 0 {
		t.Errorf("north point should be left (positive), got %f", result)
	}

	// 右侧（南）
	rightLLA := geography.NewLLAPoint(datum.WGS84, 1, -0.1, 0)
	rightProj, _ := proj.Project(rightLLA)
	result = p.CalculateSideValue(rightProj)
	if result >= 0 {
		t.Errorf("south point should be right (negative), got %f", result)
	}
}

// ============================================================================
// 测试辅助函数
// ============================================================================
// newTestPolyline 创建一条 4 点折线用于通用测试
// 点序列: (0,0) → (1,0) → (1,1) → (0,1)
func newTestPolyline(t *testing.T) Polyline {
	t.Helper()
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 1, 0),
		geography.NewLLAPoint(datum.WGS84, 0, 1, 0),
	}
	p, err := NewLLAPolyline(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("newTestPolyline failed: %v", err)
	}
	return p
}

// newTestPolylineProjected 创建一条已投影的 4 点折线
func newTestPolylineProjected(t *testing.T) Polyline {
	t.Helper()
	p := newTestPolyline(t)
	proj := projection.NewWebMercatorProjector(datum.WGS84)
	if err := p.Project(proj); err != nil {
		t.Fatalf("project failed: %v", err)
	}
	return p
}
