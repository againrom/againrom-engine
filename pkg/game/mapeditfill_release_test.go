package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// Buttons of the brush list's centre fill row.
const (
	fillPattern = iota
	fillRandom
	_ // minus
	fillPlus
)

// pressFill presses button k of the brush list's fill row n times.
func pressFill(t *testing.T, e *ui.MapEditor, k, n int) {
	t.Helper()
	for range n {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlListButton, N: 2, K: k})
		pressRect(t, e, r, ok, "a fill button")
	}
}

// fillScene opens the map, finds a pure land area and returns the editor, its
// centre and the Grass brush at a wide square size, with the fill row on the
// pattern.
func fillScene(t *testing.T, x *MapInspector) (*ui.MapEditor, image.Point) {
	t.Helper()
	e := openBrushEditor(t, x, "scenario/10.alm")
	_ = e.Key("terrain")
	at, _ := area(t, e, mapedit.ClassLand, 12)
	setBrush(t, e, mapedit.ClassGrass, 8, 1)
	e.ShowCell(at)
	pressFill(t, e, fillPattern, 1)
	return e, at
}

// TestReleaseMapEditorFillFrames draws the owner's frames of the centre fill and
// of two windows before and after a resize into AGAINROM_FILL_FRAMES when set;
// without it the frames are drawn and checked, not written.
func TestReleaseMapEditorFillFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_FILL_FRAMES")
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{})
	}
	e, at := fillScene(t, x)
	shot(e, "fill-pattern-before")
	pointerStroke(e, []image.Point{at})
	shot(e, "fill-pattern")
	_ = e.Key("undo")
	pressFill(t, e, fillPlus, 3)
	pointerStroke(e, []image.Point{at})
	shot(e, "fill-variant-3")
	_ = e.Key("undo")
	pressFill(t, e, fillRandom, 1)
	for _, name := range []string{"fill-random-a", "fill-random-b"} {
		pointerStroke(e, []image.Point{at})
		shot(e, name)
		_ = e.Key("undo")
	}

	// Two windows before and after a resize: the brush list and the record list.
	e.ShowCell(at)
	shot(e, "window-brush-before")
	drag := func(pick func(r image.Rectangle) image.Point, d image.Point) {
		t.Helper()
		borderDrag(t, e, pick, d)
	}
	corner := func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-2, r.Max.Y-2) }
	drag(corner, image.Pt(120, 140))
	shot(e, "window-brush-after")
	_ = e.Key("escape")
	_ = e.Key("structure")
	shot(e, "window-records-before")
	drag(corner, image.Pt(140, -90))
	shot(e, "window-records-after")
	if dir != "" {
		t.Logf("frames in %s", dir)
	}
}
