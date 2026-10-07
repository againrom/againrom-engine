package sim

import (
	"reflect"
	"testing"
)

// The two hand-over arms: what they write, what they leave alone, and the
// differential that says implementing them armed no trigger.
//
// Every fixture here is a world and a hand-built script rather than a map. The
// arms take a group identifier and a roster slot, both of which are plain
// numbers on an entity in this tree, so nothing about them needs a map to be
// exercised — and a fixture that went through one would be measuring the binder
// as well.

// ownerWorld is a world of five entities in two groups, one of them FELLED, at
// owners no successful arm would leave in place. It is the fixture almost every
// case below spoils or fires an arm at.
//
// A group id resolves to one owner, so the members of one group share their
// owner here: 6 for group 4 and 5 for group 7. Neither is a player any arm below
// hands to, so "this entity was written" and "this entity was left alone" are
// never the same observation.
func ownerWorld(t *testing.T, s *Script) *World {
	t.Helper()
	return scriptWorld(t, s, []Entity{
		{ID: 1, HP: 5, MaxHP: 5, Group: 4, Owner: 6},
		{ID: 2, HP: 5, MaxHP: 5, Group: 4, Owner: 6},
		{ID: 3, HP: 0, MaxHP: 5, Group: 4, Owner: 6}, // downed, and still a member
		{ID: 4, HP: 5, MaxHP: 5, Group: 7, Owner: 5},
		{ID: 5, HP: 5, MaxHP: 5, Group: 7, Owner: 5},
	})
}

// owners reads the world's owners back by entity id.
func owners(w *World) map[EntityID]uint32 {
	out := make(map[EntityID]uint32, len(w.entities))
	for _, e := range w.Entities() {
		out[e.ID] = e.Owner
	}
	return out
}

// fireOnce is a script whose one trigger holds on the first pass and runs the
// instants given, once. Its condition is a constant compared against itself,
// which is an arm this build already evaluates — so the trigger is live and
// evaluated, and nothing about these fixtures depends on an arm under test.
func fireOnce(t *testing.T, is []ScriptInstant, slots ...int32) *Script {
	t.Helper()
	return mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
		is,
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
			Instants: acts(slots...), Once: true, Latch: 0}})
}

// runPass advances one full script cycle and asserts the trigger fired, so a
// case that measured nothing cannot pass by measuring nothing.
func runPass(t *testing.T, w *World) {
	t.Helper()
	scriptTicks(w, 32, nil)
	if !w.ScriptLatched(0) {
		t.Fatal("the trigger did not fire; every case here measures what happens when it does")
	}
}

// ---------------------------------------------------------------- AC-4

func TestTheGroupArmWritesEveryMemberAndNobodyElse(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	runPass(t, w)

	want := map[EntityID]uint32{1: 9, 2: 9, 3: 9, 4: 5, 5: 5}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after the group arm the owners are %v, want %v", got, want)
	}
	// The felled member spelled out, because "all three" above would still read
	// as passing if the map comparison were ever loosened.
	for _, e := range w.Entities() {
		if e.ID == 3 && e.Alive() {
			t.Fatal("entity 3 is alive; this fixture is about a member that is not")
		}
	}
}

func TestAGroupNoEntityCarriesLeavesEveryOwnerAlone(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4242, HasGroup: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	before := owners(w)
	runPass(t, w)

	if got := owners(w); !reflect.DeepEqual(got, before) {
		t.Errorf("a group no entity carries left the owners at %v, want the %v they stood at", got, before)
	}
}

// TestTwoGroupArmsInOneTriggerEachWriteTheirOwn is AC-4's third clause: two
// instants of the same arm naming different groups, in one trigger, each writing
// its own members and neither the other's.
//
// A dispatch that had somehow made the arm stateful, or that resolved its group
// once per trigger rather than once per instant, passes every case above and
// fails this one.
func TestTwoGroupArmsInOneTriggerEachWriteTheirOwn(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveGroup, Group: 7, HasGroup: true, Player: 8, HasPlayer: true},
	}, 0, 1)
	w := ownerWorld(t, s)
	runPass(t, w)

	want := map[EntityID]uint32{1: 9, 2: 9, 3: 9, 4: 8, 5: 8}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after two group arms the owners are %v, want %v", got, want)
	}
}

