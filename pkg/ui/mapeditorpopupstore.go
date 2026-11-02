package ui

import (
	"fmt"
	"image"
)

// The placement store. Each movable editor popup (the list, the object popup
// and the prompt) has one record: the corner and size the map maker gave it,
// the window the size was given for, whether it may hang past the window's
// edges, and what its policy drops when it opens or closes. The popup
// builders only read their record through placeSpec and placedCorner; every
// drag, border drag, reopen and reset below is written once and runs on the
// record of the popup it is given. Nothing outside this file writes a popup's
// corner or size.

// popupGripSide is the side of the bottom-right square that resizes both ways.
const popupGripSide = 16

// popupStanding is one popup's row of the store.
type popupStanding struct {
	at      image.Point // the kept corner; a sized prompt's corner
	placed  bool        // at holds the corner the popup first opened at
	size    image.Point // the size it was dragged to; zero is its natural size
	minWide int         // the width it had when it was first sized
	key     string      // the window it was sized for
	slack   bool        // it was dragged, so it may hang past the window's edges
	shift   image.Point // an unsized prompt's offset from its own place
	fit     int         // the list rows a sized list popup fits
}

// popupReset names the parts of a row a policy drops.
type popupReset uint8

const (
	resetSize popupReset = 1 << iota
	resetShift
	resetSlack
	resetFit
)

// popupKeyKind says what names the window a popup shows.
type popupKeyKind uint8

const (
	keyByTitle popupKeyKind = iota
	keyByList
	keyByPrompt
)

// popupRecord is the fixed part of a popup's record: how its window is named
// and the reset policy for opening and closing it.
type popupRecord struct {
	key    popupKeyKind
	opened popupReset
	closed popupReset
}

var popupRecords = [popCount]popupRecord{
	popList:   {key: keyByList, closed: resetSize | resetFit},
	popObject: {key: keyByTitle, closed: resetSize},
	popPrompt: {key: keyByPrompt, opened: resetShift | resetSlack, closed: resetShift | resetSlack | resetSize},
}

// popupHome is where a popup first opens: a corner of the map area or of the
// window, an inset from its left edge or, with Right, from its right edge less
// the popup's Width, and Top below the bar.
type popupHome struct {
	Window bool
	Right  bool
	Inset  int
	Width  int
	Top    int
}

// The places the popups first open at.
var (
	homeListMap      = popupHome{Inset: 8, Top: 8}
	homeListWindow   = popupHome{Window: true, Inset: 8, Top: 8}
	homeObjectMap    = popupHome{Right: true, Inset: 8, Width: 420, Top: 8}
	homeObjectWindow = popupHome{Window: true, Right: true, Inset: 8, Width: 420, Top: 8}
	homeMessage      = popupHome{Window: true, Right: true, Inset: 8, Width: 660, Top: 8}
	homeTrigger      = popupHome{Right: true, Inset: 8, Width: scriptPopupWidth, Top: 8}
	homePerson       = popupHome{Inset: 16, Top: 8}
	homeMod          = popupHome{Window: true, Inset: 8, Top: 6}
)

func (e *MapEditor) homeCorner(h popupHome) image.Point {
	base := e.mapArea()
	if h.Window {
		base = image.Rectangle{Max: e.window}
	}
	y := e.toolbarRect().Max.Y + h.Top
	if h.Right {
		return image.Pt(base.Max.X-h.Inset-h.Width, y)
	}
	return image.Pt(base.Min.X+h.Inset, y)
}

// popupCentre is where a prompt opens: centred across the window, its top at
// the window's height over Div, less the popup's own height when Tall.
type popupCentre struct {
	Div  int
	Tall bool
}

var (
	centreQuestion = popupCentre{Div: 3, Tall: true}
	centreSteps    = popupCentre{Div: 8}
)

// promptCorner is a prompt of size placed at c, moved by the drag of its title.
func (e *MapEditor) promptCorner(c popupCentre, size image.Point) image.Point {
	high := e.window.Y
	if c.Tall {
		high -= size.Y
	}
	return image.Pt((e.window.X-size.X)/2, max(e.toolbarRect().Max.Y+8, high/c.Div)).Add(e.places[popPrompt].shift)
}

// placedCorner is popup id's kept corner, set to home the first time it opens.
func (e *MapEditor) placedCorner(id editorPopupID, h popupHome) image.Point {
	p := &e.places[e.storeRow(id)]
	if !p.placed {
		p.at, p.placed = e.homeCorner(h), true
	}
	return p.at
}

// personCorner is the object popup's corner while it has none of its own: its
// kept corner once it was dragged or sized, else home.
func (e *MapEditor) personCorner(h popupHome) image.Point {
	p := &e.places[e.storeRow(popObject)]
	if p.slack || p.size != (image.Point{}) {
		return p.at
	}
	return e.homeCorner(h)
}

