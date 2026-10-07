package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func testSpellBook(spell uint32) ItemInstance {
	return ItemInstance{
		Code:    0x0e17,
		Kind:    5,
		Effects: []ItemEffect{{Kind: 42, Operand: spell}},
	}
}

func bookWorld(t *testing.T, e Entity, items ...ItemInstance) *World {
	t.Helper()
	w, err := NewStockedWorld(1051, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{},
		[]Entity{e}, nil, Relations{}, nil, []Stock{{ID: e.ID, ItemInstances: items}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return w
}

func bookEntity(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("world has no entity %d", id)
	return Entity{}
}

func TestReadingAOneSpellBookLearnsAndConsumesExactlyOneUnit(t *testing.T) {
	book := testSpellBook(26)
	w := bookWorld(t, Entity{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10, Mana: 20, MaxMana: 20}, book, book)

	Step(w, []Command{{Kind: KindReadBook, Entity: 7, X: 0}})

	if got := bookEntity(t, w, 7).KnownSpells; got != uint32(1)<<26 {
		t.Fatalf("KnownSpells = %#x, want Teleport bit %#x", got, uint32(1)<<26)
	}
	stacks, ok := w.CarriedStacks(7)
	if !ok || len(stacks) != 1 || stacks[0].Count != 1 || !reflect.DeepEqual(stacks[0].Instance(), book) {
		t.Fatalf("carried after first read = %#v (ok %v), want one identical book", stacks, ok)
	}

	// A valid duplicate is still a consumed book: learning is idempotent, use
	// is not. The mask stays fixed while the final unit leaves the container.
	Step(w, []Command{{Kind: KindReadBook, Entity: 7, X: 0}})
	if got := bookEntity(t, w, 7).KnownSpells; got != uint32(1)<<26 {
		t.Fatalf("KnownSpells after duplicate = %#x, want unchanged Teleport bit", got)
	}
	if stacks, _ := w.CarriedStacks(7); len(stacks) != 0 {
		t.Fatalf("carried after second read = %#v, want empty", stacks)
	}
}

func TestReadingABookRefusesEveryInapplicableReaderAndPayloadBeforeMutation(t *testing.T) {
	valid := testSpellBook(26)
	cases := []struct {
		name  string
		actor Entity
		item  ItemInstance
		index int32
	}{
		{name: "fighter", actor: Entity{ID: 7, HP: 10, MaxHP: 10}, item: valid},
		{name: "dead mage", actor: Entity{ID: 7, HP: -10, MaxHP: 10, MaxMana: 20, Decay: DecayBones}, item: valid},
		{name: "negative index", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: valid, index: -1},
		{name: "past end", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: valid, index: 1},
		{name: "not a book", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: ItemInstance{Code: valid.Code, Kind: 4, Effects: valid.Effects}},
		{name: "no teach effect", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: ItemInstance{Code: valid.Code, Kind: 5}},
		{name: "two teach effects", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: ItemInstance{Code: valid.Code, Kind: 5, Effects: []ItemEffect{{Kind: 42, Operand: 26}, {Kind: 42, Operand: 1}}}},
		{name: "zero spell", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: testSpellBook(0)},
		{name: "spell outside mask", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: testSpellBook(32)},
		{name: "high operand residue", actor: Entity{ID: 7, HP: 10, MaxHP: 10, MaxMana: 20}, item: testSpellBook(0x1001)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := bookWorld(t, tc.actor, tc.item)
			beforeItems, _ := w.CarriedItems(7)
			Step(w, []Command{{Kind: KindReadBook, Entity: 7, X: tc.index}})
			if got := bookEntity(t, w, 7).KnownSpells; got != 0 {
				t.Fatalf("KnownSpells = %#x, want unchanged zero", got)
			}
			afterItems, _ := w.CarriedItems(7)
			if len(afterItems) != len(beforeItems) || len(afterItems) != 1 || !ItemEqual(afterItems[0], beforeItems[0]) {
				t.Fatalf("carried changed from %#v to %#v", beforeItems, afterItems)
			}
		})
	}
}

func TestLearnedBookSpellAndRemainingBookRoundTripAndReplayDeterministically(t *testing.T) {
	book := testSpellBook(26)
	a := bookWorld(t, Entity{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10, MaxMana: 20}, book, book)
	b := bookWorld(t, Entity{ID: 7, X: 2, Y: 2, HP: 10, MaxHP: 10, MaxMana: 20}, book, book)
	cmd := []Command{{Kind: KindReadBook, Entity: 7, X: 0}}
	Step(a, cmd)
	Step(b, cmd)
	if a.Hash() != b.Hash() {
		t.Fatalf("same book command hashes %#x and %#x", a.Hash(), b.Hash())
	}

	form, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after restore: %v", err)
	}
	if !bytes.Equal(form, again) || back.Hash() != a.Hash() {
		t.Fatalf("book state did not round-trip byte-identically: hashes %#x/%#x", a.Hash(), back.Hash())
	}
	if got := bookEntity(t, &back, 7).KnownSpells; got != uint32(1)<<26 {
		t.Fatalf("restored KnownSpells = %#x", got)
	}
	if stacks, _ := back.CarriedStacks(7); len(stacks) != 1 || stacks[0].Count != 1 {
		t.Fatalf("restored carried = %#v, want one remaining book", stacks)
	}
}
