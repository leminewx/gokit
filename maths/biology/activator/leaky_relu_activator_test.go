package activator

import (
	"math"
	"testing"
)

func TestLeakyReLUActivate(t *testing.T) {
	t.Parallel()

	leaky := NewLeakyReLU(0.01)
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "positive", in: 2, want: 2},
		{name: "negative", in: -2, want: -0.02},
		{name: "zero", in: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := leaky.Activate(tt.in); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Activate(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLeakyReLUDerive(t *testing.T) {
	t.Parallel()

	leaky := NewLeakyReLU(0.01)
	if got := leaky.Derive(3); got != 1 {
		t.Errorf("Derive(3) = %v, want 1", got)
	}
	if got := leaky.Derive(-3); got != 0.01 {
		t.Errorf("Derive(-3) = %v, want 0.01", got)
	}
}

func TestLeakyReLUGetName(t *testing.T) {
	t.Parallel()

	if got := NewLeakyReLU(0.01).GetName(); got != NAME_LEAKY_RELU {
		t.Errorf("GetName() = %q, want %q", got, NAME_LEAKY_RELU)
	}
}
