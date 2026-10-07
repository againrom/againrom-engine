package sim

// The group rate term as a CARRIED value: which speed a transit is computed
// from, and the three sites — and only three — that write it.
//
// The order that SETS it is the next task's; this file is about the term once it
// exists, so every world here is built holding one rather than ordered into one.
// That is deliberate: the reader and the writer set have to be right whatever
// puts the byte there, and a test that could only reach them through the order
// would stop measuring them the moment the order changed.
//
// Nothing here reads a game install: every world is built in this file out of
// bounds, positions and entities.

import "testing"

// grBounds is a corridor long enough for a mover to cross several cells without
// meeting an edge, and one row high so no test can accidentally measure a
// diagonal step.
var grBounds = Bounds{Width: 24, Height: 1}

// grEntity is one mover at the corridor's left end, ordered to its right end and
// carrying both speeds, so that which of the two a rate came from is the only
// thing a transit length can be telling us.
func grEntity(speed int32, group uint8, d Domain) Entity {
	return Entity{
		ID: 1, X: 0, Y: 0, TargetX: 20, TargetY: 0, HasTarget: true,
		Domain: d, Speed: speed, GroupSpeed: group,
	}
}

// TestTheGroupTermReplacesTheClassSpeedInEveryDomain is the reader's whole
// contract: a nonzero term is the speed the law composes, and the law's two arms
// both see it — the ground arm through its multiplier, tilt and cost divide, the
// other two through the identity.
//
// The expectation is computed from the law itself rather than from a table of
// tick counts, because what is under test is WHICH SPEED reaches rateOf and not
// what rateOf does with one; a literal here would be re-asserting 0056.
func TestTheGroupTermReplacesTheClassSpeedInEveryDomain(t *testing.T) {
	const (
		classSpeed = 32
		groupSpeed = 8
	)
	for _, d := range []Domain{DomainGround, DomainGhost, DomainAir} {
		w := mustWorld(t, 1, grBounds, []Entity{grEntity(classSpeed, groupSpeed, d)})
		Step(w, nil)

		got := w.Entities()[0]
		want := transitOf(rateOf(d, groupSpeed, 0, 0, 0, 0), false)
		if int32(got.TransitTotal) != want {
			t.Errorf("domain %d: a mover of class speed %d in a group of %d crossed in %d tick(s), "+
				"and the law at the group's own speed is %d — the term did not replace the class speed",
				uint8(d), classSpeed, groupSpeed, got.TransitTotal, want)
		}
		if unwanted := transitOf(rateOf(d, classSpeed, 0, 0, 0, 0), false); int32(got.TransitTotal) == unwanted {
			t.Errorf("domain %d: the transit is what the CLASS speed alone would give (%d) — "+
				"the two speeds are not separated by this fixture", uint8(d), unwanted)
		}
	}
}

// TestAGroupTermIsARateEvenWithNoClassSpeed pins the composition order that the
// widening rule and the term together imply. A speed of zero or less is a mover
// with no rate of its own — one cell a tick — but the law reads the group source
// first and takes the class one only when the group byte is zero, so such a
// mover under a group term is rated after all.
//
// It is not a corner: the minimum a formation order stores is taken over the
// members' speeds with a signed comparison, so a member whose own speed is
// negative is exactly how a group can carry a term its bearer could not.
func TestAGroupTermIsARateEvenWithNoClassSpeed(t *testing.T) {
	for _, speed := range []int32{0, -1, -3} {
		w := mustWorld(t, 1, grBounds, []Entity{grEntity(speed, 19, DomainGround)})
		Step(w, nil)

		got := w.Entities()[0]
		if got.TransitTotal == 0 {
			t.Errorf("class speed %d with a group term of 19 owes no transit at all — "+
				"it was read as unrated, so the term was not consulted", speed)
		}
		if want := transitOf(rateOf(DomainGround, 19, 0, 0, 0, 0), false); int32(got.TransitTotal) != want {
			t.Errorf("class speed %d with a group term of 19 crossed in %d tick(s), want %d",
				speed, got.TransitTotal, want)
		}
	}
}

func TestNoTermIsStillTheClassSpeed(t *testing.T) {
	w := mustWorld(t, 1, grBounds, []Entity{grEntity(19, 0, DomainGround)})
	Step(w, nil)

	got := w.Entities()[0]
	if want := transitOf(rateOf(DomainGround, 19, 0, 0, 0, 0), false); int32(got.TransitTotal) != want {
		t.Errorf("a mover with no group term crossed in %d tick(s), and its own speed gives %d",
			got.TransitTotal, want)
	}
}

// TestArrivingDoesNotClearTheGroupTerm is the defect, asserted as behaviour.
//
// It is written as an assertion on the CURRENT state rather than as a comment,
// so that the day someone lifts it — one call to clearGroupSpeed at restAt —
// this test fails and the change has to be argued for rather than noticed.
func TestArrivingDoesNotClearTheGroupTerm(t *testing.T) {
	w := mustWorld(t, 1, grBounds, []Entity{{
		ID: 1, X: 0, Y: 0, TargetX: 1, TargetY: 0, HasTarget: true, Speed: 63, GroupSpeed: 19,
	}})
	for tick := 0; tick < 40 && w.Entities()[0].HasTarget; tick++ {
		Step(w, nil)
	}

	got := w.Entities()[0]
	if got.HasTarget {
		t.Fatalf("the mover never arrived: it is at (%d,%d) holding (%d,%d)",
			got.X, got.Y, got.TargetX, got.TargetY)
	}
	if got.GroupSpeed != 19 {
		t.Errorf("after arriving the group term is %d, want 19 — arriving is NOT one of the "+
			"three sites that clear it, and that is the behaviour being reproduced", got.GroupSpeed)
	}
}

