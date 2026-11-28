package datum

import "testing"

func TestDatumGetCRS(t *testing.T) {
	tests := []struct {
		name string
		in   Datum
		want CRS
	}{
		{"WGS84", WGS84, CRS_WGS84},
		{"CGCS2000", CGCS2000, CRS_CGCS2000},
		{"零值", Datum(0), CRS(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.GetCRS(); got != tt.want {
				t.Fatalf("GetCRS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatumGetEllipsoid(t *testing.T) {
	tests := []struct {
		name string
		in   Datum
		want Ellipsoid
	}{
		{"WGS84", WGS84, ELLIPSOID_WGS84},
		{"CGCS2000", CGCS2000, ELLIPSOID_CGCS2000},
		{"零值", Datum(0), Ellipsoid(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.GetEllipsoid(); got != tt.want {
				t.Fatalf("GetEllipsoid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatumComposite(t *testing.T) {
	// 基准面 = 高 8 位坐标系 | 低 8 位椭球体
	if got, want := uint16(WGS84), uint16(CRS_WGS84)|uint16(ELLIPSOID_WGS84); got != want {
		t.Fatalf("WGS84 = 0x%04X, want 0x%04X", got, want)
	}
	if got, want := uint16(CGCS2000), uint16(CRS_CGCS2000)|uint16(ELLIPSOID_CGCS2000); got != want {
		t.Fatalf("CGCS2000 = 0x%04X, want 0x%04X", got, want)
	}

	if WGS84 == CGCS2000 {
		t.Fatal("WGS84 与 CGCS2000 不应相等")
	}

	// 拼接后取值应与来源一致
	if WGS84.GetCRS() != CRS_WGS84 || WGS84.GetEllipsoid() != ELLIPSOID_WGS84 {
		t.Fatalf("WGS84 分量解析错误: CRS=%v Ellipsoid=%v", WGS84.GetCRS(), WGS84.GetEllipsoid())
	}
	if CGCS2000.GetCRS() != CRS_CGCS2000 || CGCS2000.GetEllipsoid() != ELLIPSOID_CGCS2000 {
		t.Fatalf("CGCS2000 分量解析错误: CRS=%v Ellipsoid=%v", CGCS2000.GetCRS(), CGCS2000.GetEllipsoid())
	}
}
