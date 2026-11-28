package sampler

import (
	"errors"
	"math"
	"testing"
)

func TestSampleExponentialMean(t *testing.T) {
	sampler := NewDefaultSampler(23)
	const count = 5000

	total := 0.0
	for index := range count {
		value, err := sampler.SampleExponential(2)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		if value < 0 {
			t.Fatalf("第 %d 个样本为负：%g", index, value)
		}

		total += value
	}

	requireClose(t, "样本均值", total/count, 0.5, 0.05)
}

func TestSampleExponentialDeterministic(t *testing.T) {
	sampler, err := NewSampler(constantSource{value: 0.5})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	value, err := sampler.SampleExponential(2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "样本", value, math.Log(2)/2, 1e-12)
}

func TestSampleExponentialErrors(t *testing.T) {
	sampler := NewDefaultSampler(1)

	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := sampler.SampleExponential(rate); !errors.Is(err, ErrInvalidRate) {
			t.Fatalf("rate %g: err = %v, want %v", rate, err, ErrInvalidRate)
		}
	}
}
