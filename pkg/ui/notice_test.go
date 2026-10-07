package ui

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// The notice: its geometry, its wrap, its clamp and the three inputs that
// advance it.
//
// Every fixture here is built in this file. Nothing reads a game install, and no
// assertion below is about a shipped string — the words in these tests are ASCII
// this file wrote.
//
// THE FONT IS panelFont, shared with the panel's own tests: 224 records, each
// distinguishable from the others, a fixed advance of 3 plus 1 of spacing per
// glyph and 6 for the space. A wrap measured in glyph advances is then arithmetic
// a test can state rather than a number read off a real atlas.

// noticeAdvanceOf is one glyph's contribution to the pen for the shared test
// font: the record's own advance plus the font's spacing.
const noticeAdvanceOf = panelAdvance + panelSpacing

// noticeSpaceAdvance is the space record's, which additionally carries half its
// own height by the font's rule for record 0.
const noticeSpaceAdvance = panelSpaceAdvance + panelSpacing + panelCellH/2

// The claims' own numbers, restated here so that a change to the layout has to
// be made twice — once in the shipped value and once against `DLG-PANEL-035` and
// `DLG-RECT-037` — rather than silently redefining what the story reproduced.
func TestDialogueNoticeGeometryIsTheClaims(t *testing.T) {
	l := AuthoredDialogueLayout()
	if l.Style != NoticeStyleDialogue {
		t.Errorf("style = %v, want the dialogue style", l.Style)
	}
	if got, want := l.Box, image.Rect(76, 124, 564, 356); got != want {
		t.Errorf("box = %v, want %v — the panel drawn at 640x480, DLG-PANEL-035", got, want)
	}
	if got, want := l.Box.Size().Sub(image.Pt(noticeShadow, noticeShadow)), image.Pt(480, 224); got != want {
		t.Errorf("frame body = %v, want %v — the box less the 8 px shadow band", got, want)
	}
	if got, want := l.Text, image.Rect(48, 36, 428, 171); got != want {
		t.Errorf("text area = %v, want %v — panel-relative, the no-portrait layout", got, want)
	}
	if got, want := l.TextBesidePortrait, image.Rect(128, 36, 428, 171); got != want {
		t.Errorf("text beside the portrait = %v, want %v — panel-relative, 300x135 at (204,160)", got, want)
	}
	if got, want := l.Portrait, image.Rect(30, 54, 118, 168); got != want {
		t.Errorf("portrait child = %v, want %v — panel-relative", got, want)
	}
	if got, want := l.Button, image.Rect(200, 172, 280, 198); got != want {
		t.Errorf("button = %v, want %v — panel-relative", got, want)
	}
	if l.Pitch != 2 {
		t.Errorf("pitch extra = %d, want 2 — the font's height plus two", l.Pitch)
	}
}

func TestOutcomeNoticeIsVisiblyDistinct(t *testing.T) {
	d, o := AuthoredDialogueLayout(), AuthoredOutcomeLayout()
	if d.Border == o.Border {
		t.Error("the two notices share a frame colour — FR-7 requires a different one")
	}
	if d.Box == o.Box {
		t.Error("the two notices share a box — the outcome's is this project's own")
	}
	if o.Box.Min.X < 0 || o.Box.Min.Y < 0 || o.Box.Max.X > frame.W || o.Box.Max.Y > frame.H {
		t.Errorf("the outcome box %v leaves the design space", o.Box)
	}
}

// The wrap, at a width that admits exactly three of this font's glyphs. Every
// expectation here is arithmetic over the advances above rather than a number
// somebody measured once.
func TestNoticeLinesBreaksAtTheLastSpaceThatFits(t *testing.T) {
	f := panelFont()
	// Wide enough for "aa bb" (2+space+2) and not for a third pair.
	width := 4*noticeAdvanceOf + noticeSpaceAdvance
	got := NoticeLines(f, "aa bb cc dd", width)
	want := []string{"aa bb", "cc dd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q — the break is the last space that fits", got, want)
	}
}

// The break byte itself is dropped, and a run of them does not indent the line
// that follows.
func TestNoticeLinesDropsTheBreakByte(t *testing.T) {
	f := panelFont()
	got := NoticeLines(f, "aa   bb", 2*noticeAdvanceOf)
	want := []string{"aa", "bb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q — no line begins with a space", got, want)
	}
}

