package polygon

import (
	"fmt"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/geometry/line"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// Polygon 多边形接口
type Polygon interface {
	fmt.Stringer

	// Project 投影
	Project(projection.Projector) error

	// GetProjectedArea 获取多边形的投影面积（单位：m²）
	GetProjectedArea() float64
	// GetProjectedCentroid 获取多边形的质心点坐标（投影坐标系下）
	GetProjectedCentroid() projection.Point
	// GetPerimeter 获取多边形的椭球面周长（单位：m）
	GetPerimeter() float64
	// GetProjectedPerimeter 获取多边形的投影周长（单位：m）
	GetProjectedPerimeter() float64
	// GetProjectedVertexes 获取多边形所有顶点的投影坐标
	GetProjectedVertexes() []projection.Point

	// GetEdges 获取多边形所有的边（有序，含闭合边）
	GetEdges() []line.Segment
	// GetEdgeAt 获取多边形指定索引的边
	GetEdgeAt(index int) (line.Segment, bool)
	// GetEdgeNumber 获取多边形的边数量（等于顶点数）
	GetEdgeNumber() int

	// GetVertexes 获取多边形所有的顶点（有序）
	GetVertexes() []geography.LLAPoint
	// GetVertexAt 获取多边形指定索引的顶点
	GetVertexAt(index int) (geography.LLAPoint, bool)
	// GetVertexNumber 获取多边形的顶点数量
	GetVertexNumber() int

	// GetDatum 获取基准面
	GetDatum() datum.Datum

	// ContainPoint 判断多边形是否包含指定投影坐标点（射线法）
	//   - onEdge: 是否认为边界上的点属于内部，默认 false
	ContainPoint(point projection.Point, onEdge ...bool) bool
	// ContainPolygon 判断多边形是否包含指定多边形
	//   - 没有包含：返回值 = 0
	//   - 部分包含：0 < 返回值 < 1
	//   - 完全包含：返回值 = 1
	ContainPolygon(Polygon) float64
}

// IsPointInPolygonByRay 使用射线法判断目标点是否在指定的多边形内
//   - polygon：构成多边形的全部顶点（投影坐标，自动闭合）
//   - target：指定坐标
//   - onEdge：是否包含边界点，默认不包含
func IsPointInPolygonByRay(p Polygon, target projection.Point, onEdge ...bool) bool {
	// 1. 检查边界点
	containEdge := false
	if len(onEdge) > 0 {
		containEdge = onEdge[0]
	}
	if projection.IsOnPolyline(p.GetProjectedVertexes(), target) {
		return containEdge
	}

	// 2. 射线法判断（平面直角坐标）
	points := p.GetProjectedVertexes()
	inside := false
	cx, cy := target.GetValues()
	num := len(points)
	for i := 0; i < num; i++ {
		x1, y1 := points[i].GetValues()
		x2, y2 := points[(i+1)%num].GetValues()

		// 检查射线是否与边相交
		// 条件：点的 y 坐标在边的两个端点 y 坐标之间（排除水平边）
		if (y1 > cy) != (y2 > cy) {
			// 计算交点 x 坐标
			intersectX := (cy-y1)*(x2-x1)/(y2-y1) + x1
			if cx < intersectX {
				inside = !inside
			}
		}
	}

	return inside
}
