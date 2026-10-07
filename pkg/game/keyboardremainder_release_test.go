package game

import (
	"testing"
	"time"

	"againrom/pkg/ui"
)

// On an installed mission map, through the App's own key route: Backspace
// empties the message line, F12 toggles the frame-rate readout, and Alt plus a
// letter changes nothing (MENU-060, MENU-061, MENU-062).
func TestReleaseMapKeyboardRemainder(t *testing.T) {
	app, live, hero := openRefusedFleeMission(t, "keyboard remainder")
	v := live.view
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	v.PostMessage("one", ui.MessageWhite, time.Minute)
	v.PostMessage("two", ui.MessageGrey, time.Minute)
	if len(v.MessageLines()) == 0 {
		t.Fatal("control: the posts left no line")
	}
	if err := app.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if n := len(v.MessageLines()); n != 0 {
		t.Errorf("Backspace left %d message lines", n)
	}
	if v.FPSShown() {
		t.Fatal("the frame-rate readout is on at load")
	}
	if err := app.HeadlessKey("f12"); err != nil {
		t.Fatal(err)
	}
	if !v.FPSShown() {
		t.Error("F12 did not turn the readout on")
	}
	if err := app.HeadlessKey("f12"); err != nil {
		t.Fatal(err)
	}
	if v.FPSShown() {
		t.Error("a second F12 did not turn the readout off")
	}
	for _, key := range []string{"alt-b", "alt-d", "alt-i", "alt-c", "alt-j", "alt-p"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, armed := v.QuickSpellState(); armed {
		t.Error("an Alt chord armed Cast")
	}
	if sel := app.HeadlessSelection(); len(sel) != 1 || sel[0] != uint32(hero) {
		t.Errorf("an Alt chord changed the selection to %v", sel)
	}
}