// An authored line break is a break OPPORTUNITY as well as a byte that draws as
// a space. This is ours and disclosed: the atlas has no record for one, so a
// newline that did not break would make a two-line paragraph one unbreakable
// word.
func TestNoticeLinesTreatsAuthoredLineBreaksAsBreaks(t *testing.T) {
	f := panelFont()
	got := NoticeLines(f, "aa\r\nbb", 4*noticeAdvanceOf)
	want := []string{"aa", "bb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
}

// A single word wider than the area is broken INSIDE the word — the only way to
// show it at all — and only then.
func TestNoticeLinesBreaksInsideAnOverWideWord(t *testing.T) {
	f := panelFont()
	got := NoticeLines(f, "abcdefg", 3*noticeAdvanceOf)
	want := []string{"abc", "def", "g"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q — a word that does not fit alone is cut", got, want)
	}
}

// The degenerate width: narrower than one glyph. It must still terminate, and it
// must still produce every byte.
func TestNoticeLinesTerminatesBelowOneGlyph(t *testing.T) {
	f := panelFont()
	got := NoticeLines(f, "abc", 1)
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q — one byte a line rather than an endless loop", got, want)
	}
}

// The clamp: the lines the text produced, cut to the lines the area holds, with
// no scroll and no ellipsis.
func TestNoticeLayoutClampsToTheArea(t *testing.T) {
	f := panelFont()
	l := AuthoredDialogueLayout()
	pitch := f.Height() + l.Pitch
	want := l.Text.Dy() / pitch

	// One word per line, more of them than the area can hold.
	// Enough words that the wrap produces more lines than the area holds:
	// one word per (area width / word width) is the wrap's own density.
	perLine := l.Text.Dx() / (10*noticeAdvanceOf + noticeSpaceAdvance)
	long := strings.TrimSpace(strings.Repeat("abcdefghij ", (want+4)*(perLine+1)))
	all := noticeWrap(l, f, long)
	if len(all) <= want {
		t.Fatalf("setup: the text produced %d lines, which the area's %d already holds", len(all), want)
	}
	got := NoticeLayoutOf(l, f, long)
	if len(got) != want {
		t.Fatalf("%d lines drawn, want %d = area height %d / pitch %d",
			len(got), want, l.Text.Dy(), pitch)
	}
	if !reflect.DeepEqual(got, all[:want]) {
		t.Errorf("the drawn lines are not the first %d — the clamp dropped the wrong end", want)
	}
}

func TestNoticeLayoutIsPure(t *testing.T) {
	f := panelFont()
	l := AuthoredDialogueLayout()
	s := "the mission opens and the words are shown over the map"

	first := NoticeLayoutOf(l, f, s)
	for i := 0; i < 8; i++ {
		if got := NoticeLayoutOf(l, f, s); !reflect.DeepEqual(got, first) {
			t.Fatalf("call %d = %q, want %q — the layout read something that moved", i, got, first)
		}
	}
	// A different geometry gives a different answer, which is what makes the
	// agreement above evidence rather than a constant.
	narrow := l
	narrow.Text = image.Rect(l.Text.Min.X, l.Text.Min.Y, l.Text.Min.X+20*noticeAdvanceOf+noticeSpaceAdvance, l.Text.Max.Y)
	if reflect.DeepEqual(NoticeLayoutOf(narrow, f, s), first) {
		t.Error("narrowing the text area changed nothing — the layout is not reading its geometry")
	}
}

// plainDialogueLayout is the dialogue layout's geometry in the plain style: the
// generic notice mechanism, which every test below that is not about the
// dialogue panel's own look measures.
func plainDialogueLayout() NoticeLayout {
	l := AuthoredDialogueLayout()
	l.Style = NoticeStylePlain
	return l
}

// noticeInkLayout is the dialogue layout with every frame colour blacked out, so
// any pixel carrying a non-zero channel is TEXT. It is the panel tests' own
// device.
// It resolves to the WITHOUT-pane shape, which is what every assertion in this
// file was written against: the pane is 0079's and its own tests ask for it
// explicitly, so a shipped wrapping or extent assertion keeps measuring the
// rectangle it was calibrated on.
func noticeInkLayout() NoticeLayout {
	l := plainDialogueLayout().WithPortrait(false)
	black := color.RGBA{A: 0xff}
	white := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	l.Fill, l.Border, l.ButtonFill, l.ButtonBorder = black, black, black, black
	l.TextColor, l.ButtonText = white, white
	return l
}

// Nothing painted leaves the box, and the picture is the box.
func TestNoticeIsComposedAtItsAuthoredSize(t *testing.T) {
	f := panelFont()
	l := noticeInkLayout()
	img := RenderNotice(l, f, strings.Repeat("abcdefghij ", 200), nil)
	if img == nil {
		t.Fatal("RenderNotice composed nothing")
	}
	if got, want := img.Bounds().Size(), l.Box.Size(); got != want {
		t.Fatalf("picture is %v, want the authored box %v — the design space is where it is composed", got, want)
	}
	right, bottom, found := inkExtent(img)
	if !found {
		t.Fatal("no text was painted at all")
	}
	if right > l.Box.Dx() || bottom > l.Box.Dy() {
		t.Errorf("ink reaches (%d, %d), outside the %v box", right, bottom, l.Box.Size())
	}
}

