package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestReleaseMapEditorCloseWithUnsavedChangesAsks copies an installed map
// into AGAINROM_EDITOR_FRAMES (a temporary directory without it), makes an
// edit, and closes the editor and opens another map. Each asks one question
// whose frame keeps every glyph of the installed fonts; Close without saving
// ends the editor and leaves the copy's bytes and the directory as they were.
func TestReleaseMapEditorCloseWithUnsavedChangesAsks(t *testing.T) {
	x := installedInspector(t)
	_, maps := installedMapBytes(t, x)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	write := dir != ""
	if !write {
		dir = t.TempDir()
	}
	work := filepath.Join(dir, "leave-copy")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(work, "a.alm"), filepath.Join(work, "b.alm")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, maps[0], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	listing := func() string {
		entries, err := os.ReadDir(work)
		if err != nil {
			t.Fatal(err)
		}
		s := ""
		for _, en := range entries {
			raw, err := os.ReadFile(filepath.Join(work, en.Name()))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			s += fmt.Sprintf("%s:%x;", en.Name(), sum)
		}
		return s
	}
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), write, image.Rectangle{})
	}
	dirty := func() *ui.MapEditor {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(a); err != nil {
			t.Fatal(err)
		}
		if e.Dirty() {
			t.Fatal("a freshly opened copy is dirty")
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		if _, err := s.AddPlayer(); err != nil {
			t.Fatal(err)
		}
		if !e.Dirty() {
			t.Fatal("AddPlayer left the map clean")
		}
		return e
	}
	before := listing()

	e := dirty()
	if err := e.Key("close"); err != nil || e.ExitAsked() {
		t.Fatalf("close with unsaved changes: err %v exit %v", err, e.ExitAsked())
	}
	shot(e, "leave-close")
	_ = e.Key("escape")
	if e.ExitAsked() || !e.Dirty() {
		t.Fatal("Cancel did not return to the editor unchanged")
	}
	_ = e.Key("escape")
	shot(e, "leave-close-escape")
	if err := e.Key("close"); err != ebiten.Termination || !e.ExitAsked() {
		t.Fatalf("the window close button pressed again: err %v exit %v", err, e.ExitAsked())
	}
	if listing() != before {
		t.Fatal("closing without saving changed the copy or the directory")
	}

	e = dirty()
	if err := e.Open(b); err != nil || e.Document().Source != a {
		t.Fatalf("open with unsaved changes: err %v source %q", err, e.Document().Source)
	}
	shot(e, "leave-open")
	if err := e.Key("close"); err != ebiten.Termination {
		t.Fatalf("close while asked to open: %v", err)
	}
	if listing() != before {
		t.Fatal("leaving without saving changed the copy or the directory")
	}

	clean := x.Editor()
	clean.Layout(1280, 800)
	if err := clean.Open(a); err != nil {
		t.Fatal(err)
	}
	if err := clean.Key("close"); err != ebiten.Termination {
		t.Fatalf("clean close: %v", err)
	}
}
