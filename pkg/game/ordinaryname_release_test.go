package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseOrdinaryUnitNameSurvivesSaveAndColdLoad(t *testing.T) {
	_, original := groundCorpusFile(t, "2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e")
	root := os.Getenv("AGAINROM_ASSETS")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open := func(front *FrontEnd, raw []byte) {
		t.Helper()
		mission, town, err := front.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("restore", town, err)
		}
		app := front.App("ordinary unit name")
		app.Layout(1024, 768)
		if err := app.OpenMission(mission); err != nil {
			t.Fatal("open mission", err)
		}
	}
	open(f, original)
	get := func(front *FrontEnd, id sim.EntityID) ui.MapEntity {
		t.Helper()
		for _, draw := range front.live.entityDraws() {
			if draw.ID == uint32(id) {
				return draw
			}
		}
		t.Fatalf("entity %d has no draw", id)
		return ui.MapEntity{}
	}

	const ordinary = sim.EntityID(29)
	actor, ok := f.live.world.Entity(ordinary)
	if !ok || actor.TypeID != 9 || actor.Class != 9 || actor.Owner != 3 {
		t.Fatalf("original ordinary actor changed: %+v, found=%t", actor, ok)
	}
	wantName := f.live.view.Words().UnitNames[actor.TypeID]
	before := get(f, ordinary)
	if wantName == "" || before.Name != wantName || before.UnitNameIndex != int(actor.TypeID) {
		t.Fatalf("original unit name=%q index=%d; installed type name=%q", before.Name, before.UnitNameIndex, wantName)
	}
	custom := f.live.chars[ordinary]
	originalChar := custom
	custom.Name = "Named ordinary actor"
	f.live.chars[ordinary] = custom
	if got := get(f, ordinary).Name; got != custom.Name {
		t.Fatalf("explicit ordinary name=%q, want %q", got, custom.Name)
	}
	f.live.chars[ordinary] = originalChar

	var hero, sameTypeClass sim.EntityID
	var heroFound, controlFound bool
	for _, id := range f.live.mission.ids {
		if get(f, id).Name != "" {
			hero = id
			heroFound = true
			break
		}
	}
	for _, e := range f.live.world.Entities() {
		if e.ID != ordinary && e.SourceBinding.Class != 0 && e.TypeID == e.Class && !sim.InPersistBand(e.TypeID) && get(f, e.ID).Name != "" {
			sameTypeClass = e.ID
			controlFound = true
			break
		}
	}
	if !heroFound || !controlFound {
		t.Fatalf("missing hero or TypeID==Class control: hero=%d control=%d", hero, sameTypeClass)
	}
	controls := map[sim.EntityID]ui.MapEntity{hero: get(f, hero), sameTypeClass: get(f, sameTypeClass)}

	dir := t.TempDir()
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	prepared, err := seams.Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "unit-name", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("prepare SAVE", err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal("commit SAVE", err)
	}
	written, err := ReadSaveFile(filepath.Join(dir, "unit-name.sav"))
	if err != nil {
		t.Fatal("read SAVE", err)
	}
	g, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal("cold front end", err)
	}
	g.SetDeterministicFrames(true)
	open(g, written)
	check := func(stage string) {
		t.Helper()
		got := get(g, ordinary)
		if got.Name != wantName || got.UnitNameIndex != int(actor.TypeID) {
			t.Fatalf("%s ordinary name=%q index=%d, want %q/%d", stage, got.Name, got.UnitNameIndex, wantName, actor.TypeID)
		}
		for id, want := range controls {
			got := get(g, id)
			if got.Name != want.Name || got.UnitNameIndex != want.UnitNameIndex {
				t.Fatalf("%s control %d name=%q/%q index=%d/%d", stage, id, got.Name, want.Name, got.UnitNameIndex, want.UnitNameIndex)
			}
		}
	}
	check("cold LOAD")
	f.LiveAdvance(120)
	g.LiveAdvance(120)
	check("next ticks")
}