// A viewer with no font draws no notice AND takes no input for one — which is
// the whole of AC-22's "otherwise unaffected".
func TestNoticeNeedsAFont(t *testing.T) {
	v, err := NewViewer("t", grid(20, 20), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	layoutViewport(v, frame.W, frame.H)
	v.SetNotice("something", NoticeDialogue)

	if v.NoticeOpen() {
		t.Error("a fontless viewer reports a notice open — Escape would be swallowed by a box nobody can see")
	}
	if _, _, _, ok := v.noticePresent(); ok {
		t.Error("a fontless viewer presented a notice")
	}
	// What was pushed is still readable, so the driver's own state and the
	// viewer's are distinguishable.
	if s, k, open := v.NoticeState(); s != "something" || k != NoticeDialogue || !open {
		t.Errorf("NoticeState = %q/%v/%v, want the pushed value", s, k, open)
	}
}

// The picture is composed once per change and re-presented otherwise, and a
// resize rescales rather than recomposes — the geometry is authored in the
// design space, so nothing about it is a function of the window.
func TestNoticeRecomposesOnlyOnAChange(t *testing.T) {
	v := noticeViewer(t)
	v.SetNotice("one", NoticeDialogue)
	if _, _, _, ok := v.noticePresent(); !ok {
		t.Fatal("setup: no notice presented")
	}
	builds := v.noticeBuilds
	for i := 0; i < 4; i++ {
		v.noticePresent()
	}
	if v.noticeBuilds != builds {
		t.Errorf("%d compositions over four unchanged frames, want %d", v.noticeBuilds-builds, 0)
	}
	layoutViewport(v, frame.W*2, frame.H*2)
	if _, _, _, ok := v.noticePresent(); !ok {
		t.Fatal("no notice presented after a resize")
	}
	if v.noticeBuilds != builds {
		t.Errorf("a resize recomposed the picture; the design-space geometry does not depend on the window")
	}
	v.SetNotice("two", NoticeDialogue)
	v.noticePresent()
	if v.noticeBuilds != builds+1 {
		t.Errorf("changing the words composed %d times, want 1", v.noticeBuilds-builds)
	}
}

func noticeViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("t", grid(20, 20), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	v.SetFont(panelFont())
	layoutViewport(v, frame.W, frame.H)
	return v
}

// The button's hit region is the authored rectangle, mapped back through the
// same letterbox the draw maps forward through — at every window scale.
func TestNoticeButtonHitRegion(t *testing.T) {
	l := AuthoredDialogueLayout()
	btn := l.Button.Add(l.Box.Min)
	cx, cy := (btn.Min.X+btn.Max.X)/2, (btn.Min.Y+btn.Max.Y)/2

	for _, scale := range []int{1, 2} {
		v := noticeViewer(t)
		layoutViewport(v, frame.W*scale, frame.H*scale)
		v.SetNotice("words", NoticeDialogue)
		ox, oy := (v.cam.ViewW-frame.W)/2, (v.cam.ViewH-frame.H)/2
		if !v.noticeButtonAt(cx+ox, cy+oy) {
			t.Errorf("scale %d: the button's own centre is not on the button", scale)
		}
		// Just outside its left edge, and the box's own centre, which is text.
		if v.noticeButtonAt(btn.Min.X-2+ox, cy+oy) {
			t.Errorf("scale %d: a position left of the button hit it", scale)
		}
		if v.noticeButtonAt(l.Box.Min.X+4+ox, l.Box.Min.Y+4+oy) {
			t.Errorf("scale %d: the box's own corner hit the button", scale)
		}
	}
}

// A closed notice has no button to hit, so a click where one used to be reaches
// the map exactly as it would have.
func TestNoticeButtonIsGoneWhenClosed(t *testing.T) {
	l := AuthoredDialogueLayout()
	btn := l.Button.Add(l.Box.Min)
	v := noticeViewer(t)
	v.SetNotice("words", NoticeDialogue)
	v.ClearNotice()
	if v.noticeButtonAt((btn.Min.X+btn.Max.X)/2, (btn.Min.Y+btn.Max.Y)/2) {
		t.Error("a closed notice still answers for its button")
	}
}

// noticeSeam is a map seam that opens a notice and counts its advances, so the
// three inputs can be driven through the whole front-end.
type noticeSeam struct {
	v        *Viewer
	advances int
	actions  []NoticeAction
	dest     NoticeDest
	msg      string
	ticks    int
}

func (s *noticeSeam) advance(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
	s.advances++
	action := NoticeAdvance
	if len(actions) > 0 {
		action = actions[0]
	}
	s.actions = append(s.actions, action)
	if s.dest == NoticeStay {
		s.v.ClearNotice()
	}
	return s.dest, s.msg, nil
}

func TestSuccessPanelGeometryAndInputStateTable(t *testing.T) {
	l := AuthoredSuccessLayout()
	if l.Box != image.Rect(128, 200, 512, 380) ||
		l.Button.Add(l.Box.Min) != image.Rect(176, 284, 464, 308) ||
		l.SecondaryButton.Add(l.Box.Min) != image.Rect(176, 308, 464, 332) {
		t.Fatalf("success geometry = box %v, controls %v/%v", l.Box,
			l.Button.Add(l.Box.Min), l.SecondaryButton.Add(l.Box.Min))
	}

	now := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name string
		in   appInput
		want NoticeAction
	}{
		{"Return activates initial Victory focus", appInput{Enter: true}, NoticeVictory},
		{"Escape chooses Continue", appInput{Escape: true}, NoticeContinue},
		{"Victory control", appInput{PrimaryPressed: true, CursorX: 320, CursorY: 296}, NoticeVictory},
		{"Continue control", appInput{PrimaryPressed: true, CursorX: 320, CursorY: 320}, NoticeContinue},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, seam := noticeApp(t)
			seam.v.SetNotice("Mission Completed", NoticeSuccess)
			a.step(tc.in, now)
			if len(seam.actions) != 1 || seam.actions[0] != tc.want {
				t.Fatalf("actions = %v, want [%v]", seam.actions, tc.want)
			}
		})
	}
}

