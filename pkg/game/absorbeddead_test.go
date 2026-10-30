package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// An original SAV: a late Human and a Unit at stage 2, health -10001, runtime 122.
func absorbedDeadPayload() []byte {
	runtime := uint32(122)
	late := &poolFixtureActor{mapID: 91, cell: 0x0807, hp: 0xff13, maxHP: 30, stage: 4, human: true}
	absorbed := &poolFixtureActor{mapID: 92, cell: 0x0a09, hp: 0xd8ef, maxHP: 30, stage: 2, runtime: &runtime}
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {}}, []*poolFixtureActor{late, late, absorbed}))
}

func TestOriginalAbsorbedDeadBodyLoadsAndSavesTerminal(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("absorbed original dead")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9998.sav"), absorbedDeadPayload(), 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9998.sav")
	w := f.live.world
	var identity uint32
	for _, d := range w.OriginalDeadActors() {
		if d.Source.MapUnitID != 92 {
			continue
		}
		identity = d.Source.Identity
		for _, s := range []sim.DeadActorState{d.Current, d.Source.State} {
			if s.Stage != 5 || s.HP != -10001 || s.RuntimeID != 0 || s.Cell != 0x0a09 || s.Timer != 0 {
				t.Fatalf("absorbed body held as %+v, want the terminal tuple", s)
			}
		}
	}
	if identity == 0 {
		t.Fatal("absorbed dead record was not retained", w.OriginalDeadActors())
	}
	for _, e := range w.Entities() {
		if e.MapUnitID == 92 {
			t.Fatalf("absorbed body kept an entity: %+v", e)
		}
	}
	if e := poolEntity(t, w, 91); e.Alive() || e.HP != -237 || e.Decay != 4 {
		t.Fatalf("late corpse changed: %+v", e)
	}
	for range 33 {
		f.live.tick()
	}
	original := w.OriginalDeadActors()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("ordinary SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	written := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if key, err := savedStructureValue(r, "Identity"); err != nil || key != identity {
			continue
		}
		written++
		stage, e1 := savedStructureValue(r, "Stage")
		hp, e2 := savedStructureValue(r, "Health")
		runtime, e3 := savedStructureValue(r, "RuntimeID")
		if e1 != nil || e2 != nil || e3 != nil || stage != 5 || int16(hp) != -10001 || runtime != 0 {
			t.Fatalf("SAVE wrote Stage %d Health %d RuntimeID %d (%v %v %v), want 5/-10001/0", stage, int16(hp), runtime, e1, e2, e3)
		}
	}
	if written != 1 {
		t.Fatalf("SAVE wrote %d records of the absorbed body", written)
	}
	groundAppLoad(t, app, list, entries[0].Name)
	if adoptLoadedAllocator(t, w, f.live.world).Hash() != f.live.world.Hash() || !reflect.DeepEqual(original, f.live.world.OriginalDeadActors()) {
		currentMenuWorldDiagnostics(t, w, f.live.world)
		t.Fatal("our SAV lost the terminal dead state")
	}
}

func TestCurrentAbsorbedTerminalRowLoadsAndSavesTerminal(t *testing.T) {
	f, snapshot, base := partialCurrentGraph(t)
	keeper := sim.Entity{ID: 41, X: 14, Y: 15, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, TypeID: 1, TokenSize: 1}
	w, err := sim.NewWorld(9, base.Bounds(), sim.ModeCanonical, nil, []sim.Entity{keeper})
	if err != nil {
		t.Fatal(err)
	}
	held := sim.CurrentTerminalActor{ID: 7, Cell: 0x0f0f, HP: -10001, Stage: 5, MaxHP: 29, MaxHPPresent: true}
	if err := w.ImportCurrentTerminalActors([]sim.CurrentTerminalActor{held}); err != nil {
		t.Fatal(err)
	}
	snapshot.SavedDocument = nil
	snapshot.CurrentPartyIDs, snapshot.CurrentRoster = nil, nil
	snapshot.Residue.VisualIdentities, snapshot.Residue.VisualNext = nil, 0
	if snapshot.World, err = w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "absorbed row")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.Values[held.ID].CurrentTerminal == nil {
		t.Fatal("SAV lacks the terminal row", err)
	}
	a.Values[held.ID].CurrentTerminal.Stage = 2
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
		t.Fatal(err)
	}
	older, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := coldCurrentScript(t, f, older)
	if got := cold.live.world.CurrentTerminalActors(); !reflect.DeepEqual(got, []sim.CurrentTerminalActor{held}) {
		t.Fatalf("stage-2 row held as %+v, want %+v", got, held)
	}
	consumedCorpseTuple(t, bindingOrderSave(t, cold, "absorbed row resave"), held, false)
}
