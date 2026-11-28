package geography

import (
	"math"
	"testing"

	"gitee.com/leminewx/gokit"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func generateWGS84AndVincentyTestPoints() []struct {
	Start, End   LLAPoint
	Distance     float64
	StartAzimuth float64
	EndAzimuth   float64
} {
	return []struct {
		Start, End   LLAPoint
		Distance     float64
		StartAzimuth float64
		EndAzimuth   float64
	}{
		{NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494), NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951), 88.4, 146.71, 146.71},
		{NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951), NewLLAPoint(datum.WGS84, 125.1466127, 43.82570289), 172.4, 154.17, 154.17},
		{NewLLAPoint(datum.WGS84, 125.1466127, 43.82570289), NewLLAPoint(datum.WGS84, 125.1482411, 43.82378866), 249.8, 148.37, 148.37},
		{NewLLAPoint(datum.WGS84, 125.1482411, 43.82378866), NewLLAPoint(datum.WGS84, 125.1534452, 43.82569468), 469.1, 63.16, 63.17},
		{NewLLAPoint(datum.WGS84, 125.1534452, 43.82569468), NewLLAPoint(datum.WGS84, 125.1515662, 43.82741168), 243.4, 321.61, 321.61},
		{NewLLAPoint(datum.WGS84, 125.1515662, 43.82741168), NewLLAPoint(datum.WGS84, 125.3972956, 43.92575709), 22571.2, 60.96, 61.13},
		{NewLLAPoint(datum.WGS84, 125.3972956, 43.92575709), NewLLAPoint(datum.WGS84, 125.2435197, 43.94465075), 12524.4, 279.70, 279.60},
		{NewLLAPoint(datum.WGS84, 125.2435197, 43.94465075), NewLLAPoint(datum.WGS84, 116.3972001, 39.90750001), 859217.9, 241.59, 235.67},
		{NewLLAPoint(datum.WGS84, 116.3972001, 39.90750001), NewLLAPoint(datum.WGS84, 121.4737001, 31.23040001), 1066569.4, 152.94, 155.90},
		{NewLLAPoint(datum.WGS84, 121.4737001, 31.23040001), NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494), 1434916.9, 11.97, 14.22},
	}
}

