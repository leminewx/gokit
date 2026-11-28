package projection

import (
	"math"
	"testing"
)

// TestIsOnPolyline 验证折线命中判断
func TestIsOnPolyline(t *testing.T) {
	t.Parallel()

	polyline := []Point{{0, 0}, {10, 0}, {10, 10}}
	cases := []struct {
		name   string
		target Point
		want   bool
	}{
		{"在首段上", Point{5, 0}, true},
		{"在拐点上", Point{10, 0}, true},
		{"在第二段上", Point{10, 5}, true},
		{"不在折线上", Point{5, 5}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsOnPolyline(polyline, c.target); got != c.want {
				t.Errorf("IsOnPolyline(%v) = %v, want %v", c.target, got, c.want)
			}
		})
	}

	if IsOnPolyline([]Point{{0, 0}}, Point{0, 0}) {
		t.Error("少于两个顶点时应返回 false")
	}
}

// ============================================================================
// 1. 距离计算测试
// ============================================================================
func TestCalculatePointToSegmentDistance(t *testing.T) {
	tests := []struct {
		name     string
		segment  [2]Point
		target   Point
		expected float64
	}{
		{
			name:     "垂足在线段中间",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 3},
			expected: 3.0,
		},
		{
			name:     "垂足在起点之外",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-3, 4},
			expected: 5.0, // sqrt(9+16)
		},
		{
			name:     "垂足在终点之外",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{13, 4},
			expected: 5.0, // sqrt(9+16)
		},
		{
			name:     "点在线段上",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 0},
			expected: 0.0,
		},
		{
			name:     "线段退化为点",
			segment:  [2]Point{{0, 0}, {0, 0}},
			target:   Point{3, 4},
			expected: 5.0,
		},
		{
			name:     "斜线段",
			segment:  [2]Point{{0, 0}, {10, 10}},
			target:   Point{5, 0},
			expected: 3.5355339059327378, // |5*0 - 5*10| / sqrt(200) = 50/14.142 = 3.5355
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePointToSegmentDistance(tt.segment, tt.target)
			assertFloatEqual(t, tt.expected, result, "distance")
		})
	}
}

func TestCalculatePointToLineDistance(t *testing.T) {
	tests := []struct {
		name     string
		segment  [2]Point
		target   Point
		expected float64
	}{
		{
			name:     "垂足在线段中间",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 3},
			expected: 3.0,
		},
		{
			name:     "垂足在延长线上（垂直距离仍有效）",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-3, 4},
			expected: 4.0, // 到延长线的垂直距离
		},
		{
			name:     "点在线段所在直线上",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-5, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePointToLineDistance(tt.segment, tt.target)
			assertFloatEqual(t, tt.expected, result, "distance")
		})
	}
}

// ============================================================================
// 2. 点在线段上判断测试
// ============================================================================
func TestIsOnSegment(t *testing.T) {
	tests := []struct {
		name     string
		segment  [2]Point
		target   Point
		expected bool
	}{
		{
			name:     "点在线段中间",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 0},
			expected: true,
		},
		{
			name:     "点在起点",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{0, 0},
			expected: true,
		},
		{
			name:     "点在终点",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{10, 0},
			expected: true,
		},
		{
			name:     "点在延长线上（应返回false）",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-5, 0},
			expected: false,
		},
		{
			name:     "点在延长线上靠近端点（容差内）",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-1e-8, 0}, // 距离端点 1e-7 米
			expected: true,            // DISTANCE_EPSILON = 1e-7，所以应判定为在线段上
		},
		{
			name:     "点在延长线上但超出容差",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-1e-6, 0}, // 距离端点 1e-5 米
			expected: false,           // 超出 DISTANCE_EPSILON
		},
		{
			name:     "点不在直线上",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 1},
			expected: false,
		},
		{
			name:     "斜线段上的点",
			segment:  [2]Point{{0, 0}, {10, 10}},
			target:   Point{5, 5},
			expected: true,
		},
		{
			name:     "斜线段延长线上的点",
			segment:  [2]Point{{0, 0}, {10, 10}},
			target:   Point{-5, -5},
			expected: false,
		},
		{
			name:     "线段退化为点",
			segment:  [2]Point{{0, 0}, {0, 0}},
			target:   Point{0, 0},
			expected: true,
		},
		{
			name:     "线段退化为点（不在点上）",
			segment:  [2]Point{{0, 0}, {0, 0}},
			target:   Point{1, 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOnSegment(tt.segment, tt.target)
			assertBoolEqual(t, tt.expected, result, "IsOnSegment")
		})
	}
}

