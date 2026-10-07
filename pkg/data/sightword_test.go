package data

import "testing"

// The sight word is (mind+reaction)*256/25 + 4*256 in 1/256 cell with the whole
// cell byte replaced by the derived range; the fraction survives.
func TestSightWordKeepsTheFraction(t *testing.T) {
	cases := []struct {
		mind, reaction, cells int32
		want                  uint16
	}{
		{30, 23, 6, 0x061e},
		{15, 26, 5, 0x05a3},
		{18, 20, 5, 0x0585},
		{30, 23, 7, 0x071e},
	}
	for _, c := range cases {
		if got := SightWord(c.mind, c.reaction, c.cells); got != c.want {
			t.Errorf("SightWord(%d, %d, %d) = %#x, want %#x", c.mind, c.reaction, c.cells, got, c.want)
		}
	}
}
