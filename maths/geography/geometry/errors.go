package geometry

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.geography.geometry: "

var (
	// ErrTypeMismatch 表示 GeoJSON 中的 type 与目标几何类型不一致
	ErrTypeMismatch = errors.New(_ERROR_PREFIX + "geometry type mismatch")
	// ErrUnknownType 表示 GeoJSON 中的 type 不受支持
	ErrUnknownType = errors.New(_ERROR_PREFIX + "unknown geometry type")
)

// wrapTypeError 构造类型不匹配的错误，保留可用于 errors.Is 的哨兵错误
func wrapTypeError(got, want string) error {
	return fmt.Errorf("%w: got %q, want %q", ErrTypeMismatch, got, want)
}
