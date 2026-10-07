package sim

// Who is counted, who advances, and the replay that carries both across a decode.
//
// Every body holds its cell while it dwells. After that minimum window, a body
// ordinary Heal can still restore keeps the cell until revival or the -10
// finished-body floor; a terminal corpse releases it. Each cell is asked two
// questions — may a mover STAND on it, and may a mover WALK THROUGH it — because
// a rule that answered the first correctly and the second wrongly would leave a
// corridor sealed by something that can be stood on.

import (
	"bytes"
	"testing"
)

// dwnCorridor is a one-row map, which is what makes "route through" a question
// with an answer: on open ground a mover simply goes round, and every case here
// would pass under a rule that never freed a cell at all.
var dwnCorridor = Bounds{Width: 5, Height: 1}

// dwnBlocked is the fixture: a mover at (0,0), a unit standing at (2,0) driven
// to the state the case names, and the order given to the mover.
//
// The blocker is put into its state by a BLOW rather than built into it, so the
// world under test is one this package can actually reach: a downed unit exists
// only as the result of damage, and a world handed one at construction has that
// unit's order cleared for a reason unrelated to what is measured here.
func dwnBlocked(t *testing.T, blocker Command, want life) *World {
	t.Helper()
	w := mustWorld(t, 1, dwnCorridor, dwnPair())
	Step(w, []Command{blocker})
	got, n := state(occEntity(t, w, 2))
	if n != 1 || got != want {
		t.Fatalf("fixture: the blocker is %s, want %s", got, want)
	}
	return w
}

// dwnDrive runs several ticks with the order given on the first, and returns the
// two units as they finally stand. Several rather than one because a rule that
// merely DELAYED a mover would pass a single-tick check, and because the
// immobile unit has to be shown still immobile rather than late.
func dwnDrive(t *testing.T, w *World, order Command, ticks int) (mover, blocker Entity) {
	t.Helper()
	for k := 0; k < ticks; k++ {
		if k == 0 {
			Step(w, []Command{order})
		} else {
			Step(w, nil)
		}
		if got := occEntity(t, w, 2); got.X != 2 || got.Y != 0 {
			t.Fatalf("tick %d: the blocker walked to (%d,%d) — a unit that is not alive is advanced by nothing",
				k+1, got.X, got.Y)
		}
	}
	return occEntity(t, w, 1), occEntity(t, w, 2)
}

// dwnDwell is the blocker's dying time in every fixture here: how many ticks its
// body holds the cell it fell on before it is torn down.
const dwnDwell = 12

func TestBodiesHoldTheirCellThroughDwellAndRestorableBodiesKeepIt(t *testing.T) {
	finish := Command{Kind: KindDamage, Entity: 2, X: 110}
	kill := Command{Kind: KindKill, Entity: 2}
	down := Command{Kind: KindDamage, Entity: 2, X: 100}

	for _, blow := range []struct {
		name       string
		cmd        Command
		state      life
		restorable bool
	}{
		{"finished at minus ten", finish, lifeDead, false},
		{"killed to minus one", kill, lifeDead, true},
		{"damaged to exactly zero", down, lifeDowned, true},
	} {
		for _, order := range []struct {
			name string
			cmd  Command
			goal int32
		}{
			{"ordered onto the body's cell", Command{Entity: 1, X: 2, Y: 0}, 2},
			{"ordered past the body", Command{Entity: 1, X: 4, Y: 0}, 4},
		} {
			t.Run(blow.name+", "+order.name+", inside the dwell", func(t *testing.T) {
				w := dwnBlocked(t, blow.cmd, blow.state)
				mover, _ := dwnDrive(t, w, order.cmd, dwnDwell-2)
				if mover.X != 1 || mover.Y != 0 {
					t.Errorf("the mover is at (%d,%d), want (1,0) — a dwelling body is a wall "+
						"in a one-row corridor", mover.X, mover.Y)
				}
				if !mover.HasTarget {
					t.Error("the mover holds no target — it is being refused, not finished, " +
						"so it must still be trying")
				}
			})
			t.Run(blow.name+", "+order.name+", past the dwell", func(t *testing.T) {
				w := dwnBlocked(t, blow.cmd, blow.state)
				mover, _ := dwnDrive(t, w, order.cmd, dwnDwell+8)
				if blow.restorable {
					if mover.X != 1 || mover.Y != 0 {
						t.Errorf("the mover is at (%d,%d), want (1,0) — a restorable body keeps its cell after dwell",
							mover.X, mover.Y)
					}
					return
				}
				if mover.X != order.goal || mover.Y != 0 {
					t.Errorf("the mover is at (%d,%d), want (%d,0) — a torn-down body holds "+
						"no ground", mover.X, mover.Y, order.goal)
				}
				if mover.HasTarget {
					t.Errorf("the mover arrived still holding the target (%d,%d)",
						mover.TargetX, mover.TargetY)
				}
			})
		}
	}
}

