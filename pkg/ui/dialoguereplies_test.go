package ui

import (
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
)

func replyTexts(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("reply %d", i+1)
	}
	return out
}

// replyApp is a map screen with a dialogue page that offers n replies.
func replyApp(t *testing.T, replies []string) (*App, *noticeSeam) {
	t.Helper()
	a, seam := noticeApp(t)
	seam.v.SetDialogue(Dialogue{Text: "Strangers! Will you drive them out?", Replies: replies})
	if seam.v.NoticeReplyCount() != len(replies) {
		t.Fatalf("setup: %d reply rows, want %d", seam.v.NoticeReplyCount(), len(replies))
	}
	return a, seam
}

// rowCentre is the window point at the middle of reply row i (from 1).
func rowCentre(v *Viewer, i int) image.Point {
	l := v.noticeLayout()
	r := l.ReplyRows[i-1].Add(l.Box.Min)
	return r.Min.Add(r.Max).Div(2)
}

func TestReplyKeyChoosesThatReply(t *testing.T) {
	a, seam := replyApp(t, replyTexts(3))
	if err := a.HeadlessKey("2"); err != nil {
		t.Fatal(err)
	}
	if want := []NoticeAction{NoticeReply2}; !reflect.DeepEqual(seam.actions, want) {
		t.Errorf("actions = %v, want %v", seam.actions, want)
	}
}

func TestReplyKeyBeyondTheRowsDoesNothing(t *testing.T) {
	a, seam := replyApp(t, replyTexts(3))
	if err := a.HeadlessKey("4"); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 || seam.v.NoticeReplyCount() != 3 {
		t.Errorf("key 4 on three replies: actions %v, rows %d", seam.actions, seam.v.NoticeReplyCount())
	}
}

