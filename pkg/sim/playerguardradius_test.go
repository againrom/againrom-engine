package sim

import "testing"

// R2-ENGINE-1621: the second game's player guard takes its radius from the
// members' spread, the widest distance to their mean cell plus ScanRange, at
// least MinimalGuardRange 8, whatever the member count. The first game's
// guard issue gives the same radius.
func TestPlayerGuardRadiusFollowsSpreadNotCount(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			xs   []int32
			want uint8
		}{
			{"lone unit", []int32{30}, 8},
			{"two members", []int32{30, 40}, 10},
			{"three members", []int32{30, 35, 40}, 10},
		} {
			var ents []Entity
			var members []int
			for k, x := range tc.xs {
				e := asFighter(asHero+EntityID(k), SelfSlot, x, 40)
				e.ScanRange = 5
				ents = append(ents, e)
				members = append(members, k)
			}
			w := asWorld(t, second, ents...)
			w.commandStance(members, OrderGuard)
			g := w.savedGroupFor(asHero)
			if g == nil || g.AI[0x2d] != tc.want || g.AI[0x38] != tc.want {
				t.Fatalf("second=%v %s: group %v, want radius and base %d", second, tc.name, g != nil, tc.want)
			}
		}
	}
}
