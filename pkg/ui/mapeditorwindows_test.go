package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// The Z key turns the passability view on and off, as the View row does.
func TestPassabilityViewHasAKey(t *testing.T) {
	e, _ := brushEditor(t)
	if e.view.passability {
		t.Fatal("the view starts on")
	}
	_ = e.Key("passview")
	if !e.view.passability {
		t.Fatal("Z did not turn the passability view on")
	}
	_ = e.Key("passview")
	if e.view.passability {
		t.Fatal("Z did not turn the passability view off")
	}
	found := false
	for _, r := range editorKeyRows {
		found = found || r.key == "passview" && r.label == "Z"
	}
	if !found {
		t.Fatal("no key row labels the passability view Z")
	}
}

// A heading row paints a band across the row; an ordinary row does not.
func TestPopupHeadingPaintsABand(t *testing.T) {
	f := panelFont()
	s := popupFixture()
	l := layoutPopup(s, f, f)
	ink := popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}
	pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	drawPopup(pic, f, f, nil, s, l, ink, image.Point{}, false)
	band := func(i int) bool {
		r := l.Rows[i].Rect
		return pic.RGBAAt(r.Max.X-3, r.Min.Y+r.Dy()/2) == ink.TitleGround
	}
	if !band(0) {
		t.Fatal("the heading row has no band")
	}
	if band(1) {
		t.Fatal("an ordinary row has a band")
	}
}

// A dragged window may pass the window's edges but keeps its title row and
// popupKeepWide pixels inside, and a drag back moves it at once.
func TestPopupDragsPartlyOffTheWindow(t *testing.T) {
	f, rows := endless()
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
	bounds := e.popupBounds()
	_, far := dragTitle(t, e, image.Pt(5000, 5000))
	if far.Min.X != bounds.Max.X-popupKeepWide || far.Min.Y != bounds.Max.Y-far.Dy() {
		t.Fatalf("a drag past the bottom right left the title at %v in %v", far, bounds)
	}
	_, back := dragTitle(t, e, image.Pt(-30, -30))
	if back.Min != far.Min.Add(image.Pt(-30, -30)) {
		t.Fatalf("a drag back from %v went to %v", far, back)
	}
	_, left := dragTitle(t, e, image.Pt(-5000, 0))
	if left.Max.X != bounds.Min.X+popupKeepWide {
		t.Fatalf("a drag past the left edge left the title at %v", left)
	}
	_, top := dragTitle(t, e, image.Pt(0, -5000))
	if top.Min.Y != bounds.Min.Y {
		t.Fatalf("a drag past the top left the title at %v", top)
	}
}

// A press 6 px inside the border and one in the 16x16 corner resize; one in
// the body does not.
func TestPopupResizeBandAndCorner(t *testing.T) {
	press := func(pick func(r image.Rectangle) image.Point) (before, after image.Rectangle) {
		f, rows := endless()
		e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
		return edgeDrag(t, e, pick, image.Pt(20, 20))
	}
	if b, a := press(func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-6, (r.Min.Y+r.Max.Y)/2) }); a.Dx() != b.Dx()+20 || a.Dy() != b.Dy() {
		t.Fatalf("a press 6 px inside the right edge: %v -> %v", b, a)
	}
	if b, a := press(func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-15, r.Max.Y-15) }); a.Dx() != b.Dx()+20 || a.Dy() != b.Dy()+20 {
		t.Fatalf("a press in the corner: %v -> %v", b, a)
	}
	if b, a := press(func(r image.Rectangle) image.Point { return image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2) }); a.Size() != b.Size() {
		t.Fatalf("a press in the body resized: %v -> %v", b, a)
	}
}

// Typing in a focused input has its own undo and redo; focus loss ends the
// history, and the map's undo works with no input focused.
func TestTypingUndoRedo(t *testing.T) {
	e, _ := brushEditor(t)
	p := &editorMoveProbe{doc: e.doc, dirty: true}
	e.doc.Edits = p
	_ = e.Key("saveas")
	e.TextInput("ab")
	e.TextInput("cd")
	_ = e.Key("backspace")
	text := func() string { return e.prompt.path }
	if text() != "abc" {
		t.Fatalf("typed %q", text())
	}
	_ = e.Key("textundo")
	if text() != "abcd" {
		t.Fatalf("undo of the erase: %q", text())
	}
	_ = e.Key("textundo")
	_ = e.Key("textundo")
	if text() != "" {
		t.Fatalf("undo to the start: %q", text())
	}
	_ = e.Key("textundo")
	if text() != "" {
		t.Fatalf("undo past the start: %q", text())
	}
	_ = e.Key("textredo")
	if text() != "ab" {
		t.Fatalf("redo: %q", text())
	}
	e.TextInput("x")
	_ = e.Key("textredo")
	if text() != "abx" {
		t.Fatalf("typing did not cut the redo steps: %q", text())
	}
	if p.undo != 0 {
		t.Fatal("a text undo reached the map")
	}
	_ = e.Key("escape")
	_ = e.Key("saveas")
	_ = e.Key("textundo")
	if len(e.input.undo) != 0 || len(e.input.redo) != 0 {
		t.Fatalf("focus loss kept the history: %q", text())
	}
	_ = e.Key("escape")
	_ = e.Key("undo")
	if p.undo != 1 {
		t.Fatalf("map undo with no input focused ran %d times", p.undo)
	}
}

// With EDITOR_WINDOW_PICS set to a directory, writes the heading band and the
// grip as pictures from the popup painter's own frame.
func TestPopupPicturesForTheNotes(t *testing.T) {
	dir := os.Getenv("EDITOR_WINDOW_PICS")
	if dir == "" {
		t.Skip("no picture directory")
	}
	f := panelFont()
	s := popupFixture()
	l := layoutPopup(s, f, f)
	ink := popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}
	pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	drawPopup(pic, f, f, nil, s, l, ink, image.Point{}, false)
	crop := func(name string, r image.Rectangle, scale int) {
		out := image.NewRGBA(image.Rect(0, 0, r.Dx()*scale, r.Dy()*scale))
		for y := 0; y < out.Bounds().Dy(); y++ {
			for x := 0; x < out.Bounds().Dx(); x++ {
				out.SetRGBA(x, y, pic.RGBAAt(r.Min.X+x/scale, r.Min.Y+y/scale))
			}
		}
		fh, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer fh.Close()
		if err := png.Encode(fh, out); err != nil {
			t.Fatal(err)
		}
	}
	crop("popup.png", l.Rect.Inset(-4).Intersect(pic.Bounds()), 2)
	crop("grip.png", image.Rect(l.Rect.Max.X-popupCorner-4, l.Rect.Max.Y-popupCorner-4, l.Rect.Max.X+2, l.Rect.Max.Y+2), 8)
}