// TestACorpseFreesItsCellInTheTickItDies is C-2's "at once" measured where the
// word bites: the blow and the order arrive in ONE command slice, and the mover
// is walking before that tick is over.
//
// It is what fixes WHERE the occupancy plane is seeded. Built before the
// commands are applied, the plane would still be counting a unit the same tick
// has killed, and every case above would pass — a corpse freed one tick late is
// a corpse freed, and only a mover that asked within that tick can tell. The
// control beside it is the same slice with the blow left out, where the mover is
// refused.
func TestACorpseFreesItsCellInTheTickItDies(t *testing.T) {
	onto := Command{Entity: 1, X: 2, Y: 0}
	past := Command{Entity: 1, X: 4, Y: 0}
	kill := Command{Kind: KindKill, Entity: 2}

	cases := []struct {
		name      string
		downFirst bool
		slice     []Command
	}{
		{"the blow ahead of the order, onto the cell", false, []Command{kill, onto}},
		// Past the cell rather than onto it: the mover has to be routed THROUGH a
		// corpse within the tick, which is a different question of the plane than
		// standing on one.
		{"the blow ahead of the order, past the cell", false, []Command{kill, past}},
		// Behind it instead, which must not matter: phase 1 finishes every command
		// in the slice before the walk begins, so the seed sees the same world
		// either way.
		{"the order ahead of the blow", false, []Command{onto, kill}},
		// And the blow that ends a unit already DOWNED, which is the transition
		// the whole middle state exists for: the cell is held right up to it and
		// free inside the tick it lands.
		{"the blow that ends a downed unit", true,
			[]Command{{Kind: KindDamage, Entity: 2, X: 1}, onto}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorld(t, 1, dwnCorridor, dwnPair())
			if tc.downFirst {
				w = dwnBlocked(t, Command{Kind: KindDamage, Entity: 2, X: 100}, lifeDowned)
			}
			Step(w, tc.slice)

			if got := occEntity(t, w, 2); !got.Dead() {
				t.Fatalf("the blocker is at %d/%d and is not dead", got.HP, got.MaxHP)
			}
			if got := occEntity(t, w, 1); got.X != 1 || got.Y != 0 {
				t.Errorf("the mover is at (%d,%d) after the tick that felled what stood in its way, "+
					"want (1,0) — a corpse frees its cell in the tick it dies, not in the one after",
					got.X, got.Y)
			}
		})
	}

	// The control. Without it every case above would pass under a build whose
	// movers walk through anything at all.
	//
	// It runs SIX ticks and measures the blocker's own cell, because one tick no
	// longer separates the two: a mover refused that cell settles for the one
	// before it, which is where a corpse also leaves it after one tick. What only
	// the corpse permits is standing on (2,0) at all, and six ticks is more than
	// a five-cell corridor needs.
	for _, order := range []Command{onto, past} {
		held := mustWorld(t, 1, dwnCorridor, dwnPair())
		mover, _ := dwnDrive(t, held, order, 6)
		if mover.X != 1 {
			t.Fatalf("the mover reached (%d,%d) with the blocker still alive — the cases above then "+
				"witness nothing about the blow", mover.X, mover.Y)
		}
	}
}

// dwnPair is the corridor's two units at full health, fresh for each case.
func dwnPair() []Entity {
	return []Entity{
		{ID: 1, X: 0, Y: 0, HP: 100, MaxHP: 100},
		// The blocker's DYING TIME is what this file now measures against, and
		// twelve ticks is chosen to be longer than the four a mover needs to
		// cross this corridor and short enough that a run of a few dozen ticks
		// sees the far side of it.
		{ID: 2, X: 2, Y: 0, HP: 100, MaxHP: 100, DyingTime: dwnDwell},
	}
}

