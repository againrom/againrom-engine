package sim

import (
	"reflect"
	"testing"
)

func bookRootWorld(t *testing.T) *World {
	t.Helper()
	spell := SourceItemSpell{Present: true, ID: 1, Range: 9, Defensive: 0x83, ManaCost: 54321}
	book := Spellbook{State: BookPresent}
	book.Slots[0] = BookSpell{Range: spell.Range, Defensive: spell.Defensive, ManaCost: spell.ManaCost}
	value := ItemStack{Code: 0x111, Count: 1, Kind: 2, WeightPresent: true, Weight: 1,
		SourceEquipment: SourceEquipment{Class: SourceWeapon, Spell: spell}}
	next := SavedObjectID(1)
	item, effects, child, err := ConstructSavedItem(value, func() (SavedObjectID, SavedObjectToken, error) {
		id := next
		next++
		return id, SavedObjectToken{Identity: uint32(id) + 100}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	item.InFlight = 0
	owner := SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 0, Slot: 1}
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: next, Items: []SavedItemObject{item}, Effects: effects, Spells: []SavedSpellObject{*child},
		ItemRoots: []SavedItemRoot{{ID: item.ID, Owner: owner}}, BookRoots: []SavedBookRoot{
			{Entity: 0, Slots: [28]SavedObjectID{child.ID}}, {Entity: 1, Slots: [28]SavedObjectID{child.ID}}}}
	worn := [EquipSlots]ItemInstance{value.Instance()}
	w, err := NewStockedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{{ID: 0, X: 1, Y: 1, HP: 10, MaxHP: 10, KnownSpells: 2, Book: book},
			{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10, KnownSpells: 2, Book: book}}, nil, Relations{}, nil, []Stock{{ID: 0, EquippedItems: worn}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedObjects(r, nil, SavedObjectBinding{ID: item.ID, Owner: owner, Value: item.Value}); err != nil {
		t.Fatal(err)
	}
	return w
}

func checkBookRootBinary(t *testing.T, w *World) {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() || !reflect.DeepEqual(back.SavedObjects(), w.SavedObjects()) {
		t.Fatal("book identity or current graph changed across binary cycle")
	}
}

func TestSavedBookRootsShareCopyAndRetireWithoutChangingOtherOwners(t *testing.T) {
	w := bookRootWorld(t)
	checkBookRootBinary(t, w)
	original := w.SavedObjects().Spells[0]
	w.entities[0].Book.Slots[0].Range++
	w.refreshSavedBookRoots(0)
	r := w.SavedObjects()
	changed := r.BookRoots[0].Slots[0]
	if changed == 0 || changed == original.ID || r.BookRoots[1].Slots[0] != original.ID || r.Items[0].Spell != original.ID || *r.spell(original.ID) != original {
		t.Fatal("book edit changed another incoming owner", r)
	}
	checkBookRootBinary(t, w)
	w.entities[0].Book.Slots[0].ManaCost++
	w.refreshSavedBookRoots(0)
	if w.SavedObjects().BookRoots[0].Slots[0] != changed {
		t.Fatal("unique Spell edit replaced its identity")
	}
	w.entities[0].KnownSpells, w.entities[0].Book.Slots[0] = 0, BookSpell{}
	w.refreshSavedBookRoots(0)
	r = w.SavedObjects()
	if r.BookRoots[0].Slots[0] != 0 || !r.spell(changed).Retired || r.spell(original.ID).Retired {
		t.Fatal("book removal retired the wrong child")
	}
	checkBookRootBinary(t, w)
}

func TestSavedBookRootsRejectMalformedAndKeepValuesWhenUnbound(t *testing.T) {
	w := bookRootWorld(t)
	for _, edit := range []func(*SavedObjects){
		func(r *SavedObjects) { r.BookRoots[1].Entity = r.BookRoots[0].Entity },
		func(r *SavedObjects) { r.BookRoots[0].Slots[1] = r.BookRoots[0].Slots[0] },
		func(r *SavedObjects) { r.BookRoots[0].Slots[0] = r.Items[0].ID },
	} {
		r := w.SavedObjects()
		edit(r)
		if err := r.Validate(); err == nil {
			t.Fatal("malformed book identity admitted")
		}
	}
	w.savedObjects.NextID = ^SavedObjectID(0)
	w.entities[0].Book.Slots[0].Range++
	w.refreshSavedBookRoots(0)
	if w.savedObjects.BookRoots[0].Slots[0] != 0 || w.entities[0].Book.Slots[0].Range != 10 || w.savedObjects.BookRoots[1].Slots[0] == 0 {
		t.Fatal("exhausted native identity changed a value or another root")
	}
	checkBookRootBinary(t, w)
}

func TestSavedBookRefreshFindsLastOwnerAndLeavesUnboundActorAlone(t *testing.T) {
	w := bookRootWorld(t)
	before := w.SavedObjects()
	w.entities[1].Book.Slots[0].Range = 17
	w.refreshSavedBookRoots(1)
	after := w.SavedObjects()
	if after.BookRoots[0] != before.BookRoots[0] || !reflect.DeepEqual(after.Items[0], before.Items[0]) {
		t.Fatal("last book edit changed first actor or weapon")
	}
	child := after.spell(after.BookRoots[1].Slots[0])
	if child == nil || child.Value.Range != 17 || child.ID == before.BookRoots[1].Slots[0] {
		t.Fatal("last actor did not receive its own changed spell")
	}
	checkBookRootBinary(t, w)
	w.savedObjects.BookRoots = w.savedObjects.BookRoots[1:]
	before = w.SavedObjects()
	w.entities[0].Book.Slots[0].Range = 23
	w.refreshSavedBookRoots(0)
	if !reflect.DeepEqual(w.SavedObjects(), before) {
		t.Fatal("unbound actor changed another book root")
	}
}
