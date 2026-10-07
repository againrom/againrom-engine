package ui

import (
	"image"
	"testing"
	"time"
)

// Backspace on the map empties the message line and changes nothing else
// (MENU-061).
func TestBackspaceEmptiesTheMessageLineOnTheMap(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetFont(panelFont())
	v.PostMessage("first", MessageWhite, time.Minute)
	v.PostMessage("second", MessageGrey, time.Minute)
	if len(v.MessageLines()) == 0 {
		t.Fatal("control: nothing was posted")
	}
	v.sel = selection{4}
	armed, hidden := v.spellArmed, v.hudHidden
	if err := a.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if n := len(v.MessageLines()); n != 0 {
		t.Errorf("message line holds %d lines after Backspace, want 0", n)
	}
	if _, _, _, ok := v.MessageLog(); ok {
		t.Error("an emptied message line still draws")
	}
	if !equalSel(v.sel, selection{4}) || v.spellArmed != armed || v.hudHidden != hidden {
		t.Error("Backspace changed the selection, the cast arm or the panels")
	}
	v.PostMessage("after", MessageWhite, time.Minute)
	if n := len(v.MessageLines()); n != 1 {
		t.Errorf("a post after the clear leaves %d lines, want 1", n)
	}
}

func equalSel(a, b selection) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// F12 toggles the frame-rate readout; it is off at load and draws a box with
// the `%3.1f fps` line (MENU-060).
func TestF12TogglesTheFrameRateReadout(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetFont(panelFont())
	if v.FPSShown() {
		t.Fatal("the frame-rate readout is on at load")
	}
	if _, _, ok := v.fpsPresent(60); ok {
		t.Fatal("an off readout presented a picture")
	}
	if err := a.HeadlessKey("f12"); err != nil {
		t.Fatal(err)
	}
	if !v.FPSShown() {
		t.Fatal("F12 did not turn the readout on")
	}
	pic, at, ok := v.fpsPresent(59.96)
	if !ok || pic == nil {
		t.Fatal("an on readout presented no picture")
	}
	want := fpsBoxRect(v.cam.ViewW)
	if at != want.Min || pic.Bounds().Size() != want.Size() {
		t.Errorf("box at %v size %v, want %v", at, pic.Bounds().Size(), want)
	}
	if got := FPSLine(59.96); got != "60.0 fps" {
		t.Errorf("line = %q", got)
	}
	if got := FPSLine(5); got != "5.0 fps" {
		t.Errorf("line = %q", got)
	}
	if err := a.HeadlessKey("f12"); err != nil {
		t.Fatal(err)
	}
	if v.FPSShown() {
		t.Error("a second F12 did not turn the readout off")
	}
}

// C over a caster shows the closed spell bar, selects no spell and is
// idempotent; a selection with no caster leaves the bar closed (MENU-054,
// MENU-064, MENU-065).
func TestCastKeyShowsTheSpellBar(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetEntities([]MapEntity{
		{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true},
		{ID: 5, Cell: image.Pt(4, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
	})
	v.selectedSpell = 0
	if v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
	v.sel = selection{4}
	if err := a.HeadlessKey("c"); err != nil {
		t.Fatal(err)
	}
	if v.hudShown(hudPanelBook) {
		t.Error("C with no caster showed the spell bar")
	}
	v.sel = selection{4, 5}
	if err := a.HeadlessKey("c"); err != nil {
		t.Fatal(err)
	}
	if v.hudShown(hudPanelBook) || v.spellArmed {
		t.Fatal("C with no spell selected was not a no-op")
	}
	v.selectedSpell = 1
	if err := a.HeadlessKey("c"); err != nil {
		t.Fatal(err)
	}
	if !v.hudShown(hudPanelBook) || !v.spellArmed {
		t.Fatal("C over a caster with a spell selected did not show the bar and arm Cast")
	}
	if err := a.HeadlessKey("c"); err != nil {
		t.Fatal(err)
	}
	if !v.hudShown(hudPanelBook) {
		t.Error("a second C closed the bar")
	}
}

// Alt plus a letter reaches no map letter action: the keys stay inert
// (MENU-062, AI-378). Without Alt the same fields act.
func TestAltLettersAreInertOnTheMap(t *testing.T) {
	for _, key := range []string{"b", "q", "c", "d", "i", "j", "p"} {
		a, v := mkOnMap(t)
		v.SetEntities([]MapEntity{
			{ID: 5, Cell: image.Pt(4, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
		})
		v.sel = selection{5}
		v.selectedSpell = 1
		before, selBefore := v.hudHidden, append(selection(nil), v.sel...)
		if err := a.HeadlessKey("alt-" + key); err != nil {
			t.Fatal(err)
		}
		if v.hudHidden != before || v.spellArmed || v.missionMode() != modeNone || !equalSel(v.sel, selBefore) {
			t.Errorf("Alt+%s acted: panels %v armed %v mode %d", key, v.hudHidden, v.spellArmed, v.missionMode())
		}
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if v.hudHidden == before && !v.spellArmed && v.missionMode() == modeNone && equalSel(v.sel, selBefore) {
			t.Errorf("control: plain %s did nothing, the test proves nothing", key)
		}
	}
}
