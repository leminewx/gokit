package spectrum

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateFFTConstant(t *testing.T) {
	transformed, err := CalculateFFT([]float64{1, 1, 1, 1})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	want := []complex128{4, 0, 0, 0}
	for index := range want {
		if math.Abs(real(transformed[index])-real(want[index])) > 1e-12 ||
			math.Abs(imag(transformed[index])) > 1e-12 {
			t.Fatalf("X[%d] = %v, want %v", index, transformed[index], want[index])
		}
	}
}

func TestCalculateFFTImpulse(t *testing.T) {
	transformed, err := CalculateFFT([]float64{1, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range transformed {
		if math.Abs(real(transformed[index])-1) > 1e-12 || math.Abs(imag(transformed[index])) > 1e-12 {
			t.Fatalf("X[%d] = %v, want 1", index, transformed[index])
		}
	}
}

func TestCalculateFFTSingleBin(t *testing.T) {
	const size = 8
	values := make([]float64, size)
	for index := range values {
		values[index] = math.Cos(2 * math.Pi * 2 * float64(index) / size)
	}

	transformed, err := CalculateFFT(values)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// 实余弦在 k 与 N-k 处各贡献 N/2，其余频点接近 0
	requireClose(t, "X[2]", real(transformed[2]), size/2, 1e-12)
	requireClose(t, "X[6]", real(transformed[6]), size/2, 1e-12)

	for _, index := range []int{0, 1, 3, 4, 5, 7} {
		if math.Hypot(real(transformed[index]), imag(transformed[index])) > 1e-12 {
			t.Fatalf("X[%d] = %v, want ~0", index, transformed[index])
		}
	}
}

func TestCalculateFFTRoundTrip(t *testing.T) {
	values := []float64{0.5, -1.25, 3, 2, -0.75, 1, 0.25, -2}

	transformed, err := CalculateFFT(values)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	restored, err := CalculateIFFT(transformed)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range values {
		requireClose(t, "实部", real(restored[index]), values[index], 1e-12)
		requireClose(t, "虚部", imag(restored[index]), 0, 1e-12)
	}
}

func TestCalculateIFFTDC(t *testing.T) {
	restored, err := CalculateIFFT([]complex128{4, 0, 0, 0})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range restored {
		requireClose(t, "实部", real(restored[index]), 1, 1e-12)
		requireClose(t, "虚部", imag(restored[index]), 0, 1e-12)
	}
}

func TestCalculateFFTErrors(t *testing.T) {
	if _, err := CalculateFFT(nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyInput)
	}

	if _, err := CalculateFFT([]float64{1, 2, 3}); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidLength)
	}

	if _, err := CalculateFFT([]float64{1, math.NaN()}); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}

	if _, err := CalculateIFFT(nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyInput)
	}

	if _, err := CalculateIFFT([]complex128{1, 2, 3}); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidLength)
	}

	if _, err := CalculateIFFT([]complex128{1, complex(math.Inf(1), 0)}); !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
