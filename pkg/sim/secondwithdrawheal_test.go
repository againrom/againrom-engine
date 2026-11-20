package sim

import "testing"

// secondWithdrawWorld is a computer fighter of owner 2 at (20,20) and a hero
// gap cells to its right, under the second game's profile when second is set.
func secondWithdrawWorld(t *testing.T, second bool, hp, withdraw, wimpy, gap int32) *World {
	t.Helper()
	self := withdrawalFighter(1, 2, 20, 20, hp)
	self.Withdraw, self.Wimpy = withdraw, wimpy
	hero := withdrawalFighter(2, SelfSlot, 20+gap, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}), self, hero)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// A second-game unit above Withdraw stays; at Withdraw with an enemy within 2
// it backs off three cells; an enemy at 3 does not count.
func TestSecondGameWithdrawIsAbsoluteHealthAtRadiusTwo(t *testing.T) {
	for _, tc := range []struct {
		name         string
		hp, gap      int32
		wantBackedUp bool
	}{
		{"above Withdraw", 41, 2, false},
		{"at Withdraw", 40, 2, true},
		{"at Withdraw, enemy at 3", 40, 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := secondWithdrawWorld(t, true, tc.hp, 40, 0, tc.gap)
			regWithdrawalPass(w, nil)
			got := w.entities[0]
			if got.HasTarget != tc.wantBackedUp || tc.wantBackedUp && (got.TargetX != 17 || got.TargetY != 20) {
				t.Fatalf("target = (%d,%d,%t), want backed up %t to (17,20)", got.TargetX, got.TargetY, got.HasTarget, tc.wantBackedUp)
			}
		})
	}
}

// The second game's flee test looks sight plus 5 cells (here 5+5); the first
// game's looks the mover's 5.
func TestWimpyFleeRadiusByGame(t *testing.T) {
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
			w := secondWithdrawWorld(t, tc.second, 30, 0, 30, tc.gap)
			regWithdrawalPass(w, nil)
			if got := w.entities[0].HasTarget; got != tc.want {
				t.Fatalf("flees = %t, want %t", got, tc.want)
			}
		})
	}
}

// The back-off test runs after the flee test: a unit at or below both and
// beside an enemy flees, which a book of Teleport shows by its distance.
func teleportWorld(t *testing.T, second, book bool, mana, hp, withdraw, wimpy int32) *World {
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
	self.Owner, self.HP, self.MaxHP, self.Withdraw, self.Wimpy, self.Reach, self.ScanRange = 2, hp, 100, withdraw, wimpy, 1, 5
	if book {
		self.KnownSpells = 1 << id
		self.Book.State = BookPresent
		self.Book.Slots[id-1] = BookSpell{Range: 16, ManaCost: 3}
	}
	hero := withdrawalFighter(2, SelfSlot, 21, 20, 100)
	w, err := NewStockedSpelledWorld(31, engBounds, ModeCanonical, Terrain{}, []Entity{self, hero}, nil,
		engRel(t, [3]uint32{2, SelfSlot, 1}), nil, nil, []SpellRule{spell})
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

func TestSecondGameFleeAndBackOffCastTeleportWhenBookedAndPaid(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		second, book              bool
		mana, hp, withdraw, wimpy int32
		wantCastX                 int32 // 0: the unit walks to (17,20)
	}{
		{"flee casts at scan range + 1", true, true, 10, 30, 0, 30, 14},
		{"back-off casts at scan range - 1", true, true, 10, 30, 30, 0, 16},
		{"flee precedes back-off", true, true, 10, 30, 30, 30, 14},
		{"no book walks", true, false, 10, 30, 0, 30, 0},
		{"mana short walks", true, true, 2, 30, 0, 30, 0},
		{"first game walks with the same book", false, true, 10, 30, 0, 30, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := teleportWorld(t, tc.second, tc.book, tc.mana, tc.hp, tc.withdraw, tc.wimpy)
			regWithdrawalPass(w, nil)
			got := w.entities[0]
			if tc.wantCastX == 0 {
				if len(w.bookCasts) != 0 || !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
					t.Fatalf("want a walk to (17,20), got casts %+v target (%d,%d,%t)", w.bookCasts, got.TargetX, got.TargetY, got.HasTarget)
				}
				return
			}
			if len(w.bookCasts) != 1 || !w.bookCasts[0].AtCell || w.bookCasts[0].X != tc.wantCastX || w.bookCasts[0].Y != 20 || got.HasTarget {
				t.Fatalf("want a cast at (%d,20), got casts %+v target (%d,%d,%t)", tc.wantCastX, w.bookCasts, got.TargetX, got.TargetY, got.HasTarget)
			}
		})
	}
}

