package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/frame"
)

func menuActions(rows []gameMenuRow) []gameMenuAction {
	out := make([]gameMenuAction, len(rows))
	for i := range rows {
		out[i] = rows[i].Action
	}
	return out
}

func TestPauseRootPopulationAndDecodedGates(t *testing.T) {
	w := AuthoredWords()
	tests := []struct {
		name    string
		rows    []gameMenuRow
		actions []gameMenuAction
		enabled []bool
	}{
		{
			name: "campaign mission",
			rows: missionGameMenuRows(w, true, true, false, true),
			actions: []gameMenuAction{gameMenuSave, gameMenuLoad, gameMenuGameOptions,
				gameMenuSoundOptions, gameMenuQuestObjectives, gameMenuEndQuest, gameMenuReturn},
			enabled: []bool{true, false, true, true, true, true, true},
		},
		{
			name: "standalone map",
			rows: missionGameMenuRows(w, false, true, true, false),
			actions: []gameMenuAction{gameMenuSave, gameMenuDiplomacy, gameMenuGameOptions,
				gameMenuSoundOptions, gameMenuQuestObjectives, gameMenuEndQuest, gameMenuReturn},
			enabled: []bool{false, true, true, false, false, true, true},
		},
		{
			name: "town",
			rows: townGameMenuRows(w),
			actions: []gameMenuAction{gameMenuSave, gameMenuLoad, gameMenuGameOptions, gameMenuSoundOptions,
				gameMenuAbortGame, gameMenuReturn},
			enabled: []bool{true, true, true, true, true, true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := menuActions(tc.rows); !reflect.DeepEqual(got, tc.actions) {
				t.Fatalf("actions = %v, want %v", got, tc.actions)
			}
			for i, want := range tc.enabled {
				if tc.rows[i].Enabled != want {
					t.Errorf("row %d enabled = %v, want %v", i, tc.rows[i].Enabled, want)
				}
			}
		})
	}
}

// TestPauseConfirmationPopulationAndDecodedGeometry holds MENU-ITEM-012's two
// confirmation constructors together. Both use the town-sized panel, but the
// mission page has five rows and the town page has three; sharing the root
// surface or collapsing either list to a generic yes/no prompt reddens here.
func TestPauseConfirmationPopulationAndDecodedGeometry(t *testing.T) {
	w := AuthoredWords()
	tests := []struct {
		name    string
		page    gameMenuPage
		context GameMenuContext
		actions []gameMenuAction
		labels  []string
		keys    []byte
		enabled []bool
	}{
		{
			name:    "campaign before Continue",
			page:    gameMenuEndQuestConfirmation,
			context: GameMenuContext{Campaign: true, CampaignVictory: true},
			actions: []gameMenuAction{gameMenuVictory, gameMenuExitMain,
				gameMenuExitWindows, gameMenuPageReturn},
			labels:  []string{w.MenuVictory, w.MenuExitMain, w.MenuExitWindows, w.MenuReturn},
			keys:    []byte{'v', 'e', 'w', 'r'},
			enabled: []bool{false, true, true, true},
		},
		{
			name:    "campaign after Continue",
			page:    gameMenuEndQuestConfirmation,
			context: GameMenuContext{Campaign: true, CampaignVictory: true, VictoryAvailable: true},
			actions: []gameMenuAction{gameMenuVictory, gameMenuExitMain,
				gameMenuExitWindows, gameMenuPageReturn},
			labels:  []string{w.MenuVictory, w.MenuExitMain, w.MenuExitWindows, w.MenuReturn},
			keys:    []byte{'v', 'e', 'w', 'r'},
			enabled: []bool{true, true, true, true},
		},
		{
			name:    "standalone map",
			page:    gameMenuEndQuestConfirmation,
			context: GameMenuContext{Campaign: false},
			actions: []gameMenuAction{gameMenuConfirmEndQuest, gameMenuExitMain,
				gameMenuExitWindows, gameMenuPageReturn},
			labels:  []string{w.MenuChangeMap, w.MenuExitMain, w.MenuExitWindows, w.MenuReturn},
			keys:    []byte{'c', 'e', 'w', 'r'},
			enabled: []bool{true, true, true, true},
		},
		{
			name:    "town",
			page:    gameMenuAbortGameConfirmation,
			actions: []gameMenuAction{gameMenuExitMain, gameMenuExitWindows, gameMenuPageReturn},
			labels:  []string{w.MenuExitMain, w.MenuExitWindows, w.MenuReturn},
			keys:    []byte{'e', 'w', 'r'},
			enabled: []bool{true, true, true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := openMissionMenu(t)
			f.menuContext = tc.context
			f.rebuildGameMenu(tc.page, 0)
			rows := f.menuRows()
			if got := menuActions(rows); !reflect.DeepEqual(got, tc.actions) {
				t.Fatalf("actions = %v, want %v", got, tc.actions)
			}
			if len(rows) != len(tc.labels) {
				t.Fatalf("rows = %d, want %d", len(rows), len(tc.labels))
			}
			for i, want := range tc.labels {
				if rows[i].Label != want || rows[i].Enabled != tc.enabled[i] {
					t.Errorf("row %d = (%q,%v), want (%q,%v)", i, rows[i].Label, rows[i].Enabled, want, tc.enabled[i])
				}
				if got, ok := rows[i].accelerator(0); !ok || got != tc.keys[i] {
					t.Errorf("row %d accelerator = (%q,%v), want (%q,true)", i, got, ok, tc.keys[i])
				}
			}
			if got := gameMenuPanelRect(f.menuPanelSurface()); got != image.Rect(100, 100, 440, 340) {
				t.Fatalf("panel = %v, want decoded confirmation rectangle", got)
			}
		})
	}
}

