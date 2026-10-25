package sim

import "testing"

// A book that kept a stale range byte through LOAD takes the range of its
// owner's current power once the player's books are walked; another player's
// book keeps its byte (MAGIC-420).
func TestPlayerBookWalkGivesTeleportTheCurrentPowerRange(t *testing.T) {
	rule := SpellRule{ID: 26, ManaCost: 60, School: 5, MaxRange: 1}
	stale := func(id EntityID, owner uint32, x int32) Entity {
		e := spMage(id, x, 8, 26, 200, 200, 1<<26)
		e.Owner, e.Skill[5] = owner, 10
		e.Book.State = BookPresent
		e.Book.Slots[25] = BookSpell{Range: 1, ManaCost: 60}
		return e
	}
	w := spWorld(t, 91, []SpellRule{rule}, stale(1, SelfSlot, 4), stale(2, 2, 12))
	if got := SpellCharacteristicsFor(Rules{}, w.entities[0], w.spells[0]).Range; got != 1 {
		t.Fatalf("stored Teleport range %d, want 1", got)
	}
	if walked := w.RefreshPlayerBooks(SelfSlot); walked != 1 {
		t.Fatalf("walked %d books, want 1", walked)
	}
	if got := SpellCharacteristicsFor(Rules{}, w.entities[0], w.spells[0]).Range; got != 3 {
		t.Fatalf("Teleport range after the walk %d, want 1 + 6/3 = 3", got)
	}
	if got := w.entities[1].Book.Slots[25].Range; got != 1 {
		t.Fatalf("another player's Teleport range %d, want its stored 1", got)
	}
}
