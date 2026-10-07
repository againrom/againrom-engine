package sim

// The container, the purse and the transfer (0112 T1, T3): Stock, carryFault,
// normaliseHoldings, the two readers Carried and Purse, and TakeSack.
//
// The T1 section builds worlds through NewStockedWorld alone and reads them
// back through Carried and Purse, exactly as T1's own boundary says: the
// container and the purse exist, are built and are read back, and nothing
// moves anything into either. The T3 section, below the purse tests, is
// where a sack starts moving: TakeSack, and only TakeSack, writes either
// field from this point on. Nothing in either section reaches the byte form
// directly — that is T2's task and its own tests — though T3's own round
// trip test crosses it to check that what TakeSack writes is ordinary
// canonical state and not a second representation the form has to learn
// about.

import "testing"

// cyBounds is the small extent every fixture here is built against. Entity
// positions are not bounds-checked by this package (sample()'s own point),
// so the value matters only for readability.
var cyBounds = Bounds{Width: 10, Height: 10}

// mustStockedWorld builds a world over ents and stock, with no terrain, no
// script, no relation and no sacks named — the parts this file has nothing
// to say about.
func mustStockedWorld(t *testing.T, seed uint64, ents []Entity, stock []Stock) *World {
	t.Helper()
	w, err := NewStockedWorld(seed, cyBounds, ModeCanonical, Terrain{}, ents, nil, Relations{}, nil, stock)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

// TestCarriedAnswersCodesInOrderWithZeroesDropped is AC-1 and AC-2's positive
// half: a world built with codes answers them, in the order given, with the
// zeroes among them dropped — and an entity named no stock at all answers an
// empty container, not an error.
func TestCarriedAnswersCodesInOrderWithZeroesDropped(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}},
		[]Stock{{ID: 1, Items: []uint16{5, 0, 7, 0, 9}}})

	got, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok for an entity this world holds")
	}
	if !equalCodes(got, []uint16{5, 7, 9}) {
		t.Errorf("Carried(1) = %v, want [5 7 9] — insertion order, zeroes dropped", got)
	}

	// Entity 2 was named no stock at all: an empty container, not an error.
	got2, ok2 := w.Carried(2)
	if !ok2 {
		t.Fatal("Carried(2) answered not-ok for an entity this world holds")
	}
	if len(got2) != 0 {
		t.Errorf("Carried(2) = %v, want an empty container", got2)
	}

	// An id this world does not hold answers not-ok, and a nil container.
	got3, ok3 := w.Carried(99)
	if ok3 || got3 != nil {
		t.Errorf("Carried(99) = (%v, %v), want (nil, false)", got3, ok3)
	}
}

// TestAWorldNamingNoStockAnswersAnEmptyContainerForEveryEntity covers the
// Done-when clause of the same name, over every older constructor: an
// unnamed stock list materialises to exactly what such a world had before
// Stock existed — every entity carrying nothing.
func TestAWorldNamingNoStockAnswersAnEmptyContainerForEveryEntity(t *testing.T) {
	ents := []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}, {ID: 3, X: 3, Y: 3}}

	plain := mustWorld(t, 1, cyBounds, ents)
	terrain, err := NewTerrainWorld(1, cyBounds, ModeCanonical, Terrain{}, ents, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	related, err := NewRelatedWorld(1, cyBounds, ModeCanonical, Terrain{}, ents, nil, Relations{})
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	scripted, err := NewScriptedWorld(1, cyBounds, ModeCanonical, nil, ents, nil)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	loot, err := NewLootWorld(1, cyBounds, ModeCanonical, Terrain{}, ents, nil, Relations{}, nil)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	stocked := mustStockedWorld(t, 1, ents, nil)

	for i, w := range []*World{plain, terrain, related, scripted, loot, stocked} {
		for _, e := range ents {
			got, ok := w.Carried(e.ID)
			if !ok {
				t.Errorf("constructor %d: Carried(%d) answered not-ok", i, e.ID)
			}
			if len(got) != 0 {
				t.Errorf("constructor %d: Carried(%d) = %v, want an empty container", i, e.ID, got)
			}
		}
	}
}

