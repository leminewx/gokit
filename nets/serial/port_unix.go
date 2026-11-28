//go:build linux || darwin

package serial

import (
	"errors"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// port 表示基于 termios 的 POSIX 串口。
//
// 句柄以 O_NONBLOCK 打开，读写前用 poll 等待就绪，从而同时支持
// 读超时与写超时，且不会长时间占用系统线程。
type port struct {
	fd           int32
	lock         sync.RWMutex
	config       Config
	readTimeout  time.Duration
	writeTimeout time.Duration
	closed       atomic.Bool
}

var _ Port = (*port)(nil)

// open 打开并配置 POSIX 串口。
func open(config Config) (Port, error) {
	fd, err := unix.Open(config.Name, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, wrapError(ErrOpenFailed, "%q: %w", config.Name, err)
	}

	own := &port{fd: int32(fd), config: config, readTimeout: config.ReadTimeout}
	if err := own.applyConfig(config); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	return own, nil
}

// SetConfig 修改串口配置。
func (own *port) SetConfig(config Config) error {
	if own.closed.Load() {
		return os.ErrClosed
	}

	config = config.Normalize()
	if err := config.Validate(); err != nil {
		return err
	}

	return own.applyConfig(config)
}

// SetTimeouts 设置读写超时（<=0 表示无限等待）。
func (own *port) SetTimeouts(readTimeout, writeTimeout time.Duration) error {
	if own.closed.Load() {
		return os.ErrClosed
	}

	if readTimeout < 0 {
		readTimeout = 0
	}
	if writeTimeout < 0 {
		writeTimeout = 0
	}

	own.lock.Lock()
	own.readTimeout, own.writeTimeout = readTimeout, writeTimeout
	own.lock.Unlock()
	return nil
}

// Read 读取数据，超时返回 ErrTimeout。
func (own *port) Read(data []byte) (int, error) {
	if own.closed.Load() {
		return 0, os.ErrClosed
	}
	if len(data) == 0 {
		return 0, nil
	}

	if err := waitReady(int(own.fd), unix.POLLIN, own.readTimeoutValue()); err != nil {
		return 0, err
	}

	for {
		n, err := unix.Read(int(own.fd), data)
		if err == nil {
			return n, nil
		}
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.EAGAIN) {
			return 0, ErrTimeout
		}

		return 0, wrapError(ErrReadFailed, "%w", err)
	}
}

// Write 写入数据，超时返回 ErrTimeout。
func (own *port) Write(data []byte) (int, error) {
	if own.closed.Load() {
		return 0, os.ErrClosed
	}
	if len(data) == 0 {
		return 0, nil
	}

	var deadline time.Time
	if timeout := own.writeTimeoutValue(); timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	written := 0
	for written < len(data) {
		if err := waitReady(int(own.fd), unix.POLLOUT, remainingTimeout(deadline)); err != nil {
			return written, err
		}

		n, err := unix.Write(int(own.fd), data[written:])
		if n > 0 {
			written += n
		}

		switch {
		case err == nil:
		case errors.Is(err, unix.EINTR):
		case errors.Is(err, unix.EAGAIN):
			if !deadline.IsZero() && !time.Now().Before(deadline) {
				return written, ErrTimeout
			}
		default:
			return written, wrapError(ErrWriteFailed, "%w", err)
		}
	}

	return written, nil
}

// Close 关闭串口，重复调用安全。
func (own *port) Close() error {
	if !own.closed.CompareAndSwap(false, true) {
		return nil
	}

	if err := unix.Close(int(own.fd)); err != nil {
		return wrapError(ErrCloseFailed, "%w", err)
	}

	return nil
}

// applyConfig 将配置写入 termios。
func (own *port) applyConfig(config Config) error {
	termios, err := getTermios(int(own.fd))
	if err != nil {
		return wrapError(ErrTermios, "read: %w", err)
	}

	// 配置为原始模式：关闭回显、行缓冲、信号字符与软件流控，由调用方自行解析数据。
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON | unix.IXOFF | unix.IXANY
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB | unix.CSTOPB | unix.CRTSCTS
	termios.Cflag |= unix.CLOCAL | unix.CREAD

	switch config.DataBits {
	case 5:
		termios.Cflag |= unix.CS5
	case 6:
		termios.Cflag |= unix.CS6
	case 7:
		termios.Cflag |= unix.CS7
	case 8:
		termios.Cflag |= unix.CS8
	default:
		return wrapError(ErrInvalidConfig, "data bits must be 5-8, got %d", config.DataBits)
	}

	switch config.StopBits {
	case 1:
	case 2:
		termios.Cflag |= unix.CSTOPB
	default:
		return wrapError(ErrInvalidConfig, "stop bits must be 1 or 2, got %d", config.StopBits)
	}

	switch config.Parity {
	case 'N':
	case 'O':
		termios.Cflag |= unix.PARENB | unix.PARODD
	case 'E':
		termios.Cflag |= unix.PARENB
	default:
		return wrapError(ErrInvalidConfig, "parity must be N, O or E, got %q", config.Parity)
	}

	switch config.FlowControl {
	case FLOW_CONTROL_NONE:
	case FLOW_CONTROL_RTS_CTS:
		termios.Cflag |= unix.CRTSCTS
	case FLOW_CONTROL_XON_XOFF:
		termios.Iflag |= unix.IXON | unix.IXOFF
	case FLOW_CONTROL_DTR_DSR:
		return wrapError(ErrUnsupported, "DTR/DSR flow control on %s", runtime.GOOS)
	default:
		return wrapError(ErrInvalidConfig, "unknown flow control %d", config.FlowControl)
	}

	// 非阻塞读：数据是否就绪由 poll 决定。
	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = 0

	if err := setSpeed(termios, config.Baud); err != nil {
		return err
	}

	if err := setTermios(int(own.fd), termios); err != nil {
		return wrapError(ErrTermios, "set: %w", err)
	}

	own.lock.Lock()
	own.config = config
	own.lock.Unlock()
	return nil
}

// readTimeoutValue 返回当前读超时。
func (own *port) readTimeoutValue() time.Duration {
	own.lock.RLock()
	defer own.lock.RUnlock()
	return own.readTimeout
}

// writeTimeoutValue 返回当前写超时。
func (own *port) writeTimeoutValue() time.Duration {
	own.lock.RLock()
	defer own.lock.RUnlock()
	return own.writeTimeout
}

// waitReady 等待文件描述符就绪。
//
// timeout 为 0 表示无限等待，小于 0 表示已过期。
func waitReady(fd int, events int16, timeout time.Duration) error {
	if timeout < 0 {
		return ErrTimeout
	}

	millis := -1
	if timeout > 0 {
		millis = int(timeout / time.Millisecond)
		if millis == 0 {
			millis = 1
		}
	}

	fds := []unix.PollFd{{Fd: int32(fd), Events: events}}
	for {
		n, err := unix.Poll(fds, millis)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return wrapError(ErrPollFailed, "%w", err)
		}
		if n == 0 {
			return ErrTimeout
		}

		revents := fds[0].Revents
		if revents&unix.POLLNVAL != 0 {
			return os.ErrClosed
		}
		if revents&events != 0 {
			return nil
		}
		if revents&(unix.POLLHUP|unix.POLLERR) != 0 {
			return os.ErrClosed
		}
	}
}

// remainingTimeout 返回距离 deadline 的剩余时间。
func remainingTimeout(deadline time.Time) time.Duration {
	if deadline.IsZero() {
		return 0
	}

	return time.Until(deadline)
}