// TestPauseConfirmationDestinationsAreReachable exercises each terminal
// action through the production chooser. Change Map and Victory both release
// the current map to the picker; Exit to Main Menu selects the main-menu screen
// while retaining the current viewer/session seams; Exit to Windows reaches
// App's exit result instead of being a no-op.
func TestPauseConfirmationDestinationsAreReachable(t *testing.T) {
	standalone := openMissionMenu(t)
	standalone.menuContext = GameMenuContext{Campaign: false}
	standalone.rebuildGameMenu(gameMenuEndQuestConfirmation, 0)
	if !standalone.chooseGameMenuAccelerator('c') || standalone.screen != ScreenPicker {
		t.Fatalf("standalone Change Map reached %v", standalone.screen)
	}

	campaign := openMissionMenu(t)
	campaign.menuContext = GameMenuContext{Campaign: true, CampaignVictory: true, VictoryAvailable: true}
	campaign.advance = func(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
		if len(actions) != 1 || actions[0] != NoticeVictory {
			t.Fatalf("Victory seam actions = %v", actions)
		}
		return NoticeToMapList, "won", nil
	}
	campaign.rebuildGameMenu(gameMenuEndQuestConfirmation, 0)
	selectGameMenuAction(campaign, gameMenuVictory)
	campaign.chooseGameMenu()
	if campaign.screen != ScreenPicker {
		t.Fatalf("campaign Victory reached %v, want map picker", campaign.screen)
	}

	for _, tc := range []struct {
		page   gameMenuPage
		action gameMenuAction
	}{
		{gameMenuEndQuestConfirmation, gameMenuExitMain},
		{gameMenuAbortGameConfirmation, gameMenuExitMain},
	} {
		f := openMissionMenu(t)
		viewer := &Viewer{}
		f.viewer = viewer
		f.tick = func() {}
		f.rebuildGameMenu(tc.page, 0)
		selectGameMenuAction(f, tc.action)
		f.chooseGameMenu()
		if f.screen != ScreenMenu {
			t.Errorf("action %v reached %v, want main menu", tc.action, f.screen)
		}
		if f.viewer != viewer || f.tick == nil {
			t.Errorf("action %v released the retained viewer/session seams", tc.action)
		}
	}

	f := openMissionMenu(t)
	f.rebuildGameMenu(gameMenuEndQuestConfirmation, 0)
	selectGameMenuAction(f, gameMenuExitWindows)
	a := &App{flow: f}
	if exit := a.stepGameMenu(appInput{Enter: true}); !exit {
		t.Fatal("Exit to Windows did not request application exit")
	}
}

