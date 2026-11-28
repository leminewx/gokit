package projection

import (
	"errors"
	"math"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestLambertProjectOrigin(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)

	got, err := projector.Project(geography.NewLLAPoint(datum.WGS84, 0, 0))
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}

	// 赤道与本初子午线交点投影到原点
	if got != (Point{0, 0}) {
		t.Fatalf("Project(0, 0) = %v, want (0, 0)", got)
	}
}

func TestLambertProjectFormula(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)
	radius := datum.ELLIPSOID_WGS84.GetMajorRadius()

	const lon, lat = 90.0, 30.0
	got, err := projector.Project(geography.NewLLAPoint(datum.WGS84, lon, lat))
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}

	// x = R·λ, y = R·sin(φ)
	wantEast := radius * (lon * math.Pi / 180)
	wantNorth := radius * math.Sin(lat*math.Pi/180)

	easting, northing := got.GetValues()
	if math.Abs(easting-wantEast) > 1e-6 {
		t.Fatalf("easting = %v, want %v", easting, wantEast)
	}
	if math.Abs(northing-wantNorth) > 1e-6 {
		t.Fatalf("northing = %v, want %v", northing, wantNorth)
	}
}

func TestLambertProjectOutOfRange(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)

	if _, err := projector.Project(geography.NewLLAPoint(datum.WGS84, 0, 86.5)); !errors.Is(err, ErrLatitudeOutOfRange) {
		t.Fatalf("Project(86.5) error = %v, want ErrLatitudeOutOfRange", err)
	}
}

func TestLambertUnprojectOrigin(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)

	got, err := projector.Unproject(Point{0, 0})
	if err != nil {
		t.Fatalf("Unproject() error = %v", err)
	}

	lon, lat, _ := got.GetValues()
	if lon != 0 || lat != 0 {
		t.Fatalf("Unproject(0, 0) = (%v, %v), want (0, 0)", lon, lat)
	}
	if got.GetDatum() != datum.WGS84 {
		t.Fatalf("Unproject 基准面 = %v, want %v", got.GetDatum(), datum.WGS84)
	}
}

func TestLambertUnprojectNorthingOutOfRange(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)
	radius := datum.ELLIPSOID_WGS84.GetMajorRadius()

	_, err := projector.Unproject(Point{0, radius * 2})
	if !errors.Is(err, ErrNorthingOutOfRange) {
		t.Fatalf("Unproject(northing 越界) error = %v, want ErrNorthingOutOfRange", err)
	}
}

func TestLambertUnprojectAtPole(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)
	radius := datum.ELLIPSOID_WGS84.GetMajorRadius()

	// 北向值恰好取到半径时反投影得到 90°，超出 ±86° 的有效纬度范围
	_, err := projector.Unproject(Point{0, radius})
	if !errors.Is(err, ErrLatitudeOutOfRange) {
		t.Fatalf("Unproject(北极) error = %v, want ErrLatitudeOutOfRange", err)
	}
}

func TestLambertValidateLatitude(t *testing.T) {
	projector := NewLambertCylindricalEqualAreaProjector(datum.WGS84)

	for _, lat := range []float64{-86, 0, 86} {
		if err := projector.ValidateLatitude(lat); err != nil {
			t.Fatalf("ValidateLatitude(%v) error = %v", lat, err)
		}
	}

	for _, lat := range []float64{-86.0001, 86.0001} {
		if err := projector.ValidateLatitude(lat); !errors.Is(err, ErrLatitudeOutOfRange) {
			t.Fatalf("ValidateLatitude(%v) error = %v, want ErrLatitudeOutOfRange", lat, err)
		}
	}
}
