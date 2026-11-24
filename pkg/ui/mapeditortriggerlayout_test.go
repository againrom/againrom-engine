package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/scriptprofile"
	"againrom/pkg/words"
)

func triggerLayoutEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	e, _ := wizardChoiceEditor(t)
	e.SetWords(words.For(lang))
	e.SetBodyFont(wordsFont(lang))
	e.Layout(1920, 1080)
	e.obj.doc = nil
	return e
}

func TestTriggerAddRowsKeepOnlyButtonsAndHelp(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := triggerLayoutEditor(t, lang)
		for _, details := range []bool{false, true} {
			e.ShowTriggerDetails(details)
			s, _, ok := e.objectSpec()
			if !ok {
				t.Fatal("trigger window is closed")
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			for _, act := range []string{actAddPair, actAddAct} {
				found := false
				for _, rr := range l.Rows {
					k := slices.Index(e.obj.rowActs[rr.Index], act)
					if k < 0 {
						continue
					}
					row := s.Rows[rr.Index]
					if row.Kind != popupText || row.Label != "" || row.Value != "" || !rr.Control.Empty() ||
						rr.Help.Empty() || k >= len(rr.Buttons) || !rr.Help.Intersect(rr.Buttons[k]).Empty() {
						t.Fatalf("%s details %t %s: add row has text, missing help or missing button: %+v", lang, details, act, rr)
					}
					found = true
				}
				if !found {
					t.Fatalf("%s details %t: no %s button", lang, details, act)
				}
			}
		}
	}
}

type triggerMultiplayerHost struct{ *wizardWrites }

func (triggerMultiplayerHost) MapFacts() InspectionMapFacts { return InspectionMapFacts{Mode: 2} }

func TestTriggerWarningsStayBelowThePagedBody(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := triggerLayoutEditor(t, lang)
		e.doc.Edits = triggerMultiplayerHost{e.doc.Edits.(*wizardWrites)}
		e.loadObject()
		v, _ := e.scriptPopup()
		item := slices.IndexFunc(wizardTables[scriptprofile.OpsFirst][0], func(w wizItem) bool { return w.labelWord == "editor.wizard.a.giveunit" })
		values := e.wizardDefaults(false, item)
		values[1] = "10001"
		e.obj.wiz.rows = slices.Repeat([]wizRow{{item: item, origItem: -2, vals: values}}, 30)
		e.wizardFields(v)
		for _, top := range []int{0, 18} {
			e.objectTop = top
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			state := strings.Join(strings.Fields(strings.Join(s.Notes, " ")), " ")
			for _, key := range []string{"editor.wizard.nocondition", "editor.wizard.multiplayer"} {
				if !strings.Contains(state, e.word(key)) {
					t.Fatalf("%s top %d: notes omit %s: %q", lang, top, key, state)
				}
			}
			for _, r := range l.Notes {
				if r.Min.Y < l.RowsViewport.Max.Y || !r.In(l.Rect) {
					t.Fatalf("note %v leaves its place below body %v in %v", r, l.RowsViewport, l.Rect)
				}
			}
			if l.RowBar.Empty() || len(l.Notes) == 0 || s.Top != top {
				t.Fatalf("top %d: body or notes not shown", top)
			}
		}
	}
}

func TestTriggerTextRowButtonsUseFooterMinimum(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := triggerLayoutEditor(t, lang)
		s, _, _ := e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		count := 0
		for _, row := range l.Rows {
			for _, b := range row.Buttons {
				if b.Dx() < popupButtonW || !b.In(l.Rect) {
					t.Fatalf("%s: text button %v in %v needs %d px", lang, b, l.Rect, popupButtonW)
				}
				count++
			}
		}
		if count == 0 || !l.Rect.In(image.Rect(0, 0, 1920, 1080)) {
			t.Fatalf("%s: %d buttons in %v", lang, count, l.Rect)
		}
	}
}
