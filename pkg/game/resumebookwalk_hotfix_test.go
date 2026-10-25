package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A SAV LOAD keeps each stored range byte until the first live tick; that
// tick walks the books of the player's actors, once, and leaves another
// player's book as stored (MAGIC-420).
func TestFirstTickAfterSAVLoadWalksThePlayerBooks(t *testing.T) {
	mage := func(id sim.EntityID, owner uint32, x int32) sim.Entity {
		e := sim.Entity{ID: id, X: x, Y: 3, Class: 1, HP: 100, MaxHP: 100, Owner: owner, TokenSize: 1,
			Mind: 26, Mana: 200, MaxMana: 200, KnownSpells: 1 << 26, ScanRange: 4,
			AttackCharge: 1, AttackRelax: 1}
		e.Skill[5] = 10
		e.Book.State = sim.BookPresent
		e.Book.Slots[25] = sim.BookSpell{Range: 1, ManaCost: 60}
		return e
	}
	w, err := sim.NewStockedSpelledWorld(0x7e20, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical,
		sim.Terrain{Block: make([]byte, 16*16)}, []sim.Entity{mage(1, sim.SelfSlot, 2), mage(2, 2, 12)},
		nil, sim.Relations{}, nil, nil, []sim.SpellRule{{ID: 26, ManaCost: 60, School: 5, MaxRange: 1}})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: w, resumeBookWalk: true}
	mw := archerMapWorld(t, w)
	mw.mission = &missionNotices{state: ms}
	ranges := func() (own, other uint8) {
		for _, e := range w.Entities() {
			if e.ID == 1 {
				own = e.Book.Slots[25].Range
			} else {
				other = e.Book.Slots[25].Range
			}
		}
		return own, other
	}
	if own, other := ranges(); own != 1 || other != 1 {
		t.Fatalf("ranges at LOAD %d and %d, want the stored 1 and 1", own, other)
	}
	mw.tickStep(nil, false)
	if own, other := ranges(); own != 3 || other != 1 || ms.resumeBookWalk {
		t.Fatalf("after the first tick: ranges %d and %d, pending %v; want the walked 3, the stored 1, done", own, other, ms.resumeBookWalk)
	}
}
