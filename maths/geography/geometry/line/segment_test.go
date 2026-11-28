package line

import (
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

func TestNewLLASegment(t *testing.T) {
	// 116.397, 39.9075   ===>   121.4737, 31.2304
	p1 := geography.NewLLAPoint(datum.WGS84, 116.3972, 39.9075, 0)
	p2 := geography.NewLLAPoint(datum.WGS84, 121.4737, 31.2304, 0)
	seg, err := NewLLASegment(p1, p2, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLASegment failed: %v", err)
	}

	// 检查长度和方位角（参考已知值）
	length := seg.GetLength()
	_, startAz := seg.GetStartPoint()
	_, endAz := seg.GetEndPoint()

	// 北京到上海的理论值（Vincenty）
	expectedLength := 1066569.41 // 距离 ≈ 1066569.40 米
	expectedStartAz := 152.94    // 起点方位角 ≈ 152.94 度
	expectedEndAz := 155.90      // 终点方位角 ≈ 155.90 度

	if math.Abs(length-expectedLength) > 10 {
		t.Errorf("Length = %f, expected ~%f", length, expectedLength)
	} else if math.Abs(startAz-expectedStartAz) > 0.1 {
		t.Errorf("StartAz = %f, expected ~%f", startAz, expectedStartAz)
	} else if math.Abs(endAz-expectedEndAz) > 0.1 {
		t.Errorf("EndAz = %f, expected ~%f", endAz, expectedEndAz)
	}

	// 检查投影坐标
	projector := projection.NewLambertCylindricalEqualAreaProjector(datum.WGS84)
	if err := seg.Project(projector); err != nil {
		t.Fatal(err)
	}

	projPoints := seg.GetProjectedPoints()
	x1, _ := projPoints[0].GetValues()
	x2, _ := projPoints[1].GetValues()
	// 简单地验证投影后 x 坐标递增（经度增加）
	if x2 <= x1 {
		t.Errorf("Projection x should increase: %f -> %f", x1, x2)
	}
}

func TestNewLLASegment_MismatchEllipsoid(t *testing.T) {
	p1 := geography.NewLLAPoint(datum.WGS84, 116.3972, 39.9075, 0)
	p2 := geography.NewLLAPoint(datum.CGCS2000, 121.4737, 31.2304, 0)
	_, err := NewLLASegment(p1, p2, geography.CalculateVincentyDistance)
	if err == nil || err.Error() != geography.ErrDatumNotMatched.Error() {
		t.Errorf("Expected ellipsoid mismatch error, got %v", err)
	}
}

func TestNewLLASegments(t *testing.T) {
	points := []geography.LLAPoint{
		geography.NewLLAPoint(datum.WGS84, 0, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 0, 0),
		geography.NewLLAPoint(datum.WGS84, 1, 1, 0),
	}
	segments, err := NewLLASegments(points, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLASegments failed: %v", err)
	} else if len(segments) != 2 {
		t.Errorf("Expected 2 segments, got %d", len(segments))
	}

	// 测试点数不足
	_, err = NewLLASegments(points[:1], geography.CalculateVincentyDistance)
	if err == nil {
		t.Errorf("Expected ErrLineLessThan2Coordinates, got %v", err)
	}
}

func TestGeoPoint_CalculateSideValue(t *testing.T) {
	// 定义线段：从 (0,0) 到 (1,0) 的赤道线段（经度 0->1）
	p1 := geography.NewLLAPoint(datum.WGS84, 0, 0, 0)
	p2 := geography.NewLLAPoint(datum.WGS84, 1, 0, 0)
	seg, err := NewLLASegment(p1, p2, geography.CalculateVincentyDistance)
	if err != nil {
		t.Fatalf("NewLLASegment failed: %v", err)
	}

	projector := projection.NewWebMercatorProjector(datum.WGS84)
	if err = seg.Project(projector); err != nil {
		t.Fatal(err)
	}

	// 左侧点：纬度正方向（北） (0.5, 0.01)
	leftLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0.01, 0)
	leftProj, err := projector.Project(leftLLA)
	if err != nil {
		t.Fatal(err)
	}
	cross := seg.CalculateSideValue(leftProj)
	if cross <= 0 {
		t.Errorf("Left point should give positive cross, got %f", cross)
	}

	// 右侧点：纬度负方向（南） (0.5, -0.01)
	rightLLA := geography.NewLLAPoint(datum.WGS84, 0.5, -0.01, 0)
	rightProj, err := projector.Project(rightLLA)
	if err != nil {
		t.Fatal(err)
	}
	cross = seg.CalculateSideValue(rightProj)
	if cross >= 0 {
		t.Errorf("Right point should give negative cross, got %f", cross)
	}

	// 线上点： (0.5, 0)
	onLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0, 0)
	onProj, err := projector.Project(onLLA)
	if err != nil {
		t.Fatal(err)
	}
	cross = seg.CalculateSideValue(onProj)
	if math.Abs(cross) > 1e-9 {
		t.Errorf("Point on line should give zero cross, got %f", cross)
	}
}

func TestGeoPoint_CalculateSideValueOnProjection(t *testing.T) {
	p1 := geography.NewLLAPoint(datum.WGS84, 0, 0, 0)
	p2 := geography.NewLLAPoint(datum.WGS84, 1, 0, 0)
	seg, _ := NewLLASegment(p1, p2, geography.CalculateVincentyDistance)

	// 获取投影后的点
	leftLLA := geography.NewLLAPoint(datum.WGS84, 0.5, 0.01, 0)
	projector := projection.NewWebMercatorProjector(datum.WGS84)
	leftProj, err := projector.Project(leftLLA)
	if err != nil {
		t.Fatal(err)
	}

	if err = seg.Project(projection.NewWebMercatorProjector(datum.WGS84)); err != nil {
		t.Fatal(err)
	}

	cross := seg.CalculateSideValue(leftProj)
	if cross <= 0 {
		t.Errorf("Left projection point should give positive cross, got %f", cross)
	}
}

func TestString(t *testing.T) {
	p1 := geography.NewLLAPoint(datum.WGS84, 0, 0, 0)
	p2 := geography.NewLLAPoint(datum.WGS84, 1, 1, 0)
	seg, _ := NewLLASegment(p1, p2, geography.CalculateVincentyDistance)
	if str := seg.String(); str == "" {
		t.Error("String() returned empty")
	}
	t.Log(seg)
}
