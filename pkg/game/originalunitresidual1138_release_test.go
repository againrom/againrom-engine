package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseOriginalUnitResidualFieldsRestoreOnLoad1138(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	holdings, err := source.ActorHoldings()
	if err != nil {
		t.Fatalf("ActorHoldings: %v", err)
	}
	byMapUnitID := make(map[uint16]sav.ActorHoldings, len(holdings))
	for _, h := range holdings {
		byMapUnitID[h.MapUnitID] = h
	}

	type want struct {
		class           string
		ownWeight, load int16
		sightCells      uint8
		token18         uint16
	}
	cases := map[uint16]want{
		21: {class: "Human", ownWeight: 0, load: 0, sightCells: 5, token18: 2},
		44: {class: "Human", ownWeight: 3, load: 3, sightCells: 6, token18: 2},
		48: {class: "Human", ownWeight: 106, load: 106, sightCells: 6, token18: 2},
		25: {class: "Unit", ownWeight: 0, load: 0, sightCells: 8, token18: 2},
	}
	if cases[48].ownWeight == cases[21].ownWeight {
		t.Fatal("fixture missing real content: every own-weight case is the same value")
	}

	for id, w := range cases {
		h, ok := byMapUnitID[id]
		if !ok || h.Basis == nil {
			t.Fatalf("map unit %d: not found with a basis", id)
		}
		if h.Basis.Class != w.class {
			t.Errorf("map unit %d: file Class = %s, want %s", id, h.Basis.Class, w.class)
		}
		if !h.LoadState.Present || h.LoadState.OwnWeight != w.ownWeight || h.LoadState.Load != w.load {
			t.Errorf("map unit %d: file OwnWeight/Load = present=%v %d/%d, want %d/%d",
				id, h.LoadState.Present, h.LoadState.OwnWeight, h.LoadState.Load, w.ownWeight, w.load)
		}
		if cells := uint8(h.Basis.Human.Fields.Sight >> 8); cells != w.sightCells {
			t.Errorf("map unit %d: file whole-cell sight = %d, want %d", id, cells, w.sightCells)
		}
		if h.Basis.Token18 != w.token18 {
			t.Errorf("map unit %d: file Token18 = %#04x, want %#04x", id, h.Basis.Token18, w.token18)
		}
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1138 unit residual fields")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0021.sav")

	for id, w := range cases {
		entity := poolEntity(t, f.live.world, id)
		if entity.ActorLoad.OwnWeight != w.ownWeight || entity.Load != int32(w.load) {
			t.Errorf("map unit %d: LOAD live OwnWeight/Load = %d/%d, want %d/%d",
				id, entity.ActorLoad.OwnWeight, entity.Load, w.ownWeight, w.load)
		}
		if entity.ScanRange != w.sightCells {
			t.Errorf("map unit %d: LOAD live ScanRange = %d, want %d", id, entity.ScanRange, w.sightCells)
		}
	}
	t.Logf("%d map unit(s) cross-checked: file OwnWeight/Load/sight/Token18 match, and LOAD reproduces OwnWeight/Load/ScanRange live", len(cases))
}