// noticeApp builds a front-end parked on the map screen with a notice open.
func noticeApp(t *testing.T) (*App, *noticeSeam) {
	t.Helper()
	seam := &noticeSeam{}
	load := func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("n", grid(60, 60), &terrain.Tileset{})
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		hideBottomPanels(v)
		v.SetFont(panelFont())
		seam.v = v
		return v, func() { seam.ticks++ }, nil, nil, nil, seam.advance, nil, nil, nil, nil, nil
	}
	a := newTestApp(t, appRows(3), load)
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	mapAtScaleOne(a)
	return a, seam
}

// The keys advance once; the pointer needs a complete owned gesture.
func TestNoticeAdvancesOnEachOfTheThreeInputs(t *testing.T) {
	l := AuthoredDialogueLayout()
	btn := l.Button.Add(l.Box.Min)
	bx, by := (btn.Min.X+btn.Max.X)/2, (btn.Min.Y+btn.Max.Y)/2
	now := time.Unix(1_700_000_000, 0)

	for _, tc := range []struct {
		name string
		in   appInput
	}{
		{"the button", appInput{PrimaryPressed: true, CursorX: bx, CursorY: by}},
		{"RETURN", appInput{Enter: true}},
		{"ESCAPE", appInput{Escape: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, seam := noticeApp(t)
			seam.v.SetNotice("words", NoticeDialogue)

			a.step(tc.in, now)
			if tc.in.PrimaryPressed {
				if seam.advances != 0 {
					t.Fatalf("button press advanced %d pages, want 0", seam.advances)
				}
				release := tc.in
				release.PrimaryPressed, release.PrimaryReleased = false, true
				a.step(release, now)
			}
			if seam.advances != 1 {
				t.Fatalf("%d advances, want 1", seam.advances)
			}
			if a.Screen() != ScreenMap {
				t.Fatalf("screen = %v, want ScreenMap — a stay destination leaves the screen alone", a.Screen())
			}
			// A frame carrying no press advances nothing, over a notice that is
			// open again: the edge acted once and the level is not read.
			seam.v.SetNotice("words", NoticeDialogue)
			a.step(appInput{}, now)
			if seam.advances != 1 {
				t.Fatalf("%d advances over a frame with no press, want 1", seam.advances)
			}
		})
	}
}

// AC-9: Escape over an open notice advances it and does NOT leave the map
// screen; Escape with none open unwinds exactly as it did before this story.
func TestEscapeIsTakenByAnOpenNoticeAndOnlyThen(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	a, seam := noticeApp(t)
	seam.v.SetNotice("words", NoticeDialogue)
	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("Escape over an open notice left for %v", a.Screen())
	}
	if seam.advances != 1 {
		t.Fatalf("%d advances, want 1", seam.advances)
	}
	// The seam closed it; the next Escape is the map screen's own.
	a.step(appInput{Escape: true}, now)
	leaveViaMenu(a.flow) //
	if a.Screen() != ScreenPicker {
		t.Fatalf("Escape with no notice open went to %v, want ScreenPicker", a.Screen())
	}
	if seam.advances != 1 {
		t.Errorf("the unwinding Escape also advanced a notice")
	}
}

