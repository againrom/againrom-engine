package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Independent literal archive fixture: no original blob, cell projection or
// production writer provides the expected bytes. Returns the last record end
// for a late-table truncation control.
func cellFixture1106(rows [][54]byte) ([]byte, int) {
	b := savedBody(10, nil)
	b[117] = 0
	b = append(b, 1) // world selector
	b = append(b, make([]byte, 8)...)
	b = binary.LittleEndian.AppendUint16(b, 0) // blocks
	b = binary.LittleEndian.AppendUint16(b, uint16(len(rows)))
	for _, row := range rows {
		b = append(b, row[:]...)
	}
	end := len(b)
	b = append(b, make([]byte, 4+4374+4)...)
	return savedTrailer(b), end
}

func cellRows1106() [][54]byte {
	return [][54]byte{
		{0: 21, 1: 63, 2: 0xa9, 45: 0xeb, 46: 13, 47: 1, 48: 19, 49: 61, 50: 21, 51: 63, 52: 0xcc, 53: 0xdd},
		{0: 255, 1: 254, 46: 26, 47: 8, 48: 9, 49: 10, 50: 251, 51: 252},
		{0: 21, 1: 63, 46: 0, 47: 7, 48: 31, 49: 32, 50: 33, 51: 34},
	}
}

func cellWant1106() []sim.CellTail {
	return []sim.CellTail{{X: 21, Y: 63, Bytes: [6]byte{0, 7, 31, 32, 33, 34}},
		{X: 255, Y: 254, Bytes: [6]byte{26, 8, 9, 10, 251, 252}}}
}

func TestOriginalCellTriggers1106BothLoadDoorsAndAppNativeSave(t *testing.T) {
	body, _ := cellFixture1106(cellRows1106())
	payload := savedContainer(body)
	for _, inMission := range []bool{false, true} {
		t.Run(map[bool]string{false: "menu", true: "mission"}[inMission], func(t *testing.T) {
			f := currentPoolFixtureFront(t)
			app := f.App("1106 cells")
			if inMission {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game9999.sav")
			w := f.live.world
			if !reflect.DeepEqual(w.CellTails(), cellWant1106()) || w.Tick() != rawSavedSubTick1112(t, payload) || len(w.ScriptCasts()) != 0 {
				t.Fatalf("LOAD tails %+v tick %d casts %+v", w.CellTails(), w.Tick(), w.ScriptCasts())
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
				t.Fatalf("SAVE %+v %v: %s", entries, err, app.HeadlessMessage())
			}
			fresh := currentPoolFixtureFront(t)
			freshApp := fresh.App("1106 fresh")
			fs, fl, fd := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, fd)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			if fresh.live.world.Hash() != w.Hash() || !reflect.DeepEqual(fresh.live.world.CellTails(), cellWant1106()) {
				t.Fatal("native SAVE/fresh LOAD changed overlay")
			}
			for range 3 {
				a, b := sim.StepReported(w, nil), sim.StepReported(fresh.live.world, nil)
				if !reflect.DeepEqual(a, b) || w.Hash() != fresh.live.world.Hash() {
					t.Fatal("native continuation changed")
				}
			}
			ms, r, err := ResumeOriginalSave(f.Archives.Containers, payload, nil, mapload.DifficultyNormal, nil, nil)
			if err != nil || ms == nil || !r.CellTriggersApplied || r.CellTriggerRecords != 3 {
				t.Fatalf("diagnostic LOAD %+v %v", r, err)
			}
			if !reflect.DeepEqual(ms.World.CellTails(), cellWant1106()) || ms.World.Tick() != rawSavedSubTick1112(t, payload) || len(ms.World.ScriptCasts()) != 0 {
				t.Fatal("diagnostic LOAD changed overlay/advanced")
			}
		})
	}
}

func TestOriginalCellTriggers1106LateMalformedLoadKeepsSnapshot(t *testing.T) {
	f := poolFixtureFront(t)
	app := f.App("1106 refusal")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := f.live.world.ImportOriginalCellTails(cellWant1106()); err != nil {
		t.Fatal(err)
	}
	body, end := cellFixture1106(cellRows1106())
	valid := savedContainer(body)
	bad := savedContainer(body[:end-1]) // two valid rows, last byte missing
	dir := t.TempDir()
	path := filepath.Join(dir, "game9999.sav")
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	rows := list()
	if len(rows) != 1 {
		t.Fatalf("rows %+v", rows)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	oldLive, oldTown, oldHash := f.live, f.Town, f.live.world.Hash()
	// A listed file can change before activation. Exercise the actual App
	// refusal route rather than relying on the list hiding malformed input.
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(rows[0].Label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
		t.Fatal("malformed LOAD was accepted")
	}
	after, _, err := f.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) || f.live != oldLive || f.Town != oldTown || f.live.world.Hash() != oldHash {
		t.Fatal("malformed LOAD changed old session")
	}
	if ms, _, err := ResumeOriginalSave(f.Archives.Containers, bad, nil, mapload.DifficultyNormal, nil, nil); err == nil || ms != nil {
		t.Fatal("diagnostic LOAD accepted short last cell")
	}
	sf, err := sav.Open(valid)
	if err != nil {
		t.Fatal(err)
	}
	sf.Body = sf.Body[:end-1]
	if tails, present, err := originalCellTails(sf); err == nil || !present || tails != nil {
		t.Fatal("game projection returned partial table")
	}
}
