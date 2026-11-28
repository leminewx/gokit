package geography

import (
	"testing"

	"gitee.com/leminewx/gokit/maths/geography/datum"
)

func TestLLAPoint(t *testing.T) {
	t.Log(NewLLAPoint(datum.WGS84, 119.98606765511893, 30.534187452875283, 0))
}
