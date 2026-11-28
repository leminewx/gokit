package serial

import (
	"net"
)

// addr 实现 net.Addr 接口
type addr string

var _ net.Addr = addr("")

func (own addr) Network() string { return "serial" }

func (own addr) String() string { return string(own) }

// ---------------------------------------------------------------------------------------------------------------------

var ErrTimeout = &timeoutError{}

// timeoutError 实现 net.Error 接口
type timeoutError struct{}

var _ net.Error = (*timeoutError)(nil)

func (timeoutError) Error() string { return "gokit.nets.serial: i/o timeout" }

func (timeoutError) Timeout() bool { return true }

func (timeoutError) Temporary() bool { return true }
