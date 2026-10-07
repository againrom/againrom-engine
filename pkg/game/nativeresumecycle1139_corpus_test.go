//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// DIV-944
func TestNativeResumeCorpusCycle1139(t *testing.T) {
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

	type checker struct {
		name   string
		export func(*sav.File, *sim.World) error
	}
	checkers := []checker{
		{"MissionSession", exportOriginalMissionSession},
		{"CellRecords", exportOriginalCellRecords},
		{"SpellEffects", exportOriginalSpellEffects},
		{"Projectiles", exportOriginalProjectiles},
		{"Diaries", exportOriginalDiaries},
	}

	total, worlds, cities, unreadable, mismatches := 0, 0, 0, 0, 0
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
				t.Log("between-mission save: no world-state store, nothing to cycle")
				return
			}
			worlds++

			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}

			// Pre-cycle export: the world exactly as the original LOAD left
			// it. Each checker gets its own fresh decode of raw, since an
			// export mutates its target in place and the checkers below
			// would otherwise compound on one another's patches.
			before := make([][]byte, len(checkers))
			for i, c := range checkers {
				target, err := sav.Open(raw)
				if err != nil {
					t.Fatalf("sav.Open (pre-cycle %s): %v", c.name, err)
				}
				if err := c.export(target, ms.World); err != nil {
					t.Fatalf("export %s (pre-cycle): %v", c.name, err)
				}
				before[i] = target.Marshal()
			}

			// The cycle: an ordinary native SAVE (MarshalBinary) followed by
			// a fresh native LOAD (UnmarshalBinary onto a receiver that ran
			// no Import*/apply* call of its own) — the exact shape of a
			// genuine cross-process resume of the `.ags` file that SAVE
			// would have written.
			encoded, err := ms.World.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var resumed sim.World
			if err := resumed.UnmarshalBinary(encoded); err != nil {
				t.Fatalf("UnmarshalBinary: %v", err)
			}

			// Post-cycle export: the same checkers, against fresh decodes of
			// the same source file again, this time reading the resumed
			// (post-cycle) world.
			mismatchedHere := false
			for i, c := range checkers {
				target, err := sav.Open(raw)
				if err != nil {
					t.Fatalf("sav.Open (post-cycle %s): %v", c.name, err)
				}
				if err := c.export(target, &resumed); err != nil {
					t.Fatalf("export %s (post-cycle): %v", c.name, err)
				}
				after := target.Marshal()
				if len(after) != len(before[i]) {
					mismatchedHere = true
					t.Errorf("%s: native resume cycle changed the exported length: got %d, want %d", c.name, len(after), len(before[i]))
					continue
				}
				diffs := 0
				for j := range after {
					if after[j] != before[i][j] {
						diffs++
					}
				}
				if diffs != 0 {
					mismatchedHere = true
					t.Errorf("%s: native resume cycle changed %d exported byte(s)", c.name, diffs)
				}
			}
			if mismatchedHere {
				mismatches++
			}
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d mismatching",
		total, worlds, cities, unreadable, mismatches)
}
