package ui

import (
	"testing"
	"time"

	"againrom/pkg/render/menu"
)

// Both games bind button 2 to the character list; the game's data decides
// the title, Ok and whether Delete shows. A hidden Delete takes no press.
func TestCharacterPageIsTheGamesData(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for name, a := range multiplayerApps(t) {
		if got := a.assets.Binding(2); got != (menu.Button{Action: menu.ActionMultiplayer, Name: "multi player"}) {
			t.Fatalf("%s: button 2 binds %+v", name, got)
		}
		if got := a.assets.Binding(6); got.Action != menu.ActionNone {
			t.Fatalf("%s: Server binds %+v, want no action", name, got)
		}
		load := a.flow.loadPage()
		for _, del := range []bool{true, false} {
			a.SetCharacterPage(CharacterPage{Title: "T", OK: "O", Delete: del})
			if err := a.HeadlessActivate("multi player"); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			v := a.flow.loadView()
			if v.page.Texts[0].Text != "T" || v.page.Buttons[0].Caption != "O" {
				t.Errorf("%s: page title %q Ok %q", name, v.page.Texts[0].Text, v.page.Buttons[0].Caption)
			}
			if del {
				if v.page.Buttons[1] != load.Buttons[1] || !a.flow.pageButtonDisabled(v, loadDeleteButton) {
					t.Errorf("%s: Delete %+v, want the load page's Delete, disabled on the New row", name, v.page.Buttons[1])
				}
			} else {
				r := load.Buttons[1].Rect
				if v.page.Buttons[1] != (PageButton{}) {
					t.Errorf("%s: Delete %+v, want none", name, v.page.Buttons[1])
				}
				if _, on := v.buttonAt(r.Min.Add(r.Size().Div(2))); on {
					t.Errorf("%s: the hidden Delete's place takes a press", name)
				}
			}
			if a.flow.pageButtonDisabled(v, loadOKButton) {
				t.Errorf("%s: Ok disabled on the character list", name)
			}
			clickPageButton(t, a, loadCancelButton, now)
			if a.Screen() != ScreenMenu || a.CharacterList() || a.flow.loadUI.characters {
				t.Fatalf("%s: Cancel left %v", name, a.Screen())
			}
		}
	}
}
