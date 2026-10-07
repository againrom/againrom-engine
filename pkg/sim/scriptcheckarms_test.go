package sim

// Check opcodes 17 and 9, the two arms 1029 implements: what each writes into
// its register, and a trigger reading that register firing when it should and
// not firing when it should not. A register no trigger reads is invisible, so
// every arm here is witnessed twice.
//
// Every fixture is a world and a hand-built script, on scriptitem_test.go's own
// precedent: both arms take an entity, one of them an item code and the other a
// pursuit, all of them plain state in this tree. Nothing here reads a map.
//
// THE CODES ARE THE MAP'S OWN SHAPE, on scriptitem_test.go's ground: 0x0e1e is
// what mission 30's own node compiles to, class 14 index 30, and 0x0e1a is
// another of the same class.

import "testing"

const (
	caCure  uint16 = 0x0e1e
	caOther uint16 = 0x0e1a

	caSubject EntityID = 1
	caTarget  EntityID = 2

	// The two authored map ids the fixtures give their placements. They are
	// not the entity ids, so a register carrying an entity id rather than an
	// authored map id fails every case below.
	caSubjectMapID uint16 = 41
	caTargetMapID  uint16 = 77

	// caBystanderMapID belongs to the entity at id 0. mapload mints entity
	// ids from zero, so a map-loaded world holds one, and a fixture without
	// it cannot tell an arm that reads AttackTarget unguarded from one that
	// guards it: the zero-value AttackTarget names entity 0, and where no
	// such entity exists the unguarded arm answers 0 by accident.
	caBystanderMapID uint16 = 63

	// caRegister is the check's own register and caConstant the register a
	// constant check presets, which is what a trigger compares it against.
	caRegister int32 = 0
	caConstant int32 = 1
)

// caTicks is one script pass: the pass runs on phase 6 of the sixteen-tick
// cycle (TestThePassRunsOncePerFullTickOnItsOwnPhase), so seven ticks hold
// exactly one.
const caTicks = 7

// caBounds is the fixture map, large enough that no unit here is ever crowded
// against an edge.
var caBounds = Bounds{Width: 80, Height: 80}

// itemTestNode is the compiled check-17 record under test, spelt once.
func itemTestNode(unit EntityID, code uint16) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckItemTest, Register: caRegister,
		Unit: unit, HasUnit: true, Item: code, HasItem: true}
}

// targetIDNode is the compiled check-9 record under test, spelt once.
func targetIDNode(unit EntityID) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckTargetID, Register: caRegister, Unit: unit, HasUnit: true}
}

// caStockedWorld is a two-entity world running s with the stock given. Both
// entities carry an authored map id, because a map-loaded placement does.
func caStockedWorld(t *testing.T, s *Script, stock []Stock) *World {
	t.Helper()
	w, err := NewStockedWorld(1, caBounds, ModeCanonical, Terrain{},
		[]Entity{
			{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID},
			{ID: caTarget, X: 2, Y: 2, HP: 5, MaxHP: 5, MapUnitID: caTargetMapID},
		}, s, Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// ------------------------------------------------- check 17, the register

// TestTheItemTestArmAnswersOneOnlyForTheContainerThatHoldsTheCode is B2's
// register half. One node, five fixtures, and the answer read straight out of
// the register the node names.
//
// The other-container case is what makes "holds" mean holds HERE: the same code
// in the other entity's container answers 0, so an arm scanning the world
// rather than the named container fails.
func TestTheItemTestArmAnswersOneOnlyForTheContainerThatHoldsTheCode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		node  ScriptCheck
		stock []Stock
		want  int32
	}{
		{"the named container holds the code", itemTestNode(caSubject, caCure),
			[]Stock{{ID: caSubject, Items: []uint16{caCure}}}, 1},
		{"the named container holds a different code", itemTestNode(caSubject, caCure),
			[]Stock{{ID: caSubject, Items: []uint16{caOther}}}, 0},
		{"the named container is empty", itemTestNode(caSubject, caCure), nil, 0},
		{"the OTHER container holds the code", itemTestNode(caSubject, caCure),
			[]Stock{{ID: caTarget, Items: []uint16{caCure}}}, 0},
		{"the code is one of several the container holds", itemTestNode(caSubject, caCure),
			[]Stock{{ID: caSubject, Items: []uint16{caOther, caCure}}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := caStockedWorld(t, mustScript(t, []ScriptCheck{tc.node}, nil, nil), tc.stock)
			scriptTicks(w, caTicks, nil)
			if got := w.ScriptRegister(caRegister); got != tc.want {
				t.Errorf("register %d is %d, want %d", caRegister, got, tc.want)
			}
		})
	}
}

