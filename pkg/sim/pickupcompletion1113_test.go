package sim

import "testing"

// pickupWorld1113 is pickupWorld1117's own fixture (pickupcompletion1117_test.go),
// reduced to the one actor this fix's regression needs, with an explicit
// empty saved Group registry installed. ImportSavedGroups(nil, nil) is
// SavedGroups' own "explicit legacy native mode is false, imported registry
// is empty" state (savedgroups.go) — the state ANY world continued from an
// original save carries, whether or not this particular actor has yet built
// itself a member of one.
func pickupWorld1113(t *testing.T) *World {
	t.Helper()
	hero := laFighter(0, 1, 17, 20, 20)
	hero.HP, hero.MaxHP = 1000, 1000
	w, err := NewLootWorld(1113, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{},
		[]Entity{hero}, nil, engRel(t, [3]uint32{1, 2, 1}, [3]uint32{2, 1, 2}),
		[]Sack{{X: 20, Y: 20, Gold: 3, Items: []uint16{0x101}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	return w
}

// pickupTransfer1113 is pickupTransfer1117's own two calls, without that
// helper's non-saved assertions: CompleteSackPickup's own commandGroup call
// reroutes to commandSavedGroup the moment w.savedGroups is non-nil
// (group.go), so the fresh command group this leaves the actor in is a
// SavedGroup and not a w.groups record.
func pickupTransfer1113(t *testing.T, w *World) {
	t.Helper()
	if err := w.TakeSack(0, 20, 20); err != nil {
		t.Fatal(err)
	}
	if !w.CompleteSackPickup(0) {
		t.Fatal("gameplay completion refused live actor zero")
	}
	if e := w.entities[0]; e.ActorState != actorStatePickupComplete || e.CommandGroup == 0 {
		t.Fatalf("completion did not mark a pending, grouped actor: %+v", e)
	}
	if w.savedGroupFor(w.entities[0].ID) == nil {
		t.Fatal("completion's own commandGroup call did not reach commandSavedGroup")
	}
}

func TestPickup1113SavedGroupCompletionRoundTrips(t *testing.T) {
	w := pickupWorld1113(t)
	pickupTransfer1113(t, w)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatalf("saved Group pickup completion refused to load: %v", err)
	}
	if e := back.entities[0]; e.ActorState != actorStatePickupComplete {
		t.Fatalf("round trip lost the pending completion: %+v", e)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("round trip changed the world's hash")
	}
}

// TestPickup1113SavedGroupCompletionAdvancesToAcquire is stepPickupCompletions'
// own half of the same defect (pickupcompletion.go): it read groupState,
// which -- like pickupCompletionGroupsFault -- only ever scans w.groups, so
// every saved-Group world answered "no record names this pair" and forced
// every completion to Guard, never to the Acquire state
// TestPickup1117EveryPhaseCompletesBeforeNextAcquirePass pins for the plain
// (non-saved) world one Step after the same call.
func TestPickup1113SavedGroupCompletionAdvancesToAcquire(t *testing.T) {
	w := pickupWorld1113(t)
	pickupTransfer1113(t, w)
	w.stepPickupCompletions()
	if e := w.entities[0]; e.ActorState != actorStateAcquire {
		t.Fatalf("saved Group completion did not advance to acquire: %+v", e)
	}
}
