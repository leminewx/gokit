package line

import (
	"fmt"
	"math"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

type (
	// Segment 两点线段
	Segment interface {
		fmt.Stringer

		// Project 投影
		Project(projection.Projector) error

		// GetStartProjectedPoint 获取起点投影坐标
		GetStartProjectedPoint() (point projection.Point)
		// GetEndProjectedPoint 获取终点投影坐标
		GetEndProjectedPoint() (point projection.Point)
		// GetProjectedPoints 获取所有投影坐标点（有序）
		GetProjectedPoints() []projection.Point
		// GetProjectedLength 获取线段长度
		GetProjectedLength() float64

		// GetStartPoint 获取起点
		GetStartPoint() (point geography.LLAPoint, azimuth float64)
		// GetEndPoint 获取终点
		GetEndPoint() (point geography.LLAPoint, azimuth float64)
		// GetPoints 获取所有坐标点（有序）
		GetPoints() []geography.LLAPoint
		// GetPointNumber 获取坐标点数
		GetPointNumber() int

		// GetLength 获取线段长度
		GetLength() float64
		// GetDatum 获取基准面
		GetDatum() datum.Datum
		// CalculateSideValue 判断指定坐标点是否在当前线段的左侧
		//   - 大于0：在左侧
		//   - 等于0：在线上
		//   - 小于0：在右侧
		CalculateSideValue(projection.Point) float64
	}

	// Polyline 多点线段的接口
	Polyline interface {
		Segment

		// Insert 插入坐标点
		//    - index: 插入位置，默认在末尾插入
		Insert(point geography.LLAPoint, index ...uint) error
		// Delete 删除坐标点
		Delete(index int)

		// GetPointAt 获取折线指定位置的地理坐标
		GetPointAt(index int) (geography.LLAPoint, bool)
		// GetProjectedPointAt 获取折线指定位置的投影坐标
		GetProjectedPointAt(index int) (projection.Point, bool)
		// GetSegments 获取组成折线的所有线段
		GetSegments() []Segment
		// GetSegmentAt 获取折线指定位置的线段
		GetSegmentAt(index int) (Segment, bool)
	}
)

// 编译期断言：内置线段与折线实现各自接口
var (
	_ Segment  = (*LLASegment)(nil)
	_ Polyline = (*LLAPolyline)(nil)
)

// FindNearestPointOnPolyline 寻找点到多段线的最短距离和最近线段索引
func FindNearestPointOnPolyline(polyline Polyline, target projection.Point) (minDistance float64, segmentIdx int) {
	minDistance = math.MaxFloat64
	for idx, seg := range polyline.GetSegments() {
		signedDist := projection.CalculatePointToSegmentDistance([2]projection.Point(seg.GetProjectedPoints()), target)
		distance := math.Abs(signedDist)
		if distance < minDistance {
			minDistance = distance
			segmentIdx = idx
		}
	}

	return
}
