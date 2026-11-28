package datum

import (
	"fmt"
	"math"
	"testing"
)

func TestMeridianArcNumerical(t *testing.T) {
	ell := CGCS2000.GetEllipsoid()
	arc := ell.CalculateMeridianArc(40.11618753 * math.Pi / 180)
	fmt.Println("弧长数值积分结果:", arc)
	fmt.Println("椭球名称:", ell.String())
	fmt.Printf("长半轴: %.3f\n", ell.GetMajorRadius())
	fmt.Printf("第一偏心率平方: %.14f\n", ell.GetFirstEccentricitySquared())
}
