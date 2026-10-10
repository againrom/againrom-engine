package sim

import "testing"

// cbEnt is one entity for these tests: an id and a cell, and nothing else set.
// Everything a blow reads is named per test, so what a case depends on is
// visible in that case rather than in a shared fixture.
// It carries a DYING TIME, and the number is not decoration: with none, a body
// is torn down on the tick it falls and the decay ladder is free to remove it
// from the world in that same tick — so a fixture with no dwell would make every
// test below about the tick a unit dies also a test about the tick it stops
// existing. A dwell of 200 is longer than any run here, so a body felled in one
// of these worlds stays in it, at its first stage, for the whole test.
func cbEnt(id EntityID, x, y int32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200}
}

// cbWorld builds a world over an open 16x16 grid, failing the test rather than
// returning an error, so a case reads as its own statement.
func cbWorld(t *testing.T, seed uint64, ents ...Entity) *World {
	t.Helper()
	w, err := NewWorld(seed, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// cbAt is the world's entry for id, failing when the world holds none.
func cbAt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("world holds no entity %d", id)
	}
	return w.entities[i]
}

// TestTheSixAttackPhasesAreTheWholeSet is the definedness rule as a sweep
// over every byte, not over the six that are expected to pass: what the
// test is for is the bytes nobody thought of, and a table of six would
// assert only that six names exist.
func TestTheSixAttackPhasesAreTheWholeSet(t *testing.T) {
	for v := 0; v < 256; v++ {
		p := AttackPhase(v)
		want := p == AttackReady || p == AttackCharging || p == AttackRelaxing ||
			p == AttackCasting || p == AttackBoundaryOne || p == AttackBoundaryTwo
		if p.defined() != want {
			t.Errorf("AttackPhase(%d).defined() = %v, want %v", v, p.defined(), want)
		}
	}
	if AttackReady != 0 {
		t.Errorf("AttackReady is %d, want the zero value — an entity that names no phase has not begun",
			uint8(AttackReady))
	}
}

func TestAConstructedWorldHoldsNoImpossibleAttackState(t *testing.T) {
	t.Run("a unit that is not alive holds no order", func(t *testing.T) {
		for _, hp := range []int32{0, -1, -1000} {
			a := cbEnt(1, 1, 1)
			a.HP = hp
			a.AttackTarget, a.HasAttackTarget = 2, true
			a.AttackPhase, a.AttackCountdown = AttackCharging, 5
			w := cbWorld(t, 1, a, cbEnt(2, 2, 2))
			if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
				t.Errorf("hp %d: entity 1 kept %v/%d/%d", hp, e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
			}
		}
	})

	t.Run("a victim that is the attacker is normalised away", func(t *testing.T) {
		a := cbEnt(1, 1, 1)
		a.AttackTarget, a.HasAttackTarget = 1, true
		a.AttackPhase, a.AttackCountdown = AttackRelaxing, 3
		w := cbWorld(t, 1, a)
		if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
			t.Errorf("entity 1 kept %v/%d/%d", e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
		}
	})

	// The cycle is checked on the entity AS GIVEN and the unheld victim is
	// normalised after every id is known, so this case names a countdown its own
	// charge admits: a caller that supplies both an impossible cycle and an
	// unheld victim gets the error, not the normalisation.
	t.Run("a victim the world does not hold is normalised away", func(t *testing.T) {
		a := cbEnt(1, 1, 1)
		a.AttackTarget, a.HasAttackTarget = 99, true
		a.AttackCharge = 5
		a.AttackPhase, a.AttackCountdown = AttackCharging, 2
		w := cbWorld(t, 1, a, cbEnt(2, 2, 2))
		if e := cbAt(t, w, 1); e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
			t.Errorf("entity 1 kept %v/%d/%d", e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
		}
	})

	t.Run("a cycle on a unit holding no order is residue and is dropped", func(t *testing.T) {
		a := cbEnt(1, 1, 1)
		a.AttackTarget, a.AttackPhase, a.AttackCountdown = 2, AttackRelaxing, 7
		w := cbWorld(t, 1, a, cbEnt(2, 2, 2))
		e := cbAt(t, w, 1)
		if e.AttackTarget != 0 || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
			t.Errorf("entity 1 kept %d/%d/%d — a victim id under no presence is residue too",
				e.AttackTarget, e.AttackPhase, e.AttackCountdown)
		}
	})

	// The two refusals are asked of an entity that HOLDS an order, because the
	// residue rule above runs first and a cycle field on an entity holding none
	// is dropped before either is looked at. That order is the constructor's own
	// and it is what makes an undefined phase reachable here at all.
	t.Run("an undefined phase is refused", func(t *testing.T) {
		a := cbEnt(1, 1, 1)
		a.AttackTarget, a.HasAttackTarget = 2, true
		// 5 is the second scheduler boundary — 6 is the first byte past the
		// six this build now defines.
		a.AttackPhase = AttackPhase(6)
		_, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{a, cbEnt(2, 2, 2)})
		if err == nil {
			t.Fatal("NewWorld accepted phase 6")
		}
	})

	t.Run("a negative count owed is refused", func(t *testing.T) {
		a := cbEnt(1, 1, 1)
		a.AttackTarget, a.HasAttackTarget, a.AttackCountdown = 2, true, -1
		_, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{a, cbEnt(2, 2, 2)})
		if err == nil {
			t.Fatal("NewWorld accepted a negative countdown")
		}
	})
}

