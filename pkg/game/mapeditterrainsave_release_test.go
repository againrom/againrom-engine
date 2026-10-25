package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// An installed map opened in the window and saved without a stroke is written
// as the installed bytes.
func TestReleaseInstalledMapsSaveUnchangedWithoutAStroke(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := t.TempDir()
	for n, key := range installedMapKeys(x) {
		want, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		e.Layout(1280, 800)
		target := filepath.Join(dir, "m"+string(rune('a'+n/26))+string(rune('a'+n%26))+".alm")
		_ = e.Key("saveas")
		e.TextInput(target)
		if err := e.Key("enter"); err != nil {
			t.Fatalf("%s: Save As: %v", key, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want.SourceBytes) {
			t.Errorf("%s: saved %d bytes differ from the installed %d", key, len(got), len(want.SourceBytes))
		}
	}
}

// Strokes on an installed map are one undo step each with exact bytes, and the
// report runs on the painted map without an error finding.
func TestReleaseTerrainAndHeightStrokesUndoExactlyAndReport(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys := installedMapKeys(x)
	for _, key := range []string{keys[0], keys[len(keys)/2], keys[len(keys)-1]} {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		e.Layout(1280, 800)
		installed := append([]byte(nil), e.Document().SourceBytes...)
		mid := image.Pt(e.Document().Width/2, e.Document().Height/2)
		e.ShowCell(mid)
		brushWindow(t, e, "terrain", paletteRow(t, mapedit.ClassMountain))
		pointerStroke(e, []image.Point{mid, mid.Add(image.Pt(4, 0))})
		terrain := append([]byte(nil), e.Document().SourceBytes...)
		if bytes.Equal(terrain, installed) {
			t.Fatalf("%s: a mountain stroke changed nothing", key)
		}
		for _, f := range e.Document().Edits.(ui.InspectionChecker).Report() {
			if f.Error {
				t.Errorf("%s: painted map reports an error: %s", key, f.Line())
			}
		}
		brushWindow(t, e, "height", 0)
		pointerStroke(e, []image.Point{mid.Add(image.Pt(0, 6))})
		both := append([]byte(nil), e.Document().SourceBytes...)
		if bytes.Equal(both, terrain) {
			t.Fatalf("%s: a height stroke changed nothing", key)
		}
		for i, want := range [][]byte{terrain, installed} {
			if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, want) {
				t.Fatalf("%s: undo %d is not byte exact: %v", key, i, err)
			}
		}
		if e.Dirty() {
			t.Errorf("%s: two undos leave the map dirty", key)
		}
		for i, want := range [][]byte{terrain, both} {
			if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, want) {
				t.Fatalf("%s: redo %d is not byte exact: %v", key, i, err)
			}
		}
	}
}
