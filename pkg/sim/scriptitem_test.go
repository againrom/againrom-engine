package sim

// Instant opcodes 12 and 13, the two ITEM arms (0156): what each writes, what
// each refuses, and what neither touches. Every fixture here is a world and a
// hand-built script, on giveall_test.go's own precedent — both arms take an
// entity id, an item code and a container, all three plain state in this tree,
// so nothing here needs a map to exercise it.
//
// THE CODES ARE THE MAP'S OWN SHAPE. 0x0e1e is what mission 30's own node
// compiles to: class 14, index 30. 0x0e1a and 0x0e25 are two more of the same
// class, so a case holding two codes holds two codes a shipped map could
// actually have authored.

import "testing"

// The three codes every fixture here uses, and the two entity ids.
const (
	itCure  uint16 = 0x0e1e
	itOther uint16 = 0x0e1a
	itThird uint16 = 0x0e25

	itHero   EntityID = 1
	itSecond EntityID = 2
)

// itWorld is a two-entity world running s, with the stock given.
func itWorld(t *testing.T, s *Script, stock []Stock) *World {
	t.Helper()
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: itHero, X: 1, Y: 1, HP: 5, MaxHP: 5}, {ID: itSecond, X: 2, Y: 2, HP: 5, MaxHP: 5}},
		s, Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// itStacks reads one entity's container back as elements, so a case can say
// what the PLACES are and not merely which units survived. Carried() flattens,
// which cannot tell one element at count 2 from two elements at count 1 — and
// that is exactly AC-2's question.
func itStacks(t *testing.T, w *World, id EntityID) []ItemStack {
	t.Helper()
	got, ok := w.CarriedStacks(id)
	if !ok {
		t.Fatalf("CarriedStacks(%d) answered not-ok for an entity this world holds", id)
	}
	return got
}

func itEqual(got []ItemStack, want ...ItemStack) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !StackStateEqual(got[i], want[i]) {
			return false
		}
	}
	return true
}

// addNode and takeNode are the two compiled records under test, spelt once.
func addNode(code uint16) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantAddItem, Unit: itHero, HasUnit: true, Item: code, HasItem: true}
}

func takeNode(code uint16) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantTakeItem, Unit: itHero, HasUnit: true, Item: code, HasItem: true}
}

// ---------------------------------------------------------------- AC-1

// TestTheAddArmCreatesOneUnitInTheNamedUnitsContainer is AC-1: a hero holding
// nothing, one instant-12 node, one element at count 1 afterwards. The other
// entity is named by nothing and is checked too, because "creates" has to mean
// creates HERE and not simply "some container gained a code".
func TestTheAddArmCreatesOneUnitInTheNamedUnitsContainer(t *testing.T) {
	t.Parallel()

	w := itWorld(t, fireOnce(t, []ScriptInstant{addNode(itCure)}, 0), nil)
	runPass(t, w)

	if got := itStacks(t, w, itHero); !itEqual(got, ItemStack{Code: itCure, Count: 1}) {
		t.Errorf("the hero's container is %+v, want one element {0x0e1e 1}", got)
	}
	if got := itStacks(t, w, itSecond); len(got) != 0 {
		t.Errorf("the unnamed entity's container is %+v, want nothing — this arm writes one container", got)
	}
}

// ---------------------------------------------------------------- AC-2

func TestTwoAddsOfOneCodeLeaveOneElementAtCountTwo(t *testing.T) {
	t.Parallel()

	w := itWorld(t, fireOnce(t, []ScriptInstant{addNode(itCure), addNode(itCure)}, 0, 1), nil)
	runPass(t, w)

	if got := itStacks(t, w, itHero); !itEqual(got, ItemStack{Code: itCure, Count: 2}) {
		t.Errorf("the hero's container is %+v, want ONE element {0x0e1e 2} and not two places", got)
	}
}

