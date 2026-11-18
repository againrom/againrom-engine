package ui

import "image"

type editorScroll struct {
	input   scrollBarInput
	bar     vScrollBar
	model   *Picker
	details bool
	// canvas is the pixels one bar position moves W's graph, zero for a
	// bar of anything else.
	canvas int
	// text is one more than the field whose text the bar scrolls, zero for
	// a bar of anything else.
	text  int
	grabY int
	// page is the window whose body the layout pass pages that the bar
	// moves, popNone for none.
	page editorPopupID
}

// pageBar is the row bar of a body the layout pass pages; the caller has
// found the pointer on l.RowBar.
func (e *MapEditor) pageBar(id editorPopupID, s popupSpec, l popupLayout) (vScrollBar, *Picker, bool) {
	if s.MaxRows != 0 {
		return vScrollBar{}, nil, false
	}
	w := l.window(s)
	m := NewPicker(make([]PickerRow, popupRowCount(w)-w.Body.MaxRows+1)).SetWindow(1)
	m.Select(w.Body.Top)
	e.scroll.page = id
	return vScrollBar{Rect: l.RowBar, Pos: w.Body.Top, Count: m.Len()}, m, true
}

// pagedLayout is window id's spec and layout when the layout pass pages its
// body.
func (e *MapEditor) pagedLayout(id editorPopupID) (popupSpec, popupLayout, bool) {
	s, ok := e.movableSpec(id)
	if !ok || s.MaxRows != 0 {
		return popupSpec{}, popupLayout{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	return s, l, !l.RowBar.Empty()
}

// pageWheel moves the paged body of window id by the wheel; the caller has
// found the pointer over its rows or its bar.
func (e *MapEditor) pageWheel(id editorPopupID, s popupSpec, l popupLayout, wheel wheelSource) {
	w := l.window(s)
	*e.bodyPage(id) = max(0, min(w.Body.Top+editorWheelRows(&e.listWheel, wheel), popupRowCount(w)-w.Body.MaxRows))
}

func editorListStep(m *Picker, delta int) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	m.Move(delta)
	return true
}

func (e *MapEditor) scrollTarget(p image.Point) (vScrollBar, *Picker) {
	e.scroll.details, e.scroll.canvas, e.scroll.page, e.scroll.text = false, 0, popNone, 0
	switch e.chromeHit(p) {
	case popMenu:
		if b, ok := e.openMenuEntry(); ok {
			d := e.menuDropdown(b, true)
			box := d.listBox(e.font)
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List
			}
		}
	case popPrompt:
		if d, ok := e.openAreaDropdown(); ok {
			box := d.listBox(e.bodyFont())
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List
			}
		}
		if s, ok := e.promptSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList && p.In(l.List.Bar()) {
				return listBar(l.List, s.List), s.List
			}
			if p.In(l.RowBar) {
				if b, m, ok := e.pageBar(popPrompt, s, l); ok {
					return b, m
				}
			}
		}
	case popObject:
		if d, ok := e.openDropdown(); ok {
			box := d.listBox(e.bodyFont())
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List
			}
		}
		if s, _, ok := e.objectSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			for _, rr := range l.Rows {
				if p.In(rr.TextBar) {
					if b, m, field, ok := e.textBar(s, rr); ok {
						e.scroll.text = field + 1
						return b, m
					}
				}
			}
			if p.In(l.RowBar) {
				if b, m, ok := e.pageBar(popObject, s, l); ok {
					return b, m
				}
				m := NewPicker(make([]PickerRow, popupRowCount(specWindow(s))-s.MaxRows+1)).SetWindow(1)
				m.Select(s.Top)
				e.scroll.details = true
				return vScrollBar{Rect: l.RowBar, Pos: s.Top, Count: m.Len()}, m
			}
			if s.Canvas != nil && p.In(l.CanvasBar) {
				bar := l.canvasBar(s.Canvas)
				m := NewPicker(make([]PickerRow, bar.Count)).SetWindow(1)
				m.Select(bar.Pos)
				e.scroll.canvas = max(1, l.Pitch)
				return bar, m
			}
		}
	case popTree:
		if s, ok := e.treeSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList && p.In(l.List.Bar()) {
				return listBar(l.List, s.List), s.List
			}
		}
	case popList:
		if s, ok := e.listSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList && p.In(l.List.Bar()) {
				return listBar(l.List, s.List), s.List
			}
			if p.In(l.RowBar) {
				if b, m, ok := e.pageBar(popList, s, l); ok {
					return b, m
				}
			}
		}
	}
	return vScrollBar{}, nil
}

