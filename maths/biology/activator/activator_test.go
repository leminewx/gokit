package activator

import (
	"math"
	"testing"
)

func TestActivateBatch(t *testing.T) {
	t.Parallel()

	inputs := []float64{-2, 0, 3}
	got := ActivateBatch(NewReLU(), inputs)
	want := []float64{0, 0, 3}
	if len(got) != len(want) {
		t.Fatalf("ActivateBatch 长度 = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ActivateBatch[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestActivateBatchEmpty(t *testing.T) {
	t.Parallel()

	got := ActivateBatch(NewSigmoid(), nil)
	if len(got) != 0 {
		t.Errorf("空输入返回长度 = %d, want 0", len(got))
	}
}

func TestActivatorNames(t *testing.T) {
	t.Parallel()

	activators := []Activator{
		NewReLU(),
		NewLeakyReLU(0.01),
		NewSigmoid(),
		NewTanh(),
	}
	seen := make(map[Name]bool, len(activators))
	for _, act := range activators {
		name := act.GetName()
		if name == "" {
			t.Errorf("%T 的 GetName 返回空字符串", act)
			continue
		}
		if seen[name] {
			t.Errorf("激活函数名称重复：%q", name)
		}
		seen[name] = true
	}
}

// 保证浮点比较辅助函数被使用，避免后续调整用例时遗漏
var _ = math.Abs
