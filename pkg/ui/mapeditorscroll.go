package ui

import "image"

type editorScroll struct {
	input   scrollBarInput
	bar     vScrollBar
	model   *Picker
	add     bool
	details bool
	grabY   int
}

func editorListStep(m *Picker, delta int) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	m.Move(delta)
	return true
}

func (e *MapEditor) scrollTarget(p image.Point) (vScrollBar, *Picker, bool) {
	e.scroll.details = false
	switch e.chromeHit(p) {
	case popMenu:
		if b, ok := e.openMenuEntry(); ok {
			d := e.menuDropdown(b, true)
			box := d.listBox(e.font)
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List, false
			}
		}
	case popPrompt:
		if d, ok := e.openAreaDropdown(); ok {
			box := d.listBox(e.bodyFont())
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List, false
			}
		}
		if s, ok := e.promptSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList && p.In(l.List.Bar()) {
				return listBar(l.List, s.List), s.List, e.prompt.kind == promptAdd
			}
		}
	case popObject:
		if d, ok := e.openDropdown(); ok {
			box := d.listBox(e.bodyFont())
			if p.In(box.Bar()) {
				return listBar(box, d.List), d.List, false
			}
		}
		if s, _, ok := e.objectSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if p.In(l.RowBar) {
				m := NewPicker(make([]PickerRow, len(s.Rows)-s.MaxRows+1)).SetWindow(1)
				m.Select(s.Top)
				e.scroll.details = true
				return vScrollBar{Rect: l.RowBar, Pos: s.Top, Count: m.Len()}, m, false
			}
		}
	case popList:
		if s, ok := e.listSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList && p.In(l.List.Bar()) {
				return listBar(l.List, s.List), s.List, false
			}
		}
	}
	return vScrollBar{}, nil, false
}

func (e *MapEditor) scrollPointer(in Input, p image.Point, pressed bool) bool {
	s := &e.scroll
	if !s.input.active() {
		if !pressed {
			return false
		}
		b, m, add := e.scrollTarget(p)
		if m == nil {
			return false
		}
		s.bar, s.model, s.add = b, m, add
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
		if s.add {
			e.addTake(s.model.Selection())
		}
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
	e.down, e.uiDown = in.PrimaryDown, in.PrimaryDown
	return true
}
