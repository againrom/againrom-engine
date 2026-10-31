package game

import (
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// DIV-944
func TestNativeResumeCorpusCycle1139(t *testing.T) {
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
	nativeFiles, nativeRoots := 0, 0
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
			if source.World == nil {
				cities++
				t.Log("between-mission save: no world-state store, nothing to cycle")
				return
			}
			worlds++

			// Roots bound as native areas leave the carried graph at LOAD
			// (DIV-1271, DIV-2885), so the SpellEffect export refuses them;
			// the cycle must keep the native areas and the carried graph.
			document, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatalf("DecodeDocumentData: %v", err)
			}
			native, err := spellEffectRootsBoundNative(&document)
			if err != nil {
				t.Fatalf("spellEffectRootsBoundNative: %v", err)
			}
			actions, err := readCurrentActions(&document)
			if err != nil {
				t.Fatalf("readCurrentActions: %v", err)
			}
			nativeAreaRows := 0
			if actions != nil {
				nativeAreaRows = len(actions.NativeAreas)
			}

			ms, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatalf("RestoreOriginal: %v", err)
			}

			// Pre-cycle export: the world exactly as the original LOAD left
			// it. Each checker gets its own fresh decode of raw, since an
			// export mutates its target in place and the checkers below
			// would otherwise compound on one another's patches.
			before := make([][]byte, len(checkers))
			for i, c := range checkers {
				if c.name == "SpellEffects" && len(native) > 0 {
					continue
				}
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
			if len(native) > 0 {
				nativeFiles++
				nativeRoots += len(native)
				areasBefore, err := ms.World.NativeAreaSaveStates()
				if err != nil {
					t.Fatalf("NativeAreaSaveStates (pre-cycle): %v", err)
				}
				areasAfter, err := resumed.NativeAreaSaveStates()
				if err != nil {
					t.Fatalf("NativeAreaSaveStates (post-cycle): %v", err)
				}
				if len(areasBefore) != nativeAreaRows {
					mismatchedHere = true
					t.Errorf("LOAD bound %d native area(s), the file's current actions name %d", len(areasBefore), nativeAreaRows)
				}
				if !reflect.DeepEqual(areasBefore, areasAfter) {
					mismatchedHere = true
					t.Errorf("native resume cycle changed the World's native areas: %d before, %d after", len(areasBefore), len(areasAfter))
				}
				if !reflect.DeepEqual(ms.World.SavedSpellEffects(), resumed.SavedSpellEffects()) {
					mismatchedHere = true
					t.Errorf("native resume cycle changed the carried SpellEffect graph")
				}
			}
			for i, c := range checkers {
				if c.name == "SpellEffects" && len(native) > 0 {
					continue
				}
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
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d root(s) of %d file(s) bound as native effects, %d mismatching",
		total, worlds, cities, unreadable, nativeRoots, nativeFiles, mismatches)
}
