package sim

import "testing"

// staffFireArrow and shamanLightning are synthetic damage rows with the
// shipped Max Range of Fire Arrow (7) and Lightning (8); staffDrain has Drain
// Life's 3.
func staffFireArrow() SpellRule { return wpnRule(6, 6, 7) }

func shamanLightning() SpellRule {
	return wpnRule(6, 6, 8)
}

func staffDrain() SpellRule {
	return SpellRule{ID: 11, School: 1, MaxRange: 3, DamageMin: 3, DamageMax: 3, TargetsUnit: true}
}

type staffStop struct {
	edge     uint8
	cells    int64
	released bool
}

// staffRun puts a mage holding a weapon of the given reach and carrying rule at
// (2, 8) and a one-cell victim at (14, 8), 12 cells away, and runs
// until the first release. ordered issues the player's attack order; without
// it the caster is a computer unit that finds the victim itself.
func staffRun(t *testing.T, rule SpellRule, reach uint8, ordered, second bool, edit func(*Entity)) staffStop {
	t.Helper()
	caster := wpnCaster(1, 2, 8, rule.ID, 30, 1, 0)
	caster.Reach, caster.Speed = reach, 40
	caster.TypeID = HeroTypeID(true, false)
	victim := spEnt(2, 14, 8)
	caster.Owner, victim.Owner = 2, 1
	if ordered {
		caster.Owner, victim.Owner = SelfSlot, 2
	}
	if edit != nil {
		edit(&caster)
	}
	// Only the caster's side is hostile, so the victim stands.
	rel := engRel(t, [3]uint32{caster.Owner, victim.Owner, relationHostile})
	w := hlWorld(t, 5, rel, []SpellRule{rule}, caster, victim)
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	var cmds []Command
	if ordered {
		cmds = []Command{cbOrder(1, 2)}
	}
	for i := 0; i < 600; i++ {
		Step(w, cmds)
		cmds = nil
		if spAt(t, w, 2).HP < 100 {
			a, v := spAt(t, w, 1), spAt(t, w, 2)
			return staffStop{edgeDistance(&a, &v), (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: v.X, y: v.Y}), true}
		}
	}
	a, v := spAt(t, w, 1), spAt(t, w, 2)
	return staffStop{edgeDistance(&a, &v), (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: v.X, y: v.Y}), false}
}

// TestASpellWeaponMageCastsAtTheWeaponReach: a first-game Staff (reach 5) or
// Shaman Staff (reach 4) caster stops and casts at its weapon reach, not at
// the spell's Max Range (MAGIC-409, MAGIC-411). The second game is the loss
// control: the same fixture there stops at the spell's Max Range.
func TestASpellWeaponMageCastsAtTheWeaponReach(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		rule  SpellRule
		reach uint8
	}{
		{"Staff with Fire Arrow", staffFireArrow(), 5},
		{"Shaman Staff with Lightning", shamanLightning(), 4},
	} {
		got := staffRun(t, tc.rule, tc.reach, true, false, nil)
		if !got.released || got.edge != tc.reach {
			t.Errorf("%s: released %v at edge distance %d, want a release at the weapon reach %d",
				tc.name, got.released, got.edge, tc.reach)
		}
		control := staffRun(t, tc.rule, tc.reach, true, true, nil)
		if !control.released || control.cells != int64(tc.rule.MaxRange) {
			t.Errorf("%s, second game: released %v at %d cells, want a release at Max Range %d",
				tc.name, control.released, control.cells, tc.rule.MaxRange)
		}
	}
}

