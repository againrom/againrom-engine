package game

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The window every screen of the witness is drawn in. 1920x1080 places the
// native 640x480 frame at scale 2.25, where a glyph the overlay leaves to the
// raster shows its stairs.
const smoothingW, smoothingH = 1920, 1080

// smoothingRun is one screen reached through App input, and what a drawn frame
// of it may leave unsmoothed. The witness draws the frame and asks of every
// glyph it captured whether the overlay keeps it.
type smoothingRun struct {
	name string
	// open reaches the screen by App input.
	open func(t *testing.T) (*FrontEnd, *ui.App)
	// ready checks the precondition the screen is named for after steps ticks.
	// A mission run checks it once its frame is drawn.
	ready func(t *testing.T, f *FrontEnd, a *ui.App)
	steps int
	// onMap marks a mission screen. Its text is composed while the game runs
	// and not only while Draw runs, so the audit covers the whole run, and it
	// has no CPU composite to check a hidden glyph against.
	onMap bool
	// glyphless is why the screen draws no font glyph, when that is so.
	glyphless string
	// hidden is why a captured glyph does not show on this screen, when one
	// does not. The witness checks that none of its cells shows in the frame.
	hidden string
	// extra makes the checks only this screen needs, on the settled glyphs.
	extra func(t *testing.T, f *FrontEnd, a *ui.App, fates []ui.TextFate)
}

func smoothingTicks(t *testing.T, a *ui.App, n int) {
	t.Helper()
	for range n {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}

// smoothingFront answers the main menu of a fresh front end at the witness
// window.
func smoothingFront(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("text smoothing")
	a.Layout(smoothingW, smoothingH)
	t.Cleanup(a.StopAudio)
	return f, a
}

func smoothingPress(t *testing.T, a *ui.App, native image.Point) {
	t.Helper()
	w := preCreateWindowPoint(smoothingW, smoothingH, native)
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, w.X, w.Y); err != nil {
			t.Fatal(err)
		}
	}
}

func smoothingHover(t *testing.T, a *ui.App, native image.Point, ticks int) {
	t.Helper()
	w := preCreateWindowPoint(smoothingW, smoothingH, native)
	if err := a.HeadlessPointer("hover", w.X, w.Y); err != nil {
		t.Fatal(err)
	}
	smoothingTicks(t, a, ticks)
}

// smoothingPreCreate opens the character generator's first page from the main
// menu, with its tip panel open or closed through the panel's own Close.
func smoothingPreCreate(t *testing.T, closeTip bool) (*FrontEnd, *ui.App, *ui.Chargen) {
	t.Helper()
	f, a := smoothingFront(t)
	c := preCreateNameOpen(t, f, a)
	if closeTip {
		tip := c.TipPanel()
		if tip.Rect.Empty() {
			t.Fatal("the page opened without its tip panel")
		}
		r := ui.TipPanelCloseRect(tip.Rect)
		smoothingPress(t, a, r.Min.Add(r.Size().Div(2)))
		if !c.TipPanel().Rect.Empty() {
			t.Fatal("the tip panel's Close left it open")
		}
	}
	return f, a, c
}

// smoothingTown loads a saved city through the LOAD window and answers its
// application with the town square showing.
func smoothingTown(t *testing.T) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { return 0 }
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err = store.WriteOriginal("", payload); err != nil {
		t.Fatal(err)
	}
	a := f.App("text smoothing town")
	a.Layout(640, 480)
	t.Cleanup(a.StopAudio)
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	if err = a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err = a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("the saved city opened the %s screen, want the town", a.Screen())
	}
	a.Layout(smoothingW, smoothingH)
	return f, a, f.TownScreen().(*townScreen)
}

