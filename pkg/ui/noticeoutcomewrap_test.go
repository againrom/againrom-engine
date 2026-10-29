package ui

import (
	"bytes"
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/frame"
)

// An outcome body longer than one line: every wrapped line is drawn, inside
// the text rectangle and the window, and the buttons sit below the last line.
// The words are this file's own ASCII and the font is panelFont.

func longOutcomeBody(l NoticeLayout) string {
	perLine := l.Text.Dx() / (10*noticeAdvanceOf + noticeSpaceAdvance)
	lines := l.Text.Dy()/(panelFont().Height()+l.Pitch) + 3
	return strings.TrimSpace(strings.Repeat("abcdefghij ", lines*(perLine+1)))
}

func TestOutcomeBodyDrawsEveryLineInsideTheWindow(t *testing.T) {
	for _, kind := range []NoticeKind{NoticeFailure, NoticeSuccess} {
		v := noticeViewer(t)
		f := panelFont()
		stored := v.noticeLayouts[kind]
		pitch := f.Height() + stored.Pitch
		body := longOutcomeBody(stored)
		if n := len(NoticeLines(f, body, stored.Text.Dx())); n <= stored.Text.Dy()/pitch {
			t.Fatalf("kind %v setup: %d lines already fit the stored text rectangle", kind, n)
		}
		v.SetNotice(body, kind)
		l := v.noticeLayout()
		all := noticeWrap(l, f, body)
		drawn := NoticeLayoutOf(l, f, body)
		if !reflect.DeepEqual(drawn, all) {
			t.Fatalf("kind %v: %d of %d lines drawn", kind, len(drawn), len(all))
		}
		if l.Text.Dx() != stored.Text.Dx() || l.Text.Min != stored.Text.Min {
			t.Errorf("kind %v: text rectangle %v moved or changed width from %v", kind, l.Text, stored.Text)
		}
		box := image.Rectangle{Max: l.Box.Size()}
		last := l.Text.Min.Y + (len(all)-1)*pitch + f.Height()
		if !l.Text.In(box) || last > l.Text.Max.Y {
			t.Errorf("kind %v: last line ends at %d, text %v, window %v", kind, last, l.Text, box)
		}
		for i, ln := range drawn {
			if w := f.Advance(ln); w > l.Text.Dx() {
				t.Errorf("kind %v: line %d is %d wide, text is %d", kind, i, w, l.Text.Dx())
			}
		}
		for _, b := range []image.Rectangle{l.Button, l.SecondaryButton} {
			if !b.In(box) || b.Min.Y < l.Text.Max.Y {
				t.Errorf("kind %v: button %v is not below the text %v inside %v", kind, b, l.Text, box)
			}
		}
		if !l.Box.In(image.Rect(0, 0, frame.W, frame.H)) {
			t.Errorf("kind %v: window %v leaves the design space", kind, l.Box)
		}
		pic, ok := v.HeadlessNoticePicture()
		if !ok || pic.Bounds().Size() != l.Box.Size() {
			t.Errorf("kind %v: composed picture %v, window %v", kind, pic.Bounds(), l.Box)
		}
	}
}

// A body that fits keeps the stored window and composes the same bytes.
func TestShortOutcomeBodyKeepsTheWindow(t *testing.T) {
	f := panelFont()
	for _, stored := range []NoticeLayout{AuthoredFailureLayout(), AuthoredSuccessLayout(), AuthoredOutcomeLayout()} {
		l := stored.WithBody(f, "Mission Failed")
		if !reflect.DeepEqual(l, stored) {
			t.Fatalf("short body changed the layout: %+v", l)
		}
		plain := stored
		plain.GrowText = false
		if !bytes.Equal(RenderNotice(l, f, "Mission Failed", nil).Pix, RenderNotice(plain, f, "Mission Failed", nil).Pix) {
			t.Error("short body composed different bytes")
		}
	}
}

// The dialogue keeps its pages: WithBody leaves a layout without GrowText.
func TestWithBodyLeavesTheDialogueLayout(t *testing.T) {
	l := AuthoredDialogueLayout()
	if got := l.WithBody(panelFont(), longOutcomeBody(l)+" "+longOutcomeBody(l)); !reflect.DeepEqual(got, l) {
		t.Fatalf("dialogue layout changed: %+v", got)
	}
}

// The click goes to the button that was painted, after the window grew.
func TestGrownOutcomeButtonsAreHitWhereDrawn(t *testing.T) {
	v := noticeViewer(t)
	stored := v.noticeLayouts[NoticeFailure]
	v.failureLoadCheck = func() bool { return true }
	v.SetNotice(longOutcomeBody(stored), NoticeFailure)
	l := v.noticeLayout()
	if l.Button == stored.Button {
		t.Fatal("setup: the button did not move")
	}
	_, at, scale, ok := v.noticePresent()
	if !ok || scale != 1 {
		t.Fatalf("setup: present %v scale %v", ok, scale)
	}
	centre := func(r image.Rectangle) (int, int) {
		c := r.Add(at).Min.Add(r.Size().Div(2))
		return c.X, c.Y
	}
	for _, tc := range []struct {
		r    image.Rectangle
		want int
	}{{l.Button, noticePrimaryButton}, {l.SecondaryButton, noticeSecondButton}} {
		x, y := centre(tc.r)
		if id, ok := v.noticeButtonIDAt(x, y); !ok || id != tc.want {
			t.Errorf("button %v: hit %d %v, want %d", tc.r, id, ok, tc.want)
		}
	}
}

// A body taller than the design space grows the window to its height and no
// further; the clamp then keeps the lines inside.
func TestOutcomeWindowStopsAtTheDesignSpace(t *testing.T) {
	f := panelFont()
	stored := AuthoredFailureLayout()
	body := strings.Repeat(longOutcomeBody(stored)+" ", 40)
	l := stored.WithBody(f, body)
	if l.Box.Dy() <= stored.Box.Dy() || !l.Box.In(image.Rect(0, 0, frame.W, frame.H)) {
		t.Fatalf("window %v from %v", l.Box, stored.Box)
	}
	pitch := f.Height() + l.Pitch
	if frame.H-l.Box.Dy() >= pitch {
		t.Errorf("window %v stopped a whole line short of the design space", l.Box)
	}
	if got := len(NoticeLayoutOf(l, f, body)); got != l.Text.Dy()/pitch {
		t.Errorf("%d lines drawn, text holds %d", got, l.Text.Dy()/pitch)
	}
}
