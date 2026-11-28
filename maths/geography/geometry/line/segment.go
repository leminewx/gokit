package line

import (
	"fmt"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// LLASegment 经纬高的两点线
type LLASegment struct {
	startLLA geography.LLAPoint // 起点坐标
	endLLA   geography.LLAPoint // 终点坐标

	startPorj projection.Point // 起点投影坐标
	endProj   projection.Point // 终点投影坐标

	startAzimuth float64 // 起点方位角
	endAzimuth   float64 // 终点方位角

	length          float64 // 线段长度
	projectedLength float64 // 线段投影长度
}

var _ Segment = (*LLASegment)(nil)

// NewLLASegment 创建一条基于经纬高坐标的两点线
//   - start, end：起始坐标
//   - distanceCalculator：距离计算器
func NewLLASegment(start, end geography.LLAPoint, distanceCalculator geography.DistanceCalculator) (Segment, error) {
	if start.GetDatum() != end.GetDatum() {
		return nil, geography.ErrDatumNotMatched
	}

	var err error
	seg := &LLASegment{startLLA: start, endLLA: end}
	if seg.length, seg.startAzimuth, seg.endAzimuth, err = distanceCalculator(start, end); err != nil {
		return nil, err
	}

	return seg, nil
}

// NewLLASegments 创建多条基于经纬高坐标的两点线
//   - points：原始经纬高坐标点
//   - lengthCalculator：距离计算器
//   - 注意：输出会保持顺序，但会过滤掉相邻重复的坐标点
func NewLLASegments(points []geography.LLAPoint, lengthCalculator geography.DistanceCalculator) ([]Segment, error) {
	points = geography.Deduplicate(points...)
	num := len(points)
	if num < 2 {
		return nil, ErrTooFewPoints
	}

	segments := make([]Segment, 0, num)
	for i := 0; i < num-1; i++ {
		seg, err := NewLLASegment(points[i], points[i+1], lengthCalculator)
		if err != nil {
			return nil, err
		}

		segments = append(segments, seg)
	}

	return segments, nil
}

// Project 投影
func (own *LLASegment) Project(projector projection.Projector) (err error) {
	if own.startPorj, err = projector.Project(own.startLLA); err != nil {
		return err
	}

	own.endProj, err = projector.Project(own.endLLA)
	own.projectedLength = projection.CalculateEuclidDistance(own.startPorj, own.endProj)
	return err
}

// GetProjectedPoints 获取所有投影坐标点（保持画线时的顺序）
func (own *LLASegment) GetProjectedPoints() []projection.Point {
	points := make([]projection.Point, 0, 2)
	if own.startPorj[0] > geography.BASIC_EPSILON || own.endProj[0] > geography.BASIC_EPSILON {
		points = append(points, own.startPorj, own.endProj)
	}

	return points
}

// GetLength 获取线段长度
func (own *LLASegment) GetLength() float64 {
	return own.length
}

// GetProjectedLength 获取线段长度
func (own *LLASegment) GetProjectedLength() float64 {
	return own.projectedLength
}

// GetStartAndEndGeoPoints 获取起点和终点的地理坐标
func (own *LLASegment) GetStartPoint() (point geography.LLAPoint, azimuth float64) {
	return own.startLLA, own.startAzimuth
}

// GetStartProjectedPoint 获取起点投影坐标
func (own *LLASegment) GetStartProjectedPoint() (point projection.Point) {
	return own.startPorj
}

// GetStartAndEndAzimuth 获取线段起点和终点的方位角（单位：°）
func (own *LLASegment) GetEndPoint() (point geography.LLAPoint, azimuth float64) {
	return own.endLLA, own.endAzimuth
}

// GetEndProjectedPoint 获取终点投影坐标
func (own *LLASegment) GetEndProjectedPoint() (point projection.Point) {
	return own.endProj
}

// GetPoints 获取所有坐标点
func (own *LLASegment) GetPoints() []geography.LLAPoint {
	return []geography.LLAPoint{own.startLLA, own.endLLA}
}

// GetPointNumber 获取坐标点点数
func (own *LLASegment) GetPointNumber() int {
	return 2
}

// GetGeoCode 获取地理坐标系代码
func (own *LLASegment) GetDatum() datum.Datum {
	return own.startLLA.GetDatum()
}

// CalculateSideValue 判断指定坐标点是否在当前线段的左侧
//   - 大于0：在左侧
//   - 等于0：在线上
//   - 小于0：在右侧
func (own *LLASegment) CalculateSideValue(point projection.Point) float64 {
	return projection.CalculateSideValue([2]projection.Point{own.startPorj, own.endProj}, point)
}

// String 获取坐标字符串
func (own *LLASegment) String() string {
	startLon, startLat, startAlt := own.startLLA.GetValues()
	endLon, endLat, endAlt := own.endLLA.GetValues()
	return fmt.Sprintf(`{"start":{"longitude":%f,"latitude":%f,"altitude":%f,"azimuth":%f},"end":{"longitude":%f, "latitude":%f, "altitude":%f,"azimuth":%f}}`,
		startLon, startLat, startAlt, own.startAzimuth, endLon, endLat, endAlt, own.endAzimuth)
}