func TestTheSevenCombatNumbersAreCarriedWhole(t *testing.T) {
	const lo, hi int32 = -2147483648, 2147483647
	a := cbEnt(1, 1, 1)
	a.AttackCharge, a.AttackRelax = lo, hi
	a.ToHit, a.Defence, a.Absorption = hi, lo, hi
	a.DamageBase, a.DamageSpread = lo, hi
	a.AlwaysHits = true
	w := cbWorld(t, 1, a)
	got := cbAt(t, w, 1)
	if got.AttackCharge != lo || got.AttackRelax != hi || got.ToHit != hi || got.Defence != lo ||
		got.Absorption != hi || got.DamageBase != lo || got.DamageSpread != hi || !got.AlwaysHits {
		t.Errorf("the seven came back as %+v", got)
	}
}

// ---------------------------------------------------------------- the order

// cbOrder is one attack command: attacker, victim.
func cbOrder(attacker, victim EntityID) Command {
	return Command{Kind: KindAttack, Entity: attacker, X: int32(victim)}
}

func TestAnAttackOrderIsIgnoredWhereAnyOtherOrderWouldBe(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		live := cbEnt(1, 1, 1)
		downed := cbEnt(2, 2, 2)
		downed.HP = 0
		dead := cbEnt(3, 3, 3)
		dead.HP = -5
		return cbWorld(t, 7, live, downed, dead, cbEnt(4, 4, 4))
	}
	cases := []struct {
		name string
		cmd  Command
	}{
		{"an attacker the world does not hold", cbOrder(99, 4)},
		{"a victim the world does not hold", cbOrder(1, 99)},
		{"an attacker naming itself", cbOrder(1, 1)},
		{"a downed attacker", cbOrder(2, 4)},
		{"a dead attacker", cbOrder(3, 4)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quiet := build(t)
			Step(quiet, nil)
			ordered := build(t)
			Step(ordered, []Command{tc.cmd})
			if got, want := ordered.Hash(), quiet.Hash(); got != want {
				t.Errorf("the ordered world hashes %#016x, the quiet one %#016x", got, want)
			}
		})
	}
	// And the same order on a live attacker and a live victim DOES move the
	// world, so the table above measures the refusals and not a world nothing
	// can change.
	quiet := build(t)
	Step(quiet, nil)
	ordered := build(t)
	Step(ordered, []Command{cbOrder(1, 4)})
	if ordered.Hash() == quiet.Hash() {
		t.Error("a legal attack order changed nothing")
	}
}

func TestWalkingAndAttackingAreOneState(t *testing.T) {
	a := cbEnt(1, 0, 0)
	a.Speed = 12
	w := cbWorld(t, 3, a, cbEnt(2, 1, 0))

	// A walk far enough to store a route and take on a crossing.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 8, Y: 8}})
	if len(w.routes[0]) == 0 || w.entities[0].Transit == 0 {
		t.Fatalf("the fixture needs a stored route and a crossing, got %d cell(s) and %d owed",
			len(w.routes[0]), w.entities[0].Transit)
	}
	transit, total := w.entities[0].Transit, w.entities[0].TransitTotal

	Step(w, []Command{cbOrder(1, 2)})
	e := cbAt(t, w, 1)
	if !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Errorf("after the attack order the unit holds victim %d (present %v)", e.AttackTarget, e.HasAttackTarget)
	}
	if e.HasTarget || e.TargetX != 0 || e.TargetY != 0 || e.Stall != 0 || len(w.routes[0]) != 0 {
		t.Errorf("the walk left residue: target %v (%d,%d), stall %d, %d route cell(s)",
			e.HasTarget, e.TargetX, e.TargetY, e.Stall, len(w.routes[0]))
	}
	if e.Transit != transit-1 || e.TransitTotal != total {
		t.Errorf("the crossing moved to %d/%d, want %d/%d — an order does not end one",
			e.Transit, e.TransitTotal, transit-1, total)
	}

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 4, Y: 4}})
	e = cbAt(t, w, 1)
	if !e.HasTarget {
		t.Error("the move order was not taken")
	}
	if e.HasAttackTarget || e.AttackTarget != 0 || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		t.Errorf("the fight left residue: victim %d (present %v), phase %d, %d owed",
			e.AttackTarget, e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
	}
}

func TestAGroupMoveOrderEndsAFight(t *testing.T) {
	w := cbWorld(t, 3, cbEnt(1, 0, 0), cbEnt(2, 1, 0), cbEnt(3, 5, 5))
	Step(w, []Command{cbOrder(1, 3), cbOrder(2, 3)})
	Step(w, []Command{
		{Kind: KindGroupMoveTo, Entity: 1, X: 9, Y: 9, Group: 1},
		{Kind: KindGroupMoveTo, Entity: 2, X: 9, Y: 9, Group: 1},
	})
	for _, id := range []EntityID{1, 2} {
		if e := cbAt(t, w, id); e.HasAttackTarget || !e.HasTarget {
			t.Errorf("entity %d holds victim %v and destination %v", id, e.HasAttackTarget, e.HasTarget)
		}
	}
}

