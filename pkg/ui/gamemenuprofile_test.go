package ui

import "testing"

var (
	firstMenuProfile  = GameMenuProfile{}
	secondMenuProfile = GameMenuProfile{WindowsCloseMenu: true, EndQuestFocus: 1, AbortGameFocus: 2}
)

// holdSavePage shows an empty save page over the game menu.
func holdSavePage(f *flow, page SavePage) {
	f.saveScreen = &savePageState{page: page, ui: loadWindow{role: saveListRole()}}
	f.loadBack = ScreenGameMenu
	f.setScreen(ScreenSave)
}

// windowLeaves are the ways a window opened from the menu ends: the page it is
// on, and the step that ends it.
var windowLeaves = []struct {
	name string
	open func(f *flow)
	end  func(f *flow)
}{
	{"save ok", func(f *flow) { holdSavePage(f, SavePage{}) }, func(f *flow) { f.leaveLoad("Your character is saved", true) }},
	{"save cancel", func(f *flow) { holdSavePage(f, SavePage{}) }, func(f *flow) { f.leaveLoad("", false) }},
	{"save dialog", func(f *flow) { holdSavePage(f, firstSavePage()) }, func(f *flow) { f.leaveLoad("", false) }},
	{"load cancel", func(f *flow) { f.applyGameMenuAction(gameMenuLoad) }, func(f *flow) { f.closeLoad() }},
	{"options cancel", func(f *flow) { f.applyGameMenuAction(gameMenuGameOptions) }, func(f *flow) { f.applyGameMenuAction(gameMenuOptionsCancel) }},
	{"sound escape", func(f *flow) { f.applyGameMenuAction(gameMenuSoundOptions) }, func(f *flow) { f.escape() }},
	{"quest ok", func(f *flow) { f.applyGameMenuAction(gameMenuQuestObjectives) }, func(f *flow) { f.applyGameMenuAction(gameMenuPageReturn) }},
	{"end quest return", func(f *flow) { f.applyGameMenuAction(gameMenuEndQuest) }, func(f *flow) { f.applyGameMenuAction(gameMenuPageReturn) }},
}

func TestWindowLastButtonLeavesByTheGamesProfile(t *testing.T) {
	for _, tc := range windowLeaves {
		for _, p := range []struct {
			game    string
			profile GameMenuProfile
			closes  bool
		}{{"first", firstMenuProfile, false}, {"second", secondMenuProfile, true}} {
			t.Run(tc.name+"/"+p.game, func(t *testing.T) {
				f := openMissionMenu(t)
				f.pages.menu = p.profile
				tc.open(f)
				tc.end(f)
				if p.closes {
					if f.screen != ScreenMap || f.menuPage != gameMenuRoot || f.menuList != nil || f.msg != "" {
						t.Fatalf("screen %v page %v list %v msg %q, want the mission with the menu closed",
							f.screen, f.menuPage, f.menuList != nil, f.msg)
					}
					if f.viewer != nil && f.viewer.menuUp {
						t.Fatal("viewer still holds the menu up")
					}
					return
				}
				if f.screen != ScreenGameMenu || f.menuPage != gameMenuRoot || f.menuList == nil {
					t.Fatalf("screen %v page %v, want the menu root", f.screen, f.menuPage)
				}
			})
		}
	}
}

func TestSaveOkKeepsItsLineOnlyWhereTheMenuStays(t *testing.T) {
	f := openMissionMenu(t)
	holdSavePage(f, SavePage{})
	f.leaveLoad("Your character is saved", true)
	if f.msg != "Your character is saved" {
		t.Fatalf("first game msg = %q", f.msg)
	}
}

func TestConfirmationFirstFocusByProfile(t *testing.T) {
	for _, p := range []struct {
		game             string
		profile          GameMenuProfile
		endQuest, abort  int
		endRow, abortRow gameMenuAction
	}{
		{"first", firstMenuProfile, 0, 0, gameMenuConfirmEndQuest, gameMenuExitMain},
		{"second", secondMenuProfile, 1, 2, gameMenuExitMain, gameMenuPageReturn},
	} {
		t.Run(p.game, func(t *testing.T) {
			f := openMissionMenu(t)
			f.pages.menu = p.profile
			f.applyGameMenuAction(gameMenuEndQuest)
			if got := f.menuList.Selection(); got != p.endQuest || f.menuRows()[got].Action != p.endRow {
				t.Fatalf("End Quest focus row %d action %v", got, f.menuRows()[got].Action)
			}
			g := openTownMenu(t)
			g.pages.menu = p.profile
			g.applyGameMenuAction(gameMenuAbortGame)
			if got := g.menuList.Selection(); got != p.abort || g.menuRows()[got].Action != p.abortRow {
				t.Fatalf("Abort Game focus row %d action %v", got, g.menuRows()[got].Action)
			}
		})
	}
}
