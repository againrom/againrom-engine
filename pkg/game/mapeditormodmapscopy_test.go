package game

import (
	"path/filepath"
	"strings"
	"testing"
)

// The editor's copy of a handwritten mod, made while a map no mission names
// is shown, leaves that tab holding the map's file in the copy: the list's
// Open on the map shows the same tab and makes no second one over the file.
func TestCopyOfAModKeepsTheUnusedMapsTab(t *testing.T) {
	_, e := workspaceFixture(t)
	e.Layout(1280, 800)
	raw := e.Document().SourceBytes
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='m1'\nnumber=21\nmap='map1.alm'\nafter=20\nparty=[{character='npc22'}]\n")
	for _, name := range []string{"map1.alm", "maps/spare.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), raw)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	spare := func() int {
		t.Helper()
		for n, c := range modMapsField(t, e).Choices {
			if strings.HasPrefix(c.Label, "spare.alm") {
				return n + 1
			}
		}
		t.Fatalf("no row of spare.alm in %s", modMapsRows(t, e))
		return 0
	}
	if err := modMapsOpen(t, e, spare()); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	if err := workspaceField(e, "package", 0, own); err != nil {
		t.Fatal(err)
	}
	tab := e.Document().Edits.(*mapInspectionEdits)
	if want := filepath.Join(own, "maps", "spare.alm"); !sameSaveName(tab.source, want) {
		t.Fatalf("the tab holds %s, not the copy's %s", tab.source, want)
	}
	maps := e.OpenMapCount()
	if err := modMapsOpen(t, e, spare()); err != nil {
		t.Fatal(err)
	}
	if e.OpenMapCount() != maps || e.Document().Edits.(*mapInspectionEdits) != tab {
		t.Fatalf("the list's Open on the shown map: %d maps open, was %d", e.OpenMapCount(), maps)
	}
}
