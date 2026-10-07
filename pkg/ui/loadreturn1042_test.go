package ui

import "testing"

func TestCampaignLoadReturnRebuildsAndHoldsTheVisibleMenu(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	entries := []SaveEntry{{Name: "refused.ags", Label: "refused save"}}
	h.a.SetSaveSeams(
		func(bool) (string, error) { return "saved.ags", nil },
		func() []SaveEntry { return entries },
		func(string) (MapOpener, bool, error) { return nil, false, errNotASave },
	)
	h.a.SetGameMenuSettings(nil, nil,
		func() (bool, int, bool) { return true, 75, true }, nil)

	h.a.flow.openGameMenu(ScreenMap)
	oldRoot := h.a.flow.menuList
	if !selectGameMenuAction(h.a.flow, gameMenuLoad) {
		t.Fatal("campaign root has no Load row")
	}
	h.a.flow.chooseGameMenu()
	if h.a.Screen() != ScreenLoad || h.v.menuUp {
		t.Fatalf("Load state = screen %v menuUp %v, want ScreenLoad/false", h.a.Screen(), h.v.menuUp)
	}
	h.a.flow.chooseLoad() // refusal keeps the player on the load screen
	if h.a.Screen() != ScreenLoad || h.a.flow.msg == "" {
		t.Fatalf("refusal state = screen %v message %q", h.a.Screen(), h.a.flow.msg)
	}

	worldBefore := h.w.world
	animBefore := h.v.AnimationCounter()
	for i := 0; i < haltRun; i++ {
		h.frame(haltNeutral())
	}
	if h.w.world != worldBefore || h.v.AnimationCounter() != animBefore {
		t.Fatal("the load screen advanced the map or ambient animation")
	}

	// Availability changes while the list is visible. The return must read it
	// afresh, rather than exposing the stale root stored behind the load screen.
	h.a.SetSaveSeams(nil, func() []SaveEntry { return nil }, h.a.flow.loadGame)
	h.a.SetGameMenuSettings(nil, nil, nil, nil)
	h.frame(appInput{Escape: true})
	if h.a.Screen() != ScreenGameMenu || !h.v.menuUp || !h.v.popupOpen() {
		t.Fatalf("return state = screen %v menuUp %v popup %v, want visible held menu",
			h.a.Screen(), h.v.menuUp, h.v.popupOpen())
	}
	if h.a.flow.menuPage != gameMenuRoot || h.a.flow.menuList == oldRoot {
		t.Fatal("the campaign root was not centrally rebuilt")
	}
	if rows := h.a.flow.menuRows(); rows[h.a.flow.menuList.Selection()].Action != gameMenuLoad {
		t.Fatal("the rebuilt campaign root did not retain the Load selection")
	}
	wantEnabled := map[gameMenuAction]bool{
		gameMenuSave: false, gameMenuLoad: false, gameMenuGameOptions: true,
		gameMenuSoundOptions: false, gameMenuQuestObjectives: true,
		gameMenuEndQuest: true, gameMenuReturn: true,
	}
	rows := h.a.flow.menuRows()
	if len(rows) != len(wantEnabled) {
		t.Fatalf("campaign root has %d rows, want %d", len(rows), len(wantEnabled))
	}
	for _, row := range rows {
		want, ok := wantEnabled[row.Action]
		if !ok || row.Enabled != want {
			t.Errorf("rebuilt row action %d enabled=%v, want known=%v enabled=%v", row.Action, row.Enabled, ok, want)
		}
	}

	// The first visible menu frame consumes the whole hidden span as stopped.
	worldBefore, animBefore = h.w.world, h.v.AnimationCounter()
	h.frame(haltNeutral())
	if got := h.w.world - worldBefore; got != 0 {
		t.Errorf("first returned menu frame repaid %d world ticks, want 0", got)
	}
	if got := h.v.AnimationCounter() - animBefore; got != 0 {
		t.Errorf("first returned menu frame repaid %d ambient ticks, want 0", got)
	}
	if cadence, ok := h.w.lastCadence(); !ok || !cadence.stopped {
		t.Errorf("returned menu cadence = %+v ok=%v, want stopped", cadence, ok)
	}
}

func TestTownLoadReturnRebuildsTheTownRoot(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "town.ags", Label: "town"}} }
	f.loadGame = func(string) (MapOpener, bool, error) { return nil, false, errNotASave }
	f.openGameMenu(ScreenTown)
	oldRoot := f.menuList
	selectGameMenuAction(f, gameMenuLoad)
	f.chooseGameMenu()
	f.chooseLoad()
	f.closeLoad()
	if f.screen != ScreenGameMenu || f.menuPage != gameMenuRoot || f.menuList == oldRoot {
		t.Fatalf("town return = screen %v page %v rebuilt=%v", f.screen, f.menuPage, f.menuList != oldRoot)
	}
	if rows := f.menuRows(); rows[f.menuList.Selection()].Action != gameMenuLoad {
		t.Fatal("the rebuilt town root did not retain the Load selection")
	}
	want := []gameMenuAction{gameMenuSave, gameMenuLoad, gameMenuGameOptions, gameMenuSoundOptions, gameMenuAbortGame, gameMenuReturn}
	rows := f.menuRows()
	if len(rows) != len(want) {
		t.Fatalf("town root rows = %d, want %d", len(rows), len(want))
	}
	for i, action := range want {
		if rows[i].Action != action || !rows[i].Enabled {
			t.Errorf("town row %d = action %d enabled %v, want action %d enabled", i, rows[i].Action, rows[i].Enabled, action)
		}
	}
}

func TestLoadReturnControlsRemainUnchanged(t *testing.T) {
	t.Run("main menu escape", func(t *testing.T) {
		f := newFlow(NewPicker(nil), nil)
		f.openLoad(ScreenMenu)
		f.closeLoad()
		if f.screen != ScreenMenu || f.menuList != nil || f.menuPage != gameMenuRoot {
			t.Fatalf("main-menu return changed GameMenu state: screen=%v list=%v page=%v", f.screen, f.menuList, f.menuPage)
		}
	})
	t.Run("successful town load", func(t *testing.T) {
		f := newFlow(NewPicker(nil), nil)
		f.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
		f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "town.ags", Label: "town"}} }
		f.loadGame = func(string) (MapOpener, bool, error) { return nil, true, nil }
		f.openLoad(ScreenGameMenu)
		f.chooseLoad()
		if f.screen != ScreenTown {
			t.Fatalf("successful load landed on %v, want ScreenTown", f.screen)
		}
	})
}
