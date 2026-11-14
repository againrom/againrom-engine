package game

import (
	"slices"
	"strconv"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/herokind"
	"againrom/pkg/ui"
)

// startedArg is the value the recorded launch passed for flag.
func startedArg(t *testing.T, args []string, flag string) string {
	t.Helper()
	at := slices.Index(args, "-"+flag)
	if at < 0 || at+1 >= len(args) {
		t.Fatalf("the launch passes no -%s: %q", flag, args)
	}
	return args[at+1]
}

// TestReleaseTestPlayWithEveryCharacterChosenPlaysThemAll presses F5 on the
// installed campaign's mission 40, chooses every character of the party
// window on Custom, starts, and plays the mod the editor wrote the way the
// launched game does: the started mission holds the hero and every companion
// the window offered as units of the player's party on the map.
func TestReleaseTestPlayWithEveryCharacterChosenPlaysThemAll(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for presses := range 4 {
		t.Run(strconv.Itoa(presses), func(t *testing.T) { everyCharacterChosen(t, presses, true) })
	}
	t.Run("standard", func(t *testing.T) { everyCharacterChosen(t, 0, false) })
}

func everyCharacterChosen(t *testing.T, presses int, custom bool) {
	x, started := f5Inspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/40.alm"); err != nil {
		t.Fatal(err)
	}
	setRoute(t, e, map[string]int{"number": 40}, "replace")
	_ = e.Key("escape")
	s := e.Document().Edits.(*mapInspectionEdits)
	t.Logf("declares its own party: %v", s.TestPlayDeclaresParty())
	_ = e.Key("testplay")
	if !e.TestPartyOpen() {
		t.Fatalf("F5 did not ask who plays the map: %q", e.Message())
	}
	if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 3}); ok && custom {
		partyPressAt(e, r)
	}
	for range presses {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 1})
		if !ok {
			t.Fatal("the kind button takes no press")
		}
		partyPressAt(e, r)
	}
	cards := len(e.TestPartyChoice())
	for card := range cards {
		if custom && !e.TestPartyChoice()[card].On {
			partyPress(t, e, 0, card)
		}
	}
	for i, m := range e.TestPartyChoice() {
		if custom && !m.On {
			t.Fatalf("card %d is not chosen: %+v", i, e.TestPartyChoice())
		}
	}
	kind := herokind.Kind(e.TestPartyHero())
	if err := e.Key("enter"); err != nil || len(*started) != 1 {
		t.Fatalf("Start: %v, launches %d, message %q", err, len(*started), e.Message())
	}
	args := (*started)[0]
	number, err := strconv.Atoi(startedArg(t, args, "mission"))
	if err != nil {
		t.Fatal(err)
	}
	f := guestFront(t, startedArg(t, args, "mods-dir"), TestPlayModID)
	app := f.App("every character")
	sex, class := 0, 0
	if kind.Female() {
		sex = 1
	}
	if kind.Mage() {
		class = 1
	}
	res := ui.ChargenResult{Name: "Guest", Choices: []int{sex, class, 0}, Stats: []int{31, 27, 24, 29}}
	if err := app.OpenMission(f.NewGameOpener(number, res)); err != nil {
		t.Fatal(err)
	}
	party := f.LiveParty()
	var ids []string
	for _, p := range party {
		ids = append(ids, p.ID)
	}
	t.Logf("mission %d, hero kind %s: window offered %d cards, started party %d: %v", number, kind.Word(), cards, len(party), ids)
	if custom && len(party) != cards {
		t.Fatalf("the started party holds %d of the %d characters chosen: %v", len(party), cards, ids)
	}
	for i := range party {
		ent, ok := f.live.world.Entity(f.live.mission.ids[i])
		t.Logf("member %d %s at %d,%d owner %d", i, party[i].ID, ent.X, ent.Y, ent.Owner)
		if !ok || ent.OffMap {
			t.Errorf("member %d (%s) is not a unit on the map: found %v", i, party[i].ID, ok)
		}
	}
}

// TestReleaseTestPlayPartyWindowPressesSwitchOneBox presses every box of the
// party window on mission 40 on Custom at every hero kind, through the
// rectangles the window answers for, and requires the press to switch that
// character's box alone.
func TestReleaseTestPlayPartyWindowPressesSwitchOneBox(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x, _ := f5Inspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/40.alm"); err != nil {
		t.Fatal(err)
	}
	setRoute(t, e, map[string]int{"number": 40}, "replace")
	_ = e.Key("escape")
	_ = e.Key("testplay")
	if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 3}); ok {
		partyPressAt(e, r)
	}
	for round := range 4 {
		n := len(e.TestPartyChoice())
		for row := range 3 {
			for col := range n {
				r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptGridCell, N: row, K: col})
				if !ok {
					continue
				}
				before := slices.Clone(e.TestPartyChoice())
				partyPressAt(e, r)
				after := e.TestPartyChoice()
				if len(after) != n {
					t.Fatalf("round %d: the press changed the card count", round)
				}
				for i := range n {
					if i != col && after[i] != before[i] {
						t.Fatalf("round %d: press row %d card %d switched card %d: %+v -> %+v", round, row, col, i, before[i], after[i])
					}
				}
				if after[col] == before[col] {
					t.Fatalf("round %d: press row %d card %d switched nothing", round, row, col)
				}
			}
		}
		r, _ := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 1})
		partyPressAt(e, r)
	}
}
