package game

// The far side of the attack seam: one attack order in, one command on the
// queue the orders and the blows already use, and nothing else touched.

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestAnAttackOrderBecomesACommandInTheQueueTheOrdersUse is AC-7.
//
// It is measured against the BLOW seam beside it, because the two differ in
// exactly one thing and that difference is the whole content of this side: an
// attack marks its entity commanded and a blow does not, since an attack is an
// order and a blow is not.
//
// The two ids cross whole and are not looked up. A command naming an entity the
// world does not hold, a victim it does not hold, or the attacker itself is a
// no-op at the arm that applies it, so this side validates none of the three —
// asking the world what it still contains would put a world read on the issuing
// path for an answer the advance already has.
func TestAnAttackOrderBecomesACommandInTheQueueTheOrdersUse(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 1, X: 3, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
		{ID: 2, X: 5, Y: 1, HP: mapload.SpawnHP, MaxHP: mapload.SpawnHP},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: make(map[sim.EntityID]int), commanded: make(map[sim.EntityID]bool)}

	before := w.Hash()
	mw.strike(0, 2)
	mw.strike(1, 2)
	mw.strike(99, 2)  // an attacker the world does not hold
	mw.strike(0, 100) // a victim the world does not hold

	want := []sim.Command{
		{Kind: sim.KindAttack, Entity: 0, X: 2},
		{Kind: sim.KindAttack, Entity: 1, X: 2},
		{Kind: sim.KindAttack, Entity: 99, X: 2},
		{Kind: sim.KindAttack, Entity: 0, X: 100},
	}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Errorf("the queue holds\n %+v\nwant\n %+v", mw.pending, want)
	}

	// The mechanism at this seam: issuing with no advance behind it moves no
	// world field and no digest.
	if got := w.Hash(); got != before {
		t.Errorf("the world hashes %#016x after four orders and %#016x before them", got, before)
	}
	// AND EVERY ATTACKER IS MARKED, which is where this seam parts from the blow
	// seam: an attack is an order, so a unit that took one leaves the script.
	for _, id := range []sim.EntityID{0, 1, 99} {
		if !mw.commanded[id] {
			t.Errorf("entity %d is not marked commanded after an attack order", id)
		}
	}
	if len(mw.commanded) != 3 {
		t.Errorf("%d entity/entities are marked commanded, want the 3 that were ordered", len(mw.commanded))
	}
}

func TestAnAttackOrderReachesTheWorldOnlyThroughAnAdvance(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))

	victim := mw.world.Entities()[1].ID
	mw.strike(uint32(mw.world.Entities()[0].ID), uint32(victim))
	quiet := mw.world.Hash()
	if len(mw.pending) != 1 {
		t.Fatalf("the queue holds %d command(s), want 1", len(mw.pending))
	}
	if mw.world.Hash() != quiet {
		t.Error("the world moved before any advance")
	}

	mw.tick()
	if len(mw.pending) != 0 {
		t.Errorf("the queue still holds %d command(s) after an advance", len(mw.pending))
	}
	// The order is now on the entity. What it costs the victim is the cycle's,
	// and the cycle's length is the class's — so this asserts the ORDER landed
	// rather than a number the far side cannot know.
	var a sim.Entity
	for _, e := range mw.world.Entities() {
		if e.ID == mw.world.Entities()[0].ID {
			a = e
		}
	}
	if !a.HasAttackTarget || a.AttackTarget != victim {
		t.Errorf("entity %d holds victim %d (present %v), want %d",
			a.ID, a.AttackTarget, a.HasAttackTarget, victim)
	}
}
