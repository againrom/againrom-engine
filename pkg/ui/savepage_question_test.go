package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// questionConfirm is a delete confirmation that asks in a window, laid out as
// the second game's profile lays it out.
func questionConfirm() PageConfirm {
	return PageConfirm{
		Before: "Are you sure you want to delete game '", After: "' ?", Yes: "Yes", No: "No",
		Window: NoticeShape{Size: image.Pt(288, 224), TextTop: 57, TextWidth: 208, Pitch: 17,
			Answer: image.Pt(82, 24), AnswerGap: 46, AnswerBottom: 28},
	}
}

// questionSaveApp is a save page that asks in the window, over two saves,
// with Beta chosen and its Delete asked.
func questionSaveApp(t *testing.T) (*App, *pageSeamsFake) {
	t.Helper()
	fake := &pageSeamsFake{entries: []SaveEntry{
		{Name: "game0001.sav", Label: "Alpha"}, {Name: "game0000.sav", Label: "Beta"}}}
	a := lookApp(t, fake)
	font := solidFont15()
	a.SetWords(AuthoredWords(), font, func(r rune) (byte, bool) { return textinput.EncodeRune(r, font.Selector) })
	page := lookPage(fake)
	page.Confirm = questionConfirm()
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(2); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("delete"); err != nil {
		t.Fatal(err)
	}
	if v := a.flow.pages.save.view(); !v.asking() {
		t.Fatal("Delete did not open the question window")
	}
	return a, fake
}