func TestTheGroupArmDoesNotDependOnStorageOrder(t *testing.T) {
	t.Parallel()

	ents := []Entity{
		{ID: 1, HP: 5, MaxHP: 5, Group: 4, Owner: 1},
		{ID: 2, HP: 5, MaxHP: 5, Group: 7, Owner: 2},
		{ID: 3, HP: 5, MaxHP: 5, Group: 4, Owner: 3},
	}
	reversed := []Entity{ents[2], ents[1], ents[0]}

	run := func(in []Entity) *World {
		s := fireOnce(t, []ScriptInstant{
			{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
		}, 0)
		w := scriptWorld(t, s, in)
		runPass(t, w)
		return w
	}
	a, b := run(ents), run(reversed)
	if a.Hash() != b.Hash() {
		t.Errorf("the same entities in two orders hash %#016x and %#016x after the arm ran",
			a.Hash(), b.Hash())
	}
}

// ---------------------------------------------------------------- AC-5

// TestTheUnitArmWritesItsOneEntity is AC-5's first clause.
//
// The named entity is in a group of three, so an arm that resolved through the
// group instead of the id would write three owners and this would say so.
func TestTheUnitArmWritesItsOneEntity(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 2, HasUnit: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	runPass(t, w)

	want := map[EntityID]uint32{1: 6, 2: 9, 3: 6, 4: 5, 5: 5}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after the unit arm the owners are %v, want %v", got, want)
	}
}

func TestTheUnitArmOnAnEntityTheWorldNoLongerHolds(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 404, HasUnit: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	before := owners(w)
	runPass(t, w)

	if got := owners(w); !reflect.DeepEqual(got, before) {
		t.Errorf("a unit the world does not hold left the owners at %v, want the %v they stood at",
			got, before)
	}
}

func TestTheUnitArmWritesAFelledEntity(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 3, HasUnit: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	runPass(t, w)

	want := map[EntityID]uint32{1: 6, 2: 6, 3: 9, 4: 5, 5: 5}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after handing over a felled unit the owners are %v, want %v", got, want)
	}
}

func TestNeitherArmPanicsOverAnEmptyWorld(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 0, HasGroup: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveUnit, Unit: 0, HasUnit: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveGroup, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveUnit, Player: 9, HasPlayer: true},
	}, 0, 1, 2, 3)
	w := scriptWorld(t, s, nil)
	runPass(t, w)

	if n := len(w.Entities()); n != 0 {
		t.Errorf("the empty world holds %d entities", n)
	}
}

// ---------------------------------------------------------------- AC-6

func TestAnArmNamingNothingWritesNothing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		in   ScriptInstant
	}{
		{"the group arm with no player", ScriptInstant{
			Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true}},
		{"the unit arm with no player", ScriptInstant{
			Op: ScriptInstantGiveUnit, Unit: 2, HasUnit: true}},
		{"the group arm with no group", ScriptInstant{
			Op: ScriptInstantGiveGroup, Player: 9, HasPlayer: true}},
		{"the unit arm with no unit", ScriptInstant{
			Op: ScriptInstantGiveUnit, Player: 9, HasPlayer: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := ownerWorld(t, fireOnce(t, []ScriptInstant{tc.in}, 0))
			before := owners(w)
			runPass(t, w)
			got := owners(w)
			if !reflect.DeepEqual(got, before) {
				t.Errorf("owners are %v, want the %v they stood at", got, before)
			}
			for id, o := range got {
				if o == 0 {
					t.Errorf("entity %d was cleared to 0; an absent reference writes NOTHING, "+
						"which is not the same as assigning nobody", id)
				}
			}
		})
	}
}

// ---------------------------------------------------------------- AC-7

