//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestSpellEffectCorpusAudit1132(t *testing.T) {
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
				t.Log("between-mission save: no SpellEffect list, nothing to compare")
				return
			}
			worlds++
			want, present, err := source.SpellEffects()
			if err != nil || !present {
				t.Fatalf("SpellEffects: present=%v err=%v", present, err)
			}
			if len(want) > 0 {
				nonEmpty++
			}

			ms, report, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}
			if !report.SpellEffectsApplied {
				t.Fatal("spell effects were not applied")
			}
			if report.SpellEffects != len(want) {
				mismatches++
				t.Errorf("report.SpellEffects = %d, want %d", report.SpellEffects, len(want))
			}

			// The live comparison value is the world's own carried state before
			// any tick: ResumeOriginalSave advances none. This walk never calls
			// spellEffectConverter/applyOriginalSpellEffects, so a bug shared
			// with the production conversion would not silently pass both this
			// and the byte-level check below.
			live := ms.World.SavedSpellEffects()
			if msg, ok := sameSpellEffectGraphForAudit(want, live); !ok {
				mismatches++
				t.Errorf("file vs live graph: %s", msg)
			}

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes, then compare the complete body, on
			// TestCellRecordCorpusAudit1131's own Marshal/re-Open precedent.
			// exportOriginalSpellEffects only ever patches scalar bytes inside
			// the SpellEffect list's own span (TestSetSpellEffectsLeavesUnrelatedBytesAlone),
			// so a whole-body comparison against the untouched source is exactly
			// as strong as isolating that span would be.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalSpellEffects(target, ms.World); err != nil {
				t.Fatalf("exportOriginalSpellEffects: %v", err)
			}
			written, err := sav.Open(target.Marshal())
			if err != nil {
				t.Fatalf("sav.Open (re-decode written): %v", err)
			}
			if len(written.Body) != len(source.Body) {
				mismatches++
				t.Errorf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
			} else {
				diff := 0
				for i := range source.Body {
					if source.Body[i] != written.Body[i] {
						diff++
					}
				}
				if diff > 0 {
					mismatches++
					t.Errorf("native export of the SpellEffect list does not reproduce the source file: %d byte(s) differ", diff)
				}
			}
			t.Logf("top-level records=%d", len(want))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d non-empty SpellEffect list(s), %d mismatching",
		total, worlds, cities, unreadable, nonEmpty, mismatches)
}
