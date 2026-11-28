package activator

import "testing"

func TestReLUActivate(t *testing.T) {
	t.Parallel()

	relu := NewReLU()
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "positive", in: 2.5, want: 2.5},
		{name: "zero", in: 0, want: 0},
		{name: "negative", in: -3, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := relu.Activate(tt.in); got != tt.want {
				t.Errorf("Activate(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestReLUDerive(t *testing.T) {
	t.Parallel()

	relu := NewReLU()
	if got := relu.Derive(1); got != 1 {
		t.Errorf("Derive(1) = %v, want 1", got)
	}
	if got := relu.Derive(-1); got != 0 {
		t.Errorf("Derive(-1) = %v, want 0", got)
	}
}

func TestReLUGetName(t *testing.T) {
	t.Parallel()

	if got := NewReLU().GetName(); got != NAME_RELU {
		t.Errorf("GetName() = %q, want %q", got, NAME_RELU)
	}
}
