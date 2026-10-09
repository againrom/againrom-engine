//go:build sessioncorpusaudit

package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestSessionCorpusAudit1130(t *testing.T) {
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
				// A preserved file can fail to decode (game9000.sav is not
				// an Asg& container at all); detected by the open failing,
				// never by the file's name, and counted rather than
				// aborting the audit.
				unreadable++
				t.Logf("unreadable, skipped: %v", err)
				return
			}
			if source.World == nil {
				cities++
				t.Log("between-mission save: no session block, nothing to compare")
				return
			}
			worlds++
			fileState, err := source.SessionState()
			if err != nil {
				t.Fatalf("SessionState: %v", err)
			}

			ms, report, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatalf("RestoreOriginal: %v", err)
			}
			if !report.SessionApplied {
				t.Fatal("session was not applied")
			}

			liveRegs := ms.World.ScriptRegisters()
			diffRegs := 0
			for i, v := range liveRegs {
				if v != fileState.Registers[i] {
					diffRegs++
					t.Errorf("register %d: live=%d file=%d", i, v, fileState.Registers[i])
				}
			}
			if got := ms.World.RawSessionHead(); got != fileState.RawHead {
				mismatches++
				t.Errorf("RawSessionHead live/file differ:\n  live=%x\n  file=%x", got, fileState.RawHead)
			}
			if got := ms.World.RawSessionMid(); got != fileState.RawMid {
				mismatches++
				t.Errorf("RawSessionMid live/file differ:\n  live=%x\n  file=%x", got, fileState.RawMid)
			}
			if diffRegs > 0 {
				mismatches++
			}

			// Native export: write the live state back into a FRESH decode of
			// this file's own bytes, MARSHAL that (the container header plus
			// COMPRESSED body — Marshal's own contract, pkg/formats/sav/
			// container.go), then re-OPEN the marshaled bytes so the
			// comparison reads two DECOMPRESSED bodies at the same
			// SessionOff-relative offset. Comparing Marshal's compressed
			// output directly against raw at a Body offset is a category
			// error — the two are compressed and decompressed views of the
			// same file — and produces a slice-bounds panic, not a
			// meaningful mismatch.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalMissionSession(target, ms.World); err != nil {
				t.Fatalf("exportOriginalMissionSession: %v", err)
			}
			written, err := sav.Open(target.Marshal())
			if err != nil {
				t.Fatalf("sav.Open (re-decode written): %v", err)
			}
			base := target.World.SessionOff
			if !bytes.Equal(written.Body[base:base+400], source.Body[base:base+400]) {
				mismatches++
				t.Error("native export of the register span (0..400) does not reproduce the source file")
			}
			if !bytes.Equal(written.Body[base+1400:base+1848], source.Body[base+1400:base+1848]) {
				mismatches++
				t.Error("native export of the two raw spans (1400..1848) does not reproduce the source file")
			}
			nonzeroRegs, nonzeroHead, nonzeroMid := 0, 0, 0
			for _, v := range fileState.Registers {
				if v != 0 {
					nonzeroRegs++
				}
			}
			for _, b := range fileState.RawHead {
				if b != 0 {
					nonzeroHead++
				}
			}
			for _, b := range fileState.RawMid {
				if b != 0 {
					nonzeroMid++
				}
			}
			t.Logf("registers non-zero=%d/100 diff=%d; RawHead non-zero=%d/48; RawMid non-zero=%d/400",
				nonzeroRegs, diffRegs, nonzeroHead, nonzeroMid)
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d mismatching",
		total, worlds, cities, unreadable, mismatches)
}
