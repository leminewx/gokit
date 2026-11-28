package line

import (
	"math"
	"strings"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// LLAPolyline 经纬高的（不闭环）多点线段
type LLAPolyline struct {
	segments           []Segment
	length             float64
	projectedLength    float64
	projected          bool
	distanceCalculator geography.DistanceCalculator
}

// NewLLAPolyline 创建一条基于经纬高坐标的（不闭环）多点线段
//   - points: 原始经纬高坐标点
//   - distanceCalculator：距离计算器
//   - 注意：输出会保持顺序，但会过滤掉相邻重复的坐标点
func NewLLAPolyline(points []geography.LLAPoint, distanceCalculator geography.DistanceCalculator) (Polyline, error) {
	points = geography.Deduplicate(points...)
	num := len(points)
	if num < 2 {
		return nil, ErrTooFewPoints
	}

	polyline := &LLAPolyline{
		segments:           make([]Segment, 0, num-1),
		distanceCalculator: distanceCalculator,
	}
	for i := 0; i < num-1; i++ {
		seg, err := NewLLASegment(points[i], points[i+1], distanceCalculator)
		if err != nil {
			return nil, err
		}

		polyline.segments = append(polyline.segments, seg)
		polyline.length += seg.GetLength()
	}

	return polyline, nil
}

// Project 投影
func (own *LLAPolyline) Project(projector projection.Projector) (err error) {
	own.projectedLength = 0
	own.projected = false
	for _, seg := range own.segments {
		if err := seg.Project(projector); err != nil {
			return err
		}

		own.projectedLength += seg.GetProjectedLength()
	}

	own.projected = true
	return nil
}

// Insert 插入坐标点
//   - point: 新坐标点
//   - index: 插入位置（点索引），未指定则在末尾插入；0 则在头部插入，1~n-1 则在中间插入
func (own *LLAPolyline) Insert(point geography.LLAPoint, index ...uint) error {
	num := uint(own.GetPointNumber())

	// 在末尾插入
	if len(index) == 0 || index[0] == num {
		start, _ := own.GetEndPoint()
		seg, err := NewLLASegment(start, point, own.distanceCalculator)
		if err != nil {
			return err
		}

		own.length += seg.GetLength()
		own.segments = append(own.segments, seg)
		own.projected = false
		return nil
	}

	// 索引超出范围
	idx := index[0]
	if idx > num {
		return wrapIndexError("[0, %d]: %d", num, idx)
	}

	// 在头部插入（新点成为第一个点）
	if idx == 0 {
		end, _ := own.GetStartPoint()
		seg, err := NewLLASegment(point, end, own.distanceCalculator)
		if err != nil {
			return err
		}

		own.length += seg.GetLength()
		own.segments = append([]Segment{seg}, own.segments...)
		own.projected = false
		return nil
	}

	// 在中间插入：将 segments[idx-1]（即 P[idx-1]→P[idx]）拆分为 P[idx-1]→新点 和 新点→P[idx]
	oldSeg := own.segments[idx-1]
	oldStart, _ := oldSeg.GetStartPoint()
	oldEnd, _ := oldSeg.GetEndPoint()

	own.length -= oldSeg.GetLength()

	seg1, err := NewLLASegment(oldStart, point, own.distanceCalculator)
	if err != nil {
		return err
	}
	seg2, err := NewLLASegment(point, oldEnd, own.distanceCalculator)
	if err != nil {
		return err
	}

	// 用两个新线段替换旧线段
	own.segments = append(own.segments[:idx-1], append([]Segment{seg1, seg2}, own.segments[idx:]...)...)
	own.length += seg1.GetLength() + seg2.GetLength()
	own.projected = false
	return nil
}

// Delete 删除坐标点
//   - index: 待删除点的索引（0 表示第一个点，n-1 表示最后一个点）
//   - 当点数 ≤ 2 或者索引越界时，静默忽略
func (own *LLAPolyline) Delete(index int) {
	n := own.GetPointNumber()
	if n <= 2 || index < 0 || index >= n {
		return
	}

	own.projected = false

	// 删除首点：移除 segments[0]，segments[1:] 整体前移
	if index == 0 {
		own.length -= own.segments[0].GetLength()
		own.segments = own.segments[1:]
		return
	}

	// 删除尾点：移除最后一个 segment
	if index == n-1 {
		last := len(own.segments) - 1
		own.length -= own.segments[last].GetLength()
		own.segments = own.segments[:last]
		return
	}

	// 删除中间点：合并 segments[index-1] 和 segments[index]
	segA := own.segments[index-1]
	segB := own.segments[index]
	startA, _ := segA.GetStartPoint()
	endB, _ := segB.GetEndPoint()

	own.length -= segA.GetLength() + segB.GetLength()

	newSeg, err := NewLLASegment(startA, endB, own.distanceCalculator)
	if err != nil {
		// 合并失败（极端情况：对跖点等），回退
		return
	}

	own.length += newSeg.GetLength()
	own.segments = append(own.segments[:index-1], append([]Segment{newSeg}, own.segments[index+1:]...)...)
}

// GetDatum 获取基准面
func (own *LLAPolyline) GetDatum() datum.Datum {
	return own.segments[0].GetDatum()
}

// GetProjectedPoints 获取所有投影坐标点（保持画线时的顺序）
func (own *LLAPolyline) GetProjectedPoints() []projection.Point {
	if !own.projected {
		return nil
	}

	points := make([]projection.Point, 0, len(own.segments)+1)
	segNum := len(own.segments)
	for idx, seg := range own.segments {
		segPoints := seg.GetProjectedPoints()
		points = append(points, segPoints[0])
		if segNum == idx+1 {
			points = append(points, segPoints[1])
		}
	}

	return points
}

// GetLength 获取线段长度
func (own *LLAPolyline) GetLength() float64 {
	return own.length
}

// GetProjectedLength 获取线段的投影长度
func (own *LLAPolyline) GetProjectedLength() float64 {
	if !own.projected {
		return 0
	}

	return own.projectedLength
}

// GetStartAndEndGeoPoints 获取起点和终点的地理坐标
func (own *LLAPolyline) GetStartPoint() (point geography.LLAPoint, azimuth float64) {
	return own.segments[0].GetStartPoint()
}

// GetStartProjectedPoint 获取起点投影坐标
func (own *LLAPolyline) GetStartProjectedPoint() (point projection.Point) {
	return own.segments[0].GetStartProjectedPoint()
}

// GetStartAndEndAzimuth 获取线段起点和终点的方位角（单位：°）
func (own *LLAPolyline) GetEndPoint() (point geography.LLAPoint, azimuth float64) {
	return own.segments[len(own.segments)-1].GetEndPoint()
}

// GetEndProjectedPoint 获取终点投影坐标
func (own *LLAPolyline) GetEndProjectedPoint() (point projection.Point) {
	return own.segments[len(own.segments)-1].GetEndProjectedPoint()
}

// GetPoints 获取所有坐标点
func (own *LLAPolyline) GetPoints() []geography.LLAPoint {
	points := make([]geography.LLAPoint, 0, len(own.segments)+1)
	segNum := len(own.segments)
	for idx, seg := range own.segments {
		segPoints := seg.GetPoints()
		points = append(points, segPoints[0])
		if segNum == idx+1 {
			points = append(points, segPoints[1])
		}
	}

	return points
}

// GetPointNumber 获取坐标点点数
func (own *LLAPolyline) GetPointNumber() int {
	return len(own.segments) + 1
}

// CalculateSideValue 判断指定坐标点是否在当前线段的左侧（基于整体走向/有向面积法）
//   - 大于0：在左侧
//   - 等于0：在线上
//   - 小于0：在右侧
func (own *LLAPolyline) CalculateSideValue(point projection.Point) float64 {
	if !own.projected || len(own.segments) == 0 {
		return 0.0
	}

	// 1. 计算点到多段线的最短距离
	if nearestDist, _ := FindNearestPointOnPolyline(own, point); nearestDist < geography.BASIC_EPSILON {
		return 0.0
	}

	// 2. 使用有向面积计算整体方向
	area := calculateShoelaceArea(own.GetProjectedPoints(), point)

	// 动态容差
	eps := geography.BASIC_EPSILON * math.Max(1.0, own.GetLength())

	switch {
	case math.Abs(area) < eps:
		return 0.0
	case area < 0:
		return -1
	default:
		return 1
	}
}

// calculateShoelaceArea 使用鞋带公式计算有向面积
func calculateShoelaceArea(polyline []projection.Point, target projection.Point) float64 {
	// 构建闭合多边形：polyline + target + polyline[0]
	points := make([]projection.Point, len(polyline)+2)
	copy(points, polyline)
	points[len(polyline)] = target
	points[len(polyline)+1] = polyline[0]

	area := 0.0
	for i := 0; i < len(points)-1; i++ {
		x1, y1 := points[i].GetValues()
		x2, y2 := points[i+1].GetValues()
		area += x1*y2 - x2*y1
	}

	return area / 2.0
}

// GetGeographicPointAt 获取折线指定位置的坐标
func (own *LLAPolyline) GetPointAt(index int) (geography.LLAPoint, bool) {
	num := own.GetPointNumber()
	if index < 0 || num <= index {
		return geography.LLAPoint{}, false
	}

	if index == num-1 {
		end, _ := own.GetEndPoint()
		return end, true
	}

	point, _ := own.segments[index].GetStartPoint()
	return point, true
}

// GetProjectedPointAt 获取折线指定位置的投影坐标
func (own *LLAPolyline) GetProjectedPointAt(index int) (projection.Point, bool) {
	num := own.GetPointNumber()
	if index < 0 || num <= index {
		return projection.Point{}, false
	}

	if index == num-1 {
		return own.GetEndProjectedPoint(), true
	}

	return own.segments[index].GetStartProjectedPoint(), true
}

// GetSegments 获取组成折线的所有线段（返回副本）
func (own *LLAPolyline) GetSegments() []Segment {
	segments := make([]Segment, len(own.segments))
	copy(segments, own.segments)
	return segments
}

// GetSegmentAt 获取折线指定位置的线段
func (own *LLAPolyline) GetSegmentAt(index int) (Segment, bool) {
	num := len(own.segments)
	if index < 0 || num <= index {
		return nil, false
	}

	return own.segments[index], true
}

// String 获取线段字符串
func (own *LLAPolyline) String() string {
	var builder strings.Builder
	builder.WriteByte('[')
	for idx, seg := range own.segments {
		if idx > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(seg.String())
	}
	builder.WriteByte(']')
	return builder.String()
}
