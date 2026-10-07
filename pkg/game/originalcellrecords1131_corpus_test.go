//go:build sessioncorpusaudit

package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCellRecordCorpusAudit1131(t *testing.T) {
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
				unreadable++
				t.Logf("unreadable, skipped: %v", err)
				return
			}
			if source.World == nil {
				cities++
				t.Log("between-mission save: no cell-record table, nothing to compare")
				return
			}
			worlds++
			fileCells, present, err := source.Cells()
			if err != nil || !present {
				t.Fatalf("Cells: present=%v err=%v", present, err)
			}
			// Last-write-wins by key, independently of applyOriginalCellRecords'
			// own dedup loop: SAV-CELLLOAD-109's own restore order.
			want := map[uint16]sav.Cell{}
			for _, c := range fileCells {
				want[c.Key] = c
			}

			ms, report, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}
			if !report.CellRecordsApplied {
				t.Fatal("cell records were not applied")
			}

			structures, structCells, _ := ms.World.SavedStructures()
			sourceKey := make(map[sim.StructureID]uint32, len(structures))
			for _, s := range structures {
				sourceKey[s.ID] = s.SourceKey
			}
			cellByKey := make(map[uint16]sim.SavedStructureCell, len(structCells))
			for _, c := range structCells {
				cellByKey[c.Cell] = c
			}
			recByKey := make(map[uint16]sim.SavedCellRecord, len(want))
			for _, rec := range ms.World.SavedCellRecords() {
				recByKey[rec.Cell] = rec
			}
			tailByKey := make(map[uint16][6]byte, len(want))
			for _, tail := range ms.World.CellTails() {
				tailByKey[uint16(uint8(tail.Y))<<8|uint16(uint8(tail.X))] = tail.Bytes
			}
			byIdentity := make(map[uint32]sim.EntityID, len(ms.World.Entities()))
			for _, e := range ms.World.Entities() {
				if e.SourceBinding.Identity != 0 {
					byIdentity[e.SourceBinding.Identity] = e.ID
				}
			}
			checkActorSlot := func(key uint16, label string, wantKey uint32, got sim.SavedCellActorSlot) {
				if got.Key != wantKey {
					mismatches++
					t.Errorf("cell %#04x %s key = %#x, want %#x", key, label, got.Key, wantKey)
				}
				id, live := byIdentity[wantKey]
				if wantKey != 0 && live {
					if !got.Bound || got.Entity != id {
						mismatches++
						t.Errorf("cell %#04x %s: want bound to live entity %d, got bound=%v entity=%d", key, label, id, got.Bound, got.Entity)
					}
				} else if got.Bound {
					mismatches++
					t.Errorf("cell %#04x %s: bound to entity %d with no matching live actor identity", key, label, got.Entity)
				}
			}

			diffFields := 0
			for key, w := range want {
				sc := cellByKey[key]
				if sc.BaselineCost != w.BaselineCost || sc.BaselineStatic != w.BaselineStatic {
					diffFields++
					t.Errorf("cell %#04x baseline = %d/%d, want %d/%d", key, sc.BaselineCost, sc.BaselineStatic, w.BaselineCost, w.BaselineStatic)
				}
				gotBuilding := uint32(0)
				if sc.HasStructure {
					gotBuilding = sourceKey[sc.ID]
				}
				if gotBuilding != w.Building {
					diffFields++
					t.Errorf("cell %#04x Building = %#x, want %#x", key, gotBuilding, w.Building)
				}
				rec := recByKey[key]
				if rec.LayerCount != w.LayerCount {
					diffFields++
					t.Errorf("cell %#04x LayerCount = %d, want %d", key, rec.LayerCount, w.LayerCount)
				}
				if rec.Residue0 != w.Residue0 || rec.Residue1 != w.Residue1 {
					diffFields++
					t.Errorf("cell %#04x residue = %d/%v, want %d/%v", key, rec.Residue0, rec.Residue1, w.Residue0, w.Residue1)
				}
				if rec.Sack != w.Sack {
					diffFields++
					t.Errorf("cell %#04x Sack = %#x, want %#x", key, rec.Sack, w.Sack)
				}
				if rec.SpellEffects != w.SpellEffects {
					diffFields++
					t.Errorf("cell %#04x SpellEffects = %v, want %v", key, rec.SpellEffects, w.SpellEffects)
				}
				checkActorSlot(key, "Ground", w.Ground, rec.Ground)
				checkActorSlot(key, "Air", w.Air, rec.Air)
				if tailByKey[key] != w.Trigger {
					diffFields++
					t.Errorf("cell %#04x Trigger = %v, want %v", key, tailByKey[key], w.Trigger)
				}
			}
			if diffFields > 0 {
				mismatches++
			}

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes, on TestSessionCorpusAudit1130's own
			// Marshal/re-Open precedent, then compare the complete cell-record
			// table span byte for byte.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalCellRecords(target, ms.World); err != nil {
				t.Fatalf("exportOriginalCellRecords: %v", err)
			}
			written, err := sav.Open(target.Marshal())
			if err != nil {
				t.Fatalf("sav.Open (re-decode written): %v", err)
			}
			off := target.World.CellRecDataOff
			size := target.World.CellRecCount * 54
			if !bytes.Equal(written.Body[off:off+size], source.Body[off:off+size]) {
				mismatches++
				t.Error("native export of the cell-record table does not reproduce the source file")
			}
			nzBase, nzLayer, nzGround, nzAir, nzBuilding, nzSack, nzSE, nzRes0, nzRes1 := 0, 0, 0, 0, 0, 0, 0, 0, 0
			for _, w := range want {
				if w.BaselineCost != 0 || w.BaselineStatic != 0 {
					nzBase++
				}
				if w.LayerCount != 0 {
					nzLayer++
				}
				if w.Ground != 0 {
					nzGround++
				}
				if w.Air != 0 {
					nzAir++
				}
				if w.Building != 0 {
					nzBuilding++
				}
				if w.Sack != 0 {
					nzSack++
				}
				if w.SpellEffects != [6]uint32{} {
					nzSE++
				}
				if w.Residue0 != 0 {
					nzRes0++
				}
				if w.Residue1 != [2]byte{} {
					nzRes1++
				}
			}
			t.Logf("records=%d unique=%d nonzero: baseline=%d layer=%d ground=%d air=%d building=%d sack=%d spelleffects=%d residue0=%d residue1=%d; diffFields=%d",
				len(fileCells), len(want), nzBase, nzLayer, nzGround, nzAir, nzBuilding, nzSack, nzSE, nzRes0, nzRes1, diffFields)
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d mismatching",
		total, worlds, cities, unreadable, mismatches)
}
