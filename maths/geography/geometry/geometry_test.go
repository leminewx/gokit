package geometry

import (
	"testing"
)

func TestGeometriesImplementGeometry(t *testing.T) {
	t.Parallel()

	geometries := []Geometry{
		&Point{},
		&MultiPoint{},
		&LineString{},
		&MultiLineString{},
		&Polygon{},
		&MultiPolygon{},
		&GeometryCollection{},
	}

	seen := make(map[string]bool, len(geometries))
	for _, geom := range geometries {
		name := geom.GetType()
		if name == "" {
			t.Errorf("%T 的 GetType 返回空字符串", geom)
		}
		if seen[name] {
			t.Errorf("几何类型名称重复：%q", name)
		}
		seen[name] = true
	}
}
