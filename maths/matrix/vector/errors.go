package vector

import "errors"

const _ERROR_PREFIX = "gokit.maths.matrix.vector: "

var (
	// ErrShapeNotMatch 表示两个向量的形状不同
	ErrShapeNotMatch = errors.New(_ERROR_PREFIX + "vectors must have the same shape")
	// ErrIndexOutOfRange 表示向量索引越界或索引个数不合法
	ErrIndexOutOfRange = errors.New(_ERROR_PREFIX + "vector index out of range")
	// ErrOneDimensionalDotMeaningless 表示一维向量叉积无定义
	ErrOneDimensionalDotMeaningless = errors.New(_ERROR_PREFIX + "dot product of one-dimensional vectors is undefined")
)
