package sim

import "testing"

func TestGroupRouteProfilePerGame(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		rules  groupRouteProfile
	}{
		{"first", false, groupRouteProfile{}},
		{"second", true, groupRouteProfile{guardAt18: true, constructsForOlderRegistry: true, noHumanHealArm: true, recordsBlows: true, stampWindow: true, endsTargetlessEngage: true, belowFloorAt18: 20}},
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
			// Cases 1 and 2 run at programme 18 of the second game alone;
			// case 4 runs nowhere.
			p := &w.game.groupRoute
			if p.runsStampWindow(uint8(b)) != (tc.second && b == 18) || p.runsRecentBlow(uint8(b)) {
				t.Errorf("%s game: byte %d runs cases 1 and 2 %v, case 4 %v", tc.name, b, p.runsStampWindow(uint8(b)), p.runsRecentBlow(uint8(b)))
			}
		}
		if rules := w.game.groupRoute; rules != tc.rules {
			t.Errorf("%s game rules = %+v, want %+v", tc.name, rules, tc.rules)
		}
	}
}
