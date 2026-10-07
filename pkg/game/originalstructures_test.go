package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// Literal archive serialization, independent of sav.Buildings and the game
// join. Source/runtime/ALM orders are deliberately different. No install bytes.
func structureFixtureSave(rows []sav.Building, cells ...sav.DocumentCellData) []byte {
	b := savedBody(10, nil)
	b[117] = 0 // Player outcome after the five-byte "Hero" CString
	b = append(b, 1)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	classes, next := map[string]uint16{}, uint16(3) // Player class/object = 1/2
	for i, row := range rows {
		if index := classes[row.Class]; index != 0 {
			b = binary.LittleEndian.AppendUint16(b, 0x8000|index)
		} else {
			b = binary.LittleEndian.AppendUint16(b, 0xffff)
			b = binary.LittleEndian.AppendUint16(b, 1)
			b = binary.LittleEndian.AppendUint16(b, uint16(len(row.Class)))
			b = append(b, row.Class...)
			classes[row.Class], next = next, next+1
		}
		next++
		record := make([]byte, 77)
		record[0], record[1], record[2], record[3] = row.Col, row.Row, row.Col, row.Row
		record[4], record[5] = 128, 128
		binary.LittleEndian.PutUint32(record[12:], uint32(99-i))
		binary.LittleEndian.PutUint16(record[17:], uint16(row.Kind))
		binary.LittleEndian.PutUint32(record[19:], row.AuthoredID)
		binary.LittleEndian.PutUint32(record[29:], uint32(700+i))
		record[59] = row.Kind
		binary.LittleEndian.PutUint16(record[60:], row.Health)
		binary.LittleEndian.PutUint16(record[62:], row.MaxHealth)
		record[67], record[68] = row.Width, row.Height
		binary.LittleEndian.PutUint32(record[69:], row.Blocking)
		binary.LittleEndian.PutUint32(record[73:], row.Attach)
		b = append(b, record...)
		switch row.Class {
		case "Tavern":
			b = binary.LittleEndian.AppendUint32(b, row.Tavern9C)
		case "Shop":
			b = binary.LittleEndian.AppendUint32(b, row.Shop70)
		case "Outpost":
			for _, word := range row.OutpostWords {
				b = binary.LittleEndian.AppendUint32(b, word)
			}
			b = binary.LittleEndian.AppendUint16(b, uint16(len(row.OutpostRecords)))
			for _, record := range row.OutpostRecords {
				b = append(b, record[:]...)
			}
		}
	}
	b = append(b, make([]byte, 4+2)...)
	b = binary.LittleEndian.AppendUint16(b, uint16(len(cells)))
	for _, cell := range cells {
		b = binary.LittleEndian.AppendUint16(b, cell.Cell)
		raw := [52]byte{cell.Cost, cell.Static}
		binary.LittleEndian.PutUint32(raw[12:], cell.Building)
		b = append(b, raw[:]...)
	}
	b = append(b, make([]byte, 4+4374+4)...)
	return savedContainer(savedTrailer(b))
}

func TestSavedStructures1114SubclassesRemainCompleteAcrossNativeSave(t *testing.T) {
	f := structureFixtureFront(t, 51, 52, 53)
	rows := structureFixtureRows()
	rows[0].Class, rows[0].Tavern9C = "Tavern", 0x99112233
	rows[1].Class, rows[1].Shop70 = "Shop", 0x33445566
	rows[2].Class = "Outpost"
	rows[2].OutpostWords = [4]uint32{0x84, 0x88, 0x80, 0x8c}
	rows[2].OutpostRecords = [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}}
	ms, report, err := ResumeOriginalSave(f.Archives.Containers, structureFixtureSave(rows), f.Table, f.Difficulty, nil, nil)
	if err != nil || report.Structures.Restored != 3 || report.Structures.Subclasses != 3 {
		t.Fatalf("subclasses %+v %v", report.Structures, err)
	}
	source, cells, present := ms.World.SavedStructures()
	if !present || len(source) != 3 || source[0].Class != sim.SavedShop || source[0].Shop70 != 0x33445566 || source[1].Tavern9C != 0x99112233 || !reflect.DeepEqual(source[2].OutpostRecords, [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}}) {
		t.Fatalf("subclass tails %+v", source)
	}
	form, err := ms.World.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh sim.World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	got, gc, gp := fresh.SavedStructures()
	if !gp || !reflect.DeepEqual(got, source) || !reflect.DeepEqual(gc, cells) || fresh.Hash() != ms.World.Hash() {
		t.Fatal("subclass state silently omitted")
	}
}

