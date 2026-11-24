package sim

import "testing"

// A second-game flee whose Teleport cast the cell admission refuses leaves the
// caster where it stands: the original issues the cast with no cell test and a
// refused landing keeps the mana and the position (R2-ENGINE-1498).
func TestARefusedFleeTeleportLeavesTheCasterInPlace(t *testing.T) {
	for _, tc := range []struct {
		name         string
		wimpy, withd int32
	}{
		{"flee", 30, 0},
		{"back-off", 0, 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := teleportWorld(t, true, true, 10, 30, tc.withd, tc.wimpy)
			// A landing cell outside the booked reach is refused by the cast.
			w.entities[0].Book.Slots[22] = BookSpell{Range: 1, ManaCost: 3}
			w.spells[0].MaxRange = 1
			regWithdrawalPass(w, nil)
			got := w.entities[0]
			if len(w.bookCasts) != 0 || got.Mana != 10 || got.HasTarget || got.X != 20 || got.Y != 20 {
				t.Fatalf("casts %+v mana %d target (%d,%d,%t) at (%d,%d), want no cast, no walk, mana kept",
					w.bookCasts, got.Mana, got.TargetX, got.TargetY, got.HasTarget, got.X, got.Y)
			}
		})
	}
}
