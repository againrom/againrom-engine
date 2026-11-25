package scriptprofile

import "testing"

// TestForSelectsOneTable pins For to the three profiles: the Againrom record
// wins over the operation vocabulary, and a map without it takes its game's.
func TestForSelectsOneTable(t *testing.T) {
	cases := []struct {
		name     string
		ops      Ops
		extended bool
		want     Profile
	}{
		{"first game", OpsFirst, false, First},
		{"second game", OpsSecond, false, Second},
		{"extended, first game", OpsFirst, true, Extended},
		{"extended, second game", OpsSecond, true, Second.onExtendedMachine()},
	}
	if x := For(OpsSecond, true); x.Ops != OpsSecond || x.Omit != Second.Omit || x.Units != Second.Units || x.Pairs != Extended.Pairs ||
		x.Variables != Extended.Variables || x.Registers != 0 || !x.Extended || !x.Groups {
		t.Errorf("the second game on the extended machine: %+v", x)
	}
	for _, c := range cases {
		if got := For(c.ops, c.extended); got != c.want {
			t.Errorf("%s: For = %+v, want %+v", c.name, got, c.want)
		}
	}
}
