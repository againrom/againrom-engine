package game

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReleaseNativeResumeCarriesSpellEffectsAndCellRecords1139 is story
// 1139's own full-Go release witness, on originalholdings_release_test.go's
// own holdingsNativeFresh shape: original LOAD, ordinary App menu SAVE, then
// a fresh App LOAD of that native .ags in a brand-new FrontEnd/App/World —
// the exact player-visible cycle Form85 exists for
// (pkg/sim/carriedresumebinary.go). holdingsNativeFresh's own Hash() check
// already proves the whole world matches bit for bit across that cycle; this
// witness additionally names two of the five Form85 fields explicitly, by
// count, so a future regression here fails on a legible structure rather
// than only an opaque hash mismatch. TestNativeResumeCorpusCycle1139
// (nativeresumecycle1139_corpus_test.go) is the exhaustive, byte-identical
// counterpart over every reachable corpus file; this is the one full-Go
// witness driven through the production App SAVE/LOAD doors themselves,
// over both lawful installs via pipeline/check-release-tests.sh.
//
// game0018.sav is SAV-EFFECTGRAPH-366's own witnessed subject
// (originalspelleffects1132_release_test.go): a non-empty top-level
// SpellEffect list. Every world-half save carries a cell-record residue
// entry per stored file cell (originalcellrecords1131_release_test.go), so
// the same fixture also covers a non-empty cell-record residue with no
// second fixture needed.
func TestReleaseNativeResumeCarriesSpellEffectsAndCellRecords1139(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-15/game0018.sav",
		"1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1139 native resume")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0018.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0018.sav")

	beforeEffects := len(f.live.world.SavedSpellEffects())
	beforeCells := len(f.live.world.SavedCellRecords())
	if beforeEffects == 0 {
		t.Fatal("fixture carries an empty SpellEffect list; this witness requires the known non-empty one")
	}
	if beforeCells == 0 {
		t.Fatal("fixture carries no cell-record residue at all")
	}

	// The cycle itself: ordinary App menu SAVE (native .ags), then a fresh
	// FrontEnd/App/World LOAD of that exact file — holdingsNativeFresh's own
	// shape (originalholdings_release_test.go), including its Hash()
	// equality check across the whole cycle.
	fresh, _ := holdingsNativeFresh(t, f, app, store, nil)

	afterEffects := len(fresh.live.world.SavedSpellEffects())
	afterCells := len(fresh.live.world.SavedCellRecords())
	if afterEffects != beforeEffects {
		t.Fatalf("SpellEffect count after native SAVE/LOAD = %d, want %d (pre-save value)", afterEffects, beforeEffects)
	}
	if afterCells != beforeCells {
		t.Fatalf("cell-record residue count after native SAVE/LOAD = %d, want %d (pre-save value)", afterCells, beforeCells)
	}
	t.Logf("SpellEffect count=%d cell-record residue count=%d: both equal before and after the native SAVE/LOAD cycle", beforeEffects, beforeCells)
}
