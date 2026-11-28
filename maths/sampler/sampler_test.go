package sampler

import (
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

// requireClose 断言两个浮点值在给定容差内相等
func requireClose(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()

	if math.IsNaN(got) || math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.15g, want %.15g (±%g)", label, got, want, tolerance)
	}
}

// constantSource 每次返回固定值的随机源，用于断言确定结果
type constantSource struct {
	value float64
}

func (own constantSource) GetFloat64() float64 {
	return own.value
}

func TestErrorPrefix(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"nil 随机源", ErrNilSource},
		{"nil 生成器", ErrNilRandom},
		{"随机源越界", ErrInvalidSource},
		{"非法区间", ErrInvalidRange},
		{"非法标准差", ErrInvalidStdDev},
		{"非法速率", ErrInvalidRate},
		{"非法均值", ErrInvalidMean},
		{"非法数量", ErrInvalidCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.err.Error(), _ERROR_PREFIX) {
				t.Fatalf("%v 缺少错误前缀", tt.err)
			}
		})
	}
}

func TestNewSampler(t *testing.T) {
	if _, err := NewSampler(nil); !errors.Is(err, ErrNilSource) {
		t.Fatalf("err = %v, want %v", err, ErrNilSource)
	}

	sampler, err := NewSampler(constantSource{value: 0.5})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if sampler == nil {
		t.Fatalf("sampler 为 nil")
	}
}

func TestNewSource(t *testing.T) {
	if _, err := NewSource(nil); !errors.Is(err, ErrNilRandom) {
		t.Fatalf("err = %v, want %v", err, ErrNilRandom)
	}

	source, err := NewSource(rand.New(rand.NewPCG(1, 2)))
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for range 16 {
		value := source.GetFloat64()
		if value < 0 || value >= 1 {
			t.Fatalf("随机值 %g 超出 [0, 1)", value)
		}
	}
}

func TestNewDefaultSamplerReproducible(t *testing.T) {
	first := NewDefaultSampler(7)
	second := NewDefaultSampler(7)

	for index := range 8 {
		left, err := first.SampleUniform(0, 1)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		right, err := second.SampleUniform(0, 1)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		if left != right {
			t.Fatalf("第 %d 个样本不可复现：%g != %g", index, left, right)
		}
	}
}

func TestSampleUniform(t *testing.T) {
	sampler := NewDefaultSampler(11)

	total := 0.0
	for index := range 1000 {
		value, err := sampler.SampleUniform(0, 10)
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		if value < 0 || value >= 10 {
			t.Fatalf("第 %d 个样本 %g 超出 [0, 10)", index, value)
		}

		total += value
	}

	requireClose(t, "样本均值", total/1000, 5, 0.5)
}

func TestSampleUniformCustomSource(t *testing.T) {
	sampler, err := NewSampler(constantSource{value: 0.25})
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	value, err := sampler.SampleUniform(0, 8)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	requireClose(t, "样本", value, 2, 1e-12)
}

func TestSampleUniformErrors(t *testing.T) {
	sampler := NewDefaultSampler(1)

	tests := []struct {
		name  string
		lower float64
		upper float64
	}{
		{"区间退化", 1, 1},
		{"区间反向", 2, 1},
		{"下界非有限", math.Inf(-1), 1},
		{"上界 NaN", 0, math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := sampler.SampleUniform(tt.lower, tt.upper); !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidRange)
			}
		})
	}
}

func TestSampleInvalidSource(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"负值", -0.1},
		{"达到 1", 1},
		{"超过 1", 1.5},
		{"NaN", math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler, err := NewSampler(constantSource{value: tt.value})
			if err != nil {
				t.Fatalf("err = %v", err)
			}

			if _, err := sampler.SampleUniform(0, 1); !errors.Is(err, ErrInvalidSource) {
				t.Fatalf("err = %v, want %v", err, ErrInvalidSource)
			}
		})
	}
}

func TestShuffleIndices(t *testing.T) {
	sampler := NewDefaultSampler(3)

	indices, err := sampler.ShuffleIndices(10)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if len(indices) != 10 {
		t.Fatalf("len = %d, want 10", len(indices))
	}

	seen := make([]bool, 10)
	for _, index := range indices {
		if index < 0 || index >= 10 || seen[index] {
			t.Fatalf("结果不是排列：%v", indices)
		}

		seen[index] = true
	}

	replay, err := NewDefaultSampler(3).ShuffleIndices(10)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	for index := range indices {
		if indices[index] != replay[index] {
			t.Fatalf("洗牌结果不可复现：%v != %v", indices, replay)
		}
	}
}

func TestShuffleIndicesErrors(t *testing.T) {
	sampler := NewDefaultSampler(1)

	for _, count := range []int{0, -3} {
		if _, err := sampler.ShuffleIndices(count); !errors.Is(err, ErrInvalidCount) {
			t.Fatalf("count %d: err = %v, want %v", count, err, ErrInvalidCount)
		}
	}
}
