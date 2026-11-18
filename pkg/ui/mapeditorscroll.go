package ui

import "image"

// scrollArea is one scrolled region of the editor as data: where it takes
// the wheel, its bar, how it writes its position and how far a page goes.
// scrollAreas builds every region of a popup and scrollPointer and
// chromeWheel are the only routes that read them, so each region answers the
// wheel, a track click and a thumb drag the same way.
type scrollArea struct {
	// Over is where the wheel scrolls the region from; nil is the whole
	// popup.
	Over []image.Rectangle
	// Opaque makes a press inside Over that is off the bar belong to nothing
	// else, as an open list over other content does.
	Opaque bool
	// Bar is the region's bar: Pos is the first shown position and Count is
	// one more than the last first position.
	Bar vScrollBar
	// Page is the positions one track click moves.
	Page int
	// Set writes the first shown position; Pos is already clamped.
	Set func(pos int)
	// List is the list whose highlight the bar and the wheel move instead of
	// Set; Step, when set, replaces its wheel move.
	List *Picker
	Step func(rows int)
	// Wheel is the wheel remainder the region counts its notches in.
	Wheel *float64
}

// inRects reports that p lies in one of rects, or in any of them when rects
// is nil.
func inRects(p image.Point, rects []image.Rectangle) bool {
	if rects == nil {
		return true
	}
	for _, r := range rects {
		if p.In(r) {
			return true
		}
	}
	return false
}

// rangeArea is a region of first positions 0..most that Set writes.
func rangeArea(bar image.Rectangle, pos, most, page int, over []image.Rectangle, wheel *float64, set func(int)) scrollArea {
	most = max(0, most)
	return scrollArea{Over: over, Bar: vScrollBar{Rect: bar, Pos: max(0, min(pos, most)), Count: most + 1}, Page: max(1, page), Set: set, Wheel: wheel}
}

// listArea is a region over list m whose box is box.
func listArea(box listBox, m *Picker, over []image.Rectangle, wheel *float64) scrollArea {
	return scrollArea{Over: over, Bar: listBar(box, m), List: m, Wheel: wheel}
}

// dropdownArea is the region of an open drop-down d over everything under
// it.
func (e *MapEditor) dropdownArea(d dropdown, step func(int)) scrollArea {
	over := []image.Rectangle{d.listFrame(e.bodyFont())}
	if !d.ListOnly {
		over = append(over, d.Rect)
	}
	a := listArea(d.listBox(e.bodyFont()), d.List, over, &e.listWheel)
	a.Opaque, a.Step = true, step
	return a
}

// pagedArea is the region of window id's body when the layout pass pages it.
func (e *MapEditor) pagedArea(id editorPopupID, s popupSpec, l popupLayout) (scrollArea, bool) {
	if s.MaxRows != 0 || l.RowBar.Empty() {
		return scrollArea{}, false
	}
	w := l.window(s)
	return rangeArea(l.RowBar, w.Body.Top, popupRowCount(w)-w.Body.MaxRows, w.Body.MaxRows,
		[]image.Rectangle{l.RowsViewport, l.RowBar}, &e.listWheel, func(n int) { *e.bodyPage(id) = n }), true
}

// gridArea is the region of window grid g laid out as l, whose first shown
// row top holds; a nil top takes the wheel and writes nothing.
func (e *MapEditor) gridArea(l popupLayout, g *popupGrid, top *int, wheel *float64) (scrollArea, bool) {
	if g == nil || l.GridCols <= 0 {
		return scrollArea{}, false
	}
	rects := []image.Rectangle{l.Grid, l.GridBar, l.RowsViewport}
	for _, rr := range l.Rows {
		rects = append(rects, rr.Rect)
	}
	most := g.totalRows(l.GridCols) - l.GridRows
	return rangeArea(l.GridBar, l.GridTop, most, l.GridRows, rects, wheel, func(n int) {
		if top != nil {
			*top = max(0, min(n, most))
		}
	}), true
}

// listGridTop is the first shown row of the list window's grid, nil for a
// grid that does not scroll.
func (e *MapEditor) listGridTop() *int {
	switch {
	case e.listKind() == "brush":
		if e.forestOn() {
			return &e.brush.forest.top
		}
		if e.brush.mode == brushObjects {
			return e.placeTop()
		}
		return nil
	case e.filter == editorPlayersFilter:
		return &e.relationMatrixOf().top
	}
	return e.placeTop()
}

