package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseOriginalCellRecordsRestoreOnLoad1131(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game9999.sav", "5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileCells, present, err := source.Cells()
	if err != nil || !present {
		t.Fatalf("Cells: present=%v err=%v", present, err)
	}
	want := map[uint16]sav.Cell{}
	for _, c := range fileCells {
		want[c.Key] = c
	}
	nzBase, nzGround, nzAir, nzBuilding := 0, 0, 0, 0
	for _, c := range want {
		if c.BaselineCost != 0 || c.BaselineStatic != 0 {
			nzBase++
		}
		if c.Ground != 0 {
			nzGround++
		}
		if c.Air != 0 {
			nzAir++
		}
		if c.Building != 0 {
			nzBuilding++
		}
	}
	if nzBase == 0 || nzGround == 0 || nzAir == 0 || nzBuilding == 0 {
		t.Fatalf("fixture missing real content: baseline=%d ground=%d air=%d building=%d", nzBase, nzGround, nzAir, nzBuilding)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1131 cell records")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	// Independently reconstructed live comparison: reads SavedStructures,
	// SavedCellRecords and Entities directly, the same shape
	// TestCellRecordCorpusAudit1131 uses, rather than calling
	// exportOriginalCellRecords for both directions of the check.
	structures, structCells, _ := f.live.world.SavedStructures()
	sourceKey := make(map[sim.StructureID]uint32, len(structures))
	for _, s := range structures {
		sourceKey[s.ID] = s.SourceKey
	}
	cellByKey := make(map[uint16]sim.SavedStructureCell, len(structCells))
	for _, c := range structCells {
		cellByKey[c.Cell] = c
	}
	recByKey := make(map[uint16]sim.SavedCellRecord, len(want))
	for _, rec := range f.live.world.SavedCellRecords() {
		recByKey[rec.Cell] = rec
	}
	byIdentity := make(map[uint32]sim.EntityID, len(f.live.world.Entities()))
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Identity != 0 {
			byIdentity[e.SourceBinding.Identity] = e.ID
		}
	}
	checkSlot := func(key uint16, label string, wantKey uint32, got sim.SavedCellActorSlot) {
		if got.Key != wantKey {
			t.Errorf("cell %#04x %s key = %#x, want %#x", key, label, got.Key, wantKey)
		}
		if id, live := byIdentity[wantKey]; wantKey != 0 && live {
			if !got.Bound || got.Entity != id {
				t.Errorf("cell %#04x %s: want bound to live entity %d, got bound=%v entity=%d", key, label, id, got.Bound, got.Entity)
			}
		} else if got.Bound {
			t.Errorf("cell %#04x %s: bound to entity %d with no matching live identity", key, label, got.Entity)
		}
	}
	for key, w := range want {
		sc := cellByKey[key]
		if sc.BaselineCost != w.BaselineCost || sc.BaselineStatic != w.BaselineStatic {
			t.Errorf("cell %#04x baseline = %d/%d, want %d/%d", key, sc.BaselineCost, sc.BaselineStatic, w.BaselineCost, w.BaselineStatic)
		}
		gotBuilding := uint32(0)
		if sc.HasStructure {
			gotBuilding = sourceKey[sc.ID]
		}
		if gotBuilding != w.Building {
			t.Errorf("cell %#04x Building = %#x, want %#x", key, gotBuilding, w.Building)
		}
		rec := recByKey[key]
		checkSlot(key, "Ground", w.Ground, rec.Ground)
		checkSlot(key, "Air", w.Air, rec.Air)
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the complete cell-record table span to
	// reproduce the source file byte for byte.
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalCellRecords(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalCellRecords: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	off, size := target.World.CellRecDataOff, target.World.CellRecCount*54
	if !bytes.Equal(written.Body[off:off+size], source.Body[off:off+size]) {
		t.Fatal("native export of the cell-record table does not reproduce the source file")
	}
	t.Logf("records=%d baseline nonzero=%d ground=%d air=%d building=%d; LOAD and native export both reproduce the source file exactly",
		len(want), nzBase, nzGround, nzAir, nzBuilding)
}

// TestReleaseOriginalCellRecordsDiagnosticFieldsSurviveLoad1131 covers the
// three fields no reachable owner save carries nonzero (docs/1131/story.md,
// Open debt): LayerCount and the two constructor-zero residue bytes, on
// TestReleaseOriginalSessionRawSpansSurviveActorStockStaging1130's own
// diagnostic-mutation precedent for RawSessionMid. This writes one explicitly
// named byte pattern into one record's own LayerCount/Residue0/Residue1
// before LOAD — not a claim about original runtime content, only a probe of
// the carry-through mechanism (applyOriginalCellRecords, ImportOriginalCellRecords,
// and ImportOriginalLivingActors' own staging round trip, the exact route this
// story found silently dropping the whole saved-cell-record span, docs/1131/story.md).
func TestReleaseOriginalCellRecordsDiagnosticFieldsSurviveLoad1131(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game9999.sav", "5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileCells, present, err := source.Cells()
	if err != nil || !present {
		t.Fatalf("Cells: present=%v err=%v", present, err)
	}
	// Pick a record that also carries real identity-slot content, so the
	// mutated record shows the diagnostic fields survive alongside real
	// Ground/Air content rather than in isolation. Prefer a record with
	// both set; this fixture's own Ground and Air keys never land on the
	// same cell, so fall back to Ground alone, then Air alone, then any
	// record, rather than requiring a combination this file does not carry.
	idx := -1
	for _, want := range []func(sav.Cell) bool{
		func(c sav.Cell) bool { return c.Ground != 0 && c.Air != 0 },
		func(c sav.Cell) bool { return c.Ground != 0 },
		func(c sav.Cell) bool { return c.Air != 0 },
		func(sav.Cell) bool { return true },
	} {
		for i, c := range fileCells {
			if want(c) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			break
		}
	}
	if idx < 0 {
		t.Fatal("fixture carries no cell records at all")
	}
	original := fileCells[idx]
	mutated := original
	mutated.LayerCount, mutated.Residue0 = 5, 0x11
	mutated.Residue1 = [2]byte{0x22, 0x33}
	if err := source.SetCell(idx, mutated); err != nil {
		t.Fatalf("SetCell: %v", err)
	}
	changed := source.Marshal()
	reopened, err := sav.Open(changed)
	if err != nil {
		t.Fatal(err)
	}
	reopenedCells, present, err := reopened.Cells()
	if err != nil || !present {
		t.Fatalf("Cells (reopened): present=%v err=%v", present, err)
	}
	if got := reopenedCells[idx]; got.LayerCount != 5 || got.Residue0 != 0x11 || got.Residue1 != ([2]byte{0x22, 0x33}) {
		t.Fatalf("SetCell fixture setup did not take: %+v", got)
	}
	if got := reopenedCells[idx]; got.Ground != original.Ground || got.Air != original.Air || got.Building != original.Building ||
		got.BaselineCost != original.BaselineCost || got.BaselineStatic != original.BaselineStatic {
		t.Fatal("SetCell mutation disturbed a sibling field it was not meant to touch")
	}
	key := original.Key

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1131 cell record diagnostics")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	var live sim.SavedCellRecord
	found := false
	for _, rec := range f.live.world.SavedCellRecords() {
		if rec.Cell == key {
			live, found = rec, true
			break
		}
	}
	if !found {
		t.Fatalf("cell %#04x missing from the live world's saved cell records", key)
	}
	if live.LayerCount != 5 || live.Residue0 != 0x11 || live.Residue1 != ([2]byte{0x22, 0x33}) {
		t.Fatalf("LOAD diagnostic fields = layer=%d residue0=%#x residue1=%v, want layer=5 residue0=0x11 residue1=[0x22 0x33] (must survive ImportOriginalLivingActors' own staging, docs/1131/story.md)",
			live.LayerCount, live.Residue0, live.Residue1)
	}
	if live.Ground.Key != original.Ground || live.Air.Key != original.Air {
		t.Fatalf("LOAD Ground/Air keys = %#x/%#x, want %#x/%#x: diagnostic mutation must not disturb a sibling identity slot", live.Ground.Key, live.Air.Key, original.Ground, original.Air)
	}

	target, err := sav.Open(changed)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalCellRecords(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalCellRecords: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	off, size := target.World.CellRecDataOff, target.World.CellRecCount*54
	if !bytes.Equal(written.Body[off:off+size], reopened.Body[off:off+size]) {
		t.Fatal("native export of the cell-record table does not reproduce the diagnostic fixture")
	}
	t.Logf("cell %#04x: LayerCount/Residue0/Residue1 diagnostic mutation carried through LOAD, actor-stock staging, and native export", key)
}
