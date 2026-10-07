//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestGroundContainerCorpusAudit1136(t *testing.T) {
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

	total, worlds, cities, unreadable, nonEmpty, mismatches := 0, 0, 0, 0, 0, 0
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
			if source.World == nil {
				cities++
				t.Log("between-mission save: no world-state store, nothing to compare")
				return
			}
			worlds++
			sacks, present, err := source.GroundSacks()
			if err != nil {
				t.Fatalf("GroundSacks: %v", err)
			}
			if present && len(sacks) > 0 {
				nonEmpty++
			}

			ms, report, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}
			if present && len(sacks) > 0 && !report.GroundApplied {
				t.Fatal("a save with ground Sacks was not reported as having its ground state applied")
			}
			if (!present || len(sacks) == 0) && report.GroundApplied {
				t.Fatal("a save with no ground Sacks was reported as having ground state applied")
			}

			// The single carrier: sim.SavedObjectContainer reaches the same two
			// fields through an entirely different decode
			// (sav.DecodeDocumentDataWithOrigins, not GroundSacks) and is populated
			// by importSavedSackObjects inside the same ResumeOriginalSave call
			// above. Join on the raw file Identity through liveGroundContainerTail; a
			// Sack importSavedSackObjects declined to adopt has no registry entry and
			// is skipped here, not treated as a mismatch.
			registry := ms.World.SavedObjects()
			checked, skipped := 0, 0
			for _, sack := range sacks {
				tail, ok := liveGroundContainerTail(registry, sack.Identity)
				if !ok {
					skipped++
					continue
				}
				checked++
				if tail.InsertIndex != sack.InsertIndex || tail.Accumulator != sack.Accumulator {
					mismatches++
					t.Errorf("live SavedObjectContainer disagrees with the file for Sack %#x: got insert=%d acc=%d, want insert=%d acc=%d",
						sack.Identity, tail.InsertIndex, tail.Accumulator, sack.InsertIndex, sack.Accumulator)
				}
			}
			t.Logf("single carrier: %d Sack(s) cross-checked, %d not adopted by importSavedSackObjects", checked, skipped)

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes, then compare the WHOLE FILE byte for byte
			// (not only the decoded value, on this test's own doc comment):
			// SetGroundContainerTails touches nothing but the two tail dwords
			// per Sack, and this story never enacts or recomputes either one,
			// so writing the same live values back must leave every other byte
			// exactly as the source file had it.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalGroundContainers(target, ms.World); err != nil {
				t.Fatalf("exportOriginalGroundContainers: %v", err)
			}
			exportedBytes := target.Marshal()
			if len(exportedBytes) != len(raw) {
				mismatches++
				t.Errorf("native export changed the file length: got %d, want %d", len(exportedBytes), len(raw))
			} else {
				diffs := 0
				for i := range raw {
					if exportedBytes[i] != raw[i] {
						diffs++
					}
				}
				if diffs != 0 {
					mismatches++
					t.Errorf("native export changed %d byte(s) outside the two carried tail dwords", diffs)
				}
			}
			t.Logf("present=%v sacks=%d; native export reproduces the source byte for byte", present, len(sacks))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d non-empty, %d mismatching",
		total, worlds, cities, unreadable, nonEmpty, mismatches)
}