// scrollAreas is every scrolled region of popup target in the order the
// wheel and a press try them.
func (e *MapEditor) scrollAreas(target editorPopupID) []scrollArea {
	var out []scrollArea
	add := func(a scrollArea, ok bool) {
		if ok {
			out = append(out, a)
		}
	}
	switch target {
	case popLegend:
		s, _ := e.legendSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		add(e.pagedArea(popLegend, s, l))
	case popMenu:
		var a scrollArea
		if b, ok := e.openMenuEntry(); ok {
			if d := e.menuDropdown(b, true); d.Open {
				a = listArea(d.listBox(e.font), d.List, nil, &e.listWheel)
			}
		}
		if e.menuModel != nil {
			a.Step = e.menuModel.Move
		}
		add(a, a.List != nil || a.Step != nil)
	case popPrompt:
		if d, ok := e.openAreaDropdown(); ok {
			out = append(out, e.dropdownArea(d, d.List.Move))
		}
		if s, ok := e.promptSpec(); ok {
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.HasList {
				out = append(out, listArea(l.List, s.List, []image.Rectangle{l.List.Rect, l.List.Bar()}, &e.listWheel))
			}
			if s.Grid != nil {
				add(e.gridArea(l, s.Grid, &e.prompt.gridTop, &e.objectWheel))
			}
			add(e.pagedArea(popPrompt, s, l))
		}
	case popObject:
		out = e.objectAreas()
	case popTree:
		var a scrollArea
		if s, ok := e.treeSpec(); ok {
			if l := layoutPopup(s, e.font, e.bodyFont()); l.HasList {
				a = listArea(l.List, s.List, nil, &e.listWheel)
			}
		}
		if e.tree.model != nil {
			a.Step, a.Wheel = e.tree.model.Scroll, &e.listWheel
		}
		add(a, a.List != nil || a.Step != nil)
	case popList:
		out = e.listAreas()
	}
	return out
}

// listAreas is the regions of the list window.
func (e *MapEditor) listAreas() []scrollArea {
	var out []scrollArea
	if d, ok := e.placeOwnerDropdown(); ok && e.place.ownerList != nil {
		out = append(out, e.dropdownArea(d, e.place.ownerList.Move))
	}
	s, ok := e.listSpec()
	if !ok {
		return out
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if s.Grid != nil {
		if a, ok := e.gridArea(l, s.Grid, e.listGridTop(), &e.listWheel); ok {
			out = append(out, a)
		}
	}
	if a, ok := e.pagedArea(popList, s, l); ok {
		out = append(out, a)
	}
	var a scrollArea
	if l.HasList {
		a = listArea(l.List, s.List, nil, &e.listWheel)
	}
	if e.model != nil && e.listKind() != "panel" && (e.listKind() != "brush" || e.brushWindowOf().list) {
		a.Wheel = &e.listWheel
		a.Step = func(rows int) {
			if e.listKind() == "catalog" {
				e.catalogStep(rows)
			} else {
				e.model.Move(rows)
			}
		}
	}
	if a.List != nil || a.Step != nil {
		out = append(out, a)
	}
	return out
}

// objectAreas is the regions of the object window.
func (e *MapEditor) objectAreas() []scrollArea {
	var out []scrollArea
	if d, ok := e.openDropdown(); ok && d.Open {
		out = append(out, e.dropdownArea(d, e.obj.choice.MoveChoice))
	}
	s, _, ok := e.objectSpec()
	var l popupLayout
	if ok {
		l = layoutPopup(s, e.font, e.bodyFont())
	}
	if e.graphShown() || ok && s.Canvas != nil {
		a := scrollArea{Wheel: &e.objectWheel, Step: func(rows int) {
			if e.graphShown() {
				e.graphPan(image.Pt(0, rows*(fontHeight(e.bodyFont(), 15)+popupRowExtra)))
			}
		}}
		if ok && s.Canvas != nil {
			bar := l.canvasBar(s.Canvas)
			pitch := max(1, l.Pitch)
			a.Bar, a.Page = bar, max(1, l.Canvas.Dy()/pitch)
			a.Set = func(n int) { e.graphAt.Y = n * pitch }
		}
		if !e.graphShown() {
			a.Over = []image.Rectangle{a.Bar.Rect}
		}
		out = append(out, a)
	}
	if e.modWindow {
		if ms, _, ok := e.modSpec(); ok && ms.Grid != nil {
			if a, ok := e.gridArea(layoutPopup(ms, e.font, e.bodyFont()), ms.Grid, &e.modGridTop, &e.objectWheel); ok {
				out = append(out, a)
			}
		}
	}
	if !ok {
		return out
	}
	for _, rr := range l.Rows {
		if b, visible, field, ok := e.textBar(s, rr); ok {
			out = append(out, rangeArea(rr.TextBar, b.Pos, b.Count-1, visible, []image.Rectangle{rr.Rect, rr.TextBar},
				&e.objectWheel, func(n int) { e.setTextFirst(field, n) }))
		}
	}
	if a, ok := e.pagedArea(popObject, s, l); ok {
		out = append(out, a)
	} else if s.MaxRows != 0 {
		out = append(out, rangeArea(l.RowBar, s.Top, popupRowCount(specWindow(s))-s.MaxRows, s.MaxRows,
			nil, &e.objectWheel, func(n int) { e.objectTop = n }))
	}
	return out
}

// scrollAreaAt is the region of popup target the wheel scrolls from p.
func (e *MapEditor) scrollAreaAt(target editorPopupID, p image.Point) (scrollArea, bool) {
	for _, a := range e.scrollAreas(target) {
		if inRects(p, a.Over) {
			return a, true
		}
	}
	return scrollArea{}, false
}

// scrollWheelRows moves region a by rows wheel rows.
func (e *MapEditor) scrollWheelRows(a scrollArea, rows int) {
	switch {
	case a.Step != nil:
		a.Step(rows)
	case a.List != nil:
		a.List.Move(rows)
	case a.Set != nil:
		a.Set(max(0, min(a.Bar.Pos+rows, a.Bar.Count-1)))
	}
}

// scrollWheel scrolls the region of popup target under the pointer by the
// wheel.
func (e *MapEditor) scrollWheel(target editorPopupID, wheel wheelSource) {
	if a, ok := e.scrollAreaAt(target, e.pointer); ok && a.Wheel != nil {
		e.scrollWheelRows(a, editorWheelRows(a.Wheel, wheel))
	}
}

// scrollPress is the region whose bar a press at p lands on.
func (e *MapEditor) scrollPress(p image.Point) (scrollArea, bool) {
	for _, a := range e.scrollAreas(e.chromeHit(p)) {
		if !inRects(p, a.Over) {
			continue
		}
		if inRects(p, []image.Rectangle{a.Bar.Rect}) {
			return a, true
		}
		if a.Opaque {
			break
		}
	}
	return scrollArea{}, false
}

func editorListStep(m *Picker, delta int) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	m.Move(delta)
	return true
}

