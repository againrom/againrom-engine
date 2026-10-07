package sim

// Opcode 28, Give All (0129 T1): the second unit reference on ScriptInstant,
// the arm's own pour, and the four refusals that leave the world exactly as
// found. Every fixture here is a world and a hand-built script, on
// scriptowner_test.go's own precedent: the opcode takes two entity ids and a
// container, both of which are plain state in this tree, so nothing here
// needs a map to exercise it.

import "testing"

// ---------------------------------------------------------------- AC-1

// TestGiveAllMovesTheWholeContainerInOrder is AC-1: the receiver ends up
// holding its own code first, then the giver's three in the giver's own
// order, and the giver ends up holding nothing. Asserted as the exact two
// lists, on this file's own instruction and every sibling in this package's.
func TestGiveAllMovesTheWholeContainerInOrder(t *testing.T) {
	t.Parallel()

	const giver, receiver EntityID = 1, 2
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true, Unit2: receiver, HasUnit2: true},
	}, 0)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: giver, X: 1, Y: 1}, {ID: receiver, X: 2, Y: 2}}, s, Relations{}, nil,
		[]Stock{
			{ID: giver, Items: []uint16{0x0e06, 0x0e06, 0x0e06}},
			{ID: receiver, Items: []uint16{7}},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	runPass(t, w)

	gotReceiver, ok := w.Carried(receiver)
	if !ok {
		t.Fatal("Carried(receiver) answered not-ok for an entity this world holds")
	}
	if !equalCodes(gotReceiver, []uint16{7, 0x0e06, 0x0e06, 0x0e06}) {
		t.Errorf("Carried(receiver) = %v, want [7 0xe06 0xe06 0xe06] — its own code first, "+
			"then the giver's three in the giver's own order", gotReceiver)
	}

	gotGiver, ok := w.Carried(giver)
	if !ok {
		t.Fatal("Carried(giver) answered not-ok for an entity this world holds")
	}
	if len(gotGiver) != 0 {
		t.Errorf("Carried(giver) = %v, want nothing — the whole container moved", gotGiver)
	}
}

// ---------------------------------------------------------------- AC-2

// TestGiveAllLeavesEquipmentAlone is AC-2: with both entities wearing
// something, the pour leaves both equipment records byte-for-byte what they
// were, and the giver — now holding nothing — still wears what she wore.
func TestGiveAllLeavesEquipmentAlone(t *testing.T) {
	t.Parallel()

	const giver, receiver EntityID = 1, 2
	var giverGear, receiverGear [EquipSlots]uint16
	giverGear[0] = 0x101
	receiverGear[5] = 0x202

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true, Unit2: receiver, HasUnit2: true},
	}, 0)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: giver, X: 1, Y: 1}, {ID: receiver, X: 2, Y: 2}}, s, Relations{}, nil,
		[]Stock{
			{ID: giver, Items: []uint16{9}, Equipped: giverGear},
			{ID: receiver, Equipped: receiverGear},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	runPass(t, w)

	if got, ok := w.Equipped(giver); !ok || got != giverGear {
		t.Errorf("Equipped(giver) = %v, ok=%v, want %v unchanged — the giver keeps her armour",
			got, ok, giverGear)
	}
	if got, ok := w.Equipped(receiver); !ok || got != receiverGear {
		t.Errorf("Equipped(receiver) = %v, ok=%v, want %v unchanged", got, ok, receiverGear)
	}
	if got, _ := w.Carried(giver); len(got) != 0 {
		t.Errorf("Carried(giver) = %v after the pour, want nothing", got)
	}
}

// ---------------------------------------------------------------- AC-4

// TestGiveAllPreservesTheReceiversOwnOrderAcrossASecondPour is AC-4: two
// opcode-28 nodes firing in one trigger, A into C and then B into C, leave C
// holding its own codes, then A's, then B's.
func TestGiveAllPreservesTheReceiversOwnOrderAcrossASecondPour(t *testing.T) {
	t.Parallel()

	const a, b, c EntityID = 1, 2, 3
	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: a, HasUnit: true, Unit2: c, HasUnit2: true},
		{Op: ScriptInstantGiveAll, Unit: b, HasUnit: true, Unit2: c, HasUnit2: true},
	}, 0, 1)
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: a, X: 1, Y: 1}, {ID: b, X: 2, Y: 2}, {ID: c, X: 3, Y: 3}}, s, Relations{}, nil,
		[]Stock{
			{ID: a, Items: []uint16{1, 2}},
			{ID: b, Items: []uint16{3, 4}},
			{ID: c, Items: []uint16{9}},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	runPass(t, w)

	got, ok := w.Carried(c)
	if !ok {
		t.Fatal("Carried(c) answered not-ok for an entity this world holds")
	}
	if !equalCodes(got, []uint16{9, 1, 2, 3, 4}) {
		t.Errorf("Carried(c) = %v, want [9 1 2 3 4] — its own code, then A's pour, then B's", got)
	}

	// The two givers really were emptied, so the receiver's list above is
	// evidence of two pours and not of a fixture that started that way.
	for _, id := range []EntityID{a, b} {
		if got, _ := w.Carried(id); len(got) != 0 {
			t.Errorf("Carried(%d) = %v, want nothing after it gave everything away", id, got)
		}
	}
}

// ---------------------------------------------------------------- AC-3

func TestGiveAllsFourRefusalsChangeNothing(t *testing.T) {
	t.Parallel()

	const decoy, giver, receiver EntityID = 0, 10, 20
	ents := []Entity{{ID: decoy, X: 5, Y: 5}, {ID: giver, X: 1, Y: 1}, {ID: receiver, X: 2, Y: 2}}
	stock := []Stock{
		{ID: decoy, Items: []uint16{99}},
		{ID: giver, Items: []uint16{5, 6}},
		{ID: receiver, Items: []uint16{7}},
	}

	tests := []struct {
		name string
		in   ScriptInstant
	}{
		{"no first reference",
			ScriptInstant{Op: ScriptInstantGiveAll, Unit2: receiver, HasUnit2: true}},
		{"no second reference",
			ScriptInstant{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true}},
		{"a reference naming an entity this world does not hold",
			ScriptInstant{Op: ScriptInstantGiveAll, Unit: 12345, HasUnit: true,
				Unit2: receiver, HasUnit2: true}},
		{"the two references resolving to one entity",
			ScriptInstant{Op: ScriptInstantGiveAll, Unit: giver, HasUnit: true,
				Unit2: giver, HasUnit2: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := mustStockedWorld(t, 1, ents, stock)

			before := mustMarshal(t, w)
			rngBefore := w.rng.state

			w.runInstant(tc.in)

			after := mustMarshal(t, w)
			if string(before) != string(after) {
				t.Errorf("the world's byte form moved under a refusal:\n before % x\n after  % x",
					before, after)
			}
			if w.rng.state != rngBefore {
				t.Errorf("the generator moved from %#016x to %#016x under a refusal (P-2)",
					rngBefore, w.rng.state)
			}
		})
	}
}

// ---------------------------------------------------------------- AC-6

// TestGiveAllIsNoLongerReportedAsAGap is AC-6: a script carrying opcode 28 is
// not in Script.Unsupported, and a trigger naming it is not inert.
func TestGiveAllIsNoLongerReportedAsAGap(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true},
	}, 0)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Errorf("Unsupported() = %+v, want none — opcode 28 is implemented", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Errorf("InertTriggers() = %v, want none — every check the trigger reads is supported",
			inert)
	}
}
