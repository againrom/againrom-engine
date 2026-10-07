package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func deadFixturePayload(timer int8, duplicate bool) []byte {
	zero := uint32(0)
	late := &poolFixtureActor{mapID: 91, cell: 0x0807, hp: 0xff13, maxHP: 30, stage: 4, timer: timer, human: true}
	terminal := &poolFixtureActor{mapID: 92, cell: 0x0a09, hp: 0xd8df, maxHP: 30, stage: 5, runtime: &zero}
	if duplicate {
		terminal.mapID = 91
	}
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{}, {}}, []*poolFixtureActor{late, late, terminal}))
}

func TestOriginalDead1100ExactSharedArchiveProjection(t *testing.T) {
	payload := deadFixturePayload(-128, false)
	f, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	dead, err := f.DeadActors()
	if err != nil {
		t.Fatal(err)
	}
	if len(dead) != 2 || dead[0].Class != "Human" || dead[0].MapUnitID != 91 || dead[0].HP != -237 || dead[0].Stage != 4 || dead[0].Timer != -128 || dead[0].Cell != 0x0807 || dead[0].RuntimeID == 0 || dead[1].Class != "Unit" || dead[1].HP != -10017 || dead[1].RuntimeID != 0 || dead[1].ArchiveIndex <= dead[0].ArchiveIndex {
		t.Fatalf("wrong exact projection: %+v", dead)
	}
	if _, present, err := f.GroundSacks(); err != nil || !present {
		t.Fatalf("shared archive no longer reaches ground: %t %v", present, err)
	}
	dead[0].HP = 0
	again, _ := f.DeadActors()
	if again[0].HP != -237 {
		t.Fatal("projection aliases archive")
	}
	for _, end := range []int{0, 75, again[0].Off, again[1].Off + 10} {
		if _, err := (&sav.File{Body: f.Body[:end], Head: f.Head}).DeadActors(); err == nil {
			t.Fatalf("accepted truncated list %d", end)
		}
	}
	// This projection-only case refers back from the dead manager into the
	// already parsed Player graph. Resetting the archive index cannot read it.
	shared := &poolFixtureActor{mapID: 93, cell: 0x0807, hp: 0xff13, maxHP: 30, stage: 3, human: true}
	body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{{shared}}}}, []*poolFixtureActor{shared, shared})
	sf := &sav.File{Body: body, Head: sav.Head{End: 75, PlayerCount: 2}}
	sharedDead, err := sf.DeadActors()
	if err != nil || len(sharedDead) != 1 || sharedDead[0].MapUnitID != 93 || sharedDead[0].HP != -237 || sharedDead[0].Stage != 3 {
		t.Fatalf("shared archive reference lost: %+v %v", sharedDead, err)
	}
	if _, present, err := sf.GroundSacks(); err != nil || !present {
		t.Fatalf("shared archive suffix failed: %t %v", present, err)
	}
}

