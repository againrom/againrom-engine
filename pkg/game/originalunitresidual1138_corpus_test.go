//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// DIV-968
func TestUnitResidualCorpusAudit1138(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	f.SetDeterministicFrames(true)

	total, worlds, cities, unreadable := 0, 0, 0, 0
	actorsChecked, loadMismatches, sightMismatches := 0, 0, 0
	token18Values := map[uint16]int{}
	trailerTurnValues, trailerScriptValues := map[uint32]int{}, map[uint32]int{}
	trailerRoundTripMismatches := 0

	walkErr := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		total++
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source, err := sav.Open(raw)
			if err != nil {
				unreadable++
				t.Logf("unreadable, skipped: %v", err)
				return
			}

			// Trailer: present at the end of every document body regardless
			// of World (SAV-790), so every openable file is checked here.
			trailerTurnValues[source.Trailer.TurnTracing]++
			trailerScriptValues[source.Trailer.ScriptTracing]++
			reopened, err := sav.Open(source.Marshal())
			if err != nil {
				t.Fatalf("re-open after Marshal: %v", err)
			}
			if reopened.Trailer != source.Trailer {
				trailerRoundTripMismatches++
				t.Errorf("trailer changed across a Marshal round trip")
			}

			if source.World == nil {
				cities++
				t.Log("between-mission save: no world-state store, nothing further to compare")
				return
			}
			worlds++

			holdings, err := source.ActorHoldings()
			if err != nil {
				t.Fatalf("ActorHoldings: %v", err)
			}
			byMapUnitID := make(map[uint16]sav.ActorHoldings, len(holdings))
			for _, h := range holdings {
				if h.Basis != nil {
					token18Values[h.Basis.Token18]++
				}
				if h.MapUnitID != 0 {
					byMapUnitID[h.MapUnitID] = h
				}
			}

			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}
			for _, e := range ms.World.Entities() {
				if e.MapUnitID == 0 {
					continue
				}
				h, ok := byMapUnitID[e.MapUnitID]
				if !ok || h.Basis == nil || !h.LoadState.Present {
					continue
				}
				actorsChecked++
				if e.ActorLoad.OwnWeight != h.LoadState.OwnWeight || e.Load != int32(h.LoadState.Load) {
					loadMismatches++
					t.Errorf("map unit %d: live OwnWeight/Load %d/%d, file %d/%d",
						e.MapUnitID, e.ActorLoad.OwnWeight, e.Load, h.LoadState.OwnWeight, h.LoadState.Load)
				}
				if wantCells := uint8(h.Basis.Human.Fields.Sight >> 8); e.ScanRange != wantCells {
					sightMismatches++
					t.Errorf("map unit %d: live ScanRange %d, file whole-cell sight %d (raw %#04x)",
						e.MapUnitID, e.ScanRange, wantCells, h.Basis.Human.Fields.Sight)
				}
			}
			t.Logf("present=%v holdings=%d", source.World != nil, len(holdings))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable",
		total, worlds, cities, unreadable)
	t.Logf("%d actor(s) cross-checked: %d OwnWeight/Load mismatches, %d sight mismatches",
		actorsChecked, loadMismatches, sightMismatches)
	t.Logf("Token18 corpus distribution: %v (SAV-797: only 0 and 2 on the research corpus)", token18Values)
	// SAV-649's own "0 in 44 of 45, 520 in one" finding is about the marker-
	// adjacent global dword this package deliberately still skips uninspected
	// (the four bytes right before TrailerBody, SAV-648) -- a different field
	// from TurnTracing/ScriptTracing below, which SAV-647/SAV-694 report as
	// debug toggles taking small values (0/1) and are logged here only as a
	// decode sanity check, not against SAV-649's own number.
	t.Logf("Trailer TurnTracing distribution: %v; ScriptTracing distribution: %v",
		trailerTurnValues, trailerScriptValues)
	if trailerRoundTripMismatches != 0 {
		t.Fatalf("%d file(s) failed a trailer Marshal round trip", trailerRoundTripMismatches)
	}
	if loadMismatches != 0 || sightMismatches != 0 {
		t.Fatalf("%d load mismatch(es), %d sight mismatch(es)", loadMismatches, sightMismatches)
	}
}
