package ui

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// gameMenuBaseRows is how many rows the first game's root list of a surface
// has.
func gameMenuBaseRows(s gameMenuSurface) int {
	if s == gameMenuTown {
		return 6
	}
	return 7
}

// popupRows is the second game's town popup (R2-ENGINE-441), written apart
// from the data.
func popupRows(w Words, s gameMenuRowState) []gameMenuRow {
	return []gameMenuRow{
		{Label: w.MenuSave, Fallback: 'S', FixedKey: true, Action: gameMenuSave, Enabled: townCanSave(s.town)},
		{Label: w.MenuLoad, Fallback: 'L', FixedKey: true, Action: gameMenuLoad, Enabled: true},
		{Label: w.MenuSoundOptions, Fallback: 'N', FixedKey: true, Action: gameMenuSoundOptions, Enabled: true},
		{Label: w.MenuAbort, Fallback: 'E', FixedKey: true, Action: gameMenuAbortGame, Enabled: true},
		{Label: w.MenuReturn, Fallback: 'R', FixedKey: true, Action: gameMenuReturn, Enabled: true},
	}
}

// russianWords carries the RU install's five popup rows in CP866: "~Сохранить
// игру", "~Восстановить игру", "~Звуковые опции", "~Выход из игры" and
// "Вер~нуться к игре". Load and Abort both mark 0x82.
func russianWords() Words {
	w := installedWords()
	w.MenuSave = "~\x91\xae\xe5\xe0\xa0\xad\xa8\xe2\xec \xa8\xa3\xe0\xe3"
	w.MenuLoad = "~\x82\xae\xe1\xe1\xe2\xa0\xad\xae\xa2\xa8\xe2\xec \xa8\xa3\xe0\xe3"
	w.MenuSoundOptions = "~\x87\xa2\xe3\xaa\xae\xa2\xeb\xa5 \xae\xaf\xe6\xa8\xa8"
	w.MenuAbort = "~\x82\xeb\xe5\xae\xa4 \xa8\xa7 \xa8\xa3\xe0\xeb"
	w.MenuReturn = "\x82\xa5\xe0~\xad\xe3\xe2\xec\xe1\xef \xaa \xa8\xa3\xe0\xa5"
	return w
}

// On RU data the popup keeps the keys S, L, N, E and R of R2-ENGINE-441: the
// labels' `~` letters do not replace them, so Abort is not lost behind Load.
func TestSecondGameRussianPopupKeysAreTheClaimsLetters(t *testing.T) {
	w := russianWords()
	f := secondGameTownMenu(t, w)
	f.menuFont = &text.Font{Selector: text.SelectorConverting}
	f.encodeMenuKey = func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) }
	var got []string
	for _, r := range f.menuRows() {
		key, _ := r.accelerator(f.menuSelector())
		got = append(got, fmt.Sprintf("%q/%c", r.Label, key))
	}
	want := fmt.Sprint([]string{fmt.Sprintf("%q/s", w.MenuSave), fmt.Sprintf("%q/l", w.MenuLoad),
		fmt.Sprintf("%q/n", w.MenuSoundOptions), fmt.Sprintf("%q/e", w.MenuAbort), fmt.Sprintf("%q/r", w.MenuReturn)})
	if fmt.Sprint(got) != want {
		t.Fatalf("RU rows %v\nwant %v", got, want)
	}
	if !f.chooseGameMenuAccelerator('e') || f.menuPage != gameMenuAbortGameConfirmation {
		t.Fatalf("e reached page %v, want the abort confirmation", f.menuPage)
	}
	f = secondGameTownMenu(t, w)
	f.menuFont = &text.Font{Selector: text.SelectorConverting}
	f.encodeMenuKey = func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) }
	if f.chooseGameMenuAccelerator('в') || f.screen != ScreenGameMenu {
		t.Fatalf("в chose a row; screen %v", f.screen)
	}
}

func secondGameTownMenu(t *testing.T, w Words) *flow {
	t.Helper()
	f := openTownMenu(t)
	f.words = w
	f.pages.menu = gameMenuProfiles[1].profile
	f.openGameMenu(ScreenTown)
	return f
}

// markedWords gives every menu word its own text, so a row reading the wrong
// word shows it.
func markedWords() Words {
	w := installedWords()
	w.MenuGameOptions, w.MenuQuestObjectives, w.MenuEndQuest, w.MenuDiplomacy = "game options", "objectives", "end quest", "diplomacy"
	return w
}

