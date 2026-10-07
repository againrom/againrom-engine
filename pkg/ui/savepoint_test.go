package ui

import "testing"

type savePointTown struct {
	*stubTown
	ready bool
}

func (s *savePointTown) CanSave() bool { return s.ready }

func TestSavePointPendingTownBlocksF2AndMenuButKeepsF3(t *testing.T) {
	for _, route := range []string{"F2", "menu", "F3", "F2 and F3"} {
		t.Run(route, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			town := &savePointTown{stubTown: &stubTown{rows: []TownRow{{Text: "town", Choosable: true}}}}
			h.a.SetTown(town)
			if !h.a.flow.showTown("") {
				t.Fatal("town unavailable")
			}
			spy := &spySaveSeams{}
			spy.install(h.a)
			switch route {
			case "F2":
				h.frame(appInput{SaveGame: true})
				if h.a.Screen() != ScreenTown {
					t.Fatal("pending return opened SAVE", h.a.Screen())
				}
			case "menu":
				h.frame(appInput{Escape: true})
				if h.a.Screen() != ScreenGameMenu {
					t.Fatal("town menu unavailable")
				}
				for _, row := range h.a.flow.menuRows() {
					if row.Action == gameMenuSave && row.Enabled {
						t.Fatal("pending return enabled menu SAVE")
					}
				}
				h.a.flow.applyGameMenuAction(gameMenuSave)
			case "F3":
				h.frame(appInput{LoadGame: true})
				if h.a.Screen() != ScreenLoad {
					t.Fatal("pending return blocked F3", h.a.Screen())
				}
			case "F2 and F3":
				h.frame(appInput{SaveGame: true, LoadGame: true})
				if h.a.Screen() != ScreenLoad {
					t.Fatal("blocked F2 swallowed F3", h.a.Screen())
				}
			}
			if len(spy.saved) != 0 {
				t.Fatal("pending return reached SAVE seam")
			}
		})
	}
}

func TestSavePointMissionAndResolvedTownKeepF2(t *testing.T) {
	for _, city := range []bool{false, true} {
		h := newHaltFix(t, haltOpts{})
		if city {
			h.a.SetTown(&savePointTown{stubTown: &stubTown{rows: []TownRow{{Text: "town", Choosable: true}}}, ready: true})
			h.a.flow.showTown("")
		}
		spy := &spySaveSeams{}
		spy.install(h.a)
		h.frame(appInput{SaveGame: true})
		if len(spy.saved) != 1 || spy.saved[0] == city {
			t.Fatal("valid save point lost F2", city, spy.saved)
		}
	}
}

func TestSavePointEndingHallAndCreditsNeverSave(t *testing.T) {
	for _, surface := range []string{"ending", "ending hall", "menu hall", "credits roll"} {
		t.Run(surface, func(t *testing.T) {
			a := endingTestApp(t)
			legacy, dialog := &spySaveSeams{}, &saveDialogSpy{}
			legacy.install(a)
			dialog.install(a)
			var destination string
			switch surface {
			case "ending hall":
				destination = a.flow.endingWords().Back
			case "menu hall":
				view := a.flow.ending
				a.SetHallOfFame(func() EndingView { return view })
				a.flow.toMenu()
				destination = "hall of fame"
			case "credits roll":
				a.SetCredits(func() CreditsView { return CreditsView{Lines: []string{"Author"}} })
				a.flow.showCampaignEnding()
			}
			if destination != "" {
				if err := a.HeadlessActivate(destination); err != nil {
					t.Fatal("open surface", err)
				}
			}
			for _, back := range []Screen{ScreenEnding, ScreenMap, ScreenTown} {
				screen := a.Screen()
				a.flow.menuBack = back
				a.flow.openSaveDialog()
				a.flow.applyGameMenuAction(gameMenuSave)
				a.SetSaveDialogSeams(SaveDialogSeams{})
				a.flow.applyGameMenuAction(gameMenuSave)
				dialog.install(a)
				if a.Screen() != screen || a.flow.saveDialog != nil {
					t.Fatal("direct SAVE entry opened a non-game save point", back, a.Screen())
				}
			}
			if err := a.HeadlessKey("f2"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() == ScreenSave {
				t.Fatal("F2 opened SAVE outside a city or mission")
			}
			if err := a.HeadlessActivate("Save game"); err == nil {
				t.Error("surface exposes a SAVE control outside a city or mission")
				if err := a.HeadlessSaveAction("save"); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.HeadlessGameMenuAction("save"); err == nil {
				t.Error("surface exposes the game-menu SAVE action")
			}
			if len(legacy.saved) != 0 || len(dialog.requests) != 0 || len(dialog.commits) != 0 || len(dialog.lists) != 0 {
				t.Fatalf("non-game surface reached SAVE: automatic=%v prepared=%v committed=%v listed=%v", legacy.saved, dialog.requests, dialog.commits, dialog.lists)
			}
		})
	}
}
