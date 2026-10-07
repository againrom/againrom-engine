package game

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// helpWitnessParty is one hero standing at a fixed cell: a mage with known
// spells and a mana pool, or a fighter with neither.
func helpWitnessParty(mage bool) []mapload.PartyMember {
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	m := mapload.PartyMember{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: mage, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: mage}, Hero: hero,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, HealthRegenPeriod: 100}}
	if mage {
		m.KnownSpells = 1<<1 | 1<<16
		m.Saved.Mana, m.Saved.MaxMana, m.Saved.ManaRegenPeriod = 1000, 1000, 50
	} else {
		m.Class = 0
	}
	return m1(m)
}

func m1(m mapload.PartyMember) []mapload.PartyMember { return []mapload.PartyMember{m} }

// helpWitnessMap opens mission 10 under the party and closes the opening
// notices, leaving a running map session.
func helpWitnessMap(t *testing.T, f *FrontEnd, mage bool) *ui.App {
	t.Helper()
	f.SetDeterministicFrames(true)
	f.SetTipsOff(true)
	a := f.App("help witness")
	a.SetCutscenes(nil)
	a.Layout(640, 480)
	if err := a.OpenMission(f.MissionOpenerWith(10, helpWitnessParty(mage))); err != nil {
		t.Fatal(err)
	}
	helpCloseNotices(t, f, a)
	return a
}

// helpCloseNotices dismisses the mission's own notices so the map is the
// surface F1 meets.
func helpCloseNotices(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	for i := 0; i < 16 && f.live.view.NoticeOpen(); i++ {
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.view.NoticeOpen() {
		t.Fatal("setup: a notice stays open over the map")
	}
}

// helpFrame composes the open help panel, with the layout, font and text the
// viewer draws with, over a flat 640x480 ground at the panel's own origin. The
// map screen draws straight onto the display, so no frame of it can be read
// back.
func helpFrame(t *testing.T, f *FrontEnd) *image.RGBA {
	t.Helper()
	layout, body, ok := f.live.view.HelpPanel()
	if !ok {
		t.Fatal("no help panel is open to compose")
	}
	room := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(room, room.Bounds(), &image.Uniform{C: color.RGBA{R: 120, G: 110, B: 100, A: 255}}, image.Point{}, draw.Src)
	ui.ComposeDialogueNotice(room, layout, f.Font.Value(), body, nil, layout.Box.Min)
	return room
}

func helpKeys(t *testing.T, a *ui.App, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := a.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
}

func helpSteps(t *testing.T, a *ui.App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}

func helpShot(t *testing.T, name string, pix *image.RGBA) {
	t.Helper()
	writeModShot(t, name, pix)
}

// helpDiffers reports whether two frames differ inside r.
func helpDiffers(a, b *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return true
			}
		}
	}
	return false
}

// TestReleaseHelpPanelOverTheMap is the F1 witness on both installs: the panel
// opens over the running map with the install's text, the world is stopped
// while it stands, it scrolls, OK and Esc close it, and F1 does nothing in the
// town or over another popup.
func TestReleaseHelpPanelOverTheMap(t *testing.T) {
	locale := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	f := releaseFront(t)
	a := helpWitnessMap(t, f, true)
	v := f.live.view

	// Control: with no panel the world runs.
	tick := f.live.world.Tick()
	helpSteps(t, a, 8)
	if f.live.world.Tick() == tick {
		t.Fatal("control: the world did not advance with no panel open")
	}
	helpCloseNotices(t, f, a)

	helpKeys(t, a, "f1")
	if !v.HelpOpen() {
		t.Fatal("F1 over the map opened no help panel")
	}
	text := v.Words().HelpText
	wantBytes, wantLines := map[string]int{"en": 965, "ru": 1313}[locale], map[string]int{"en": 35, "ru": 48}[locale]
	if wantBytes == 0 {
		t.Fatalf("unexpected install %q", locale)
	}
	if len(text) != wantBytes {
		t.Errorf("help text is %d bytes, TEXT-087 gives %d", len(text), wantBytes)
	}
	lines, visible := v.HelpLines()
	if lines != wantLines || visible != 12 {
		t.Errorf("help body wraps to %d lines with %d visible, TEXT-088 gives %d and 12", lines, visible, wantLines)
	}
	first, last := v.HelpScroll()
	if first != 0 || last != lines-visible {
		t.Errorf("scroll %d of %d, want 0 of %d", first, last, lines-visible)
	}
	open := helpFrame(t, f)
	panel := image.Rect(76, 60, 564, 420)
	ground := image.NewRGBA(open.Bounds())
	draw.Draw(ground, ground.Bounds(), &image.Uniform{C: color.RGBA{R: 120, G: 110, B: 100, A: 255}}, image.Point{}, draw.Src)
	if !helpDiffers(ground, open, panel) {
		t.Fatal("the composed panel equals the bare ground")
	}
	for _, edge := range []image.Rectangle{image.Rect(0, 0, 640, 60), image.Rect(0, 420, 640, 480), image.Rect(0, 60, 76, 420), image.Rect(564, 60, 640, 420)} {
		if helpDiffers(ground, open, edge) {
			t.Fatalf("the panel paints outside its 488x360 box at (76,60): %v", edge)
		}
	}
	helpShot(t, "help-open", open)

	// The world is stopped while the panel stands.
	tick, hash := f.live.world.Tick(), f.live.world.Hash()
	helpSteps(t, a, 12)
	if f.live.world.Tick() != tick || f.live.world.Hash() != hash {
		t.Fatal("the world advanced while the help panel was open")
	}

	// F1 over help changes nothing.
	helpKeys(t, a, "f1")
	if first, _ := v.HelpScroll(); !v.HelpOpen() || first != 0 {
		t.Error("F1 over help reopened or scrolled it")
	}

	// Scrolling by page changes the body and clamps at the end.
	helpKeys(t, a, "pagedown")
	if first, _ := v.HelpScroll(); first != visible-1 {
		t.Errorf("Page Down scrolled to %d, want %d", first, visible-1)
	}
	scrolled := helpFrame(t, f)
	if !helpDiffers(open, scrolled, panel) {
		t.Error("scrolling left the panel unchanged")
	}
	helpShot(t, "help-scrolled", scrolled)
	for i := 0; i < 8; i++ {
		helpKeys(t, a, "pagedown")
	}
	if first, last := v.HelpScroll(); first != last {
		t.Errorf("Page Down stopped at %d, want the end %d", first, last)
	}
	helpShot(t, "help-end", helpFrame(t, f))

	// Esc closes and the world runs again.
	helpKeys(t, a, "escape")
	if v.HelpOpen() || v.NoticeOpen() {
		t.Fatal("Esc did not close the help panel")
	}
	tick = f.live.world.Tick()
	helpSteps(t, a, 8)
	if f.live.world.Tick() == tick {
		t.Error("the world stayed stopped after the panel closed")
	}

	// OK closes it too, through the panel's button.
	helpKeys(t, a, "f1")
	if !v.HelpOpen() {
		t.Fatal("F1 did not reopen help")
	}
	if first, _ := v.HelpScroll(); first != 0 {
		t.Errorf("reopened help starts at %d, want the top", first)
	}
	if err := a.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if v.HelpOpen() {
		t.Error("the OK button did not close the help panel")
	}

	// F1 over the in-game menu opens no panel.
	helpKeys(t, a, "escape")
	if a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Esc opened screen %v, want the in-game menu", a.Screen())
	}
	helpKeys(t, a, "f1")
	if v.HelpOpen() || a.Screen() != ui.ScreenGameMenu {
		t.Error("F1 over the in-game menu opened help")
	}
}