// smoothingCompleted wins the final mission and opens an ending page.
func smoothingCompleted(t *testing.T, page string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 150)
	a := f.App("text smoothing ending")
	a.Layout(640, 480)
	t.Cleanup(a.StopAudio)
	if err := a.OpenMission(f.MissionOpenerWith(150, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	campaignWin1176(t, f, a, 150)
	campaignReturn(t, f, a)
	if a.Screen() != ui.ScreenCredits {
		t.Fatalf("the completed campaign opened the %s screen, want its credits", a.Screen())
	}
	if page == "hall" {
		if err := a.HeadlessKey("enter"); err != nil || a.Screen() != ui.ScreenEnding {
			t.Fatal("a key did not end the roll into the hall", err, a.Screen())
		}
	}
	a.Layout(smoothingW, smoothingH)
	return f, a
}

// smoothingRoom enters a town building through its door. A room that opens on
// a conversation is left on it when talk is set and read through otherwise.
func smoothingRoom(t *testing.T, a *ui.App, s *townScreen, door string, talk bool) {
	t.Helper()
	if err := a.HeadlessActivate(door); err != nil {
		t.Fatal(err)
	}
	smoothingTicks(t, a, 1)
	if !talk {
		for n := 0; s.room == roomTalk && n < 64; n++ {
			if err := a.HeadlessActivate("dialogue"); err != nil {
				t.Fatal(err)
			}
		}
		if s.room == roomTalk {
			t.Fatalf("the %s conversation did not end in 64 pages", door)
		}
	}
	s.CloseTip()
}

// smoothingMission opens the first mission with its party.
func smoothingMission(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f, a := smoothingFront(t)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	return f, a
}

func smoothingCaster(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f, a := smoothingFront(t)
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
		Hero:    data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}, KnownSpells: 1 << 23,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		t.Fatal(err)
	}
	smoothingTicks(t, a, 1)
	if _, _, err := a.HeadlessSpellPoint(23); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	x, y, err := a.HeadlessSpellPoint(23)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessKey("ctrl-f5"); err != nil {
		t.Fatal(err)
	}
	return f, a
}

// smoothingPickupMessage leaves the mission's message line showing the line of
// a pickup: the hero drops the single item of a stack on the ground and takes
// it up again, in the window that scan of the ground needs.
func smoothingPickupMessage(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("text smoothing message")
	a.Layout(messageWitnessW, messageWitnessH)
	t.Cleanup(a.StopAudio)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	hero := live.mission.ids[0]
	smoothingTicks(t, a, 90)
	for i := 0; i < 16 && a.HeadlessNoticeOpen(); i++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	smoothingTicks(t, a, 1)
	stacks, _ := live.world.CarriedStacks(hero)
	cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool {
		return s.Count == 1 && len(messageStacks(live, hero, s.Code)) == 1
	})
	if cell < 0 {
		t.Fatal("the hero carries no stack of one item to drop and take up")
	}
	e, _ := live.entity(hero)
	sx, sy := messageDropOne(t, a, live, cell, stacks[cell].Code, e.X-2, e.Y)
	messagePickUp(t, a, live, hero, sx, sy)
	return f, a
}

