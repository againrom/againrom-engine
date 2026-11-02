package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Resizable editor windows. The editor has three movable popup places: the
// list, the object popup and the prompt, and a floating side panel's window
// stands in the same store. Each is built by layoutPopup from a
// popupSpec, and a size dragged on a border is the spec's Size, so the one
// builder lays every window out from its current size: fields stretch,
// lists and scrolled rows fit the height, wrapped text re-wraps to the width.
// A window keeps its size while it stays open and returns to its natural
// size when it closes.

// popupResizeEnabled is the builder's resize switch; a test turns it off as
// the loss control for the resize proofs.
var popupResizeEnabled = true

// resizable reports whether popup id is one of the three resizable places.
func (e *MapEditor) resizable(id editorPopupID) bool {
	floating := id == popTree && e.panelFloats(panelLeft) || id == popProps && e.panelFloats(panelRight)
	return popupResizeEnabled && (id == popList || id == popObject || id == popPrompt || floating)
}

// listSpec is the list popup as data, at the size it was dragged to.
func (e *MapEditor) listSpec() (popupSpec, bool) {
	e.dropForeignSize(popList, e.windowKey(popList, ""))
	s, ok := e.listSpecBase()
	if !ok {
		e.slackSpec(popList, &s)
		e.popupClosed(popList)
		return s, false
	}
	if e.reportDocked() {
		return e.dockReport(s), true
	}
	e.adoptKept(popList, &s)
	e.placeSpec(popList, &s)
	if e.places[popList].size == (image.Point{}) {
		e.keepInRoom(popList, &s)
		return s, true
	}
	if s.List != nil {
		if n := layoutPopup(s, e.font, e.bodyFont()).ListRows; n != s.ListRows {
			e.fitListRows(n)
			s, _ = e.listSpecBase()
			e.placeSpec(popList, &s)
		}
	}
	return s, true
}

// objectSpec is the selected record's popup as data, at the size it was
// dragged to, and the reference each link row opens.
func (e *MapEditor) objectSpec() (popupSpec, []int, bool) {
	s, links, ok := e.objectSpecBase()
	e.slackSpec(popObject, &s)
	e.panels.hosted = ok && e.objectInPanel()
	if !ok {
		e.popupClosed(popObject)
		return s, links, false
	}
	if e.panels.hosted {
		return e.objectInputRows(e.hostedObject(s)), links, true
	}
	if e.dropForeignSize(popObject, s.Title) {
		s, links, ok = e.objectSpecBase()
		e.slackSpec(popObject, &s)
	}
	if e.places[e.storeRow(popObject)].size == (image.Point{}) && s.Size == (image.Point{}) {
		e.keepInRoom(popObject, &s)
		return e.objectInputRows(s), links, true
	}
	e.placeSpec(popObject, &s)
	if s.MaxRows > 0 {
		s.MaxRows = layoutPopup(s, e.font, e.bodyFont()).RowsFit
		e.objectTop = max(0, min(e.objectTop, popupRowCount(s)-s.MaxRows))
		s.Top = e.objectTop
	}
	e.keepInRoom(popObject, &s)
	return e.objectInputRows(s), links, true
}

// promptSpec is the prompt and the message as one popup, at the size and the
// place it was dragged to.
func (e *MapEditor) promptSpec() (popupSpec, bool) {
	e.dropForeignSize(popPrompt, e.windowKey(popPrompt, ""))
	s, ok := e.promptSpecBase()
	if !ok {
		e.popupClosed(popPrompt)
		return s, false
	}
	return e.addInputRows(e.sizedPrompt(s)), true
}

// sizedPrompt is a prompt-place spec at the size and the place it was dragged
// to. The window, its hit tests and its drop-downs all take the spec through
// here, so a press lands on the layout that is drawn.
func (e *MapEditor) sizedPrompt(s popupSpec) popupSpec {
	e.placeSpec(popPrompt, &s)
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

// resizeEdgeAt is the border of popup id a press at p would grab, none when
// the popup is not resizable there.
func (e *MapEditor) resizeEdgeAt(id editorPopupID, p image.Point) popupEdge {
	if _, ok := e.panelOf(id); ok {
		if id == popObject && e.openDropdownHit(p) {
			return 0
		}
		return e.panelEdgeAt(id, p)
	}
	if !e.resizable(id) || id == popObject && e.openDropdownHit(p) || id == popPrompt && e.areaDropdownHit(p) {
		return 0
	}
	s, ok := e.movableSpec(id)
	if !ok {
		return 0
	}
	return layoutPopup(s, e.font, e.bodyFont()).edgeAt(p)
}

// resizePress starts a resize drag when p lies on a border of popup id.
func (e *MapEditor) resizePress(id editorPopupID, p image.Point) bool {
	edge := e.resizeEdgeAt(id, p)
	if edge == 0 {
		return false
	}
	e.drag, e.dragFrom, e.dragEdge = id, p, edge
	return true
}

// setEditorCursorShape sets the window's pointer shape; a test replaces it.
var setEditorCursorShape = ebiten.SetCursorShape

// edgeCursorShape is the system resize cursor of a border: sideways for a
// side, up and down for the top or bottom, diagonal for a corner.
func edgeCursorShape(edge popupEdge) ebiten.CursorShapeType {
	horizontal := edge&(edgeLeft|edgeRight) != 0
	vertical := edge&(edgeTop|edgeBottom) != 0
	switch {
	case horizontal && vertical:
		if (edge&edgeLeft != 0) == (edge&edgeTop != 0) {
			return ebiten.CursorShapeNWSEResize
		}
		return ebiten.CursorShapeNESWResize
	case horizontal:
		return ebiten.CursorShapeEWResize
	case vertical:
		return ebiten.CursorShapeNSResize
	}
	return ebiten.CursorShapeDefault
}

// windowCursor is the one place the editor sets its window's pointer shape:
// the resize cursor of the popup border a press would grab, or of the border
// held in a resize drag, else the default. It only sets a shape that changed.
func (e *MapEditor) windowCursor() {
	shape := ebiten.CursorShapeDefault
	switch {
	case e.drag != popNone && e.dragEdge != 0:
		shape = edgeCursorShape(e.dragEdge)
	case e.drag == popNone && e.pointerOK && e.menu == "" && !e.down:
		target := e.chromeHit(e.pointer)
		modal := e.prompt.open() && !e.addPicking() || e.message != ""
		if !modal || target == popPrompt {
			shape = edgeCursorShape(e.resizeEdgeAt(target, e.pointer))
		}
	}
	if shape != e.cursorShape {
		e.cursorShape = shape
		setEditorCursorShape(shape)
	}
}

// resizePopup moves the held border by d: a panel's inner border, or a floating
// popup's border through the store.
func (e *MapEditor) resizePopup(d image.Point) {
	if i, ok := e.panelOf(e.drag); ok {
		e.resizePanel(i, d)
		return
	}
	e.resizeFloating(d)
}
