package activator

import (
	"math"
	"testing"
)

func TestTanhActivate(t *testing.T) {
	t.Parallel()

	tanh := NewTanh()
	if got := tanh.Activate(0); got != 0 {
		t.Errorf("Activate(0) = %v, want 0", got)
	}
	if got, want := tanh.Activate(1), math.Tanh(1); got != want {
		t.Errorf("Activate(1) = %v, want %v", got, want)
	}
	if got := tanh.Activate(100); math.Abs(got-1) > 1e-9 {
		t.Errorf("Activate(100) = %v, want 1", got)
	}
}

func TestTanhDerive(t *testing.T) {
	t.Parallel()

	tanh := NewTanh()
	if got := tanh.Derive(0); got != 1 {
		t.Errorf("Derive(0) = %v, want 1", got)
	}
	if got := tanh.Derive(100); math.Abs(got) > 1e-9 {
		t.Errorf("Derive(100) = %v, want 0", got)
	}
}

func TestTanhGetName(t *testing.T) {
	t.Parallel()

	if got := NewTanh().GetName(); got != NAME_TANH {
		t.Errorf("GetName() = %q, want %q", got, NAME_TANH)
	}
}
