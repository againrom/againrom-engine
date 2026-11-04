package sim

import "testing"

func TestGroupRouteProfilePerGame(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		rules  groupRouteProfile
	}{
		{"first", false, groupRouteProfile{}},
		{"second", true, groupRouteProfile{guardAt18: true, constructsForOlderRegistry: true, noHumanHealArm: true, strikeCases: true}},
	} {
		w := guardRouteWorld(t, tc.second, 0)
		for b := 0; b < 256; b++ {
			want := uint8(b)
			if tc.second && b == 18 {
				want = 1
			}
			var g SavedGroup
			g.AI[0x20] = uint8(b)
			if got := w.savedRoute(&g); got != want {
				t.Errorf("%s game: byte %d routes to %d, want %d", tc.name, b, got, want)
			}
		}
		if rules := w.game.groupRoute; rules != tc.rules {
			t.Errorf("%s game rules = %+v, want %+v", tc.name, rules, tc.rules)
		}
	}
}
