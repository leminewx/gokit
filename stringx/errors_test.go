package stringx

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")

	withArgs := wrapError(sentinel, "index %d (%q)", 2, "abc")
	if !errors.Is(withArgs, sentinel) {
		t.Errorf("wrapError(带参数) 未保留哨兵错误链: %v", withArgs)
	}
	if want := `sentinel: index 2 ("abc")`; withArgs.Error() != want {
		t.Errorf("wrapError(带参数) = %q, want %q", withArgs.Error(), want)
	}

	noArgs := wrapError(sentinel, "detail only")
	if !errors.Is(noArgs, sentinel) {
		t.Errorf("wrapError(无参数) 未保留哨兵错误链: %v", noArgs)
	}
	if want := "sentinel: detail only"; noArgs.Error() != want {
		t.Errorf("wrapError(无参数) = %q, want %q", noArgs.Error(), want)
	}
}

func TestErrInvalidNumberPrefix(t *testing.T) {
	t.Parallel()

	if want := "gokit.stringx: "; !strings.HasPrefix(ErrInvalidNumber.Error(), want) {
		t.Errorf("ErrInvalidNumber = %q, want 前缀 %q", ErrInvalidNumber.Error(), want)
	}
}
