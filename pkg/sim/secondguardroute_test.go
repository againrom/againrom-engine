package sim

import "testing"

const (
	guardRouteHero EntityID = 1
	guardRouteFoe  EntityID = 2
	guardRouteSlot uint32   = 3
	// guardRouteB is the programme byte the second game's file holds for a
	// computer Group, stated here so the test reads no production constant.
	guardRouteB uint8 = 18
)

// guardRouteWorld is a mission LOAD as the original's file leaves it: the
// hero and a hostile unit two cells apart, the hostile's Group at programme
// byte route, the hero's at Stand Ground. A second-game LOAD reads no member
// order; a first-game one carries the stance setter's.
func guardRouteWorld(t *testing.T, second bool, route uint8) *World {
	t.Helper()
	hero := engFighter(guardRouteHero, SelfSlot, 20, 20)
	foe := engFighter(guardRouteFoe, guardRouteSlot, 22, 20)
	foe.Group = 1
	rel := engRel(t, [3]uint32{guardRouteSlot, SelfSlot, 1}, [3]uint32{SelfSlot, guardRouteSlot, 1})
	var script *Script
	if second {
		script = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, foe}, script, rel)
	if err != nil {
		t.Fatal(err)
	}
	if w.SecondGame() != second {
		t.Fatalf("fixture: SecondGame %v, want %v", w.SecondGame(), second)
	}
	guard := SavedGroup{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: guardRouteSlot},
		Members: []SavedGroupMember{{Archive: 2, Entity: guardRouteFoe, Bound: true}}}
	guard.AI[0x20], guard.AI[0x2d], guard.AI[0x38], guard.AI[0x45] = route, 10, 10, 1
	stand := SavedGroup{ID: 2, Owner: SavedGroupReference{Class: 1, Owner: SelfSlot},
		Members: []SavedGroupMember{{Archive: 1, Entity: guardRouteHero, Bound: true}}}
	stand.AI[0x20], stand.AI[0x45] = orderStandGround, 1
	var orders []SavedActorOrder
	if !second {
		orders = []SavedActorOrder{constructedOrder(laEnt(t, w, guardRouteFoe), uint32(actorStateGuard))}
	}
	if err := w.ImportOriginalGroups([]SavedGroup{guard, stand}, orders); err != nil {
		t.Fatal(err)
	}
	return w
}

// guardRouteAttacks reports whether the hostile holds the hero as its victim
// within four Group decisions.
func guardRouteAttacks(w *World) bool {
	for range 4 {
		engRun(w, 1)
		if victim, ok := engVictim(w, guardRouteFoe); ok && victim == guardRouteHero {
			return true
		}
	}
	return false
}

// A second-game Group at programme byte 18, the second guard handler, takes
// the guard route after a LOAD, and its member attacks the hero next to it
// (R2-ENGINE-507, DIV-4285).
func TestSecondGameGuardRouteAttacksAfterLoad(t *testing.T) {
	w := guardRouteWorld(t, true, guardRouteB)
	if !guardRouteAttacks(w) {
		t.Fatal("the second-game Group at byte 18 never ordered its member against the hero")
	}
	if issues := w.SavedGroupIssues(); len(issues) != 0 {
		t.Fatalf("the loaded registry reports %v", issues)
	}
	if g, _, _ := w.SavedGroups(); g[0].AI[0x20] != guardRouteB {
		t.Fatalf("the dispatch rewrote the programme byte to %d", g[0].AI[0x20])
	}
}

// The first game's Guard Group attacks the hero after a LOAD, and its
// dispatch runs no route for byte 18 (AI-ORDER-010).
func TestFirstGameGuardRouteAttacksAfterLoad(t *testing.T) {
	if !guardRouteAttacks(guardRouteWorld(t, false, orderGuard)) {
		t.Fatal("the first-game Guard Group never ordered its member against the hero")
	}
	if guardRouteAttacks(guardRouteWorld(t, false, guardRouteB)) {
		t.Fatal("the first-game dispatch ran a route for byte 18")
	}
}
