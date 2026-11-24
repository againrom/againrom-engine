package sim

import (
	"reflect"
	"testing"
)

// The second game's selector keeps a side target only while 256 x edge
// distance + turn is within 256 x the spell's range byte; the first game's
// selector does not read the range (R2-ENGINE-1766).
func TestPrismaticSideTargetsStayWithinTheRangeBound(t *testing.T) {
	rule := prismaticTestRule
	rule.MaxRange = 3
	build := func(second bool) *World {
		w := prismaticWorld(t, 100, // cap 7
			prismaticFoeAt(2, 12, 1),
			prismaticFoeAt(3, 3, 1), // edge 2, ahead of the primary
			prismaticFoeAt(4, 1, 8), // edge 7
			prismaticFoeAt(5, 1, 4), // edge 3, turned away from the primary
		)
		if second {
			w.game = newGameProfile(ScriptROM2)
		}
		return w
	}
	victims := func(w *World) []EntityID {
		var out []EntityID
		for _, i := range w.prismaticVictims(0, 1, rule, 100) {
			out = append(out, w.entities[i].ID)
		}
		return out
	}
	if got, want := victims(build(true)), []EntityID{2, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("second game: victims %v, want the primary and the foe inside the bound %v", got, want)
	}
	if got, want := victims(build(false)), []EntityID{2, 3, 5, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("first game: victims %v, want every ranked foe %v", got, want)
	}
}