// dist ≈ {diff: 0.048, rate: 0.05%, truth:      88.400}, az1 ≈ {diff: 0.002, rate: 0.00%, truth: 146.710}, az2 ≈ {diff: 0.002, rate: 0.00%, truth: 146.710}
// dist ≈ {diff: 0.001, rate: 0.00%, truth:     172.400}, az1 ≈ {diff: 0.000, rate: 0.00%, truth: 154.170}, az2 ≈ {diff: 0.001, rate: 0.00%, truth: 154.170}
// dist ≈ {diff: 0.010, rate: 0.00%, truth:     249.800}, az1 ≈ {diff: 0.001, rate: 0.00%, truth: 148.370}, az2 ≈ {diff: 0.002, rate: 0.00%, truth: 148.370}
// dist ≈ {diff: 0.046, rate: 0.01%, truth:     469.100}, az1 ≈ {diff: 0.004, rate: 0.01%, truth:  63.160}, az2 ≈ {diff: 0.002, rate: 0.00%, truth:  63.170}
// dist ≈ {diff: 0.008, rate: 0.00%, truth:     243.400}, az1 ≈ {diff: 0.002, rate: 0.00%, truth: 321.610}, az2 ≈ {diff: 0.000, rate: 0.00%, truth: 321.610}
// dist ≈ {diff: 0.035, rate: 0.00%, truth:   22571.200}, az1 ≈ {diff: 0.000, rate: 0.00%, truth:  60.960}, az2 ≈ {diff: 0.000, rate: 0.00%, truth:  61.130}
// dist ≈ {diff: 0.016, rate: 0.00%, truth:   12524.400}, az1 ≈ {diff: 0.003, rate: 0.00%, truth: 279.700}, az2 ≈ {diff: 0.004, rate: 0.00%, truth: 279.600}
// dist ≈ {diff: 0.024, rate: 0.00%, truth:  859217.900}, az1 ≈ {diff: 0.003, rate: 0.00%, truth: 241.590}, az2 ≈ {diff: 0.004, rate: 0.00%, truth: 235.670}
// dist ≈ {diff: 0.020, rate: 0.00%, truth: 1066569.400}, az1 ≈ {diff: 0.005, rate: 0.00%, truth: 152.940}, az2 ≈ {diff: 0.002, rate: 0.00%, truth: 155.900}
// dist ≈ {diff: 0.007, rate: 0.00%, truth: 1434916.900}, az1 ≈ {diff: 0.001, rate: 0.01%, truth:  11.970}, az2 ≈ {diff: 0.001, rate: 0.01%, truth:  14.220}
func TestCalculateWGS84VincentyDistance(t *testing.T) {
	segments := generateWGS84AndVincentyTestPoints()
	for _, seg := range segments {
		dist, az1, az2, err := CalculateVincentyDistance(seg.Start, seg.End)
		if err != nil {
			t.Fatalf("Vincenty error: %v", err)
		}

		if math.Abs((dist-seg.Distance)/seg.Distance) > 0.001 {
			t.Errorf("Distance = %f, expected ~%f", dist, seg.Distance)
		}
		if math.Abs(az1-seg.StartAzimuth) > 1 {
			t.Errorf("Start azimuth = %f, expected ~%f", az1, seg.StartAzimuth)
		}
		if math.Abs(az2-seg.EndAzimuth) > 1 {
			t.Errorf("End azimuth = %f, expected ~%f", az2, seg.EndAzimuth)
		}

		t.Logf("dist ≈ {diff: %.3f, rate: %.2f%%, truth: %11.3f}, az1 ≈ {diff: %.3f, rate: %.2f%%, truth: %7.3f}, az2 ≈ {diff: %.3f, rate: %.2f%%, truth: %7.3f}",
			math.Abs(seg.Distance-dist), math.Abs(seg.Distance-dist)/seg.Distance*100, seg.Distance,
			math.Abs(seg.StartAzimuth-az1), math.Abs(seg.StartAzimuth-az1)/seg.StartAzimuth*100, seg.StartAzimuth,
			math.Abs(seg.EndAzimuth-az2), math.Abs(seg.EndAzimuth-az2)/seg.EndAzimuth*100, seg.EndAzimuth)
	}
}

func generateWGS84AndHaversineTestPoints() []struct {
	Start, End   LLAPoint
	Distance     float64
	StartAzimuth float64
	EndAzimuth   float64
} {
	return []struct {
		Start, End   LLAPoint
		Distance     float64
		StartAzimuth float64
		EndAzimuth   float64
	}{
		{NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494), NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951), 88.4, 146.80, 146.80},
		{NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951), NewLLAPoint(datum.WGS84, 125.1466127, 43.82570289), 172.4, 154.25, 154.25},
		{NewLLAPoint(datum.WGS84, 125.1466127, 43.82570289), NewLLAPoint(datum.WGS84, 125.1482411, 43.82378866), 249.7, 148.46, 148.46},
		{NewLLAPoint(datum.WGS84, 125.1482411, 43.82378866), NewLLAPoint(datum.WGS84, 125.1534452, 43.82569468), 468.2, 63.08, 63.09},
		{NewLLAPoint(datum.WGS84, 125.1534452, 43.82569468), NewLLAPoint(datum.WGS84, 125.1515662, 43.82741168), 243.3, 321.71, 321.71},
		{NewLLAPoint(datum.WGS84, 125.1515662, 43.82741168), NewLLAPoint(datum.WGS84, 125.3972956, 43.92575709), 22528.2, 60.88, 61.05},
		{NewLLAPoint(datum.WGS84, 125.3972956, 43.92575709), NewLLAPoint(datum.WGS84, 125.2435197, 43.94465075), 12491.4, 279.74, 279.63},
		{NewLLAPoint(datum.WGS84, 125.2435197, 43.94465075), NewLLAPoint(datum.WGS84, 116.3972001, 39.90750001), 857846.6, 241.49, 235.57},
		{NewLLAPoint(datum.WGS84, 116.3972001, 39.90750001), NewLLAPoint(datum.WGS84, 121.4737001, 31.23040001), 1068033.0, 153.03, 156.00},
		{NewLLAPoint(datum.WGS84, 121.4737001, 31.23040001), NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494), 1437279.1, 11.92, 14.17},
	}
}

