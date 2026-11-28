package sampler

import (
	"errors"
	"math"
	"testing"
)

func TestSampleNormalMoments(t *testing.T) {
	sampler := NewDefaultSampler(19)
	const count = 5000

	total, squared := 0.0, 0.0
	for range count {
		value, err := sampler.SampleNormal(3, 2)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		total += value
		squared += value * value
	}

	mean := total / count
	variance := squared/count - mean*mean

	requireClose(t, "样本均值", mean, 3, 0.15)
	requireClose(t, "样本标准差", math.Sqrt(variance), 2, 0.15)
}

func TestSampleNormalDeterministic(t *testing.T) {
	sampler, err := NewSampler(constantSource{value: 0.5})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	value, err := sampler.SampleNormal(0, 1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// 两个均匀值都是 0.5 时，Box-Muller 给出 -√(2ln2)
	requireClose(t, "样本", value, -math.Sqrt(2*math.Log(2)), 1e-12)
}

func TestSampleNormalErrors(t *testing.T) {
	sampler := NewDefaultSampler(1)

	tests := []struct {
		name   string
		mean   float64
		stdDev float64
		want   error
	}{
		{"零标准差", 0, 0, ErrInvalidStdDev},
		{"负标准差", 0, -1, ErrInvalidStdDev},
		{"NaN 标准差", 0, math.NaN(), ErrInvalidStdDev},
		{"NaN 均值", math.NaN(), 1, ErrNonFinite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := sampler.SampleNormal(tt.mean, tt.stdDev); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