func (e *MapEditor) scrollPointer(in Input, p image.Point, pressed bool) bool {
	s := &e.scroll
	if !s.input.active() {
		if !pressed {
			return false
		}
		b, m := e.scrollTarget(p)
		if m == nil {
			return false
		}
		s.bar, s.model = b, m
		s.grabY = p.Y - b.Thumb().Min.Y
	}
	s.bar.Pos, s.bar.Count = max(s.model.Selection(), 0), s.model.Len()
	req, pos := s.input.step(s.bar, p, true, appInput{Viewer: in, PrimaryPressed: pressed, PrimaryReleased: !in.PrimaryDown})
	if req == barSetPos {
		span := s.bar.Rect.Dy() - 3*s.bar.Rect.Dx() + 8
		if span > 0 {
			y := p.Y - s.grabY - s.bar.Rect.Min.Y - s.bar.Rect.Dx() + 4
			pos = min(max((y*(s.bar.Count-1)+span/2)/span, 0), s.bar.Count-1)
		}
	}
	oldSelection := s.model.Selection()
	if listBarRequest(s.model, req, pos) {
		if e.catalog && s.model == e.model && s.model.Len() > 0 && !s.model.Rows()[s.model.Selection()].Choosable {
			delta := 1
			if s.model.Selection() < oldSelection {
				delta = -1
			}
			e.catalogStep(delta)
		}
	}
	if s.details {
		e.objectTop = s.model.Selection()
	}
	if s.text > 0 {
		e.setTextFirst(s.text-1, s.model.Selection())
	}
	if s.page != popNone {
		*e.bodyPage(s.page) = s.model.Selection()
	}
	if s.canvas > 0 {
		e.graphAt.Y = s.model.Selection() * s.canvas
	}
	e.down, e.uiDown = in.PrimaryDown, in.PrimaryDown
	return true
}

// multilineStart is the first of n wrapped lines a multi-line row shows when
// its box holds visible: the line the reader scrolled to, or the end while
// the row is typed in and not scrolled.
func multilineStart(row popupRow, n, visible int) int {
	if row.Focus && !row.Scrolled {
		return max(0, n-visible)
	}
	return max(0, min(row.First, n-visible))
}

// textBar is the scroll bar of the whole multi-line row rr that layout l
// shows, with its field; its model holds one position per hidden line.
func (e *MapEditor) textBar(s popupSpec, rr popupRowRect) (vScrollBar, *Picker, int, bool) {
	field, ok := e.obj.rowField[rr.Index]
	if !ok || rr.TextBar.Empty() || rr.Index >= len(s.Rows) || field < 0 || field >= len(e.obj.buf) {
		return vScrollBar{}, nil, 0, false
	}
	row := s.Rows[rr.Index]
	lines, visible := multilineView(e.bodyFont(), row, rr.Control)
	m := NewPicker(make([]PickerRow, max(1, len(lines)-visible+1))).SetWindow(1)
	m.Select(multilineStart(row, len(lines), visible))
	return vScrollBar{Rect: rr.TextBar, Pos: m.Selection(), Count: m.Len()}, m, field, true
}

// setTextFirst scrolls text field f to show line first at its top.
func (e *MapEditor) setTextFirst(f, first int) {
	if e.obj.textFirst == nil {
		e.obj.textFirst = map[int]textScroll{}
	}
	e.obj.textFirst[f] = textScroll{first: first, of: e.obj.buf[f], caret: e.textCaret(f)}
}

// textCaret is the caret of text field f while the keyboard edits it, else 0.
func (e *MapEditor) textCaret(f int) int {
	if e.obj.focus == f && e.input.target == &e.obj.buf[f] {
		return e.input.model.Caret
	}
	return 0
}

// textWheel scrolls the whole multi-line row under the pointer whose text is
// taller than its box; it reports that one took the wheel.
func (e *MapEditor) textWheel(wheel wheelSource) bool {
	s, _, ok := e.objectSpec()
	if !ok {
		return false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	part, index := l.hit(e.pointer)
	if part != popupRowControl && part != popupRowLine {
		return false
	}
	for _, rr := range l.Rows {
		if rr.Index != index || rr.TextBar.Empty() {
			continue
		}
		if bar, m, field, ok := e.textBar(s, rr); ok {
			e.setTextFirst(field, max(0, min(bar.Pos+editorWheelRows(&e.objectWheel, wheel), m.Len()-1)))
			return true
		}
	}
	return false
}
