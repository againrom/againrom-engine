package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseMilestone2TerrainCells(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0")
	source, err := sav.Open(raw)
	if err != nil || source.World == nil {
		t.Fatalf("world fixture: %v", err)
	}
	blocks, cells, err := terrainCellsRaw(source.Body, source.World.BlocksOff)
	if err != nil || len(blocks) == 0 || len(cells) == 0 {
		t.Fatalf("empty or invalid terrain fixture: %v", err)
	}
	check := func(w *sim.World) {
		t.Helper()
		planes, _ := w.SavedCellPlanes()
		_, baselines, present := w.SavedStructures()
		diffs := terrainAcceptanceDifferences(blocks, planes)
		diffs = append(diffs, cellAcceptanceDifferences(cells, baselines, w.SavedCellRecords(), present)...)
		if len(diffs) != 0 {
			t.Fatal(diffs)
		}
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	check(ms.World)
	f.SetDeterministicFrames(true)
	app := f.App("terrain and cell acceptance")
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	check(f.live.world)
	driver := f.live
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
	check(fresh.live.world)
	initial := driver.world.Tick()
	for i := 0; i < 20; i++ {
		driver.tick()
		fresh.live.tick()
		if driver.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("native continuation differs at step %d", i)
		}
	}
	if driver.world.Tick() == initial {
		t.Fatal("continuation did not advance")
	}
	t.Logf("%d raw terrain deltas and %d final cell keys agree on both original LOAD doors and ordinary SAVE/fresh LOAD; 20 continuation hashes agree", len(blocks), len(cellAcceptanceFinal(cells)))
}
