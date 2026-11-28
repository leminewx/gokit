package geometry

import (
	"encoding/json"
	"strings"
)

var (
	_ Geometry = (*Point)(nil)
	_ Geometry = (*MultiPoint)(nil)
	_ Geometry = (*LineString)(nil)
	_ Geometry = (*MultiLineString)(nil)
	_ Geometry = (*Polygon)(nil)
	_ Geometry = (*MultiPolygon)(nil)
	_ Geometry = (*GeometryCollection)(nil)
)

// 编译期断言：全部几何类型支持按 GeoJSON 规范序列化与反序列化
var (
	_ json.Marshaler   = (*Point)(nil)
	_ json.Unmarshaler = (*Point)(nil)
	_ json.Marshaler   = (*MultiPoint)(nil)
	_ json.Unmarshaler = (*MultiPoint)(nil)
	_ json.Marshaler   = (*LineString)(nil)
	_ json.Unmarshaler = (*LineString)(nil)
	_ json.Marshaler   = (*MultiLineString)(nil)
	_ json.Unmarshaler = (*MultiLineString)(nil)
	_ json.Marshaler   = (*Polygon)(nil)
	_ json.Unmarshaler = (*Polygon)(nil)
	_ json.Marshaler   = (*MultiPolygon)(nil)
	_ json.Unmarshaler = (*MultiPolygon)(nil)
	_ json.Marshaler   = (*GeometryCollection)(nil)
	_ json.Unmarshaler = (*GeometryCollection)(nil)
)

// MarshalJSON 按 GeoJSON 规范序列化点几何
func (own *Point) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析点几何
func (own *Point) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化多点几何
func (own *MultiPoint) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析多点几何
func (own *MultiPoint) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化线几何
func (own *LineString) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析线几何
func (own *LineString) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化多线几何
func (own *MultiLineString) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析多线几何
func (own *MultiLineString) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化面几何
func (own *Polygon) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析面几何
func (own *Polygon) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化多面几何
func (own *MultiPolygon) MarshalJSON() ([]byte, error) {
	return marshalCoordinates(own.GetType(), own.Coordinates)
}

// UnmarshalJSON 按 GeoJSON 规范解析多面几何
func (own *MultiPolygon) UnmarshalJSON(data []byte) error {
	return unmarshalCoordinates(data, own.GetType(), &own.Coordinates)
}

// MarshalJSON 按 GeoJSON 规范序列化几何集合
func (own *GeometryCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       string     `json:"type"`
		Geometries []Geometry `json:"geometries"`
	}{
		Type:       own.GetType(),
		Geometries: own.Geometries,
	})
}

// UnmarshalJSON 按 GeoJSON 规范解析几何集合，并按 type 字段分发到具体类型
func (own *GeometryCollection) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type       string            `json:"type"`
		Geometries []json.RawMessage `json:"geometries"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if !strings.EqualFold(raw.Type, own.GetType()) {
		return wrapTypeError(raw.Type, own.GetType())
	}

	geometries := make([]Geometry, 0, len(raw.Geometries))
	for _, item := range raw.Geometries {
		geom, err := decodeGeometry(item)
		if err != nil {
			return err
		}
		geometries = append(geometries, geom)
	}

	own.Geometries = geometries
	return nil
}

// marshalCoordinates 序列化带 type 与 coordinates 字段的 GeoJSON 对象
func marshalCoordinates(geomType string, coordinates any) ([]byte, error) {
	return json.Marshal(struct {
		Type        string `json:"type"`
		Coordinates any    `json:"coordinates"`
	}{
		Type:        geomType,
		Coordinates: coordinates,
	})
}

// unmarshalCoordinates 解析 GeoJSON 对象并校验 type 与目标类型一致
func unmarshalCoordinates(data []byte, wantType string, coordinates any) error {
	var raw struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if !strings.EqualFold(raw.Type, wantType) {
		return wrapTypeError(raw.Type, wantType)
	}

	return json.Unmarshal(raw.Coordinates, coordinates)
}

// decodeGeometry 按 type 字段将原始 JSON 解析为具体几何类型
func decodeGeometry(data []byte) (Geometry, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, err
	}

	var geom Geometry
	switch strings.ToLower(head.Type) {
	case "point":
		geom = &Point{}
	case "multipoint":
		geom = &MultiPoint{}
	case "linestring":
		geom = &LineString{}
	case "multilinestring":
		geom = &MultiLineString{}
	case "polygon":
		geom = &Polygon{}
	case "multipolygon":
		geom = &MultiPolygon{}
	case "geometrycollection":
		geom = &GeometryCollection{}
	default:
		return nil, ErrUnknownType
	}

	if err := json.Unmarshal(data, geom); err != nil {
		return nil, err
	}

	return geom, nil
}
