package projection

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.geography.projection: "

var (
	// ErrNorthingOutOfRange 表示投影坐标的北向值超出有效范围
	ErrNorthingOutOfRange = errors.New(_ERROR_PREFIX + "northing out of range")
	// ErrLatitudeOutOfRange 表示纬度超出投影的有效范围
	ErrLatitudeOutOfRange = errors.New(_ERROR_PREFIX + "latitude out of range")
)

// wrapLatitudeError 构造纬度越界的错误，保留可用于 errors.Is 的哨兵错误
func wrapLatitudeError(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrLatitudeOutOfRange}, args...)...)
}

// wrapNorthingError 构造北向值越界的错误，保留可用于 errors.Is 的哨兵错误
func wrapNorthingError(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrNorthingOutOfRange}, args...)...)
}
