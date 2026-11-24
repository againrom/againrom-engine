package sim

import "testing"

// orderedRetreatWorld is a player's mage (owner SelfSlot) at (20,20) with a
// hostile gap cells to its right, under the second game's profile when second
// is set, after the explicit withdraw order reached it.
func orderedRetreatWorld(t *testing.T, second, book bool, mana, gap int32) *World {
	t.Helper()
	id := uint16(26)
	if second {
		id = 23
	}
	spell := SpellRule{ID: id, ManaCost: 3, School: 5, MaxRange: 16, Defensive: true}
	if second {
		spell = secondRow(spell)
	}
	self := spMage(1, 20, 20, 30, 60, mana, 0)
	self.Owner, self.HP, self.MaxHP, self.Reach, self.ScanRange = SelfSlot, 100, 100, 1, 5
	if book {
		self.KnownSpells = 1 << id
		self.Book.State = BookPresent
		self.Book.Slots[id-1] = BookSpell{Range: 16, ManaCost: 3}
	}
	foe := withdrawalFighter(2, 2, 20+gap, 20, 100)
	w, err := NewStockedSpelledWorld(31, engBounds, ModeCanonical, Terrain{}, []Entity{self, foe}, nil,
		engRel(t, [3]uint32{SelfSlot, 2, 1}), nil, nil, []SpellRule{spell})
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	commandBeforePass(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	return w
}

// The explicit withdraw order of the second game searches hostiles within 10
// cells: one at 10 makes the unit walk away, one at 11 does not. The first
// game keeps its mover radius of 5.
func TestSecondGameExplicitWithdrawSearchesTenCells(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		gap    int32
		want   bool
	}{
		{"second game at 10", true, 10, true},
		{"second game at 11", true, 11, false},
		{"first game at 5", false, 5, true},
		{"first game at 6", false, 6, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := orderedRetreatWorld(t, tc.second, false, 0, tc.gap)
			for range 3 {
				Step(w, nil)
			}
			e := w.entities[0]
			if got := e.X < 20 || e.HasTarget; got != tc.want {
				t.Fatalf("flees = %t, want %t (x %d, target %t)", got, tc.want, e.X, e.HasTarget)
			}
		})
	}
}

// A player's mage under the second game's explicit withdraw order casts
// Teleport when it books the spell and has the mana, and walks otherwise. The
// first game walks with the same book.
func TestSecondGameExplicitWithdrawCastsTeleport(t *testing.T) {
	for _, tc := range []struct {
		name         string
		second, book bool
		mana         int32
		wantCast     bool
	}{
		{"booked and paid", true, true, 10, true},
		{"mana short", true, true, 2, false},
		{"no book", true, false, 10, false},
		{"first game", false, true, 10, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := orderedRetreatWorld(t, tc.second, tc.book, tc.mana, 1)
			cast := false
			for range 3 {
				Step(w, nil)
				cast = cast || len(w.bookCasts) != 0
			}
			e := w.entities[0]
			if cast != tc.wantCast {
				t.Fatalf("casts = %t, want %t (target %d,%d,%t)", cast, tc.wantCast, e.TargetX, e.TargetY, e.HasTarget)
			}
			if !tc.wantCast && !e.HasTarget && e.X >= 20 {
				t.Fatalf("no cast and no walk: x %d target %t", e.X, e.HasTarget)
			}
		})
	}
}