func TestAUnitThatIsNotAliveIsGivenNoTargetAtAll(t *testing.T) {
	for _, tc := range []struct {
		name string
		blow Command
		want life
	}{
		{"a corpse", Command{Kind: KindKill, Entity: 2}, lifeDead},
		{"a downed unit", Command{Kind: KindDamage, Entity: 2, X: 100}, lifeDowned},
	} {
		t.Run(tc.name+", ordered on a later tick", func(t *testing.T) {
			w := dwnBlocked(t, tc.blow, tc.want)
			quiet := dwnBlocked(t, tc.blow, tc.want)
			Step(w, []Command{{Entity: 2, X: 4, Y: 0}})
			Step(quiet, nil)
			dwnNoOrder(t, w, quiet, 2)
		})
		t.Run(tc.name+", ordered behind the blow in one slice", func(t *testing.T) {
			w := mustWorld(t, 1, dwnCorridor, dwnPair())
			quiet := mustWorld(t, 1, dwnCorridor, dwnPair())
			Step(w, []Command{tc.blow, {Entity: 2, X: 4, Y: 0}})
			Step(quiet, []Command{tc.blow})
			if got, want := w.Hash(), quiet.Hash(); got != want {
				t.Errorf("the order behind the blow hashes %#016x and the blow alone %#016x — "+
					"the order must have been ignored outright", got, want)
			}
			dwnNoOrder(t, w, quiet, 2)
		})
	}
}

// dwnNoOrder is the check both arms above make: the unit at id holds no target,
// no residue in the coordinates, no stall and no route, and the world is field
// for field and digest for digest where a world that took no such order stands.
func dwnNoOrder(t *testing.T, w, quiet *World, id EntityID) {
	t.Helper()
	e := occEntity(t, w, id)
	if e.HasTarget || e.TargetX != 0 || e.TargetY != 0 || e.Stall != 0 {
		t.Errorf("unit %d is %+v — an order for a unit that is not alive leaves no target and no residue",
			id, e)
	}
	i := indexOfEntity(w.entities, id)
	if i >= 0 && len(w.routes[i]) != 0 {
		t.Errorf("unit %d holds the route %s", id, fmtRoute(w.routes[i]))
	}
	if got, want := snap(w), snap(quiet); !equalState(got, want) {
		t.Errorf("the world is\n %+v\nwant\n %+v", got, want)
	}
	if got, want := w.Hash(), quiet.Hash(); got != want {
		t.Errorf("the world hashes %#016x and the one that took no such order %#016x", got, want)
	}
}

// TestTheMoveLoopRefusesToAdvanceAUnitThatIsNotAlive reaches the walk's OWN
// test, which no command can reach: the arm that applies a move-to refuses to
// put a target on a unit that is not alive, so a world holding one cannot be
// built through the exported surface at all.
//
// It is built through the fields instead, which is why these tests live in
// package sim.
func TestTheMoveLoopRefusesToAdvanceAUnitThatIsNotAlive(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hp, maxHP int32
	}{
		{"a corpse", -1, 100},
		{"a downed unit", 0, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{{ID: 1, X: 0, Y: 0}})
			// The target is written through the field, and so is the health: a
			// world holding this pair is one no constructor, decoder or command
			// produces.
			w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 6, 6, true
			w.entities[0].HP, w.entities[0].MaxHP = tc.hp, tc.maxHP

			for k := 0; k < 4; k++ {
				Step(w, nil)
				got := occEntity(t, w, 1)
				if got.X != 0 || got.Y != 0 {
					t.Fatalf("tick %d: the unit walked to (%d,%d)", k+1, got.X, got.Y)
				}
				if len(w.routes[0]) != 0 {
					t.Fatalf("tick %d: the unit was given the route %s — it was searched for",
						k+1, fmtRoute(w.routes[0]))
				}
			}
		})
	}
}

// ---------------------------------------------------------------- AC-8

