package game

import (
	"slices"
	"testing"

	"againrom/pkg/mod"
)

func TestMapDraftNamesFollowTheCompanionRule(t *testing.T) {
	d := &editorModDocument{Files: map[string][]byte{}}
	for _, n := range []string{
		"maps/Crypt.alm", "maps/crypt.variables", "maps/Crypt.Variables",
		"maps/crypt.text/en/a.txt", "maps/Crypt.TEXT/en/b.txt", "maps/crypt.text/en/deep/c.txt", "maps/crypt.text/ru/d.txt",
		"extra/e.txt", "extra/deep/f.txt", "maps/other.variables", "maps/crypt.textual/g.txt",
	} {
		d.Files[n] = []byte("x")
	}
	got := (&MapInspector{}).mapDraftNames(d, mod.Mission{Map: "maps/Crypt.alm", Texts: "Extra"})
	want := []string{
		"extra/e.txt", "maps/Crypt.Variables", "maps/Crypt.TEXT/en/b.txt", "maps/Crypt.alm",
		"maps/crypt.text/en/a.txt", "maps/crypt.text/en/deep/c.txt", "maps/crypt.text/ru/d.txt", "maps/crypt.variables",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("drafts %v, want %v", got, want)
	}
}