// TestReleaseHelpKeyDoesNothingInTown presses F1 in the town reached through
// chargen and mission 10 and a cold LOAD of the town save.
func TestReleaseHelpKeyDoesNothingInTown(t *testing.T) {
	_, a, _ := modJoinCold(t, currentTownSave(t, modJoinTown(t, false)), false)
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("setup: screen %v, want the town", a.Screen())
	}
	before := modFrame(t, a)
	helpKeys(t, a, "f1")
	if a.Screen() != ui.ScreenTown || a.HeadlessNoticeOpen() {
		t.Errorf("F1 in town changed the screen to %v or opened a notice", a.Screen())
	}
	after := modFrame(t, a)
	if helpDiffers(before, after, before.Bounds()) {
		t.Error("F1 in town changed the frame")
	}
	helpShot(t, "help-town-f1", after)
}

// TestReleaseCastKeyOnTheMap is the C witness on both installs.
func TestReleaseCastKeyOnTheMap(t *testing.T) {
	castState := func(f *FrontEnd) (current uint32, armed bool) {
		_, current, armed = f.live.view.QuickSpellState()
		return current, armed
	}

	t.Run("empty selection is not consumed", func(t *testing.T) {
		f := releaseFront(t)
		a := helpWitnessMap(t, f, true)
		if sel := a.HeadlessSelection(); len(sel) != 0 {
			t.Fatalf("setup: selection %v", sel)
		}
		helpKeys(t, a, "0")
		hash := f.live.world.Hash()
		helpKeys(t, a, "c")
		if _, armed := castState(f); armed || f.live.world.Hash() != hash || len(f.live.pending) != 0 {
			t.Error("C with no selection changed state")
		}
	})

	t.Run("a fighter selected is consumed silently", func(t *testing.T) {
		f := releaseFront(t)
		a := helpWitnessMap(t, f, false)
		if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
			t.Fatal(err)
		}
		helpKeys(t, a, "0")
		hash, msg := f.live.world.Hash(), a.HeadlessMessage()
		helpKeys(t, a, "c")
		if current, armed := castState(f); armed || current != 0 || f.live.world.Hash() != hash ||
			a.HeadlessMessage() != msg || f.live.view.NoticeOpen() || len(f.live.pending) != 0 {
			t.Error("C over a selection with no caster changed state or posted a message")
		}
	})

	t.Run("a mage with no spell selected: C is a no-op", func(t *testing.T) {
		f := releaseFront(t)
		a := helpWitnessMap(t, f, true)
		if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
			t.Fatal(err)
		}
		if _, armed := castState(f); armed {
			t.Fatal("control: Cast armed before C")
		}
		helpKeys(t, a, "c")
		current, armed := castState(f)
		if armed || current != 0 {
			t.Fatalf("C over a mage with no spell: armed=%v current spell=%d, want no-op", armed, current)
		}
	})
}
