package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// A draft that deletes a mod file under another letter case removes it from
// the inventory as well as from the project list.
func TestDraftDeleteUnderAnotherCaseLeavesTheInventory(t *testing.T) {
	s, dir := modSpeakerProject(t)
	speakers := filepath.Join(dir, filepath.FromSlash(mod.SpeakersFile))
	if err := os.MkdirAll(filepath.Dir(speakers), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(speakers, []byte("# speakers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree, err := mod.OpenTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(mod.SpeakersFile)
	drafts := map[string][]byte{upper: nil}
	m := &editorMod{}
	if err := m.readInventory(tree, drafts); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.files, mod.SpeakersFile) {
		t.Fatalf("inventory keeps the file the draft deletes: %q", m.files)
	}
	s.inspector.mod = m
	if files := s.projectFiles(&editorModDocument{Files: drafts}); slices.Contains(files, mod.SpeakersFile) {
		t.Fatalf("project list keeps the file the draft deletes: %q", files)
	}
}

// A draft added under another letter case replaces the mod's file of that
// name; a deleted name added again ends with the added file.
func TestDraftAddFoldsOntoTheBaseNameAndReAddWins(t *testing.T) {
	base := []string{"maps/en/map.alm", "mod.toml"}
	got := projectView(base, map[string][]byte{"Maps/EN/map.alm": {1}}).names
	if !slices.Equal(got, []string{"mod.toml", "Maps/EN/map.alm"}) {
		t.Fatalf("fold add = %q", got)
	}
	got = projectView(base, map[string][]byte{"maps/en/map.alm": nil, "MAPS/en/map.alm": {1}}).names
	if !slices.Equal(got, []string{"mod.toml", "MAPS/en/map.alm"}) {
		t.Fatalf("delete then add = %q", got)
	}
	got = projectView(base, map[string][]byte{"maps/en/map.alm": {1}}).names
	if !slices.Equal(got, base) {
		t.Fatalf("exact add = %q", got)
	}
}