// smoothingViews is every screen the witness names, each reached by App input.
func smoothingViews() []smoothingRun {
	front := func(target string, ticks int) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a := smoothingFront(t)
			if target != "" {
				if err := a.HeadlessActivate(target); err != nil {
					t.Fatal(err)
				}
			}
			smoothingTicks(t, a, ticks)
			return f, a
		}
	}
	preCreate := func(closeTip bool, hover bool) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a, _ := smoothingPreCreate(t, closeTip)
			if hover {
				smoothingHover(t, a, image.Pt(300, 328), 30)
			}
			return f, a
		}
	}
	detailed := func(t *testing.T) (*FrontEnd, *ui.App) {
		f, a, c := smoothingPreCreate(t, true)
		for y := 0; y < 480; y++ {
			for x := 0; x < 640; x++ {
				if got, ok := ui.PreCreateControlAt(c, image.Pt(x, y)); ok && got == "forward" {
					smoothingPress(t, a, image.Pt(x, y))
					if st, _ := a.HeadlessChargenState(); st.Stage != ui.ChargenStageDetailed {
						t.Fatalf("Forward left the %s page", st.Stage)
					}
					return f, a
				}
			}
		}
		t.Fatal("the pre-create page has no Forward control")
		return nil, nil
	}
	town := func(enter func(*testing.T, *ui.App, *townScreen)) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a, s := smoothingTown(t)
			enter(t, a, s)
			return f, a
		}
	}
	room := func(door string, talk bool) func(*testing.T) (*FrontEnd, *ui.App) {
		return town(func(t *testing.T, a *ui.App, s *townScreen) { smoothingRoom(t, a, s, door, talk) })
	}
	menu := func(action string) func(*testing.T) (*FrontEnd, *ui.App) {
		return town(func(t *testing.T, a *ui.App, s *townScreen) {
			s.CloseTip()
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if action != "" {
				if err := a.HeadlessGameMenuAction(action); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	missionMenu := func(action string) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a := smoothingMission(t)
			f.ConfigureSaveSeams(a, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if action != "" {
				if err := a.HeadlessGameMenuAction(action); err != nil {
					t.Fatal(err)
				}
			}
			return f, a
		}
	}
	tooltipShows := func(what string) func(*testing.T, *FrontEnd, *ui.App) {
		return func(t *testing.T, _ *FrontEnd, a *ui.App) {
			if st, _ := a.HeadlessTooltip(); !st.Visible {
				t.Fatalf("%s raised no tooltip", what)
			}
		}
	}
	missionHover := func(at func(*ui.App) (int, int, error)) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a := smoothingMission(t)
			smoothingTicks(t, a, 90)
			for i := 0; i < 16 && a.HeadlessNoticeOpen(); i++ {
				if err := a.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.HeadlessKey("select-all"); err != nil {
				t.Fatal(err)
			}
			smoothingTicks(t, a, 2)
			a.Layout(smoothingW, smoothingH)
			x, y, err := at(a)
			if err != nil {
				t.Fatal(err)
			}
			if err = a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			return f, a
		}
	}
	ending := func(page string) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) { return smoothingCompleted(t, page) }
	}
	worldMap := func(hover string) func(*testing.T) (*FrontEnd, *ui.App) {
		return func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a, s := worldMapAtChapter(t, 120)
			missions := s.WorldMapView().Missions
			if len(missions) < 2 || missions[0].Number != 130 || missions[1].Number != 131 {
				t.Fatalf("the map offers %d missions after mission 120, want 130 and 131 first", len(missions))
			}
			var x, y int
			var err error
			switch hover {
			case "mission":
				x, y, err = a.HeadlessWorldMapMissionPoint(missions[0].Number)
			case "town":
				x, y, err = a.HeadlessWorldMapTownPoint()
			}
			if err != nil {
				t.Fatal(err)
			}
			if hover != "" {
				if err = a.HeadlessPointer("hover", x, y); err != nil {
					t.Fatal(err)
				}
			}
			smoothingTicks(t, a, 3)
			return f, a
		}
	}
	return []smoothingRun{
		{name: "main menu", open: front("", 0),
			glyphless: "the menu's words are shipped bitmap art"},
		{name: "load dialog", open: front("load game", 0)},
		{name: "hall of fame", open: front("hall of fame", 0)},
		{name: "credits", open: front("credits", 300)},
		{name: "cutscene library", open: front("cutscenes", 0),
			hidden: "the up and down labels lie under the scroll arrow pictures"},
		{name: "pre-create page, tip open", open: preCreate(false, false),
			hidden: "the ornate tip frame covers the name prompt and field",
			extra:  smoothingPreCreateTipOcclusion,
			ready: func(t *testing.T, _ *FrontEnd, a *ui.App) {
				if st, _ := a.HeadlessChargenState(); st.Stage != ui.ChargenStagePreCreate {
					t.Fatalf("the screen shows the %s page", st.Stage)
				}
			}},
		{name: "pre-create page, tip closed", open: preCreate(true, false)},
		{name: "pre-create page, pointer on the name field", open: preCreate(true, true),
			ready:  tooltipShows("the name field"),
			hidden: "the tooltip covers whole glyphs of the prompt and the name"},
		{name: "character creation, detailed page", open: detailed},
		{name: "town square, tip open", open: town(func(*testing.T, *ui.App, *townScreen) {})},
		{name: "town square, tip closed", open: town(func(t *testing.T, a *ui.App, s *townScreen) { s.CloseTip() }),
			glyphless: "the square's own words are shipped bitmap art and the tip was its only font text"},
		{name: "shop, conversation", open: room("SHOP", true),
			ready: func(t *testing.T, f *FrontEnd, a *ui.App) {
				if s := f.TownScreen().(*townScreen); s.room != roomTalk {
					t.Fatal("the shop opened without its conversation")
				}
			},
			hidden: "the conversation panel covers glyphs of the room behind it"},
		{name: "shop, shelves and prices", open: room("SHOP", false)},
		{name: "shop, item popup", open: func(t *testing.T) (*FrontEnd, *ui.App) {
			f, a, s := smoothingTown(t)
			smoothingRoom(t, a, s, "SHOP", false)
			x, y, err := a.HeadlessShopPoint("shelf", 0)
			if err != nil {
				t.Fatal(err)
			}
			if err = a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			smoothingTicks(t, a, 30)
			return f, a
		}, ready: tooltipShows("the first shelf cell"),
			hidden: "the popup covers glyphs of the shelf behind it"},
		{name: "school", open: room("SCHOOL", false)},
		{name: "tavern", open: room("TAVERN", false)},
		{name: "world map, scrolls after mission 120", open: worldMap(""), extra: checkScrollTitles},
		{name: "world map, pointer on a mission scroll", open: worldMap("mission"), extra: checkScrollTitles},
		{name: "world map, pointer on the town scroll", open: worldMap("town"), extra: checkScrollTitles},
		{name: "ending, credits", open: ending("credits"), steps: 300},
		{name: "ending, hall of fame after the campaign", open: ending("hall")},
		{name: "game menu in town", open: menu("")},
		{name: "game options in town", open: menu("game-options")},
		{name: "sound options in town", open: menu("sound-options")},
		{name: "save dialog", open: menu("save")},
		{name: "load dialog from the game menu", open: menu("load")},
		{name: "mission, opening notice", open: smoothingMission, steps: 90, onMap: true,
			ready: func(t *testing.T, _ *FrontEnd, a *ui.App) {
				if !a.HeadlessNoticeOpen() {
					t.Fatal("the mission opened no notice")
				}
			}},
		{name: "mission, panels and character card", steps: 30, onMap: true,
			open: func(t *testing.T) (*FrontEnd, *ui.App) {
				f, a := smoothingMission(t)
				smoothingTicks(t, a, 90)
				for i := 0; i < 16 && a.HeadlessNoticeOpen(); i++ {
					if err := a.HeadlessKey("enter"); err != nil {
						t.Fatal(err)
					}
				}
				if err := a.HeadlessKey("select-all"); err != nil {
					t.Fatal(err)
				}
				return f, a
			}},
		{name: "mission, pickup message line", open: smoothingPickupMessage, steps: 1, onMap: true,
			ready: func(t *testing.T, f *FrontEnd, _ *ui.App) {
				if len(f.live.view.MessageLines()) == 0 {
					t.Fatal("the pickup left no line in the message line")
				}
			}},
		{name: "mission, bound quick-spell digits", open: smoothingCaster, steps: 1, onMap: true,
			ready: func(t *testing.T, f *FrontEnd, a *ui.App) {
				slots, current, _ := f.live.view.QuickSpellState()
				if slots[0] != 23 || current != 23 {
					t.Fatalf("quick slots %v, current %d; want slot F5 and current 23", slots, current)
				}
				if _, _, err := a.HeadlessSpellPoint(23); err != nil {
					t.Fatal(err)
				}
			},
			extra: func(t *testing.T, f *FrontEnd, a *ui.App, fates []ui.TextFate) {
				x, y, err := a.HeadlessSpellPoint(23)
				if err != nil {
					t.Fatal(err)
				}
				size := f.live.view.FrameSize()
				at, ok := frame.Fit(size.X, size.Y, smoothingW, smoothingH).WindowToFrame(x, y)
				if !ok {
					t.Fatal("the visible spell cell lies outside the frame")
				}
				cell := image.Rect(at.X-24, at.Y-24, at.X+24, at.Y+24)
				marks := 0
				for _, ft := range fates {
					if ft.Call.Glyph == f.tipFont().GlyphFor('5') && image.Pt(ft.Call.X, ft.Call.Y).In(cell) && ft.Fate == textsmooth.Kept {
						marks++
					}
				}
				if marks != 2 {
					t.Fatalf("the bound cell kept %d digit glyphs, want its F5 face and shadow", marks)
				}
			}},
		{name: "mission, debug readout", steps: 1, onMap: true,
			open: func(t *testing.T) (*FrontEnd, *ui.App) {
				f, a := smoothingMission(t)
				if err := a.HeadlessKey("f11"); err != nil {
					t.Fatal(err)
				}
				return f, a
			},
			ready: func(t *testing.T, f *FrontEnd, _ *ui.App) {
				if !f.live.view.ReadoutShown() {
					t.Fatal("F11 left the diagnostic readout hidden")
				}
			}},
		{name: "mission, command button tip", steps: 60, onMap: true,
			open:  missionHover(func(a *ui.App) (int, int, error) { return a.HeadlessCommandPoint(0) }),
			ready: tooltipShows("the first command button")},
		{name: "mission, pack cell item popup", steps: 60, onMap: true,
			open:  missionHover(func(a *ui.App) (int, int, error) { return a.HeadlessPackCellPoint(0) }),
			ready: tooltipShows("the first pack cell")},
		{name: "mission, doll slot item popup", steps: 60, onMap: true,
			open:  missionHover(func(a *ui.App) (int, int, error) { return a.HeadlessDollSlotPoint(1) }),
			ready: tooltipShows("the first doll slot")},
		{name: "mission, game menu", open: missionMenu(""), steps: 20, onMap: true},
		{name: "mission, quest objectives", open: missionMenu("objectives"), steps: 20, onMap: true},
		{name: "mission, game options", open: missionMenu("game-options"), steps: 20, onMap: true},
		{name: "mission, sound options", open: missionMenu("sound-options"), steps: 20, onMap: true},
		{name: "mission, save dialog", open: missionMenu("save"), steps: 20, onMap: true},
	}
}

