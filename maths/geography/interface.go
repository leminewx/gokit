package geography

type (
	// # Equalter 判断两个坐标是否相同
	Equalter[T any] interface {
		// IsEqualTo 判断两个坐标是否相同
		IsEqualTo(T) bool
	}
)

// 编译期断言：三类坐标均实现 Equalter，便于泛型算法按值判等
var (
	_ Equalter[LLAPoint]  = LLAPoint{}
	_ Equalter[ECEFPoint] = ECEFPoint{}
	_ Equalter[ENUPoint]  = ENUPoint{}
)
