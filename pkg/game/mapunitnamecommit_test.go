package game

import (
	"fmt"
	"testing"

	"againrom/pkg/ui"
)

// A typed unit name is written by Enter and by Save, in both views.
func TestEditorUnitNameCommitsByEnterAndSave(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, plain := range []bool{true, false} {
			for _, commit := range []string{"enter", "saveas"} {
				t.Run(fmt.Sprintf("%s/plain=%v/%s", lang, plain, commit), func(t *testing.T) {
					s, _ := viewsMap(t)
					s.inspector.archives.Base.Profile.Language = lang
					s.inspector.recordOptions = nil
					id, err := unitID(s.model, 0)
					if err != nil || id == 0 {
						t.Fatalf("the fixture unit has id %d: %v", id, err)
					}
					name := map[string]string{"": "Bjorn the Red", "russian": "Бьорн Рыжий"}[lang]
					e, c := motionWindow(t, s, lang, plain, false)
					if !showField(t, e, c, unitNameField) {
						t.Fatal("the name does not show")
					}
					r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: fieldIndex(t, e.ObjectFields(), unitNameField)})
					c.click(e, r, ok, "name")
					e.TextInput(name)
					if commit == "enter" {
						_ = e.Key("enter")
					} else {
						n, ok := e.MenuRow("file", "saveas")
						if !ok {
							t.Fatal("the Map menu has no Save row")
						}
						r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlBar, Key: "file"})
						c.click(e, r, ok, "the Map entry")
						r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlMenuRow, N: n})
						c.click(e, r, ok, "the Save row")
					}
					if got := s.authoredUnitNames()[id]; got != name {
						t.Fatalf("after %s the map's names page gives unit %d %q; want %q", commit, id, got, name)
					}
				})
			}
		}
	}
}
