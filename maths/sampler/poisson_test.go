package sampler

import (
	"errors"
	"math"
	"testing"
)

func TestSamplePoissonExactMean(t *testing.T) {
	sampler := NewDefaultSampler(29)
	const count = 3000

	total := 0
	for index := range count {
		value, err := sampler.SamplePoisson(4)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		if value < 0 {
			t.Fatalf("第 %d 个样本为负：%d", index, value)
		}

		total += value
	}

	requireClose(t, "样本均值", float64(total)/count, 4, 0.2)
}

func TestSamplePoissonLargeMean(t *testing.T) {
	sampler := NewDefaultSampler(31)
	const count = 2000

	total := 0
	for index := range count {
		value, err := sampler.SamplePoisson(200)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		if value < 0 {
			t.Fatalf("第 %d 个样本为负：%d", index, value)
		}

		total += value
	}

	requireClose(t, "样本均值", float64(total)/count, 200, 5)
}

func TestSamplePoissonDeterministic(t *testing.T) {
	sampler, err := NewSampler(constantSource{value: 0.5})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	// 均值 1 时阈值 e⁻¹ ≈ 0.3679，乘积 0.5、0.25 后停止，返回 1
	value, err := sampler.SamplePoisson(1)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if value != 1 {
		t.Fatalf("样本 = %d, want 1", value)
	}
}

func TestSamplePoissonErrors(t *testing.T) {
	sampler := NewDefaultSampler(1)

	for _, mean := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := sampler.SamplePoisson(mean); !errors.Is(err, ErrInvalidMean) {
			t.Fatalf("mean %g: err = %v, want %v", mean, err, ErrInvalidMean)
		}
	}
}