func TestBeingFelledEndsAFight(t *testing.T) {
	for _, tc := range []struct {
		name string
		blow Command
	}{
		{"killed outright", Command{Kind: KindKill, Entity: 1}},
		{"damaged to death", Command{Kind: KindDamage, Entity: 1, X: 1000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := cbWorld(t, 5, cbEnt(1, 0, 0), cbEnt(2, 1, 0))
			Step(w, []Command{cbOrder(1, 2)})
			Step(w, []Command{tc.blow})
			e := cbAt(t, w, 1)
			if e.Alive() {
				t.Fatalf("the blow left the attacker at %d/%d", e.HP, e.MaxHP)
			}
			if e.HasAttackTarget || e.AttackTarget != 0 || e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
				t.Errorf("a felled attacker kept victim %d (present %v), phase %d, %d owed",
					e.AttackTarget, e.HasAttackTarget, e.AttackPhase, e.AttackCountdown)
			}
		})
	}
}

// ---------------------------------------------------------------- the cycle

// cbFighter is an attacker that cannot miss and whose damage is fixed: a base
// above any absorption these tests give, a spread of zero, and the always-hits
// mark. What it leaves free is the cadence, which each case names.
func cbFighter(id EntityID, x, y int32, charge, relax int32) Entity {
	e := cbEnt(id, x, y)
	e.AttackCharge, e.AttackRelax = charge, relax
	e.DamageBase, e.AlwaysHits = 10, true
	return e
}

// cbDraws is how many times an advance turned the world's generator, counted by
// walking a copy of the state forward until it meets the world's. It measures
// DRAWS and not outcomes, which is the only way to say that a refusal stands
// before a roll rather than after it.
func cbDraws(t *testing.T, before, after uint64) int {
	t.Helper()
	probe := rng{state: before}
	for n := 0; n <= 12; n++ {
		if probe.state == after {
			return n
		}
		probe.next()
	}
	t.Fatal("the generator advanced more than twelve times in one advance")
	return 0
}

func TestABlowLandsOnTheChargeThAdvanceAndThePeriodIsTheCycle(t *testing.T) {
	for _, tc := range []struct{ charge, relax int32 }{
		{5, 3}, {1, 0}, {0, 0}, {-4, -9}, {2, 7}, {8, 1},
	} {
		w := cbWorld(t, 11, cbFighter(1, 0, 0, tc.charge, tc.relax), cbEnt(2, 1, 0))
		// The advance that applies the order is the first, and this is it.
		Step(w, []Command{cbOrder(1, 2)})
		var falls []int
		last := cbAt(t, w, 2).HP
		if last != 100 {
			falls = append(falls, 1)
		}
		for k := 2; k <= 61; k++ {
			Step(w, nil)
			if hp := cbAt(t, w, 2).HP; hp != last {
				falls = append(falls, k)
				last = hp
			}
		}

		wantFirst := int(tc.charge)
		if wantFirst < 1 {
			wantFirst = 1
		}
		wantRelax := int(tc.relax)
		if wantRelax < 0 {
			wantRelax = 0
		}
		if len(falls) < 3 {
			t.Fatalf("charge %d relax %d: only %d blow(s) in 61 advances", tc.charge, tc.relax, len(falls))
		}
		if falls[0] != wantFirst {
			t.Errorf("charge %d relax %d: first blow on advance %d, want %d",
				tc.charge, tc.relax, falls[0], wantFirst)
		}
		lo := wantFirst + wantRelax + 2 // the two scheduler boundary turns
		for k := 1; k < len(falls); k++ {
			if gap := falls[k] - falls[k-1]; gap < lo || gap > lo+relaxJitter {
				t.Errorf("charge %d relax %d: blows %d and %d are %d advance(s) apart, want %d to %d",
					tc.charge, tc.relax, k-1, k, gap, lo, lo+relaxJitter)
			}
		}
	}
}

func TestSpeedMovesNothingAboutTheCycle(t *testing.T) {
	build := func(t *testing.T, speed int32) *World {
		t.Helper()
		a := cbFighter(1, 0, 0, 4, 2)
		a.Speed = speed
		return cbWorld(t, 21, a, cbEnt(2, 1, 0))
	}
	slow, fast := build(t, 1), build(t, 250)
	Step(slow, []Command{cbOrder(1, 2)})
	Step(fast, []Command{cbOrder(1, 2)})
	for k := 0; k < 40; k++ {
		se, fe := slow.Entities(), fast.Entities()
		for i := range se {
			se[i].Speed, fe[i].Speed = 0, 0
			if se[i] != fe[i] {
				t.Fatalf("tick %d: entity %d is %+v and %+v", k, se[i].ID, se[i], fe[i])
			}
		}
		Step(slow, nil)
		Step(fast, nil)
	}
}

func TestAnOrderedAttackKillsItsVictim(t *testing.T) {
	victim := cbEnt(2, 1, 0)
	victim.HP, victim.MaxHP = 20, 20
	w := cbWorld(t, 33, cbFighter(1, 0, 0, 3, 1), victim)
	Step(w, []Command{cbOrder(1, 2)})

	downed := false
	for k := 0; k < 100; k++ {
		Step(w, nil)
		e := cbAt(t, w, 2)
		if e.Downed() {
			downed = true
		}
		if e.Dead() {
			break
		}
	}
	v := cbAt(t, w, 2)
	if !v.Dead() {
		t.Fatalf("the victim is at %d/%d after 100 advances", v.HP, v.MaxHP)
	}
	if !downed {
		t.Error("the victim never passed through downed — 20 health and a blow of 10 must land on 0")
	}
	if v.HasTarget || v.HasAttackTarget {
		t.Error("a felled victim kept an order")
	}
	if a := cbAt(t, w, 1); a.HasAttackTarget {
		t.Fatal("health -10 left the retained order active")
	}
	vi := indexOfEntity(w.entities, 2)
	w.entities[vi].HP, w.entities[vi].Dwell = decayGoneHP-1, 0
	Step(w, nil)
	if a := cbAt(t, w, 1); a.HasAttackTarget {
		t.Errorf("target teardown left victim %d", a.AttackTarget)
	}
}