// dist ≈ {diff:    0.122, rate: 0.14%, truth:      88.400}, az1 ≈ {diff: 0.004, rate: 0.00%, truth: 146.800}, az2 ≈ {diff: 0.004, rate: 0.00%, truth: 146.800}
// dist ≈ {diff:    0.213, rate: 0.12%, truth:     172.400}, az1 ≈ {diff: 0.001, rate: 0.00%, truth: 154.250}, az2 ≈ {diff: 0.001, rate: 0.00%, truth: 154.250}
// dist ≈ {diff:    0.323, rate: 0.13%, truth:     249.700}, az1 ≈ {diff: 0.001, rate: 0.00%, truth: 148.460}, az2 ≈ {diff: 0.002, rate: 0.00%, truth: 148.460}
// dist ≈ {diff:    0.528, rate: 0.11%, truth:     468.200}, az1 ≈ {diff: 0.003, rate: 0.01%, truth:  63.080}, az2 ≈ {diff: 0.003, rate: 0.00%, truth:  63.090}
// dist ≈ {diff:    0.225, rate: 0.09%, truth:     243.300}, az1 ≈ {diff: 0.001, rate: 0.00%, truth: 321.710}, az2 ≈ {diff: 0.002, rate: 0.00%, truth: 321.710}
// dist ≈ {diff:   25.162, rate: 0.11%, truth:   22528.200}, az1 ≈ {diff: 0.005, rate: 0.01%, truth:  60.880}, az2 ≈ {diff: 0.005, rate: 0.01%, truth:  61.050}
// dist ≈ {diff:   14.016, rate: 0.11%, truth:   12491.400}, az1 ≈ {diff: 0.004, rate: 0.00%, truth: 279.740}, az2 ≈ {diff: 0.001, rate: 0.00%, truth: 279.630}
// dist ≈ {diff:  959.817, rate: 0.11%, truth:  857846.600}, az1 ≈ {diff: 0.002, rate: 0.00%, truth: 241.490}, az2 ≈ {diff: 0.001, rate: 0.00%, truth: 235.570}
// dist ≈ {diff: 1194.986, rate: 0.11%, truth: 1068033.000}, az1 ≈ {diff: 0.004, rate: 0.00%, truth: 153.030}, az2 ≈ {diff: 0.004, rate: 0.00%, truth: 156.000}
// dist ≈ {diff: 1608.123, rate: 0.11%, truth: 1437279.100}, az1 ≈ {diff: 0.002, rate: 0.01%, truth:  11.920}, az2 ≈ {diff: 0.001, rate: 0.01%, truth:  14.170}
func TestCalculateWGS84HaversineDistance(t *testing.T) {
	segments := generateWGS84AndHaversineTestPoints()
	for _, seg := range segments {
		dist, az1, az2, err := CalculateHaversineDistance(seg.Start, seg.End)
		if err != nil {
			t.Fatalf("Vincenty error: %v", err)
		}

		if math.Abs((dist-seg.Distance)/seg.Distance) > 0.002 {
			t.Errorf("Distance = %f, expected ~%f", dist, seg.Distance)
		}
		if math.Abs(az1-seg.StartAzimuth) > 1 {
			t.Errorf("Start azimuth = %f, expected ~%f", az1, seg.StartAzimuth)
		}
		if math.Abs(az2-seg.EndAzimuth) > 1 {
			t.Errorf("End azimuth = %f, expected ~%f", az2, seg.EndAzimuth)
		}

		t.Logf("dist ≈ {diff: %8.3f, rate: %.2f%%, truth: %11.3f}, az1 ≈ {diff: %.3f, rate: %.2f%%, truth: %7.3f}, az2 ≈ {diff: %.3f, rate: %.2f%%, truth: %7.3f}",
			math.Abs(seg.Distance-dist), math.Abs(seg.Distance-dist)/seg.Distance*100, seg.Distance,
			math.Abs(seg.StartAzimuth-az1), math.Abs(seg.StartAzimuth-az1)/seg.StartAzimuth*100, seg.StartAzimuth,
			math.Abs(seg.EndAzimuth-az2), math.Abs(seg.EndAzimuth-az2)/seg.EndAzimuth*100, seg.EndAzimuth)
	}
}

