package ui

import (
	"fmt"
	"image"
)

// Resizable editor windows. The editor has three movable popup places: the
// list, the object popup and the prompt. Each is built by layoutPopup from a
// popupSpec, and a size dragged on a border is the spec's Size, so the one
// builder lays every window out from its current size: fields stretch,
// lists and scrolled rows fit the height, wrapped text re-wraps to the width.
// A window keeps its size while it stays open and returns to its natural
// size when it closes.

// popupResizeEnabled is the builder's resize switch; a test turns it off as
// the loss control for the resize proofs.
var popupResizeEnabled = true

// resizable reports whether popup id is one of the three resizable places.
func resizable(id editorPopupID) bool {
	return popupResizeEnabled && (id == popList || id == popObject || id == popPrompt)
}

// popupWide is the width a builder wraps and clips its text to: its natural
// width, or while the popup is sized the width the size leaves beside the
// scroll bar.
func (e *MapEditor) popupWide(id editorPopupID, natural int) int {
	if sz := e.sizes[id]; sz.X > 0 {
		return max(160, sz.X-widgetSpriteSize-2)
	}
	return natural
}

// windowKey names the window a place shows, so that a size dragged on one
// window is not kept by the window that replaces it. The object popup is
// named by its title, which the caller has built.
func (e *MapEditor) windowKey(id editorPopupID, title string) string {
	switch id {
	case popList:
		return e.listKind()
	case popPrompt:
		return fmt.Sprint(e.prompt.kind, e.message != "")
	}
	return title
}

// dropForeignSize returns popup id to its natural size when the window it
// shows is not the one that was sized.
func (e *MapEditor) dropForeignSize(id editorPopupID, key string) bool {
	if e.sizes[id] == (image.Point{}) || e.sizeKey[id] == key {
		return false
	}
	e.sizes[id] = image.Point{}
	if id == popList {
		e.listFit = 0
	}
	return true
}

// listSpec is the list popup as data, at the size it was dragged to.
func (e *MapEditor) listSpec() (popupSpec, bool) {
	e.dropForeignSize(popList, e.listKind())
	s, ok := e.listSpecBase()
	if !ok {
		e.sizes[popList], e.listFit = image.Point{}, 0
		return s, false
	}
	size := e.sizes[popList]
	if size == (image.Point{}) {
		return s, true
	}
	s.Size, s.MinWidth = size, e.minWide[popList]
	if s.List != nil {
		if n := layoutPopup(s, e.font, e.bodyFont()).ListRows; n != s.ListRows {
			e.listFit = n
			s, _ = e.listSpecBase()
			s.Size, s.MinWidth = size, e.minWide[popList]
		}
	}
	return s, true
}

// objectSpec is the selected record's popup as data, at the size it was
// dragged to, and the reference each link row opens.
func (e *MapEditor) objectSpec() (popupSpec, []int, bool) {
	s, links, ok := e.objectSpecBase()
	if !ok {
		e.sizes[popObject] = image.Point{}
		return s, links, false
	}
	if e.dropForeignSize(popObject, s.Title) {
		s, links, ok = e.objectSpecBase()
	}
	size := e.sizes[popObject]
	if size == (image.Point{}) {
		return e.objectInputRows(s), links, true
	}
	s.Size, s.MinWidth = size, e.minWide[popObject]
	if s.MaxRows > 0 {
		s.MaxRows = layoutPopup(s, e.font, e.bodyFont()).RowsFit
		e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
		s.Top = e.objectTop
	}
	return e.objectInputRows(s), links, true
}

// promptSpec is the prompt and the message as one popup, at the size and the
// place it was dragged to.
func (e *MapEditor) promptSpec() (popupSpec, bool) {
	e.dropForeignSize(popPrompt, e.windowKey(popPrompt, ""))
	s, ok := e.promptSpecBase()
	if !ok {
		e.sizes[popPrompt] = image.Point{}
		return s, false
	}
	return e.addInputRows(e.sizedPrompt(s)), true
}