// TestTheHealthPairIsCanonicalInFormAndDigest is AC-8's first clause: two worlds
// differing in one unit's health, or in one unit's maximum, differ in form and
// in digest; two built the same way agree in both.
//
// The pair is changed on a unit holding NO order, so what moves is the health
// alone. The identical pair is the control the other two are read against —
// without it a build whose form depended on nothing at all would pass by
// differing everywhere.
func TestTheHealthPairIsCanonicalInFormAndDigest(t *testing.T) {
	build := func(hp, maxHP int32) *World {
		return mustWorld(t, 0x1234abcd, Bounds{Width: 6, Height: 6}, []Entity{
			{ID: 1, X: 1, Y: 1, TargetX: 4, TargetY: 4, HasTarget: true, HP: 50, MaxHP: 50},
			{ID: 2, X: 3, Y: 3, HP: hp, MaxHP: maxHP},
		})
	}

	base, sameAsBase := build(70, 90), build(70, 90)
	otherHP, otherMax := build(71, 90), build(70, 91)

	form := func(w *World) []byte {
		b, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		return b
	}

	if !bytes.Equal(form(base), form(sameAsBase)) || base.Hash() != sameAsBase.Hash() {
		t.Errorf("two worlds built alike differ:\n % x\n % x", form(base), form(sameAsBase))
	}
	for _, tc := range []struct {
		what string
		w    *World
	}{
		{"one health", otherHP},
		{"one maximum", otherMax},
	} {
		if bytes.Equal(form(base), form(tc.w)) {
			t.Errorf("worlds differing in %s marshal alike:\n % x", tc.what, form(base))
		}
		if base.Hash() == tc.w.Hash() {
			t.Errorf("worlds differing in %s both hash %#016x", tc.what, base.Hash())
		}
	}
}

// dwnStart is the corpus: five units on open ground, ids non-adjacent so that a
// rule comparing a unit against its neighbour in the slice cannot answer
// anything here by accident. Two carry a health system and are the ones the
// blows reach; one has none, so a damage on it is a no-op inside a live stream
// rather than in a case built for it.
var dwnStart = []Entity{
	{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100},
	{ID: 4, X: 8, Y: 1, HP: 100, MaxHP: 100},
	{ID: 7, X: 1, Y: 8},
	{ID: 10, X: 8, Y: 8, HP: 40, MaxHP: 40},
	// Unit 13 is the one the blows walk down the state machine, and it is the one
	// carrying a DYING TIME: its cell is a wall for exactly as long as its body
	// dwells, and free from the tick that dwell runs out.
	{ID: 13, X: 4, Y: 4, HP: 100, MaxHP: 100, DyingTime: 8},
}

// dwnBounds is the corpus's map: small enough that the walks meet, large enough
// that a detour round a downed unit exists and a route is worth searching for.
var dwnBounds = Bounds{Width: 12, Height: 12}

// dwnSchedule mixes ALL THREE KINDS across a run: orders that send units across
// each other's paths, a unit damaged to downed and later past it, a unit killed
// outright, blows that are no-ops, an undefined kind, and orders given AFTER the
// blows so that a world cut before them is not walking a path already settled.
//
// Unit 13 is driven to zero at tick 4 and to the finished-body floor at tick
// 10. Its own dying time decides when that terminal body releases its cell —
// which is the one thing a run over this corpus can show that a run over a
// quiet world cannot.
var dwnSchedule = [][]Command{
	{{Entity: 1, X: 8, Y: 8}, {Entity: 4, X: 1, Y: 8}, {Entity: 10, X: 1, Y: 1}},
	nil,
	{{Kind: KindDamage, Entity: 13, X: 60}},
	{{Kind: KindDamage, Entity: 13, X: 40}, {Entity: 7, X: 8, Y: 4}},
	// Unit 13 is now at zero on (4,4), where it started and where it stays: it is
	// given no order in this whole schedule, so its cell is fixed. Unit 1 is sent
	// onto that cell and must not reach it while the body is still dwelling.
	{{Entity: 1, X: 4, Y: 4}},
	{{Kind: KindKill, Entity: 10}},
	{{Kind: KindDamage, Entity: 7, X: 25}, {Kind: KindDamage, Entity: 4, X: 0}},
	{{Kind: 9, Entity: 1, X: 3, Y: 3}, {Kind: KindKill, Entity: 99}},
	nil,
	// And now 13 reaches the -10 finished-body floor. Its dwell runs out two ticks
	// later and the cell is free; the same order is given again, and this time
	// somebody stands there.
	{{Kind: KindDamage, Entity: 13, X: 10}},
	{{Entity: 1, X: 4, Y: 4}, {Entity: 4, X: 4, Y: 4}},
	nil, nil, nil, nil, nil, nil, nil, nil, nil,
	nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
}

