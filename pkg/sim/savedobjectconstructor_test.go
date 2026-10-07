package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestConstructSavedItem1172DistinctChildrenAndCurrentOperands(t *testing.T) {
	value := ItemStack{Code: 0x0101, Count: 3, Kind: 2, Price: 91, Weight: -7, WeightPresent: true,
		Effects: []ItemEffect{{Kind: 3, Mode: 2, Operand: 0x81234567}, {Kind: 3, Mode: 2, Operand: 0x81234567}},
		SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 9, OwnKind: 4,
			Spell: SourceItemSpell{Present: true, ID: 12, Range: 8, Defensive: 1, ManaCost: 17}}}
	value.SourceEquipment.Attack[3], value.SourceEquipment.Defence[7] = 28, 51
	before := value.Clone()
	next := SavedObjectID(1)
	mint := func() (SavedObjectID, SavedObjectToken, error) {
		id := next
		next++
		return id, SavedObjectToken{Identity: uint32(id) * 16, T0E: 0x21}, nil
	}
	row, effects, spell, err := ConstructSavedItem(value, mint)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value, before) {
		t.Fatal("constructor changed caller's current item")
	}
	want := value.Clone()
	want.ObjectID = row.ID
	if !StackStateEqual(row.Value, want) || row.Token.T1C != uint32(value.Price) || row.Token.T0C != 9 || row.F45 != 0 || row.F46 != 0 || row.F47 != 0 || row.F48 != 0 {
		t.Fatal("constructor lost current operands or explicit native defaults")
	}
	if len(effects) != 2 || effects[0].ID == effects[1].ID || effects[0].Token.Identity == effects[1].Token.Identity || len(row.Effects) != 2 || row.Effects[0] != effects[0].ID || row.Effects[1] != effects[1].ID {
		t.Fatal("equal-valued Effect instances collapsed or reordered")
	}
	if spell == nil || row.Spell != spell.ID || spell.This == 0 || spell.Value != value.SourceEquipment.Spell {
		t.Fatal("owned Spell edge/operands lost")
	}
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: next, Items: []SavedItemObject{row}, Effects: effects, Spells: []SavedSpellObject{*spell}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Effects[1].ID = r.Effects[0].ID
	if err := r.Validate(); err == nil {
		t.Fatal("collapsed child admitted")
	}
}

func TestConstructSavedItem1172RefusesUnspecifiedAndOverflow(t *testing.T) {
	called := false
	mint := func() (SavedObjectID, SavedObjectToken, error) {
		called = true
		return 1, SavedObjectToken{Identity: 1}, nil
	}
	for _, value := range []ItemStack{
		{Code: 0x0e1e, Count: 1},
		{Code: 0x0e1e, Count: 65536, WeightPresent: true},
	} {
		if _, _, _, err := ConstructSavedItem(value, mint); err == nil {
			t.Fatal("unsupported constructor accepted", value)
		}
	}
	if called {
		t.Fatal("refused constructor consumed allocator")
	}
	base := ItemStack{Code: 0x0101, Count: 1, WeightPresent: true}
	row, _, _, err := ConstructSavedItem(base, mint)
	if err != nil || row.Value.SourceEquipment.Class != 0 || row.Value.Code != base.Code {
		t.Fatal("explicit base Item became equipment", row, err)
	}
}

func acquiredPackWorld1172(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 1172, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20}})
	if err := w.ImportSavedObjects(&SavedObjects{Version: SavedObjectsVersion, NextID: 1}, nil); err != nil {
		t.Fatal(err)
	}
	item := ItemInstance{Code: 0x0e11, Kind: 1, Price: 13, WeightPresent: true, Weight: 2}
	w.sacks = []Sack{makeSack(1, 1, 0, []ItemInstance{item})}
	if err := w.TakeSack(7, 1, 1); err != nil {
		t.Fatal(err)
	}
	if !w.addCarried(1, StackItem(item, 1)) {
		t.Fatal("second native acquisition failed")
	}
	w.recomputeLoad(1)
	a, b := w.carried[0][0], w.carried[1][0]
	b.ObjectID = a.ObjectID // compare operands independently of the identity relation
	if w.carried[0][0].ObjectID == w.carried[1][0].ObjectID || !StackStateEqual(a, b) {
		t.Fatal("control needs equal-valued Items with distinct owners/identities")
	}
	if w.savedObjects.item(w.carried[0][0].ObjectID).Token.T08 != 1 || w.savedObjects.item(w.carried[1][0].ObjectID).Token.T08 != 0 {
		t.Fatal("ground pickup and ordinary constructor flags did not use their own policies")
	}
	_ = mustMarshal(t, w)
	return w
}

func TestAcquiredSavedItem1172EqualCodeOwnersAndSingleCauseLosses(t *testing.T) {
	for name, edit := range map[string]func(*World){
		"swapped equal-code owners": func(w *World) {
			w.carried[0][0].ObjectID, w.carried[1][0].ObjectID = w.carried[1][0].ObjectID, w.carried[0][0].ObjectID
		},
		"foreign handle":       func(w *World) { w.carried[0][0].ObjectID = 999 },
		"orphan current item":  func(w *World) { w.carried[0] = nil },
		"stale registry count": func(w *World) { w.savedObjects.Items[0].Value.Count++ },
	} {
		t.Run(name, func(t *testing.T) {
			w := acquiredPackWorld1172(t)
			edit(w)
			before := w.encode()
			if out, err := w.MarshalBinary(); err == nil || out != nil {
				t.Fatal("single-cause current item loss was admitted")
			}
			if !bytes.Equal(before, w.encode()) {
				t.Fatal("admission repaired the contradictory identity")
			}
		})
	}
	w := acquiredPackWorld1172(t)
	originalID := w.carried[0][0].ObjectID
	w = reloadOperations1115(t, w)
	if err := w.MoveCarried(7, 8, 0x0e11, 1); err != nil {
		t.Fatal(err)
	}
	if len(w.carried[0]) != 0 || len(w.carried[1]) != 1 || w.carried[1][0].Count != 2 || !w.savedObjects.item(originalID).Retired {
		t.Fatal("next production transfer did not merge/retire the exact acquired item")
	}
	_ = reloadOperations1115(t, w)
}

func TestAcquiredSavedItem1172FailedInsertionDoesNotAllocate(t *testing.T) {
	w := acquiredPackWorld1172(t)
	w.entities[0].ActorLoad.Present, w.entities[0].ActorLoad.ContainerPresent = true, false
	before := w.encode()
	if w.addCarried(0, ItemStack{Code: 0x0e12, Count: 1, WeightPresent: true}) {
		t.Fatal("absent destination accepted acquisition")
	}
	if !bytes.Equal(before, w.encode()) {
		t.Fatal("failed insertion allocated identities or changed current state")
	}
}