// A click on the button must not also reach the map behind it. The press is
// consumed, so no selection gesture begins.
func TestTheButtonPressDoesNotReachTheMap(t *testing.T) {
	l := AuthoredDialogueLayout()
	btn := l.Button.Add(l.Box.Min)
	now := time.Unix(1_700_000_000, 0)

	a, seam := noticeApp(t)
	seam.v.SetNotice("words", NoticeDialogue)
	a.step(appInput{PrimaryPressed: true, CursorX: (btn.Min.X + btn.Max.X) / 2,
		CursorY: (btn.Min.Y + btn.Max.Y) / 2}, now)

	if seam.v.held {
		t.Error("the map screen latched the button's press as the start of a gesture")
	}
}

// The world keeps being ASKED to advance behind an open notice, and a press
// away from the button is not a dismissal.
func TestAnOpenNoticeLeavesTheAdvanceAloneAndIsNotDismissedByAPressAtTheMap(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, seam := noticeApp(t)
	seam.v.SetNotice("words", NoticeDialogue)

	before := seam.ticks
	a.step(appInput{CursorX: 5, CursorY: 5, PrimaryPressed: true}, now)
	if seam.ticks == before {
		t.Error("the world was not advanced on a frame with a notice open")
	}
	if seam.advances != 0 {
		t.Error("a press away from the button advanced the notice")
	}
	if seam.v.held {
		t.Error("a press away from the button reached the map's gesture through an open popup")
	}
}

func TestAdvanceNavigatesTheSeamsDestination(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	t.Run("lost", func(t *testing.T) {
		a, seam := noticeApp(t)
		seam.dest = NoticeToMenu
		seam.v.SetNotice("you lost", NoticeOutcome)
		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenMenu {
			t.Fatalf("screen = %v, want ScreenMenu", a.Screen())
		}
		if a.flow.viewer != nil || a.flow.tick != nil || a.flow.advance != nil {
			t.Error("the map seam outlived the screen it belonged to")
		}
	})

	t.Run("won", func(t *testing.T) {
		a, seam := noticeApp(t)
		seam.dest, seam.msg = NoticeToMapList, "the town is not built"
		seam.v.SetNotice("you won", NoticeOutcome)
		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenPicker {
			t.Fatalf("screen = %v, want ScreenPicker", a.Screen())
		}
		if a.flow.msg != "the town is not built" {
			t.Fatalf("the map list says %q, want the seam's own message", a.flow.msg)
		}
	})
}

// A map screen the loader gave no advance seam — every map opened from the
// picker — closes whatever is open and stays put rather than becoming a screen
// with no way out.
func TestAdvanceWithNoSeamClosesTheNotice(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v", a.Screen())
	}
	a.flow.viewer.SetFont(panelFont())
	a.flow.viewer.SetNotice("orphan", NoticeDialogue)

	a.step(appInput{Enter: true}, now)
	if a.flow.viewer == nil || a.flow.viewer.NoticeOpen() {
		t.Error("a notice with no advance seam survived a press")
	}
	if a.Screen() != ScreenMap {
		t.Errorf("screen = %v, want ScreenMap", a.Screen())
	}
}

// The seam carries the destination, the message and — since 0131 — an
// opener: two scalars and a function value able to enter a map screen, never
// a mission number.
func TestNoticeSeamCarriesTheDestinationTheMessageAndAnOpener(t *testing.T) {
	adv := reflect.TypeOf(MapAdvance(nil))
	if adv.NumIn() != 1 || !adv.IsVariadic() || adv.NumOut() != 3 {
		t.Fatalf("MapAdvance is %v, want func(...NoticeAction) (NoticeDest, string, MapOpener)", adv)
	}
	if got, want := adv.In(0).Elem(), reflect.TypeOf(NoticeAction(0)); got != want {
		t.Errorf("the action is %v, want %v", got, want)
	}
	if k := adv.Out(0).Kind(); k != reflect.Int {
		t.Errorf("the destination is a %v, want an integer", k)
	}
	if k := adv.Out(1).Kind(); k != reflect.String {
		t.Errorf("the message is a %v, want a string", k)
	}
	if got, want := adv.Out(2), reflect.TypeOf(MapOpener(nil)); got != want {
		t.Errorf("the third result is %v, want MapOpener", got)
	}
	set := reflect.TypeOf((*Viewer).SetNotice)
	if k := set.In(1).Kind(); k != reflect.String {
		t.Errorf("SetNotice takes a %v first, want a string", k)
	}
	if k := set.In(2).Kind(); k != reflect.Uint8 {
		t.Errorf("SetNotice's kind is a %v, want an integer", k)
	}
}

// The dialogue geometry, drawn with THIS font, holds more than one line — which
// is what makes every wrap assertion above about a box a mission's words
// actually reach.
func TestTheDialogueAreaHoldsSeveralLines(t *testing.T) {
	f := panelFont()
	l := AuthoredDialogueLayout()
	if n := l.Text.Dy() / (f.Height() + l.Pitch); n < 2 {
		t.Fatalf("the text area holds %d lines at this font — the clamp test is vacuous", n)
	}
	if f.GlyphFor(' ') == nil {
		t.Fatal("the test font has no space record")
	}
}

