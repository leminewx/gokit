// Package geometry 提供与 GeoJSON 字段对齐的几何数据类型：Point、MultiPoint、
// LineString、MultiLineString、Polygon、MultiPolygon、GeometryCollection，以及
// Feature / FeatureCollection。
//
// 所有几何类型实现 Geometry 接口（GetType、MarshalWKT 与 encoding/json 的
// Marshaler / Unmarshaler），可直接序列化与反序列化。坐标数组按 GeoJSON 约定组织，
// 不做单位换算与坐标投影。
package geometry
