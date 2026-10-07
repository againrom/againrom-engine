package sim

// The world decoded beside itself.
//
// The whole of what a resumed world needs is what its bytes carry. A stored
// route is now part of that — it is canonical state and the form carries it —
// so the claim has grown rather than moved: nothing a tick reads may live
// outside the encoding, routes included. The only way to measure it is to throw
// away everything that is not in the form: at every tick the running world is
// marshalled, a second world is built by decoding those bytes and nothing else,
// and the two are stepped on together. Routing state the encoding cannot reach
// then shows up as a divergence instead of as a pin that passes.
//
// The second world is never constructed to be equal — it is only ever decoded.
// A constructed twin would agree with the first by having been given the same
// inputs, which is a different and much weaker fact.

import "testing"

// replayUnits is a mix chosen so that every field the byte form carries is in
// motion at some point in the run: units that walk, one that arrives and then
// holds nothing, one whose order is off the map, which no far search can serve,
// and A PAIR THAT FIGHT — one striking, one struck, so the victim, the phase, the
// count owed and the seven numbers a blow reads all move across the crossing,
// and the world's own GENERATOR moves with them. Without that pair the claim in
// this file's own doc comment would be false the moment the record widened.
//
// Their cadence is short and their damage small, so the run above sees several
// whole cycles and the victim survives all of them: a corpse would stop the
// fields moving halfway through.
func replayUnits() []Entity {
	return []Entity{
		{ID: 1, X: 0, Y: 0},
		{ID: 2, X: 0, Y: 8},
		{ID: 3, X: 2, Y: 2},
		{ID: 4, X: 6, Y: 6},
		{ID: 7, X: 7, Y: 7},
		{ID: 8, X: 4, Y: 0, HP: 500, MaxHP: 500,
			AttackCharge: 3, AttackRelax: 1, ToHit: 20, DamageBase: 2, DamageSpread: 5},
		{ID: 9, X: 5, Y: 0, HP: 500, MaxHP: 500,
			AttackCharge: 2, AttackRelax: 2, ToHit: 5, Defence: 10, Absorption: 1,
			DamageBase: 1, DamageSpread: 3, AlwaysHits: true},
	}
}

func replayOrders() []Command {
	return []Command{
		{Entity: 1, X: 8, Y: 0},  // crosses the wall through its gap
		{Entity: 2, X: 8, Y: 8},  // crosses it the other way round
		{Entity: 3, X: 3, Y: 3},  // arrives at once, then holds nothing
		{Entity: 4, X: 0, Y: 6},  // crosses back
		{Entity: 7, X: 20, Y: 7}, // off the map: the order ends in the tick that finds it
		{Kind: KindAttack, Entity: 8, X: 9},
		{Kind: KindAttack, Entity: 9, X: 8},
	}
}

func TestADecodedWorldStepsOnToTheSameDigests(t *testing.T) {
	const ticks = 20

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, wallBounds, mode, wallGrid(), replayUnits())

		for tick := 1; tick <= ticks; tick++ {
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("mode %v tick %d: MarshalBinary: %v", mode, tick, err)
			}

			// The twin is built by decoding and by nothing else, over a world
			// that shares neither its bounds, its mode, its grid nor its units.
			twin := mustWorldGrid(t, 99, Bounds{Width: 1, Height: 1}, ModeCanonical, nil, nil)
			if err := twin.UnmarshalBinary(form); err != nil {
				t.Fatalf("mode %v tick %d: UnmarshalBinary: %v", mode, tick, err)
			}
			if got := twin.Hash(); got != w.Hash() {
				t.Fatalf("mode %v tick %d: the decoded world hashes %#016x, the original %#016x",
					mode, tick, got, w.Hash())
			}

			var cmds []Command
			if tick == 1 {
				cmds = replayOrders()
			}
			Step(w, cmds)
			Step(twin, cmds)

			if got, want := twin.Hash(), w.Hash(); got != want {
				t.Fatalf("mode %v: after tick %d the decoded world hashes %#016x and the original %#016x",
					mode, tick, got, want)
			}
			// The digest is one number over the whole form, so an equal digest
			// is already a strong statement; the fields are compared beside it
			// so that a failure says WHICH one moved.
			a, b := w.Entities(), twin.Entities()
			if len(a) != len(b) {
				t.Fatalf("mode %v tick %d: %d entities against %d", mode, tick, len(a), len(b))
			}
			for i := range a {
				if a[i] != b[i] {
					t.Fatalf("mode %v tick %d: entity %d is %+v in the original and %+v in the decoded world",
						mode, tick, a[i].ID, a[i], b[i])
				}
				// And the routes cell for cell. A twin that had lost its routes
				// would still agree on every entity field this tick; what it would
				// have lost is where each unit is going NEXT, which the tick after
				// puts into the entity fields too.
				checkRoute(t, "the decoded world's route", twin.routes[i], true, w.routes[i])
			}
			if w.Tick() != twin.Tick() || w.Bounds() != twin.Bounds() {
				t.Fatalf("mode %v tick %d: the pair disagree on tick or bounds", mode, tick)
			}
		}

		// The run has to be one where something happened, or the pin above is a
		// pin on two worlds standing still.
		seen := map[string]bool{}
		for _, e := range w.Entities() {
			switch {
			case e.ID == 7:
				seen["gave up"] = e == Entity{ID: 7, X: 7, Y: 7, ActorState: actorStateGuard, Reach: 1, PostX: 7, PostY: 7}
			case e.HasTarget:
			default:
				seen["arrived"] = true
			}
		}
		if !seen["gave up"] {
			t.Errorf("mode %v: the off-map order did not end cleared with no residue", mode)
		}
		if !seen["arrived"] {
			t.Errorf("mode %v: no unit arrived, so the run measured nothing", mode)
		}
	}
}

func TestTheStallCountReachesTheDigest(t *testing.T) {
	w := boxedWorld(t)
	Step(w, boxedOrder())

	prev := w.Hash()
	for tick := 2; tick < stallLimit; tick++ {
		Step(w, nil)
		got := w.Hash()
		if got == prev {
			t.Fatalf("tick %d: the digest is unchanged at %#016x, but the stall count rose", tick, got)
		}
		prev = got
	}
}