func TestSecondGameTownMenuIsThePopupsFiveButtons(t *testing.T) {
	for _, w := range []Words{installedWords(), markedWords(), AuthoredWords(), russianWords()} {
		f := secondGameTownMenu(t, w)
		var got []string
		for i, r := range f.menuRows() {
			key, _ := r.accelerator(0)
			got = append(got, fmt.Sprintf("%s/%c/%d", r.text(), key, r.Action))
			if !r.Enabled {
				t.Fatalf("row %d disabled", i)
			}
		}
		want := fmt.Sprint([]string{
			gameMenuLabelText(w.MenuSave) + "/s/" + fmt.Sprint(int(gameMenuSave)),
			gameMenuLabelText(w.MenuLoad) + "/l/" + fmt.Sprint(int(gameMenuLoad)),
			gameMenuLabelText(w.MenuSoundOptions) + "/n/" + fmt.Sprint(int(gameMenuSoundOptions)),
			gameMenuLabelText(w.MenuAbort) + "/e/" + fmt.Sprint(int(gameMenuAbortGame)),
			gameMenuLabelText(w.MenuReturn) + "/r/" + fmt.Sprint(int(gameMenuReturn)),
		})
		if fmt.Sprint(got) != want {
			t.Fatalf("rows %v\nwant %v", got, want)
		}
	}
	w := installedWords()
	f := secondGameTownMenu(t, w)
	var labels []string
	for _, r := range f.menuRows() {
		labels = append(labels, r.text())
	}
	if fmt.Sprint(labels) != "[Save Game Load Game Sound Options Abort Game Return to Game]" {
		t.Fatalf("English labels %v", labels)
	}
}

func TestSecondGameTownMenuRowsPostTheirActions(t *testing.T) {
	for i, check := range []func(*flow, int) error{
		func(f *flow, saved int) error {
			if saved != 1 || f.screen != ScreenGameMenu {
				return fmt.Errorf("save ran %d times, screen %v", saved, f.screen)
			}
			return nil
		},
		func(f *flow, _ int) error {
			if f.screen != ScreenLoad || f.loadBack != ScreenGameMenu {
				return fmt.Errorf("screen %v, want the load page", f.screen)
			}
			return nil
		},
		func(f *flow, _ int) error {
			if f.screen != ScreenGameMenu || f.menuPage != gameMenuSoundOptionsPage {
				return fmt.Errorf("page %v, want sound options", f.menuPage)
			}
			return nil
		},
		func(f *flow, _ int) error {
			if f.menuPage != gameMenuAbortGameConfirmation || f.menuRows()[f.menuList.Selection()].Action != gameMenuPageReturn {
				return fmt.Errorf("page %v, want the abort confirmation on its return row", f.menuPage)
			}
			return nil
		},
		func(f *flow, _ int) error {
			if f.screen != ScreenTown || f.menuList != nil {
				return fmt.Errorf("screen %v, want the square with the menu closed", f.screen)
			}
			return nil
		},
	} {
		f := secondGameTownMenu(t, installedWords())
		saved := 0
		f.saveGame = func(bool) (string, error) { saved++; return "s.ags", nil }
		f.menuList.Select(i)
		f.chooseGameMenu()
		if err := check(f, saved); err != nil {
			t.Errorf("row %d: %v", i, err)
		}
	}
}

func TestSecondGameTownPopupPanelAndRows(t *testing.T) {
	f := secondGameTownMenu(t, installedWords())
	panel := gameMenuPanelRectFor(gameMenuTown, f.menuRows(), f.menuGeometryProfile())
	if panel != image.Rect(100, 100, 540, 440) {
		t.Fatalf("popup panel %v", panel)
	}
	for n := range f.menuRows() {
		r := gameMenuRowRect(panel, n)
		if got, ok := gameMenuRowAt(panel, len(f.menuRows()), r.Min.Add(image.Pt(5, 5))); !ok || got != n {
			t.Fatalf("row %d at %v hits %d %v", n, r, got, ok)
		}
	}
	first := openTownMenu(t)
	if got := gameMenuPanelRectFor(gameMenuTown, first.menuRows(), first.menuGeometryProfile()); got != image.Rect(100, 100, 440, 340) {
		t.Fatalf("first game town panel %v", got)
	}
	mission := openMissionMenu(t)
	mission.pages.menu = gameMenuProfiles[1].profile
	mission.rebuildGameMenu(gameMenuRoot, 0)
	if rows := mission.menuRows(); len(rows) != gameMenuBaseRows(gameMenuMission) || rows[len(rows)-1].Action != gameMenuReturn {
		t.Fatalf("second game mission rows %v", rows)
	}
}