// ============================================================================
// 3. 方向判断测试
// ============================================================================
func TestCalculateSideValue(t *testing.T) {
	tests := []struct {
		name     string
		segment  [2]Point
		target   Point
		expected float64 // >0: left, 0: on, <0: right
	}{
		{
			name:     "点在左侧",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 3},
			expected: 30.0, // cross = 10*3 - 5*0 = 30
		},
		{
			name:     "点在右侧",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, -3},
			expected: -30.0,
		},
		{
			name:     "点在线段上",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{5, 0},
			expected: 0.0,
		},
		{
			name:     "点在延长线上（仍在直线上）",
			segment:  [2]Point{{0, 0}, {10, 0}},
			target:   Point{-5, 0},
			expected: 0.0,
		},
		{
			name:     "斜线段-左侧",
			segment:  [2]Point{{0, 0}, {10, 10}},
			target:   Point{0, 10},
			expected: 100.0, // cross = 10*10 - 0*10 = 100
		},
		{
			name:     "斜线段-右侧",
			segment:  [2]Point{{0, 0}, {10, 10}},
			target:   Point{10, 0},
			expected: -100.0,
		},
		{
			name:     "线段退化为点",
			segment:  [2]Point{{0, 0}, {0, 0}},
			target:   Point{5, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSideValue(tt.segment, tt.target)
			// 只检查符号，不检查精确值（因为叉积值会随坐标变化）
			if tt.expected > 0 && result <= 0 {
				t.Errorf("%s: expected >0, got %.10f", tt.name, result)
			} else if tt.expected < 0 && result >= 0 {
				t.Errorf("%s: expected <0, got %.10f", tt.name, result)
			} else if tt.expected == 0 && math.Abs(result) > 1e-6 {
				t.Errorf("%s: expected 0, got %.10f", tt.name, result)
			}
		})
	}
}

// ============================================================================
// 4. 线段相交测试
// ============================================================================
func TestAreSegmentsIntersecting(t *testing.T) {
	tests := []struct {
		name            string
		segmentA        [2]Point
		segmentB        [2]Point
		shouldIntersect bool
		expectedPoints  [][]float64 // 预期的交点坐标列表
	}{
		{
			name:            "X形交叉",
			segmentA:        [2]Point{{0, 0}, {10, 10}},
			segmentB:        [2]Point{{0, 10}, {10, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 5}},
		},
		{
			name:            "T形接触",
			segmentA:        [2]Point{{0, 0}, {10, 0}},
			segmentB:        [2]Point{{5, 0}, {5, 10}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 0}},
		},
		{
			name:            "端点接触",
			segmentA:        [2]Point{{0, 0}, {5, 5}},
			segmentB:        [2]Point{{5, 5}, {10, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 5}},
		},
		{
			name:            "共线部分重叠",
			segmentA:        [2]Point{{0, 0}, {10, 0}},
			segmentB:        [2]Point{{5, 0}, {15, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 0}, {10, 0}},
		},
		{
			name:            "共线-线段包含关系",
			segmentA:        [2]Point{{0, 0}, {10, 0}},
			segmentB:        [2]Point{{3, 0}, {7, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{3, 0}, {7, 0}},
		},
		{
			name:            "共线-端点接触",
			segmentA:        [2]Point{{0, 0}, {5, 0}},
			segmentB:        [2]Point{{5, 0}, {10, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 0}},
		},
		{
			name:            "共线-分离",
			segmentA:        [2]Point{{0, 0}, {5, 0}},
			segmentB:        [2]Point{{6, 0}, {10, 0}},
			shouldIntersect: false,
			expectedPoints:  nil,
		},
		{
			name:            "共线-同一条线段",
			segmentA:        [2]Point{{0, 0}, {10, 0}},
			segmentB:        [2]Point{{0, 0}, {10, 0}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{0, 0}, {10, 0}},
		},
		{
			name:            "平行不相交",
			segmentA:        [2]Point{{0, 0}, {10, 0}},
			segmentB:        [2]Point{{0, 5}, {10, 5}},
			shouldIntersect: false,
			expectedPoints:  nil,
		},
		{
			name:            "不相交-交叉但不在线段范围内",
			segmentA:        [2]Point{{0, 0}, {5, 5}},
			segmentB:        [2]Point{{6, 0}, {10, 4}},
			shouldIntersect: false,
			expectedPoints:  nil,
		},
		{
			name:            "垂直相交",
			segmentA:        [2]Point{{5, 0}, {5, 10}},
			segmentB:        [2]Point{{0, 5}, {10, 5}},
			shouldIntersect: true,
			expectedPoints:  [][]float64{{5, 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			points, intersects := IntersectSegments(tt.segmentA, tt.segmentB)
			assertBoolEqual(t, tt.shouldIntersect, intersects, "intersects")

			if tt.shouldIntersect {
				assertLenEqual(t, len(tt.expectedPoints), len(points), "points count")
				for i, expected := range tt.expectedPoints {
					if i < len(points) {
						expectedPoint := Point{expected[0], expected[1]}
						assertPointsEqual(t, expectedPoint, points[i], "point")
					}
				}
			} else {
				assertLenEqual(t, 0, len(points), "points count should be 0")
			}
		})
	}
}

// ============================================================================
// 5. 多边形面积测试
// ============================================================================
func TestCalculatePolygonArea(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		expected float64
	}{
		{
			name:     "矩形",
			points:   []Point{{0, 0}, {10, 0}, {10, 5}, {0, 5}},
			expected: 50.0,
		},
		{
			name:     "三角形",
			points:   []Point{{0, 0}, {10, 0}, {5, 5}},
			expected: 25.0,
		},
		{
			name:     "逆时针矩形",
			points:   []Point{{0, 0}, {0, 5}, {10, 5}, {10, 0}},
			expected: 50.0,
		},
		{
			name:     "少于3个点",
			points:   []Point{{0, 0}, {10, 0}},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePolygonArea(tt.points...)
			assertFloatEqual(t, tt.expected, result, "area")
		})
	}
}

