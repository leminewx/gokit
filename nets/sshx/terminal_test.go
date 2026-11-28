package sshx

import "testing"

func TestTerminalNormalizeDefaults(t *testing.T) {
	got := Terminal{}.normalize()

	want := Terminal{
		Term:        "xterm",
		Width:       120,
		Height:      40,
		TtyOpIspeed: 14400,
		TtyOpOspeed: 14400,
	}
	if got != want {
		t.Fatalf("normalize() = %+v, want %+v", got, want)
	}
}

func TestTerminalNormalizeKeepsExplicitValues(t *testing.T) {
	origin := Terminal{
		Term:        "vt100",
		Width:       80,
		Height:      24,
		TtyOpIspeed: 9600,
		TtyOpOspeed: 9600,
	}

	if got := origin.normalize(); got != origin {
		t.Fatalf("normalize() = %+v, want %+v", got, origin)
	}
}

func TestTerminalNormalizeNonPositiveValues(t *testing.T) {
	tests := []struct {
		name string
		in   Terminal
		want Terminal
	}{
		{"负数宽高", Terminal{Width: -1, Height: -1}, Terminal{Term: "xterm", Width: 120, Height: 40, TtyOpIspeed: 14400, TtyOpOspeed: 14400}},
		{"零值速度", Terminal{Term: "vt100", Width: 1, Height: 1}, Terminal{Term: "vt100", Width: 1, Height: 1, TtyOpIspeed: 14400, TtyOpOspeed: 14400}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.normalize(); got != tt.want {
				t.Fatalf("normalize() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTerminalNormalizeDoesNotMutateReceiver(t *testing.T) {
	origin := Terminal{}
	_ = origin.normalize()

	if origin != (Terminal{}) {
		t.Fatalf("normalize 不应修改接收者: %+v", origin)
	}
}