// placeSpec gives a spec the slack, the size and, for a sized prompt, the
// corner of popup id's record.
func (e *MapEditor) placeSpec(id editorPopupID, s *popupSpec) {
	p := &e.places[e.storeRow(id)]
	s.Slack = p.slack
	if p.size == (image.Point{}) {
		return
	}
	s.Size, s.MinWidth = p.size, p.minWide
	if id == popPrompt {
		s.At = p.at
	}
}

// keepInRoom moves spec s of popup id, while nobody dragged or sized it, left
// and up by what its laid out window hangs past the map area's right and
// bottom edges, never past the map area's left or top edge: a window wider or
// taller than its home allows opens clear of the docked panels and strips at
// the right and below.
func (e *MapEditor) keepInRoom(id editorPopupID, s *popupSpec) {
	p := e.places[e.storeRow(id)]
	if p.slack || p.size != (image.Point{}) {
		return
	}
	room, r := e.mapArea(), layoutPopup(*s, e.font, e.bodyFont()).Rect
	d := image.Pt(min(0, room.Max.X-r.Max.X), min(0, room.Max.Y-r.Max.Y))
	if d == (image.Point{}) {
		return
	}
	at := r.Min.Add(d)
	s.At = image.Pt(max(at.X, min(r.Min.X, room.Min.X)), max(at.Y, min(r.Min.Y, room.Min.Y)))
}

// floatPlace stands popup id's window at rectangle r: the corner and size of
// a floating side panel's or the toolbar's window, which may hang past the
// window's edges.
func (e *MapEditor) floatPlace(id editorPopupID, r image.Rectangle) {
	e.places[id] = popupStanding{at: r.Min, placed: true, size: r.Size(), minWide: 1, slack: true}
}

// floatRect is the corner and size popup id's window was given.
func (e *MapEditor) floatRect(id editorPopupID) image.Rectangle {
	p := e.places[id]
	return image.Rectangle{Min: p.at, Max: p.at.Add(p.size)}
}

// floatSpec is spec s of the floating side panel window in store row id,
// placed at its kept corner and size, fitted to its width as a window in the
// panel is (popupFlowOf). Its title row's button is the close button.
func (e *MapEditor) floatSpec(s popupSpec, id editorPopupID) popupSpec {
	e.placeSpec(id, &s)
	s.At, s.Bounds, s.Fit = e.places[id].at, e.popupBounds(), true
	return s
}

// contentSizedSpec is spec s of the floating window in store row id whose
// content gives its size: placed as floatSpec places it, at that size, and
// never sized by hand.
func (e *MapEditor) contentSizedSpec(s popupSpec, id editorPopupID, size image.Point) popupSpec {
	s = e.floatSpec(s, id)
	s.Size, s.MinWidth = size, 1
	return s
}

// dockSpec places spec s in the rectangle r of a docked panel: it fills r, and
// is never dragged.
func dockSpec(s *popupSpec, r image.Rectangle) {
	s.At, s.Size, s.Bounds, s.MinWidth, s.Slack, s.Fit = r.Min, r.Size(), r, 1, false, true
}

// naturalSize gives spec s the size its content asks for, no narrower than its
// width.
func naturalSize(s *popupSpec, size image.Point) {
	s.Size = size
	s.MinWidth = s.Width
}

// slackSpec gives spec s the slack of popup id's record.
func (e *MapEditor) slackSpec(id editorPopupID, s *popupSpec) {
	s.Slack = e.places[e.storeRow(id)].slack
}

// storeRow is the row of the store that popup id stands in: its own, except
// that the object popup of a floating right panel is that panel's window, so
// it keeps the panel's row.
func (e *MapEditor) storeRow(id editorPopupID) editorPopupID {
	if id == popObject && e.panelFloats(panelRight) {
		return popProps
	}
	return id
}

// fitListRows keeps the list rows the sized list popup fits.
func (e *MapEditor) fitListRows(n int) { e.places[popList].fit = n }

// popupWide is the width a builder wraps and clips its text to: its natural
// width, or while the popup is sized the width the size leaves beside the
// scroll bar.
func (e *MapEditor) popupWide(id editorPopupID, natural int) int {
	if id == popObject && e.objectInPanel() {
		return e.panelWide()
	}
	if sz := e.places[e.storeRow(id)].size; sz.X > 0 {
		return max(160, sz.X-widgetSpriteSize-2)
	}
	return natural
}

