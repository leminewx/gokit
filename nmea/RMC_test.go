package nmea

import "testing"

func TestRMC(t *testing.T) {
	// $RMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,,,D*50
	sentence := "$GPRMC,071654.65,A,4006.97231763,N,11632.90263794,E,14.252,264.800,291125,,,D*50"
	if rmc, err := ParseRMC(sentence); err != nil {
		t.Error(err)
	} else if rmc.String() != sentence {
		t.Errorf("%s != %s", rmc.String(), sentence)
	}
}
