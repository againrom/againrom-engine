package sim

import "testing"

func TestBlockedManualTeleportReplacesCurrentAction(t *testing.T) {
	w := manualCastWorld(t)
	if !w.beginBookSpellOnce(0, 2, 1) {
		t.Fatal("predecessor refused")
	}
	w.grid[3*16+6] = 1
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, Spell: 26, X: 6, Y: 3}})
	if len(w.bookCasts) != 1 || w.bookCasts[0].Spell != 26 {
		t.Fatal("blocked Teleport did not replace the prior action", w.bookCasts)
	}
}

func TestBlockedTeleportEmitsBothEndpointsAndSpendsExactlyOnce(t *testing.T) {
	for _, mana := range []int32{59, 60, 100} {
		w := teleportHiddenWorld(t, -1)
		w.entities[0].Mana = mana
		w.grid[3*int(w.bounds.Width)+35] |= blockGround
		var casts []CastEvent
		for tick := 0; tick < 32; tick++ {
			var commands []Command
			if tick == 0 {
				commands = []Command{{Kind: KindCastAt, Entity: 1, X: 35, Y: 3, Spell: 26}}
			}
			events := StepObserved(w, commands)
			casts = append(casts, events...)
		}
		got := spAt(t, w, 1)
		if got.X != 2 || got.Y != 3 {
			t.Fatal("blocked cast relocated", got.X, got.Y)
		}
		wantMana, wantCasts := mana, 0
		if mana >= 60 {
			wantMana -= 60
			wantCasts = 1
		}
		if got.Mana != wantMana || len(casts) != wantCasts {
			t.Fatal("cast cost or presentation count", mana, got.Mana, casts)
		}
		if len(casts) > 0 && (casts[0].FromX != 2 || casts[0].FromY != 3 || casts[0].ToX != 35 || casts[0].ToY != 3) {
			t.Fatal("lost teleport endpoints", casts)
		}
	}
}