// TestTheItemTestArmReadsTheContainerAndNotTheWornPlaces is the arm's own
// scope. TRIG-ITEMTEST-040 puts the finder's receiver at unit+0x7c, the
// container, and the twelve worn places are a different array. A unit wearing
// the code and carrying nothing answers 0.
func TestTheItemTestArmReadsTheContainerAndNotTheWornPlaces(t *testing.T) {
	t.Parallel()

	var worn [EquipSlots]uint16
	worn[0] = caCure
	w := caStockedWorld(t, mustScript(t, []ScriptCheck{itemTestNode(caSubject, caCure)}, nil, nil),
		[]Stock{{ID: caSubject, Equipped: worn}})
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("a unit wearing the code and carrying none answers %d, want 0 — the arm reads "+
			"the container alone", got)
	}
}

// TestTheItemTestArmAnswersZeroWhenTheNodeNamesNoItem is the shipped corpus's
// own boundary: every check-17 node in it binds an item reference, so a node
// without one is a shape no map authors. The arm answers 0 because the presence
// flag is unset, tested before any container is read.
func TestTheItemTestArmAnswersZeroWhenTheNodeNamesNoItem(t *testing.T) {
	t.Parallel()

	node := ScriptCheck{Op: ScriptCheckItemTest, Register: caRegister, Unit: caSubject, HasUnit: true}
	w := caStockedWorld(t, mustScript(t, []ScriptCheck{node}, nil, nil),
		[]Stock{{ID: caSubject, Items: []uint16{caCure}}})
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("a node naming no item answers %d, want 0", got)
	}

	// THE PRESENCE FLAG IS WHAT DECIDES, NOT THE CODE. This node's item word
	// carries a code the subject is holding while its flag says no item was
	// authored, which is the state an arm reading the code alone cannot tell
	// from a real reference. Answering 1 here would mean the compiled record's
	// unset field had been read as data.
	stale := ScriptCheck{Op: ScriptCheckItemTest, Register: caRegister, Unit: caSubject,
		HasUnit: true, Item: caCure}
	w = caStockedWorld(t, mustScript(t, []ScriptCheck{stale}, nil, nil),
		[]Stock{{ID: caSubject, Items: []uint16{caCure}}})
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("a node whose item flag is unset answers %d for a held code, want 0", got)
	}
}

// TestTheItemTestArmWritesNothingButItsRegister is TRIG-ITEMTEST-040's own
// word: the arm reads and does not write. The container it read is compared
// element by element after the pass, because a flattened read cannot tell one
// element at count 2 from two at count 1.
func TestTheItemTestArmWritesNothingButItsRegister(t *testing.T) {
	t.Parallel()

	w := caStockedWorld(t, mustScript(t, []ScriptCheck{itemTestNode(caSubject, caCure)}, nil, nil),
		[]Stock{{ID: caSubject, Items: []uint16{caCure, caOther}}})
	before := append([]Entity(nil), w.Entities()...)
	scriptTicks(w, caTicks, nil)

	got, ok := w.CarriedStacks(caSubject)
	if !ok {
		t.Fatalf("CarriedStacks(%d) answered not-ok for an entity this world holds", caSubject)
	}
	if len(got) != 2 || got[0].Code != caCure || got[0].Count != 1 ||
		got[1].Code != caOther || got[1].Count != 1 {
		t.Errorf("the container the arm read is %+v, want the two elements it was stocked with", got)
	}
	after := w.Entities()
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("entity %d is %+v after the pass, want %+v — the arm writes no entity field",
				before[i].ID, after[i], before[i])
		}
	}
}

