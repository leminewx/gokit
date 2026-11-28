package nets

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.nets: "

var (
	// ErrInvalidCIDR 表示 CIDR 表达式无法解析
	ErrInvalidCIDR = errors.New(_ERROR_PREFIX + "invalid CIDR")
	// ErrIPv4Only 表示传入的网段不是 IPv4
	ErrIPv4Only = errors.New(_ERROR_PREFIX + "only IPv4 is supported")
	// ErrInvalidMask 表示网段掩码非法
	ErrInvalidMask = errors.New(_ERROR_PREFIX + "invalid mask size")
	// ErrNoValidIP 表示网卡上不存在可用 IP 地址
	ErrNoValidIP = errors.New(_ERROR_PREFIX + "no valid IP address found")
	// ErrInvalidPreference 表示 IP 偏好取值非法
	ErrInvalidPreference = errors.New(_ERROR_PREFIX + "invalid IP preference")
	// ErrTooManyAddresses 表示网段地址数量超过生成上限
	ErrTooManyAddresses = errors.New(_ERROR_PREFIX + "too many addresses to generate")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