func TestOriginalDead1100AppLoadSaveContinuationAndRollback(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("1100-original-dead")
	dir := t.TempDir()
	path := filepath.Join(dir, "game9999.sav")
	payload := deadFixturePayload(0, false)
	if err := os.WriteFile(path, payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	w := f.live.world
	if w.Tick() != rawSavedSubTick1112(t, payload) || len(w.OriginalDeadActors()) != 2 {
		t.Fatal("dead list not applied before publication")
	}
	e := poolEntity(t, w, 91)
	if e.Alive() || e.HP != -237 || e.Decay != 4 || e.X != 7 || e.Y != 8 {
		t.Fatalf("corpse resurrected: %+v", e)
	}
	for _, entity := range w.Entities() {
		if entity.MapUnitID == 92 {
			t.Fatal("terminal actor respawned")
		}
	}
	// Both semantic validation and ambiguous authored binding leave the old
	// frontend/world candidate untouched, including SAVE ownership.
	for _, bad := range [][]byte{deadFixturePayload(-128, false), deadFixturePayload(0, true)} {
		before := w.Hash()
		driver := f.live
		if opener, _, err := f.RestoreOriginal(bad); err == nil || opener != nil {
			t.Fatal("unsupported dead load accepted")
		}
		if f.live != driver || f.live.world != w || w.Hash() != before {
			t.Fatal("rejected dead load published partial candidate")
		}
	}
	for range 33 {
		f.live.tick()
	}
	original := w.OriginalDeadActors()
	if original[0].Source.State.HP != original[0].Current.HP || original[0].Current.HP >= -237 {
		t.Fatal("death mutation did not publish the current health mirror")
	}
	for cycle := 0; cycle < 2; cycle++ {
		w = f.live.world
		hash, prior := w.Hash(), f.live
		original = w.OriginalDeadActors()
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != cycle+1 || filepath.Ext(entries[0].Name) != ".sav" {
			t.Fatalf("ordinary SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
		}
		groundAppLoad(t, app, list, entries[0].Name)
		if f.live.world.Hash() != hash || !reflect.DeepEqual(original, f.live.world.OriginalDeadActors()) {
			currentMenuWorldDiagnostics(t, w, f.live.world)
			t.Fatalf("cycle%d ordinary SAV lost exact dead state", cycle)
		}
		for tick := 0; tick < 64; tick++ {
			prior.tick()
			f.live.tick()
			if w.Hash() != f.live.world.Hash() {
				t.Fatalf("cycle%d successor%d diverged", cycle, tick)
			}
		}
	}
	ms, report, err := ResumeOriginalSave(f.Archives.Containers, deadFixturePayload(0, false), f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil || report.CorpsesRestored != 1 || report.TerminalRestored != 1 || len(ms.World.OriginalDeadActors()) != 2 {
		t.Fatalf("diagnostic door: %+v %v", report, err)
	}
}

func TestOriginalDead1100AmbiguousMapFailsBeforeAnyImport(t *testing.T) {
	f := poolFixtureFront(t, 91, 91, 92)
	payload := deadFixturePayload(0, false)
	if _, _, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, f.Difficulty, nil, f.Bodies); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("duplicate ALM accepted: %v", err)
	}
	// Pure projection takes the archive list, not the heuristic actor scan.
	sf, _ := sav.Open(payload)
	sf.Actors = nil
	got, err := sf.DeadActors()
	if err != nil || len(got) != 2 {
		t.Fatalf("projection relies on head scan: %+v %v", got, err)
	}
}

func assertDeadWorld1100(t *testing.T, w *sim.World, want []struct {
	mapID uint16
	stage uint8
	hp    int16
	x, y  int32
}) {
	t.Helper()
	records := w.OriginalDeadActors()
	if len(records) != len(want) {
		t.Fatalf("dead population=%d want=%d", len(records), len(want))
	}
	for i, v := range want {
		r := records[i]
		s := r.Current
		if r.Source.MapUnitID != v.mapID || s.Stage != v.stage || s.HP != v.hp || s.Cell != uint16(v.y)<<8|uint16(v.x) || s.FineX != 128 || s.FineY != 128 || s.Timer != 0 {
			t.Fatalf("dead[%d]=%+v want=%+v", i, r, v)
		}
		found := false
		for _, e := range w.Entities() {
			if e.MapUnitID != v.mapID {
				continue
			}
			found = true
			if v.stage == 5 || e.Alive() || e.HP != int32(v.hp) || uint8(e.Decay) != v.stage || e.Dwell != 0 || e.OffMap || e.X != v.x || e.Y != v.y {
				t.Fatalf("bad corpse entity: %+v", e)
			}
			for _, stock := range w.Stock() {
				if stock.ID == e.ID && (len(stock.Items) != 0 || len(stock.ItemInstances) != 0 || stock.Equipped != [sim.EquipSlots]uint16{}) {
					t.Fatal("fresh map corpse loot retained")
				}
			}
		}
		if found != (v.stage < 5) {
			t.Fatalf("dead mapID=%d entity present=%t", v.mapID, found)
		}
	}
}
