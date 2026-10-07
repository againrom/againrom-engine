package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The literal1114 fixture has moved/source-only Buildings, repeated roots and
// overwritten cell keys. Add the independently spelled1115 container tail so
// this witness cannot pass through the partial-document Unavailable arm.
func composedDocumentFront1115(t *testing.T, generated ...bool) *FrontEnd {
	t.Helper()
	f := structureFront1114(t, generated...)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

func checkComposedDocument1115(t *testing.T, f *FrontEnd) Snapshot {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil || s.SavedDocument == nil || s.SavedDocument.Document == nil || s.SavedDocument.Unavailable != "" {
		t.Fatalf("complete document absent: %v", err)
	}
	var current sim.World
	if len(s.World) == 0 || current.UnmarshalBinary(s.World) != nil || len(current.Entities()) == 0 {
		t.Fatal("composed actor binding/native form")
	}
	groups, orders, present := f.live.world.SavedGroups()
	if !present || len(groups) < 1 || len(orders) != 1 {
		t.Fatal("current Groups absent", groups, orders)
	}
	meta, cells, present := f.live.world.SavedStructures()
	_, actorCells, _, _ := f.live.world.SavedActorMotions()
	var sourceCells []sim.SavedStructureCell
	for _, cell := range cells {
		switch cell.Cell {
		case 0x0c0c, 0x0c0d, 0x0c0e, 0xffff:
			sourceCells = append(sourceCells, cell)
		default:
			matched := false
			for _, actorCell := range actorCells {
				if actorCell.Cell == cell.Cell {
					matched = !cell.HasStructure && cell.ID == 0 && cell.BaselineCost == actorCell.Payload[0] && cell.BaselineStatic == actorCell.Payload[1]
					break
				}
			}
			if !matched {
				t.Fatal("additional saved structure cell has no matching actor-cell baseline", cell)
			}
		}
	}
	cells = sourceCells
	if !present || len(meta) != 2 || len(cells) != 4 || meta[0].ArchiveIndex != 6 || meta[1].ArchiveIndex != 7 || meta[1].HasAuthored {
		t.Fatal("saved structure identities absent", meta, cells)
	}
	if cells[0].Cell != 0x0c0c || cells[0].HasStructure || cells[1].Cell != 0x0c0d || !cells[1].HasStructure || cells[1].ID != 4 || cells[2].ID != 0 || !cells[2].HasStructure || cells[3].Cell != 0xffff {
		t.Fatal("current explicit saved-cell links changed", cells)
	}
	st := f.live.world.Structures()
	if len(st) != 2 || st[0].Col != 12 || st[0].Row != 12 || st[1].ID != 4 || st[1].Col != 20 || st[1].Row != 14 || st[1].Field42 != 0xffff {
		t.Fatal("current saved roster changed", st)
	}
	buildings := composedDocumentBuildings1115(t, s)
	for i, current := range st {
		if buildings[i].Health != current.Field42 || buildings[i].MaxHealth != current.MaxHealth || buildings[i].Col != byte(current.Col) || buildings[i].Row != byte(current.Row) || buildings[i].Identity != meta[i].SourceKey {
			t.Fatal("whole SAV document did not project current structure", i, buildings[i], current)
		}
	}
	return s
}

func composedDocumentBuildings1115(t *testing.T, snapshot Snapshot) []sav.Building {
	t.Helper()
	encoded, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	buildings, present, err := f.Buildings()
	if err != nil || !present || len(buildings) != 2 {
		t.Fatal("whole SAV document structure roster", present, buildings, err)
	}
	return buildings
}

func TestDocument1115ComposedGroupsStructuresMenuSaveFreshProcess(t *testing.T) {
	if path := os.Getenv("AGAINROM_COMPOSED1115_NATIVE"); path != "" {
		f := composedDocumentFront1115(t, os.Getenv("AGAINROM_COMPOSED_GENERATED") == "true")
		app := f.App("fresh composed SAV checkpoint")
		store := SaveStore{Dir: filepath.Dir(path)}
		save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		name := filepath.Base(path)
		if IsOriginal(name) {
			name = localOriginalSaveToken(name)
		}
		groundAppLoad(t, app, list, name)
		checkComposedDocument1115(t, f)
		retained := f.live.mission.state.savedDocument
		if retained == nil || retained.Document == nil {
			t.Fatal("cold composed input graph absent")
		}
		doc, err := canonicalDocument1115(t, *retained.Document)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(doc)) != os.Getenv("AGAINROM_COMPOSED1115_DOCUMENT") || fmt.Sprintf("%x", f.live.world.Hash()) != os.Getenv("AGAINROM_COMPOSED1115_WORLD") {
			t.Fatal("cold complete graph or current World differs", err)
		}
		if hp := f.live.world.Structures()[0].Field42; hp == 0 || hp >= 90 {
			t.Fatal("fresh LOAD lost the actual structure hit", hp)
		}
		for range 20 {
			f.live.tick()
		}
		if fmt.Sprintf("%x", f.live.world.Hash()) != os.Getenv("AGAINROM_COMPOSED1115_NEXT") {
			t.Fatal("fresh current Group/action continuation differs")
		}
		checkComposedDocument1115(t, f)
		t.Log("composed complete document, Groups, Structures and next20 driver ticks PASS")
		return
	}
	for _, fromMap := range []bool{false, true} {
		t.Run(fmt.Sprintf("from-map-%t", fromMap), func(t *testing.T) {
			f := composedDocumentFront1115(t, fromMap)
			raw := completeDocumentTail1115(t, f, structureSave1114(t, 0, false))
			ms, report, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, nil)
			if err != nil || report.GroupsRestored != 1 || ms.savedDocument == nil || ms.savedDocument.Document == nil {
				t.Fatal("low-level composed LOAD", err)
			}
			assertStructures1114(t, ms.World)
			app := f.App("composed source SAV checkpoint")
			app.Layout(1024, 768)
			if fromMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			originals, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
			path := filepath.Join(originals, "game1115.sav")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1115.sav")
			clear(raw)
			// Remove only this test-created synthetic source, never an install.
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			before := checkComposedDocument1115(t, f)
			assertStructures1114(t, f.live.world)
			for i := range f.live.fog.visible {
				f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
			}
			f.live.push()
			actor := structureWitnessActor1114(t, f.live.world)
			if err := app.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
				t.Fatal(err)
			}
			inspectionCentre(f.live, 12, 12)
			if err := app.HeadlessKey("attack"); err != nil {
				t.Fatal(err)
			}
			x, y, err := f.live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: 0})
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err := app.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 256 && f.live.world.Structures()[0].Field42 == 90; i++ {
				f.live.tick()
			}
			if hp := f.live.world.Structures()[0].Field42; hp == 0 || hp >= 90 {
				t.Fatal("ordinary attack did not damage moved Building", hp)
			}
			f.live.enqueue(uint32(actor.ID), 11, 13)
			f.live.tick()
			groups, _, _ := f.live.world.SavedGroups()
			if len(groups) < 2 || !groups[len(groups)-1].Authored {
				t.Fatal("ordinary replacement did not reach current Group registry")
			}
			current := checkComposedDocument1115(t, f)
			// Current structure values change without rebinding repeated roots or
			// mutating an older Snapshot. Group export coverage remains separate.
			if !reflect.DeepEqual(before.SavedDocument.Document.World.Buildings, current.SavedDocument.Document.World.Buildings) {
				t.Fatal("source root identity changed during native action")
			}
			if old, now := composedDocumentBuildings1115(t, before), composedDocumentBuildings1115(t, current); old[0].Health != 90 || now[0].Health >= 90 || now[0].Health == 0 {
				t.Fatal("current document replayed old health or mutated old Snapshot", old[0].Health, now[0].Health)
			}
			doc, err := sav.EncodeDocumentData(*current.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			worldHash := fmt.Sprintf("%x", f.live.world.Hash())
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal("ordinary SAVE", entries, err, app.HeadlessMessage())
			}
			written, err := store.Read(entries[0].Name)
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			doc, err = canonicalDocument1115(t, ordinary)
			if err != nil {
				t.Fatal(err)
			}
			for range 20 {
				f.live.tick()
			}
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestDocument1115ComposedGroupsStructuresMenuSaveFreshProcess$", "-test.v")
			cmd.Env = append(os.Environ(), "AGAINROM_COMPOSED1115_NATIVE="+filepath.Join(store.Dir, entries[0].Name), "AGAINROM_COMPOSED1115_WORLD="+worldHash, "AGAINROM_COMPOSED1115_DOCUMENT="+fmt.Sprintf("%x", sha256.Sum256(doc)), "AGAINROM_COMPOSED1115_NEXT="+fmt.Sprintf("%x", f.live.world.Hash()))
			cmd.Env = append(cmd.Env, fmt.Sprintf("AGAINROM_COMPOSED_GENERATED=%t", fromMap))
			out, err := cmd.CombinedOutput()
			if err != nil || !bytes.Contains(out, []byte("next20 driver ticks PASS")) {
				t.Fatalf("fresh composed process: %v\n%s", err, out)
			}
			t.Log(string(out))
		})
	}
}