// TestAnAddOntoAHeldCodeKeepsThatElementsPlace is AC-2's second clause: a hero
// already holding two codes gains a unit of the FIRST one, and the element
// order is what it was. The merge joins the element at that element's own
// place; it does not move the code to the tail.
func TestAnAddOntoAHeldCodeKeepsThatElementsPlace(t *testing.T) {
	t.Parallel()

	w := itWorld(t, fireOnce(t, []ScriptInstant{addNode(itOther)}, 0),
		[]Stock{{ID: itHero, Items: []uint16{itOther, itThird}}})
	runPass(t, w)

	got := itStacks(t, w, itHero)
	if !itEqual(got, ItemStack{Code: itOther, Count: 2}, ItemStack{Code: itThird, Count: 1}) {
		t.Errorf("the hero's container is %+v, want [{0x0e1a 2} {0x0e25 1}] — the first element's "+
			"count raised at its own place", got)
	}
}

// ---------------------------------------------------------------- AC-3

// TestTheTakeArmRemovesExactlyOneUnit is AC-3, all three of its clauses in one
// table: a count of 2 falls to 1 at the same place, a count of 1 removes the
// element whole, and a code the container does not hold changes nothing.
//
// The middle case is the one the arm could most easily get wrong in the other
// direction — removing the whole stack — and the first is the one it could get
// wrong by removing the element at any count. Both are asserted as the exact
// element list.
func TestTheTakeArmRemovesExactlyOneUnit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		held  []uint16
		taken uint16
		want  []ItemStack
	}{
		{"a stack of two loses one and keeps its place",
			[]uint16{itCure, itCure}, itCure, []ItemStack{{Code: itCure, Count: 1}}},
		{"a stack of one is removed whole",
			[]uint16{itCure}, itCure, nil},
		{"a stack of five loses exactly one",
			[]uint16{itCure, itCure, itCure, itCure, itCure}, itCure, []ItemStack{{Code: itCure, Count: 4}}},
		{"a code the container does not hold changes nothing",
			[]uint16{itOther}, itCure, []ItemStack{{Code: itOther, Count: 1}}},
		{"an empty container changes nothing",
			nil, itCure, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var stock []Stock
			if len(c.held) > 0 {
				stock = []Stock{{ID: itHero, Items: c.held}}
			}
			w := itWorld(t, fireOnce(t, []ScriptInstant{takeNode(c.taken)}, 0), stock)
			runPass(t, w)

			if got := itStacks(t, w, itHero); !itEqual(got, c.want...) {
				t.Errorf("the hero's container is %+v, want %+v", got, c.want)
			}
		})
	}
}

// ---------------------------------------------------------------- AC-4

// TestTheTakeArmLeavesEveryOtherElementWhere ItWas is AC-4: a container holding
// three codes loses one unit of the middle one, and the other two keep their
// codes, their counts and their places.
func TestTheTakeArmLeavesEveryOtherElementWhereItWas(t *testing.T) {
	t.Parallel()

	w := itWorld(t, fireOnce(t, []ScriptInstant{takeNode(itOther)}, 0),
		[]Stock{{ID: itHero, Items: []uint16{itCure, itOther, itOther, itThird}}})
	runPass(t, w)

	got := itStacks(t, w, itHero)
	want := []ItemStack{{Code: itCure, Count: 1}, {Code: itOther, Count: 1}, {Code: itThird, Count: 1}}
	if !itEqual(got, want...) {
		t.Errorf("the hero's container is %+v, want %+v — only the named code's count moved", got, want)
	}
}

