package sim

import "testing"

func TestGroupRouteProfilePerGame(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		rules  groupRouteProfile
	}{
		{"first", false, groupRouteProfile{guardOverMembers: true, humanHealArm: true, engagedFoesClipped: true}},
		{"second", true, groupRouteProfile{constructsForOlderRegistry: true, heldCircle: true}},
	} {
		w := guardRouteWorld(t, tc.second, 0)
		p := w.groupRoute()
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
		rules := *p
		rules.routes = [256]uint8{}
		if rules != tc.rules {
			t.Errorf("%s game rules = %+v, want %+v", tc.name, rules, tc.rules)
		}
	}
}
