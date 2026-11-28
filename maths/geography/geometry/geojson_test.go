package geometry

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestGeoJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		geom Geometry
	}{
		{name: "point", geom: &Point{Coordinates: []float64{1.5, 2.5}}},
		{name: "multi point", geom: &MultiPoint{Coordinates: [][]float64{{1, 2}, {3, 4}}}},
		{name: "line string", geom: &LineString{Coordinates: [][]float64{{0, 0}, {1, 1}}}},
		{name: "multi line string", geom: &MultiLineString{Coordinates: [][][]float64{{{0, 0}, {1, 1}}}}},
		{name: "polygon", geom: &Polygon{Coordinates: [][][]float64{{{0, 0}, {1, 1}, {1, 0}, {0, 0}}}}},
		{name: "multi polygon", geom: &MultiPolygon{Coordinates: [][][][]float64{{{{0, 0}, {1, 1}, {1, 0}, {0, 0}}}}}},
		{name: "geometry collection", geom: &GeometryCollection{Geometries: []Geometry{
			&Point{Coordinates: []float64{1, 2}},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.geom)
			if err != nil {
				t.Fatalf("Marshal 返回错误: %v", err)
			}

			decoded, err := decodeGeometry(data)
			if err != nil {
				t.Fatalf("decodeGeometry 返回错误: %v", err)
			}
			if decoded.GetType() != tt.geom.GetType() {
				t.Fatalf("GetType() = %q, want %q", decoded.GetType(), tt.geom.GetType())
			}

			decodedData, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("再次 Marshal 返回错误: %v", err)
			}
			if string(decodedData) != string(data) {
				t.Errorf("round trip = %s, want %s", decodedData, data)
			}
		})
	}
}

func TestGeoJSONTypeMismatch(t *testing.T) {
	t.Parallel()

	var point Point
	err := json.Unmarshal([]byte(`{"type":"LineString","coordinates":[[0,0]]}`), &point)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Unmarshal 错误 = %v, want ErrTypeMismatch", err)
	}
}

func TestGeoJSONUnknownType(t *testing.T) {
	t.Parallel()

	if _, err := decodeGeometry([]byte(`{"type":"Unknown","coordinates":[]}`)); !errors.Is(err, ErrUnknownType) {
		t.Errorf("decodeGeometry 错误 = %v, want ErrUnknownType", err)
	}
}

func TestGeometryCollectionInvalidChild(t *testing.T) {
	t.Parallel()

	var collection GeometryCollection
	data := []byte(`{"type":"GeometryCollection","geometries":[{"type":"Bad","coordinates":[]}]}`)
	if err := json.Unmarshal(data, &collection); !errors.Is(err, ErrUnknownType) {
		t.Errorf("Unmarshal 错误 = %v, want ErrUnknownType", err)
	}
}
