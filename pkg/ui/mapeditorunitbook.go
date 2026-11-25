package ui

import "strconv"

// unitBookView is the spell book window of a unit window: the unit's record
// key, its book field's key, and the depth of the back stack the window
// opened at. The window shows while the selection and the stack stand as
// it opened them; Back returns to the unit window.
type unitBookView struct {
	record uint64
	field  string
	depth  int
}

// openUnitBook opens the spell book window of field i of the shown unit
// window, once the window may leave.
func (e *MapEditor) openUnitBook(i int) {
	if e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) || !e.objectMayLeave() {
		return
	}
	e.objectBack = append(e.objectBack, e.objectPlace())
	e.unitBook = unitBookView{record: e.doc.Records[e.selected].Key, field: e.obj.fields[i].Key, depth: len(e.objectBack)}
	e.obj.open, e.obj.focus, e.obj.note = -1, -1, ""
}

// unitBookShown is the book field the spell book window shows; false when
// the window does not show.
func (e *MapEditor) unitBookShown() (InspectionField, bool) {
	b := e.unitBook
	if b.field == "" || e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) ||
		e.doc.Records[e.selected].Key != b.record || len(e.objectBack) != b.depth {
		return InspectionField{}, false
	}
	e.loadObject()
	for _, f := range e.obj.fields {
		if f.Key == b.field && f.Book != nil {
			return f, true
		}
	}
	return InspectionField{}, false
}

var unitBookDesc = embeddedPopupDesc("unitbook")

// unitBookSpec is the spell book window of book field f: the field's note,
// the book as the game draws it with each cell a switch, the refusal of the
// last press, and Back.
func (e *MapEditor) unitBookSpec(f InspectionField) (popupSpec, []int) {
	r := e.doc.Records[e.selected]
	width := bookBarW + 2*(bookGrow+bookGrowGap) + 2*popupPad
	sc := &popupScope{Lists: map[string][]popupScope{}}
	for list, text := range map[string]string{"lines": e.editorHostText(f.Meaning), "notes": e.obj.note} {
		for _, line := range editorTextLines(e.bodyFont(), text, width-2*popupPad) {
			if text == "" {
				break
			}
			sc.Lists[list] = append(sc.Lists[list], popupScope{Values: map[string]string{"line": line}})
		}
	}
	s, _ := unitBookDesc().build(sc, e.word)
	s.Title, s.At, s.Width, s.Bounds = e.recordTitle(r)+": "+e.editorHostText(f.Label), e.keptCorner(popObject), width, e.popupBounds()
	s.Canvas, _ = e.modBook(f.Book, width)
	e.objectBackButton(&s)
	return s, make([]int, len(s.Rows))
}

// unitBookPress is a press on cell n of the book of field f: the cell's
// switch, as one edit; a refusal stands under the book.
func (e *MapEditor) unitBookPress(f InspectionField, n int) {
	b := f.Book
	if b.Toggle == "" || n < 0 || n >= len(b.Cells) || b.Cells[n].ID == 0 {
		return
	}
	on := int64(0)
	if b.Cells[n].Dim {
		on = 1
	}
	r := e.doc.Records[e.selected]
	e.obj.note = ""
	if _, err := e.commandSetField(fieldEdit{Kind: r.Kind, Index: r.Index, Value: InspectionFieldValue{Key: b.Toggle + strconv.Itoa(int(b.Cells[n].ID)), Value: on}}); err != nil {
		e.obj.note = e.editorHostText(err.Error())
	}
}
