package sim

// The two blows: what they do to a health number, what they do to the order the
// unit was holding, and the cases in which they do nothing at all.
//
// Nothing here reads a byte form. What a blow leaves behind is checked as the
// FIELDS of the unit it named, and what a no-op leaves behind is checked as the
// whole world's digest against a tick that carried no command — the one
// comparison that can see a field moved somewhere this file did not think to
// look.

import (
	"fmt"
	"testing"
)

// dmgCorridor is the fixture behind every "the order was cleared" assertion
// here: a mover holding all three of what an order consists of — a target, a
// STORED ROUTE, and a NONZERO stall count — so that a blow's clearing is
// measured against a unit that had all three to lose.
//
// The shape is a three-cell corridor with a second unit standing in the middle
// of it. The far search reads terrain alone, finds the whole corridor and stores
// the route; the near search reads occupancy, cannot get past the unit in the
// way, and raises the stall instead — and the route it could not walk stays
// stored. One tick puts the mover in exactly that state, and the fixture check
// below says so rather than assuming it.
//
// The blocker is built at the same health as the mover so that a case which
// kills the mover has not also changed what stands in front of it.
func dmgCorridor(t *testing.T, hp, maxHP int32) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{Width: 3, Height: 1}, []Entity{
		{ID: 1, X: 0, Y: 0, HP: hp, MaxHP: maxHP},
		{ID: 2, X: 1, Y: 0, HP: hp, MaxHP: maxHP},
	})
	Step(w, []Command{{Entity: 1, X: 2, Y: 0}})
	e := occEntity(t, w, 1)
	if !e.HasTarget || e.Stall == 0 || len(w.routes[0]) == 0 {
		t.Fatalf("fixture: unit 1 is %+v holding the route %s — it must hold a target, a stall count and a route",
			e, fmtRoute(w.routes[0]))
	}
	if e.X != 0 || e.Y != 0 {
		t.Fatalf("fixture: unit 1 walked to (%d,%d); the blocker must hold it where it started", e.X, e.Y)
	}
	return w
}

// dmgOrderGone checks the whole of what an order is, on the unit at index i: no
// target, no residue in the coordinates it named, a zero stall count, and no
// stored route. Three fields and a slice, because clearing three of the four and
// leaving one behind is a world the byte form refuses.
func dmgOrderGone(t *testing.T, w *World, what string, i int) {
	t.Helper()
	e := w.entities[i]
	if e.HasTarget || e.TargetX != 0 || e.TargetY != 0 || e.Stall != 0 || len(w.routes[i]) != 0 {
		t.Errorf("%s: unit %d holds target %v (%d,%d), stall %d and the route %s — a felled unit holds none of them",
			what, e.ID, e.HasTarget, e.TargetX, e.TargetY, e.Stall, fmtRoute(w.routes[i]))
	}
}

// TestTheDamageLadderIsWalkedRungByRung is AC-2. The health is asserted after
// EVERY one of the eleven blows and not at the ends, so a rule that clamped, or
// that took two rungs at once, fails at the rung it went wrong on rather than at
// the bottom of the ladder where several defects look alike.
//
// The eleventh is the one the ladder is for: the tenth leaves the unit at
// exactly zero, which is downed and not dead, and the eleventh drives it through
// that floor to -10. A health clamped at zero would be indistinguishable from
// this run at rung ten and would stop moving at rung eleven.
func TestTheDamageLadderIsWalkedRungByRung(t *testing.T) {
	rungs := []struct {
		hp   int32
		want life
	}{
		{90, lifeAlive}, {80, lifeAlive}, {70, lifeAlive}, {60, lifeAlive}, {50, lifeAlive},
		{40, lifeAlive}, {30, lifeAlive}, {20, lifeAlive}, {10, lifeAlive},
		{0, lifeDowned},
		{-10, lifeDead},
	}

	w := dmgCorridor(t, 100, 100)
	for k, rung := range rungs {
		Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 10}})
		e := occEntity(t, w, 1)
		if e.HP != rung.hp {
			t.Fatalf("after blow %d the unit is at %d health, want %d", k+1, e.HP, rung.hp)
		}
		got, n := state(e)
		if n != 1 || got != rung.want {
			t.Fatalf("after blow %d the unit at %d/%d is %s (%d predicates), want %s",
				k+1, e.HP, e.MaxHP, got, n, rung.want)
		}
		// Its order survives every rung above the floor and is gone from the
		// floor down. The tenth blow is where it goes, and the eleventh may not
		// bring it back.
		if k < 9 {
			if !e.HasTarget || e.Stall == 0 || len(w.routes[0]) == 0 {
				t.Errorf("after blow %d the unit is %+v holding the route %s — a blow that left it "+
					"alive takes nothing of its order", k+1, e, fmtRoute(w.routes[0]))
			}
			continue
		}
		dmgOrderGone(t, w, fmt.Sprintf("after blow %d", k+1), 0)
	}
}