// ------------------------------------------------- check 17, the trigger

// TestATriggerReadingTheItemTestFiresOnlyWhileTheCodeIsHeld is B2's second
// half. The register is what the arm writes; this is what the register is for.
// The same script runs over two stocks and the win counter is the answer.
func TestATriggerReadingTheItemTestFiresOnlyWhileTheCodeIsHeld(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		stock []Stock
		want  uint32
	}{
		{"holding the code", []Stock{{ID: caSubject, Items: []uint16{caCure}}}, 1},
		{"holding another code", []Stock{{ID: caSubject, Items: []uint16{caOther}}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t,
				[]ScriptCheck{itemTestNode(caSubject, caCure), constCheck(caConstant, 1)},
				[]ScriptInstant{{Op: ScriptInstantWin}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)},
					Instants: acts(0), Once: true}})
			if got := s.InertTriggers(); len(got) != 0 {
				t.Fatalf("InertTriggers() is %v, want none — check 17 is implemented, so no "+
					"trigger reading it may be poisoned", got)
			}
			w := caStockedWorld(t, s, tc.stock)
			scriptTicks(w, caTicks, nil)
			won, lost := w.ScriptCounters()
			if won != tc.want || lost != 0 {
				t.Errorf("the counters are won %d lost %d, want won %d lost 0", won, lost, tc.want)
			}
		})
	}
}

// ------------------------------------------------- check 9, the register

// TestTheTargetIDArmAnswersThePursuedUnitsAuthoredMapID is B3's register half,
// and the case that says whose id it is: the subject and the target carry
// different authored ids, and the answer is the target's.
func TestTheTargetIDArmAnswersThePursuedUnitsAuthoredMapID(t *testing.T) {
	t.Parallel()

	w := caStockedWorld(t, mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil), nil)
	scriptTicks(w, caTicks, []Command{{Kind: KindAttack, Entity: caSubject, X: int32(caTarget)}})
	if got, want := w.ScriptRegister(caRegister), int32(caTargetMapID); got != want {
		t.Errorf("register %d is %d, want the pursued unit's authored map id %d (the subject's own "+
			"is %d, and the two entity ids are %d and %d)",
			caRegister, got, want, caSubjectMapID, caSubject, caTarget)
	}
}

// TestTheTargetIDArmAnswersZeroWhenTheSubjectIsNotPursuing is
// TRIG-TARGETID-032's zero result: no order at all, and the register takes 0.
//
// THE WORLD HOLDS AN ENTITY AT ID 0 and that entity carries an authored map
// id. A subject holding no pursuit has AttackTarget at its zero value, which
// names entity 0; an arm that read the field without testing HasAttackTarget
// would answer that bystander's id here and 0 in a world with no entity 0.
func TestTheTargetIDArmAnswersZeroWhenTheSubjectIsNotPursuing(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil)
	w, err := NewScriptedWorld(1, caBounds, ModeCanonical, nil,
		[]Entity{
			{ID: 0, X: 3, Y: 3, HP: 5, MaxHP: 5, MapUnitID: caBystanderMapID},
			{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID},
			{ID: caTarget, X: 2, Y: 2, HP: 5, MaxHP: 5, MapUnitID: caTargetMapID},
		}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("register %d is %d, want 0 — this unit holds no order, and %d is the "+
			"authored id of the entity at id 0", caRegister, got, caBystanderMapID)
	}
}

// TestTheTargetIDArmAnswersZeroForAMovementOrder is the same zero result from
// the other side: the subject holds an order, and it is not a pursuit.
func TestTheTargetIDArmAnswersZeroForAMovementOrder(t *testing.T) {
	t.Parallel()

	w := caStockedWorld(t, mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil), nil)
	scriptTicks(w, caTicks, []Command{{Kind: KindMoveTo, Entity: caSubject, X: 6, Y: 6}})
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("register %d is %d, want 0 — a walk is not a pursuit", caRegister, got)
	}
}