// ============================================================================
// 6. 欧氏距离测试
// ============================================================================
func TestCalculateEuclidDistance(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		expected float64
	}{
		{
			name:     "两个点",
			points:   []Point{{0, 0}, {3, 4}},
			expected: 5.0,
		},
		{
			name:     "三个点组成的折线",
			points:   []Point{{0, 0}, {3, 4}, {6, 8}},
			expected: 10.0, // 5 + 5
		},
		{
			name:     "单点",
			points:   []Point{{0, 0}},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateEuclidDistance(tt.points...)
			assertFloatEqual(t, tt.expected, result, "length")
		})
	}
}

// ============================================================================
// 7. 去重测试
// ============================================================================
func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name     string
		points   []Point
		expected []Point
	}{
		{
			name:     "无重复",
			points:   []Point{{0, 0}, {1, 1}, {2, 2}},
			expected: []Point{{0, 0}, {1, 1}, {2, 2}},
		},
		{
			name:     "有重复点",
			points:   []Point{{0, 0}, {1, 1}, {0, 0}, {2, 2}, {1, 1}},
			expected: []Point{{0, 0}, {1, 1}, {2, 2}},
		},
		{
			name:     "全部重复",
			points:   []Point{{0, 0}, {0, 0}, {0, 0}},
			expected: []Point{{0, 0}},
		},
		{
			name:     "空列表",
			points:   []Point{},
			expected: []Point{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Deduplicate(tt.points...)
			assertLenEqual(t, len(tt.expected), len(result), "length")
			for i, expected := range tt.expected {
				assertPointsEqual(t, expected, result[i], "point")
			}
		})
	}
}

// ============================================================================
// 8. 集成测试：真实场景模拟
// ============================================================================
func TestRealWorldScenario_VehicleCrossingLine(t *testing.T) {
	// 模拟场景：车辆轨迹是否穿越边界线
	// 边界线：从 (0,10) 到 (10,10) 的水平线
	boundary := [2]Point{{0, 10}, {10, 10}}

	// 车辆轨迹：从 (2,8) 到 (8,12) 穿越边界
	trajectory := [2]Point{{2, 8}, {8, 12}}

	points, intersects := IntersectSegments(boundary, trajectory)
	assertBoolEqual(t, true, intersects, "should intersect")
	assertLenEqual(t, 1, len(points), "should have 1 intersection point")

	if len(points) > 0 {
		// 交点应该在 y=10 上
		assertFloatEqual(t, 10.0, points[0][1], "intersection y should be 10")
	}

	// 判断车辆在穿越前后相对于边界的方向
	startDir := CalculateSideValue(boundary, Point{2, 8})
	endDir := CalculateSideValue(boundary, Point{8, 12})

	// 起点在下方（右侧），终点在上方（左侧）
	if startDir > 0 || endDir < 0 {
		t.Errorf("start should be right (negative), got %.10f; end should be left (positive), got %.10f", startDir, endDir)
	}
}

func TestRealWorldScenario_PointOnRoad(t *testing.T) {
	// 模拟场景：判断车辆是否在路线上
	// 路线：一条折线
	roadSegment := [2]Point{{0, 0}, {10, 10}}

	// 车辆在路线上（有微小偏移）
	carOnRoad := Point{5.0000001, 5.0000001}
	// 车辆偏离路线
	carOffRoad := Point{5, 5.1}

	assertBoolEqual(t, true, IsOnSegment(roadSegment, carOnRoad), "car should be on road")
	assertBoolEqual(t, false, IsOnSegment(roadSegment, carOffRoad), "car should be off road")
}

