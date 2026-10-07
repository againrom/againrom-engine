package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseSAVOriginalLoadPrerequisites(t *testing.T) {
	t.Run("mission", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		party := f.ChargenParty(ui.ChargenResult{Name: "LoadProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
		app := f.App("original LOAD prerequisites")
		if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
			t.Fatal(err)
		}
		app.Layout(1024, 768)
		snapshot, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentWorldSave(snapshot, "fresh world")
		if err != nil {
			t.Fatal(err)
		}
		assertOriginalJoinFields(t, raw, "LoadProbe", "20.alm")
		legacy, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		legacy.Head.MapName = `Scenario\20.alm`
		mustSetText(&legacy.Objects[legacy.Players[0]-1], "Name", "sElF")
		mustSetValue(&legacy.Objects[legacy.Players[0]-1], "F3D", 0)
		raw, err = sav.EncodeDocumentData(legacy)
		if err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(store.Dir, "legacy.sav"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		for pass := 0; pass < 2; pass++ {
			cold := releaseFront(t)
			cold.SetDeterministicFrames(true)
			_, _, load := cold.SaveSeams(store, OriginalStore{Dir: store.Dir}, nil)
			open, town, err := load("legacy.sav")
			if err != nil || town || open == nil {
				t.Fatalf("mission LOAD: town=%t err=%v", town, err)
			}
			app := cold.App("continued original LOAD prerequisites")
			if err := app.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			app.Layout(1024, 768)
			save, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
			name, err := save(true)
			if err != nil || !IsOriginal(name) {
				t.Fatalf("ordinary SAVE %q: %v", name, err)
			}
			raw, err = store.Read(name)
			if err != nil {
				t.Fatal(err)
			}
			assertOriginalJoinFields(t, raw, "LoadProbe", "20.alm")
			if err := os.WriteFile(filepath.Join(store.Dir, "legacy.sav"), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
	})
	for _, reserved := range []string{"Self", "cOmPuTeR"} {
		t.Run("town/"+reserved, func(t *testing.T) {
			f := currentTown(t, nil, nil)
			wantName := nativeCityHeroOf(f.Carried).Name
			legacy, err := sav.DecodeDocumentData(currentTownSave(t, f))
			if err != nil {
				t.Fatal(err)
			}
			mustSetText(&legacy.Objects[legacy.Players[0]-1], "Name", reserved)
			raw, err := sav.EncodeDocumentData(legacy)
			if err != nil {
				t.Fatal(err)
			}
			for pass := 0; pass < 2; pass++ {
				cold := currentTownReload(t, raw)
				raw = currentTownSave(t, cold)
				assertOriginalJoinFields(t, raw, wantName, "")
				if cold.Town.Gold() != f.Town.Gold() || len(cold.Carried) != len(f.Carried) {
					t.Fatal("participant repair changed the purse or roster")
				}
			}
		})
	}
}

func assertOriginalJoinFields(t *testing.T, raw []byte, wantName, wantMap string) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if wantMap != "" && f.Head.MapName != wantMap {
		t.Errorf("original prefixes Scenario to SAV map %q; want relative %q", f.Head.MapName, wantMap)
	}
	if len(f.Players) == 0 {
		t.Fatal("no participant")
	}
	if f.Players[0].Name != wantName {
		t.Errorf("original join participant = %q; want name %q", f.Players[0].Name, wantName)
	}
	if f.World != nil && f.Players[0].Fields.Value["F3D"] != 1 {
		t.Errorf("original LOAD would place retained actors again: F3D=%d", f.Players[0].Fields.Value["F3D"])
	}
	if f.World == nil && f.Players[0].Fields.Value["F3D"] != 0 {
		t.Errorf("town keeps a mission placement latch: F3D=%d", f.Players[0].Fields.Value["F3D"])
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range doc.State.ValueRecords {
		if value.Path == "/Character/Name" && string(value.Value.Bytes) != wantName+"\x00" {
			t.Errorf("character name = %q; want %q", value.Value.Bytes, wantName)
		}
	}
}