// TestTwoStockNamingTheSameIdAppendInArgumentOrder is normaliseHoldings's own
// rule restated for Stock, on TestThreeEntriesOnOneCellJoinInArgumentOrder's
// own ground (sackform_test.go): a third entry tells apart a fold that
// appends from one that, say, prepends, and interleaved zeroes must not
// disturb the survivors' order.
func TestTwoStockNamingTheSameIdAppendInArgumentOrder(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, []Stock{
		{ID: 1, Items: []uint16{1, 0, 2}},
		{ID: 1, Items: []uint16{0, 3}},
		{ID: 1, Items: []uint16{4}},
	})
	got, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok")
	}
	if !equalCodes(got, []uint16{1, 2, 3, 4}) {
		t.Errorf("Carried(1) = %v, want [1 2 3 4] — argument order, zeroes dropped", got)
	}
}

// TestAStockNamingAnUnknownEntityIsRefused: a Stock names an entity, not a
// cell, so an id this world does not hold is a caller's claim this
// constructor cannot honour — refused rather than folded away, exactly as an
// out-of-bounds sack is (sackFault) and not dropped the way a zero code is.
func TestAStockNamingAnUnknownEntityIsRefused(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{}, []Entity{{ID: 1, X: 1, Y: 1}},
		nil, Relations{}, nil, []Stock{{ID: 99, Items: []uint16{1}}})
	if err == nil {
		t.Fatal("a stock naming an unknown entity was accepted")
	}
	if w != nil {
		t.Error("a refused construction returned a world alongside its error")
	}

	// The control: the same stock naming the entity that DOES exist is
	// accepted, so the case above is refused for naming an unknown id and not
	// because this fixture cannot be built at all.
	if _, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{}, []Entity{{ID: 1, X: 1, Y: 1}},
		nil, Relations{}, nil, []Stock{{ID: 1, Items: []uint16{1}}}); err != nil {
		t.Errorf("the well-formed control was refused: %v", err)
	}
}

func TestCarriedHandsBackACopy(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, []Stock{{ID: 1, Items: []uint16{9}}})

	first, _ := w.Carried(1)
	first[0] = 111

	second, _ := w.Carried(1)
	if second[0] != 9 {
		t.Errorf("mutating one call's result reached the world: Carried(1) = %v", second)
	}

	// And the constructor's own argument is copied too: a caller's slice
	// stays the caller's to reuse or mutate.
	given := []uint16{1, 2}
	w2 := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, []Stock{{ID: 1, Items: given}})
	given[0] = 0xffff
	if got, _ := w2.Carried(1); got[0] != 1 {
		t.Errorf("mutating the constructor's argument reached the world: Carried(1) = %v", got)
	}
}

// TestPurseIsReadablePerSlotAndOutOfRangeAnswersZero is the Done-when clause
// of the same name. T1 builds no writer for a purse — that is TakeSack's,
// T3's own task — so every in-range slot a fresh world holds reads zero; what
// this test asserts is that the read itself is total over uint32 and that
// relationSlots is exactly where it stops answering a real cell, on
// relationIndex's own out-of-range rule (relations_test.go).
func TestPurseIsReadablePerSlotAndOutOfRangeAnswersZero(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1}}, nil)

	for _, slot := range []uint32{0, 1, relationSlots - 1} {
		if got := w.Purse(slot); got != 0 {
			t.Errorf("Purse(%d) = %d, want 0 — nothing has credited a purse yet", slot, got)
		}
	}
	for _, slot := range []uint32{relationSlots, relationSlots + 1, 1 << 31, ^uint32(0)} {
		if got := w.Purse(slot); got != 0 {
			t.Errorf("Purse(%d) = %d, want 0 — slot %d is at or past relationSlots (%d)",
				slot, got, slot, uint32(relationSlots))
		}
	}
}

// ------------------------------------------------------------------- T3, AC-6
//
// TakeSack — the sack becomes what one actor carries. Built through
// NewStockedWorld directly (mustStockedWorld carries no sacks parameter), on
// TestAStockNamingAnUnknownEntityIsRefused's own precedent above.

func TestSetPurseWritesOneValidSlotAndRefusesAnInvalidOne(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1}}, nil)
	if !w.SetPurse(SelfSlot, 700) {
		t.Fatal("SetPurse refused SelfSlot")
	}
	if got := w.Purse(SelfSlot); got != 700 {
		t.Fatalf("Purse(SelfSlot) = %d, want 700", got)
	}
	if w.SetPurse(relationSlots, 9) {
		t.Fatal("SetPurse accepted a slot past the purse array")
	}
	if got := w.Purse(SelfSlot); got != 700 {
		t.Fatalf("a refused SetPurse changed SelfSlot to %d", got)
	}
}

