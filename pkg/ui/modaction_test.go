package ui

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func modActionScreens() []ModScreen {
	return []ModScreen{
		{Mod: "m", Key: "leave", Kind: ModActionAbandon, Title: "Leave it", MenuLabel: "Abandon mission", Game: true},
		{Mod: "m", Key: "again", Kind: ModActionRestart, Title: "Again", MenuLabel: "Restart mission", Game: true},
	}
}

// actionApp opens a mission whose menu context says leave and whose advance seam
// records each action and answers it with dest.
func actionApp(t *testing.T, leave bool, dest NoticeDest, asked *[]NoticeAction, reopened *int) *App {
	t.Helper()
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.SetModScreens(modActionScreens()); err != nil {
		t.Fatal(err)
	}
	build := func() (*Viewer, error) {
		v, err := NewViewer("m", grid(60, 60), &terrain.Tileset{})
		if err == nil {
			v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{Campaign: true, LeaveToTown: leave} })
		}
		return v, err
	}
	var open MapOpener
	open = func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := build()
		advance := func(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
			*asked = append(*asked, actions...)
			if dest == NoticeToMission {
				return dest, "", func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
					*reopened++
					v, err := build()
					return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
				}
			}
			return dest, "back in town", nil
		}
		return v, nil, nil, nil, nil, advance, nil, nil, nil, nil, err
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestModActionRowsAppearOnlyOverAMissionThatCanBeLeft(t *testing.T) {
	var asked []NoticeAction
	var again int
	a := actionApp(t, true, NoticeToTown, &asked, &again)
	a.step(appInput{Escape: true}, atAt)
	rows := a.flow.menuRows()
	base := gameMenuBaseRows(gameMenuMission)
	if len(rows) != base+2 || rows[base].Action != gameMenuModAction || rows[base].Label != "Abandon mission" ||
		rows[base+1].Action != gameMenuModAction || rows[base+1].Label != "Restart mission" || !rows[base].Enabled || !rows[base].Literal {
		t.Fatalf("rows %+v", rows[base:])
	}
	// The town menu has none of them.
	town := newTestApp(t, appRows(1), okLoader(t))
	if err := town.SetModScreens(modActionScreens()); err != nil {
		t.Fatal(err)
	}
	town.flow.openGameMenu(ScreenTown)
	if got := town.flow.menuRows(); len(got) != gameMenuBaseRows(gameMenuTown) {
		t.Fatalf("the town menu has %d rows", len(got))
	}
	// Loss control: a mission that cannot be left shows none either.
	b := actionApp(t, false, NoticeToTown, &asked, &again)
	b.step(appInput{Escape: true}, atAt)
	if got := b.flow.menuRows(); len(got) != base {
		t.Fatalf("a mission with LeaveToTown false has %d rows", len(got))
	}
}

func TestModActionAsksBeforeItActs(t *testing.T) {
	var asked []NoticeAction
	var again int
	a := actionApp(t, true, NoticeToTown, &asked, &again)
	a.step(appInput{Escape: true}, atAt)
	base := gameMenuBaseRows(gameMenuMission)
	a.flow.menuList.Select(base)
	a.chooseGameMenu()
	if a.flow.menuPage != gameMenuModActionConfirmation || len(asked) != 0 || a.Screen() != ScreenGameMenu {
		t.Fatalf("page %v, asked %v, screen %v", a.flow.menuPage, asked, a.Screen())
	}
	rows := a.flow.menuRows()
	if len(rows) != 2 || rows[0].Label != "Leave it" || rows[0].Action != gameMenuModActionConfirm || rows[1].Action != gameMenuPageReturn {
		t.Fatalf("confirm rows %+v", rows)
	}
	if a.flow.menuList.Selection() != 1 {
		t.Fatalf("the confirming page starts on row %d, not on Return", a.flow.menuList.Selection())
	}
	// Return, and Escape, go back to the root with nothing asked.
	a.flow.menuList.Select(1)
	a.chooseGameMenu()
	if a.flow.menuPage != gameMenuRoot || len(asked) != 0 {
		t.Fatalf("return: page %v asked %v", a.flow.menuPage, asked)
	}
	a.flow.menuList.Select(base)
	a.chooseGameMenu()
	a.step(appInput{Escape: true}, atAt)
	if a.flow.menuPage != gameMenuRoot || len(asked) != 0 || a.Screen() != ScreenGameMenu {
		t.Fatalf("escape: page %v asked %v screen %v", a.flow.menuPage, asked, a.Screen())
	}
}

