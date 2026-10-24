package game

import (
	"fmt"
	"path/filepath"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func loadedSpellReleaseKey(t *testing.T, a *ui.App, key string) {
	t.Helper()
	if err := a.HeadlessKey(key); err != nil {
		t.Fatal(err)
	}
}

func loadedSpellReleaseTap(t *testing.T, a *ui.App, x, y int, edges ...string) {
	t.Helper()
	for _, edge := range edges {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func loadedSpellReleaseBook(a *ui.App, spell uint32) bool {
	_, _, err := a.HeadlessSpellPoint(spell)
	return err == nil
}

// loadedSpellReleaseCast taps the entity, or the ground for id 0, and reports
// whether the tap queued a cast.
func loadedSpellReleaseCast(t *testing.T, f *FrontEnd, a *ui.App, id uint32) bool {
	t.Helper()
	before := len(f.live.pending)
	x, y, err := a.HeadlessGroundPoint()
	if id != 0 {
		x, y, err = a.HeadlessEntityPoint(id)
	}
	if err != nil {
		t.Fatal(err)
	}
	loadedSpellReleaseTap(t, a, x, y, "press", "release")
	for _, c := range f.live.pending[before:] {
		if c.Kind == sim.KindCast || c.Kind == sim.KindCastAt {
			return true
		}
	}
	return false
}

// loadedSpellReleaseTrace plays a click on the mage, C, a second click and a
// right click on the ground, and states what each did.
func loadedSpellReleaseTrace(t *testing.T, f *FrontEnd, a *ui.App, id, spell uint32) string {
	t.Helper()
	_, current, armed := f.live.view.QuickSpellState()
	out := fmt.Sprintf("spell %d armed %v book %v; click cast %v", current, armed, loadedSpellReleaseBook(a, spell), loadedSpellReleaseCast(t, f, a, id))
	loadedSpellReleaseKey(t, a, "c")
	_, _, armed = f.live.view.QuickSpellState()
	out += fmt.Sprintf("; C armed %v book %v", armed, loadedSpellReleaseBook(a, spell))
	out += fmt.Sprintf("; click cast %v book %v", loadedSpellReleaseCast(t, f, a, id), loadedSpellReleaseBook(a, spell))
	x, y, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	loadedSpellReleaseTap(t, a, x, y, "right-press", "right-release")
	_, current, _ = f.live.view.QuickSpellState()
	return out + fmt.Sprintf("; right click selection %d spell %d", len(f.live.view.SelectedUnits()), current)
}

// TestReleaseLoadedSpellBehavesAsBeforeSave selects spell 16 for a mission-10
// mage in three ways, saves through the game menu and loads the file twice
// through the LOAD dialog. A spell chosen in the book, with the book closed or
// open, behaves after LOAD as before SAVE. A quick-key arm with the book closed
// writes the same SAV leaves as a closed book-chosen spell and loads as one.
func TestReleaseLoadedSpellBehavesAsBeforeSave(t *testing.T) {
	const spell = 16
	loaded := map[string]string{}
	for _, arm := range []string{"book closed", "book open", "quick key"} {
		t.Run(arm, func(t *testing.T) {
			f := releaseFront(t)
			if f.Archives.Game() != base.GameROM1 {
				t.Skip("a first-game mission; the second game has its own witness below")
			}
			f.SetDeterministicFrames(true)
			hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
			hero.Skill[1] = 100
			party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
				Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1<<1 | 1<<16 | 1<<6 | 1<<19,
				Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000, HealthRegenPeriod: 100, ManaRegenPeriod: 50}}}
			store := SaveStore{Dir: t.TempDir()}
			a := f.App("loaded spell witness")
			a.Layout(1024, 768)
			f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
			if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
				t.Fatal(err)
			}
			missionAutosaveRow(t, f, store, 10)
			for i := 0; a.HeadlessNoticeOpen() && i < 16; i++ {
				loadedSpellReleaseKey(t, a, "enter")
			}
			loadedSpellReleaseKey(t, a, "0")
			id := uint32(f.live.mission.ids[0])
			inspectionCentre(f.live, 29, 50)
			if err := a.HeadlessSelectEntity(id); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if !loadedSpellReleaseBook(a, spell) {
				loadedSpellReleaseKey(t, a, "book")
			}
			x, y, err := a.HeadlessSpellPoint(spell)
			if err != nil {
				t.Fatal(err)
			}
			switch arm {
			case "quick key":
				loadedSpellReleaseTap(t, a, x, y, "hover")
				loadedSpellReleaseKey(t, a, "ctrl-f5")
				loadedSpellReleaseKey(t, a, "book")
				loadedSpellReleaseKey(t, a, "f5")
			default:
				loadedSpellReleaseTap(t, a, x, y, "press", "release")
				if arm == "book closed" {
					loadedSpellReleaseKey(t, a, "book")
				}
			}
			loadedSpellReleaseKey(t, a, "escape")
			if err := a.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessSaveEdit(store.Dir, "Loaded spell", ui.SaveSAV); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessSaveAction("save"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ui.ScreenMap {
				loadedSpellReleaseKey(t, a, "escape")
			}
			if a.Screen() != ui.ScreenMap {
				t.Fatalf("SAVE left screen %s", a.Screen())
			}
			before := loadedSpellReleaseTrace(t, f, a, id, spell)
			t.Logf("before SAVE: %s", before)
			for cycle := 0; cycle < 2; cycle++ {
				g := releaseFront(t)
				g.SetDeterministicFrames(true)
				b := g.App("loaded spell LOAD")
				b.Layout(1024, 768)
				save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
				b.SetSaveSeams(save, list, load)
				groundAppLoad(t, b, list, "Loaded spell.sav")
				for i := 0; b.HeadlessNoticeOpen() && i < 16; i++ {
					loadedSpellReleaseKey(t, b, "enter")
				}
				after := loadedSpellReleaseTrace(t, g, b, id, spell)
				t.Logf("LOAD %d: %s", cycle, after)
				switch arm {
				case "quick key":
					if want := loaded["book closed"]; want != "" && after != want {
						t.Errorf("LOAD %d of a quick-key arm: %s\nclosed book-chosen LOAD: %s", cycle, after, want)
					}
				default:
					if after != before {
						t.Errorf("LOAD %d: %s\nbefore SAVE: %s", cycle, after, before)
					}
					loaded[arm] = after
				}
			}
		})
	}
}

