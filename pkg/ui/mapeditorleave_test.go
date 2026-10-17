package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/words"

	"github.com/hajimehoshi/ebiten/v2"
)

func leaveEditor(t *testing.T, dirty bool) (*MapEditor, *reportEdits) {
	t.Helper()
	edits := &reportEdits{dirty: dirty}
	e := reportEditor(t, edits)
	e.SetWords(words.For(""))
	return e, edits
}

func leaveButton(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	l := layoutPopup(e.leaveSpec(), e.font, e.bodyFont())
	if n >= len(l.Buttons) {
		t.Fatalf("question has %d buttons, want button %d", len(l.Buttons), n)
	}
	c := l.Buttons[n].Min.Add(l.Buttons[n].Size().Div(2))
	now := time.Now()
	e.Pointer(Input{CursorX: c.X, CursorY: c.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: c.X, CursorY: c.Y}, now.Add(time.Millisecond))
}

func asked(e *MapEditor) bool { return e.prompt.kind == promptLeave }

func TestCloseKeyAsksWhenDirtyAndClosesWhenClean(t *testing.T) {
	e, _ := leaveEditor(t, true)
	if err := e.Key("close"); err != nil || e.ExitAsked() || !asked(e) {
		t.Fatalf("dirty close: err %v exit %v asked %v", err, e.ExitAsked(), asked(e))
	}
	if e.message != "" {
		t.Fatalf("a status message stayed: %q", e.message)
	}
	e, _ = leaveEditor(t, false)
	if err := e.Key("close"); err != ebiten.Termination || asked(e) {
		t.Fatalf("clean close: err %v asked %v", err, asked(e))
	}
}

func TestEscapeLastStepAsksWhenDirty(t *testing.T) {
	e, _ := leaveEditor(t, true)
	if err := e.Key("escape"); err != nil || !asked(e) || e.ExitAsked() {
		t.Fatalf("dirty Esc: err %v asked %v", err, asked(e))
	}
	e, _ = leaveEditor(t, false)
	if err := e.Key("escape"); err != ebiten.Termination {
		t.Fatalf("clean Esc: %v", err)
	}
}

func TestCloseWithoutSavingTerminatesAndWritesNothing(t *testing.T) {
	e, edits := leaveEditor(t, true)
	_ = e.Key("close")
	leaveButton(t, e, leaveGo)
	if !e.ExitAsked() || e.Update() != ebiten.Termination {
		t.Fatal("Close without saving did not close the editor")
	}
	if edits.saves != 0 || !e.Dirty() {
		t.Fatalf("Close without saving wrote %d times, dirty %v", edits.saves, e.Dirty())
	}
}

func TestCloseAgainWhileAskedClosesWithoutSaving(t *testing.T) {
	e, edits := leaveEditor(t, true)
	_ = e.Key("close")
	if err := e.Key("close"); err != ebiten.Termination || !e.ExitAsked() || edits.saves != 0 {
		t.Fatalf("second close: err %v exit %v saves %d", err, e.ExitAsked(), edits.saves)
	}
}

func TestCancelAndEscapeReturnToTheEditorUnchanged(t *testing.T) {
	for _, how := range []string{"button", "escape", "x"} {
		e, edits := leaveEditor(t, true)
		doc, sel := e.doc, e.selected
		_ = e.Key("close")
		switch how {
		case "button":
			leaveButton(t, e, leaveStay)
		case "escape":
			_ = e.Key("escape")
		case "x":
			l := layoutPopup(e.leaveSpec(), e.font, e.bodyFont())
			c := l.Close.Min.Add(l.Close.Size().Div(2))
			now := time.Now()
			e.Pointer(Input{CursorX: c.X, CursorY: c.Y, PrimaryDown: true}, now)
			e.Pointer(Input{CursorX: c.X, CursorY: c.Y}, now.Add(time.Millisecond))
		}
		if asked(e) || e.prompt.open() || e.ExitAsked() || e.doc != doc || e.selected != sel || !e.Dirty() || edits.saves != 0 || e.message != "" {
			t.Fatalf("%s: asked %v exit %v dirty %v saves %d message %q", how, asked(e), e.ExitAsked(), e.Dirty(), edits.saves, e.message)
		}
	}
}

func TestSaveAsFromTheQuestionClosesOnlyAfterTheSave(t *testing.T) {
	e, edits := leaveEditor(t, true)
	_ = e.Key("close")
	leaveButton(t, e, leaveSave)
	if e.prompt.kind != promptSave || e.ExitAsked() || edits.saves != 0 {
		t.Fatalf("Save As button: prompt %v exit %v saves %d", e.prompt.kind, e.ExitAsked(), edits.saves)
	}
	e.TextInput("out.alm")
	if err := e.Key("enter"); err != ebiten.Termination || !e.ExitAsked() || edits.saves != 1 || e.Dirty() {
		t.Fatalf("after the save: err %v exit %v saves %d dirty %v", err, e.ExitAsked(), edits.saves, e.Dirty())
	}
}

