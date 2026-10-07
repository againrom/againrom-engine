package ui

import "testing"

func TestGroupDigits(t *testing.T) {
	for n, want := range map[int64]string{
		0: "0", 7: "7", 999: "999", 1000: "1,000", 534000: "534,000",
		4234782: "4,234,782", -999: "-999", -1000: "-1,000", -4234782: "-4,234,782",
	} {
		if got := GroupDigits(n); got != want {
			t.Errorf("GroupDigits(%d) = %q, want %q", n, got, want)
		}
	}
}