// scrollRequest applies a bar request to region a.
func (e *MapEditor) scrollRequest(a *scrollArea, req barRequest, pos int) {
	if a.List != nil {
		old := a.List.Selection()
		if listBarRequest(a.List, req, pos) {
			if e.catalog && a.List == e.model && a.List.Len() > 0 && !a.List.Rows()[a.List.Selection()].Choosable {
				delta := 1
				if a.List.Selection() < old {
					delta = -1
				}
				e.catalogStep(delta)
			}
		}
		return
	}
	if req == barNone {
		return
	}
	a.Bar.Pos = rangeBarRequest(a.Bar.Pos, a.Bar.Count, a.Page, req, pos)
	if a.Set != nil {
		a.Set(a.Bar.Pos)
	}
}

func (e *MapEditor) scrollPointer(in Input, p image.Point, pressed bool) bool {
	s := &e.scroll
	if !s.input.active() {
		if !pressed {
			return false
		}
		a, ok := e.scrollPress(p)
		if !ok {
			return false
		}
		s.area = a
		s.grabY = p.Y - a.Bar.Thumb().Min.Y
	}
	a := &s.area
	if a.List != nil {
		a.Bar.Pos, a.Bar.Count = max(a.List.Selection(), 0), a.List.Len()
	}
	req, pos := s.input.step(a.Bar, p, true, appInput{Viewer: in, PrimaryPressed: pressed, PrimaryReleased: !in.PrimaryDown})
	if req == barSetPos {
		span := a.Bar.Rect.Dy() - 3*a.Bar.Rect.Dx() + 8
		if span > 0 {
			y := p.Y - s.grabY - a.Bar.Rect.Min.Y - a.Bar.Rect.Dx() + 4
			pos = min(max((y*(a.Bar.Count-1)+span/2)/span, 0), a.Bar.Count-1)
		}
	}
	e.scrollRequest(a, req, pos)
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

type editorScroll struct {
	input scrollBarInput
	area  scrollArea
	grabY int
}

// textBar is the scroll bar of the whole multi-line row rr that layout l
// shows, the lines its box holds, and its field; one position per hidden line.
func (e *MapEditor) textBar(s popupSpec, rr popupRowRect) (vScrollBar, int, int, bool) {
	field, ok := e.obj.rowField[rr.Index]
	if !ok || rr.TextBar.Empty() || rr.Index >= len(s.Rows) || field < 0 || field >= len(e.obj.buf) {
		return vScrollBar{}, 0, 0, false
	}
	row := s.Rows[rr.Index]
	lines, visible := multilineView(e.bodyFont(), row, rr.Control)
	return vScrollBar{Rect: rr.TextBar, Pos: multilineStart(row, len(lines), visible), Count: max(1, len(lines)-visible+1)}, visible, field, true
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
