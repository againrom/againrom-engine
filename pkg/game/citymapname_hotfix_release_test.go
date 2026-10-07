package game

import (
	"testing"
)

// saveReloadTownMapName saves the town through the ordinary save seam, loads
// the file into a fresh front end, and returns the map name that front end's
// next town SAV writes.
func saveReloadTownMapName(t *testing.T, f *FrontEnd) string {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("town save = %q, %v; want a SAV", name, err)
	}
	restored := releaseFront(t)
	_, list, load := restored.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	entries := list()
	if len(entries) != 1 {
		t.Fatalf("save list %+v, want one entry", entries)
	}
	if _, town, err := load(entries[0].Name); err != nil || !town {
		t.Fatalf("load the town SAV: town %v err %v", town, err)
	}
	return writtenTownData(t, restored).MapName
}

// A town SAV names the map of the last mission played, as every pure original
// town in the lawful corpus does. A loaded original town keeps the name its
// file holds until a mission replaces it, and a reload of the written file
// writes the same name again.
func TestReleaseCitySAVNamesTheLoadedTownsMap(t *testing.T) {
	f, _ := openOriginalSalesTown(t)
	if got := writtenTownData(t, f).MapName; got != "31.alm" {
		t.Fatalf("town SAV map name %q, want the loaded 31.alm", got)
	}
	if got := saveReloadTownMapName(t, f); got != "31.alm" {
		t.Fatalf("reloaded town SAV map name %q, want 31.alm", got)
	}
}

// After mission 20 ends and the party returns to town, the town SAV names
// that mission's map.
func TestReleaseCitySAVNamesTheMissionJustPlayed(t *testing.T) {
	f := currentTown(t, nil, nil)
	if got := writtenTownData(t, f).MapName; got != "20.alm" {
		t.Fatalf("town SAV map name %q after mission 20, want 20.alm", got)
	}
	if got := saveReloadTownMapName(t, f); got != "20.alm" {
		t.Fatalf("reloaded town SAV map name %q, want 20.alm", got)
	}
}

// A native town reached before any mission has no mission to name and writes
// an empty map name (DIV-1412); the written file loads.
func TestReleaseCitySAVBeforeAnyMissionNamesNoMap(t *testing.T) {
	f := releaseFront(t)
	reachabilityWalkArrive(t, f, "Map Hero")
	if got := writtenTownData(t, f).MapName; got != "" {
		t.Fatalf("town SAV map name %q before any mission, want empty", got)
	}
	if got := saveReloadTownMapName(t, f); got != "" {
		t.Fatalf("reloaded town SAV map name %q, want empty", got)
	}
}
