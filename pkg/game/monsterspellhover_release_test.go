package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// monsterHoverScan hovers the drawn card on the frame's grid and returns the
// first pixel whose hover target begins with prefix.
func monsterHoverScan(t *testing.T, app *ui.App, step int, prefix string) (image.Point, bool) {
	t.Helper()
	card, at, err := app.HeadlessDrawnMissionCard()
	if err != nil {
		t.Fatal(err)
	}
	bounds := card.Bounds().Add(at)
	for y := (bounds.Min.Y + step - 1) / step * step; y < bounds.Max.Y; y += step {
		for x := (bounds.Min.X + step - 1) / step * step; x < bounds.Max.X; x += step {
			if err := app.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			if state, _ := app.HeadlessTooltip(); strings.HasPrefix(state.Target, prefix) {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}

// TestReleaseTooltipMonsterSpellListOnCreatureCard hovers the spellcaster row
// of the ogre turtle's information card through the production pointer and
// timer, and reads the installed heading and the turtle's own class spell
// names. A plain creature's card offers no such target, and moving off the row
// hides the popup.
func TestReleaseTooltipMonsterSpellListOnCreatureCard(t *testing.T) {
	f, app, turtle := magicWitnessOpen(t)
	app.SetTooltipDelayPreference(300, nil)
	app.SetTooltipFont(f.Font.Value())
	plain := magicWitnessPlain(t, f)

	if !magicWitnessCaption(t, f, app, turtle.ID) {
		t.Fatal("the turtle's card does not state the spellcaster caption")
	}
	const w, h = 1024, 768
	at, ok := monsterHoverScan(t, app, 3, "stat-spells/")
	if !ok {
		t.Fatal("no pixel of the turtle's card hovers the spellcaster row")
	}
	if err := app.HeadlessPointer("hover", at.X, at.Y); err != nil {
		t.Fatal(err)
	}
	state, _ := app.HeadlessTooltip()
	if state.Visible {
		t.Fatal("hint visible before the delay elapsed")
	}
	for tick := 0; tick < 40 && !state.Visible; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		state, _ = app.HeadlessTooltip()
	}
	if !state.Visible || !state.Bounds.In(image.Rect(0, 0, w, h)) {
		t.Fatalf("hint not shown inside the frame after the delay: %+v", state)
	}
	subject, _ := f.live.view.InspectionPanel()
	lines, ok := ui.MonsterSpellHint(f.Words, subject.KnownSpells)
	if !ok || !strings.HasPrefix(lines[0], f.Words.Hover[192]) {
		t.Fatalf("hint lines %q lack the installed heading", lines)
	}
	for _, slot := range turtle.CreatureSpells {
		if slot.ID == 0 {
			continue
		}
		if name := f.Words.ItemSpellNames[slot.ID]; name == "" || !strings.Contains(lines[0], name) {
			t.Fatalf("hint %q lacks installed name %q of slot spell %d", lines[0], name, slot.ID)
		}
	}
	if err := app.HeadlessPointer("hover", 2, 2); err != nil {
		t.Fatal(err)
	}
	if state, _ = app.HeadlessTooltip(); state.Visible {
		t.Fatalf("hint survived moving off the row: %+v", state)
	}

	if magicWitnessCaption(t, f, app, plain) {
		t.Fatal("a plain creature's card states the spellcaster caption")
	}
	if _, ok := monsterHoverScan(t, app, 3, "stat-spells/"); ok {
		t.Fatal("a plain creature's card offers the monster spell list")
	}
}
