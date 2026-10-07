package game

import (
	"testing"
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// On an installed mission the open Drop Gold editor deletes before the caret on
// Backspace, takes nothing from Alt+Backspace and never clears the message line
// (MENU-082, MENU-088).
func TestReleasePurseEditorBackspaceAndAltBackspace(t *testing.T) {
	app, live, hero := openRefusedFleeMission(t, "purse editor keys")
	w := live.world
	w.SetPurse(sim.SelfSlot, 2500)
	live.push()
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	if !live.view.SaveApplication().InventoryOpen {
		if err := app.HeadlessKey("i"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	cell := -1
	for i, p := range live.invSubject.PackPurse {
		if p {
			cell = i
		}
	}
	if cell < 0 {
		t.Fatalf("setup: the pack shows no purse: %+v", live.invSubject.PackPurse)
	}
	x, y, err := app.HeadlessPackCellPoint(cell)
	if err != nil {
		t.Fatal(err)
	}
	pursePointer(t, app, x, y, "press", "release", "press", "release")
	if state, open := app.HeadlessGold(); !open || !state.Open || state.Text != "0" {
		t.Fatalf("setup: editor %+v, %v", state, open)
	}
	live.view.PostMessage("keep me", ui.MessageWhite, time.Hour)
	if err := app.HeadlessType("12", false); err != nil {
		t.Fatal(err)
	}
	want := func(text string, caret int, label string) {
		t.Helper()
		got, open := app.HeadlessGold()
		if !open || got.Text != text || got.Caret != caret {
			t.Fatalf("%s: editor %+v, want text %q caret %d", label, got, text, caret)
		}
		if len(live.view.MessageLines()) == 0 {
			t.Fatalf("%s cleared the message line", label)
		}
	}
	want("120", 2, "typing")
	if err := app.HeadlessKey("alt-backspace"); err != nil {
		t.Fatal(err)
	}
	want("120", 2, "Alt+Backspace")
	if err := app.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	want("10", 1, "Backspace")
	if err := app.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	want("0", 0, "second Backspace")
	if err := app.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	want("0", 0, "Backspace at caret zero")
	if w.Purse(sim.SelfSlot) != 2500 || len(live.pending) != 0 {
		t.Fatalf("editing moved the purse to %d with %d requests", w.Purse(sim.SelfSlot), len(live.pending))
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if _, open := app.HeadlessGold(); open {
		t.Fatal("Escape left the editor open")
	}
	if err := app.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if n := len(live.view.MessageLines()); n != 0 {
		t.Fatalf("control: Backspace with the editor closed left %d message lines", n)
	}
}