func TestRefusedSaveFromTheQuestionKeepsTheEditor(t *testing.T) {
	e, edits := leaveEditor(t, true)
	edits.refuse, edits.rows = true, reportRows
	_ = e.Key("close")
	leaveButton(t, e, leaveSave)
	e.TextInput("out.alm")
	if err := e.Key("enter"); err == nil || err == ebiten.Termination || e.ExitAsked() || !e.Dirty() {
		t.Fatalf("refused save: err %v exit %v dirty %v", err, e.ExitAsked(), e.Dirty())
	}
	// The refused step is forgotten: a later plain save does not close.
	edits.refuse, edits.rows = false, nil
	_ = e.Key("saveas")
	e.TextInput("again.alm")
	if err := e.Key("enter"); err != nil || e.ExitAsked() {
		t.Fatalf("later save: err %v exit %v", err, e.ExitAsked())
	}
}

func TestCancelledSaveDialogForgetsTheClose(t *testing.T) {
	e, edits := leaveEditor(t, true)
	_ = e.Key("close")
	leaveButton(t, e, leaveSave)
	_ = e.Key("escape")
	if e.saveThen != nil || e.prompt.open() || e.ExitAsked() {
		t.Fatal("Esc in the Save As prompt kept the close")
	}
	e.TextInput("")
	_ = e.Key("saveas")
	e.TextInput("later.alm")
	if err := e.Key("enter"); err != nil || e.ExitAsked() || edits.saves != 1 {
		t.Fatalf("later save closed: err %v exit %v saves %d", err, e.ExitAsked(), edits.saves)
	}
}

func TestOpenWithUnsavedChangesAsksTheSameQuestion(t *testing.T) {
	e, edits := leaveEditor(t, true)
	if err := e.Open("scenario/11.alm"); err != nil || !asked(e) || e.doc.Source != "scenario/10.alm" {
		t.Fatalf("dirty Open: err %v asked %v source %q", err, asked(e), e.doc.Source)
	}
	if got := e.leaveSpec().Buttons[leaveGo]; got != words.For("").Text("editor.leave.open") {
		t.Fatalf("second button %q", got)
	}
	leaveButton(t, e, leaveStay)
	if asked(e) || e.doc.Source != "scenario/10.alm" || !e.Dirty() {
		t.Fatal("Cancel changed the editor")
	}
	_ = e.Open("scenario/11.alm")
	leaveButton(t, e, leaveGo)
	if asked(e) || e.doc.Source != "scenario/11.alm" || e.ExitAsked() || edits.saves != 0 {
		t.Fatalf("Open without saving: source %q exit %v saves %d", e.doc.Source, e.ExitAsked(), edits.saves)
	}
}

func TestOpenSaveAsSavesThenOpens(t *testing.T) {
	e, edits := leaveEditor(t, true)
	_ = e.Open("scenario/11.alm")
	leaveButton(t, e, leaveSave)
	e.TextInput("out.alm")
	if err := e.Key("enter"); err != nil || edits.saves != 1 || e.doc.Source != "scenario/11.alm" || e.ExitAsked() {
		t.Fatalf("err %v saves %d source %q exit %v", err, edits.saves, e.doc.Source, e.ExitAsked())
	}
}

func TestOpenWithoutChangesOpensAtOnce(t *testing.T) {
	e, _ := leaveEditor(t, false)
	if err := e.Open("scenario/11.alm"); err != nil || asked(e) || e.doc.Source != "scenario/11.alm" {
		t.Fatalf("clean Open: err %v asked %v source %q", err, asked(e), e.doc.Source)
	}
}

func TestLeaveQuestionNamesTheMapInBothLanguages(t *testing.T) {
	for _, lang := range []string{"english", "russian"} {
		e, _ := leaveEditor(t, true)
		b := words.For(lang)
		e.SetWords(b)
		_ = e.Key("close")
		s := e.leaveSpec()
		if s.Title != b.Text("editor.leave.title.close") || len(s.Notes) == 0 || s.Buttons[leaveGo] != b.Text("editor.leave.close") || s.Buttons[leaveStay] != b.Text("editor.button.cancel") {
			t.Fatalf("%s: %+v", lang, s)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		for i, r := range l.Buttons {
			if !r.In(l.Rect) || r.Dx() < e.bodyFont().Advance(s.Buttons[i]) {
				t.Fatalf("%s: button %d %v clipped in %v", lang, i, r, l.Rect)
			}
		}
		if en, ru := words.For("english").Text("editor.leave.text"), words.For("russian").Text("editor.leave.text"); en == ru {
			t.Fatal("the question text has no Russian")
		}
	}
}

var _ = image.Point{}