// ---------------------------------------------------------------------------
// The window's two shapes.
// ---------------------------------------------------------------------------

// THE TWO SHAPES DIFFER IN EXACTLY TWO FIELDS. Everything else the window is —
// its box, its button, the word on it, the pitch and all eight colours — is
// asserted equal field by field, by comparing the two resolved values with those
// two fields normalised away. A comparison of named fields would silently pass a
// field added later; this cannot.
func TestTheTwoNoticeShapesDifferInExactlyTwoFields(t *testing.T) {
	with := AuthoredDialogueLayout().WithPortrait(true)
	without := AuthoredDialogueLayout().WithPortrait(false)

	if with.Portrait != image.Rect(30, 54, 118, 168) {
		t.Errorf("pane = %v, want 30,54-118,168", with.Portrait)
	}
	if with.Text != image.Rect(128, 36, 428, 171) {
		t.Errorf("text beside the pane = %v, want 128,36-428,171", with.Text)
	}
	if without.Text != image.Rect(48, 36, 428, 171) {
		t.Errorf("text with no pane = %v, want 48,36-428,171", without.Text)
	}
	if !without.Portrait.Empty() {
		t.Errorf("a window with no pane must carry no pane rectangle, got %v", without.Portrait)
	}

	a, b := with, without
	a.Text, b.Text = image.Rectangle{}, image.Rectangle{}
	a.Portrait, b.Portrait = image.Rectangle{}, image.Rectangle{}
	if a != b {
		t.Errorf("the two shapes differ somewhere other than the pane and the text area:\n with = %+v\n w/o  = %+v", a, b)
	}
}

// Asking a layout that HAS no pane for one yields the layout unchanged, rather
// than a window whose text area is empty. That is what lets the outcome notice
// carry no portrait fields instead of carrying them switched off.
func TestWithPortraitOnALayoutThatHasNone(t *testing.T) {
	o := AuthoredOutcomeLayout()
	if got := o.WithPortrait(true); got != o {
		t.Fatalf("WithPortrait(true) on a paneless layout changed it:\n got  = %+v\n want = %+v", got, o)
	}
	if got := o.WithPortrait(false); got != o {
		t.Fatalf("WithPortrait(false) on a paneless layout changed it")
	}
}

// WithPortrait returns a VALUE. The layout it was asked of is untouched, so two
// viewers showing opposite shapes cannot reach each other through it.
func TestWithPortraitDoesNotMutate(t *testing.T) {
	l := AuthoredDialogueLayout()
	before := l
	_, _ = l.WithPortrait(true), l.WithPortrait(false)
	if l != before {
		t.Fatal("WithPortrait mutated the layout it was asked of")
	}
}

// noticePaneLayout is the ink layout resolved WITH a pane, and with the pane
// painted in colours no other part of the window uses so the pane's own pixels
// are identifiable.
func noticePaneLayout() NoticeLayout {
	l := noticeInkLayout()
	l.Portrait = AuthoredDialogueLayout().Portrait
	l.TextBesidePortrait = AuthoredDialogueLayout().TextBesidePortrait
	l.PortraitAt = AuthoredDialogueLayout().PortraitAt
	l.PortraitWindow = AuthoredDialogueLayout().PortraitWindow
	l = l.WithPortrait(true)
	l.PortraitFill = color.RGBA{R: 0x40, A: 0xff}
	l.PortraitBorder = color.RGBA{R: 0x80, A: 0xff}
	return l
}

// A named speaker whose picture cannot be had gets a pane with nothing in it —
// PAINTED, not left as window fill. "No picture for this speaker" and "this
// window has no pane" must not be the same pixels.
func TestAnEmptyPaneIsStillPainted(t *testing.T) {
	f := panelFont()
	l := noticePaneLayout()
	img := RenderNotice(l, f, "who said that", nil)
	if img == nil {
		t.Fatal("RenderNotice composed nothing")
	}
	p := l.Portrait
	if got, want := img.RGBAAt(p.Min.X, p.Min.Y), l.PortraitBorder; got != want {
		t.Errorf("pane top-left = %v, want the pane border %v", got, want)
	}
	if got, want := img.RGBAAt(p.Min.X+1, p.Min.Y+1), l.PortraitFill; got != want {
		t.Errorf("pane interior = %v, want the pane fill %v", got, want)
	}
	// And a window with no pane paints nothing there.
	plain := RenderNotice(noticeInkLayout(), f, "who said that", nil)
	if got := plain.RGBAAt(p.Min.X+1, p.Min.Y+1); got == l.PortraitFill {
		t.Error("a window with no pane painted the pane's fill where the pane would be")
	}
}

