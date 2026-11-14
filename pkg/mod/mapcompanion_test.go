package mod

import (
	"slices"
	"testing"
)

func TestMapCompanionsFoldThePagesFolderInTheFolderAdd(t *testing.T) {
	got := FolderMaps([]string{"Stem.alm", "stem.TEXT/en/title.txt", "stem.variables", "stemx.text/en/title.txt"})
	if len(got) != 1 {
		t.Fatalf("maps = %+v, want the one map Stem.alm", got)
	}
	stem := got[0]
	if stem.Name != "Stem.alm" || len(stem.Files) != 3 {
		t.Fatalf("Stem.alm carries %v, want its map, the pages file and the labels; stemx.text is not a companion", stem.Files)
	}
	if !slices.Contains(stem.Files, "stem.TEXT/en/title.txt") {
		t.Fatalf("Stem.alm does not carry stem.TEXT/en/title.txt: %v", stem.Files)
	}
}

func TestMapCompanionsAreLabelsAndPagesOnly(t *testing.T) {
	p := MissionPlace{Map: "maps/stem.alm"}
	for f, want := range map[string]bool{
		"maps/STEM.variables":        true,
		"maps/stem.text/en/a.txt":    true,
		"maps/stem.text":             false,
		"maps/stem.almx":             false,
		"maps/stem.alm":              false,
		"maps/stem2.text/en/a.txt":   false,
		"maps/stem.variables.bak":    false,
		"other/maps/stem.text/x.txt": false,
	} {
		if got := p.Companion(f); got != want {
			t.Errorf("Companion(%q) = %v, want %v", f, got, want)
		}
	}
}