func TestABlowThatCannotLandRemovesNothing(t *testing.T) {
	t.Run("a miss removes nothing and a hit removes base plus spread less absorption", func(t *testing.T) {
		a := cbEnt(1, 0, 0)
		a.AttackCharge, a.AttackRelax = 1, 0
		a.DamageBase, a.DamageSpread = 4, 6
		// A to-hit no roll can bring past the defence, so the only blows that
		// land are the auto-hit band's.
		a.ToHit = 0
		v := cbEnt(2, 1, 0)
		v.Defence, v.Absorption = 1000, 2
		v.HP, v.MaxHP = 1<<30, 1<<30
		w := cbWorld(t, 44, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		misses, hits := 0, 0
		last := cbAt(t, w, 2).HP
		for k := 0; k < 400; k++ {
			Step(w, nil)
			hp := cbAt(t, w, 2).HP
			if hp == last {
				misses++
				continue
			}
			hits++
			if d := last - hp; d < 4-2 || d > 4+6-2 {
				t.Fatalf("a landed blow removed %d, want %d to %d", d, 4-2, 4+6-2)
			}
			last = hp
		}
		if misses == 0 || hits == 0 {
			t.Errorf("%d miss(es) and %d hit(s) over 400 advances — the fixture must produce both",
				misses, hits)
		}
	})

	t.Run("absorption above the damage removes nothing", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		a.DamageBase, a.DamageSpread = 3, 4
		a.AlwaysHits, a.ToHit = false, 2147483647
		v := cbEnt(2, 1, 0)
		v.Absorption = 7
		w := cbWorld(t, 55, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		for k := 0; k < 200; k++ {
			Step(w, nil)
			if hp := cbAt(t, w, 2).HP; hp != 100 {
				t.Fatalf("advance %d: health fell to %d through an absorption of 7", k, hp)
			}
		}
	})

	t.Run("a victim with no health system is never wounded", func(t *testing.T) {
		v := cbEnt(2, 1, 0)
		v.HP, v.MaxHP = 0, 0
		w := cbWorld(t, 66, cbFighter(1, 0, 0, 1, 0), v)
		Step(w, []Command{cbOrder(1, 2)})
		for k := 0; k < 100; k++ {
			Step(w, nil)
			if e := cbAt(t, w, 2); e.HP != 0 || !e.Alive() {
				t.Fatalf("advance %d: the unit is at %d/%d", k, e.HP, e.MaxHP)
			}
		}
	})
}

func TestAlwaysHitsBeatsAnyDefence(t *testing.T) {
	v := cbEnt(2, 1, 0)
	v.Defence = 2147483647
	a := cbFighter(1, 0, 0, 1, 0)
	a.ToHit = -2147483648
	w := cbWorld(t, 77, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	if hp := cbAt(t, w, 2).HP; hp != 90 {
		t.Errorf("the victim is at %d, want 90 — an always-hits blow lands whatever the defence", hp)
	}
}

func TestANonPositiveCadenceIsTheFloorAndNotItsOwnNumber(t *testing.T) {
	build := func(t *testing.T, relax int32) *World {
		t.Helper()
		return cbWorld(t, 909, cbFighter(1, 0, 0, 3, relax), cbEnt(2, 1, 0))
	}
	zero, negative := build(t, 0), build(t, -9)
	Step(zero, []Command{cbOrder(1, 2)})
	Step(negative, []Command{cbOrder(1, 2)})
	for k := 0; k < 50; k++ {
		ze, ne := zero.Entities(), negative.Entities()
		for i := range ze {
			ze[i].AttackRelax, ne[i].AttackRelax = 0, 0
			if ze[i] != ne[i] {
				t.Fatalf("advance %d: a relax of 0 gives %+v and one of -9 gives %+v", k, ze[i], ne[i])
			}
		}
		Step(zero, nil)
		Step(negative, nil)
	}
}

// TestTheHitTestDoesNotWrapAtTheEndsOfTheRange is the arithmetic's own witness.
// The seven numbers are carried whole, so a to-hit at the top of the range plus a
// positive roll is a sum no int32 holds: taken narrow it wraps negative and turns
// a certain hit into a certain miss.
//
// The seed puts the roll at 88 — positive, so the sum decides it, and below the
// auto-hit band, so the band does not.
func TestTheHitTestDoesNotWrapAtTheEndsOfTheRange(t *testing.T) {
	a := cbEnt(1, 0, 0)
	a.AttackCharge, a.AttackRelax = 1, 0
	a.DamageBase, a.ToHit = 10, 2147483647
	v := cbEnt(2, 1, 0)
	v.Defence = 2147483647
	w := cbWorld(t, 12, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	if hp := cbAt(t, w, 2).HP; hp != 90 {
		t.Errorf("the victim is at %d, want 90 — to-hit %d plus a roll of 88 beats a defence of %d",
			hp, a.ToHit, v.Defence)
	}
}

// TestABlowAtTheTopOfTheRangeNeitherWrapsNorResurrects is the damage half of the
// same arithmetic, and the sharper half: the damage a blow rolls is a sum of two
// whole int32s and the subtraction that follows can leave a value below the least
// int32 there is.
//
// Taken narrow, the sum wraps NEGATIVE and the blow removes nothing; the
// subtraction wraps POSITIVE and a downed unit comes back at the top of the
// range. Both are worse than the value being wrong — one blow does nothing and
// the other undoes a death — so both are pinned here rather than argued.
func TestABlowAtTheTopOfTheRangeNeitherWrapsNorResurrects(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.DamageBase, a.DamageSpread = 2147483647, 5
	v := cbEnt(2, 1, 0)
	v.HP, v.MaxHP = 0, 100 // downed: a further blow takes this combat path
	// A NEGATIVE absorption, which the field carries like any other value, so the
	// subtraction adds and the blow is past the least int32 for every roll the
	// spread can produce rather than for most of them.
	v.Absorption = -10
	w := cbWorld(t, 314, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	got := cbAt(t, w, 2)
	if got.HP != minHP {
		t.Errorf("the victim is at %d, want the least health there is (%d) — a blow of about "+
			"2^31 on a unit at zero saturates and never wraps", got.HP, minHP)
	}
	if !got.Dead() {
		t.Error("a downed unit struck for 2^31 is not dead")
	}
}

func TestTheAutoHitBandStartsAtNinety(t *testing.T) {
	for _, tc := range []struct {
		roll int32
		seed uint64
		land bool
	}{
		{88, 12, false},
		{89, 371, false},
		{90, 137, true},
		{91, 132, true},
	} {
		a := cbEnt(1, 0, 0)
		a.AttackCharge, a.AttackRelax = 1, 0
		a.DamageBase, a.ToHit = 10, -2147483648
		v := cbEnt(2, 1, 0)
		v.Defence = 2147483647
		w := cbWorld(t, tc.seed, a, v)
		// The order's own advance is the strike, so the roll is the second draw
		// off the seed — which is the draw the seed was chosen for.
		Step(w, []Command{cbOrder(1, 2)})
		landed := cbAt(t, w, 2).HP != 100
		if landed != tc.land {
			t.Errorf("a roll of %d landed=%v, want %v — the band is [%d, 100]",
				tc.roll, landed, tc.land, autoHitRoll)
		}
	}
	if autoHitRoll != 90 {
		t.Errorf("the auto-hit band starts at %d, want 90", autoHitRoll)
	}
}

func TestAnOutOfReachOrderWaitsBeforeItsCycle(t *testing.T) {
	probe := func(t *testing.T, dx int32) (int, int32, AttackPhase) {
		t.Helper()
		w := cbWorld(t, 88, cbFighter(1, 0, 0, 1, 9), cbEnt(2, dx, 0))
		before := w.rng.state
		Step(w, []Command{cbOrder(1, 2)}) // charge 1, so the blow is on this advance
		return cbDraws(t, before, w.rng.state), cbAt(t, w, 2).HP, cbAt(t, w, 1).AttackPhase
	}
	if n, hp, ph := probe(t, 1); n != 3 || hp == 100 || ph != AttackRelaxing {
		t.Errorf("in reach: %d draw(s) — two for the blow and one jitter — the victim at %d, phase %d; "+
			"want 3, a wound and relaxing", n, hp, ph)
	}
	if n, hp, ph := probe(t, 4); n != 0 || hp != 100 || ph != AttackReady {
		t.Errorf("out of reach: %d draw(s), the victim at %d, phase %d; want 0, no wound and the cycle waiting",
			n, hp, ph)
	}
}

// TestTheHigherIdMayFinishMinusSixButTheMinusTenBoundaryClearsIt keeps the
// finishable negative band distinct from the corpse boundary.
func TestTheHigherIdMayFinishMinusSixButTheMinusTenBoundaryClearsIt(t *testing.T) {
	victim := cbEnt(3, 1, 0)
	victim.HP, victim.MaxHP = 4, 4
	w := cbWorld(t, 99, cbFighter(1, 0, 0, 1, 20), cbFighter(2, 2, 0, 1, 20), victim)
	before := w.rng.state
	Step(w, []Command{cbOrder(1, 3), cbOrder(2, 3)}) // charge 1: both would strike here
	if v := cbAt(t, w, 3); !v.Dead() {
		t.Fatalf("the victim is at %d after the first blow", v.HP)
	}
	// Both blows draw hit and damage. The blow crossing -10 draws no recovery
	// jitter because it clears the cycle at the damage site.
	if n := cbDraws(t, before, w.rng.state); n != 5 {
		t.Errorf("the killing advance cost %d draw(s), want 5", n)
	}
	if e := cbAt(t, w, 2); e.HasAttackTarget {
		t.Errorf("the higher id retained victim %d after taking it through -10", e.AttackTarget)
	}
}

func TestReIssuingAnOrderDoesNotResetTheCycle(t *testing.T) {
	same := cbWorld(t, 101, cbFighter(1, 0, 0, 4, 1), cbEnt(2, 1, 0))
	for k := 0; k < 40; k++ {
		Step(same, []Command{cbOrder(1, 2)})
	}
	if hp := cbAt(t, same, 2).HP; hp == 100 {
		t.Error("re-issuing the same order every tick landed no blow in 40 advances")
	}

	other := cbWorld(t, 101, cbFighter(1, 0, 0, 4, 1), cbEnt(2, 1, 0), cbEnt(3, 0, 1))
	for k := 0; k < 40; k++ {
		Step(other, []Command{cbOrder(1, EntityID(2+k%2))})
	}
	for _, id := range []EntityID{2, 3} {
		if hp := cbAt(t, other, id).HP; hp >= 100 {
			t.Errorf("entity %d is at %d: alternating accepted requests must let both victims take a blow",
				id, hp)
		}
	}
}

func TestADoubledAttackOrderIsIdempotentAndADoubledDamageCommandIsNot(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		return cbWorld(t, 123, cbFighter(1, 0, 0, 2, 1), cbEnt(2, 1, 0))
	}
	order := cbOrder(1, 2)
	for k := 0; k < 12; k++ {
		once, twice := build(t), build(t)
		for j := 0; j <= k; j++ {
			Step(once, []Command{order})
			Step(twice, []Command{order, order})
		}
		if got, want := twice.Hash(), once.Hash(); got != want {
			t.Fatalf("after %d doubled advance(s) the world hashes %#016x, want %#016x", k+1, got, want)
		}
	}

	blow := Command{Kind: KindDamage, Entity: 2, X: 7}
	once, twice := build(t), build(t)
	Step(once, []Command{blow})
	Step(twice, []Command{blow, blow})
	if cbAt(t, once, 2).HP != 93 || cbAt(t, twice, 2).HP != 86 {
		t.Errorf("one damage command left %d and two left %d, want 93 and 86 — the debug arm "+
			"accumulates, and narrowing it to a set-health form must fail here",
			cbAt(t, once, 2).HP, cbAt(t, twice, 2).HP)
	}
}

// ------------------------------------------------------------- the relation

func TestNothingFlipsOnAMissAnOutOfReachSwingOrANamelessParty(t *testing.T) {
	t.Run("a miss", func(t *testing.T) {
		// Seed 12 rolls 88 here — TestTheAutoHitBandStartsAtNinety's own
		// proven miss: below the auto-hit band, and a to-hit at the bottom of
		// the range against a defence at the top so the sum never beats it.
		a := cbEnt(1, 0, 0)
		a.Owner = 2
		a.AttackCharge, a.AttackRelax = 1, 0
		a.DamageBase, a.ToHit = 10, -2147483648
		v := cbEnt(2, 1, 0)
		v.Owner = 3
		v.Defence = 2147483647
		w := cbWorld(t, 12, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		if got := cbAt(t, w, 2).HP; got != 100 {
			t.Fatalf("the fixture is meant to miss; the victim is at %d", got)
		}
		if rel := w.Relations(); rel.Hostile(2, 3) || rel.Hostile(3, 2) {
			t.Errorf("a miss flipped the relation: [2][3]=%v [3][2]=%v",
				rel.Hostile(2, 3), rel.Hostile(3, 2))
		}
	})

	t.Run("out of reach", func(t *testing.T) {
		// TestAnOutOfReachOrderWaitsBeforeItsCycle's own out-of-reach
		// case: dx 4 against a reach of 1, and AlwaysHits so only the reach
		// test can be what refuses the blow.
		a := cbFighter(1, 0, 0, 1, 9)
		a.Owner = 2
		v := cbEnt(2, 4, 0)
		v.Owner = 3
		w := cbWorld(t, 88, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		if got := cbAt(t, w, 2).HP; got != 100 {
			t.Fatalf("the fixture is meant to stay out of reach; the victim is at %d", got)
		}
		if rel := w.Relations(); rel.Hostile(2, 3) || rel.Hostile(3, 2) {
			t.Errorf("an out-of-reach swing flipped the relation: [2][3]=%v [3][2]=%v",
				rel.Hostile(2, 3), rel.Hostile(3, 2))
		}
	})

	t.Run("either party names no roster slot", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		v := cbEnt(2, 1, 0)
		v.Owner = 3
		w := cbWorld(t, 55, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		if got := cbAt(t, w, 2).HP; got == 100 {
			t.Fatalf("the fixture is meant to land; the victim is still at %d", got)
		}
		if rel := w.Relations(); rel.Hostile(3, 0) || rel.Hostile(0, 3) {
			t.Errorf("a blow with a nameless attacker flipped the relation: [3][0]=%v [0][3]=%v",
				rel.Hostile(3, 0), rel.Hostile(0, 3))
		}
	})
}

// TestAnAbsorbedBlowAndABlowOnADownedVictimBothFlip is AC-6: a blow whose
// damage is entirely absorbed still flips, and so does a blow on a victim
// already below 1 health — the attempt is made before absorption and
// before any test on what damage survives it.
func TestAnAbsorbedBlowAndABlowOnADownedVictimBothFlip(t *testing.T) {
	t.Run("damage entirely absorbed", func(t *testing.T) {
		// TestABlowThatCannotLandRemovesNothing's own "absorption above the
		// damage" fixture: a base+spread of 3 to 7 against an absorption of 7
		// removes nothing, ever.
		a := cbFighter(1, 0, 0, 1, 0)
		a.Owner = 2
		a.DamageBase, a.DamageSpread = 3, 4
		a.AlwaysHits, a.ToHit = false, 2147483647
		v := cbEnt(2, 1, 0)
		v.Owner = 3
		v.Absorption = 7
		w := cbWorld(t, 55, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		if got := cbAt(t, w, 2).HP; got != 100 {
			t.Fatalf("the fixture is meant to be fully absorbed; the victim is at %d", got)
		}
		if rel := w.Relations(); !rel.Hostile(2, 3) || !rel.Hostile(3, 2) {
			t.Errorf("a fully absorbed blow left [2][3]=%v [3][2]=%v, want both hostile",
				rel.Hostile(2, 3), rel.Hostile(3, 2))
		}
	})

	t.Run("victim already below 1 health", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		a.Owner = 2
		v := cbEnt(2, 1, 0)
		v.Owner = 3
		v.HP, v.MaxHP = 0, 100
		w := cbWorld(t, 66, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		if rel := w.Relations(); !rel.Hostile(2, 3) || !rel.Hostile(3, 2) {
			t.Errorf("a blow on a victim below 1 health left [2][3]=%v [3][2]=%v, want both hostile",
				rel.Hostile(2, 3), rel.Hostile(3, 2))
		}
	})
}

func TestFR6BothArmsOfTheDiagonal(t *testing.T) {
	t.Run("no relation at all: the self-blow sets the diagonal", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		a.Owner = 5
		v := cbEnt(2, 1, 0)
		v.Owner = 5
		w := cbWorld(t, 77, a, v) // NewWorld: the world names no relation
		Step(w, []Command{cbOrder(1, 2)})
		if rel := w.Relations(); !rel.Hostile(5, 5) {
			t.Errorf("a self-blow on a world named no relation left [5][5]=%v, want hostile",
				rel.Hostile(5, 5))
		}
	})

	t.Run("a loaded map's forced 2 is left alone", func(t *testing.T) {
		a := cbFighter(1, 0, 0, 1, 0)
		a.Owner = 5
		v := cbEnt(2, 1, 0)
		v.Owner = 5
		rel := engRel(t, [3]uint32{5, 5, 2}) // the loader's own forced diagonal
		w, err := NewRelatedWorld(77, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
			[]Entity{a, v}, nil, rel)
		if err != nil {
			t.Fatalf("NewRelatedWorld: %v", err)
		}
		Step(w, []Command{cbOrder(1, 2)})
		if got := w.Relations().Byte(5, 5); got != 2 {
			t.Errorf("a self-blow on a loaded diagonal of 2 left it at %#02x, want 2 unchanged", got)
		}
	})
}

// TestAWorldMidCycleAdvancesTheSameFromItsBytes is the round trip carrying the
// whole fight, generator and all: a world cut mid-charge and one decoded from its
// bytes agree at every tick for sixty of them.
func TestAWorldMidCycleAdvancesTheSameFromItsBytes(t *testing.T) {
	a := cbFighter(1, 0, 0, 5, 2)
	a.DamageBase, a.DamageSpread, a.AlwaysHits = 3, 9, false
	a.ToHit = 30
	v := cbEnt(2, 1, 0)
	v.Defence, v.Absorption = 25, 1
	v.HP, v.MaxHP = 1<<20, 1<<20
	first := cbWorld(t, 4242, a, v)
	Step(first, []Command{cbOrder(1, 2)})
	Step(first, nil)
	Step(first, nil)
	if e := cbAt(t, first, 1); e.AttackPhase != AttackCharging {
		t.Fatalf("the fixture is meant to be caught mid-charge, and is in phase %d", e.AttackPhase)
	}

	form, err := first.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var second World
	if err := second.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	for k := 0; k < 60; k++ {
		Step(first, nil)
		Step(&second, nil)
		if got, want := second.Hash(), first.Hash(); got != want {
			t.Fatalf("advance %d: the decoded world hashes %#016x, the first %#016x", k, got, want)
		}
	}
}

// TestACrossingCancelsTheAttackPhaseLiveAndRestoredAlike is HERO-CROSSHOLD-146:
// a crossing tick — whether it began in the live world (Transit != 0) or was
// resumed from a SAV (motionActive) — cancels the loaded attack phase back to
// AttackReady on every tick it runs, and leaves the countdown exactly as it
// stood. Before the fix the two disagreed: a live crossing fell through to the
// ordinary cycle and decremented the countdown, while a restored one froze the
// whole cycle untouched, so the two produced different countdowns from the
// same starting charge.
func TestACrossingCancelsTheAttackPhaseLiveAndRestoredAlike(t *testing.T) {
	crossingEntity := func(cross func(w *World)) Entity {
		a := cbEnt(1, 0, 0)
		a.HasAttackTarget, a.AttackTarget = true, 2
		a.AttackCharge = 10
		a.AttackPhase, a.AttackCountdown = AttackCharging, 7
		w := cbWorld(t, 1, a, cbEnt(2, 5, 5))
		cross(w)
		w.advanceAttack(0, nil)
		return cbAt(t, w, 1)
	}
	live := crossingEntity(func(w *World) { w.entities[0].Transit = 3 })
	restored := crossingEntity(func(w *World) {
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true, Active: true}}}
	})
	for name, e := range map[string]Entity{"live": live, "restored": restored} {
		if e.AttackPhase != AttackReady || e.AttackCountdown != 7 {
			t.Errorf("%s crossing tick left phase %d and %d tick(s) owed, want AttackReady and the stale 7 untouched",
				name, e.AttackPhase, e.AttackCountdown)
		}
	}
	if live.AttackPhase != restored.AttackPhase || live.AttackCountdown != restored.AttackCountdown {
		t.Fatalf("live and restored crossings disagree: live %d/%d, restored %d/%d",
			live.AttackPhase, live.AttackCountdown, restored.AttackPhase, restored.AttackCountdown)
	}
}

// TestATurnWithholdsTheChargeLoadLiveAndRestoredAlike is AI-FACE-066's
// facing precondition at the act-state entry: an attacker at AttackReady that
// still owes a turn does not load its charge, whether the turn is live
// (Turning()) or a centred turn a SAV restores un-executed (savedTurnQueued,
// folded into motionActive).
func TestATurnWithholdsTheChargeLoadLiveAndRestoredAlike(t *testing.T) {
	turningEntity := func(turn func(w *World)) Entity {
		a := cbFighter(1, 0, 0, 10, 3)
		a.HasAttackTarget, a.AttackTarget = true, 2
		w := cbWorld(t, 1, a, cbEnt(2, 1, 0))
		turn(w)
		w.advanceAttack(0, nil)
		return cbAt(t, w, 1)
	}
	live := turningEntity(func(w *World) {
		w.entities[0].TurnRemaining, w.entities[0].TurnTotal, w.entities[0].DesiredFacing = 3, 3, facingOfDir(2)
	})
	restored := turningEntity(func(w *World) {
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{
			Entity:   1,
			Current:  true,
			Position: SavedActorPosition{FineX: 128, FineY: 128},
			Mover:    [180]byte{0: 0, 1: facingOfDir(2), 10: 4},
		}}}
	})
	for name, e := range map[string]Entity{"live": live, "restored": restored} {
		if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
			t.Errorf("%s turn at AttackReady left phase %d and %d tick(s) owed, want no charge loaded",
				name, e.AttackPhase, e.AttackCountdown)
		}
	}
}

