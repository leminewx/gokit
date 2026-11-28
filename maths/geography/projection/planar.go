package projection

import (
	"math"

	"gitee.com/leminewx/gokit/maths/geography"
)

// 平面几何计算：以下函数均作用于投影平面坐标（Point），长度单位为米
const (
	// _MIN_POLYGON_AREA 多边形最小面积阈值
	_MIN_POLYGON_AREA = 1e-10
	// _SEGMENT_DEGENERATE_THRESHOLD 线段退化阈值
	_SEGMENT_DEGENERATE_THRESHOLD = 1e-30
)

// CalculatePolygonAreaAndCentroid 计算指定的多边形的面积和质心（面积中心）
//   - points: 多边形的顶点序列（按顺时针或逆时针顺序，至少3个点，自动闭环）
//   - 注意: 当多边形退化（面积小于阈值）时，回退到顶点算术平均中心
func CalculatePolygonAreaAndCentroid(points ...Point) (area float64, centroid Point, err error) {
	num := len(points)
	if num < 3 {
		return 0, Point{}, geography.ErrPolygonLessThan3Points
	}

	var signedArea, sumCx, sumCy float64
	minX, minY := points[0].GetValues()
	maxX, maxY := minX, minY
	for i := 0; i < num; i++ {
		x1, y1 := points[i].GetValues()
		x2, y2 := points[(i+1)%num].GetValues()
		cross := x1*y2 - x2*y1
		signedArea += cross
		sumCx += (x1 + x2) * cross
		sumCy += (y1 + y2) * cross

		// 更新包围盒
		if x1 < minX {
			minX = x1
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 < minY {
			minY = y1
		}
		if y1 > maxY {
			maxY = y1
		}
	}

	signedArea *= 0.5
	areaAbs := math.Abs(signedArea)

	// 退化判断：有向面积接近零，或者面积相对于包围盒过小
	diag := math.Hypot(maxX-minX, maxY-minY)
	if areaAbs < _MIN_POLYGON_AREA || areaAbs < _MIN_POLYGON_AREA*diag*diag {
		var sumX, sumY float64
		for _, p := range points {
			x, y := p.GetValues()
			sumX += x
			sumY += y
		}

		return areaAbs, Point{sumX / float64(num), sumY / float64(num)}, nil
	}

	x := sumCx / (6 * signedArea)
	y := sumCy / (6 * signedArea)
	return areaAbs, Point{x, y}, nil
}

// CalculatePolygonArea 计算指定的多边形的面积（单位：m^2）
//   - points: 多边形的顶点序列（按顺时针或逆时针顺序，至少3个点，自动闭环）
//   - 注意: 调用前确保投影点处于同一投影坐标系
func CalculatePolygonArea(points ...Point) float64 {
	num := len(points)
	if num < 3 {
		return 0
	}

	var sum float64
	for i := 0; i < num; i++ {
		x1, y1 := points[i].GetValues()
		x2, y2 := points[(i+1)%num].GetValues()
		sum += x1*y2 - x2*y1
	}

	return math.Abs(sum) / 2.0
}

// CalculatePointToSegmentDistance 计算点到线段的最小距离
func CalculatePointToSegmentDistance(segment [2]Point, target Point) float64 {
	px, py := target.GetValues()
	ax, ay := segment[0].GetValues()
	bx, by := segment[1].GetValues()

	abx, aby := bx-ax, by-ay
	apx, apy := px-ax, py-ay
	dot := abx*apx + aby*apy
	if dot <= 0 {
		return math.Hypot(apx, apy) // 到 A 的距离
	}

	ab2 := abx*abx + aby*aby
	if dot >= ab2 {
		return math.Hypot(px-bx, py-by) // 到 B 的距离
	}

	// 垂足
	t := dot / ab2
	hx, hy := ax+t*abx, ay+t*aby
	return math.Hypot(px-hx, py-hy)
}

// CalculatePointToLineDistance 计算点到线段所在直线的垂直距离
func CalculatePointToLineDistance(segment [2]Point, target Point) float64 {
	x1, y1 := segment[0].GetValues()
	x2, y2 := segment[1].GetValues()
	cx, cy := target.GetValues()

	// 方向向量
	vx, vy := x2-x1, y2-y1
	wx, wy := cx-x1, cy-y1

	// 叉积
	cross := vx*wy - wx*vy

	// 线段长度
	length := math.Sqrt(vx*vx + vy*vy)

	if length < _SEGMENT_DEGENERATE_THRESHOLD {
		// 线段退化为点
		return math.Sqrt(wx*wx + wy*wy)
	}

	// 垂直距离 = |叉积| / 线段长度
	return math.Abs(cross) / length
}

// CalculateEuclidDistance 计算投影点之间的欧氏距离（两点线段累加）
func CalculateEuclidDistance(points ...Point) (length float64) {
	for i := 0; i < len(points)-1; i++ {
		x1, y1 := points[i].GetValues()
		x2, y2 := points[i+1].GetValues()
		length += math.Hypot(x2-x1, y2-y1)
	}

	return
}

// IsOnPolyline 判断目标坐标点是否在指定的多点线段上面
func IsOnPolyline(polyline []Point, target Point) bool {
	num := len(polyline)
	if num < 2 {
		return false
	}

	for i := 0; i < num-1; i++ {
		if IsOnSegment([2]Point{polyline[i], polyline[(i+1)%num]}, target) {
			return true
		}
	}

	return false
}

// IntersectSegments 判断两条（两点）线段是否相交
//   - 返回相交点列表与是否相交
func IntersectSegments(segmentA, segmentB [2]Point) (intersections []Point, ok bool) {
	a := segmentA[0]
	b := segmentA[1]
	c := segmentB[0]
	d := segmentB[1]

	ax, ay := a.GetValues()
	bx, by := b.GetValues()
	cx, cy := c.GetValues()
	dx, dy := d.GetValues()

	cross := func(x1, y1, x2, y2 float64) float64 {
		return x1*y2 - x2*y1
	}

	// 计算四个叉积
	o1 := cross(bx-ax, by-ay, cx-ax, cy-ay)
	o2 := cross(bx-ax, by-ay, dx-ax, dy-ay)
	o3 := cross(dx-cx, dy-cy, ax-cx, ay-cy)
	o4 := cross(dx-cx, dy-cy, bx-cx, by-cy)

	// 共线情况：两条线段在同一条直线上
	if math.Abs(o1) < geography.BASIC_EPSILON && math.Abs(o2) < geography.BASIC_EPSILON && math.Abs(o3) < geography.BASIC_EPSILON {
		return handleCollinearIntersection(a, b, c, d)
	}

	// 非共线但端点落在另一条线段上（端点接触）
	// 判断端点是否在另一条线段上
	if math.Abs(o1) < geography.BASIC_EPSILON && IsOnSegment([2]Point{a, b}, c) {
		return []Point{c}, true
	}
	if math.Abs(o2) < geography.BASIC_EPSILON && IsOnSegment([2]Point{a, b}, d) {
		return []Point{d}, true
	}
	if math.Abs(o3) < geography.BASIC_EPSILON && IsOnSegment([2]Point{c, d}, a) {
		return []Point{a}, true
	}
	if math.Abs(o4) < geography.BASIC_EPSILON && IsOnSegment([2]Point{c, d}, b) {
		return []Point{b}, true
	}

	// 非共线：标准相交判定（跨立试验）
	if o1*o2 < 0 && o3*o4 < 0 {
		// 计算交点
		intersection := calculateIntersection(a, b, c, d)
		return []Point{intersection}, true
	}

	// 不相交
	return nil, false
}

// handleCollinearIntersection 处理共线线段相交的情况
// 返回是否相交以及所有交点（重叠区间的端点）
func handleCollinearIntersection(a, b, c, d Point) ([]Point, bool) {
	// 使用投影参数 t 判断重叠区间
	// 将点投影到线段AB的方向上
	ax, ay := a.GetValues()
	bx, by := b.GetValues()
	cx, cy := c.GetValues()
	dx, dy := d.GetValues()

	// 计算方向向量和长度平方
	vx, vy := bx-ax, by-ay
	lengthSquared := vx*vx + vy*vy

	if lengthSquared < _SEGMENT_DEGENERATE_THRESHOLD {
		// 线段A退化为点
		if IsOnSegment([2]Point{a, c}, d) {
			return []Point{a}, true
		}

		return nil, false
	}

	// 计算四个端点在AB方向上的投影参数 t
	tA := 0.0
	tB := 1.0
	tC := ((cx-ax)*vx + (cy-ay)*vy) / lengthSquared
	tD := ((dx-ax)*vx + (dy-ay)*vy) / lengthSquared

	// 获取两条线段在AB方向上的区间 [minT1, maxT1] 和 [minT2, maxT2]
	minT1, maxT1 := math.Min(tA, tB), math.Max(tA, tB)
	minT2, maxT2 := math.Min(tC, tD), math.Max(tC, tD)

	// 判断区间是否重叠（带容差）
	if maxT1 < minT2-geography.BASIC_EPSILON || maxT2 < minT1-geography.BASIC_EPSILON {
		return nil, false // 无重叠
	}

	// 计算重叠区间的端点
	overlapStart := math.Max(minT1, minT2)
	overlapEnd := math.Min(maxT1, maxT2)

	// 构建交点列表
	intersections := make([]Point, 0, 2)

	// 起点（如果重叠区间有长度）
	if overlapEnd-overlapStart > geography.BASIC_EPSILON {
		startPoint := Point{ax + vx*overlapStart, ay + vy*overlapStart}
		endPoint := Point{ax + vx*overlapEnd, ay + vy*overlapEnd}
		intersections = append(intersections, startPoint, endPoint)
	} else {
		// 只有一个交点（端点接触）
		point := Point{ax + vx*overlapStart, ay + vy*overlapStart}
		intersections = append(intersections, point)
	}

	return intersections, true
}

// calculateIntersection 计算两条非共线线段的交点
//   - 参数方程：P = A + t*(B-A) = C + u*(D-C)
func calculateIntersection(a, b, c, d Point) Point {
	ax, ay := a.GetValues()
	bx, by := b.GetValues()
	cx, cy := c.GetValues()
	dx, dy := d.GetValues()

	// 计算分母
	denom := (bx-ax)*(dy-cy) - (by-ay)*(dx-cx)

	// 如果分母为0，说明平行（不应发生，因为调用前已判断相交）
	if math.Abs(denom) < _SEGMENT_DEGENERATE_THRESHOLD {
		return a // 返回默认值
	}

	// 计算参数 t
	t := ((cx-ax)*(dy-cy) - (cy-ay)*(dx-cx)) / denom

	// 计算交点坐标
	x := ax + t*(bx-ax)
	y := ay + t*(by-ay)

	return Point{x, y}
}

// IsOnSegment 判断指定点是否在线段上（使用容差）
func IsOnSegment(segment [2]Point, target Point) bool {
	// 1. 判断是否在直线上（垂直距离）
	if CalculatePointToLineDistance(segment, target) >= geography.BASIC_EPSILON {
		return false
	}

	// 2. 判断是否在线段范围内（投影参数 t）
	px, py := target.GetValues()
	ax, ay := segment[0].GetValues()
	bx, by := segment[1].GetValues()

	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay

	lengthSquared := vx*vx + vy*vy
	if lengthSquared < _SEGMENT_DEGENERATE_THRESHOLD {
		return math.Hypot(wx, wy) < geography.BASIC_EPSILON
	}

	t := (wx*vx + wy*vy) / lengthSquared

	// 投影参数的容差应该基于线段长度动态计算
	// 将距离容差转换为投影参数容差
	eps := geography.BASIC_EPSILON / (math.Sqrt(lengthSquared) + _SEGMENT_DEGENERATE_THRESHOLD)

	return t >= -eps && t <= 1.0+eps
}

// CalculateSideValue 判断指定点位于线段的左侧、线上还是右侧
//   - 大于0：在左侧
//   - 等于0：在线上
//   - 小于0：在右侧
func CalculateSideValue(segment [2]Point, target Point) float64 {
	cx, cy := target.GetValues()
	x1, y1 := segment[0].GetValues()
	x2, y2 := segment[1].GetValues()

	// 计算方向向量和待判断向量
	vx, vy := x2-x1, y2-y1
	wx, wy := cx-x1, cy-y1

	// 计算叉积（判定左右的核心）
	cross := vx*wy - wx*vy

	// 计算线段长度的平方（用于归一化和距离计算）
	lengthSquared := vx*vx + vy*vy

	// 防止线段退化为点
	if lengthSquared < _SEGMENT_DEGENERATE_THRESHOLD {
		// 左右方向无意义，按规则返回
		return 0.0
	}

	// 动态容差（自适应线段长度）
	eps := geography.BASIC_EPSILON * math.Max(1.0, math.Sqrt(lengthSquared))

	// 返回带符号的叉积值（用于方向判断）
	if math.Abs(cross) < eps {
		return 0.0
	}

	return cross
}

// Deduplicate 删除重复的投影坐标点（保留首次出现的顺序）
func Deduplicate(points ...Point) []Point {
	filtered := make([]Point, 0, len(points))
	seen := make(map[Point]struct{}, len(points))
	for _, point := range points {
		if _, ok := seen[point]; ok {
			continue
		}

		seen[point] = struct{}{}
		filtered = append(filtered, point)
	}

	return filtered
}
