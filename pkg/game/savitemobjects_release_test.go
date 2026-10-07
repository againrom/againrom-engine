package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func itemObjectRecord1115(t *testing.T, s Snapshot, id sim.SavedObjectID) sav.DocumentRecordData {
	t.Helper()
	for _, binding := range s.SavedDocument.Objects.Items {
		if binding.ID == id && binding.ObjectIndex != 0 {
			return s.SavedDocument.Document.Objects[binding.ObjectIndex-1]
		}
	}
	t.Fatal("live Item lost its current document edge", id)
	return sav.DocumentRecordData{}
}

func itemObjectValue1115(t *testing.T, r sav.DocumentRecordData, name string) uint32 {
	t.Helper()
	for _, v := range r.Values {
		if v.Name == name {
			return v.Value
		}
	}
	t.Fatal("independent source record lacks operand", r.Class, name)
	return 0
}

// itemObjectCheckpoint SAVEs through the menu and LOADs the .sav through the
// load window in a fresh FrontEnd.
func itemObjectCheckpoint(t *testing.T, f *FrontEnd, app *ui.App) (*FrontEnd, *ui.App, []byte) {
	t.Helper()
	store, name, raw := menuSAVE(t, f, app, OriginalStore{})
	cold, coldApp := loadSAVWindow(t, store, name)
	requireSameItemObjects(t, f.live.world, cold.live.world)
	return cold, coldApp, raw
}

// Every Item, Effect, Spell and Sack keeps its ID, value and registry place.
func requireSameItemObjects(t *testing.T, saved, loaded *sim.World) {
	t.Helper()
	if !reflect.DeepEqual(loaded.SavedObjects(), saved.SavedObjects()) {
		t.Fatalf("loaded registry %+v, saved %+v", loaded.SavedObjects(), saved.SavedObjects())
	}
}

// An altered copy of raw loads Item id one weight heavier at the same place.
func requireAlteredItemWeight(t *testing.T, saved *sim.World, raw []byte, id sim.SavedObjectID) {
	t.Helper()
	row, ok := saved.SavedObjects().Item(id)
	if !ok {
		t.Fatal("loss-control Item is absent", id)
	}
	altered := loadAlteredSAV(t, raw, func(doc *sav.DocumentData) bool {
		return setSavedItemWeight(doc, row.Value.Code, row.Value.Weight+1)
	}).live.world.SavedObjects()
	got, ok := altered.Item(id)
	if !ok || got.Value.Weight != row.Value.Weight+1 || !reflect.DeepEqual(altered.Locations(id), saved.SavedObjects().Locations(id)) {
		t.Fatalf("altered Item %d loaded as %+v at %v", id, got.Value, altered.Locations(id))
	}
}

func setSavedItemWeight(doc *sav.DocumentData, code uint16, weight int16) bool {
	found := -1
	for i := range doc.Objects {
		if c, err := savedStructureValue(&doc.Objects[i], "F40"); err == nil && c == uint32(code) {
			if found >= 0 {
				return false
			}
			found = i
		}
	}
	return found >= 0 && savedStructureSetValue(&doc.Objects[found], "F4A", uint32(uint16(weight))) == nil
}

