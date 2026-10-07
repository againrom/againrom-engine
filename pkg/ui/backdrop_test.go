package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// backdropView is the drawable area every case below measures against. It is
// not the design space: the map screen fills the window, so the rectangle the
// dim covers is the camera's and has nothing to do with the notice's letterbox.
const backdropViewW, backdropViewH = 1024, 768

// backdropViewer is a map-screen viewer that CAN draw a notice: a synthetic
// grid, a font, and a camera sized like a window.
func backdropViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("b", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	layoutViewport(v, backdropViewW, backdropViewH)
	return v
}

// backdropFontless is the same viewer with no font — the state a front-end whose
// lettering failed to load is in, and AC-8's subject.
func backdropFontless(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("b", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, backdropViewW, backdropViewH)
	return v
}

// AC-6 — a notice open yields one dim over the WHOLE drawable area, in the
// authored colour. The rectangle is hand-written from the two constants above,
// never read back off the camera.
func TestAnOpenNoticeDimsTheWholeView(t *testing.T) {
	v := backdropViewer(t)
	v.SetNotice("words", NoticeDialogue)

	r, c, ok := v.noticeBackdropOf()
	if !ok {
		t.Fatal("a notice is open and no dim was composed")
	}
	if want := image.Rect(0, 0, backdropViewW, backdropViewH); r != want {
		t.Errorf("dim covers %v, want the whole view %v", r, want)
	}
	if want := AuthoredNoticeBackdrop(); c != want {
		t.Errorf("dim colour = %v, want the authored %v", c, want)
	}
	// The authored value must actually darken: a transparent one would satisfy
	// every rectangle assertion above and dim nothing at all.
	if c.A == 0 {
		t.Error("the authored dim is fully transparent — it would darken nothing")
	}
	if c.A == 0xff {
		t.Error("the authored dim is opaque — it would hide the map rather than dim it")
	}
}

// AC-4 — the dim is a property of a notice being open and not of its kind. Both
// kinds are driven through the same decision and must agree exactly.
func TestBothNoticeKindsDimIdentically(t *testing.T) {
	var got [2]struct {
		r  image.Rectangle
		c  color.RGBA
		ok bool
	}
	for i, kind := range []NoticeKind{NoticeDialogue, NoticeOutcome} {
		v := backdropViewer(t)
		v.SetNotice("words", kind)
		got[i].r, got[i].c, got[i].ok = v.noticeBackdropOf()
		if !got[i].ok {
			t.Fatalf("notice kind %d is open and no dim was composed", kind)
		}
	}
	if got[0] != got[1] {
		t.Errorf("the outcome notice dims %v/%v and the dialogue one %v/%v — the dim is a "+
			"function of the kind", got[1].r, got[1].c, got[0].r, got[0].c)
	}
}

// AC-7 — no notice, no dim; and a fully transparent value composes NOTHING
// rather than composing something invisible, so the frame is the one composed
// before this story rather than one that merely looks like it.
func TestNoDimWithoutANoticeOrWithATransparentValue(t *testing.T) {
	t.Run("no notice open", func(t *testing.T) {
		v := backdropViewer(t)
		if _, _, ok := v.noticeBackdropOf(); ok {
			t.Error("a viewer with no notice open composed a dim")
		}
	})

	t.Run("a notice opened and then closed", func(t *testing.T) {
		v := backdropViewer(t)
		v.SetNotice("words", NoticeDialogue)
		if _, _, ok := v.noticeBackdropOf(); !ok {
			t.Fatal("setup: no dim while the notice was open")
		}
		v.ClearNotice()
		if _, _, ok := v.noticeBackdropOf(); ok {
			t.Error("the dim outlived the notice")
		}
	})

	t.Run("a fully transparent value, with a notice open", func(t *testing.T) {
		v := backdropViewer(t)
		v.SetNoticeBackdrop(color.RGBA{})
		v.SetNotice("words", NoticeDialogue)
		if v.NoticeOpen() != true {
			t.Fatal("setup: the notice is not open")
		}
		if _, _, ok := v.noticeBackdropOf(); ok {
			t.Error("a transparent backdrop still composed a dim")
		}
	})
}