// A second-game computer Human takes the class of its book, not its mana: a
// mana pool without a book backs off at nothing.
func TestSecondGameClassReadsTheBookNotTheManaPool(t *testing.T) {
	booked := Entity{ID: 1, Owner: 3, X: 2, Y: 2, HP: 100, MaxHP: 100, MaxMana: 50, Mana: 50, Reach: 1, Humanoid: true, KnownSpells: 1 << 24}
	manaOnly := Entity{ID: 2, Owner: 3, X: 4, Y: 2, HP: 100, MaxHP: 100, MaxMana: 50, Mana: 50, Reach: 1, Humanoid: true}
	w := secondWorld(t, 1, Bounds{Width: 16, Height: 16}, nil, booked, manaOnly)
	es := w.Entities()
	if es[0].Withdraw != 100 || es[0].Wimpy != 25 || es[1].Withdraw != 0 || es[1].Wimpy != 0 {
		t.Fatalf("thresholds (book %d/%d, mana only %d/%d), want (100/25, 0/0)", es[0].Withdraw, es[0].Wimpy, es[1].Withdraw, es[1].Wimpy)
	}
}

// The first game's decision is unchanged: the mover's radius of 5, no spell
// and the built thresholds.
func TestFirstGameWithdrawUnchanged(t *testing.T) {
	w := teleportWorld(t, false, true, 10, 30, 30, 30)
	if w.game.fleeSightExtra != 0 || w.game.fleeTeleports {
		t.Fatalf("first game profile carries the second game's escape rules: %+v", w.game)
	}
	regWithdrawalPass(w, nil)
	got := w.entities[0]
	if len(w.bookCasts) != 0 || !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 || got.Withdraw != 30 || got.Wimpy != 30 {
		t.Fatalf("first game: casts %+v target (%d,%d,%t) thresholds %d/%d", w.bookCasts, got.TargetX, got.TargetY, got.HasTarget, got.Withdraw, got.Wimpy)
	}
}

// firstHeal steps the computer-heal world and returns the target of the
// mage's first Heal, or zero.
func firstHeal(w *World, heal uint16) EntityID {
	for range 200 {
		for _, event := range StepObserved(w, nil) {
			if event.Caster == 1 && event.Spell == heal {
				return event.Target
			}
		}
	}
	return 0
}

// A second-game computer mage with Heal in its book heals itself first when
// wounded, then the first wounded unit of Heal's area list within 6 (nearest
// ring first), and a unit without Heal in its book never heals.
func TestSecondGameComputerHealOrderAndBook(t *testing.T) {
	far, near := woundedOf(3, healAlliedOwner, 7), woundedOf(4, healAlliedOwner, 5)
	t.Run("self before others", func(t *testing.T) {
		w, heal := computerHealWorld(t, true, 1, false, far, near)
		w.entities[0].HP = 40
		if got := firstHeal(w, heal); got != 1 {
			t.Fatalf("first Heal went to %d, want the mage itself", got)
		}
	})
	t.Run("nearest wounded within 6", func(t *testing.T) {
		w, heal := computerHealWorld(t, true, 1, false, far, near)
		if got := firstHeal(w, heal); got != 4 {
			t.Fatalf("first Heal went to %d, want 4", got)
		}
	})
	t.Run("seven away is out of reach", func(t *testing.T) {
		w, heal := computerHealWorld(t, true, 1, false, woundedOf(3, healAlliedOwner, 9))
		if got := healTargets(w, heal); got[3] != 0 {
			t.Fatalf("a unit 7 cells away took Heal: %v", got)
		}
	})
	t.Run("no Heal in the book", func(t *testing.T) {
		w, heal := computerHealWorld(t, true, 1, false, near)
		w.entities[0].KnownSpells, w.entities[0].HP = 1<<5, 40
		if got := healTargets(w, heal); len(got) != 0 {
			t.Fatalf("a mage without Heal healed: %v", got)
		}
	})
}
