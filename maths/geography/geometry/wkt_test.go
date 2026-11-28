package geometry

import (
	"testing"
)

func TestGeometryWKT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		geom Geometry
		want string
	}{
		{
			name: "point",
			geom: &Point{Coordinates: []float64{116.4, 39.9}},
			want: "POINT(116.4 39.9)",
		},
		{
			name: "multi point",
			geom: &MultiPoint{Coordinates: [][]float64{{1, 2}, {3, 4}}},
			want: "MULTIPOINT((1 2), (3 4))",
		},
		{
			name: "line string",
			geom: &LineString{Coordinates: [][]float64{{0, 0}, {1, 1}}},
			want: "LINESTRING((0 0, 1 1))",
		},
		{
			name: "multi line string",
			geom: &MultiLineString{Coordinates: [][][]float64{
				{{0, 0}, {1, 1}},
				{{2, 2}, {3, 3}},
			}},
			want: "MULTILINESTRING((0 0, 1 1), (2 2, 3 3))",
		},
		{
			name: "polygon",
			geom: &Polygon{Coordinates: [][][]float64{
				{{0, 0}, {4, 0}, {4, 4}, {0, 0}},
			}},
			want: "POLYGON((0 0, 4 0, 4 4, 0 0))",
		},
		{
			name: "multi polygon",
			geom: &MultiPolygon{Coordinates: [][][][]float64{
				{
					{{0, 0}, {4, 0}, {4, 4}, {0, 0}},
					{{1, 1}, {2, 1}, {2, 2}, {1, 1}},
				},
			}},
			want: "MULTIPOLYGON(((0 0, 4 0, 4 4, 0 0), (1 1, 2 1, 2 2, 1 1)))",
		},
		{
			name: "geometry collection",
			geom: &GeometryCollection{Geometries: []Geometry{
				&Point{Coordinates: []float64{1, 2}},
				&LineString{Coordinates: [][]float64{{0, 0}, {1, 1}}},
			}},
			want: "GEOMETRYCOLLECTION(POINT(1 2), LINESTRING((0 0, 1 1)))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.geom.MarshalWKT(); got != tt.want {
				t.Errorf("MarshalWKT() = %q, want %q", got, tt.want)
			}
		})
	}
}
