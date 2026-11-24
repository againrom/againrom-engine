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
		{"extended, second game", OpsSecond, true, Extended},
	}
	for _, c := range cases {
		if got := For(c.ops, c.extended); got != c.want {
			t.Errorf("%s: For = %+v, want %+v", c.name, got, c.want)
		}
	}
}