// windowKey names the window a place shows, so that a size dragged on one
// window is not kept by the window that replaces it. The object popup is
// named by its title, which the caller has built.
func (e *MapEditor) windowKey(id editorPopupID, title string) string {
	switch popupRecords[id].key {
	case keyByList:
		if name := e.keptWindow(id); name != "" {
			return e.listKind() + "/" + name
		}
		return e.listKind()
	case keyByPrompt:
		return fmt.Sprint(e.prompt.kind, e.message != "")
	}
	return title
}

// keptWindows names the list windows that keep the size they were dragged to
// past their closing and the editor's run, by the record kind they place.
var keptWindows = map[RecordKind]string{KindUnit: "units", KindStructure: "structures"}

// keptSizeMost bounds a kept side that the profile file gives.
const keptSizeMost = 16384

// keptWindow is the name of the keeping window popup id shows, empty for a
// window that keeps nothing.
func (e *MapEditor) keptWindow(id editorPopupID) string {
	if id != popList {
		return ""
	}
	kind, _, ok := e.placeGridKind()
	if !ok {
		return ""
	}
	return keptWindows[kind]
}

// keepSize keeps popup id's size for the keeping window it shows.
func (e *MapEditor) keepSize(id editorPopupID) {
	name := e.keptWindow(id)
	if name == "" || e.places[id].size == (image.Point{}) {
		return
	}
	if e.kept == nil {
		e.kept = map[string]image.Point{}
	}
	e.kept[name] = e.places[id].size
}

// adoptKept gives popup id, opened unsized with spec s, the size kept for the
// window it shows.
func (e *MapEditor) adoptKept(id editorPopupID, s *popupSpec) {
	p := &e.places[id]
	sz, ok := e.kept[e.keptWindow(id)]
	if !ok || p.size != (image.Point{}) {
		return
	}
	p.size, p.key = sz, e.windowKey(id, s.Title)
	p.minWide = layoutPopup(*s, e.font, e.bodyFont()).Rect.Dx()
}

// dropPopup applies the parts of a policy to popup id's row.
func (e *MapEditor) dropPopup(id editorPopupID, what popupReset) {
	p := &e.places[id]
	if what&resetSize != 0 {
		p.size = image.Point{}
	}
	if what&resetShift != 0 {
		p.shift = image.Point{}
	}
	if what&resetSlack != 0 {
		p.slack = false
	}
	if what&resetFit != 0 {
		p.fit = 0
	}
}

// popupOpened and popupClosed run the reset policy of popup id.
func (e *MapEditor) popupOpened(id editorPopupID) { e.dropPopup(id, popupRecords[id].opened) }
func (e *MapEditor) popupClosed(id editorPopupID) { e.dropPopup(id, popupRecords[id].closed) }

// dropForeignSize returns popup id to its natural size when the window it
// shows is not the one that was sized.
func (e *MapEditor) dropForeignSize(id editorPopupID, key string) bool {
	if e.storeRow(id) != id {
		return false
	}
	p := &e.places[id]
	if p.size == (image.Point{}) || p.key == key {
		return false
	}
	p.size = image.Point{}
	if id == popList {
		p.fit = 0
	}
	return true
}

// dragPopup moves the dragged popup by d. The popup builder's slack clamp
// keeps its title row and part of its width inside the window, and the kept
// place is the clamped one, so a drag back moves it at once.
func (e *MapEditor) dragPopup(d image.Point) {
	id := e.drag
	s, ok := e.movableSpec(id)
	if !ok {
		return
	}
	from := s.At
	shown := layoutPopup(s, e.font, e.bodyFont()).Rect.Min
	p := &e.places[e.storeRow(id)]
	p.slack, s.Slack = true, true
	s.At = shown.Add(d)
	to := layoutPopup(s, e.font, e.bodyFont()).Rect.Min
	if id == popPrompt && p.size == (image.Point{}) {
		p.shift = p.shift.Add(to.Sub(from))
		return
	}
	p.at = to
}

// resizeFloating moves the held border of the floating popup e.drag by d. The
// size cannot pass the least the content allows (layoutPopup holds it) or the
// window's bounds, and the border opposite the held one stays where it is.
func (e *MapEditor) resizeFloating(d image.Point) {
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
	p := &e.places[e.storeRow(id)]
	if p.size == (image.Point{}) {
		p.minWide = r.Dx()
		p.key = e.windowKey(id, s.Title)
		if id == popPrompt {
			p.at = r.Min
		}
	}
	p.size = image.Pt(max(1, w), max(1, h))
	e.keepSize(id)
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
	row := e.storeRow(id)
	e.places[row].at = at
	s, ok := e.movableSpec(id)
	if !ok {
		return
	}
	e.places[row].at = layoutPopup(s, e.font, e.bodyFont()).Rect.Min
}
