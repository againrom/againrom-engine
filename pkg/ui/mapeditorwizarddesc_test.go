package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/textfit"
	"againrom/pkg/words"
)

func TestTriggerWindowsHoldTheLayoutProperties(t *testing.T) {
	size := image.Pt(1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := triggerLayoutEditor(t, lang)
			trigger := &e.doc.Script.Triggers[0]
			trigger.Left = slices.Repeat([]uint32{1}, 24)
			trigger.Right = slices.Repeat([]uint32{2}, 24)
			trigger.Cmp = make([]uint32, 24)
			trigger.Acts = slices.Repeat([]uint32{11}, 18)
			for _, details := range []bool{false, true} {
				e.ShowTriggerDetails(details)
				e.objectTop = 0
				s, _, _ := e.objectSpec()
				first := layoutPopup(s, e.font, e.bodyFont())
				if !s.Fit || first.TitleHelp.Empty() || first.RowBar.Empty() {
					t.Fatalf("details %t: missing fitted body, title help or scroll bar", details)
				}
				checkWindowProperties(t, e, s, size)

				e.ScrollDetails(100)
				s, _, _ = e.objectSpec()
				last := layoutPopup(s, e.font, e.bodyFont())
				if len(last.Rows) == 0 || last.Rows[0].Index <= first.Rows[0].Index ||
					last.Rows[len(last.Rows)-1].Index != len(s.Rows)-1 ||
					first.Title != last.Title || !slices.Equal(first.Buttons, last.Buttons) {
					t.Fatalf("details %t: scrolling did not move only the body to the last row", details)
				}
				checkWindowProperties(t, e, s, size)
				pressed := false
				for _, rr := range last.Rows {
					k := slices.Index(e.obj.rowActs[rr.Index], actAddAct)
					if k < 0 {
						continue
					}
					at := center(rr.Buttons[k])
					if part, i := last.hit(at); part != popupRowButton || i != rr.Index {
						t.Fatalf("details %t: hit at Add action gives %d row %d", details, part, i)
					}
					before, _ := e.WizardRowCounts()
					e.objectPress(at)
					after, _ := e.WizardRowCounts()
					if details && !e.obj.shownAction(18) || !details && after != before+1 {
						t.Fatalf("details %t: Add action hit failed to add a slot", details)
					}
					pressed = true
					break
				}
				if !pressed {
					t.Fatalf("details %t: the last page does not show Add action", details)
				}
			}
		})
	}
}

func TestTriggerWizardClippedChoiceShowsItsWholeValue(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := paramPickEditorWith(t, func(doc *InspectionDocument) {
				doc.Script.Targets[scriptUnit][0].Choice = strings.TrimSpace(strings.Repeat("Watch by the east gate ", 30))
			})
			e.SetWords(words.For(lang))
			e.SetBodyFont(wordsFont(lang))
			e.Layout(1920, 1080)
			e.Select(0, false)
			wizardPress(t, e, -1, actAddPair)
			wizardChoose(t, e, "c0.p0", 101)
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			for _, rr := range l.Rows {
				if f, ok := e.obj.rowField[rr.Index]; !ok || f != e.WizardField("c0.p0") {
					continue
				}
				value := s.Rows[rr.Index].Value
				if clipped := editorClip(e.bodyFont(), value, choiceTextRoom(rr.Control)); !strings.HasSuffix(clipped, textfit.Mark) {
					t.Fatalf("choice is not shown with the cut mark: %q", clipped)
				}
				bounds := image.Rect(0, 0, 1920, 1080)
				tip, ok := popupTipAt(bounds, e.font, e.bodyFont(), s, l, center(rr.Control), true)
				if !ok || !strings.HasSuffix(tip.Text, value) || !tip.Rect.In(bounds) || tip.Rect.Dx() != popupTipW {
					t.Fatalf("cut choice has no whole, bounded tip: %v %q", tip.Rect, tip.Text)
				}
				return
			}
			t.Fatal("unit choice is missing")
		})
	}
}