// TestTheTakeArmWritesOnlyTheNamedUnitsContainer is AC-4's second clause: the
// other entity holds the same code and keeps every unit of it.
func TestTheTakeArmWritesOnlyTheNamedUnitsContainer(t *testing.T) {
	t.Parallel()

	w := itWorld(t, fireOnce(t, []ScriptInstant{takeNode(itCure)}, 0),
		[]Stock{{ID: itHero, Items: []uint16{itCure}}, {ID: itSecond, Items: []uint16{itCure, itCure}}})
	runPass(t, w)

	if got := itStacks(t, w, itHero); len(got) != 0 {
		t.Errorf("the hero's container is %+v, want nothing", got)
	}
	if got := itStacks(t, w, itSecond); !itEqual(got, ItemStack{Code: itCure, Count: 2}) {
		t.Errorf("the other entity's container is %+v, want {0x0e1e 2} untouched", got)
	}
}

// ---------------------------------------------------------------- AC-5

// TestNeitherItemArmTouchesAnythingButTheContainer is AC-5. Both arms run
// against a world whose entities wear armour, hold gold, stand somewhere, own a
// slot and belong to a group; afterwards every one of those is what it was.
//
// The equipment record is the sharpest of them: it is the other half of "what
// this actor has", it holds one of the very codes the take arm names, and an
// arm that searched it as well as the container would empty a worn slot here
// and nowhere else in this file.
func TestNeitherItemArmTouchesAnythingButTheContainer(t *testing.T) {
	t.Parallel()

	var gear [EquipSlots]uint16
	gear[0], gear[3] = itCure, itThird

	s := fireOnce(t, []ScriptInstant{addNode(itOther), takeNode(itCure)}, 0, 1)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{
			{ID: itHero, X: 3, Y: 4, HP: 4, MaxHP: 9, Owner: 1, Group: 6},
			{ID: itSecond, X: 7, Y: 8, HP: 5, MaxHP: 5, Owner: 2, Group: 6},
		}, s, Relations{}, nil, []Stock{{ID: itHero, Items: []uint16{itCure}, Equipped: gear}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if !w.SetPurse(1, 250) {
		t.Fatal("SetPurse(1, 250) was refused")
	}
	before := w.Entities()
	tick := w.Tick()

	runPass(t, w)

	if got, ok := w.Equipped(itHero); !ok || got != gear {
		t.Errorf("Equipped(hero) = %v (ok=%v), want %v — the worn places are not the container",
			got, ok, gear)
	}
	if got := w.Purse(1); got != 250 {
		t.Errorf("Purse(1) = %d, want 250", got)
	}
	if w.Tick() == tick {
		t.Fatal("the world did not advance; this case measures a pass that ran")
	}
	after := w.Entities()
	if len(after) != len(before) {
		t.Fatalf("the world holds %d entities, want %d", len(after), len(before))
	}
	for i := range after {
		if after[i] != before[i] {
			t.Errorf("entity %d is %+v, want %+v — neither arm writes an entity record",
				after[i].ID, after[i], before[i])
		}
	}
}

// ---------------------------------------------------------------- AC-6

// itSubject is everything the two item arms are able to write, read out of a
// world in one value: both containers, both equipment records, every purse and
// every entity record.
//
// It is NOT the world's digest, and that is deliberate. The digest covers the
// compiled program as well as the state, so two worlds running two different
// nodes differ whatever their arms did — which makes a digest useless as a
// control for "this node changed nothing". What a refusal has to leave alone is
// the state, and this is the whole of the state either arm can reach.
type itSubject struct {
	held     [2][]ItemStack
	worn     [2][EquipSlots]uint16
	purses   [relationSlots]uint32
	entities []Entity
}

func itRead(t *testing.T, w *World) itSubject {
	t.Helper()
	var s itSubject
	for i, id := range []EntityID{itHero, itSecond} {
		s.held[i] = itStacks(t, w, id)
		worn, ok := w.Equipped(id)
		if !ok {
			t.Fatalf("Equipped(%d) answered not-ok for an entity this world holds", id)
		}
		s.worn[i] = worn
	}
	for slot := range s.purses {
		s.purses[slot] = w.Purse(uint32(slot))
	}
	s.entities = w.Entities()
	return s
}

func (a itSubject) equal(b itSubject) bool {
	for i := range a.held {
		if !itEqual(a.held[i], b.held[i]...) || a.worn[i] != b.worn[i] {
			return false
		}
	}
	if a.purses != b.purses || len(a.entities) != len(b.entities) {
		return false
	}
	for i := range a.entities {
		if a.entities[i] != b.entities[i] {
			return false
		}
	}
	return true
}

// TestTheFourRefusalsLeaveTheWorldExactlyAsFound is AC-6: a node binding no
// unit, a node binding no item, a node naming an entity this world does not
// hold, and a node whose code is zero. Each is run on both arms, and the
// witness is itSubject — everything either arm is able to write — read before
// the pass and again after it.
func TestTheFourRefusalsLeaveTheWorldExactlyAsFound(t *testing.T) {
	t.Parallel()

	refusals := []struct {
		name string
		node func(op int32) ScriptInstant
	}{
		// Both of the first two carry a VALUE with the flag clear, and that
		// is what makes them witness the flags. A record left wholly zero
		// would be refused by the id lookup and by the zero code anyway, so
		// removing either flag test would cost nothing and the case would
		// pass over an arm that no longer reads the flag at all.
		{"no unit reference", func(op int32) ScriptInstant {
			return ScriptInstant{Op: op, Unit: itHero, Item: itCure, HasItem: true}
		}},
		{"no item reference", func(op int32) ScriptInstant {
			return ScriptInstant{Op: op, Unit: itHero, HasUnit: true, Item: itCure}
		}},
		{"a unit this world does not hold", func(op int32) ScriptInstant {
			return ScriptInstant{Op: op, Unit: 99, HasUnit: true, Item: itCure, HasItem: true}
		}},
		{"a zero code", func(op int32) ScriptInstant {
			return ScriptInstant{Op: op, Unit: itHero, HasUnit: true, Item: 0, HasItem: true}
		}},
	}
	arms := []struct {
		name string
		op   int32
	}{{"add", ScriptInstantAddItem}, {"take", ScriptInstantTakeItem}}

	for _, a := range arms {
		for _, r := range refusals {
			t.Run(a.name+", "+r.name, func(t *testing.T) {
				t.Parallel()

				var gear [EquipSlots]uint16
				gear[2] = itCure
				w := itWorld(t, fireOnce(t, []ScriptInstant{r.node(a.op)}, 0),
					[]Stock{{ID: itHero, Items: []uint16{itCure, itOther}, Equipped: gear},
						{ID: itSecond, Items: []uint16{itCure}}})
				if !w.SetPurse(1, 77) {
					t.Fatal("SetPurse(1, 77) was refused")
				}
				before := itRead(t, w)

				runPass(t, w)

				if after := itRead(t, w); !after.equal(before) {
					t.Errorf("the refused node left\n %+v\nwant it exactly as found\n %+v", after, before)
				}
			})
		}
	}
}

// ---------------------------------------------------------------- AC-7

// TestAScriptCarryingItemNodesRoundTripsThroughTheByteForm is AC-7: a world
// whose compiled program holds both arms encodes and decodes to a world holding
// the same program, codes and presence flags included.
//
// It ALSO checks the record that binds NO item, because a presence flag that
// was never written and a presence flag written as false are the same byte and
// only a record that has one of each can tell the encoder read the field at all.
func TestAScriptCarryingItemNodesRoundTripsThroughTheByteForm(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		addNode(itCure),
		takeNode(itThird),
		{Op: ScriptInstantGiveMoney, Player: 1, HasPlayer: true},
	}, 0, 1, 2)
	w := itWorld(t, s, []Stock{{ID: itHero, Items: []uint16{itThird}}})

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a world whose script names items does not round-trip through its byte form")
	}

	got := back.script.instants
	want := w.script.instants
	if len(got) != len(want) {
		t.Fatalf("the decoded program holds %d instant(s), want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("instant %d decoded to %+v, want %+v", i, got[i], want[i])
		}
	}
	if want[2].HasItem || want[2].Item != 0 {
		t.Fatal("the third node was supposed to bind no item; this case cannot tell a written " +
			"flag from an unwritten one without it")
	}

	// The two previous versions are refused rather than migrated. 45 was never
	// written by this tree at all — it is out to a story running in parallel —
	// and 44 is the sharp case: every byte before the second instant record is
	// identical, so it would decode the first record intact and read every one
	// after it three bytes into its neighbour.
	for _, v := range []byte{44, 45} {
		spoiled := append([]byte(nil), form...)
		spoiled[0] = v
		if err := (&World{}).UnmarshalBinary(spoiled); err == nil {
			t.Errorf("a buffer declaring version %d was accepted", v)
		}
	}
}