// One lawful source has four Sacks, an already-filled twelve-row hero pack,
// counts up to twelve, two party members and six worn pieces. EN/RU are two
// consumers of the same recording, not independently recorded original runs.
func TestReleaseOriginalItemObjects1115PickupTransferEquipContinuation(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	sourceHash := sha256.Sum256(raw)
	defer func() {
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(after) != sourceHash {
			t.Fatal("read-only natural source changed", err)
		}
	}()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := source.Party()
	if err != nil || len(party) != 2 {
		t.Fatal("natural party population changed", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || len(doc.World.Sacks) != 4 {
		t.Fatal("natural Sack population changed", err)
	}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("natural original LOAD", err)
	}
	app := f.App("natural current Item ownership")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var hero, other sim.EntityID
	for _, character := range party {
		var id sim.EntityID
		for _, entity := range f.live.world.Entities() {
			if entity.SourceBinding.Identity == character.Key {
				if id != 0 {
					t.Fatal("natural actor source key is ambiguous")
				}
				id = entity.ID
			}
		}
		if id == 0 {
			t.Fatal("natural party actor is absent")
		}
		if character.Hero {
			hero = id
			want := []uint16{1, 1, 1, 10, 9, 1, 1, 1, 2, 12, 2, 1}
			if len(character.Items) != len(want) {
				t.Fatal("natural nonempty pack control changed")
			}
			for i, piece := range character.Items {
				if piece.Stack != want[i] {
					t.Fatal("natural source stack count changed", i)
				}
			}
		} else {
			other = id
		}
		pack, _ := f.live.world.CarriedStacks(id)
		if len(pack) != len(character.Items) {
			t.Fatal("source Item rows folded before identity adoption")
		}
		registry := f.live.world.SavedObjects()
		for i, piece := range character.Items {
			st := pack[i]
			row, ok := registry.Item(st.ObjectID)
			if !ok || st.Code != piece.Code || st.Count != uint32(piece.Stack) || st.Price != piece.Price || st.Weight != piece.Weight || !st.WeightPresent || st.Kind != piece.Kind || !registry.HasLocation(row.ID, sim.SavedItemLocation{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: id}, Index: uint32(i)}) {
				t.Fatal("source pack values/order/ownership changed", i)
			}
		}
		worn, _ := f.live.world.EquippedItems(id)
		if len(character.Worn) != 3 {
			t.Fatal("natural worn control changed")
		}
		for _, piece := range character.Worn {
			slot := uint32(piece.Code>>8) & 15
			row, ok := registry.Item(worn[slot-1].ObjectID)
			if !ok || row.Value.Code != piece.Code || !registry.HasRoot(row.ID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: id, Slot: slot}) {
				t.Fatal("source worn identity absent", slot)
			}
		}
	}
	if hero == 0 || other == 0 || hero == other {
		t.Fatal("natural two-owner control missing")
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	// Check named wire operands independently of the new production projector.
	for _, binding := range before.SavedDocument.Objects.Items {
		row, ok := f.live.world.SavedObjects().Item(binding.ID)
		if !ok || binding.ObjectIndex == 0 {
			t.Fatal("original Item lacks a live registry row")
		}
		record := doc.Objects[binding.ObjectIndex-1]
		for name, want := range map[string]uint32{
			"F40": uint32(row.Value.Code), "F42": row.Value.Count, "F44": uint32(row.Value.Kind),
			"F45": uint32(row.F45), "F46": uint32(row.F46), "F47": uint32(row.F47), "F48": uint32(row.F48),
			"T08": row.Token.T08, "Identity": row.Token.Identity, "Reference": row.Token.Reference,
		} {
			if itemObjectValue1115(t, record, name) != want {
				t.Fatal("native Item did not retain independent source operand", name)
			}
		}
	}
	f, app, _ = itemObjectCheckpoint(t, f, app)
	w := f.live.world
	sack := groundAt(w.Sacks(), 42, 39)
	if sack == nil || sack.ObjectID == 0 || sack.Gold != 264 || len(sack.ItemInstances) != 2 || sack.ItemInstances[0].Code != 13345 || sack.ItemInstances[1].Code != 4396 {
		t.Fatal("natural item-Sack control missing")
	}
	picked := []sim.SavedObjectID{sack.ItemInstances[0].ObjectID, sack.ItemInstances[1].ObjectID}
	if picked[0] == 0 || picked[1] == 0 || picked[0] == picked[1] {
		t.Fatal("source Sack Items have no distinct identities")
	}
	liveTakeAt(t, f.live, hero, 42, 39)
	afterPickup, _, err := f.Snapshot(true)
	if err != nil || len(afterPickup.SavedDocument.Document.Objects) != len(before.SavedDocument.Document.Objects)-1 || len(afterPickup.SavedDocument.Document.World.Sacks) != 3 {
		t.Fatal("pickup did not retire only the Sack after rewiring its Items", err)
	}
	for _, id := range picked {
		row, ok := w.SavedObjects().Item(id)
		record := itemObjectRecord1115(t, afterPickup, id)
		at := w.SavedObjects().Locations(id)
		if !ok || len(at) != 1 || at[0].Owner != (sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: hero}) || row.Token.T08 != 0 || itemObjectValue1115(t, record, "T08") != 0 {
			t.Fatal("pickup stamp/current owner did not reach native and document")
		}
	}
	f, app, _ = itemObjectCheckpoint(t, f, app)
	w = f.live.world
	if err := w.MoveCarried(hero, other, 0xa70f, 3); err != nil {
		t.Fatal("natural three-unit partial transfer", err)
	}
	for _, check := range []struct {
		id    sim.EntityID
		count uint32
	}{{hero, 7}, {other, 3}} {
		pack, _ := w.CarriedStacks(check.id)
		found := false
		for _, st := range pack {
			if st.Code == 0xa70f {
				found = true
				if st.ObjectID == 0 || st.Count != check.count {
					t.Fatal("partial transfer lost identity or quantity")
				}
			}
		}
		if !found {
			t.Fatal("partial transfer lost stack")
		}
	}
	f, app, _ = itemObjectCheckpoint(t, f, app)
	pack, _ := f.live.world.CarriedStacks(hero)
	index := -1
	for i, st := range pack {
		if st.ObjectID == picked[0] {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("picked Armor missing before equip")
	}
	f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindEquip, Entity: hero, X: int32(index), Y: 4})
	f.live.tick()
	worn, _ := f.live.world.EquippedItems(hero)
	if worn[3].ObjectID != picked[0] {
		t.Fatal("actual equip did not retain the picked Armor identity")
	}
	cold, _, written := itemObjectCheckpoint(t, f, app)
	if !cold.live.world.SavedObjects().HasRoot(picked[0], sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: hero, Slot: 4}) {
		t.Fatal("loaded Armor is not worn by the hero")
	}
	requireAlteredItemWeight(t, f.live.world, written, picked[0])
	for range 40 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() || !bytes.Equal(f.live.fog.project(), cold.live.fog.project()) {
			t.Fatal("post-item-action native continuation differs")
		}
	}
	t.Log("natural M31: nonempty12-row pack, source counts10/9/12, six worn Items; two Sack Items picked; partial3 transferred, Armor equipped; menu SAVE to .sav and fresh load-window LOAD at four cuts, altered Armor weight loads, next40 ticks")
}
