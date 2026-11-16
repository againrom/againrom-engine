package ui

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/words"
)

type wizardFill func(scriptView, *popupSpec, func(popupRow, int, ...string), func(string, string))

// wizardWindowWith is scriptSpec's wizard window with fill building its
// rows: the same base, add and text, and the same row captions.
func wizardWindowWith(e *MapEditor, v scriptView, fill wizardFill) (popupSpec, map[int]int, map[int][]string) {
	o := &e.obj
	s := popupSpec{At: e.keptCorner(popObject), Width: e.popupWide(popObject, scriptPopupWidth), Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 150}
	fields, acts := map[int]int{}, map[int][]string{}
	width := e.popupWide(popObject, scriptPopupWidth) - 2*popupPad - 8
	add := func(row popupRow, field int, a ...string) {
		if field >= 0 {
			fields[len(s.Rows)] = field
			row.Focus = field == o.focus
			if row.Kind == popupEdit && row.Focus {
				if b := e.bodyFont(); b != nil {
					row.Caret = b.Advance(row.Value)
				}
			}
		}
		if len(a) > 0 {
			acts[len(s.Rows)] = a
		}
		s.Rows = append(s.Rows, row)
	}
	text := func(line string, ink string) {
		body := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(body)]
		w := width
		if b := e.bodyFont(); b != nil {
			w -= b.Advance(indent)
		}
		for _, part := range editorTextLines(e.bodyFont(), body, w) {
			add(popupRow{Kind: popupText, Label: indent + part, Ink: inspectionRoleColor(ink)}, -1)
		}
	}
	fill(v, &s, add, text)
	if o.note != "" {
		s.Notes = append(s.Notes, editorTextLines(e.bodyFont(), e.editorHostText(o.note), width)...)
	}
	for r, a := range acts {
		for _, act := range a {
			s.Rows[r].Buttons = append(s.Rows[r].Buttons, e.actWord(act))
		}
	}
	s.MaxRows = e.objectRowsShown(true)
	s.Top = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	return s, fields, acts
}

// sameWizard checks that the wizard the editor shows now is the one its
// description builds and that it equals the wizard built in code before:
// rows, kinds, labels, values, inks, buttons, fields, actions and every
// rectangle layoutPopup gives.
func sameWizard(t *testing.T, e *MapEditor, state string) {
	t.Helper()
	v, ok := e.scriptPopup()
	if !ok || e.obj.wiz == nil {
		t.Fatalf("%s: no wizard open", state)
	}
	shown, _ := e.scriptSpec(v)
	after, afterFields, afterActs := wizardWindowWith(e, v, e.wizardSpec)
	before, beforeFields, beforeActs := wizardWindowWith(e, v, e.wizardSpecBefore)
	if !reflect.DeepEqual(shown.Rows, after.Rows) || shown.Title != after.Title || !reflect.DeepEqual(shown.Buttons, after.Buttons) ||
		!reflect.DeepEqual(shown.Disabled, after.Disabled) || shown.Width != after.Width {
		t.Fatalf("%s: the shown wizard is not the one the description builds", state)
	}
	if !reflect.DeepEqual(e.obj.rowField, afterFields) || !reflect.DeepEqual(e.obj.rowActs, afterActs) {
		t.Fatalf("%s: the shown wizard's fields or actions differ", state)
	}
	if len(before.Rows) != len(after.Rows) {
		t.Fatalf("%s: %d rows before, %d from the description", state, len(before.Rows), len(after.Rows))
	}
	for i := range before.Rows {
		if !reflect.DeepEqual(before.Rows[i], after.Rows[i]) {
			t.Errorf("%s: row %d before %+v\nfrom the description %+v", state, i, before.Rows[i], after.Rows[i])
		}
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeFields, afterFields) || !reflect.DeepEqual(beforeActs, afterActs) {
		t.Errorf("%s: the window, its fields or its actions differ:\n%+v\n%+v\n%v %v\n%v %v", state,
			before.Buttons, after.Buttons, beforeFields, afterFields, beforeActs, afterActs)
	}
	lb, la := layoutPopup(before, e.font, e.bodyFont()), layoutPopup(after, e.font, e.bodyFont())
	if lb.Rect != la.Rect || lb.Title != la.Title || !reflect.DeepEqual(lb.Rows, la.Rows) || !reflect.DeepEqual(lb.Buttons, la.Buttons) ||
		!reflect.DeepEqual(lb.Notes, la.Notes) || lb.RowsViewport != la.RowsViewport {
		t.Errorf("%s: the rectangles differ", state)
	}
}

// wizardItemStates walks every condition and every action item on the
// wizard's last condition and last action row, checking each.
func wizardItemStates(t *testing.T, e *MapEditor, lang string) int {
	t.Helper()
	n := 0
	for _, condition := range []bool{true, false} {
		w := e.obj.wiz
		r := -1
		for k, row := range w.rows {
			if row.condition == condition {
				r = k
			}
		}
		if r < 0 {
			t.Fatalf("%s: no row of condition %v", lang, condition)
		}
		key := w.wizardKey(r, "item")
		for _, c := range e.obj.fields[e.WizardField(key)].Choices {
			wizardChoose(t, e, key, c.Value)
			sameWizard(t, e, lang+" "+key+" "+c.Label)
			n++
		}
	}
	return n
}

// The trigger wizard built from its description is the wizard built in
// code before, row for row and rectangle for rectangle, in English and
// Russian: on a trigger with ready and kept rows, after adding rows and an
// OR group, and with every condition and action item chosen.
func TestTriggerWizardFromItsDescriptionIsTheWizardBefore(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	sameWizard(t, e, "ready and kept rows")
	wizardPress(t, e, -1, actAddGroup)
	wizardPress(t, e, -1, actAddPair)
	sameWizard(t, e, "an OR group")
	for _, lang := range []string{"en", "ru"} {
		e, _ := paramPickEditor(t)
		e.SetWords(words.For(lang))
		e.SetBodyFont(wordsFont(lang))
		e.Select(0, false)
		if !e.WizardShown() {
			t.Fatalf("%s: the trigger opens no wizard", lang)
		}
		sameWizard(t, e, lang+" kept condition")
		wizardPress(t, e, -1, actAddPair)
		wizardPress(t, e, -1, actAddAct)
		sameWizard(t, e, lang+" added rows")
		if n := wizardItemStates(t, e, lang); n < 20 {
			t.Errorf("%s: %d item states", lang, n)
		}
	}
}

// Each row button of the description is captioned with the word the
// trigger window gives its action, and the description's width and label
// width are the trigger window's.
func TestTriggerWizardDescriptionButtonsAreTheWindowsActions(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	d := triggerWizardDesc()
	if d.Width != scriptPopupWidth || d.LabelWidth != 150 {
		t.Errorf("description width %d, label width %d", d.Width, d.LabelWidth)
	}
	n := 0
	var rows func([]popupDescRow)
	rows = func(list []popupDescRow) {
		for _, r := range list {
			for _, b := range r.Buttons {
				n++
				if e.word(b.Word) != e.actWord(b.Act) {
					t.Errorf("button %q for %q: the window captions it %q", b.Word, b.Act, e.actWord(b.Act))
				}
			}
			rows(r.Rows)
		}
	}
	rows(d.Rows)
	if n == 0 {
		t.Fatal("no row button")
	}
}