// TestACirclingVictimNeitherStallsNorHastensTheCycle is the latch
// (AI-ORDER-039, AI-RETREAT-272, HERO-CADENCE-112): a victim that steps to the
// next cell of the ring around its attacker every k ticks costs the attacker
// only the re-face at AttackReady. Its blows still land, and no two fall closer
// than a stationary pair's shortest period, charge + relax + 2.
func TestACirclingVictimNeitherStallsNorHastensTheCycle(t *testing.T) {
	const charge, relax, window = 5, 3, 240
	ring := [8][2]int32{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}
	blows := func(t *testing.T, k int) []int {
		a := cbFighter(1, 5, 5, charge, relax)
		a.RotationSpeed = 32
		v := cbEnt(2, 5, 4)
		v.HP, v.MaxHP = 1_000_000, 1_000_000
		w := cbWorld(t, 3, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		var at []int
		last, slot := cbAt(t, w, 2).HP, 0
		for n := 1; n <= window; n++ {
			var order []Command
			if k > 0 && n%k == 0 {
				slot = (slot + 1) % len(ring)
				order = []Command{MoveTo(2, CellPoint{X: 5 + ring[slot][0], Y: 5 + ring[slot][1]})}
			}
			Step(w, order)
			if hp := cbAt(t, w, 2).HP; hp != last {
				at, last = append(at, n), hp
			}
		}
		return at
	}
	still := blows(t, 0)
	for _, k := range []int{2, 4, 6, 8} {
		got := blows(t, k)
		if len(got) < len(still)/2 || len(got) > len(still) {
			t.Errorf("k=%d: %d blow(s) in %d ticks, a stationary pair lands %d", k, len(got), window, len(still))
		}
		for j := 1; j < len(got); j++ {
			if gap := got[j] - got[j-1]; gap < charge+relax+2 {
				t.Errorf("k=%d: blows at %d and %d are %d tick(s) apart, under the %d-tick cycle",
					k, got[j-1], got[j], gap, charge+relax+2)
			}
		}
	}
}

// TestADyingPursuersOrderStaysFrozenUntilTeardown is HERO-DYINGTICK-145: the
// dying branch never reaches the order machine, so a restored dying pursuer
// keeps its victim, phase and countdown through every dying tick, and the
// byte form accepts each of them. The order ends with the dying window.
func TestADyingPursuersOrderStaysFrozenUntilTeardown(t *testing.T) {
	w := cbWorld(t, 1, cbFighter(1, 5, 5, 10, 3), cbEnt(2, 5, 4))
	d := &w.entities[0]
	d.HP, d.Decay, d.Dwell = 0, DecayFallen, 3
	d.HasAttackTarget, d.AttackTarget, d.AttackPhase, d.AttackCountdown = true, 2, AttackCharging, 6
	for n := 1; n <= 3; n++ {
		Step(w, nil)
		e := cbAt(t, w, 1)
		if _, err := w.MarshalBinary(); err != nil {
			t.Fatalf("tick %d: %v", n, err)
		}
		if e.Dying() {
			if !e.HasAttackTarget || e.AttackTarget != 2 || e.AttackPhase != AttackCharging || e.AttackCountdown != 6 {
				t.Fatalf("dying tick %d: order %v->%d phase %d countdown %d, want frozen 2/%d/6",
					n, e.HasAttackTarget, e.AttackTarget, e.AttackPhase, e.AttackCountdown, AttackCharging)
			}
		} else if e.HasAttackTarget || e.AttackPhase != AttackReady {
			t.Fatalf("tick %d: the dying window ended and the order %v->%d phase %d remains",
				n, e.HasAttackTarget, e.AttackTarget, e.AttackPhase)
		}
	}
	if entityRef(cbAt(t, w, 1)).Dying() {
		t.Fatal("a dwell of 3 is still dying after 3 ticks")
	}
}
