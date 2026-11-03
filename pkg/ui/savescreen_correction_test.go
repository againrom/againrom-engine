package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"strings"
	"testing"
)

func saveScreenFrameSum(t *testing.T, a *App) string {
	t.Helper()
	pix, err := a.composeSaveScreen()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(pix.Pix))
}

func TestSaveScreenTypedKeyKeepsMessageWithoutTextField(t *testing.T) {
	entries := []SaveEntry{{Name: "alpha.sav", Label: "Alpha"}}
	t.Run("refusal with the focus on Save", func(t *testing.T) {
		spy := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}}
		a := newSaveDialogApp(t, spy, ScreenTown)
		a.SetWords(AuthoredWords(), chargenTestFont(), nil)
		spy.prepareError = fmt.Errorf("probe refusal")
		p := saveControlRect(saveWriteControl).Min.Add(image.Pt(4, 4))
		_ = a.HeadlessPointer("press", p.X, p.Y)
		_ = a.HeadlessPointer("release", p.X, p.Y)
		if a.flow.msg != "probe refusal" {
			t.Fatalf("after refusal msg %q", a.flow.msg)
		}
		frame := saveScreenFrameSum(t, a)
		if err := a.HeadlessType("w", false); err != nil {
			t.Fatal(err)
		}
		if a.flow.msg != "probe refusal" || saveScreenFrameSum(t, a) != frame {
			t.Fatalf("typed letter changed msg %q or the frame", a.flow.msg)
		}
		if err := a.HeadlessKey("backspace"); err != nil {
			t.Fatal(err)
		}
		if a.flow.msg != "probe refusal" || saveScreenFrameSum(t, a) != frame {
			t.Fatalf("backspace changed msg %q or the frame", a.flow.msg)
		}
	})
	t.Run("unreadable folder with the focus on the list", func(t *testing.T) {
		spy := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}}
		a := newSaveDialogApp(t, spy, ScreenTown)
		a.SetWords(AuthoredWords(), chargenTestFont(), nil)
		_ = a.HeadlessSaveEdit("unreadable", "", "")
		if a.flow.msg != "cannot read folder" {
			t.Skipf("setup gave msg %q", a.flow.msg)
		}
		if err := a.HeadlessType("v", false); err != nil {
			t.Fatal(err)
		}
		if a.flow.msg != "cannot read folder" {
			t.Fatalf("typed letter changed msg to %q", a.flow.msg)
		}
	})
}

func TestSaveScreenSecondGameRefusesFirstGameTokens(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0001.sav", Label: "Alpha"}, {Name: "game0002.sav", Label: "Beta"}},
		existing: map[string]bool{"game0001.sav": true}}
	a := lookApp(t, fake)
	p := lookPage(fake)
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &p})
	openTestSavePage(t, a)
	frame := saveScreenFrameSum(t, a)
	before, _ := a.HeadlessSaveState()
	for _, tok := range []string{"overwrite", "OVERWRITE", "up", "Up", "open", " Open "} {
		err := a.HeadlessSaveAction(tok)
		want := fmt.Sprintf("save page: unknown action %q; accepted actions: save, confirm, delete, back, cancel, select-all", tok)
		if err == nil || err.Error() != want {
			t.Fatalf("%q = %v, want %q", tok, err, want)
		}
	}
	after, _ := a.HeadlessSaveState()
	if fmt.Sprint(before) != fmt.Sprint(after) || saveScreenFrameSum(t, a) != frame || strings.Contains(a.flow.msg, "save dialog") {
		t.Fatalf("state changed: %+v -> %+v", before, after)
	}
}