// TestTheTargetIDArmAnswersZeroForATargetWithNoAuthoredMapID is DIV-241's own
// case. A party member is placed by the loader rather than by the map file and
// carries no authored id, so the id this arm can answer with is 0. The original
// would answer whatever runtime id it had assigned.
func TestTheTargetIDArmAnswersZeroForATargetWithNoAuthoredMapID(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil)
	w, err := NewScriptedWorld(1, caBounds, ModeCanonical, nil,
		[]Entity{
			{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID},
			{ID: caTarget, X: 2, Y: 2, HP: 5, MaxHP: 5},
		}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	scriptTicks(w, caTicks, []Command{{Kind: KindAttack, Entity: caSubject, X: int32(caTarget)}})
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("register %d is %d, want 0 — the pursued unit carries no authored map id",
			caRegister, got)
	}
}

// TestTheTargetIDArmAnswersZeroForATargetTheWorldNoLongerHolds is DIV-240's own
// case, and it witnesses an INVARIANT rather than a branch. The original's arm
// has no null branch for a pursuit whose target is gone and faults there. This
// build cannot hold that state at all: the constructor normalises a pursuit
// naming an entity the world does not hold (world.go's second pass), and the
// removal sweep does the same after a unit leaves (step.go's keep pass). So the
// register is 0 because the pursuit was cleared, not because the arm's own
// index guard fired.
//
// Both halves are asserted here, because the second alone would pass against a
// build that kept the dangling pursuit and answered 0 for a different reason.
func TestTheTargetIDArmAnswersZeroForATargetTheWorldNoLongerHolds(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{targetIDNode(caSubject)}, nil, nil)
	w, err := NewScriptedWorld(1, caBounds, ModeCanonical, nil,
		[]Entity{{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID,
			AttackTarget: caTarget, HasAttackTarget: true}}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for _, e := range w.Entities() {
		if e.ID == caSubject && e.HasAttackTarget {
			t.Fatalf("the world kept a pursuit naming entity %d, which it does not hold", caTarget)
		}
	}
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != 0 {
		t.Errorf("register %d is %d, want 0 — the pursued entity is not in this world",
			caRegister, got)
	}
}

// ------------------------------------------------- check 9, the trigger

// TestATriggerReadingTheTargetIDFiresOnlyForTheAuthoredIDItNames is B3's second
// half, and the shape a shipped map authors: a check-9 node compared against a
// constant naming one unit's authored id. It fires while the subject pursues
// that unit and not while it pursues another.
func TestATriggerReadingTheTargetIDFiresOnlyForTheAuthoredIDItNames(t *testing.T) {
	t.Parallel()

	const caThird EntityID = 3
	const caThirdMapID uint16 = 99

	for _, tc := range []struct {
		name   string
		pursue EntityID
		want   uint32
	}{
		{"pursuing the unit the constant names", caTarget, 1},
		{"pursuing another unit", caThird, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t,
				[]ScriptCheck{targetIDNode(caSubject), constCheck(caConstant, int32(caTargetMapID))},
				[]ScriptInstant{{Op: ScriptInstantWin}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)},
					Instants: acts(0), Once: true}})
			if got := s.InertTriggers(); len(got) != 0 {
				t.Fatalf("InertTriggers() is %v, want none — check 9 is implemented, so no "+
					"trigger reading it may be poisoned", got)
			}
			w, err := NewScriptedWorld(1, caBounds, ModeCanonical, nil,
				[]Entity{
					{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID},
					{ID: caTarget, X: 2, Y: 2, HP: 5, MaxHP: 5, MapUnitID: caTargetMapID},
					{ID: caThird, X: 3, Y: 3, HP: 5, MaxHP: 5, MapUnitID: caThirdMapID},
				}, s)
			if err != nil {
				t.Fatalf("NewScriptedWorld: %v", err)
			}
			scriptTicks(w, caTicks, []Command{{Kind: KindAttack, Entity: caSubject, X: int32(tc.pursue)}})
			won, lost := w.ScriptCounters()
			if won != tc.want || lost != 0 {
				t.Errorf("the counters are won %d lost %d, want won %d lost 0", won, lost, tc.want)
			}
		})
	}
}
