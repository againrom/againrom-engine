package ui

import (
	"strings"
	"testing"
)

func TestTriggerWizardExplainsKeptRecordsOnTheirRow(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := triggerLayoutEditor(t, lang)
		s, _, ok := e.objectSpec()
		if !ok || s.TitleHelp != e.word("editor.wizard.help") {
			t.Fatalf("%s: wizard has no title help", lang)
		}
		help := e.word("editor.wizard.otherhint")
		l := layoutPopup(s, e.font, e.bodyFont())
		kept := 0
		for _, rr := range l.Rows {
			row := s.Rows[rr.Index]
			if strings.Contains(row.Label, help) {
				t.Fatalf("%s: explanation is a body row", lang)
			}
			if row.Help == help {
				tip, ok := popupTipAt(s.Bounds, e.font, e.bodyFont(), s, l, center(rr.Help), true)
				if row.Kind != popupChoice || !ok || !tip.Help || tip.Text != help {
					t.Fatalf("%s: kept row has no help tip: %+v", lang, row)
				}
				kept++
			}
		}
		if kept != 2 {
			t.Fatalf("%s: %d kept rows with help, want 2", lang, kept)
		}
	}
}