func TestModActionAbandonLeavesTheMapForTheTown(t *testing.T) {
	var asked []NoticeAction
	var again int
	a := actionApp(t, true, NoticeToTown, &asked, &again)
	a.step(appInput{Escape: true}, atAt)
	a.flow.menuList.Select(gameMenuBaseRows(gameMenuMission))
	a.chooseGameMenu()
	a.flow.menuList.Select(0)
	a.chooseGameMenu()
	if len(asked) != 1 || asked[0] != NoticeAbandon {
		t.Fatalf("asked %v", asked)
	}
	// This front end has no town, so the answer lands on the map list with the
	// seam's own sentence, as the completion route does.
	if a.Screen() != ScreenPicker || a.flow.viewer != nil || a.flow.msg != "back in town" || a.flow.completedCutscene != "" {
		t.Fatalf("screen %v viewer %v msg %q cutscene %q", a.Screen(), a.flow.viewer, a.flow.msg, a.flow.completedCutscene)
	}
	if a.flow.menuPage != gameMenuRoot || a.flow.menuList != nil || a.flow.modUI.pending != 0 {
		t.Fatalf("the menu was left open: page %v pending %d", a.flow.menuPage, a.flow.modUI.pending)
	}
}

func TestModActionRestartReopensTheMission(t *testing.T) {
	var asked []NoticeAction
	var again int
	a := actionApp(t, true, NoticeToMission, &asked, &again)
	first := a.flow.viewer
	a.step(appInput{Escape: true}, atAt)
	a.flow.menuList.Select(gameMenuBaseRows(gameMenuMission) + 1)
	a.chooseGameMenu()
	a.flow.menuList.Select(0)
	a.chooseGameMenu()
	if len(asked) != 1 || asked[0] != NoticeRestart || again != 1 {
		t.Fatalf("asked %v reopened %d", asked, again)
	}
	if a.Screen() != ScreenMap || a.flow.viewer == nil || a.flow.viewer == first || a.flow.menuList != nil {
		t.Fatalf("screen %v, same viewer %v", a.Screen(), a.flow.viewer == first)
	}
}

func TestModActionStaysWhenTheSeamRefuses(t *testing.T) {
	var asked []NoticeAction
	var again int
	a := actionApp(t, true, NoticeStay, &asked, &again)
	a.step(appInput{Escape: true}, atAt)
	a.flow.menuList.Select(gameMenuBaseRows(gameMenuMission))
	a.chooseGameMenu()
	a.flow.menuList.Select(0)
	a.chooseGameMenu()
	if len(asked) != 1 || a.Screen() != ScreenGameMenu || a.flow.menuPage != gameMenuRoot || a.flow.viewer == nil {
		t.Fatalf("asked %v screen %v page %v", asked, a.Screen(), a.flow.menuPage)
	}
}

func TestSetModScreensRefusesAnActionOutsideTheGameMenu(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	bad := ModScreen{Mod: "m", Key: "k", Kind: ModActionAbandon, Title: "T", MenuLabel: "L", Main: true}
	if err := a.SetModScreens([]ModScreen{bad}); err == nil {
		t.Fatal("an action in the main menu was accepted")
	}
	bad.Main, bad.Game = false, false
	if err := a.SetModScreens([]ModScreen{bad}); err == nil {
		t.Fatal("an action in no menu was accepted")
	}
}

func TestLongModRowLabelIsClippedToItsRow(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	long := "An abandon label that is far too long to fit its menu row at all"
	if err := a.SetModScreens([]ModScreen{{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", MenuLabel: long, Game: true, Paragraphs: []string{"p"}}}); err != nil {
		t.Fatal(err)
	}
	a.flow.openGameMenu(ScreenTown)
	rows := a.flow.menuRows()
	got := rows[len(rows)-1].Label
	if len(got) >= len(long) || got[len(got)-len(clipMark):] != clipMark {
		t.Fatalf("label %q", got)
	}
}
