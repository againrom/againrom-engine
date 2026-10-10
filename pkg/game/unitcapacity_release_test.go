package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// The original divides a Unit's load by its capacity. A SAV holding a zero
// Unit capacity therefore crashes the original after LOAD. Every SAVE route
// writes the Unit constructor capacity, and LOAD repairs an engine file that
// still holds zero. A Human's capacity and an original's Unit words are not
// touched.
func TestReleaseUnitCapacitySAVRule(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("unit capacity")
	if err := app.OpenMission(f.MissionOpenerWith(10, MissionParty(nil, data.BodyList{}, nil))); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		f.live.tick()
	}
	_, _, written := menuSAVE(t, f, app, OriginalStore{})
	requireUnitCapacityRule(t, "created mission SAVE", written)
	if back, _, _ := loadUnitCapacityFile(t, written); back.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("LOAD of the created mission SAVE changed the World")
	}

	zeroed := alterSAV(t, written, func(doc *sav.DocumentData) bool {
		n := 0
		for i := range doc.Objects {
			if doc.Objects[i].Class == "Unit" && savedStructureSetValue(&doc.Objects[i], "Capacity", 0) == nil &&
				savedStructureSetValue(&doc.Objects[i], "U8E", 0) == nil {
				n++
			}
		}
		return n != 0
	})
	g, gapp, orig := loadUnitCapacityFile(t, zeroed)
	loaded := 0
	for _, e := range g.live.world.Entities() {
		if !e.Humanoid {
			loaded++
			if e.Capacity != data.UnitCapacity() || e.ActorLoad.Present && e.ActorLoad.OwnWeight == 0 && e.Load != e.ActorLoad.CurrentLoad() {
				t.Fatalf("LOAD kept Unit %d capacity %d, own weight %d, load %d", e.ID, e.Capacity, e.ActorLoad.OwnWeight, e.Load)
			}
		}
	}
	if loaded == 0 {
		t.Fatal("LOAD bound no Unit")
	}
	_, _, repaired := menuSAVE(t, g, gapp, orig)
	requireUnitCapacityRule(t, "LOAD and SAVE of a zero-capacity file", repaired)

	source, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_ASSETS"), "game0000.sav"))
	if err != nil {
		t.Fatal(err)
	}
	o, oapp, oorig := loadUnitCapacityFile(t, source)
	_, _, resaved := menuSAVE(t, o, oapp, oorig)
	before, after := unitStatWords(t, source), unitStatWords(t, resaved)
	if len(before) == 0 || len(before) != len(after) {
		t.Fatalf("original Unit records: %d before, %d after", len(before), len(after))
	}
	for key, words := range before {
		if after[key] != words {
			t.Fatalf("original Unit %#x words %v became %v", key, words, after[key])
		}
	}
}

func loadUnitCapacityFile(t *testing.T, raw []byte) (*FrontEnd, *ui.App, OriginalStore) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("unit capacity load")
	orig := OriginalStore{Dir: dir}
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, orig, func() time.Time { return time.Unix(1, 0) })
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0001.sav")
	return f, app, orig
}

func requireUnitCapacityRule(t *testing.T, name string, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	units, humans := 0, 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		body, _ := savedStructureValue(r, "Body")
		capacity, _ := savedStructureValue(r, "Capacity")
		own, _ := savedStructureValue(r, "U8E")
		load, _ := savedStructureValue(r, "U90")
		switch r.Class {
		case "Unit":
			units++
			if capacity != uint32(data.UnitCapacity()) || own != load {
				t.Errorf("%s: Unit capacity %d, own weight %d, load %d", name, capacity, own, load)
			}
		case "Human":
			humans++
			if capacity != 10*body+1 {
				t.Errorf("%s: Human body %d capacity %d", name, body, capacity)
			}
		}
	}
	if units == 0 || humans == 0 {
		t.Fatalf("%s: %d Unit and %d Human records", name, units, humans)
	}
}

func unitStatWords(t *testing.T, raw []byte) map[uint32][sav.UnitStatWords]uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint32][sav.UnitStatWords]uint32{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		key, err := savedStructureValue(r, "Identity")
		if _, dup := out[key]; err != nil || dup {
			t.Fatalf("Unit record identity %d: %v, duplicate %t", key, err, dup)
		}
		var words [sav.UnitStatWords]uint32
		for j, n := range []string{"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"} {
			words[j], _ = savedStructureValue(r, n)
		}
		out[key] = words
	}
	return out
}