// TestTakeSackMovesEveryCodeAndAllTheGoldInSackOrderAndRemovesIt is AC-6: the
// transfer appends every code in the sack's own order, credits the picking
// entity's owner's purse with the sack's gold, and the sack is gone from
// Sacks() afterwards.
func TestMoveCarriedMovesACompleteStackCountAndRefusesAtomically(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1}, {ID: 2}}, []Stock{
		{ID: 1, Items: []uint16{9, 9, 5}},
		{ID: 2, Items: []uint16{9}},
	})
	if err := w.MoveCarried(1, 2, 9, 2); err != nil {
		t.Fatalf("MoveCarried: %v", err)
	}
	if got, _ := w.CarriedStacks(1); len(got) != 1 || got[0].Code != 5 || got[0].Count != 1 {
		t.Fatalf("source stacks = %v, want only code 5", got)
	}
	if got, _ := w.CarriedStacks(2); len(got) != 1 || got[0].Code != 9 || got[0].Count != 3 {
		t.Fatalf("destination stacks = %v, want code 9 at merged count 3", got)
	}

	before := w.Hash()
	if err := w.MoveCarried(1, 2, 5, 2); err == nil {
		t.Fatal("MoveCarried accepted more units than the source carries")
	}
	if got := w.Hash(); got != before {
		t.Fatalf("refused MoveCarried changed hash from %d to %d", before, got)
	}
}

func TestTakeSackMovesEveryCodeAndAllTheGoldInSackOrderAndRemovesIt(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3, Owner: 5}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{9, 5, 7}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	if err := w.TakeSack(1, 3, 3); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}

	got, ok := w.Carried(1)
	if !ok {
		t.Fatal("Carried(1) answered not-ok")
	}
	if !equalCodes(got, []uint16{9, 5, 7}) {
		t.Errorf("Carried(1) = %v, want [9 5 7] — the sack's own order", got)
	}
	if got := w.Purse(5); got != 40 {
		t.Errorf("Purse(5) = %d, want 40 — the sack's gold credited to the picking entity's owner", got)
	}
	if after := w.Sacks(); len(after) != 0 {
		t.Errorf("Sacks() = %+v after TakeSack, want none left", after)
	}
}

func TestTakeSackWrapsGoldAt32Bits(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 1, Y: 1, Owner: 2}, {ID: 2, X: 3, Y: 3, Owner: 2}}, nil, Relations{},
		[]Sack{{X: 1, Y: 1, Gold: ^uint32(0)}, {X: 3, Y: 3, Gold: 10}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if err := w.TakeSack(1, 1, 1); err != nil {
		t.Fatalf("TakeSack(1): %v", err)
	}
	if got := w.Purse(2); got != ^uint32(0) {
		t.Fatalf("Purse(2) = %d, want %d", got, ^uint32(0))
	}
	if err := w.TakeSack(2, 3, 3); err != nil {
		t.Fatalf("TakeSack(2): %v", err)
	}
	if got := w.Purse(2); got != 9 {
		t.Errorf("Purse(2) = %d, want 9 — %d + 10 wraps at 32 bits", got, ^uint32(0))
	}
}

// ------------------------------------------------------------------- T3, AC-7

// TestTakeSackRefusesAnUnknownEntityAndChangesNothing is AC-7's first half:
// an id this world does not hold is refused, and nothing about the world —
// the sack included — moves.
func TestTakeSackRefusesAnUnknownEntityAndChangesNothing(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Gold: 7, Items: []uint16{9}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	if err := w.TakeSack(99, 3, 3); err == nil {
		t.Fatal("TakeSack named an entity this world does not hold and was not refused")
	}
	if got, _ := w.Carried(1); len(got) != 0 {
		t.Errorf("Carried(1) = %v after a refused TakeSack, want nothing moved", got)
	}
	after := w.Sacks()
	if len(after) != 1 || after[0].X != 3 || after[0].Y != 3 || after[0].Gold != 7 || !equalCodes(after[0].Items, []uint16{9}) {
		t.Errorf("Sacks() changed after a refused TakeSack: %+v", after)
	}
}