// questionAnswers are the answer buttons drawn over the page.
func questionAnswers(t *testing.T, a *App, v pageView) map[string]image.Rectangle {
	t.Helper()
	out := map[string]image.Rectangle{}
	widgetRecord = func(c widgetCall) {
		if b, ok := c.state.(pushButton); ok && c.kind == widgetPushButton {
			out[b.Label] = c.rect
		}
	}
	defer func() { widgetRecord = nil }()
	if _, err := a.composePage(v); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestSecondGameDeleteAsksInASmallWindowWithYesAndNo: Delete opens a window
// 288x224 centred on the page, the question wrapped into centred lines 17
// pixels apart from 57 below its top, and Yes and No 82x24, 46 apart, 28
// above its bottom. The page's own body still shows the name and the list.
// The old page drew the question in its body and kept Ok, Delete and Cancel
// as the answers: no Yes or No was drawn.
func TestSecondGameDeleteAsksInASmallWindowWithYesAndNo(t *testing.T) {
	a, _ := questionSaveApp(t)
	v := a.flow.pages.save.view()
	font := a.flow.menuFont
	w := &v.ui.question
	l := w.layout(font, v.page.Panel)
	box := image.Rect(206, 128, 494, 352)
	if l.Body != box {
		t.Fatalf("window %v, want %v centred on the page %v", l.Body, box, v.page.Panel)
	}
	answers := questionAnswers(t, a, v)
	if answers["Yes"] != image.Rect(245, 300, 327, 324) || answers["No"] != image.Rect(373, 300, 455, 324) {
		t.Fatalf("answers %v", answers)
	}
	lines := w.noticeLines(font, l, func(s string) string { return s })
	var joined []string
	for _, ln := range lines {
		var b strings.Builder
		for _, r := range ln.runs {
			b.WriteString(r.Text)
		}
		joined = append(joined, b.String())
	}
	if want := "Are you sure you want to delete game 'Beta' ?"; strings.Join(joined, " ") != want || len(lines) != 2 {
		t.Fatalf("question lines %q, want %q in two lines", joined, want)
	}
	calls := text.Record(func() {
		if _, err := a.composePage(v); err != nil {
			t.Fatal(err)
		}
	})
	for i, s := range joined {
		y := box.Min.Y + 57 + 17*i
		var xs []int
		for _, c := range calls {
			if c.Y == y && c.Color == townShellText && c.X >= box.Min.X && c.X < box.Max.X {
				xs = append(xs, c.X)
			}
		}
		width := font.Advance(s)
		if len(xs) == 0 || slices.Min(xs) != (box.Min.X+box.Max.X)/2-width/2 {
			t.Fatalf("line %d %q at y %d: glyph x %v, want centred in %v", i, s, y, xs, box)
		}
	}
	if !slices.ContainsFunc(calls, func(c text.DrawCall) bool { return c.Y < box.Min.Y }) {
		t.Fatal("the page under the window drew nothing")
	}
}

// TestSecondGameDeleteNoAndEscapeKeepTheRowChosen: No and Esc close the
// window with nothing deleted and the row still chosen.
func TestSecondGameDeleteNoAndEscapeKeepTheRowChosen(t *testing.T) {
	for _, way := range []string{"no", "escape"} {
		t.Run(way, func(t *testing.T) {
			a, fake := questionSaveApp(t)
			if way == "no" {
				v := a.flow.pages.save.view()
				no := questionAnswers(t, a, v)["No"].Min.Add(image.Pt(4, 4))
				for _, action := range []string{"press", "release"} {
					if err := a.HeadlessPointer(action, no.X, no.Y); err != nil {
						t.Fatal(err)
					}
				}
			} else if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			s := a.flow.pages.save
			if a.Screen() != ScreenSave || s.ui.confirm || len(fake.entries) != 2 || s.list.Selection() != 2 || string(s.name) != "Beta" {
				t.Fatalf("after %s: screen %v asking %v saves %d row %d name %q", way, a.Screen(), s.ui.confirm, len(fake.entries), s.list.Selection(), string(s.name))
			}
		})
	}
}

// TestSecondGameDeleteYesDeletes: Yes, by click or Enter, deletes the save
// and leaves the Empty Slot chosen with an empty name, the window closed.
// A click on the page's Ok under the open window deletes nothing; the old
// page took Ok as the answer.
func TestSecondGameDeleteYesDeletes(t *testing.T) {
	for _, answer := range []string{"click", "enter"} {
		t.Run(answer, func(t *testing.T) {
			a, fake := questionSaveApp(t)
			v := a.flow.pages.save.view()
			a.flow.pages.save.page.Buttons[loadOKButton].Rect = image.Rect(120, 400, 216, 424)
			for _, action := range []string{"press", "release"} {
				if err := a.HeadlessPointer(action, 130, 410); err != nil {
					t.Fatal(err)
				}
			}
			if len(fake.entries) != 2 || !a.flow.pages.save.ui.confirm {
				t.Fatalf("the page's Ok under the window deleted: %d saves, asking %v", len(fake.entries), a.flow.pages.save.ui.confirm)
			}
			if answer == "click" {
				yes := questionAnswers(t, a, v)["Yes"].Min.Add(image.Pt(4, 4))
				for _, action := range []string{"press", "release"} {
					if err := a.HeadlessPointer(action, yes.X, yes.Y); err != nil {
						t.Fatal(err)
					}
				}
			} else if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			s := a.flow.pages.save
			if len(fake.entries) != 1 || fake.entries[0].Label != "Alpha" || s.ui.confirm || s.list.Selection() != 0 || len(s.name) != 0 {
				t.Fatalf("after Yes: saves %+v asking %v row %d name %q", fake.entries, s.ui.confirm, s.list.Selection(), string(s.name))
			}
		})
	}
}

// TestSecondGameLoadPageAsksInTheSameWindow: the Load page's Delete key opens
// the same window; No keeps the row, Yes deletes it.
func TestSecondGameLoadPageAsksInTheSameWindow(t *testing.T) {
	a, calls := loadScrollApp(t, 3)
	font := solidFont15()
	a.SetWords(AuthoredWords(), font, nil)
	page := firstLoadPage(LoadWindowWords{})
	page.Confirm = questionConfirm()
	a.SetSavePages(SavePages{Load: page})
	a.flow.openLoad(a.flow.loadBack)
	a.flow.loadList.Select(1)
	loadScrollKey(t, a, "delete")
	v := a.flow.loadView()
	if !v.asking() || len(calls.prepared) != 1 || calls.prepared[0] != "slot-01.sav" {
		t.Fatalf("Delete key: asking %v prepared %v", v.asking(), calls.prepared)
	}
	answers := questionAnswers(t, a, v)
	no := answers["No"].Min.Add(image.Pt(4, 4))
	for _, action := range []string{"press", "release"} {
		loadScrollPointer(t, a, action, no.X, no.Y)
	}
	if a.flow.loadUI.confirm || calls.removed != 0 || a.flow.loadList.Selection() != 1 {
		t.Fatalf("No: asking %v removed %d row %d", a.flow.loadUI.confirm, calls.removed, a.flow.loadList.Selection())
	}
	loadScrollKey(t, a, "delete")
	yes := answers["Yes"].Min.Add(image.Pt(4, 4))
	for _, action := range []string{"press", "release"} {
		loadScrollPointer(t, a, action, yes.X, yes.Y)
	}
	if a.flow.loadUI.confirm || calls.removed != 1 {
		t.Fatalf("Yes: asking %v removed %d", a.flow.loadUI.confirm, calls.removed)
	}
}

// TestFirstGameLoadPageStillAsksInItsBody: the first game's Load page asks in
// its own body, Before at its place and the label in its box, with no window
// and the page's Ok as the answer.
func TestFirstGameLoadPageStillAsksInItsBody(t *testing.T) {
	a, calls := loadScrollApp(t, 3)
	a.SetWords(AuthoredWords(), solidFont15(), nil)
	loadScrollKey(t, a, "delete")
	v := a.flow.loadView()
	if v.asking() || !v.confirmingInPage() {
		t.Fatalf("first game: asking %v in page %v", v.asking(), v.confirmingInPage())
	}
	c := v.page.Confirm
	answers := questionAnswers(t, a, v)
	if _, ok := answers["Yes"]; ok {
		t.Fatalf("first game drew a Yes: %v", answers)
	}
	calls2 := text.Record(func() {
		if _, err := a.composePage(v); err != nil {
			t.Fatal(err)
		}
	})
	if !slices.ContainsFunc(calls2, func(d text.DrawCall) bool { return d.X == c.At.X && d.Y == c.At.Y }) {
		t.Fatalf("no question text at %v", c.At)
	}
	loadScrollKey(t, a, "enter")
	if a.flow.loadUI.confirm || calls.removed != 1 {
		t.Fatalf("Ok: asking %v removed %d", a.flow.loadUI.confirm, calls.removed)
	}
}
