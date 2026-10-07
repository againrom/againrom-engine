package ui

import (
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// TestHeadlessGameMenuActionsDoNotDependOnDisplayedWords witnesses the RU
// failure found by the cross-install scenario gate: the scenario command is
// English vocabulary, but SAVE, LOAD and ABORT GAME are localized production
// rows. Before HeadlessGameMenuAction resolved the row's action, the runner
// searched for literal "SAVE" / "LOAD" / "ABORT GAME" and could not use a
// non-English menu (1020: the "abort" action, added for the same reason as
// "save" and "load").
func TestHeadlessGameMenuActionsDoNotDependOnDisplayedWords(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	w := AuthoredWords()
	w.MenuSave = "~SPEICHERN"
	w.MenuLoad = "~LADEN"
	w.MenuAbort = "~ABBRECHEN"
	w.SaveAcknowledgement = "installed save acknowledgement"
	a.SetWords(w, nil, nil)

	saved := 0
	a.SetSaveSeams(
		func(bool) (string, error) {
			saved++
			return "localized.ags", nil
		},
		func() []SaveEntry { return []SaveEntry{{Name: "localized.ags", Label: "Spielstand"}} },
		func(string) (MapOpener, bool, error) { return nil, true, nil },
	)
	a.flow.openGameMenu(ScreenTown)

	if err := a.HeadlessGameMenuAction("save"); err != nil {
		t.Fatalf("HeadlessGameMenuAction(save) = %v, want nil", err)
	}
	if saved != 1 || a.HeadlessMessage() != "installed save acknowledgement" {
		t.Fatalf("save result = count %d, message %q", saved, a.HeadlessMessage())
	}
	if err := a.HeadlessGameMenuAction("load"); err != nil {
		t.Fatalf("HeadlessGameMenuAction(load) = %v, want nil", err)
	}
	if a.Screen() != ScreenLoad {
		t.Fatalf("Screen() after load = %s, want %s", a.Screen(), ScreenLoad)
	}
	a.flow.openGameMenu(ScreenTown)
	if err := a.HeadlessGameMenuAction("abort"); err != nil {
		t.Fatalf("HeadlessGameMenuAction(abort) = %v, want nil", err)
	}
	if err := a.HeadlessGameMenuAction("confirm-abort"); err != nil {
		t.Fatalf("HeadlessGameMenuAction(confirm-abort) = %v, want nil", err)
	}
	if a.Screen() != ScreenMenu {
		t.Fatalf("Screen() after abort = %s, want %s", a.Screen(), ScreenMenu)
	}
}

// TestATypedCyrillicRuneReachesTheMenuThroughTheStep drives the accelerator
// through a.step, which is where the frame's typed characters actually arrive.
//
// THE OTHER ACCELERATOR TESTS CALL chooseGameMenuAccelerator DIRECTLY, so the
// dispatch above it is not witnessed by them: reverting stepGameMenu's loop to
// the byte-by-byte walk 1014 replaced leaves the whole suite green, which the
// story's own adversarial review demonstrated. A Cyrillic character is more
// than one byte in ebiten's UTF-8, so the byte walk hands the encoder two
// fragments of one rune and no row can match. This is the test that reddens.
func TestATypedCyrillicRuneReachesTheMenuThroughTheStep(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	w := AuthoredWords()
	// "~Сохранить игру" (MENU-KEY-013 evidence), written as bytes and never as
	// literal non-ASCII text. The mark is on 0x91, uppercase С.
	w.MenuSave = string([]byte{0x7e, 0x91, 0xae, 0xe5, 0xe0, 0xa0, 0xad,
		0xa8, 0xe2, 0xec, 0x20, 0xa8, 0xa3, 0xe0, 0xe3})
	a.SetWords(w, &text.Font{Selector: text.SelectorConverting},
		func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) })

	saved := 0
	a.SetSaveSeams(
		func(bool) (string, error) { saved++; return "s.ags", nil },
		func() []SaveEntry { return []SaveEntry{{Name: "s.ags", Label: "a save"}} },
		func(string) (MapOpener, bool, error) { return nil, true, nil },
	)
	a.flow.openGameMenu(ScreenTown)

	if err := a.HeadlessType("с", false); err != nil {
		t.Fatalf("HeadlessType = %v, want nil", err)
	}
	if saved != 1 {
		t.Fatalf("typed с saved %d time(s), want 1", saved)
	}
	if got := a.HeadlessMessage(); got != "Your character is saved" {
		t.Fatalf("message after typing с = %q, want %q", got, "Your character is saved")
	}
}
