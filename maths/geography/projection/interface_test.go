package projection

import (
	"errors"
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestProjectorMetadata(t *testing.T) {
	tests := []struct {
		name string
		in   Projector
		code Code
	}{
		{"WebMercator", NewWebMercatorProjector(datum.WGS84), EPSG_3857},
		{"Lambert", NewLambertCylindricalEqualAreaProjector(datum.CGCS2000), EPSG_6933},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.GetCode(); got != tt.code {
				t.Fatalf("GetCode() = %v, want %v", got, tt.code)
			}

			if got := tt.in.GetDatum(); got != datum.WGS84 && got != datum.CGCS2000 {
				t.Fatalf("GetDatum() = %v, 不是有效基准面", got)
			}
		})
	}
}

func TestProjectorRoundTrip(t *testing.T) {
	const epsilon = 1e-9

	projectors := []Projector{
		NewWebMercatorProjector(datum.WGS84),
		NewLambertCylindricalEqualAreaProjector(datum.WGS84),
	}

	for _, projector := range projectors {
		t.Run(string(projector.GetCode()), func(t *testing.T) {
			origin := geography.NewLLAPoint(datum.WGS84, 113.264499, 22.542901)

			projected, err := projector.Project(origin)
			if err != nil {
				t.Fatalf("Project() error = %v", err)
			}

			back, err := projector.Unproject(projected)
			if err != nil {
				t.Fatalf("Unproject() error = %v", err)
			}

			gotLon, gotLat, _ := back.GetValues()
			wantLon, wantLat, _ := origin.GetValues()
			if math.Abs(gotLon-wantLon) > epsilon || math.Abs(gotLat-wantLat) > epsilon {
				t.Fatalf("往返坐标 = (%.12f, %.12f), want (%.12f, %.12f)", gotLon, gotLat, wantLon, wantLat)
			}

			if got := back.GetDatum(); got != datum.WGS84 {
				t.Fatalf("往返基准面 = %v, want %v", got, datum.WGS84)
			}
		})
	}
}

func TestProjectorValidateLatitude(t *testing.T) {
	tests := []struct {
		name    string
		in      Projector
		lat     float64
		wantErr bool
	}{
		{"WebMercator 赤道", NewWebMercatorProjector(datum.WGS84), 0, false},
		{"WebMercator 边界", NewWebMercatorProjector(datum.WGS84), 85.05112878, false},
		{"WebMercator 越界", NewWebMercatorProjector(datum.WGS84), 85.05112879, true},
		{"Lambert 南界", NewLambertCylindricalEqualAreaProjector(datum.WGS84), -86, false},
		{"Lambert 越界", NewLambertCylindricalEqualAreaProjector(datum.WGS84), -86.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.ValidateLatitude(tt.lat)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateLatitude(%v) 应返回错误", tt.lat)
				}
				if !errors.Is(err, ErrLatitudeOutOfRange) {
					t.Fatalf("ValidateLatitude 错误未包裹 ErrLatitudeOutOfRange: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateLatitude(%v) error = %v", tt.lat, err)
			}
		})
	}
}