func TestReplyDownThenEnterChoosesTheLitReply(t *testing.T) {
	a, seam := replyApp(t, replyTexts(3))
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 {
		t.Fatalf("Enter with no reply lit chose %v", seam.actions)
	}
	for _, step := range []struct {
		key string
		lit int
	}{{"down", 1}, {"down", 2}, {"down", 3}, {"down", 1}, {"up", 3}, {"up", 2}} {
		if err := a.HeadlessKey(step.key); err != nil {
			t.Fatal(err)
		}
		if seam.v.noticeReplyLit != step.lit {
			t.Fatalf("after %s the lit reply is %d, want %d", step.key, seam.v.noticeReplyLit, step.lit)
		}
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if want := []NoticeAction{NoticeReply2}; !reflect.DeepEqual(seam.actions, want) {
		t.Errorf("actions = %v, want %v", seam.actions, want)
	}
}

func TestReplyClickNeedsPressAndReleaseOnTheSameRow(t *testing.T) {
	a, seam := replyApp(t, replyTexts(4))
	third, first := rowCentre(seam.v, 3), rowCentre(seam.v, 1)
	for _, step := range []struct {
		action string
		at     image.Point
	}{{"press", third}, {"release", first}} {
		if err := a.HeadlessPointer(step.action, step.at.X, step.at.Y); err != nil {
			t.Fatal(err)
		}
	}
	if len(seam.actions) != 0 {
		t.Fatalf("press on row 3 and release on row 1 chose %v", seam.actions)
	}
	for _, step := range []struct {
		action string
		at     image.Point
	}{{"press", third}, {"release", third}} {
		if err := a.HeadlessPointer(step.action, step.at.X, step.at.Y); err != nil {
			t.Fatal(err)
		}
	}
	if want := []NoticeAction{NoticeReply3}; !reflect.DeepEqual(seam.actions, want) {
		t.Errorf("actions = %v, want %v", seam.actions, want)
	}
}

func TestReplyPointerLightsTheRowUnderIt(t *testing.T) {
	a, seam := replyApp(t, replyTexts(3))
	at := rowCentre(seam.v, 2)
	if err := a.HeadlessPointer("hover", at.X, at.Y); err != nil {
		t.Fatal(err)
	}
	if seam.v.noticeReplyLit != 2 {
		t.Errorf("lit reply = %d, want 2", seam.v.noticeReplyLit)
	}
	body := seam.v.noticeLayout().Box.Min.Add(image.Pt(4, 4))
	if err := a.HeadlessPointer("press", body.X, body.Y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", body.X, body.Y); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 {
		t.Errorf("a click on the panel's body chose %v", seam.actions)
	}
}

func TestReplyEscapeDoesNothing(t *testing.T) {
	a, seam := replyApp(t, replyTexts(2))
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if len(seam.actions) != 0 || !seam.v.NoticeOpen() || seam.v.NoticeReplyCount() != 2 {
		t.Errorf("Escape: actions %v, open %v, rows %d", seam.actions, seam.v.NoticeOpen(), seam.v.NoticeReplyCount())
	}
}

func TestOneReplyKeepsTheButtonAndEscapeStillCloses(t *testing.T) {
	a, seam := replyApp(t, nil)
	seam.v.SetDialogue(Dialogue{Text: "One word.", Replies: replyTexts(1)})
	if seam.v.NoticeReplyCount() != 0 || seam.v.noticeLayout().Button.Empty() {
		t.Fatalf("one reply: rows %d, button %v", seam.v.NoticeReplyCount(), seam.v.noticeLayout().Button)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if want := []NoticeAction{NoticeAdvance}; !reflect.DeepEqual(seam.actions, want) {
		t.Errorf("actions = %v, want %v", seam.actions, want)
	}
}

// The lit row is a colour and a pressed frame: lighting a reply changes the
// picture, and the rows sit inside the box apart from each other.
func TestReplyLightChangesThePictureAndRowsDoNotOverlap(t *testing.T) {
	v := noticeViewer(t)
	v.SetDialogue(Dialogue{Text: "Strangers!", Replies: replyTexts(4)})
	l := v.noticeLayout()
	inner := image.Rect(0, 0, l.Box.Dx(), l.Box.Dy())
	for i, r := range l.ReplyRows {
		if !r.In(inner) || r.Min.Y < l.Text.Max.Y {
			t.Errorf("row %d %v outside %v or above the text end %d", i+1, r, inner, l.Text.Max.Y)
		}
		if i > 0 && r.Overlaps(l.ReplyRows[i-1]) {
			t.Errorf("rows %d and %d overlap", i, i+1)
		}
	}
	dark, _, _, _ := v.noticePresent()
	before := append([]byte(nil), dark.Pix...)
	v.SetDialogueReplyLit(1)
	lit, _, _, _ := v.noticePresent()
	if reflect.DeepEqual(before, lit.Pix) {
		t.Error("lighting reply 1 left the picture unchanged")
	}
	if p := layoutNoticeBox(v.noticeLayout(), v.font).Pushes; !p[0].sunk() || !p[0].Hover || p[1].sunk() || p[1].Hover {
		t.Error("the lit row is not hover-inked and sunk, or another row is")
	}
}

// A four-reply page with a long reply grows the box in whole 64-pixel steps
// and keeps it centred on the mission canvas, at the smallest window and a
// large one.
func TestReplyBoxGrowsInStepsAndStaysCentred(t *testing.T) {
	long := strings.Repeat("a reply that runs past the end of its row ", 24)
	huge := strings.Repeat("a reply that runs past the end of its row ", 60)
	cases := []struct {
		name    string
		replies []string
		grows   bool
	}{
		{"two short", replyTexts(2), false},
		{"four, two long", []string{"one", long, long, "four"}, true},
		{"four, one huge", []string{"one", huge, "three", "four"}, true},
	}
	for _, size := range []image.Point{{640, 480}, {1920, 1080}} {
		for _, portrait := range []bool{false, true} {
			for _, tc := range cases {
				t.Run(fmt.Sprintf("%dx%d/portrait=%v/%s", size.X, size.Y, portrait, tc.name), func(t *testing.T) {
					v := noticeViewer(t)
					v.Layout(size.X, size.Y)
					base := AuthoredDialogueLayout().WithPortrait(portrait)
					v.SetDialogue(Dialogue{Text: "Strangers! Will you drive them out?", Portrait: portrait, Replies: tc.replies})
					l := v.noticeLayout()
					if len(l.ReplyRows) != len(tc.replies) {
						t.Fatalf("%d rows for %d replies", len(l.ReplyRows), len(tc.replies))
					}
					grow := l.Box.Dy() - base.Box.Dy()
					if grow < 0 || grow%replyBoxStep != 0 {
						t.Errorf("box grew by %d, not a whole step of %d", grow, replyBoxStep)
					}
					if tc.grows && grow == 0 {
						t.Errorf("box did not grow")
					}
					if !tc.grows && grow != 0 {
						t.Errorf("box grew by %d", grow)
					}
					if l.Box.Min.Y+l.Box.Max.Y != base.Box.Min.Y+base.Box.Max.Y || l.Box.Min.X != base.Box.Min.X || l.Box.Dx() != base.Box.Dx() {
						t.Errorf("box %v is not centred like %v", l.Box, base.Box)
					}
					_, at, scale, ok := v.noticePresent()
					if !ok || scale != 1 {
						t.Fatalf("present %v scale %v", ok, scale)
					}
					if want := image.Pt((v.cam.ViewW-640)/2+l.Box.Min.X, (v.cam.ViewH-480)/2+l.Box.Min.Y); at != want {
						t.Errorf("origin %v, want %v", at, want)
					}
					for i, r := range l.ReplyRows {
						if !r.In(image.Rect(0, 0, l.Box.Dx(), l.Box.Dy())) {
							t.Errorf("row %d %v outside the box", i+1, r)
						}
					}
					if tc.grows && l.ReplyRows[1].Dy() <= l.ReplyRows[0].Dy() {
						t.Errorf("the long reply's row %v is not taller than a short one's %v", l.ReplyRows[1], l.ReplyRows[0])
					}
				})
			}
		}
	}
}

func TestPageWithoutRepliesKeepsItsLayout(t *testing.T) {
	v := noticeViewer(t)
	v.SetDialogue(Dialogue{Text: "Strangers!"})
	l, want := v.noticeLayout(), AuthoredDialogueLayout()
	if v.NoticeReplyCount() != 0 || l.Box != want.Box || l.Button != want.Button || l.Text != want.Text {
		t.Errorf("a page with no replies changed the layout: rows %d, box %v, button %v", v.NoticeReplyCount(), l.Box, l.Button)
	}
}

// A save is closed while the replies show, by the gate that closes every
// save point under an open notice.
func TestSaveGateClosesWhileRepliesShow(t *testing.T) {
	a, _ := replyApp(t, replyTexts(3))
	for _, k := range []saveKind{saveKey, saveKeyWithLoad, saveQuickKey, saveAutosave} {
		if got := a.flow.savePoint(k); got.open() {
			t.Errorf("save kind %d is open with replies showing", k)
		}
	}
	if got := a.flow.savePoint(saveKey); got.row != pointMapPopup {
		t.Errorf("save key is closed by row %v, want the open notice's row %v", got.row, pointMapPopup)
	}
	if err := a.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMap || a.flow.viewer.NoticeReplyCount() != 3 {
		t.Errorf("the save key left the replies: screen %v, rows %d", a.Screen(), a.flow.viewer.NoticeReplyCount())
	}
}

func TestDisabledReplyIsNotChoosableByKeyOrClick(t *testing.T) {
	a, seam := noticeApp(t)
	seam.v.SetDialogue(Dialogue{Text: "Words", Replies: replyTexts(3), RepliesOff: []bool{false, true, false}})
	if !seam.v.NoticeReplyOff(2) || seam.v.NoticeReplyOff(1) {
		t.Fatal("setup: the second row is not the greyed one")
	}
	if err := a.HeadlessKey("2"); err != nil {
		t.Fatal(err)
	}
	row := rowCentre(seam.v, 2)
	for _, action := range []string{"press", "release"} {
		if err := a.HeadlessPointer(action, row.X, row.Y); err != nil {
			t.Fatal(err)
		}
	}
	if len(seam.actions) != 0 {
		t.Fatalf("a greyed row chose %v", seam.actions)
	}
	for _, step := range []struct {
		key string
		lit int
	}{{"down", 1}, {"down", 3}, {"down", 1}, {"up", 3}} {
		if err := a.HeadlessKey(step.key); err != nil {
			t.Fatal(err)
		}
		if seam.v.noticeReplyLit != step.lit {
			t.Fatalf("after %s the lit reply is %d, want %d", step.key, seam.v.noticeReplyLit, step.lit)
		}
	}
}
