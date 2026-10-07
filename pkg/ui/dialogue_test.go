package ui

// The speaker half of a dialogue window: which of its two shapes is open,
// whose face stands in its pane, and what survives a page.
//
// Every fixture here is built in this file. No shipped byte, no game install and
// no non-ASCII literal enters one — a "face" below is a rectangle of one colour
// made in test code, which is all this tier is ever handed.

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// solidFace is a picture of one colour, the size of the pane's interior — what a
// resolved speaker's face would arrive as.
func solidFace(c color.RGBA) *image.RGBA {
	p := AuthoredDialogueLayout().Portrait
	img := image.NewRGBA(image.Rect(0, 0, p.Dx()-2, p.Dy()-2))
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// The shape comes from the FILE'S answer the driver pushes, and it reaches the
// layout the window is composed with.
func TestTheShapePushedIsTheShapeComposed(t *testing.T) {
	for _, tc := range []struct {
		name     string
		portrait bool
		wantPane bool
		wantText image.Rectangle
	}{
		{"a file that mentions a speaker", true, true, image.Rect(128, 36, 428, 171)},
		{"a file that mentions none", false, false, image.Rect(48, 36, 428, 171)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := noticeViewer(t)
			v.SetDialogue(Dialogue{Text: "hello", Portrait: tc.portrait})
			l := v.noticeLayout()
			if got := !l.Portrait.Empty(); got != tc.wantPane {
				t.Errorf("pane drawn = %v, want %v", got, tc.wantPane)
			}
			if l.Text != tc.wantText {
				t.Errorf("text area = %v, want %v", l.Text, tc.wantText)
			}
		})
	}
}

// A file that mentions a speaker only in its prose gets the pane and never a
// face — the pane stands empty for the whole window.
func TestAPaneWithNoSpeakerNeverGetsAFace(t *testing.T) {
	v := noticeViewer(t)
	for part := 1; part <= 3; part++ {
		v.SetDialogue(Dialogue{Text: "the npc guild is closed", Portrait: true})
		pane, face := v.NoticeSpeaker()
		if !pane {
			t.Fatalf("part %d: the pane must be drawn — the file mentions a speaker", part)
		}
		if face != nil {
			t.Fatalf("part %d: a face appeared with no part naming a speaker", part)
		}
	}
}

// A part that names NOBODY leaves the standing face alone; a part that names
// somebody replaces it. Both are pushed with the same words, so nothing but the
// speaker fields can be doing the work.
func TestPagingKeepsAndReplacesTheFace(t *testing.T) {
	v := noticeViewer(t)
	red, blue := solidFace(color.RGBA{R: 0xff, A: 0xff}), solidFace(color.RGBA{B: 0xff, A: 0xff})

	v.SetDialogue(Dialogue{Text: "part one", Portrait: true, Speaks: true, Face: red})
	if _, got := v.NoticeSpeaker(); got != red {
		t.Fatal("the first speaker's face did not arrive")
	}

	v.SetDialogue(Dialogue{Text: "part two", Portrait: true})
	if _, got := v.NoticeSpeaker(); got != red {
		t.Fatal("a part naming nobody must LEAVE THE STANDING FACE — that is the original's behaviour, not an omission")
	}

	v.SetDialogue(Dialogue{Text: "part three", Portrait: true, Speaks: true, Face: blue})
	if _, got := v.NoticeSpeaker(); got != blue {
		t.Fatal("a part naming a different speaker must replace the face")
	}

	// A named speaker whose picture could not be found EMPTIES the pane. This is
	// the case a single nullable picture could not distinguish from the one
	// above it, and putting the last speaker's face on these words would be
	// wrong on screen.
	v.SetDialogue(Dialogue{Text: "part four", Portrait: true, Speaks: true, Face: nil})
	if _, got := v.NoticeSpeaker(); got != nil {
		t.Fatal("a named speaker with no picture must empty the pane, not keep the previous face")
	}
}

// A face belongs to ONE window. Every path that opens or closes a window settles
// both the shape and the face, so one mission's speaker can never stand beside
// another's words.
func TestAFaceDoesNotOutliveItsWindow(t *testing.T) {
	for _, tc := range []struct {
		name  string
		close func(v *Viewer)
	}{
		{"an outcome notice replaces it", func(v *Viewer) { v.SetNotice("MISSION COMPLETE", NoticeOutcome) }},
		{"a plain dialogue push replaces it", func(v *Viewer) { v.SetNotice("plain", NoticeDialogue) }},
		{"it is closed", func(v *Viewer) { v.ClearNotice() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := noticeViewer(t)
			v.SetDialogue(Dialogue{Text: "part one", Portrait: true, Speaks: true,
				Face: solidFace(color.RGBA{R: 0xff, A: 0xff})})
			tc.close(v)
			pane, face := v.NoticeSpeaker()
			if pane || face != nil {
				t.Fatalf("the window kept pane=%v face=%v across %s", pane, face != nil, tc.name)
			}
		})
	}
}