// 测试 Vincenty 周长计算
func TestCalculatePolygonVincentyPerimeter(t *testing.T) {
	// 构造一个三角形（北京、上海、长春）
	p1 := NewLLAPoint(datum.WGS84, 116.3972, 39.9075, 0) // 北京
	p2 := NewLLAPoint(datum.WGS84, 121.4737, 31.2304, 0) // 上海
	p3 := NewLLAPoint(datum.WGS84, 125.3245, 43.8965, 0) // 长春

	// 手工计算每一边的 Vincenty 距离（作为期望值）
	dist12, _, _, err := CalculateVincentyDistance(p1, p2)
	if err != nil {
		t.Fatalf("Vincenty distance 1-2 error: %v", err)
	}
	dist23, _, _, err := CalculateVincentyDistance(p2, p3)
	if err != nil {
		t.Fatalf("Vincenty distance 2-3 error: %v", err)
	}
	dist31, _, _, err := CalculateVincentyDistance(p3, p1)
	if err != nil {
		t.Fatalf("Vincenty distance 3-1 error: %v", err)
	}
	expected := dist12 + dist23 + dist31

	// 调用函数计算多边形周长
	perimeter, err := CalculatePolygonVincentyPerimeter(p1, p2, p3)
	if err != nil {
		t.Fatalf("CalculatePolygonVincentyPerimeter error: %v", err)
	}

	// 允许相对误差 1e-10（极高精度）
	if relativeError(perimeter, expected) > 1e-10 {
		t.Errorf("Perimeter = %f, expected %f", perimeter, expected)
	}

	// 测试点数不足的情况（少于3个点）
	_, err = CalculatePolygonVincentyPerimeter(p1, p2)
	if err == nil {
		t.Error("Expected error for less than 3 points, got nil")
	}
}

// 测试 Haversine 周长计算
func TestCalculatePolygonHaversinePerimeter(t *testing.T) {
	// 使用相同的三角形
	p1 := NewLLAPoint(datum.WGS84, 116.3972, 39.9075, 0)
	p2 := NewLLAPoint(datum.WGS84, 121.4737, 31.2304, 0)
	p3 := NewLLAPoint(datum.WGS84, 125.3245, 43.8965, 0)

	// 手工计算每一边的 Haversine 距离
	dist12, _, _, err := CalculateHaversineDistance(p1, p2)
	if err != nil {
		t.Fatalf("Haversine distance 1-2 error: %v", err)
	}
	dist23, _, _, err := CalculateHaversineDistance(p2, p3)
	if err != nil {
		t.Fatalf("Haversine distance 2-3 error: %v", err)
	}
	dist31, _, _, err := CalculateHaversineDistance(p3, p1)
	if err != nil {
		t.Fatalf("Haversine distance 3-1 error: %v", err)
	}
	expected := dist12 + dist23 + dist31

	perimeter, err := CalculatePolygonHaversinePerimeter(p1, p2, p3)
	if err != nil {
		t.Fatalf("CalculatePolygonHaversinePerimeter error: %v", err)
	}

	// 允许相对误差 1e-10
	if relativeError(perimeter, expected) > 1e-10 {
		t.Errorf("Perimeter = %f, expected %f", perimeter, expected)
	}

	// 测试点数不足
	_, err = CalculatePolygonHaversinePerimeter(p1, p2)
	if err == nil {
		t.Error("Expected error for less than 3 points, got nil")
	}
}