// checkScrollTitles holds the three top scrolls of the world map, the two
// mission scrolls and the town's own, to their titles: every character of a
// title that is inked shows as a smoothed glyph inside its scroll. The names
// painted into the map picture are art and are not glyphs.
func checkScrollTitles(t *testing.T, f *FrontEnd, a *ui.App, fates []ui.TextFate) {
	t.Helper()
	view := f.TownScreen().(*townScreen).WorldMapView()
	home := view.Words.WorldHomeTitle
	if home == "" {
		home = ui.AuthoredWords().WorldHomeTitle
	}
	readable := func(title string) string {
		if f.textSelector() == text.SelectorConverting {
			if dec, err := charmap.CodePage866.NewDecoder().String(title); err == nil {
				title = dec
			}
		}
		return strings.TrimSpace(title)
	}
	cards := []struct {
		name  string
		slot  int
		title string
	}{
		{"town scroll", -1, home},
		{"mission 130 scroll", 0, view.Missions[0].Title},
		{"mission 131 scroll", 1, view.Missions[1].Title},
	}
	for _, c := range cards {
		card := ui.WorldMapCardRect(c.slot)
		inked := 0
		for _, b := range []byte(c.title) {
			if b > ' ' {
				inked++
			}
		}
		kept := 0
		for _, ft := range fates {
			g := ft.Call.Glyph
			if g == nil || g.Width <= 0 || !image.Pt(ft.Call.X, ft.Call.Y).In(card) || ft.Call.Y >= card.Min.Y+card.Dy()/2 {
				continue
			}
			if ft.Fate == textsmooth.Kept || ft.Fate == textsmooth.Repeat {
				kept++
			}
		}
		t.Logf("%s %q: %d inked characters, %d smoothed glyphs in the title row of %v", c.name, readable(c.title), inked, kept, card)
		if kept < inked {
			t.Errorf("%s %q: %d smoothed glyphs for %d inked characters", c.name, readable(c.title), kept, inked)
		}
	}
}