// TestASpellWeaponReachAboveTheSpellRangeStillCasts: Drain Life (Max Range 3)
// on a Staff (reach 5) casts at 5. A minimum of the two would stop at 3
// (MAGIC-409 excludes it); the second game keeps 3.
func TestASpellWeaponReachAboveTheSpellRangeStillCasts(t *testing.T) {
	t.Parallel()
	got := staffRun(t, staffDrain(), 5, true, false, nil)
	if !got.released || got.edge != 5 {
		t.Errorf("released %v at edge distance %d, want a release at the weapon reach 5", got.released, got.edge)
	}
	control := staffRun(t, staffDrain(), 5, true, true, nil)
	if !control.released || control.cells != 3 {
		t.Errorf("second game: released %v at %d cells, want a release at Max Range 3", control.released, control.cells)
	}
}

// TestAComputerSpellWeaponMageCastsAtTheWeaponReach: a computer caster that
// finds its victim itself takes the same weapon path (MAGIC-409, AI-678).
func TestAComputerSpellWeaponMageCastsAtTheWeaponReach(t *testing.T) {
	t.Parallel()
	got := staffRun(t, staffFireArrow(), 5, false, false, nil)
	if !got.released || got.edge != 5 {
		t.Errorf("released %v at edge distance %d, want a release at the weapon reach 5", got.released, got.edge)
	}
	control := staffRun(t, staffFireArrow(), 5, false, true, nil)
	if !control.released || control.cells != 7 {
		t.Errorf("second game: released %v at %d cells, want a release at Max Range 7", control.released, control.cells)
	}
}

// TestAFemaleAndAMaleSpellWeaponMageStopAlike: the stop reads no sex (AI-678).
func TestAFemaleAndAMaleSpellWeaponMageStopAlike(t *testing.T) {
	t.Parallel()
	male := staffRun(t, staffFireArrow(), 5, true, false, nil)
	female := staffRun(t, staffFireArrow(), 5, true, false, func(e *Entity) { e.TypeID = HeroTypeID(true, true) })
	if male != female || !male.released || male.edge != 5 {
		t.Errorf("male %+v, female %+v, want one release at the weapon reach 5", male, female)
	}
}

// TestTheWeaponCastReleaseTestsTheWeaponReach: a first-game wind-up ending
// with the victim one cell past the reach releases nothing, though the spell's
// Max Range of 7 covers it; at the reach it releases. The second game's
// release at Max Range is TestASpellWeaponMageCastsAtTheWeaponReach's control.
func TestTheWeaponCastReleaseTestsTheWeaponReach(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		x   int32
		hit bool
	}{{5, true}, {6, false}} {
		caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
		caster.Reach = 5
		w := spWorld(t, 1, []SpellRule{staffFireArrow()}, caster, spEnt(2, tc.x, 0))
		w.entities[0].AttackTarget, w.entities[0].HasAttackTarget = 2, true
		w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCasting, 1
		Step(w, nil)
		if hit := spAt(t, w, 2).HP < 100; hit != tc.hit {
			t.Errorf("the release at edge distance %d hit %v, want %v", tc.x, hit, tc.hit)
		}
	}
}

// TestABookCastKeepsTheSpellRange: a mage holding a reach-5 spell staff still
// casts its book Fire Arrow at range 7 (AI-678 keeps Spell+9 for the book).
func TestABookCastKeepsTheSpellRange(t *testing.T) {
	t.Parallel()
	w := reachWorld(t, 3, 1)
	i := indexOfEntity(w.entities, 1)
	w.entities[i].WeaponSpell, w.entities[i].WeaponSpellLevel, w.entities[i].Reach = 1, 30, 5
	if got := w.BookSpellRefusal(1, 2, 1); got != "" {
		t.Fatalf("a book cast at distance 7: refusal %q, want none", got)
	}
	if events := spRunCast(w, spCast(1, 2, 1)); len(events) != 1 {
		t.Fatalf("the book cast at distance 7 released %d events, want 1", len(events))
	}
	if a := spAt(t, w, 1); a.X != 10 || a.Y != 10 {
		t.Fatalf("the book caster moved to (%d, %d), want it to cast from (10, 10)", a.X, a.Y)
	}
	if spAt(t, w, 2).HP >= 100 {
		t.Fatal("the book cast at distance 7 did not hit")
	}
}
