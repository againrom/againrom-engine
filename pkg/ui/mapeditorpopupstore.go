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
	p := &e.places[id]
	if !p.placed {
		p.at, p.placed = e.homeCorner(h), true
	}
	return p.at
}

// personCorner is the object popup's corner while it has none of its own: its
// kept corner once it was dragged or sized, else home.
func (e *MapEditor) personCorner(h popupHome) image.Point {
	p := &e.places[popObject]
	if p.slack || p.size != (image.Point{}) {
		return p.at
	}
	return e.homeCorner(h)
}

// placeSpec gives a spec the slack, the size and, for a sized prompt, the
// corner of popup id's record.
func (e *MapEditor) placeSpec(id editorPopupID, s *popupSpec) {
	p := &e.places[id]
	s.Slack = p.slack
	if p.size == (image.Point{}) {
		return
	}
	s.Size, s.MinWidth = p.size, p.minWide
	if id == popPrompt {
		s.At = p.at
	}
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
func (e *MapEditor) slackSpec(id editorPopupID, s *popupSpec) { s.Slack = e.places[id].slack }

// fitListRows keeps the list rows the sized list popup fits.
func (e *MapEditor) fitListRows(n int) { e.places[popList].fit = n }

// popupWide is the width a builder wraps and clips its text to: its natural
// width, or while the popup is sized the width the size leaves beside the
// scroll bar.
func (e *MapEditor) popupWide(id editorPopupID, natural int) int {
	if id == popObject && e.objectInPanel() {
		return e.panelWide()
	}
	if sz := e.places[id].size; sz.X > 0 {
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
		return e.listKind()
	case keyByPrompt:
		return fmt.Sprint(e.prompt.kind, e.message != "")
	}
	return title
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
	p := &e.places[id]
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
	p := &e.places[id]
	if p.size == (image.Point{}) {
		p.minWide = r.Dx()
		p.key = e.windowKey(id, s.Title)
		if id == popPrompt {
			p.at = r.Min
		}
	}
	p.size = image.Pt(max(1, w), max(1, h))
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
	e.places[id].at = at
	s, ok := e.movableSpec(id)
	if !ok {
		return
	}
	e.places[id].at = layoutPopup(s, e.font, e.bodyFont()).Rect.Min
}