// TestPauseDestinationsTakeProductionEnterAndAcceleratorPaths is B2-B5 over
// the same chooser and accelerator dispatcher a running App uses.
func TestPauseDestinationsTakeProductionEnterAndAcceleratorPaths(t *testing.T) {
	tips := true
	soundOn, volume := true, 50
	f := openMissionMenu(t)
	f.menuContext.Objective = "Protect the bridge until relief arrives."
	f.menuTips = func() bool { return tips }
	f.setMenuTips = func(on bool) { tips = on }
	f.menuSound = func() (bool, int, bool) { return soundOn, volume, true }
	f.setMenuSound = func(on bool, v int) error { soundOn, volume = on, v; return nil }
	f.rebuildGameMenu(gameMenuRoot, 0)

	selectGameMenuAction(f, gameMenuGameOptions)
	f.chooseGameMenu()
	if f.menuPage != gameMenuGameOptionsPage || !f.chooseGameMenuAccelerator('t') || tips {
		t.Fatalf("Game Options did not toggle persisted tips: page %v tips %v", f.menuPage, tips)
	}
	f.rebuildGameMenu(gameMenuRoot, 0)
	if !f.chooseGameMenuAccelerator('n') || f.menuPage != gameMenuSoundOptionsPage {
		t.Fatalf("Sound Options accelerator reached page %v", f.menuPage)
	}
	if !f.chooseGameMenuAccelerator('d') || volume != 25 {
		t.Fatalf("volume down produced %d, want 25", volume)
	}
	f.rebuildGameMenu(gameMenuRoot, 0)
	selectGameMenuAction(f, gameMenuQuestObjectives)
	f.chooseGameMenu()
	if f.menuPage != gameMenuQuestObjectivesPage || len(f.menuRows()) < 3 {
		t.Fatalf("Quest Objectives page = %v with %d rows", f.menuPage, len(f.menuRows()))
	}
	if got := f.menuRows()[1].text(); got != "Protect the bridge until relief" {
		t.Fatalf("objective first line = %q", got)
	}
	f.rebuildGameMenu(gameMenuRoot, 0)
	selectGameMenuAction(f, gameMenuEndQuest)
	f.chooseGameMenu()
	if f.menuPage != gameMenuEndQuestConfirmation || f.screen != ScreenGameMenu {
		t.Fatalf("End Quest left without confirmation: page %v screen %v", f.menuPage, f.screen)
	}
	if f.escape() || f.menuPage != gameMenuRoot {
		t.Fatalf("Escape did not cancel confirmation back to the root")
	}
}

// TestScrolledDiplomacyReturnUsesTheVisiblePointerSlot is the long-page input
// mutation witness. A pointer row is a visible slot plus Picker.Visible's top;
// dropping that addition selects disabled information row 6 and cannot return.
func TestScrolledDiplomacyReturnUsesTheVisiblePointerSlot(t *testing.T) {
	f := openMissionMenu(t)
	f.menuContext.Campaign = false
	for slot := uint32(2); slot <= 11; slot++ {
		f.menuContext.Relations = append(f.menuContext.Relations, GameMenuRelation{Slot: slot, State: "NEUTRAL"})
	}
	f.rebuildGameMenu(gameMenuDiplomacyPage, 0)
	last := len(f.menuRows()) - 1
	f.menuList.Select(last)
	top, count := f.menuList.Visible()
	if top == 0 || count != 7 {
		t.Fatalf("fixture did not scroll: Visible = (%d,%d)", top, count)
	}
	r := gameMenuRowRect(gameMenuMission, count-1)
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	a := &App{flow: f, place: frame.Fit(frame.W, frame.H, frame.W, frame.H)}
	a.stepGameMenu(appInput{PrimaryPressed: true, CursorX: p.X, CursorY: p.Y})
	a.stepGameMenu(appInput{PrimaryReleased: true, CursorX: p.X, CursorY: p.Y})
	if f.menuPage != gameMenuRoot {
		t.Fatalf("pointer on visible Return left page %v, want root", f.menuPage)
	}
}