// ============================================================================
// 9. 基准测试（性能）
// ============================================================================
func BenchmarkCalculatePointToSegmentDistance(b *testing.B) {
	segment := [2]Point{{0, 0}, {1000, 1000}}
	target := Point{500, 500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculatePointToSegmentDistance(segment, target)
	}
}

func BenchmarkIsOnSegment(b *testing.B) {
	segment := [2]Point{{0, 0}, {1000, 1000}}
	target := Point{500, 500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsOnSegment(segment, target)
	}
}

func BenchmarkAreSegmentsIntersecting(b *testing.B) {
	segA := [2]Point{{0, 0}, {1000, 1000}}
	segB := [2]Point{{0, 1000}, {1000, 0}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IntersectSegments(segA, segB)
	}
}

// ============================================================================
// 10. 边界条件测试
// ============================================================================
func TestEdgeCases(t *testing.T) {
	t.Run("零长度线段", func(t *testing.T) {
		seg := [2]Point{{5, 5}, {5, 5}}
		target := Point{5, 5}

		// IsOnSegment 应返回 true
		assertBoolEqual(t, true, IsOnSegment(seg, target), "point on zero-length segment")

		// CalculateSideValue 应返回 0
		result := CalculateSideValue(seg, target)
		assertFloatEqual(t, 0.0, result, "CalculateSideValue on zero-length segment should return 0")
	})

	t.Run("极大坐标值", func(t *testing.T) {
		// UTM 坐标可能达到 500000 量级
		seg := [2]Point{{500000, 5000000}, {500010, 5000010}}
		target := Point{500005, 5000005}

		// 应该能正确处理，不溢出
		result := CalculatePointToSegmentDistance(seg, target)
		if math.IsNaN(result) || math.IsInf(result, 0) {
			t.Errorf("large coordinates caused invalid result: %v", result)
		}
	})

	t.Run("共线但不相交", func(t *testing.T) {
		segA := [2]Point{{0, 0}, {5, 0}}
		segB := [2]Point{{6, 0}, {10, 0}}

		points, intersects := IntersectSegments(segA, segB)
		assertBoolEqual(t, false, intersects, "should not intersect")
		assertLenEqual(t, 0, len(points), "points should be empty")
	})

	t.Run("点在线段端点容差内", func(t *testing.T) {
		seg := [2]Point{{0, 0}, {10, 0}}

		// 在端点外 5e-8 米（小于 DISTANCE_EPSILON = 1e-7）
		target := Point{-5e-8, 0}
		assertBoolEqual(t, true, IsOnSegment(seg, target), "point within tolerance should be on segment")

		// 在端点外 5e-6 米（大于 DISTANCE_EPSILON）
		target2 := Point{-5e-6, 0}
		assertBoolEqual(t, false, IsOnSegment(seg, target2), "point beyond tolerance should not be on segment")
	})
}

// _EPSILON 娴偣姣旇緝瀹瑰樊
const _EPSILON = 1e-9

// almostEqual 鍒ゆ柇涓や釜娴偣鏁板湪瀹瑰樊鑼冨洿鍐呮槸鍚︾浉绛?
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < _EPSILON
}

// almostEqualPoint 鍒ゆ柇涓や釜鎶曞奖鐐规槸鍚﹁繎浼奸噸鍚?
func almostEqualPoint(p1, p2 Point) bool {
	return almostEqual(p1[0], p2[0]) && almostEqual(p1[1], p2[1])
}

// assertPointsEqual 鏂█涓や釜鎶曞奖鐐硅繎浼奸噸鍚?
func assertPointsEqual(t *testing.T, expected, actual Point, msg string) {
	t.Helper()
	if !almostEqualPoint(expected, actual) {
		t.Errorf("%s: got %v, want %v", msg, actual, expected)
	}
}

// assertBoolEqual 鏂█甯冨皵鍊肩浉绛?
func assertBoolEqual(t *testing.T, expected, actual bool, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: got %v, want %v", msg, actual, expected)
	}
}

// assertFloatEqual 鏂█娴偣鏁拌繎浼肩浉绛?
func assertFloatEqual(t *testing.T, expected, actual float64, msg string) {
	t.Helper()
	if !almostEqual(expected, actual) {
		t.Errorf("%s: got %v, want %v", msg, actual, expected)
	}
}

// assertLenEqual 鏂█闀垮害鐩哥瓑
func assertLenEqual(t *testing.T, expected, actual int, msg string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: got %d, want %d", msg, actual, expected)
	}
}