// coordPicture is a 160x240 picture — every speaker's own size
// (`SPR256-PICT-043`) — whose every pixel NAMES ITS OWN COORDINATE: red is x,
// green is y. A pane pixel therefore says exactly which source pixel reached it,
// which is what makes a window's placement testable without a golden image.
func coordPicture() *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 160, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 160; x++ {
			pic.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), A: 0xff})
		}
	}
	return pic
}

// THE PANE SHOWS A WINDOW OF THE PICTURE AND NOT THE PICTURE (`REG-NPC-089`,
// `REG-NPC-091`): a 72x96 rectangle cut at the speaker's own origin and blitted
// to one point inside the pane, with the pane's own ground everywhere else.
//
// 0141 first centred the whole 160x240 picture in the 88x114 pane and clipped
// it, which is what the owner saw as a portrait sitting too far left.
func TestTheFaceWindowIsCutFromThePictureAndPlacedInThePane(t *testing.T) {
	f := panelFont()
	l := noticePaneLayout()
	p, at, w := l.Portrait, l.PortraitAt, l.PortraitWindow

	img := RenderNotice(l, f, "hello", coordPicture())
	for _, tc := range []struct {
		name   string
		dx, dy int
		want   color.RGBA
	}{
		{"the window's own top-left corner", 0, 0,
			color.RGBA{R: uint8(w.Min.X), G: uint8(w.Min.Y), A: 0xff}},
		{"its bottom-right", w.Dx() - 1, w.Dy() - 1,
			color.RGBA{R: uint8(w.Max.X - 1), G: uint8(w.Max.Y - 1), A: 0xff}},
		{"one column past its right edge is the pane's ground", w.Dx(), 0, l.PortraitFill},
		{"one row past its bottom edge is too", 0, w.Dy(), l.PortraitFill},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := img.RGBAAt(p.Min.X+at.X+tc.dx, p.Min.Y+at.Y+tc.dy); got != tc.want {
				t.Errorf("pane+(%d,%d) = %v, want %v", at.X+tc.dx, at.Y+tc.dy, got, tc.want)
			}
		})
	}
	// THE MARGIN IS THE POINT OF THE PLACEMENT: the pane's own corner is its
	// ground, and its border is untouched, whatever the picture holds.
	if got := img.RGBAAt(p.Min.X, p.Min.Y); got != l.PortraitBorder {
		t.Errorf("the pane's border = %v, want %v", got, l.PortraitBorder)
	}
	if got := img.RGBAAt(p.Min.X+1, p.Min.Y+1); got != l.PortraitFill {
		t.Errorf("the pane's own corner = %v, want the ground %v", got, l.PortraitFill)
	}
}

// THE WINDOW IS PER SPEAKER, and 48 of the 105 shipped records state their own
// (`REG-NPC-089`). Two speakers of ONE picture must show two different parts of
// it — the whole reason the key exists.
func TestASpeakersOwnWindowMovesWhatThePaneShows(t *testing.T) {
	f := panelFont()
	base := noticePaneLayout()
	pic := coordPicture()
	p, at := base.Portrait, base.PortraitAt

	// npc51's own shipped origin, and npc104's.
	l51 := base.WithFaceWindow(NoticeFaceWindow(40, 10))
	l104 := base.WithFaceWindow(NoticeFaceWindow(26, 23))

	got51 := RenderNotice(l51, f, "hello", pic).RGBAAt(p.Min.X+at.X, p.Min.Y+at.Y)
	got104 := RenderNotice(l104, f, "hello", pic).RGBAAt(p.Min.X+at.X, p.Min.Y+at.Y)
	if want := (color.RGBA{R: 40, G: 10, A: 0xff}); got51 != want {
		t.Errorf("a stated window (40,10) showed %v, want %v", got51, want)
	}
	if want := (color.RGBA{R: 26, G: 23, A: 0xff}); got104 != want {
		t.Errorf("a stated window (26,23) showed %v, want %v", got104, want)
	}
	// A STATED WINDOW IS FOUR ROWS TALLER THAN THE DEFAULT — 96 against the
	// literal 92 rectangle the engine passes when the key is absent — so the
	// bottom row of one is the pane's ground under the other.
	deep := RenderNotice(l51, f, "hello", pic).RGBAAt(p.Min.X+at.X, p.Min.Y+at.Y+95)
	if want := (color.RGBA{R: 40, G: 105, A: 0xff}); deep != want {
		t.Errorf("the stated window's last row showed %v, want %v", deep, want)
	}

	// AND AN EMPTY ONE CHANGES NOTHING: a record stating no origin hands over
	// the zero rectangle and the layout's own default stands.
	if base.WithFaceWindow(image.Rectangle{}) != base {
		t.Error("an empty window replaced the layout's default")
	}
}