// TestAScriptHoldingBothArmsCrossesTheFormAndStepsOnAlike is AC-7 end to end
// over a world the arms have already run on: every owner and every reference
// crosses record for record, and the two worlds step on to identical digests.
//
// The second half is what a round-trip comparison alone would miss. The arms are
// one-shot here, so a decoded world whose LATCHES came back wrong would re-run
// them — and against a world whose owners they had already written, re-running
// them changes nothing observable. Stepping both on and comparing digests is
// what would catch it.
func TestAScriptHoldingBothArmsCrossesTheFormAndStepsOnAlike(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveUnit, Unit: 4, HasUnit: true, Player: 8, HasPlayer: true},
	}, 0, 1)
	w := ownerWorld(t, s)
	runPass(t, w)

	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := back.Entities(), w.Entities(); !reflect.DeepEqual(got, want) {
		t.Errorf("the round trip changed the entities:\n got %+v\nwant %+v", got, want)
	}
	if got, want := back.Script().Instants(), s.Instants(); !reflect.DeepEqual(got, want) {
		t.Errorf("the round trip changed the instants:\n got %+v\nwant %+v", got, want)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
	for i := 0; i < 64; i++ {
		Step(w, nil)
		Step(&back, nil)
		if back.Hash() != w.Hash() {
			t.Fatalf("tick %d after the cut: the resumed world hashes %#016x and the original %#016x",
				i+1, back.Hash(), w.Hash())
		}
	}
}

// ---------------------------------------------------------------- AC-8

func TestImplementingTheseArmsArmedNoTrigger(t *testing.T) {
	t.Parallel()

	const notAnArm = scriptCheckSentinelOp // a check this build does not evaluate
	const notAnInstant int32 = 11          // an instant it does not run: the item transfer
	if scriptCheckSupported(notAnArm) || scriptInstantSupported(ScriptInstant{Op: notAnInstant}) {
		t.Fatal("an arm this differential needs unimplemented is implemented")
	}

	// One shape of script, twice: the instants differ and nothing else does.
	build := func(is []ScriptInstant) *World {
		s := mustScript(t,
			[]ScriptCheck{
				constCheck(0, 1),
				constCheck(1, 1),
				{Op: notAnArm, Register: 2, Args: [scriptParams]int32{38, 64}},
			},
			is,
			[]ScriptTrigger{
				// Live, and fires: it runs the arms under test.
				{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
					Instants: acts(0, 1), Once: true, Latch: 0},
				// INERT: it reads the register the unimplemented check owns.
				{Pairs: [3]ScriptPair{pair(2, 0, ScriptCmpEQ)},
					Instants: acts(0), Once: true, Latch: 1},
				// Live, and does NOT hold, so its latch is written 0 and stays.
				{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpLT)},
					Instants: acts(1), Once: true, Latch: 2},
			})
		return ownerWorld(t, s)
	}

	withArms := build([]ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveUnit, Unit: 4, HasUnit: true, Player: 8, HasPlayer: true},
	})
	without := build([]ScriptInstant{{Op: notAnInstant}, {Op: notAnInstant}})

	// The INERT SET, before a tick is run.
	a, b := withArms.Script().InertTriggers(), without.Script().InertTriggers()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("the inert set is %v with the arms and %v without", a, b)
	}
	if len(a) != 1 || a[0] != 1 {
		t.Errorf("the inert set is %v, want exactly the trigger reading the unimplemented check", a)
	}

	scriptTicks(withArms, 32, nil)
	scriptTicks(without, 32, nil)

	// The LATCH VECTOR after one pass, position for position — which is also the
	// record of which triggers were evaluated: an inert trigger is skipped before
	// its latch is touched, so its latch is the only one that stays untouched.
	for i := int32(0); i < 3; i++ {
		if x, y := withArms.ScriptLatched(i), without.ScriptLatched(i); x != y {
			t.Errorf("latch %d is %v with the arms and %v without", i, x, y)
		}
	}
	if !withArms.ScriptLatched(0) {
		t.Error("the live trigger that holds did not fire in either build")
	}
	if withArms.ScriptLatched(1) || withArms.ScriptLatched(2) {
		t.Error("a trigger that is inert or does not hold latched")
	}
	// And the arms really did run in the first build, so the agreement above is
	// two builds doing the same thing to triggers rather than two doing nothing.
	if got, want := owners(withArms), (map[EntityID]uint32{1: 9, 2: 9, 3: 9, 4: 8, 5: 5}); !reflect.DeepEqual(got, want) {
		t.Errorf("with the arms the owners are %v, want %v", got, want)
	}
	if got, want := owners(without), (map[EntityID]uint32{1: 6, 2: 6, 3: 6, 4: 5, 5: 5}); !reflect.DeepEqual(got, want) {
		t.Errorf("without the arms the owners are %v, want the ones the world was built with %v", got, want)
	}
	// The OUTCOME is untouched in both: no arm here wins or loses a mission, and
	// an instant arm cannot decide one.
	if withArms.Outcome() != OutcomeUndecided || without.Outcome() != OutcomeUndecided {
		t.Errorf("outcomes %d and %d; these arms decide no mission",
			withArms.Outcome(), without.Outcome())
	}
}

