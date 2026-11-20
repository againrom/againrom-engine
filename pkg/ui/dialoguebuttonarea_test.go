package ui

import (
	"image"
	"testing"
)

// exactWindowPoint is the first window pixel the native point p of the open
// dialogue is drawn at, with no search for a nearby pixel.
func exactWindowPoint(t *testing.T, a *App, p image.Point) (image.Point, bool) {
	t.Helper()
	v := a.flow.viewer
	fx, fy, ok := v.noticePlace().FrameToWindow(p)
	if !ok {
		return image.Point{}, false
	}
	x, y, ok := v.place.FrameToWindow(image.Pt(fx, fy))
	return image.Pt(x, y), ok
}

// A page with one reply keeps its button, and every pixel of it turns the page.
func TestOneReplyPageButtonAdvancesFromEveryCorner(t *testing.T) {
	a, seam := noticeApp(t)
	a.Layout(1280, 960)
	seam.dest = NoticeStay
	seam.v.SetDialogue(Dialogue{Text: "Strangers! Will you drive them out?", Replies: replyTexts(1)})
	if seam.v.NoticeReplyCount() != 0 {
		t.Fatal("a page with one reply shows reply rows")
	}
	b := seam.v.noticeLayout().Button
	b = b.Add(seam.v.noticeLayout().Box.Min)
	for i, p := range []image.Point{b.Min, b.Max.Sub(image.Pt(1, 1)), image.Pt(b.Min.X, b.Max.Y-1), image.Pt(b.Max.X-1, b.Min.Y)} {
		seam.v.SetDialogue(Dialogue{Text: "Strangers! Will you drive them out?", Replies: replyTexts(1)})
		w, ok := exactWindowPoint(t, a, p)
		if !ok {
			t.Fatalf("button pixel %v has no window pixel", p)
		}
		for _, action := range []string{"press", "release"} {
			if err := a.HeadlessPointer(action, w.X, w.Y); err != nil {
				t.Fatal(err)
			}
		}
		if seam.advances != i+1 {
			t.Fatalf("corner %v advanced %d pages, want %d", p, seam.advances, i+1)
		}
	}
}

// A click on any pixel of the drawn page button turns the page; a click on the
// text area or on empty screen outside the window does not (the panel's text
// child answers a press with nothing; only the button posts the page command).
func TestDialogueButtonAdvancesFromEveryCornerAndTextAreaDoesNot(t *testing.T) {
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(1280, 960), image.Pt(1600, 900), image.Pt(1366, 768)} {
		t.Run(size.String(), func(t *testing.T) {
			f := newDialoguePointerFixture(t, "mission")
			f.app.Layout(size.X, size.Y)
			b := f.button
			want := 0
			for _, p := range []image.Point{b.Min, b.Max.Sub(image.Pt(1, 1)), image.Pt(b.Min.X, b.Max.Y-1), image.Pt(b.Max.X-1, b.Min.Y)} {
				w, ok := exactWindowPoint(t, f.app, p)
				if !ok {
					t.Fatalf("button pixel %v has no window pixel", p)
				}
				want++
				f.pointer(t, "press", w, want-1)
				f.pointer(t, "release", w, want)
			}
			text, ok := exactWindowPoint(t, f.app, f.body)
			if !ok {
				t.Fatal("text pixel has no window pixel")
			}
			for _, w := range []image.Point{text, image.Pt(1, 1)} {
				f.pointer(t, "press", w, want)
				f.pointer(t, "release", w, want)
			}
		})
	}
}
