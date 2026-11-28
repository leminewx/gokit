package geography

import (
	"testing"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestEqualter(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"LLA 相同", LLAPoint{}.IsEqualTo(LLAPoint{}), true},
		{"LLA 不同", NewLLAPoint(0, 1, 2).IsEqualTo(NewLLAPoint(0, 1, 3)), false},
		{"ECEF 相同", NewECEFPoint(datum.WGS84, 1, 2, 3).IsEqualTo(NewECEFPoint(datum.WGS84, 1, 2, 3)), true},
		{"ECEF 不同", NewECEFPoint(datum.WGS84, 1, 2, 3).IsEqualTo(NewECEFPoint(datum.WGS84, 1, 2, 4)), false},
		{"ENU 相同", NewENUPoint(datum.WGS84, 1, 2, 3).IsEqualTo(NewENUPoint(datum.WGS84, 1, 2, 3)), true},
		{"ENU 不同", NewENUPoint(datum.WGS84, 1, 2, 3).IsEqualTo(NewENUPoint(datum.WGS84, 1, 2, 4)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("IsEqualTo() = %v, want %v", tt.got, tt.want)
			}
		})
	}
}