// shows reports whether any painted cell of the call holds the colour the call
// gives it in pix.
func shows(call text.DrawCall, pix *image.RGBA) bool {
	g := call.Glyph
	for i, p := range g.Pixels {
		at := image.Pt(call.X+i%g.Width, call.Y+i/g.Width)
		if p.Painted && at.In(pix.Bounds()) && pix.RGBAAt(at.X, at.Y) == call.ShownColor(p.Level) {
			return true
		}
	}
	return false
}

func smoothingPreCreateTipOcclusion(t *testing.T, f *FrontEnd, a *ui.App, fates []ui.TextFate) {
	t.Helper()
	state, ok := a.HeadlessChargenState()
	if !ok || state.Stage != ui.ChargenStagePreCreate {
		t.Fatal("tip occlusion audit needs the pre-create page")
	}
	if f.ChargenAssets == nil || f.ChargenAssets.Presentation == nil || f.ChargenAssets.Presentation.NameFont == nil {
		t.Fatal("tip occlusion audit needs the installed prompt/name font")
	}
	font := f.ChargenAssets.Presentation.NameFont
	type promptCell struct {
		glyph *text.Glyph
		x, y  int
		ink   color.RGBA
	}
	allowed := map[promptCell]bool{}
	for _, line := range []struct {
		value string
		at    image.Point
		ink   color.RGBA
	}{
		{f.ChargenAssets.Prompt, image.Pt(224, 305), color.RGBA{65, 47, 20, 255}},
		{state.Name + "|", image.Pt(224, 321), color.RGBA{101, 39, 61, 255}},
	} {
		for i := 0; i < len(line.value); i++ {
			allowed[promptCell{font.GlyphFor(line.value[i]), line.at.X + font.Advance(line.value[:i]), line.at.Y, line.ink}] = true
		}
	}
	pieces := releaseTipFramePieces(t)
	opaque := map[image.Point]bool{}
	for _, tile := range releaseTipTiles(image.Rect(160, 280, 464, 472)) {
		pic := pieces[tile.piece]
		for y := 0; y < pic.Rect.Dy(); y++ {
			for x := 0; x < pic.Rect.Dx(); x++ {
				if pic.RGBAAt(x, y).A == 255 {
					opaque[tile.at.Add(image.Pt(x, y))] = true
				}
			}
		}
	}
	for _, ft := range fates {
		if ft.Fate != textsmooth.Hidden {
			continue
		}
		c := ft.Call
		if c.Flat || !allowed[promptCell{c.Glyph, c.X, c.Y, c.Color}] {
			t.Fatalf("hidden glyph at (%d,%d) is outside the installed prompt/name draws", c.X, c.Y)
		}
		painted := 0
		for n, p := range c.Glyph.Pixels {
			if !p.Painted {
				continue
			}
			painted++
			at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			if !opaque[at] {
				t.Fatalf("hidden prompt/name glyph at (%d,%d) has a cell at %v outside the opaque tip frame", c.X, c.Y, at)
			}
		}
		if painted == 0 {
			t.Fatal("tip occlusion classified an empty glyph as hidden")
		}
	}
}