// TestTheCorpusCrossesADecodeAtEveryTick is AC-8's second clause and SC-6. The
// corpus is walked to QUIESCENCE under the stream above; at every tick it is
// marshalled, decoded into a second world, and that second world is stepped
// through the whole remainder of the schedule beside the first — digest for
// digest and form for form at EVERY later tick, not only the last.
//
// A comparison made at the end alone would pass a build whose two worlds parted
// and came back together, and a health that failed to cross a decode is exactly
// the shape that could: two units both standing still is the same picture
// whether one of them is a corpse or a wall.
func TestTheCorpusCrossesADecodeAtEveryTick(t *testing.T) {
	w := mustWorld(t, 0xc0ffee, dwnBounds, dwnStart)
	forms := [][]byte{dwnForm(t, w)}
	digests := []uint64{w.Hash()}

	seen := map[life]bool{}
	stalled := false
	sharedWhileDwelling, sharedAfterTeardown := false, false
	for _, cmds := range dwnSchedule {
		Step(w, cmds)
		ents := w.Entities()
		// Unit 13 is the one the blows walk down the state machine, and it is
		// given no order anywhere in the schedule — so its cell is fixed and "who
		// is standing on it" is a question about occupancy alone.
		body := ents[indexOfEntity(w.entities, 13)]
		dwelling := body.Decay == DecayFallen && body.Dwell > 0
		tornDown := body.Decay != DecayNone && body.Dwell == 0
		for _, e := range ents {
			got, _ := state(e)
			seen[got] = true
			stalled = stalled || e.Stall > 0
			if e.ID == body.ID || e.X != body.X || e.Y != body.Y {
				continue
			}
			sharedWhileDwelling = sharedWhileDwelling || dwelling
			sharedAfterTeardown = sharedAfterTeardown || tornDown
		}
		forms = append(forms, dwnForm(t, w))
		digests = append(digests, w.Hash())
	}

	// The run's claim to have exercised anything, on the test's own arithmetic.
	// All three states occurred, somebody was refused a move, and the world came
	// to rest — a corpus still walking at the end was not walked to quiescence
	// and says nothing about what a decode carries across.
	for _, l := range []life{lifeAlive, lifeDowned, lifeDead} {
		if !seen[l] {
			t.Fatalf("no unit was ever %s during the run", l)
		}
	}
	if !stalled {
		t.Fatal("no unit was ever refused a move; the run met no obstruction")
	}
	// The occupancy rule inside the run, and both halves of it. A mover is sent
	// onto unit 13's cell while its body is still dwelling and again after the
	// dwell has run out, so the corpus itself — not only the cases built for it —
	// parts the two sides of the teardown.
	if sharedWhileDwelling {
		t.Error("a unit stood on a dwelling body's cell during the run")
	}
	if !sharedAfterTeardown {
		t.Fatal("no unit ever stood on the torn-down body's cell; the run does not part the two sides")
	}
	for _, e := range w.Entities() {
		if e.HasTarget {
			t.Fatalf("the run ended with unit %d still walking to (%d,%d) — it did not reach quiescence",
				e.ID, e.TargetX, e.TargetY)
		}
	}

	for cut := range forms {
		var shadow World
		if err := shadow.UnmarshalBinary(forms[cut]); err != nil {
			t.Fatalf("cut at tick %d: UnmarshalBinary: %v", cut, err)
		}
		if got := shadow.Hash(); got != digests[cut] {
			t.Fatalf("cut at tick %d: the decoded world hashes %#016x, the first %#016x",
				cut, got, digests[cut])
		}
		for tick := cut; tick < len(dwnSchedule); tick++ {
			Step(&shadow, dwnSchedule[tick])
			if got, want := shadow.Hash(), digests[tick+1]; got != want {
				t.Fatalf("cut at tick %d, then tick %d: the decoded world hashes %#016x and the "+
					"first %#016x\n decoded %+v\n first   %+v",
					cut, tick+1, got, want, shadow.Entities(), forms[tick+1])
			}
			if got := dwnForm(t, &shadow); !bytes.Equal(got, forms[tick+1]) {
				t.Fatalf("cut at tick %d, then tick %d: the two forms differ\n % x\n % x",
					cut, tick+1, got, forms[tick+1])
			}
		}
	}
}

func dwnForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}
