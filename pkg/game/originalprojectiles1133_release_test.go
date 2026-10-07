package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseOriginalProjectilesRestoreOnLoad1133(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileStore, present, err := source.Projectiles()
	if err != nil || !present {
		t.Fatalf("Projectiles: present=%v err=%v", present, err)
	}
	if fileStore.FreeIndex != 267 || len(fileStore.Items) != 1 || fileStore.Items[0].ID != 266 {
		t.Fatalf("fixture missing real content: FreeIndex=%d items=%+v, want FreeIndex=267 one item id=266", fileStore.FreeIndex, fileStore.Items)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1133 projectiles")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0018.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0018.sav")

	got := f.live.world.SavedProjectiles()
	if got.FreeIndex != fileStore.FreeIndex {
		t.Fatalf("LOAD FreeIndex = %#x, want %#x", got.FreeIndex, fileStore.FreeIndex)
	}
	if !reflect.DeepEqual(got.IDs, fileStore.IDs) {
		t.Fatalf("LOAD IDs = %v, want %v", got.IDs, fileStore.IDs)
	}
	if len(got.Items) != 1 || got.Items[0] != savProjectileToSaved(fileStore.Items[0]) {
		t.Fatalf("LOAD Items = %+v, want one entry matching %+v", got.Items, fileStore.Items[0])
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the decoded Projectiles value to
	// reproduce the source file (the whole state-store span does not
	// reproduce byte for byte — originalprojectiles1133_corpus_test.go finds
	// this on every world-half owner save, not particular to this fixture).
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalProjectiles(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalProjectiles: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	exported, exportedPresent, err := written.Projectiles()
	if err != nil || !exportedPresent {
		t.Fatalf("Projectiles (re-decode written): present=%v err=%v", exportedPresent, err)
	}
	if exported.FreeIndex != fileStore.FreeIndex || !reflect.DeepEqual(exported.IDs, fileStore.IDs) ||
		len(exported.Items) != 1 || exported.Items[0] != fileStore.Items[0] {
		t.Fatalf("native export = %+v, want %+v", exported, fileStore)
	}
	t.Logf("FreeIndex=%#x id=%d: LOAD and native export both reproduce the source file's decoded Projectiles value", fileStore.FreeIndex, fileStore.Items[0].ID)
}

// TestReleaseOriginalProjectilesEmptyStoreSurvivesLoad1133 covers the
// corpus' ordinary case: FreeIndex present and nonzero, IDs empty, no
// Prj<id> section.
func TestReleaseOriginalProjectilesEmptyStoreSurvivesLoad1133(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game9999.sav", "5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	fileStore, present, err := source.Projectiles()
	if err != nil || !present {
		t.Fatalf("Projectiles: present=%v err=%v", present, err)
	}
	if len(fileStore.Items) != 0 || len(fileStore.IDs) != 0 {
		t.Fatalf("fixture is not the empty-store case: %+v", fileStore)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1133 projectiles empty")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: dir}, nil))
	_, list, _ := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game9999.sav")

	got := f.live.world.SavedProjectiles()
	if got.FreeIndex != fileStore.FreeIndex {
		t.Fatalf("LOAD FreeIndex = %#x, want %#x", got.FreeIndex, fileStore.FreeIndex)
	}
	if len(got.Items) != 0 || len(got.IDs) != 0 {
		t.Fatalf("LOAD Items/IDs = %+v/%v, want none", got.Items, got.IDs)
	}

	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalProjectiles(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalProjectiles: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	exported, exportedPresent, err := written.Projectiles()
	if err != nil || !exportedPresent {
		t.Fatalf("Projectiles (re-decode written): present=%v err=%v", exportedPresent, err)
	}
	if exported.FreeIndex != fileStore.FreeIndex || len(exported.Items) != 0 || len(exported.IDs) != 0 {
		t.Fatalf("native export = %+v, want FreeIndex=%#x and no items/IDs", exported, fileStore.FreeIndex)
	}
	t.Logf("FreeIndex=%#x, no live projectiles: LOAD and native export both reproduce the source file's decoded Projectiles value", fileStore.FreeIndex)
}