// A WINDOW REACHING PAST THE PICTURE IS CUT, NOT REFUSED. `REG-NPC-089` bounds
// the shipped origins to a window that fits, and this is what a registry outside
// that domain draws: the overlapping part where it would have landed, and the
// pane's own ground for the rest.
func TestAFaceWindowOffThePictureDrawsWhatOverlaps(t *testing.T) {
	f := panelFont()
	l := noticePaneLayout().WithFaceWindow(NoticeFaceWindow(140, 220))
	p, at := l.Portrait, l.PortraitAt
	img := RenderNotice(l, f, "hello", coordPicture())

	if got, want := img.RGBAAt(p.Min.X+at.X, p.Min.Y+at.Y), (color.RGBA{R: 140, G: 220, A: 0xff}); got != want {
		t.Errorf("the overlapping corner = %v, want %v", got, want)
	}
	if got := img.RGBAAt(p.Min.X+at.X+20, p.Min.Y+at.Y); got != l.PortraitFill {
		t.Errorf("past the picture's right edge = %v, want the ground %v", got, l.PortraitFill)
	}
	// Wholly off it draws nothing at all — the same picture as no face.
	off := RenderNotice(noticePaneLayout().WithFaceWindow(NoticeFaceWindow(400, 400)), f, "hello", coordPicture())
	empty := RenderNotice(noticePaneLayout(), f, "hello", nil)
	if !bytes.Equal(off.Pix, empty.Pix) {
		t.Error("a window wholly off the picture drew something")
	}
}

func TestATransparentPictureLeavesThePanesOwnGround(t *testing.T) {
	f := panelFont()
	l := plainDialogueLayout().WithPortrait(true)
	if l.PortraitFill != (color.RGBA{A: 0xff}) {
		t.Fatalf("the authored pane's ground is %v, want opaque black", l.PortraitFill)
	}

	// A picture of the pane's own size with nothing in it but a single opaque
	// pixel: everything around it must stay the ground, opaque.
	pic := image.NewRGBA(image.Rect(0, 0, 160, 240))
	pic.SetRGBA(l.PortraitWindow.Min.X, l.PortraitWindow.Min.Y, color.RGBA{G: 0xff, A: 0xff})

	img := RenderNotice(l, f, "hello", pic)
	p, at := l.Portrait, l.PortraitAt
	if got, want := img.RGBAAt(p.Min.X+at.X, p.Min.Y+at.Y), (color.RGBA{G: 0xff, A: 0xff}); got != want {
		t.Errorf("the one opaque pixel = %v, want %v", got, want)
	}
	if got := img.RGBAAt(p.Min.X+at.X+1, p.Min.Y+at.Y); got != l.PortraitFill {
		t.Errorf("beside it = %v, want opaque black — a transparent source must not reach the pane", got)
	}
	// Every pixel of the pane's interior is OPAQUE: one alpha of zero anywhere
	// inside it is the defect this test exists for.
	for y := p.Min.Y + 1; y < p.Max.Y-1; y++ {
		for x := p.Min.X + 1; x < p.Max.X-1; x++ {
			if a := img.RGBAAt(x, y).A; a != 0xff {
				t.Fatalf("the pane at (%d,%d) has alpha %d — the window is a hole", x, y, a)
			}
		}
	}
}

// A face handed to a window with no pane is drawn nowhere: there is nowhere to
// draw it, and it must not land anywhere else.
func TestAFaceWithNoPaneIsDrawnNowhere(t *testing.T) {
	f := panelFont()
	blue := color.RGBA{B: 0xff, A: 0xff}
	face := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			face.SetRGBA(x, y, blue)
		}
	}
	l := noticeInkLayout()
	with := RenderNotice(l, f, "hello", face)
	without := RenderNotice(l, f, "hello", nil)
	if !bytes.Equal(with.Pix, without.Pix) {
		t.Fatal("a face changed a window that has no pane to put it in")
	}
}

// The narrower text area with a pane is the contract, and it must actually
// narrow the wrap: the same words break into more lines beside a pane than
// without one.
//
// It is asked of the WRAP and not of the clamped list, because the two shapes
// share a text HEIGHT — the clamp is the same number in both, so a long enough
// string is cut to it either way and the difference the pane makes disappears
// behind the very rule that hides it.
func TestThePaneNarrowsTheWrap(t *testing.T) {
	f := panelFont()
	s := strings.Repeat("abcdefghij ", 40)
	wide := NoticeLines(f, s, noticeInkLayout().Text.Dx())
	narrow := NoticeLines(f, s, noticePaneLayout().Text.Dx())
	if len(narrow) <= len(wide) {
		t.Fatalf("beside a pane the words broke into %d lines and without one into %d; the pane must narrow the area",
			len(narrow), len(wide))
	}
}