// TestTakeSackRefusesACellWithNoSackAndChangesNothing is AC-7's second half:
// a cell holding no sack is refused, and nothing moves.
func TestTakeSackRefusesACellWithNoSackAndChangesNothing(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3}}, nil, Relations{}, nil, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	if err := w.TakeSack(1, 3, 3); err == nil {
		t.Fatal("TakeSack named a cell with no sack and was not refused")
	}
	if got, _ := w.Carried(1); len(got) != 0 {
		t.Errorf("Carried(1) = %v after a refused TakeSack, want nothing moved", got)
	}
	if got := w.Purse(0); got != 0 {
		t.Errorf("Purse(0) = %d after a refused TakeSack, want 0", got)
	}
}

func TestTakeSackRefusesAnOwnerPastTheRosterBeforeMovingAnything(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3, Owner: relationSlots}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Gold: 7, Items: []uint16{9}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	if err := w.TakeSack(1, 3, 3); err == nil {
		t.Fatal("TakeSack named an entity whose owner sits at relationSlots and was not refused")
	}
	if got, _ := w.Carried(1); len(got) != 0 {
		t.Errorf("Carried(1) = %v after a refused TakeSack, want nothing moved", got)
	}
	after := w.Sacks()
	if len(after) != 1 || after[0].X != 3 || after[0].Y != 3 || after[0].Gold != 7 || !equalCodes(after[0].Items, []uint16{9}) {
		t.Errorf("Sacks() changed after a refused TakeSack: %+v", after)
	}

	// The control: Owner one below the same bound is accepted, so the case
	// above is refused for the owner and not because this fixture cannot be
	// taken at all.
	w2, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3, Owner: relationSlots - 1}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Gold: 7, Items: []uint16{9}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld (control): %v", err)
	}
	if err := w2.TakeSack(1, 3, 3); err != nil {
		t.Errorf("the well-formed control was refused: %v", err)
	}
}

// ---------------------------------------------------------- round trip

// TestAWorldThatHasTakenASackStillRoundTripsByteIdentically is T3's own
// Done-when clause: TakeSack writes ordinary canonical state — the
// container and the purse T2 already carries through the form — so acting
// on a live world through it round-trips exactly as any other world does.
func TestAWorldThatHasTakenASackStillRoundTripsByteIdentically(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3, Owner: 5}}, nil, Relations{},
		[]Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{9, 5, 7}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if err := w.TakeSack(1, 3, 3); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if string(again) != string(b) {
		t.Fatal("a world that has taken a sack does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Fatal("a round-tripped world that has taken a sack hashes differently from the original")
	}
}

func TestReplaceStockRestoresOneReboundActorsWholeLoadout(t *testing.T) {
	w, err := NewStockedWorld(1, cyBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 3, Y: 3}, {ID: 2, X: 4, Y: 3}}, nil, Relations{}, nil,
		[]Stock{{ID: 1, Items: []uint16{1}, Equipped: [EquipSlots]uint16{1}},
			{ID: 2, Items: []uint16{9}, Equipped: [EquipSlots]uint16{9}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	wantWorn := [EquipSlots]uint16{0x2126, 0xb223, 0x3344}
	wantItems := []uint16{0x2126, 0xb223, 0x3344, 0x5001, 0x5001, 0x5002}
	if !w.ReplaceStock(Stock{ID: 1, Items: wantItems, Equipped: wantWorn}) {
		t.Fatal("ReplaceStock refused an actor held by the world")
	}
	if got, _ := w.Equipped(1); got != wantWorn {
		t.Errorf("restored equipment = %#v, want %#v", got, wantWorn)
	}
	if got, _ := w.Carried(1); !equalCodes(got, wantItems) {
		t.Errorf("restored container = %v, want %v", got, wantItems)
	}
	if got, _ := w.Equipped(2); got[0] != 9 {
		t.Errorf("unrelated actor's equipment changed to %#v", got)
	}
	if got, _ := w.Carried(2); !equalCodes(got, []uint16{9}) {
		t.Errorf("unrelated actor's container changed to %v", got)
	}
	if w.ReplaceStock(Stock{ID: 99, Items: []uint16{7}}) {
		t.Fatal("ReplaceStock accepted an absent actor")
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, _ := back.Equipped(1); got != wantWorn {
		t.Errorf("round-tripped equipment = %#v, want %#v", got, wantWorn)
	}
	if got, _ := back.Carried(1); !equalCodes(got, wantItems) {
		t.Errorf("round-tripped container = %v, want %v", got, wantItems)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("restored stock changed hash across byte-form round trip")
	}
}