// sizedPrompt is a prompt-place spec at the size and the place it was dragged
// to. The window, its hit tests and its drop-downs all take the spec through
// here, so a press lands on the layout that is drawn.
func (e *MapEditor) sizedPrompt(s popupSpec) popupSpec {
	if size := e.sizes[popPrompt]; size != (image.Point{}) {
		s.Size, s.At, s.MinWidth = size, e.promptAbs, e.minWide[popPrompt]
	}
	return s
}

// PopupRect is the rectangle of the front popup: the prompt, the object
// popup or the list.
func (e *MapEditor) PopupRect() (image.Rectangle, bool) {
	for _, id := range []editorPopupID{popPrompt, popObject, popList} {
		if s, ok := e.movableSpec(id); ok {
			return layoutPopup(s, e.font, e.bodyFont()).Rect, true
		}
	}
	return image.Rectangle{}, false
}

// resizePress starts a resize drag when p lies on a border of popup id.
func (e *MapEditor) resizePress(id editorPopupID, p image.Point) bool {
	if !resizable(id) || id == popObject && e.openDropdownHit(p) || id == popPrompt && e.areaDropdownHit(p) {
		return false
	}
	s, ok := e.movableSpec(id)
	if !ok {
		return false
	}
	edge := layoutPopup(s, e.font, e.bodyFont()).edgeAt(p)
	if edge == 0 {
		return false
	}
	e.drag, e.dragFrom, e.dragEdge = id, p, edge
	return true
}

// resizePopup moves the held border by d. The size cannot pass the least the
// content allows (layoutPopup holds it) or the window's bounds, and the border
// opposite the held one stays where it is.
func (e *MapEditor) resizePopup(d image.Point) {
	id := e.drag
	s, ok := e.movableSpec(id)
	if !ok {
		return
	}
	r := layoutPopup(s, e.font, e.bodyFont()).Rect
	bounds := e.popupBounds()
	w, h := r.Dx(), r.Dy()
	switch {
	case e.dragEdge&edgeRight != 0:
		w = min(w+d.X, bounds.Max.X-r.Min.X)
	case e.dragEdge&edgeLeft != 0:
		w = min(w-d.X, r.Max.X-bounds.Min.X)
	}
	switch {
	case e.dragEdge&edgeBottom != 0:
		h = min(h+d.Y, bounds.Max.Y-r.Min.Y)
	case e.dragEdge&edgeTop != 0:
		h = min(h-d.Y, r.Max.Y-bounds.Min.Y)
	}
	if e.sizes[id] == (image.Point{}) {
		e.minWide[id] = r.Dx()
		e.sizeKey[id] = e.windowKey(id, s.Title)
		if id == popPrompt {
			e.promptAbs = r.Min
		}
	}
	e.sizes[id] = image.Pt(max(1, w), max(1, h))
	s, _ = e.movableSpec(id)
	now := layoutPopup(s, e.font, e.bodyFont()).Rect
	at := r.Min
	if e.dragEdge&edgeLeft != 0 {
		at.X = r.Max.X - now.Dx()
	}
	if e.dragEdge&edgeTop != 0 {
		at.Y = r.Max.Y - now.Dy()
	}
	e.placePopup(id, at)
}

// placePopup keeps popup id's corner at at, as far as the bounds allow.
func (e *MapEditor) placePopup(id editorPopupID, at image.Point) {
	switch id {
	case popList:
		e.listAt = at
	case popObject:
		e.objectAt = at
	case popPrompt:
		e.promptAbs = at
	}
	s, ok := e.movableSpec(id)
	if !ok {
		return
	}
	to := layoutPopup(s, e.font, e.bodyFont()).Rect.Min
	switch id {
	case popList:
		e.listAt = to
	case popObject:
		e.objectAt = to
	case popPrompt:
		e.promptAbs = to
	}
}