// TestGivingUpDoesNotClearTheGroupTerm is the same absence at the other clearing
// site: a unit that spends its stall count and drops an order it cannot walk
// keeps the term, because giving up unlinks it from nothing.
//
// The fixture seals the mover in with a blocking neighbour rather than with
// terrain, so the failure is a NEAR search's — the one that stalls — and not a
// far search's, which would end the order in its first tick.
func TestGivingUpDoesNotClearTheGroupTerm(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 3, Height: 1}, []Entity{
		{ID: 1, X: 0, Y: 0, TargetX: 2, TargetY: 0, HasTarget: true, Speed: 19, GroupSpeed: 8},
		{ID: 2, X: 1, Y: 0},
	})
	for tick := 0; tick < stallLimit+2 && w.Entities()[0].HasTarget; tick++ {
		Step(w, nil)
	}

	got := w.Entities()[0]
	if got.HasTarget {
		t.Fatalf("the mover never gave up: stall %d", got.Stall)
	}
	if got.GroupSpeed != 8 {
		t.Errorf("after giving up the group term is %d, want 8", got.GroupSpeed)
	}
}

// TestAPlainOrderClearsTheGroupTerm is the first of the three writers. There is
// no non-group player move in what is being reconstructed — every player order
// allocates a group, and a fresh group's term is zero — so a plain move-to
// dropping the term is that order's own behaviour rather than a rule of ours.
func TestAPlainOrderClearsTheGroupTerm(t *testing.T) {
	w := mustWorld(t, 1, grBounds, []Entity{{ID: 1, X: 0, Y: 0, Speed: 19, GroupSpeed: 8}})
	Step(w, []Command{{Entity: 1, X: 5, Y: 0}})

	if got := w.Entities()[0]; got.GroupSpeed != 0 {
		t.Errorf("after a plain move order the group term is %d, want 0 — a plain order is a "+
			"fresh group of one, and a fresh group carries no term", got.GroupSpeed)
	}
}

// TestBeingFelledClearsTheGroupTerm is the third writer, and the distinction it
// pins is the one a reader most easily gets backwards: what death drops is the
// MEMBER'S link to its group, never the group's own byte. So the unit that dies
// loses the term and the units it was walking with keep it.
//
// Both blows are exercised, because one rule in one place is only one rule if
// both arms reach it.
func TestBeingFelledClearsTheGroupTerm(t *testing.T) {
	cases := []struct {
		name string
		cmd  Command
	}{
		{"killed", Command{Kind: KindKill, Entity: 1}},
		{"damaged to death", Command{Kind: KindDamage, Entity: 1, X: 50}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustWorld(t, 1, grBounds, []Entity{
				{ID: 1, X: 0, Y: 0, HP: 10, MaxHP: 10, Speed: 19, GroupSpeed: 8},
				{ID: 2, X: 2, Y: 0, HP: 10, MaxHP: 10, Speed: 40, GroupSpeed: 8},
			})
			Step(w, []Command{c.cmd})

			ents := w.Entities()
			if ents[0].GroupSpeed != 0 {
				t.Errorf("the felled unit's group term is %d, want 0 — a felled member is "+
					"unlinked from its group", ents[0].GroupSpeed)
			}
			if ents[1].GroupSpeed != 8 {
				t.Errorf("the survivor's group term is %d, want 8 — the group keeps the "+
					"departed member's speed", ents[1].GroupSpeed)
			}
		})
	}
}

// TestTheConstructorNormalisesATermOnAUnitThatIsNotAlive puts the term in the
// same relation to the constructor as the target and the transit pair already
// are: normalised here, refused by the decoder, so the constructor cannot build
// a world its own byte form will not read back.
func TestTheConstructorNormalisesATermOnAUnitThatIsNotAlive(t *testing.T) {
	cases := []struct {
		name   string
		hp, mx int32
	}{
		{"dead", -1, 10},
		{"downed", 0, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustWorld(t, 1, grBounds, []Entity{
				{ID: 1, X: 0, Y: 0, HP: c.hp, MaxHP: c.mx, Speed: 19, GroupSpeed: 8},
			})
			if got := w.Entities()[0]; got.GroupSpeed != 0 {
				t.Errorf("a %s unit was built holding a group term of %d, want 0", c.name, got.GroupSpeed)
			}
		})
	}
}

func TestTheGroupTermSurvivesARoundTripAndDistinguishesTwoWorlds(t *testing.T) {
	build := func(term uint8) *World {
		return mustWorld(t, 7, grBounds, []Entity{
			{ID: 1, X: 0, Y: 0, TargetX: 20, TargetY: 0, HasTarget: true, Speed: 40, GroupSpeed: term},
			{ID: 2, X: 5, Y: 0, Speed: 40, GroupSpeed: term},
		})
	}

	w := build(8)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := snap(&back), snap(w); !equalState(got, want) {
		t.Errorf("the round trip gives\n %+v\nfrom\n %+v", got, want)
	}
	if got, want := back.Hash(), w.Hash(); got != want {
		t.Errorf("the round trip hashes %#016x, the original %#016x", got, want)
	}

	if a, b := build(8).Hash(), build(9).Hash(); a == b {
		t.Errorf("two worlds differing only in the group term both hash %#016x — the byte is outside "+
			"the digest, and a save would resume at a speed nothing recorded", a)
	}

	// And the decoded world walks at the term, not at the class speed: the byte
	// came back meaning what it meant.
	Step(&back, nil)
	if got, want := int32(back.Entities()[0].TransitTotal),
		transitOf(rateOf(DomainGround, 8, 0, 0, 0, 0), false); got != want {
		t.Errorf("the decoded mover crossed in %d tick(s), want the term's %d", got, want)
	}
}
