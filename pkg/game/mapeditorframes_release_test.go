package game

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// writeEditorFrame draws the editor headless, checks that the window's text
// overlay keeps every glyph of the installed fonts outside cover, and writes
// the frame as a PNG to path when write is set.
func writeEditorFrame(t *testing.T, e *ui.MapEditor, path string, write bool, cover image.Rectangle) {
	t.Helper()
	screen := ebiten.NewImage(1280, 800)
	e.Draw(screen)
	screen.Dispose()
	for _, ft := range e.TextFates() {
		if ft.Fate == textsmooth.Hidden && image.Pt(ft.Call.X, ft.Call.Y).In(cover) {
			continue
		}
		if ft.Fate != textsmooth.Kept && ft.Fate != textsmooth.Blank && ft.Fate != textsmooth.Repeat {
			t.Errorf("%s: glyph at %d,%d is %s", filepath.Base(path), ft.Call.X, ft.Call.Y, ft.Fate)
		}
	}
	pic, _, err := e.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if !write {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, pic); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestReleaseMapEditorOwnerFrames writes the editor's popups, rendered
// headless from the install, as PNGs into AGAINROM_EDITOR_FRAMES: the frames
// the owner looks at. Without that directory it only drives the same steps.
// In each state the window's text overlay keeps every glyph of the installed
// fonts: none is undecided or left baked, and only text under an open
// drop-down's list is hidden.
func TestReleaseMapEditorOwnerFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	now := time.Unix(1, 0)
	click := func(e *ui.MapEditor, r image.Rectangle, ok bool, what string) {
		t.Helper()
		if !ok {
			t.Fatalf("%s is not shown", what)
		}
		p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
		now = now.Add(time.Second)
	}
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", cover)
	}
	fresh := func() *ui.MapEditor {
		e := x.Editor()
		e.Layout(1280, 800)
		return e
	}
	opened := func() *ui.MapEditor {
		e := fresh()
		if err := e.Open("scenario/10.alm"); err != nil {
			t.Fatal(err)
		}
		closeMapPanel(t, e)
		return e
	}
	first := func(e *ui.MapEditor, kind string) int {
		for i, r := range e.Document().Records {
			if r.Kind == kind {
				return i
			}
		}
		t.Fatalf("the map has no %s", kind)
		return -1
	}

	shot(fresh(), "map-picker", image.Rectangle{})

	e := opened()
	r, ok := e.BarRect("map")
	click(e, r, ok, "the Map entry")
	shot(e, "map-menu", image.Rectangle{})

	e = opened()
	unit := first(e, "Unit")
	e.Select(unit, true)
	rec := e.Document().Records[unit]
	choice := -1
	for i, f := range objectEdits(t, e).Fields(rec.Kind, rec.Index) {
		if len(f.Choices) > 0 {
			choice = i
			break
		}
	}
	if choice < 0 {
		t.Fatal("the unit has no choice field")
	}
	r, ok = e.FieldRect(choice)
	click(e, r, ok, "the unit's choice field")
	shot(e, "unit-dropdown", image.Rect(r.Min.X, r.Max.Y, 1280, 800)) // the list lies below the field

	e = opened()
	e.Select(first(e, "Structure"), true)
	shot(e, "structure", image.Rectangle{})

	e = opened()
	r, ok = e.BarRect("groups")
	click(e, r, ok, "the Groups entry")
	r, ok = e.GroupsRowRect(2, 15, false)
	click(e, r, ok, "the second group")
	if len(e.GroupMembers()) == 0 {
		t.Fatal("the second group has no members")
	}
	shot(e, "groups", image.Rectangle{})

	e = opened()
	r, ok = e.BarRect("scripts")
	click(e, r, ok, "the Scripts entry")
	r, ok = e.ListRowRect(1)
	click(e, r, ok, "the first trigger")
	shot(e, "triggers", image.Rectangle{})
}
