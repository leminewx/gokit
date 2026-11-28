package nmea

import "testing"

func TestGGA(t *testing.T) {
	// $GGA,071654.66,4006.97231404,N,11632.90258655,E,4,37,0.610,35.091,M,-7.863,M,1.400,4070*51
	sentence := "$GPGGA,071654.66,4006.97231404,N,11632.90258655,E,4,37,0.610,35.091,M,-7.863,M,1.400,4070*51"
	if gga, err := ParseGGA(sentence); err != nil {
		t.Error(err)
	} else if gga.String() != sentence {
		t.Errorf("%s != %s", gga.String(), sentence)
	}
}
