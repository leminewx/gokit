package geography

import "errors"

const _ERROR_PREFIX = "gokit.maths.geography: "

var (
	// ErrDatumNotMatched 表示两个坐标点的地理基准面不匹配
	ErrDatumNotMatched = errors.New(_ERROR_PREFIX + "geographic datum mismatch")
	// ErrVincentyNotConverged 表示 Vincenty 反算未收敛
	ErrVincentyNotConverged = errors.New(_ERROR_PREFIX + "vincenty calculation did not converge")
	// ErrPolygonLessThan3Points 表示多边形少于 3 个坐标点
	ErrPolygonLessThan3Points = errors.New(_ERROR_PREFIX + "polygon has fewer than 3 points")
)