// TestABlowThatFellsAUnitClearsItsWholeOrder is AC-3. Each case gets a world of
// its own, and the four differ in exactly what the contract says they do: where
// a single damage lands relative to the floor, and what a kill does to each of
// the three states it can find a unit in.
func TestABlowThatFellsAUnitClearsItsWholeOrder(t *testing.T) {
	cases := []struct {
		name    string
		hp      int32
		maxHP   int32
		cmd     Command
		wantHP  int32
		wantSt  life
		cleared bool
	}{
		// Ten damage at five health does not stop at the floor on its way past
		// it: the unit ends at -5 and dead, never at 0 and downed. A clamp is
		// exactly what this case refuses.
		{"ten damage at five health goes through the floor, not to it",
			5, 100, Command{Kind: KindDamage, Entity: 1, X: 10}, -5, lifeDead, true},
		{"a kill on a unit at full health, under order",
			100, 100, Command{Kind: KindKill, Entity: 1}, killHP, lifeDead, true},
		{"a kill on a unit already downed",
			0, 100, Command{Kind: KindKill, Entity: 1}, killHP, lifeDead, true},
		// A maximum of zero is a unit with no health system: damage does not
		// reach it, and a kill still does.
		{"a kill on a unit with no health system",
			0, 0, Command{Kind: KindKill, Entity: 1}, killHP, lifeDead, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A unit at zero health with a positive maximum is DOWNED, and the
			// constructor clears such a unit's order — so the fixture's own check
			// that the mover holds one would fail for those cases. They are built
			// alive and driven to their state by a first blow instead. Authored
			// map bodies exercise the other construction route separately.
			var w *World
			switch {
			case tc.hp == 0 && tc.maxHP > 0:
				w = dmgCorridor(t, tc.maxHP, tc.maxHP)
				Step(w, []Command{{Kind: KindDamage, Entity: 1, X: tc.maxHP}})
				if got := occEntity(t, w, 1); got.HP != 0 {
					t.Fatalf("fixture: the unit was driven to %d health, want 0", got.HP)
				}
			default:
				w = dmgCorridor(t, tc.hp, tc.maxHP)
			}

			// The unit beside the one the blow names, recorded before it lands. A
			// rule that wrote through the wrong index would leave this world
			// looking right where the case looks and wrong one record along.
			beside := occEntity(t, w, 2)

			Step(w, []Command{tc.cmd})

			e := occEntity(t, w, 1)
			if e.HP != tc.wantHP {
				t.Errorf("the unit is at %d health, want %d", e.HP, tc.wantHP)
			}
			if got, n := state(e); n != 1 || got != tc.wantSt {
				t.Errorf("the unit at %d/%d is %s (%d predicates), want %s", e.HP, e.MaxHP, got, n, tc.wantSt)
			}
			if tc.cleared {
				dmgOrderGone(t, w, tc.name, 0)
			}
			if got := occEntity(t, w, 2); got != beside {
				t.Errorf("the unit beside the one named is %+v, want %+v — a blow reaches one record",
					got, beside)
			}
		})
	}
}

// ---------------------------------------------------------------- AC-4

// dmgNoopWorld is the fixture every no-op case is measured on: a unit at full
// health, one already dead, one with no health system, and one under an order so
// that the tick has real work to do — a comparison over a world where nothing
// moves would agree for reasons that have nothing to do with the command.
func dmgNoopWorld(t *testing.T) *World {
	t.Helper()
	return mustWorld(t, 0x5eed, Bounds{Width: 12, Height: 12}, []Entity{
		{ID: 1, X: 0, Y: 0, HP: 100, MaxHP: 100},
		{ID: 2, X: 4, Y: 4, HP: decayBonesHP, MaxHP: 100},
		{ID: 3, X: 2, Y: 2},
		{ID: 4, X: 9, Y: 9, TargetX: 1, TargetY: 8, HasTarget: true, HP: 50, MaxHP: 60},
	})
}

