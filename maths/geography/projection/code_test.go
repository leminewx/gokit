package projection

import "testing"

func TestCodeValues(t *testing.T) {
	tests := []struct {
		name string
		in   Code
		want string
	}{
		{"EPSG_3857", EPSG_3857, "EPSG:3857"},
		{"EPSG_6933", EPSG_6933, "EPSG:6933"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.in); got != tt.want {
				t.Fatalf("Code = %q, want %q", got, tt.want)
			}
		})
	}

	if EPSG_3857 == EPSG_6933 {
		t.Fatal("EPSG_3857 与 EPSG_6933 不应相等")
	}
}
