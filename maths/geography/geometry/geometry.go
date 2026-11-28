package geometry

import "encoding/json"

// Geometry 所有几何类型必须实现的接口
type Geometry interface {
	// GetType 返回几何类型名称
	GetType() string // 几何类型名称，如 "Point"
	//
	MarshalWKT() string // 转换为 WKT 字符串
	// GeoJSON 序列化通过 json.Marshaler 接口实现
	json.Marshaler
	json.Unmarshaler
}

// 具体几何类型（字段与 GeoJSON 标准完全对齐，便于自动解析）
type Point struct {
	Coordinates []float64 `json:"coordinates"`
}

type MultiPoint struct {
	Coordinates [][]float64 `json:"coordinates"`
}

type LineString struct {
	Coordinates [][]float64 `json:"coordinates"`
}

type MultiLineString struct {
	Coordinates [][][]float64 `json:"coordinates"`
}

type Polygon struct {
	Coordinates [][][]float64 `json:"coordinates"` // 外环+内环
}

type MultiPolygon struct {
	Coordinates [][][][]float64 `json:"coordinates"`
}

type GeometryCollection struct {
	Geometries []Geometry `json:"geometries"`
}

// Feature 与 FeatureCollection
type Feature struct {
	ID         any            `json:"id,omitempty"`
	Geometry   Geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
	BBox       []float64      `json:"bbox,omitempty"`
}

type FeatureCollection struct {
	Features []*Feature `json:"features"`
	BBox     []float64  `json:"bbox,omitempty"`
}
