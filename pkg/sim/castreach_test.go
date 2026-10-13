package sim

import "testing"

// reachWorld puts a mage at (10, 10) with the reach-7 arrow row and one target
// at (tx, 10) whose body is size tokens wide.
func reachWorld(t *testing.T, tx int32, size uint8) *World {
	t.Helper()
	caster := spMage(1, 10, 10, 30, 50, 50, 1<<1)
	caster.Owner = 1
	victim := spEnt(2, tx, 10)
	victim.Owner = 2
	victim.TokenSize = size
	return hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow()}, caster, victim)
}

// TestAUnitCastReachIsTheFootprintGap: a wide target is reached across the gap
// between its edge and the caster, not from its anchor cell (MAGIC-REACH-179).
func TestAUnitCastReachIsTheFootprintGap(t *testing.T) {
	t.Parallel()

	// Anchor 8 cells away, footprint distance 7: inside the reach of 7.
	wide := reachWorld(t, 2, 2)
	if got := wide.BookSpellRefusal(1, 2, 1); got != "" {
		t.Fatalf("a two-wide target at anchor distance 8 (gap distance 7): refusal %q, want none", got)
	}
	if events := spRunCast(wide, spCast(1, 2, 1)); len(events) != 1 {
		t.Fatalf("the admitted cast released %d events, want 1", len(events))
	}
	if got := spAt(t, wide, 2).HP; got >= 100 {
		t.Fatalf("the wide target holds %d health after the release, want a hit", got)
	}

	// The same distance to a one-cell body is out of reach: the control.
	narrow := reachWorld(t, 2, 1)
	if got := narrow.BookSpellRefusal(1, 2, 1); got != refusalTargetOutOfRange {
		t.Fatalf("a one-cell target at distance 8: refusal %q, want %q", got, refusalTargetOutOfRange)
	}

	// One cell farther than the gap allows is out of reach for the wide body.
	far := reachWorld(t, 1, 2)
	if got := far.BookSpellRefusal(1, 2, 1); got != refusalTargetOutOfRange {
		t.Fatalf("a two-wide target at gap distance 8: refusal %q, want %q", got, refusalTargetOutOfRange)
	}
}

// TestACasterAlwaysReachesItself: the self-target bypass holds at reach 0.
func TestACasterAlwaysReachesItself(t *testing.T) {
	t.Parallel()

	a := Entity{ID: 1, X: 3, Y: 3, TokenSize: 1}
	if !(&World{}).withinCastReach(a, a, 0) {
		t.Fatal("the caster is out of reach of itself at reach 0")
	}
	b := Entity{ID: 2, X: 3, Y: 3, TokenSize: 1}
	if (&World{}).withinCastReach(a, b, 0) {
		t.Fatal("another body in the caster's cell is within reach 0")
	}
}

// TestAScrollCastAtAWideTargetStartsAcrossTheFootprintGap: the scroll route
// asks the same reach gate as the book, so the caster starts the wind-up where
// the gap distance is within reach and does not first walk to the anchor.
func TestAScrollCastAtAWideTargetStartsAcrossTheFootprintGap(t *testing.T) {
	t.Parallel()

	rule := SpellRule{ID: 1, ManaCost: 100, School: 1, MaxRange: 3, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}
	reach := int32(spellRangeOf(Rules{}, rule, 60, rangeCast))
	build := func(size uint8) *World {
		caster := spEnt(1, 10, 1)
		caster.AttackCharge, caster.AttackRelax = 4, 2
		victim := spEnt(2, 9-reach, 1)
		victim.TokenSize = size
		w := spWorld(t, 1090, []SpellRule{rule}, caster, victim)
		w.carried[0] = []ItemStack{StackItem(ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}, 1)}
		return w
	}
	wide, narrow := build(2), build(1)
	Step(wide, []Command{UseScroll(1, 0, 2)})
	Step(narrow, []Command{UseScroll(1, 0, 2)})
	Step(wide, nil)
	Step(narrow, nil)
	if c := wide.ScrollCasts(); len(c) != 1 || !c[0].Started {
		t.Fatalf("a wide target at gap distance %d (reach %d): scroll %+v, want a started wind-up", reach, reach, c)
	}
	if c := narrow.ScrollCasts(); len(c) != 1 || c[0].Started {
		t.Fatalf("a one-cell target one cell out of reach: scroll %+v, want an approach", c)
	}
}

// TestTheSecondGameCastReachStaysTheAnchorCells: in the second game a cast at a
// wide target is admitted, released and gated at the anchor-cell distance, as
// before the footprint gap reached the first game (R2-ENGINE-348).
func TestTheSecondGameCastReachStaysTheAnchorCells(t *testing.T) {
	t.Parallel()

	second := func(tx int32, size uint8) *World {
		w := reachWorld(t, tx, size)
		w.rom2 = &rom2ScriptState{}
		return w
	}
	// Anchor distance 8, gap distance 7: out of reach here, in reach in the first game.
	wide := second(2, 2)
	if got := wide.BookSpellRefusal(1, 2, 1); got != refusalTargetOutOfRange {
		t.Fatalf("second game, a two-wide target at anchor distance 8: refusal %q, want %q", got, refusalTargetOutOfRange)
	}
	if events := spRunCast(wide, spCast(1, 2, 1)); len(events) != 0 {
		t.Fatalf("second game: the release at anchor distance 8 produced %d events, want none", len(events))
	}
	// Anchor distance 7: admitted and released, wide or not.
	near := second(3, 2)
	if got := near.BookSpellRefusal(1, 2, 1); got != "" {
		t.Fatalf("second game, a two-wide target at anchor distance 7: refusal %q, want none", got)
	}
	if events := spRunCast(near, spCast(1, 2, 1)); len(events) != 1 {
		t.Fatalf("second game: the admitted cast released %d events, want 1", len(events))
	}
	if got := spAt(t, near, 2).HP; got >= 100 {
		t.Fatalf("second game: the wide target holds %d health after the release, want a hit", got)
	}
	// The gate agrees for the scroll route.
	if (&World{rom2: &rom2ScriptState{}}).withinCastReach(Entity{ID: 1, X: 10, Y: 10, TokenSize: 1}, Entity{ID: 2, X: 2, Y: 10, TokenSize: 2}, 7) {
		t.Fatal("second game: the gate admits anchor distance 8 at reach 7")
	}
}