// smoothingResult is what one witness frame settled.
type smoothingResult struct {
	captured, kept, hidden, blank int
	audited, missed               int
}

// checkSmoothing draws one frame of the run and holds it to the witness.
func checkSmoothing(t *testing.T, run smoothingRun) smoothingResult {
	t.Helper()
	if s := frame.Fit(640, 480, smoothingW, smoothingH).Scale(); s <= 1 {
		t.Fatalf("the window fits the frame at scale %v, want above 1", s)
	}
	t.Cleanup(func() { text.StopAudit() })
	if run.onMap {
		text.StartAudit()
	}
	f, a := run.open(t)
	a.Layout(smoothingW, smoothingH)
	smoothingTicks(t, a, run.steps)
	if run.ready != nil && !run.onMap {
		run.ready(t, f, a)
	}
	if !run.onMap {
		// A composition made while a tick runs is measured or discarded and
		// never presented; only what Draw paints is frame content.
		text.StartAudit()
	}
	a.Draw(ebiten.NewImage(smoothingW, smoothingH))
	audit := text.StopAudit()
	if run.ready != nil && run.onMap {
		// Reading a state back can compose a picture outside every window, so a
		// mission run reads it once the audit is closed.
		run.ready(t, f, a)
	}
	captured, kept, fallbacks := a.TextSettle()
	fates := a.TextFates()

	res := smoothingResult{captured: captured, kept: kept, audited: audit.Captured, missed: audit.Uncaptured}
	if audit.Uncaptured != 0 {
		t.Errorf("%d glyphs were painted outside every capture window: %v", audit.Uncaptured, audit.Sites)
	}
	if fallbacks != 0 {
		t.Errorf("the frame needed %d readbacks; the pixel log should decide every glyph", fallbacks)
	}
	if len(fates) != captured {
		t.Errorf("the frame captured %d glyphs and the fates name %d", captured, len(fates))
	}
	var cpu *image.RGBA
	if !run.onMap {
		var err error
		if cpu, _, err = a.HeadlessFrame(); err != nil {
			t.Fatalf("no CPU composite of the %s screen: %v", a.Screen(), err)
		}
	}
	listed := 0
	for _, ft := range fates {
		switch ft.Fate {
		case textsmooth.Kept, textsmooth.Repeat:
		case textsmooth.Blank:
			res.blank++
		case textsmooth.Hidden:
			res.hidden++
			switch {
			case run.hidden == "":
				t.Errorf("a glyph at (%d,%d) is hidden and this screen names no reason", ft.Call.X, ft.Call.Y)
			case cpu != nil && shows(ft.Call, cpu):
				t.Errorf("a glyph at (%d,%d) is hidden and its cells show in the frame", ft.Call.X, ft.Call.Y)
			}
		default:
			if listed++; listed <= 8 {
				t.Errorf("a glyph at (%d,%d) colour %v is %s: it stays as the raster stepped it", ft.Call.X, ft.Call.Y, ft.Call.Color, ft.Fate)
			}
		}
	}
	if cpu != nil {
		if visible, _ := visibleGlyphs(text.Captured(), cpu); kept < visible {
			t.Errorf("the overlay smoothed %d glyphs and %d show in the frame", kept, visible)
		}
	}
	switch {
	case run.glyphless != "":
		if captured != 0 || audit.Captured != 0 {
			t.Errorf("the screen is said to draw no glyph (%s) and captured %d", run.glyphless, captured)
		}
	case kept == 0:
		t.Errorf("the screen kept no glyph")
	}
	if run.extra != nil {
		run.extra(t, f, a, fates)
	}
	return res
}

// Every named screen paints its text through a capture window and the overlay
// keeps every glyph that shows, at a window scale above 1, on both installs.
// Each screen is reached through App input and drawn once. A glyph the overlay
// leaves baked, a glyph painted outside every capture window and a frame that
// needed a GPU readback each fail. A screen that hides a glyph by design
// states why, and the witness checks that none of that glyph's cells shows in
// the CPU composite of the frame.
func TestReleaseTextSmoothingKeepsEveryGlyphOfEveryNamedScreen(t *testing.T) {
	for _, run := range smoothingViews() {
		t.Run(run.name, func(t *testing.T) {
			res := checkSmoothing(t, run)
			t.Logf("SMOOTHING %-46s captured %4d kept %4d blank %3d hidden %3d audited %4d uncaptured %d",
				run.name, res.captured, res.kept, res.blank, res.hidden, res.audited, res.missed)
			if run.hidden != "" {
				t.Logf("SMOOTHING %-46s hidden by design: %s", run.name, run.hidden)
			}
			if run.glyphless != "" {
				t.Logf("SMOOTHING %-46s draws no glyph: %s", run.name, run.glyphless)
			}
		})
	}
}
