package ui

import (
	"image"
	"strings"
	"testing"
	"time"
)

// dragFadeEditor is the modes editor at 1920x1080 with unit 0 selected and
// its object window standing over map cells, away from unit 0.
func dragFadeEditor(t *testing.T) (*MapEditor, *modesCalls, popupLayout) {
	t.Helper()
	e, calls := newModesEditor(t, "full")
	e.Layout(1920, 1080)
	e.Fit()
	e.places[popObject].at, e.places[popObject].placed = e.CellPoint(image.Pt(26, 20)), true
	if !e.Select(0, false) {
		t.Fatal("unit 0 does not select")
	}
	s, _, ok := e.objectSpec()
	if !ok || e.panels.hosted {
		t.Fatal("no floating object window")
	}
	return e, calls, layoutPopup(s, e.font, e.bodyFont())
}

func dragFadePress(e *MapEditor, p image.Point, down bool) {
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: down}, time.Unix(1, 0))
}

// cellUnder is the map cell under window point p, as a press reads it.
func cellUnder(e *MapEditor, p image.Point) image.Point {
	x, y, _ := e.doc.Viewer.groundCellAt(float64(p.X-e.mapArea().Min.X), float64(p.Y))
	return image.Pt(x, y)
}

func maxAlpha(pic *image.RGBA, r image.Rectangle) (most uint8) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			most = max(most, pic.RGBAAt(x, y).A)
		}
	}
	return most
}

func sameRect(a, b *image.RGBA, r image.Rectangle) bool { return diffBox(a, b, r).Empty() }

// diffBox is the smallest rectangle holding every pixel of r where a and b
// differ.
func diffBox(a, b *image.RGBA, r image.Rectangle) image.Rectangle {
	var box image.Rectangle
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

// While a moved drag is on, the floating object window, its text included,
// draws at the fade opacity; the bar, the strip and the docked panels keep
// their pixels.
func TestEditorMovedDragFadesFloatingWindows(t *testing.T) {
	e, _, l := dragFadeEditor(t)
	e.showPanel(panelLeft, true)
	still := e.ChromeImage()
	win := l.Rect
	if maxAlpha(still, win) != 255 {
		t.Fatal("the window is not opaque at the start")
	}
	start := e.CellPoint(image.Pt(10, 10))
	dragFadePress(e, start, true)
	above := image.Rect(0, 0, e.window.X, e.statusRect().Min.Y)
	pressed := e.ChromeImage()
	if !sameRect(pressed, still, above) {
		t.Fatalf("a press without a move changed the chrome at %v", diffBox(pressed, still, above))
	}
	dragFadePress(e, e.CellPoint(image.Pt(14, 12)), true)
	if !e.dragFades() {
		t.Fatal("the drag has not moved")
	}
	faded := e.ChromeImage()
	most := maxAlpha(faded, win)
	if most == 0 || float64(most) > 255*editorDragFadeAlpha+1 {
		t.Fatalf("window alpha %d during the drag, want about %.0f", most, 255*editorDragFadeAlpha)
	}
	keep := []image.Rectangle{image.Rect(0, 0, e.window.X, e.barHeight()), e.statusRect()}
	keep = append(keep, e.panelRects()...)
	// The same state drawn without the fade: only the floating windows differ.
	e.recordDrag.moved = false
	plain := e.ChromeImage()
	e.recordDrag.moved = true
	for _, r := range keep {
		if !sameRect(faded, plain, r) {
			t.Fatalf("the chrome changed at %v during the drag", r)
		}
	}
	if len(e.panelRects()) == 0 {
		t.Fatal("no docked panel in the test")
	}
}

// The release over a faded window puts the object on the cell under the
// pointer, the window takes no press and opens no tip, and the window is
// opaque again after the release.
func TestEditorReleaseOverFadedWindowPlacesOnTheMap(t *testing.T) {
	e, calls, l := dragFadeEditor(t)
	at := l.Close.Min.Add(image.Pt(2, 2))
	dragFadePress(e, e.CellPoint(image.Pt(10, 10)), true)
	dragFadePress(e, e.CellPoint(image.Pt(14, 12)), true)
	dragFadePress(e, at, true)
	if e.TipShown() != "" || maxAlpha(e.ChromeImage(), l.Close) > 40 {
		t.Fatal("the faded window shows a tip or a hover")
	}
	want := cellUnder(e, at).Sub(cellUnder(e, e.CellPoint(image.Pt(10, 10))).Sub(image.Pt(10, 10)))
	dragFadePress(e, at, false)
	if len(calls.lines) != 1 || !strings.HasPrefix(calls.lines[0], "Move Unit 0 ") || !strings.HasSuffix(calls.lines[0], want.String()) {
		t.Fatalf("the release wrote %v, want a move to %v", calls.lines, want)
	}
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the release pressed the window's close button")
	}
	if maxAlpha(e.ChromeImage(), l.Rect) != 255 || e.dragFades() {
		t.Fatal("the window stays faded after the release")
	}
}

// A cancelled drag brings the window back.
func TestEditorCancelledDragRestoresWindows(t *testing.T) {
	e, _, l := dragFadeEditor(t)
	still := e.ChromeImage()
	dragFadePress(e, e.CellPoint(image.Pt(10, 10)), true)
	dragFadePress(e, e.CellPoint(image.Pt(14, 12)), true)
	if maxAlpha(e.ChromeImage(), l.Rect) == 255 {
		t.Fatal("the window is not faded during the drag")
	}
	e.cancelRecordDrag()
	above := image.Rect(0, 0, e.window.X, e.statusRect().Min.Y)
	if got := e.ChromeImage(); !sameRect(got, still, above) {
		t.Fatalf("the chrome differs from the start after the cancel at %v", diffBox(got, still, above))
	}
}

// With smoothed text on, the faded windows keep their text as plain pixels:
// no smoothed glyph is drawn inside them and the picture the window shows is
// faded.
func TestEditorMovedDragFadesSmoothedWindowText(t *testing.T) {
	e, _, l := dragFadeEditor(t)
	e.textSmoothingEnabled = true
	dragFadePress(e, e.CellPoint(image.Pt(10, 10)), true)
	dragFadePress(e, e.CellPoint(image.Pt(14, 12)), true)
	pic, blit, _, kept := e.chromePicture()
	for _, c := range kept {
		if image.Pt(c.X, c.Y).In(l.Rect) {
			t.Fatalf("a smoothed glyph at %d,%d stays at full opacity in the faded window", c.X, c.Y)
		}
	}
	for _, p := range []*image.RGBA{pic, blit} {
		if most := maxAlpha(p, l.Rect); most == 0 || most > 40 {
			t.Fatalf("window alpha %d with smoothed text", most)
		}
	}
}
