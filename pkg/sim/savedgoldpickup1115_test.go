package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

const goldObject1115 SavedObjectID = 0x100000002

func goldPickupSavedWorld(t *testing.T, gold uint32) *World {
	t.Helper()
	w := mustWorld(t, 1115, Bounds{32, 32}, []Entity{{ID: 7, X: 1, Y: 1, Owner: 1, HP: 20, MaxHP: 20}})
	if err := w.ReplaceGroundSacks([]Sack{{X: 10, Y: 20, Gold: gold}}); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalStructures(nil, nil, nil, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	payload := sackPayload1115(sackKey1115)
	if err := w.ImportOriginalActorMotions(nil, []SavedActorCell{{Cell: sackCell1115, Payload: payload}}, nil); err != nil {
		t.Fatal(err)
	}
	planes := new(SavedCellPlanes)
	planes.Cost[sackCell1115], planes.CostKnown[sackCell1115] = 37, 1
	planes.Static[sackCell1115], planes.Dynamic[sackCell1115] = 0x32, 0x90
	planes.Costs[0], planes.Costs[5] = 255, 6
	if err := w.ImportOriginalCellPlanes(planes); err != nil {
		t.Fatal(err)
	}
	token := SavedObjectToken{Identity: sackKey1115, Reference: 0xffffffff, RuntimeID: 0x80000000, T08: 0xabcd, T1C: 17, T0E: 23, T18: 42}
	binary.LittleEndian.PutUint16(token.Position[2:], sackCell1115)
	token.Position[4], token.Position[5] = 128, 128
	r := &SavedObjects{Version: 1, NextID: goldObject1115 + 1,
		Sacks:      []SavedSackObject{{ID: goldObject1115, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Token: token, Gold: gold}},
		SackRoots:  []SavedObjectID{goldObject1115, goldObject1115},
		Containers: []SavedObjectContainer{{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: goldObject1115}, Present: true, InsertIndex: 7, Accumulator: -19}},
	}
	if err := w.ImportSavedObjects(r, []SavedSackBinding{{ID: goldObject1115, X: 10, Y: 20, Gold: gold}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSavedGoldPickup1115RealTransferRetiresSackAndCell(t *testing.T) {
	for _, gold := range []uint32{0, 31, 0x80000001, 0xffffffff} {
		w := goldPickupSavedWorld(t, gold)
		w.SetPurse(1, 2)
		before := mustMarshal(t, w)
		want := copySackCellSavedWorld(t, w)
		// Independently stated result: the actual TakeSack path must change
		// all four owners, not just make the native Sack invisible.
		want.purses[1] = gold + 2
		want.sacks = nil
		want.savedObjects.Sacks[0].Retired = true
		want.savedObjects.Sacks[0].Gold = 0
		want.savedObjects.SackRoots, want.savedObjects.Containers = nil, nil
		want.savedMotion.Cells, want.savedStructureCells = nil, nil
		// Static bit1 blocks air. This fixture has no native magic wall
		// (bit2) to preserve, unlike the separate cell-helper fixture.
		literalSackPlanes1115(want, 11, 0x12, 0x32, 2)
		if err := w.TakeSack(7, 10, 20); err != nil {
			t.Fatal(err)
		}
		assertSackCellWorld1115(t, w, want, before, true)
		frozen := mustMarshal(t, w)
		if err := w.TakeSack(7, 10, 20); err == nil || !bytes.Equal(frozen, mustMarshal(t, w)) {
			t.Fatal("second pickup revived Sack or credited gold twice")
		}
		cold := copySackCellSavedWorld(t, w)
		for range 20 {
			Step(w, nil)
			Step(cold, nil)
			if w.Hash() != cold.Hash() {
				t.Fatal("native continuation changed after gold Sack retirement")
			}
		}
	}
}

func TestSavedGoldPickup1115FailureIsAtomicAndGetterDetached(t *testing.T) {
	w := goldPickupSavedWorld(t, 19)
	copy := w.SavedObjects()
	copy.Sacks[0].Gold = 3
	copy.Containers[0].Accumulator = 88
	if reflect.DeepEqual(copy, w.SavedObjects()) {
		t.Fatal("getter exposes world registry")
	}
	// Unresolved Building authority prevents recompute. The attempted
	// removal must not clear +10, credit gold, retire roots or edit planes.
	binary.LittleEndian.PutUint32(w.savedMotion.Cells[0].Payload[12:], 0x87654321)
	before := mustMarshal(t, w)
	if err := w.TakeSack(7, 10, 20); err == nil {
		t.Fatal("unknown cell recompute admitted")
	}
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("failed pickup partially changed live owners")
	}
}

func TestSavedGoldPickup1115NativeItemsStayUnboundButMove(t *testing.T) {
	w := goldPickupSavedWorld(t, 0xffffffff)
	w.pourSack(10, 20, 2, []ItemInstance{PlainItem(0x3185)})
	if w.savedObjects.Sacks[0].Gold != 1 || w.savedObjects.Sacks[0].Coverage.Unknown&SavedUnknownContainerLoad == 0 || w.sacks[0].ItemInstances[0].ObjectID != 0 {
		t.Fatal("native addition erased identity, invented item authority or lost unsigned gold")
	}
	cold := copySackCellSavedWorld(t, w)
	if err := cold.TakeSack(7, 10, 20); err != nil {
		t.Fatal(err)
	}
	items, ok := cold.CarriedStacks(7)
	if !ok || len(items) != 1 || items[0].Code != 0x3185 || items[0].Count != 1 || items[0].ObjectID != 0 || cold.Purse(1) != 1 {
		t.Fatal("actual mixed pickup lost the ordinary native item or purse")
	}
	if !cold.SavedObjects().Sacks[0].Retired {
		t.Fatal("mixed pickup did not retire exact source Sack")
	}
	_ = copySackCellSavedWorld(t, cold)
}

func TestSavedGoldPickup1115ImportRejectsWrongKeyWithoutMutation(t *testing.T) {
	w := goldPickupSavedWorld(t, 11)
	registry := w.SavedObjects()
	w.savedObjects = nil
	w.sacks[0].ObjectID = 0
	before := mustMarshal(t, w)
	registry.Sacks[0].Token.Identity++
	if err := w.ImportSavedObjects(registry, []SavedSackBinding{{ID: goldObject1115, X: 10, Y: 20, Gold: 11}}); err == nil {
		t.Fatal("binding guessed by same cell/value despite different exact source key")
	}
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("failed import changed native state")
	}
}
