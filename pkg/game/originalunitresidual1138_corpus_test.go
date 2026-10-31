package game

import (
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

// DIV-968
func TestUnitResidualCorpusAudit1138(t *testing.T) {
	sessionCorpusAudit(t)
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
	actorsChecked, loadMismatches, sightMismatches, savedActors, rewrittenHumans := 0, 0, 0, 0, 0
	token18Values := map[uint16]int{}
	trailerTurnValues, trailerScriptValues := map[uint32]int{}, map[uint32]int{}
	trailerRoundTripMismatches := 0

	walkErr := walkFirstGameSaves(corpus, func(path, rel string) error {
		total++
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
			for _, h := range holdings {
				if h.Basis != nil {
					token18Values[h.Basis.Token18]++
				}
			}
			graph, err := source.ActorGraph()
			if err != nil {
				t.Fatalf("ActorGraph: %v", err)
			}
			byArchive := archiveActorsForAudit(graph)
			byMapUnit := unitResidualMapUnitsForAudit(byArchive)

			loaded, err := loadOriginalFront(f, raw)
			if err != nil {
				t.Fatalf("RestoreOriginal: %v", err)
			}
			var unbound []sav.ActorRecord
			for _, e := range loaded.CurrentMission().World.Entities() {
				if e.MapUnitID == 0 {
					continue
				}
				a, ok := unitResidualRecordForAudit(e, byArchive, byMapUnit)
				if !ok || a.Character.Basis == nil || !a.Character.LoadState.Present {
					continue
				}
				actorsChecked++
				own, load, sight := unitResidualWantForAudit(a.Character)
				// An entity without a load record holds no own weight word;
				// its SAVE derives one, and is checked below.
				ownHeld := !e.ActorLoad.Present || e.ActorLoad.OwnWeight == own
				if !e.ActorLoad.Present {
					unbound = append(unbound, a)
				}
				if !ownHeld || e.Load != int32(load) {
					loadMismatches++
					t.Errorf("map unit %d (archive %d): live OwnWeight/Load %d/%d, want %d/%d (file %d/%d)",
						e.MapUnitID, a.ArchiveIndex, e.ActorLoad.OwnWeight, e.Load, own, load, a.Character.LoadState.OwnWeight, a.Character.LoadState.Load)
				}
				if e.ScanRange != sight {
					sightMismatches++
					t.Errorf("map unit %d (archive %d): live ScanRange %d, file whole-cell sight %d (raw %#04x)",
						e.MapUnitID, a.ArchiveIndex, e.ScanRange, sight, a.Character.Basis.Human.Fields.Sight)
				}
			}
			if len(unbound) > 0 {
				savedActors += len(unbound)
				humans, l, s := unitResidualSaveForAudit(t, loaded, unbound)
				rewrittenHumans, loadMismatches, sightMismatches = rewrittenHumans+humans, loadMismatches+l, sightMismatches+s
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
	t.Logf("%d actor(s) cross-checked, %d without a load record also through SAVE (%d native Human own weight(s) of 0 written by DIV-3358's rule): %d OwnWeight/Load mismatches, %d sight mismatches",
		actorsChecked, savedActors, rewrittenHumans, loadMismatches, sightMismatches)
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

// unitResidualSaveForAudit checks the player's SAVE of actors LOAD kept
// without a load record: each is written once under its map unit ID with the
// file's load, sight and Unit own weight. A native Human the file holds with
// own weight 0 is written with its worn set's weight (DIV-3358) and counted.
func unitResidualSaveForAudit(t *testing.T, loaded *FrontEnd, records []sav.ActorRecord) (humans, loads, sights int) {
	t.Helper()
	s, _, err := loaded.Snapshot(true)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	saved, _, err := loaded.playerMissionSave(s, "unit residual audit")
	if err != nil {
		t.Fatalf("SAVE: %v", err)
	}
	out, err := sav.Open(saved)
	if err != nil {
		t.Fatalf("sav.Open (SAVE): %v", err)
	}
	graph, err := out.ActorGraph()
	if err != nil {
		t.Fatalf("SAVE's ActorGraph: %v", err)
	}
	written := unitResidualMapUnitsForAudit(archiveActorsForAudit(graph))
	for _, a := range records {
		got := written[a.MapUnitID]
		if len(got) != 1 || got[0].Character.Basis == nil || !got[0].Character.LoadState.Present {
			loads++
			t.Errorf("map unit %d: SAVE wrote %d record(s)", a.MapUnitID, len(got))
			continue
		}
		own, load, sight := unitResidualWantForAudit(a.Character)
		g := got[0].Character
		ownHeld := g.LoadState.OwnWeight == own
		if !ownHeld && a.Character.Basis.Class != "Unit" && own == 0 {
			humans++
			ownHeld = true
		}
		if !ownHeld || g.LoadState.Load != load {
			loads++
			t.Errorf("map unit %d: SAVE wrote OwnWeight/Load %d/%d, file %d/%d", a.MapUnitID, g.LoadState.OwnWeight, g.LoadState.Load, own, load)
		}
		if got := uint8(g.Basis.Human.Fields.Sight >> 8); got != sight {
			sights++
			t.Errorf("map unit %d: SAVE wrote whole-cell sight %d, file %d", a.MapUnitID, got, sight)
		}
	}
	return humans, loads, sights
}
