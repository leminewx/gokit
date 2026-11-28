package geometry

import (
	"strconv"
	"strings"
)

// GetType 返回几何类型名称
func (own *Point) GetType() string { return "Point" }

// MarshalWKT 转换为 WKT 字符串
func (own *Point) MarshalWKT() string {
	return formatCoordinate("POINT", own.Coordinates)
}

// GetType 返回几何类型名称
func (own *MultiPoint) GetType() string { return "MultiPoint" }

// MarshalWKT 转换为 WKT 字符串
func (own *MultiPoint) MarshalWKT() string {
	parts := make([]string, len(own.Coordinates))
	for i, coord := range own.Coordinates {
		parts[i] = "(" + formatNumbers(coord) + ")"
	}

	return "MULTIPOINT(" + strings.Join(parts, ", ") + ")"
}

// GetType 返回几何类型名称
func (own *LineString) GetType() string { return "LineString" }

// MarshalWKT 转换为 WKT 字符串
func (own *LineString) MarshalWKT() string {
	return formatRing("LINESTRING", own.Coordinates)
}

// GetType 返回几何类型名称
func (own *MultiLineString) GetType() string { return "MultiLineString" }

// MarshalWKT 转换为 WKT 字符串
func (own *MultiLineString) MarshalWKT() string {
	parts := make([]string, len(own.Coordinates))
	for i, line := range own.Coordinates {
		parts[i] = "(" + joinCoordinates(line) + ")"
	}

	return "MULTILINESTRING(" + strings.Join(parts, ", ") + ")"
}

// GetType 返回几何类型名称
func (own *Polygon) GetType() string { return "Polygon" }

// MarshalWKT 转换为 WKT 字符串
func (own *Polygon) MarshalWKT() string {
	return formatRings("POLYGON", own.Coordinates)
}

// GetType 返回几何类型名称
func (own *MultiPolygon) GetType() string { return "MultiPolygon" }

// MarshalWKT 转换为 WKT 字符串
func (own *MultiPolygon) MarshalWKT() string {
	parts := make([]string, len(own.Coordinates))
	for i, polygon := range own.Coordinates {
		parts[i] = formatPolygon(polygon)
	}

	return "MULTIPOLYGON(" + strings.Join(parts, ", ") + ")"
}

// GetType 返回几何类型名称
func (own *GeometryCollection) GetType() string { return "GeometryCollection" }

// MarshalWKT 转换为 WKT 字符串
func (own *GeometryCollection) MarshalWKT() string {
	parts := make([]string, 0, len(own.Geometries))
	for _, geom := range own.Geometries {
		if geom == nil {
			continue
		}

		parts = append(parts, geom.MarshalWKT())
	}

	return "GEOMETRYCOLLECTION(" + strings.Join(parts, ", ") + ")"
}

// formatCoordinate 将单个坐标格式化为 "TYPE(x y)" 形式的 WKT
func formatCoordinate(geomType string, coord []float64) string {
	return geomType + "(" + formatNumbers(coord) + ")"
}

// formatNumbers 将坐标分量拼接为 WKT 数字序列
func formatNumbers(values []float64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.FormatFloat(value, 'g', -1, 64)
	}

	return strings.Join(parts, " ")
}

// formatRing 将坐标序列格式化为 "TYPE((x y, x y))" 形式的 WKT
func formatRing(geomType string, coords [][]float64) string {
	return geomType + "((" + joinCoordinates(coords) + "))"
}

// formatRings 将多个环格式化为 "TYPE((...), (...))" 形式的 WKT
func formatRings(geomType string, rings [][][]float64) string {
	return geomType + formatPolygon(rings)
}

// formatPolygon 将多个环格式化为 "((...), (...))" 形式的 WKT 坐标体
func formatPolygon(rings [][][]float64) string {
	parts := make([]string, len(rings))
	for i, ring := range rings {
		parts[i] = "(" + joinCoordinates(ring) + ")"
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

// joinCoordinates 将坐标序列拼接为 "x y, x y" 形式
func joinCoordinates(coords [][]float64) string {
	parts := make([]string, len(coords))
	for i, coord := range coords {
		parts[i] = formatNumbers(coord)
	}

	return strings.Join(parts, ", ")
}
