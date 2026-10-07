package sim

import (
	"reflect"
	"testing"
)

func TestCurrentUnboundSackValuesKeepRootAndOrderedUnits(t *testing.T) {
	w := mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1}})
	initial := []Sack{{X: 1, Y: 2, Gold: 17, ItemInstances: []ItemInstance{PlainItem(3)}}, {X: 2, Y: 3, Gold: 23}}
	if err := w.ReplaceGroundSacks(initial); err != nil {
		t.Fatal(err)
	}
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: 1}
	if err := w.ImportSavedObjects(r, nil); err != nil {
		t.Fatal(err)
	}
	value := Sack{X: 1, Y: 2, Gold: 19, ItemInstances: []ItemInstance{PlainItem(5), PlainItem(3), PlainItem(5)}}
	for _, bad := range [][]Sack{{value, value}, {{X: 3, Y: 3}}, {{ObjectID: 1, X: 1, Y: 2}}, {{X: 1, Y: 2, ItemInstances: []ItemInstance{{Code: 3, Weight: 4}}}}} {
		before := w.Hash()
		if err := w.ReplaceCurrentObjects(r, nil, nil, bad...); err == nil || w.Hash() != before {
			t.Fatal("invalid ground target partially changed the World")
		}
	}
	if err := w.ReplaceCurrentObjects(r, nil, nil, value); err != nil {
		t.Fatal(err)
	}
	got := w.Sacks()
	if len(got) != 2 || got[0].ObjectID != 0 || got[0].Gold != 19 || got[1].Gold != 23 {
		t.Fatal("ordinary ground values changed the root population or order", got)
	}
	items, err := sackItems(got[0])
	if err != nil || !reflect.DeepEqual(items, value.ItemInstances) {
		t.Fatal("ordinary ground values changed the ordered items", items, err)
	}
	value.ItemInstances[0].Code = 7
	items, err = sackItems(w.Sacks()[0])
	if err != nil || items[0].Code != 5 {
		t.Fatal("ordinary ground values retained caller storage")
	}
}

func TestCurrentUnboundSackRequiresExplicitIdentityRemoval(t *testing.T) {
	w := goldPickupSavedWorld(t, 31)
	old := w.Sacks()[0]
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: w.SavedObjects().NextID}
	value := Sack{X: old.X, Y: old.Y, Gold: old.Gold}
	before := w.Hash()
	for _, identities := range []map[SavedObjectID]SavedObjectID{nil, {old.ObjectID + 1: 0}, {old.ObjectID: old.ObjectID}} {
		if err := w.ReplaceCurrentObjects(r, identities, nil, value); err == nil || w.Hash() != before {
			t.Fatal("unregistered ground replaced an unrelated bound Sack", identities, err)
		}
	}
	if err := w.ReplaceCurrentObjects(r, map[SavedObjectID]SavedObjectID{old.ObjectID: 0}, nil, value); err != nil {
		t.Fatal(err)
	}
	if got := w.Sacks(); len(got) != 1 || got[0].ObjectID != 0 || got[0].X != old.X || got[0].Y != old.Y || got[0].Gold != old.Gold || len(w.SavedObjects().Sacks) != 0 {
		t.Fatal("explicit native handle absence changed Sack values", got)
	}
}
