package serial

import (
	"testing"
)

func TestAddr(t *testing.T) {
	var target addr = "/dev/ttyS0"

	if got := target.Network(); got != "serial" {
		t.Fatalf("Network() = %q, want %q", got, "serial")
	}
	if got := target.String(); got != "/dev/ttyS0" {
		t.Fatalf("String() = %q, want %q", got, "/dev/ttyS0")
	}
}

func TestTimeoutError(t *testing.T) {
	if got := ErrTimeout.Error(); got != "gokit.nets.serial: i/o timeout" {
		t.Fatalf("ErrTimeout.Error() = %q", got)
	}
	if !ErrTimeout.Timeout() {
		t.Fatal("ErrTimeout.Timeout() 应为 true")
	}
	if !ErrTimeout.Temporary() {
		t.Fatal("ErrTimeout.Temporary() 应为 true")
	}
}