// ---------------------------------------------------------------- AC-9

// TestTheEscortChoreographyRunsEndToEnd is AC-9: the hand-over shape a shipped
// escort mission authors, driven on a synthetic world and a hand-built script.
//
// A group of three is handed to one player; later, one member of that group is
// handed to a DIFFERENT player. That order is the whole point — it is what an
// escort does, and it is the case where the two arms overlap on one entity, so a
// build in which the group arm re-ran or the unit arm resolved through the group
// would end with the escortee back on the wrong roster.
//
// The second trigger is gated on a register the first trigger's own instant
// writes, so the two fire in different passes and nothing here depends on
// instant order within one trigger. Both conditions are arms this build already
// evaluates, so no arm outside the supported set is needed to advance it.
func TestTheEscortChoreographyRunsEndToEnd(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{
			constCheck(0, 1), // 0: the literal 1
			constCheck(1, 1), // 1: the literal 1, so 0 == 1 holds at once
			constCheck(2, 0), // 2: the gate, raised by the first trigger
			{Op: ScriptCheckVariable, Register: 3, Args: [scriptParams]int32{2}},
		},
		[]ScriptInstant{
			{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 1, HasPlayer: true},
			{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{2}},
			{Op: ScriptInstantGiveUnit, Unit: 2, HasUnit: true, Player: 3, HasPlayer: true},
		},
		[]ScriptTrigger{
			// Pass 1: hand the group of three to player 1, and raise the gate.
			{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
				Instants: acts(0, 1), Once: true, Latch: 0},
			// A later pass: the gate is up, so hand one of those three to player 3.
			{Pairs: [3]ScriptPair{pair(3, 0, ScriptCmpEQ)},
				Instants: acts(2), Once: true, Latch: 1},
		})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("this choreography needs %+v, and it is supposed to need nothing", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Fatalf("triggers %v are inert; this choreography needs both live", inert)
	}

	w := ownerWorld(t, s)

	// After the FIRST pass: the group is player 1's, entity 2 included, and the
	// second trigger has not fired. SIXTEEN ticks is exactly one pass — the pass
	// runs on phase 6 of the sixteen-tick cycle — and running two here would have
	// hidden the ordering this case exists to show.
	scriptTicks(w, 16, nil)
	want := map[EntityID]uint32{1: 1, 2: 1, 3: 1, 4: 5, 5: 5}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Fatalf("after the first pass the owners are %v, want %v", got, want)
	}
	if w.ScriptLatched(1) {
		t.Fatal("the second trigger fired in the first pass; the two are supposed to be ordered")
	}

	// After a LATER pass: the gate is up, and one member has moved on to player
	// 3 while the other two stay where the first trigger put them.
	scriptTicks(w, 64, nil)
	want = map[EntityID]uint32{1: 1, 2: 3, 3: 1, 4: 5, 5: 5}
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after the second pass the owners are %v, want %v", got, want)
	}
	if !w.ScriptLatched(0) || !w.ScriptLatched(1) {
		t.Errorf("latches %v/%v, want both triggers to have fired",
			w.ScriptLatched(0), w.ScriptLatched(1))
	}

	// And it STAYS there: both triggers are one-shot, so no later pass hands the
	// escortee back. A group arm that re-fired would put entity 2 on player 1
	// again, which is the failure this whole ordering exists to be able to see.
	scriptTicks(w, 128, nil)
	if got := owners(w); !reflect.DeepEqual(got, want) {
		t.Errorf("after a hundred and twenty-eight more ticks the owners are %v, want %v", got, want)
	}
}
