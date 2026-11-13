package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// writeEditorFrame draws the editor headless, checks that the window's text
// overlay keeps every glyph outside cover, the hover lines, the open menu's
// list and a window the chrome drew after the glyph, and writes the frame as a
// PNG to path when write is set. A glyph its own window's parts hide fails.
func writeEditorFrame(t *testing.T, e *ui.MapEditor, path string, write bool, cover image.Rectangle) {
	t.Helper()
	screen := ebiten.NewImage(1280, 800)
	e.Draw(screen)
	screen.Dispose()
	var menu image.Rectangle
	for n := 0; ; n++ {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlMenuRow, N: n})
		if !ok {
			break
		}
		menu = menu.Union(r)
	}
	covers := append(e.PopupTipRects(), cover, menu)
	windows := e.ChromeWindows()
	for i, ft := range e.TextFates() {
		if ft.Fate == textsmooth.Hidden && (slices.ContainsFunc(covers, image.Pt(ft.Call.X, ft.Call.Y).In) || underLaterWindow(ft.Call, i, windows)) {
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
	witness.PNG(t, path, pic)
}

// underLaterWindow reports whether every painted cell of glyph call c, the
// frame's text call i, that its clip shows lies in a window the chrome drew
// after the call.
func underLaterWindow(c text.DrawCall, i int, windows []ui.ChromeWindow) bool {
	g := c.Glyph
	if g == nil || g.Width <= 0 {
		return false
	}
	cells := 0
	for k, p := range g.Pixels[:min(len(g.Pixels), g.Width*g.Height)] {
		at := image.Pt(c.X+k%g.Width, c.Y+k/g.Width)
		if !p.Painted || !c.Clip.Empty() && !at.In(c.Clip) {
			continue
		}
		if !slices.ContainsFunc(windows, func(w ui.ChromeWindow) bool { return w.Text > i && at.In(w.Rect) }) {
			return false
		}
		cells++
	}
	return cells > 0
}

// TestReleaseMapEditorOwnerFrames writes the editor's popups, rendered
// headless from the install, as PNGs into AGAINROM_EDITOR_FRAMES: the frames
// the owner looks at. Without that directory it only drives the same steps.
// In each state the window's text overlay keeps every glyph of the installed
// fonts: none is undecided or left baked, and only text under an open
// drop-down's list or a cut row's hover line is hidden.
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
			if string(r.Kind) == kind {
				return i
			}
		}
		t.Fatalf("the map has no %s", kind)
		return -1
	}

	shot(fresh(), "map-picker", image.Rectangle{})

	e := opened()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "file"})
	click(e, r, ok, "the File entry")
	shot(e, "file-menu", image.Rectangle{})

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
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: choice})
	click(e, r, ok, "the unit's choice field")
	shot(e, "unit-dropdown", image.Rect(r.Min.X, r.Max.Y, 1280, 800)) // the list lies below the field

	e = opened()
	e.Select(first(e, "Structure"), true)
	shot(e, "structure", image.Rectangle{})

	// The left panel shows before a category entry, as F6 shows it: the entry
	// leaves the panels as they are and its palette opens beside them.
	e = opened()
	showTree(e)
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "groups"})
	click(e, r, ok, "the Groups entry")
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlGroupsRow, N: 2, K: 15, Flag: false})
	click(e, r, ok, "the second group")
	if len(e.GroupMembers()) == 0 {
		t.Fatal("the second group has no members")
	}
	shot(e, "groups", image.Rectangle{})

	e = opened()
	showTree(e)
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "scripts"})
	click(e, r, ok, "the Scripts entry")
	r, ok = treeMemberRect(e, "scripts", 0)
	click(e, r, ok, "the first trigger")
	shot(e, "triggers", image.Rectangle{})

	e = editorChainReleaseProject(t)
	e.Layout(640, 480)
	// The mod window opens over the Map panel's rows; its frames show the
	// window alone, as the other frames do. The project's set-up leaves the
	// mod window open, and Esc closes it before the panel.
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	closeMapPanel(t, e)
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	shot(e, "mod-window-small", image.Rectangle{})
	// The mod's maps are W's last page; the window's key turns on to its first.
	modMapsPage(t, e)
	shot(e, "mod-maps-small", image.Rectangle{})
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("source") {
		t.Fatal("missing mission choice")
	}
	_ = e.Key("clear")
	e.TextInput("2")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("party.1.weapon") {
		t.Fatal("missing member equipment")
	}
	shot(e, "mod-party-small", image.Rectangle{})
	publishEditorFixture(t, e, t.TempDir())
	e.FocusModField("publish")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	shot(e, "mod-replace-small", image.Rectangle{})

	// A mod that holds terrain only opens on its overview page.
	e = x.Editor()
	e.Layout(640, 480)
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	if err := workspaceField(e, "package", 0, snowFieldMod(t, nil)); err != nil {
		t.Fatal(err)
	}
	if !e.ModWindowOpen() || !slices.Contains(e.PopupLines(), "terrain: "+mod.TerrainFile) {
		t.Fatalf("W does not show the terrain mod's overview: %q", e.PopupLines())
	}
	shot(e, "mod-overview-small", image.Rectangle{})
}