// The outcome notice carries no pane whatever its words say — locked twice: its
// push settles the shape to false, AND its layout has no pane to resolve to.
func TestTheOutcomeNoticeNeverHasAPane(t *testing.T) {
	v := noticeViewer(t)
	v.SetNotice("the npc reports: MISSION COMPLETE", NoticeOutcome)
	if pane, _ := v.NoticeSpeaker(); pane {
		t.Error("the outcome push left the shape set")
	}
	if l := v.noticeLayout(); !l.Portrait.Empty() {
		t.Errorf("the outcome layout resolved to a pane: %v", l.Portrait)
	}
	// And even with the flag forced on, the layout has nothing to resolve to.
	v.noticePortrait = true
	if l := v.noticeLayout(); !l.Portrait.Empty() {
		t.Errorf("the outcome layout gained a pane from the flag alone: %v", l.Portrait)
	}
}

// The cached picture is rebuilt when the shape or the face changes, with the
// words held identical — the case a key carrying only the words would present
// stale.
func TestTheCacheSeesTheShapeAndTheFace(t *testing.T) {
	v := noticeViewer(t)
	red, blue := solidFace(color.RGBA{R: 0xff, A: 0xff}), solidFace(color.RGBA{B: 0xff, A: 0xff})

	v.SetDialogue(Dialogue{Text: "same words", Portrait: true, Speaks: true, Face: red})
	if _, _, _, ok := v.noticePresent(); !ok {
		t.Fatal("nothing was presented")
	}
	builds := v.noticeBuilds

	// Nothing changed: no rebuild.
	v.noticePresent()
	if v.noticeBuilds != builds {
		t.Fatalf("an unchanged frame composed %d times, want 0", v.noticeBuilds-builds)
	}

	for _, tc := range []struct {
		name string
		push Dialogue
	}{
		{"a different face under the same words", Dialogue{Text: "same words", Portrait: true, Speaks: true, Face: blue}},
		{"the same face under a different shape", Dialogue{Text: "same words", Portrait: false, Speaks: true, Face: blue}},
	} {
		v.push(tc.push)
		v.noticePresent()
		if v.noticeBuilds != builds+1 {
			t.Errorf("%s composed %d times, want 1", tc.name, v.noticeBuilds-builds)
		}
		builds = v.noticeBuilds
	}
}

// push is SetDialogue by another name, so the table above reads as a list of
// pushes rather than as a closure per row.
func (v *Viewer) push(d Dialogue) { v.SetDialogue(d) }

// A viewer that cannot draw a notice is untouched by all of it: speaker-bearing
// dialogues are pushed, and nothing is drawn and nothing is held.
func TestAFontlessViewerIsUntouchedByTheSpeaker(t *testing.T) {
	v, err := NewViewer("t", grid(20, 20), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, frame.W, frame.H)
	v.SetDialogue(Dialogue{Text: "part one", Portrait: true, Speaks: true,
		Face: solidFace(color.RGBA{R: 0xff, A: 0xff})})
	if v.NoticeOpen() {
		t.Error("a viewer with no font must take no input for a notice")
	}
	if _, _, _, ok := v.noticePresent(); ok {
		t.Error("a viewer with no font composed a notice")
	}
	if _, _, ok := v.noticeBackdropOf(); ok {
		t.Error("a viewer with no font darkened the map")
	}
}

// The pane and the face reach the PIXELS, not merely the state. Two windows
// differing only in the speaker draw differently, and the pane's rectangle is
// where the difference is.
func TestTheFaceReachesTheComposedPicture(t *testing.T) {
	f := panelFont()
	l := AuthoredDialogueLayout().WithPortrait(true)
	green := color.RGBA{G: 0xff, A: 0xff}

	withFace := RenderNotice(l, f, "hello", solidFace(green))
	without := RenderNotice(l, f, "hello", nil)
	p := l.Portrait
	cx, cy := p.Min.X+p.Dx()/2, p.Min.Y+p.Dy()/2
	if got := withFace.RGBAAt(cx, cy); got != green {
		t.Errorf("pane centre with a face = %v, want %v", got, green)
	}
	if got := without.RGBAAt(cx, cy); got == green {
		t.Error("the empty pane carries the face's colour")
	}
}