func TestTheDimIsTheValueItWasGiven(t *testing.T) {
	v := backdropViewer(t)
	v.SetNotice("words", NoticeDialogue)
	want := color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x44}
	v.SetNoticeBackdrop(want)

	r, c, ok := v.noticeBackdropOf()
	if !ok {
		t.Fatal("a supplied opaque-enough backdrop composed no dim")
	}
	if c != want {
		t.Errorf("dim colour = %v, want the supplied %v", c, want)
	}
	if got := image.Rect(0, 0, backdropViewW, backdropViewH); r != got {
		t.Errorf("supplying a colour moved the rectangle to %v", r)
	}
}

func TestAViewerThatCannotDrawANoticeDimsNothing(t *testing.T) {
	v := backdropFontless(t)
	for i := 0; i < 4; i++ {
		v.SetNotice("words", NoticeDialogue)
		v.SetNotice("more words", NoticeOutcome)
		if _, _, ok := v.noticeBackdropOf(); ok {
			t.Fatalf("push %d: a viewer with no font composed a dim", i)
		}
		if v.NoticeOpen() {
			t.Fatalf("push %d: a viewer with no font reports a notice showing", i)
		}
	}
	// And it gains the dim the moment it CAN draw the box, from the notice that
	// was already pushed — so the two answers move together.
	v.SetFont(panelFont())
	if _, _, ok := v.noticeBackdropOf(); !ok {
		t.Error("a font arriving over an already-pushed notice did not bring the dim with it")
	}
}

// AC-9 — a viewer whose camera has no area composes no dim and does not panic.
// A window of zero size is what a front-end has before its first Layout.
func TestNoDimWithoutADrawableArea(t *testing.T) {
	v, err := NewViewer("b", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFont(panelFont())
	v.cam.ViewW, v.cam.ViewH = 0, 0
	v.SetNotice("words", NoticeDialogue)
	if _, _, ok := v.noticeBackdropOf(); ok {
		t.Error("a viewer with no drawable area composed a dim")
	}
}

// 0077 AC-6 — THE ORDER, witnessed by parsing drawFrame rather than by reading pixels
// back (which this package cannot do before the game starts).
//
// The claim is that the dim stands after everything the front end draws over the
// map and before the popup alone. Both halves are checked, because only together
// do they say anything: after the panel alone would allow it over the notice too,
// and before the notice alone would allow it under the terrain.
func TestTheDimIsComposedAfterEverythingButThePopup(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "viewer.go", nil, 0)
	if err != nil {
		t.Fatalf("parse viewer.go: %v", err)
	}
	var body *ast.BlockStmt
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "drawFrame" || fd.Recv == nil {
			return true
		}
		body = fd.Body
		return false
	})
	if body == nil {
		t.Fatal("no Viewer.drawFrame found in viewer.go")
	}

	// The index of the top-level statement in drawFrame that mentions each name. A
	// name that appears in no statement, or in more than one, is itself a
	// finding: this test's whole meaning is that each of these is one site.
	at := func(name string) int {
		found := -1
		for i, stmt := range body.List {
			mentions := false
			ast.Inspect(stmt, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == name {
					mentions = true
				}
				return true
			})
			if mentions {
				if found >= 0 {
					t.Fatalf("%q is mentioned by two top-level statements of drawFrame (%d and %d) — "+
						"this ordering test can no longer say where it stands", name, found, i)
				}
				found = i
			}
		}
		if found < 0 {
			t.Fatalf("no top-level statement of drawFrame mentions %q", name)
		}
		return found
	}

	dim := at("noticeBackdropOf")
	// Everything the front end draws over the map: the overlay passes and the
	// route strokes are the last of the map picture proper, and the unit
	// information panel and the debug readout are the two boxes that used to
	// stand over the dim and now stand under it.
	for _, under := range []string{"overlayPasses", "displayedPathSegments", "panelPresent", "readoutPresent"} {
		if i := at(under); i >= dim {
			t.Errorf("the dim is composed at statement %d, at or before %q at %d — "+
				"that surface would not dim", dim, under, i)
		}
	}
	// The popup, and it alone, keeps its ordinary brightness.
	if i := at("dialogueBackdrop"); i <= dim {
		t.Errorf("the dim is composed at statement %d, at or after the popup at %d — "+
			"the packed dialogue would precede the HUD", dim, i)
	}
}
