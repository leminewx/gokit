package polygon

import (
	"strings"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
	"gitee.com/leminewx/gokit/maths/geography/geometry/line"
	"gitee.com/leminewx/gokit/maths/geography/projection"
)

// LLAPolygon 经纬高多边形（闭合）
type LLAPolygon struct {
	vertexes     []geography.LLAPoint // 顶点序列（去重后）
	edges        []line.Segment       // 边（含闭合边，edges[i] = vertexes[i] → vertexes[i+1%n]）
	perimeter    float64              // 椭球面周长（m）
	projected    bool                 // 是否已投影
	projVertexes []projection.Point   // 投影后的顶点

	projPerimeter float64          // 投影周长（m）
	area          float64          // 投影面积（m²）
	centroid      projection.Point // 质心（投影坐标系）

	distanceCalculator geography.DistanceCalculator
}

var _ Polygon = (*LLAPolygon)(nil)

// NewLLAPolygon 创建一个基于经纬高坐标的多边形
//   - vertexes: 多边形顶点序列（按顺时针或逆时针有序，至少 3 个点），自动闭合、自动去重
//   - distanceCalculator: 距离计算器
func NewLLAPolygon(vertexes []geography.LLAPoint, distanceCalculator geography.DistanceCalculator) (Polygon, error) {
	vertexes = geography.Deduplicate(vertexes...)
	num := len(vertexes)
	if num < 3 {
		return nil, wrapVertexError("got %d", num)
	}

	poly := &LLAPolygon{
		vertexes:           vertexes,
		edges:              make([]line.Segment, 0, num),
		distanceCalculator: distanceCalculator,
	}

	// 创建所有边（包含闭合边）
	for i := 0; i < num; i++ {
		seg, err := line.NewLLASegment(vertexes[i], vertexes[(i+1)%num], distanceCalculator)
		if err != nil {
			return nil, err
		}

		poly.edges = append(poly.edges, seg)
		poly.perimeter += seg.GetLength()
	}

	return poly, nil
}

// Project 投影
func (own *LLAPolygon) Project(projector projection.Projector) (err error) {
	own.projected = false
	own.projPerimeter = 0
	own.projVertexes = make([]projection.Point, 0, own.GetVertexNumber())

	for _, e := range own.edges {
		if err := e.Project(projector); err != nil {
			return err
		}
	}

	// 收集投影顶点（去重首尾：闭合边导致首尾投影点重复）
	own.projVertexes = own.collectProjectedVertexes()
	own.projected = true

	// 计算投影周长、面积和质心
	own.projPerimeter = projection.CalculateEuclidDistance(append(own.projVertexes, own.projVertexes[0])...)
	own.area, own.centroid, err = projection.CalculatePolygonAreaAndCentroid(own.projVertexes...)
	return err
}

// collectProjectedVertexes 收集投影顶点（去重闭合边首尾重复）
func (own *LLAPolygon) collectProjectedVertexes() []projection.Point {
	n := len(own.edges)
	pts := make([]projection.Point, 0, n)
	for i, e := range own.edges {
		pts = append(pts, e.GetStartProjectedPoint())
		if i == n-1 {
			// 最后一条边的终点可能与第一条边的起点重复，仅当不同时添加
			end := e.GetEndProjectedPoint()
			first := pts[0]
			if end[0] != first[0] || end[1] != first[1] {
				pts = append(pts, end)
			}
		}
	}
	return pts
}

// GetProjectedArea 获取多边形投影面积
func (own *LLAPolygon) GetProjectedArea() float64 {
	return own.area
}

// GetProjectedCentroid 获取多边形质心
func (own *LLAPolygon) GetProjectedCentroid() projection.Point {
	return own.centroid
}

// GetPerimeter 获取椭球面周长
func (own *LLAPolygon) GetPerimeter() float64 {
	return own.perimeter
}

// GetProjectedPerimeter 获取投影周长
func (own *LLAPolygon) GetProjectedPerimeter() float64 {
	if !own.projected {
		return 0
	}
	return own.projPerimeter
}

// GetEdges 获取所有边（含闭合边，返回副本）
func (own *LLAPolygon) GetEdges() []line.Segment {
	edges := make([]line.Segment, len(own.edges))
	copy(edges, own.edges)
	return edges
}

// GetEdgeAt 获取指定索引的边
func (own *LLAPolygon) GetEdgeAt(index int) (line.Segment, bool) {
	if index < 0 || index >= len(own.edges) {
		return nil, false
	}
	return own.edges[index], true
}

// GetEdgeNumber 获取边数量
func (own *LLAPolygon) GetEdgeNumber() int {
	return len(own.edges)
}

// GetVertexes 获取所有顶点（返回副本）
func (own *LLAPolygon) GetVertexes() []geography.LLAPoint {
	vts := make([]geography.LLAPoint, len(own.vertexes))
	copy(vts, own.vertexes)
	return vts
}

// GetVertexAt 获取指定索引的顶点
func (own *LLAPolygon) GetVertexAt(index int) (geography.LLAPoint, bool) {
	if index < 0 || index >= len(own.vertexes) {
		return geography.LLAPoint{}, false
	}
	return own.vertexes[index], true
}

// GetVertexNumber 获取顶点数量
func (own *LLAPolygon) GetVertexNumber() int {
	return len(own.vertexes)
}

// GetProjectedVertexes 获取投影顶点（返回副本）
func (own *LLAPolygon) GetProjectedVertexes() []projection.Point {
	if !own.projected {
		return nil
	}
	vts := make([]projection.Point, len(own.projVertexes))
	copy(vts, own.projVertexes)
	return vts
}

// GetDatum 获取基准面
func (own *LLAPolygon) GetDatum() datum.Datum {
	return own.vertexes[0].GetDatum()
}

// ContainPoint 判断多边形是否包含指定投影坐标点（射线法）
func (own *LLAPolygon) ContainPoint(point projection.Point, onEdge ...bool) bool {
	if !own.projected {
		return false
	}
	return IsPointInPolygonByRay(own, point, onEdge...)
}

// ContainPolygon 判断多边形是否包含指定多边形（基于顶点包含比例）
//   - 没有包含：返回值 = 0
//   - 部分包含：0 < 返回值 < 1
//   - 完全包含：返回值 = 1
func (own *LLAPolygon) ContainPolygon(other Polygon) float64 {
	if !own.projected || other.GetVertexNumber() == 0 {
		return 0
	}

	innerVts := other.GetProjectedVertexes()
	if len(innerVts) == 0 {
		return 0
	}

	contained := 0
	for _, v := range innerVts {
		if own.ContainPoint(v) {
			contained++
		}
	}

	return float64(contained) / float64(len(innerVts))
}

// String 获取多边形字符串表示
func (own *LLAPolygon) String() string {
	var builder strings.Builder
	builder.WriteByte('[')
	for idx, v := range own.vertexes {
		if idx > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(v.String())
	}
	builder.WriteByte(']')
	return builder.String()
}