func TestEveryNoOpBlowLeavesTheWorldWhereAQuietTickLeavesIt(t *testing.T) {
	const absent EntityID = 77

	cases := []struct {
		name string
		cmd  Command
	}{
		{"a kill naming an entity the world does not hold", Command{Kind: KindKill, Entity: absent}},
		{"a damage naming an entity the world does not hold", Command{Kind: KindDamage, Entity: absent, X: 10}},
		{"a kill on a unit already dead", Command{Kind: KindKill, Entity: 2}},
		{"a damage on a unit already dead", Command{Kind: KindDamage, Entity: 2, X: 10}},
		{"a damage of nothing", Command{Kind: KindDamage, Entity: 1, X: 0}},
		{"a damage of a negative amount, which may not heal", Command{Kind: KindDamage, Entity: 1, X: -7}},
		{"ten damage on a unit with no health system", Command{Kind: KindDamage, Entity: 3, X: 10}},
		// Kind 3 stood here until 0059 defined it as the group move-to. The case
		// is not weakened by moving to 4: what it witnesses is that a kind with
		// no arm reaches none, and 4 is the lowest byte with no arm today.
		{"a kind this build does not define", Command{Kind: 4, Entity: 1, X: 10, Y: 10}},
		{"the highest kind byte there is", Command{Kind: 255, Entity: 1, X: 10, Y: 10}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quiet := dmgNoopWorld(t)
			Step(quiet, nil)

			w := dmgNoopWorld(t)
			Step(w, []Command{tc.cmd})

			if got, want := w.Hash(), quiet.Hash(); got != want {
				t.Errorf("the world hashes %#016x and a tick carrying no command leaves %#016x",
					got, want)
			}
			if got, want := snap(w), snap(quiet); !equalState(got, want) {
				t.Errorf("the world is\n %+v\na tick carrying no command leaves\n %+v", got, want)
			}
		})
	}

	// The control: a blow that IS applied moves the digest. Without it every case
	// above would pass under a Step that ignored commands outright.
	quiet := dmgNoopWorld(t)
	Step(quiet, nil)
	live := dmgNoopWorld(t)
	Step(live, []Command{{Kind: KindKill, Entity: 1}})
	if live.Hash() == quiet.Hash() {
		t.Errorf("a kill that lands leaves the same digest %#016x as a quiet tick — "+
			"the cases above then witness nothing", quiet.Hash())
	}
}

// TestAnUndefinedKindIsIgnoredRatherThanReadAsAMoveTo is the sharper half of the
// undefined-kind case. The digest comparison above shows such a command changes
// nothing; this shows WHY that is not merely because the command happened to be
// harmless — a build that read an unknown kind as the zero one would set a
// target from X and Y, which is a change the quiet world does not have.
func TestAnUndefinedKindIsIgnoredRatherThanReadAsAMoveTo(t *testing.T) {
	// 3 was in this list until 0059 gave it to the group move-to; 4 replaces it,
	// and the list still runs from the lowest undefined byte to the highest.
	for _, kind := range []uint8{4, 5, 100, 255} {
		w := mustWorld(t, 3, Bounds{Width: 8, Height: 8}, []Entity{{ID: 1, X: 0, Y: 0, HP: 10, MaxHP: 10}})
		Step(w, []Command{{Kind: kind, Entity: 1, X: 5, Y: 5}})
		want := Entity{ID: 1, X: 0, Y: 0, HP: 10, MaxHP: 10, ActorState: actorStateGuard, Reach: 1}
		if got := occEntity(t, w, 1); got != want {
			t.Errorf("kind %d left the unit %+v, want %+v — an undefined kind takes no arm at all",
				kind, got, want)
		}
	}
}

func TestAMoveToIsStillTheKindACommandNamingNoneHas(t *testing.T) {
	if KindMoveTo != 0 {
		t.Fatalf("KindMoveTo is %d; a command naming no kind must be a move-to", KindMoveTo)
	}
	b := Bounds{Width: 8, Height: 8}
	named := mustWorld(t, 9, b, []Entity{{ID: 1, X: 0, Y: 0}})
	Step(named, []Command{{Kind: KindMoveTo, Entity: 1, X: 4, Y: 4}})
	bare := mustWorld(t, 9, b, []Entity{{ID: 1, X: 0, Y: 0}})
	Step(bare, []Command{{Entity: 1, X: 4, Y: 4}})
	if got, want := named.Hash(), bare.Hash(); got != want {
		t.Errorf("naming the move-to hashes %#016x and leaving the kind out %#016x", got, want)
	}
}
