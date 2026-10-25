package game

import (
	"againrom/internal/firstgame"
	"againrom/pkg/sim"
	"testing"

	"againrom/pkg/formats/sav"
)

func lootSnapshot1172(t *testing.T) (Snapshot, []byte, int) {
	t.Helper()
	f, _, _ := returnPlayed1169(t, false)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.ExportCurrentSave(s, "current pack controls")
	if err != nil {
		t.Fatal(err)
	}
	for i, binding := range s.OriginalCity.Bindings {
		if binding.Returned != nil && binding.Returned.Holdings != nil && len(binding.Returned.Holdings.Inventory) != 0 {
			return s, b, i
		}
	}
	t.Fatal("fixture lacks current acquired pack")
	return Snapshot{}, nil, 0
}

func TestReleaseImportedLoot1172SingleCauseAdmission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	source, _, at := lootSnapshot1172(t)
	for name, edit := range map[string]func(*SnapshotCityReturn){
		"omitted item": func(r *SnapshotCityReturn) { r.Holdings.Inventory = nil },
		"foreign item": func(r *SnapshotCityReturn) { r.Holdings.Inventory[0] = 65535 },
		"stale count": func(r *SnapshotCityReturn) {
			record := &r.Holdings.Objects[r.Holdings.Inventory[0]-1]
			for i := range record.Values {
				if record.Values[i].Name == "F42" {
					record.Values[i].Value++
					return
				}
			}
			t.Fatal("control lacks count")
		},
		"stale insertion index":          func(r *SnapshotCityReturn) { r.Holdings.InsertIndex++ },
		"stale pack load":                func(r *SnapshotCityReturn) { r.Holdings.Accumulator++ },
		"missing persisted return graph": func(r *SnapshotCityReturn) { r.Holdings = nil },
	} {
		t.Run(name, func(t *testing.T) {
			state, err := originalCityFromSnapshot(source)
			if err != nil {
				t.Fatal(err)
			}
			s := source
			s.OriginalCity = state.snapshot()

			edit(s.OriginalCity.Bindings[at].Returned)
			f := releaseFront(t)
			before := f.originalCity
			if _, err := originalCityFromSnapshot(s); err == nil || f.originalCity != before {
				t.Fatal("loss was admitted or changed the live front end", err)
			}
		})
	}
}

// A held current Item whose Reference names no object is written with its
// current holder's Player key. City topology IDs and native item handles are
// separate: follow the topology binding to the ordinary record.
func TestReleaseImportedLootUnclosedOwnerReferenceUsesCurrentHolder(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, _, _ := returnPlayed1169(t, false)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.CityObjects == nil {
		t.Fatal("current city has no object graph")
	}
	var itemID sim.SavedObjectID
	for _, roots := range s.CityObjects.Roots {
		if len(roots.Pack) != 0 {
			itemID = roots.Pack[0]
			break
		}
	}
	record, ok := s.CityObjects.ItemRecords[itemID]
	if !ok || itemID == 0 {
		t.Fatal("current pack has no retained item record")
	}
	code := uint32(record.Value.Code)
	record.Token.Reference = 0xdeadbeef
	s.CityObjects.ItemRecords[itemID] = record
	check := func(raw []byte) {
		t.Helper()
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Players) != 1 {
			t.Fatal("current fixture must have one holder Player", doc.Players)
		}
		want, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "This")
		if err != nil || want == 0 {
			t.Fatal("current holder has no owner key", err)
		}
		actions, err := readCurrentActions(&doc)
		if err != nil || actions == nil {
			t.Fatal("current SAV has no ownership", err)
		}
		found := 0
		for _, binding := range actions.Objects {
			if binding.ID != itemID {
				continue
			}
			if binding.Object == 0 || int(binding.Object) > len(doc.Objects) {
				t.Fatal("item has no current record")
			}
			policies := 0
			for _, policy := range actions.Ownership {
				if policy.Kind == 1 && policy.Object == binding.Object {
					policies++
				}
			}
			if policies != 1 {
				t.Fatal("topology item has no unique current ownership policy", itemID, binding.Object, policies)
			}
			item := &doc.Objects[binding.Object-1]
			got, err := savedStructureValue(item, "Reference")
			if err != nil || got != want || got == 0xdeadbeef {
				t.Fatalf("current item owner=%#x want holder %#x: %v", got, want, err)
			}
			if gotCode, err := savedStructureValue(item, "F40"); err != nil || gotCode != code {
				t.Fatal("owner repair changed item code", gotCode, code, err)
			}
			found++
		}
		if found != 1 {
			t.Fatal("current item ownership count", found)
		}
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	check(raw)
	cold := releaseFront(t)
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold current owner LOAD", town, err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("ordinary current owner SAVE", name, err)
	}
	raw, err = store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	check(raw)
}
