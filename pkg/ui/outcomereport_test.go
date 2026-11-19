package ui

import (
	"image"
	"testing"
)

// screenBox is the open kind's popup rectangle on the live frame, in frame pixels.
func screenBox(v *Viewer, kind NoticeKind) image.Rectangle {
	v.noticeKind = kind
	l := v.noticeLayout()
	ox, oy := v.noticePlace().Origin()
	return l.Box.Add(image.Pt(int(ox), int(oy)))
}

func TestSecondGameSuccessPopupAndButtons(t *testing.T) {
	for _, c := range []struct {
		size image.Point
		want image.Rectangle
	}{
		{image.Pt(640, 480), image.Rect(172, 156, 468, 324)},
		{image.Pt(800, 600), image.Rect(252, 216, 548, 384)},
		{image.Pt(1024, 768), image.Rect(364, 300, 660, 468)},
	} {
		_, v := inspectionFixture(t, c.size)
		v.frameW, v.frameH = c.size.X, c.size.Y
		v.SetMissionScreen(MissionScreen{NoticeOnScreen: true, Outcome: SecondGameOutcomeReport()})
		if got := screenBox(v, NoticeSuccess); got != c.want {
			t.Fatalf("%v popup %v, want %v", c.size, got, c.want)
		}
		l := v.noticeLayout()
		if l.Button != image.Rect(40, 84, 248, 108) || l.SecondaryButton != image.Rect(40, 108, 248, 132) {
			t.Fatalf("%v buttons %v %v", c.size, l.Button, l.SecondaryButton)
		}
	}
}

func TestSecondGameFailurePageGeometry(t *testing.T) {
	_, v := inspectionFixture(t, image.Pt(640, 480))
	v.frameW, v.frameH = 640, 480
	v.SetMissionScreen(MissionScreen{NoticeOnScreen: true, Outcome: SecondGameOutcomeReport()})
	got := screenBox(v, NoticeFailure)
	if got.Min.X != 76 || got.Dx() != 488 {
		t.Fatalf("failure page %v, want x 76 width 488", got)
	}
	l := v.noticeLayout()
	if l.Button.Min.X != 61 || l.Button.Max.X != 427 || l.Button.Dy() != 24 || l.SecondaryButton.Min.Y != l.Button.Max.Y {
		t.Fatalf("failure buttons %v %v", l.Button, l.SecondaryButton)
	}
}

func TestFirstGameOutcomePagesAreAuthored(t *testing.T) {
	_, v := inspectionFixture(t, image.Pt(1024, 768))
	v.SetMissionScreen(MissionScreen{NoticeOnScreen: true})
	s, f := AuthoredSuccessLayout(), AuthoredFailureLayout()
	for _, c := range []struct {
		kind NoticeKind
		want Dialog
	}{{NoticeSuccess, s}, {NoticeFailure, f}} {
		v.noticeKind = c.kind
		l := v.noticeLayout()
		if l.Box != c.want.Box || l.Text != c.want.Text || l.Button != c.want.Button ||
			l.SecondaryButton != c.want.SecondaryButton || l.FrameInset != 0 || l.CenterText {
			t.Fatalf("kind %d changed: %+v", c.kind, l)
		}
	}
	if s.Box != image.Rect(128, 200, 512, 380) {
		t.Fatalf("first-game success box %v", s.Box)
	}
}

func TestMovieRectAtEachMode(t *testing.T) {
	for _, c := range []struct {
		screen image.Point
		want   image.Rectangle
	}{
		{image.Pt(640, 480), image.Rect(0, 60, 640, 420)},
		{image.Pt(800, 600), image.Rect(80, 120, 720, 480)},
		{image.Pt(1024, 768), image.Rect(192, 204, 832, 564)},
	} {
		if got := movieRect(c.screen, 640, 360); got != c.want {
			t.Fatalf("%v movie %v, want %v", c.screen, got, c.want)
		}
	}
	if got := movieRect(image.Pt(640, 480), 640, 480); got != image.Rect(0, 0, 640, 480) {
		t.Fatalf("full-height movie %v", got)
	}
}

func TestCutsceneScreenFollowsProfile(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.frameW, v.frameH = 1024, 768
	v.SetMissionScreen(MissionScreen{})
	if got := a.cutsceneScreen(); got != image.Pt(640, 480) {
		t.Fatalf("first game movie screen %v", got)
	}
	v.SetMissionScreen(MissionScreen{MovieOnScreen: true})
	if got := a.cutsceneScreen(); got != image.Pt(1024, 768) {
		t.Fatalf("second game movie screen %v", got)
	}
}
