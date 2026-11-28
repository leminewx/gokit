//go:build !windows && !linux && !darwin

package serial

// open 在不支持串口的平台上直接返回 ErrUnsupported。
func open(config Config) (Port, error) {
	return nil, ErrUnsupported
}
