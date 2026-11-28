package projection

import (
	"fmt"
	"testing"

	"gitee.com/leminewx/gokit/maths/geography"
	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestLocalProjector(t *testing.T) {
	center := geography.NewLLAPoint(datum.CGCS2000, 116.54899394, 40.11618753)
	base := NewLambertCylindricalEqualAreaProjector(datum.WGS84)
	projector := NewLocalProjector(base, center)

	fmt.Println(base.Project(geography.NewLLAPoint(datum.CGCS2000, 116.5487705, 40.11626593)))

	for _, point := range []geography.LLAPoint{
		// test
		geography.NewLLAPoint(datum.CGCS2000, 116.54899394, 40.11618753),

		// truth
		geography.NewLLAPoint(datum.CGCS2000, 116.5487705, 40.11626593),
		geography.NewLLAPoint(datum.CGCS2000, 116.547373, 40.11616953),
		geography.NewLLAPoint(datum.CGCS2000, 116.5473945, 40.11596852),
		geography.NewLLAPoint(datum.CGCS2000, 116.5487839, 40.11605057),
		geography.NewLLAPoint(datum.CGCS2000, 116.5489529, 40.11595621),
		geography.NewLLAPoint(datum.CGCS2000, 116.5490601, 40.1151173),
		geography.NewLLAPoint(datum.CGCS2000, 116.5492345, 40.11513576),
		geography.NewLLAPoint(datum.CGCS2000, 116.5491326, 40.11597467),
		geography.NewLLAPoint(datum.CGCS2000, 116.549264, 40.11609774),
		geography.NewLLAPoint(datum.CGCS2000, 116.5504308, 40.1162126),
		geography.NewLLAPoint(datum.CGCS2000, 116.5503798, 40.11641566),
		geography.NewLLAPoint(datum.CGCS2000, 116.5492399, 40.11631516),
		geography.NewLLAPoint(datum.CGCS2000, 116.5490896, 40.11639515),
		geography.NewLLAPoint(datum.CGCS2000, 116.5489824, 40.11717662),
		geography.NewLLAPoint(datum.CGCS2000, 116.548808, 40.11716226),
		geography.NewLLAPoint(datum.CGCS2000, 116.5488912, 40.11638695),
		geography.NewLLAPoint(datum.CGCS2000, 116.5487705, 40.11626593),
	} {
		p, err := projector.Project(point)
		if err != nil {
			t.Fatal(err)
		}
		x, y := p.GetValues()
		fmt.Println(x, ",", y)
	}
}
