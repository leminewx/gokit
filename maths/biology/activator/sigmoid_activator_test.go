package activator

import (
	"math"
	"testing"
)

func TestSigmoidActivate(t *testing.T) {
	t.Parallel()

	sigmoid := NewSigmoid()
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "zero", in: 0, want: 0.5},
		{name: "positive", in: 2, want: 0.8807970779778823},
		{name: "negative", in: -2, want: 0.11920292202211755},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sigmoid.Activate(tt.in); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Activate(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSigmoidDerive(t *testing.T) {
	t.Parallel()

	sigmoid := NewSigmoid()
	// 导数在 0 点取最大值 0.25
	if got := sigmoid.Derive(0); math.Abs(got-0.25) > 1e-9 {
		t.Errorf("Derive(0) = %v, want 0.25", got)
	}
	if got := sigmoid.Derive(10); got <= 0 || got >= 0.25 {
		t.Errorf("Derive(10) = %v, want in (0, 0.25)", got)
	}
}

func TestSigmoidGetName(t *testing.T) {
	t.Parallel()

	if got := NewSigmoid().GetName(); got != NAME_SIGMOID {
		t.Errorf("GetName() = %q, want %q", got, NAME_SIGMOID)
	}
}
