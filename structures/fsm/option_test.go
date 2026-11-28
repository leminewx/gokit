package fsm

import "testing"

func TestDefaultErrorPolicies(t *testing.T) {
	t.Parallel()

	machine := New[int]()
	if !machine.stopOnEventError || !machine.stopOnLeaveError || !machine.stopOnEnterError {
		t.Errorf("默认错误策略 = (%v, %v, %v), want 全为 true",
			machine.stopOnEventError, machine.stopOnLeaveError, machine.stopOnEnterError)
	}
}

func TestWithEventErrorPolicy(t *testing.T) {
	t.Parallel()

	machine := New(WithEventErrorPolicy[int](false))
	if machine.stopOnEventError {
		t.Error("stopOnEventError = true, want false")
	}
	if !machine.stopOnLeaveError || !machine.stopOnEnterError {
		t.Error("WithEventErrorPolicy 不应影响其他策略")
	}
}

func TestWithLeaveErrorPolicy(t *testing.T) {
	t.Parallel()

	machine := New(WithLeaveErrorPolicy[int](false))
	if machine.stopOnLeaveError {
		t.Error("stopOnLeaveError = true, want false")
	}
	if !machine.stopOnEventError || !machine.stopOnEnterError {
		t.Error("WithLeaveErrorPolicy 不应影响其他策略")
	}
}

func TestWithEnterErrorPolicy(t *testing.T) {
	t.Parallel()

	machine := New(WithEnterErrorPolicy[int](false))
	if machine.stopOnEnterError {
		t.Error("stopOnEnterError = true, want false")
	}
	if !machine.stopOnEventError || !machine.stopOnLeaveError {
		t.Error("WithEnterErrorPolicy 不应影响其他策略")
	}
}

func TestWithStopOnError(t *testing.T) {
	t.Parallel()

	machine := New(WithStopOnError[int](false))
	if machine.stopOnEventError || machine.stopOnLeaveError || machine.stopOnEnterError {
		t.Errorf("stopOnError 组合 = (%v, %v, %v), want 全为 false",
			machine.stopOnEventError, machine.stopOnLeaveError, machine.stopOnEnterError)
	}

	// 后应用的选项覆盖先应用的选项
	machine = New(WithStopOnError[int](false), WithEnterErrorPolicy[int](true))
	if !machine.stopOnEnterError {
		t.Error("后应用的 WithEnterErrorPolicy 未生效")
	}
}