// TestReleaseOriginalSavedSpellLoadsAsTheBookChosenArm loads three SAVs the
// original wrote with SpellBook/Pressed set through the LOAD dialog: the
// current spell is the saved one, and it casts on a map click only while the
// book is shown.
func TestReleaseOriginalSavedSpellLoadsAsTheBookChosenArm(t *testing.T) {
	for _, relative := range []string{
		"2026-10-06/game0007-original-m70-boltcoming.sav",
		"2026-10-07/original-game0011-after-beforekargallas.sav",
		"2026-10-14/exp-teleport-series/game0023.sav",
	} {
		t.Run(filepath.Base(relative), func(t *testing.T) {
			path, raw := groundCorpusFile(t, relative, "")
			f := releaseFront(t)
			if f.Archives.Game() != base.GameROM1 {
				t.Skip("the saves are first-game missions")
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			state, err := readOriginalApplicationState(&doc)
			if err != nil {
				t.Fatal(err)
			}
			spell, err := originalPressedSpellID(state.Pressed)
			if err != nil || spell == 0 {
				t.Fatalf("SpellBook/Pressed %d: spell %d %v", state.Pressed, spell, err)
			}
			f.SetDeterministicFrames(true)
			a := f.App("original saved spell")
			a.Layout(1024, 768)
			save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
			a.SetSaveSeams(save, list, load)
			groundAppLoad(t, a, list, filepath.Base(path))
			for i := 0; a.HeadlessNoticeOpen() && i < 16; i++ {
				loadedSpellReleaseKey(t, a, "enter")
			}
			_, current, armed := f.live.view.QuickSpellState()
			open := state.BookOpen != 0
			t.Logf("Pressed %d spell %d IsOpen %d selection %v: current %d armed %v", state.Pressed, spell, state.BookOpen, f.live.view.SelectedUnits(), current, armed)
			if current != spell {
				t.Fatalf("current spell %d, want %d", current, spell)
			}
			if !open && armed {
				t.Fatal("the saved spell is live with the book closed")
			}
			if open && !armed {
				t.Fatal("the saved spell is not live with the book open")
			}
			if open {
				loadedSpellReleaseKey(t, a, "book")
				if _, _, armed = f.live.view.QuickSpellState(); armed {
					t.Fatal("closing the book left the saved spell live")
				}
			}
			if loadedSpellReleaseCast(t, f, a, 0) {
				t.Fatal("a ground click with the book closed cast the saved spell")
			}
			if _, current, _ = f.live.view.QuickSpellState(); current != spell {
				t.Fatalf("the ground click changed the current spell to %d", current)
			}
		})
	}
}

// TestReleaseSecondGameLoadedSpellBehavesAsBeforeSave is the same witness for
// the second game's generated mage on mission 10 with Shield chosen in the
// book, saved by a named SAVE and loaded twice.
func TestReleaseSecondGameLoadedSpellBehavesAsBeforeSave(t *testing.T) {
	const spell = 27
	for _, arm := range []string{"book closed", "book open"} {
		t.Run(arm, func(t *testing.T) {
			f, app := secondGeneratedMage(t, false)
			for _, target := range []string{"TAVERN", "TALK 517"} {
				if err := app.HeadlessActivate(target); err != nil {
					t.Fatal(err)
				}
			}
			for app.HeadlessActivate("notice") == nil {
			}
			for _, target := range []string{"EXIT", "GATES", "mission 10"} {
				if err := app.HeadlessActivate(target); err != nil {
					t.Fatal(err)
				}
			}
			for range 20 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			dismissSecondMissionDialogue(t, app, f.live)
			app.Layout(1024, 768)
			loadedSpellReleaseKey(t, app, "0")
			out := secondMissionSaveDirectory(t)
			id := f.live.mission.ids[0]
			spellbookShow(t, app, f.live, id)
			x, y, err := app.HeadlessSpellPoint(spell)
			if err != nil {
				t.Fatal(err)
			}
			spellbookClick(t, app, x, y)
			if arm == "book closed" {
				loadedSpellReleaseKey(t, app, "book")
			}
			e, _ := f.live.entity(id)
			inspectionCentre(f.live, int(e.X), int(e.Y))
			f.live.push()
			name := "Loaded spell " + map[string]string{"book closed": "closed", "book open": "open"}[arm]
			secondMissionNamedSave(t, f, app, out, name)
			before := loadedSpellReleaseTrace(t, f, app, uint32(id), spell)
			t.Logf("before SAVE: %s", before)
			for cycle := 0; cycle < 2; cycle++ {
				cold, a := secondMissionCold(t, out, secondSaveFile(t, out, name))
				e, _ := cold.live.entity(id)
				inspectionCentre(cold.live, int(e.X), int(e.Y))
				cold.live.push()
				after := loadedSpellReleaseTrace(t, cold, a, uint32(id), spell)
				t.Logf("LOAD %d: %s", cycle, after)
				if after != before {
					t.Errorf("LOAD %d: %s\nbefore SAVE: %s", cycle, after, before)
				}
			}
		})
	}
}
