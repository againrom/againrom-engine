package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

// gameMenuProfiles are both games' profiles as pkg/game sets them.
var gameMenuProfiles = []struct {
	game    string
	profile GameMenuProfile
}{
	{"first", GameMenuProfile{}},
	{"second", GameMenuProfile{Rows: SecondGameMenuRows, TownRootPanel: image.Rect(100, 100, 540, 440), NativeMissionWindows: true, MissionBackdrop: color.RGBA{A: 64}, WindowsCloseMenu: true, EndQuestFocus: 1, AbortGameFocus: 2}},
}

// decodedRootRows is each place's decoded list, written apart from the data.
func decodedRootRows(w Words, town bool, s gameMenuRowState) []gameMenuRow {
	if town {
		return []gameMenuRow{
			{Label: w.MenuSave, Fallback: 'S', Action: gameMenuSave, Enabled: townCanSave(s.town)},
			{Label: w.MenuLoad, Fallback: 'L', Action: gameMenuLoad, Enabled: true},
			{Label: w.MenuGameOptions, Fallback: 'O', Action: gameMenuGameOptions, Enabled: true},
			{Label: w.MenuSoundOptions, Fallback: 'N', Action: gameMenuSoundOptions, Enabled: true},
			{Label: w.MenuAbort, Fallback: 'E', Action: gameMenuAbortGame, Enabled: true},
			{Label: w.MenuReturn, Fallback: 'R', Action: gameMenuReturn, Enabled: true},
		}
	}
	second := gameMenuRow{Label: w.MenuDiplomacy, Fallback: 'D', Action: gameMenuDiplomacy, Enabled: true}
	if s.campaign {
		second = gameMenuRow{Label: w.MenuLoad, Fallback: 'L', Action: gameMenuLoad, Enabled: s.canLoad}
	}
	return []gameMenuRow{
		{Label: w.MenuSave, Fallback: 'S', Action: gameMenuSave, Enabled: s.campaign && s.canSave},
		second,
		{Label: w.MenuGameOptions, Fallback: 'O', Action: gameMenuGameOptions, Enabled: true},
		{Label: w.MenuSoundOptions, Fallback: 'N', Action: gameMenuSoundOptions, Enabled: s.canSound},
		{Label: w.MenuQuestObjectives, Fallback: 'M', Action: gameMenuQuestObjectives, Enabled: s.campaign},
		{Label: w.MenuEndQuest, Fallback: 'E', Action: gameMenuEndQuest, Enabled: true},
		{Label: w.MenuReturn, Fallback: 'R', Action: gameMenuReturn, Enabled: true},
	}
}

type savingTown struct {
	TownScreen
	can bool
}

func (t savingTown) CanSave() bool { return t.can }

func TestProfileRowsAreTheDecodedRowsForBothGamesAndPlaces(t *testing.T) {
	for _, g := range gameMenuProfiles {
		for _, w := range []Words{AuthoredWords(), installedWords()} {
			for bits := 0; bits < 32; bits++ {
				s := gameMenuRowState{campaign: bits&1 != 0, canSave: bits&2 != 0, canLoad: bits&4 != 0,
					canSound: bits&8 != 0, town: savingTown{can: bits&16 != 0}}
				for _, place := range []gameMenuSurface{gameMenuMission, gameMenuTown} {
					got := gameMenuRootRows(g.profile.rowsAt(place), w, s)
					want := decodedRootRows(w, place == gameMenuTown, s)
					if g.game == "second" && place == gameMenuTown {
						want = popupRows(w, s)
					}
					if fmt.Sprint(got) != fmt.Sprint(want) {
						t.Fatalf("%s game, place %d, state %+v:\n got %v\nwant %v", g.game, place, s, got, want)
					}
				}
			}
		}
	}
}

func TestMenuRowsReadTheProfileRows(t *testing.T) {
	f := openMissionMenu(t)
	f.pages.menu.Rows.Mission = []GameMenuRowSpec{{menuWordReturn, 'R', gameMenuReturn, menuRowAlways, menuRowAlways}}
	rows := f.menuRows()
	if len(rows) != 1 || rows[0].Action != gameMenuReturn || rows[0].Label != f.words.MenuReturn {
		t.Fatalf("rows %v, want the profile's one row", rows)
	}
}

func TestPlacementIsTodaysForBothGamesAndPlaces(t *testing.T) {
	decoded := map[gameMenuSurface]image.Rectangle{
		gameMenuMission: image.Rect(100, 60, 440, 400),
		gameMenuTown:    image.Rect(100, 100, 440, 340),
	}
	centred := map[gameMenuSurface]image.Rectangle{
		gameMenuMission: newDialogGeometry(340, 340).Body(),
		gameMenuTown:    newDialogGeometry(340, 240).Body(),
	}
	for _, g := range gameMenuProfiles {
		second := g.profile.NativeMissionWindows
		if got := g.profile.placement().backdrop; got != g.profile.MissionBackdrop {
			t.Fatalf("%s game backdrop %v", g.game, got)
		}
		for _, place := range []gameMenuSurface{gameMenuMission, gameMenuTown} {
			want := decoded[place]
			if second {
				want = centred[place]
			}
			if got := gameMenuPanelRect(place, g.profile); got != want {
				t.Fatalf("%s game place %d profile panel %v want %v", g.game, place, got, want)
			}
			if got := gameMenuPanelRect(place); got != decoded[place] {
				t.Fatalf("place %d default panel %v", place, got)
			}
		}
		mission := openMissionMenu(t)
		mission.pages.menu = g.profile
		if got := mission.menuPlacement(); got.overMission() != second || gameMenuPanelRect(gameMenuMission, got) != map[bool]image.Rectangle{false: decoded[gameMenuMission], true: centred[gameMenuMission]}[second] {
			t.Fatalf("%s game mission menu placement %+v", g.game, got)
		}
		town := openTownMenu(t)
		town.pages.menu = g.profile
		root := decoded[gameMenuTown]
		if second {
			root = image.Rect(100, 100, 540, 440)
		}
		if got := town.menuPlacement(); got.overMission() || gameMenuPanelRect(gameMenuTown, got) != root || got.backdrop != g.profile.MissionBackdrop {
			t.Fatalf("%s game town root placement %+v", g.game, got)
		}
		town.applyGameMenuAction(gameMenuAbortGame)
		if got := town.menuPlacement(); got.overMission() || gameMenuPanelRect(gameMenuTown, got) != decoded[gameMenuTown] || got.backdrop != g.profile.MissionBackdrop {
			t.Fatalf("%s game town menu placement %+v", g.game, got)
		}
	}
}