func TestTheContainerInvariantSurvivesBothArms(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		addNode(itCure), addNode(itCure), takeNode(itCure), takeNode(itCure),
	}, 0, 1, 2, 3)
	w := itWorld(t, s, []Stock{{ID: itHero, Items: []uint16{itOther}}})
	runPass(t, w)

	got := itStacks(t, w, itHero)
	if !itEqual(got, ItemStack{Code: itOther, Count: 1}) {
		t.Fatalf("the hero's container is %+v, want [{0x0e1a 1}] — two adds and two takes of "+
			"one code cancel and leave the untouched element alone", got)
	}
	seen := map[uint16]bool{}
	for _, st := range got {
		if st.Count == 0 {
			t.Errorf("element %#04x holds nothing; no container may carry a place with a count of zero", st.Code)
		}
		if seen[st.Code] {
			t.Errorf("code %#04x occupies two elements; a folded container holds at most one place per code", st.Code)
		}
		seen[st.Code] = true
	}
}

func TestWhichContainerAnItemArmWritesIsTheIdAndNotThePlace(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, ents []Entity) *World {
		t.Helper()
		s := fireOnce(t, []ScriptInstant{addNode(itCure)}, 0)
		w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{}, ents, s, Relations{}, nil,
			[]Stock{{ID: itSecond, Items: []uint16{itOther}}})
		if err != nil {
			t.Fatalf("NewStockedWorld: %v", err)
		}
		runPass(t, w)
		return w
	}
	forward := build(t, []Entity{{ID: itHero, X: 1, Y: 1, HP: 5, MaxHP: 5}, {ID: itSecond, X: 2, Y: 2, HP: 5, MaxHP: 5}})
	reverse := build(t, []Entity{{ID: itSecond, X: 2, Y: 2, HP: 5, MaxHP: 5}, {ID: itHero, X: 1, Y: 1, HP: 5, MaxHP: 5}})

	for _, w := range []*World{forward, reverse} {
		if got := itStacks(t, w, itHero); !itEqual(got, ItemStack{Code: itCure, Count: 1}) {
			t.Errorf("the hero's container is %+v, want {0x0e1e 1} whichever order the world was declared in", got)
		}
		if got := itStacks(t, w, itSecond); !itEqual(got, ItemStack{Code: itOther, Count: 1}) {
			t.Errorf("the other entity's container is %+v, want {0x0e1a 1} untouched", got)
		}
	}
}

func TestNeitherItemArmArmsATrigger(t *testing.T) {
	t.Parallel()

	for _, op := range []int32{ScriptInstantAddItem, ScriptInstantTakeItem} {
		if !scriptInstantSupported(ScriptInstant{Op: op}) {
			t.Errorf("instant opcode %d reports unsupported", op)
		}
	}
	if !scriptCheckSupported(ScriptCheckItemTest) {
		t.Error("check opcode 17 reports unsupported; 1029 implements it")
	}
	if scriptCheckSupported(scriptCheckSentinelOp) {
		t.Errorf("check opcode %d reports supported; the check vocabulary is supposed to be "+
			"unmoved by the two instant arms", scriptCheckSentinelOp)
	}
}
