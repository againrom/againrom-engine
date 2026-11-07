package game

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// A name typed one character at a time, as the window delivers keys, keeps
// the space inside it through the field, the names page and the game's name
// lookup; the ends of the name are still trimmed.
func TestEditorUnitNameKeepsASpaceTypedKeyByKey(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, docked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/docked=%v", lang, docked), func(t *testing.T) {
				s, _ := viewsMap(t)
				s.inspector.archives.Base.Profile.Language = lang
				s.inspector.recordOptions = nil
				words := ui.AuthoredWords()
				s.inspector.panelWords, s.unitViews = &words, nil
				id, err := unitID(s.model, 0)
				if err != nil || id == 0 {
					t.Fatalf("the fixture unit has id %d: %v", id, err)
				}
				name := map[string]string{"": "Old Bob", "russian": "Старый Боб"}[lang]
				e, c := motionWindow(t, s, lang, false, docked)
				if !showField(t, e, c, unitNameField) {
					t.Fatal("the name does not show")
				}
				r, ok := e.FieldRect(fieldIndex(t, e.ObjectFields(), unitNameField))
				c.click(e, r, ok, unitNameField)
				// The window rebuilds the draft whenever a typed field rests;
				// the pause after each word is where a space used to go.
				for _, word := range strings.SplitAfter("  "+name+"  ", " ") {
					for _, ch := range word {
						e.TextInput(string(ch))
					}
					e.UnitCharacterSubject()
					time.Sleep(350 * time.Millisecond)
					e.UnitCharacterSubject()
				}
				r, ok = e.PopupButtonRect(0)
				c.click(e, r, ok, "Apply")
				want := fmt.Sprintf("%d=%s\r\n", id, name)
				if got := string(s.workspace.Pages[unitNamesPage]); got != want {
					t.Fatalf("page %q, want %q", got, want)
				}
				if got := unitNames(string(s.workspace.Pages[unitNamesPage]))[id]; got != name {
					t.Fatalf("the game's lookup reads %q, want %q", got, name)
				}
			})
		}
	}
}