func structureFixtureRows() []sav.Building {
	return []sav.Building{
		{Class: "Building", AuthoredID: 52, Col: 8, Row: 6, Kind: 1, Health: 0xffff, MaxHealth: 73, Width: 1, Height: 1, Blocking: 1, Attach: 1},
		{Class: "Building", AuthoredID: 51, Col: 6, Row: 6, Kind: 1, Health: 7, MaxHealth: 31, Width: 1, Height: 1, Blocking: 1, Attach: 1},
		{Class: "Building", AuthoredID: 53, Col: 10, Row: 6, Kind: 1, Health: 0, MaxHealth: 19, Width: 1, Height: 1, Blocking: 1, Attach: 1},
	}
}

func structureFixtureFront(t *testing.T, ids ...uint16) *FrontEnd {
	t.Helper()
	f := missionFrontEnd(t)
	objects := make([]synth.ALMObject, len(ids))
	for i, id := range ids {
		objects[i] = synth.ALMObject{X: uint32(6+2*i) << 8, Y: 6 << 8, Kind: 1, Field12: id}
	}
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Objects: objects})
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: "10.alm", Data: b}, {Path: "npc.reg", Data: synth.NPCReg(nil)}}), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	f.Table = &mapload.Table{Buildings: dbCollection{{}, {name: "Fixture", params: []int32{1, 1, 0, 100, 1, 1}}}}
	f.SetDeterministicFrames(true)
	return f
}

func assertOriginalStructureHealth(t *testing.T, w *sim.World) {
	t.Helper()
	structures := w.Structures()
	if len(structures) != 3 {
		t.Fatalf("structures = %+v", structures)
	}
	for i, pair := range [][2]uint16{{7, 31}, {0xffff, 73}, {0, 19}} {
		if structures[i].Field42 != pair[0] || structures[i].MaxHealth != pair[1] {
			t.Fatalf("structure %d health %d/%d, want %v", i, structures[i].Field42, structures[i].MaxHealth, pair)
		}
	}
}

func TestOriginalStructuresAppLoadAndNativeSaveKeepBothWords(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "empty cell registry"
		if linked {
			name = "explicit aliases and subclasses"
		}
		t.Run(name, func(t *testing.T) {
			f := withCurrentMenuDefinitions(t, structureFixtureFront(t, 51, 52, 53))
			rows := structureFixtureRows()
			if linked {
				rows[0].Class, rows[0].Tavern9C = "Tavern", 0x99112233
				rows[1].Class, rows[1].Shop70 = "Shop", 0x33445566
				rows[2].Class = "Outpost"
				rows[2].OutpostWords = [4]uint32{0x84, 0x88, 0x80, 0x8c}
				rows[2].OutpostRecords = [][8]byte{{1, 2, 3, 4, 5, 6, 7, 8}}
			}
			payload := structureFixtureSave(rows)
			wantCells := []sim.SavedStructureCell{}
			wantLinks := map[uint16]uint32{}
			if linked {
				payload = structureFixtureSave(rows,
					sav.DocumentCellData{Cell: 0x0606, Cost: 9, Static: 5},
					sav.DocumentCellData{Cell: 0x0808, Building: 700, Cost: 17, Static: 11},
					sav.DocumentCellData{Cell: 0x0809, Building: 700, Cost: 19, Static: 13})
				wantCells = []sim.SavedStructureCell{
					{Cell: 0x0606, BaselineCost: 9, BaselineStatic: 5},
					{Cell: 0x0808, ID: 1, HasStructure: true, BaselineCost: 17, BaselineStatic: 11},
					{Cell: 0x0809, ID: 1, HasStructure: true, BaselineCost: 19, BaselineStatic: 13},
				}
				wantLinks = map[uint16]uint32{0x0808: 700, 0x0809: 700}
			}
			app := f.App("current structures")
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game9999.sav")
			assertOriginalStructureHealth(t, f.live.world)
			if f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatal("LOAD advanced before imported health was visible")
			}
			sources, cells, present := f.live.world.SavedStructures()
			if !present || !reflect.DeepEqual(cells, wantCells) {
				t.Fatal("fixture did not establish the explicit current cell registry", cells)
			}
			seen := map[string]bool{}
			for cycle := 0; cycle < 2; cycle++ {
				prior := f.live
				hash := prior.world.Hash()
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessGameMenuAction("save"); err != nil {
					t.Fatal(err)
				}
				entries, err := store.List()
				if err != nil || len(entries) != cycle+1 {
					t.Fatalf("SAVE = %+v %v: %s", entries, err, app.HeadlessMessage())
				}
				var latest string
				for _, entry := range entries {
					if !seen[entry.Name] {
						latest = entry.Name
						seen[entry.Name] = true
					}
				}
				if filepath.Ext(latest) != ".sav" || hash != prior.world.Hash() {
					t.Fatal("SAVE changed current state or used another format")
				}
				raw, err := os.ReadFile(filepath.Join(store.Dir, latest))
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				links := map[uint16]uint32{}
				for _, cell := range doc.World.Cells {
					if cell.Building != 0 {
						links[cell.Cell] = cell.Building
					}
				}
				if !reflect.DeepEqual(links, wantLinks) {
					t.Fatal("SAVE reconstructed geometric structure links", links, wantLinks)
				}
				cold := withCurrentMenuDefinitions(t, structureFixtureFront(t, 51, 52, 53))
				coldApp := cold.App("cold current structures")
				cs, cl, cr := cold.SaveSeams(store, OriginalStore{}, nil)
				coldApp.SetSaveSeams(cs, cl, cr)
				groundAppLoad(t, coldApp, cl, latest)
				assertOriginalStructureHealth(t, cold.live.world)
				gotSource, gotCells, gotPresent := cold.live.world.SavedStructures()
				if !gotPresent || !reflect.DeepEqual(sources, gotSource) || !reflect.DeepEqual(cells, gotCells) || cold.live.world.Hash() != hash {
					currentMenuWorldDiagnostics(t, prior.world, cold.live.world)
					t.Fatal("cold SAVE/LOAD changed current structures", cycle)
				}
				for range 32 {
					prior.tick()
					cold.live.tick()
					if prior.world.Hash() != cold.live.world.Hash() {
						t.Fatal("current structure continuation differs")
					}
				}
				f, app = cold, coldApp
			}
			ms, report, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, mapload.DifficultyNormal, nil, nil)
			if err != nil || report.Structures.Restored != 3 {
				t.Fatalf("diagnostic restore: %+v %v", report.Structures, err)
			}
			assertOriginalStructureHealth(t, ms.World)
		})
	}
}

