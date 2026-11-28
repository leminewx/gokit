//go:build linux

package serial

import (
	"golang.org/x/sys/unix"
)

// getTermios 读取 Linux 终端的 termios 配置。
func getTermios(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}

// setTermios 写入 Linux 终端的 termios 配置。
func setTermios(fd int, termios *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}

// setSpeed 设置 Linux 终端的波特率（同时写入 c_cflag 与 ispeed/ospeed）。
func setSpeed(termios *unix.Termios, baud int) error {
	speed, err := linuxSpeed(baud)
	if err != nil {
		return err
	}

	termios.Cflag &^= unix.CBAUD
	termios.Cflag |= speed
	termios.Ispeed = speed
	termios.Ospeed = speed
	return nil
}

// linuxSpeed 将整数波特率映射为 Linux 的 Bxxx 常量。
func linuxSpeed(baud int) (uint32, error) {
	switch baud {
	case 50:
		return unix.B50, nil
	case 75:
		return unix.B75, nil
	case 110:
		return unix.B110, nil
	case 134:
		return unix.B134, nil
	case 150:
		return unix.B150, nil
	case 200:
		return unix.B200, nil
	case 300:
		return unix.B300, nil
	case 600:
		return unix.B600, nil
	case 1200:
		return unix.B1200, nil
	case 1800:
		return unix.B1800, nil
	case 2400:
		return unix.B2400, nil
	case 4800:
		return unix.B4800, nil
	case 9600:
		return unix.B9600, nil
	case 19200:
		return unix.B19200, nil
	case 38400:
		return unix.B38400, nil
	case 57600:
		return unix.B57600, nil
	case 115200:
		return unix.B115200, nil
	case 230400:
		return unix.B230400, nil
	default:
		return 0, wrapError(ErrInvalidConfig, "unsupported baud rate %d on linux", baud)
	}
}