// 可选：测试一个更大的多边形（正方形）验证周长累加逻辑
func TestCalculatePolygonPerimeterOnSquare(t *testing.T) {
	// 赤道附近约 1km 边长的正方形
	p1 := NewLLAPoint(datum.WGS84, 0, 0, 0)
	p2 := NewLLAPoint(datum.WGS84, 0.009, 0, 0) // 东西边长 ≈ 1000m
	p3 := NewLLAPoint(datum.WGS84, 0.009, 0.009, 0)
	p4 := NewLLAPoint(datum.WGS84, 0, 0.009, 0)

	// 计算 Vincenty 周长
	vinPeri, err := CalculatePolygonVincentyPerimeter(p1, p2, p3, p4)
	if err != nil {
		t.Fatalf("Vincenty perimeter error: %v", err)
	}
	// 计算 Haversine 周长
	havPeri, err := CalculatePolygonHaversinePerimeter(p1, p2, p3, p4)
	if err != nil {
		t.Fatalf("Haversine perimeter error: %v", err)
	}

	// 两种方法应非常接近（小于 0.05%），因为范围小
	if math.Abs(vinPeri-havPeri)/vinPeri > 0.004 {
		t.Errorf("Vincenty perimeter (%.2f) and Haversine perimeter (%.2f) differ too much", vinPeri, havPeri)
	}
}

func TestDeduplicate(t *testing.T) {
	t.Log(Deduplicate(NewLLAPoints(datum.WGS84,
		[3]float64{116.1, 39.3},
		[3]float64{116.2, 39.2},
		[3]float64{116.3, 39.1},
	)...))
}

func TestGetValueWithDefault(t *testing.T) {
	t.Log(gokit.GetValueWithDefault(datum.WGS84))
	t.Log(gokit.GetValueWithDefault(datum.WGS84, datum.CGCS2000))
}

// 辅助函数：比较两个浮点数是否在相对误差范围内
func relativeError(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	return math.Abs(a-b) / math.Max(math.Abs(a), math.Abs(b))
}

// *********************************************************************************************************************
// 性能测试
// *********************************************************************************************************************

// cpu: Intel(R) Core(TM) i5-14600KF		20    	 4722488	       256.8 ns/op	       0 B/op	       0 allocs/op
func BenchmarkVincentyDistance(b *testing.B) {
	p1 := NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494)
	p2 := NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, _, _, err := CalculateVincentyDistance(p1, p2); err != nil {
			b.Fatal(err)
		}
	}
}

// cpu: Intel(R) Core(TM) i5-14600KF		20    	 8326503	       145.3 ns/op	       0 B/op	       0 allocs/op
func BenchmarkHaversineDistance(b *testing.B) {
	p1 := NewLLAPoint(datum.WGS84, 125.1450754, 43.82776494)
	p2 := NewLLAPoint(datum.WGS84, 125.1456789, 43.82709951)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, _, _, err := CalculateHaversineDistance(p1, p2); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCalcDistance(t *testing.T) {
	p1 := NewLLAPoint(datum.WGS84, 116.54886819165127, 40.11639243511271)
	p2 := NewLLAPoint(datum.WGS84)
	t.Log(CalculateVincentyDistance(p1, p2))
}