func TestOriginalStructuresUniqueJoinAndCountedLimits(t *testing.T) {
	f := structureFixtureFront(t, 51, 52, 53)
	ms, err := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	fixture := structureFixtureRows()[:2]
	fixture[0].Col = 15
	fixture[1].AuthoredID = 0
	sf, err := sav.Open(structureFixtureSave(fixture))
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := sf.Buildings()
	if err != nil {
		t.Fatal(err)
	}
	var report OriginalSaveResume
	if err := applyOriginalStructures(ms, rows, true, f.Table, &report, sf); err != nil {
		t.Fatal(err)
	}
	want := originalStructureCounts{Restored: 2, Unbound: 1, Absent: 2}
	if report.Structures != want || len(ms.World.Structures()) != 2 || ms.World.Structures()[0].Col != 15 {
		t.Fatalf("roster %+v want %+v", report.Structures, want)
	}
	before := ms.World.Hash()
	if err := applyOriginalStructures(ms, []sav.Building{rows[0], rows[0]}, true, f.Table, &report); err == nil || ms.World.Hash() != before {
		t.Fatalf("duplicate source not atomic: %v", err)
	}
	ms.Map.Objects[0].Field12 = 52
	if err := applyOriginalStructures(ms, []sav.Building{rows[0]}, true, f.Table, &report); err == nil || ms.World.Hash() != before {
		t.Fatalf("duplicate target not atomic: %v", err)
	}
	if err := applyOriginalStructures(ms, nil, true, f.Table, &report); err != nil || len(ms.World.Structures()) != 0 || report.Structures.Absent != 3 {
		t.Fatalf("saved empty roster retained ALM ghosts: %+v %v", report.Structures, err)
	}
}

func TestOriginalStructuresRejectedAppLoadKeepsLiveSession(t *testing.T) {
	for _, targetCollision := range []bool{false, true} {
		f := structureFixtureFront(t, 51, 52, 53)
		rows := structureFixtureRows()
		if targetCollision {
			f = structureFixtureFront(t, 51, 51, 53)
		} else {
			rows = append(rows, rows[1])
		}
		app := f.App("1098 refusal")
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		old := f.live
		before, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		opener, _, err := f.RestoreOriginal(structureFixtureSave(rows))
		if err == nil {
			err = app.OpenMission(opener)
		}
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("ambiguous load accepted: %v", err)
		}
		after, _, err := f.Snapshot(true)
		if err != nil || f.live != old || !reflect.DeepEqual(before, after) {
			t.Fatalf("refused load changed live session: %v", err)
		}
	}
}
