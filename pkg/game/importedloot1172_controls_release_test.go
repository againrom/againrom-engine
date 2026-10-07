package game

import (
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
	b, err := EncodeSave(s, "current pack controls")
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
	_, raw, at := lootSnapshot1172(t)
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
			s, _, err := DecodeSave(raw)
			if err != nil {
				t.Fatal(err)
			}
			edit(s.OriginalCity.Bindings[at].Returned)
			if _, _, err := DecodeSave(city1095CorruptEnvelope(t, s)); err == nil {
				t.Fatal("loss entered a current AGS envelope")
			}
			f := releaseFront(t)
			before := f.originalCity
			if _, _, err := f.Restore(s); err == nil || f.originalCity != before {
				t.Fatal("loss was admitted or changed the live front end", err)
			}
		})
	}
}

// TestReleaseImportedLootUnclosedOwnerReferenceWritesZero gives a current
// pack item an owner Reference that names no object. The town SAVE and the
// converter both write SAV with that Reference as 0, the value the original's
// Token load gives an absent key (SAV-PTRMAP-035).
func TestReleaseImportedLootUnclosedOwnerReferenceWritesZero(t *testing.T) {
	s, _, at := lootSnapshot1172(t)
	g := s.OriginalCity.Bindings[at].Returned.Holdings
	r := &g.Objects[g.Inventory[0]-1]
	found := false
	for i := range r.Values {
		if r.Values[i].Name == "Reference" {
			r.Values[i].Value, found = 0xdeadbeef, true
		}
	}
	code, err := savedStructureValue(r, "F40")
	if !found || err != nil || sav.ValidateCityItemGraph(g) != nil {
		t.Fatal("control must be a well-formed graph with an unclosed owner reference", err)
	}
	native, err := EncodeSave(s, "unclosed owner reference")
	if err != nil {
		t.Fatal(err)
	}
	cleared := func(raw []byte) {
		t.Helper()
		d, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		zero := false
		for i := range d.Objects {
			o := &d.Objects[i]
			reference, err := savedStructureValue(o, "Reference")
			if err != nil {
				continue
			}
			if reference == 0xdeadbeef {
				t.Fatal("unclosed owner reference written verbatim")
			}
			if c, err := savedStructureValue(o, "F40"); err == nil && o.Class == r.Class && c == code && reference == 0 {
				zero = true
			}
		}
		if !zero {
			t.Fatalf("no written %s %#x carries a zero owner reference", r.Class, code)
		}
	}
	f := releaseFront(t)
	if _, town, err := f.Restore(s); err != nil || !town {
		t.Fatal("native state must remain loadable", town, err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("ordinary SAVE did not write SAV", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	cleared(raw)
	output, _, err := releaseFront(t).ConvertSave(native, "sav", nil)
	if err != nil || output == nil {
		t.Fatal("converter refused the unclosed owner reference", err)
	}
	cleared(output)
}
