package ui

import (
	"fmt"
	"image"
)

// The placement store. Each movable editor popup (the list, the object popup
// and the prompt) has one record: the corner and size the map maker gave it,
// the window the size was given for, whether it may hang past the window's
// edges, and what its policy drops when it opens or closes. Every window
// first opens centred in the editor window (centreCorner), unless its kind
// states its own place (openPlace); after a drag or a size it keeps its
// record. The popup builders only read their record through placeSpec and
// keptCorner; every drag, border drag, reopen and reset below is written once
// and runs on the record of the popup it is given. Nothing outside this file
// writes a popup's corner or size.

// popupGripSide is the side of the bottom-right square that resizes both ways.
const popupGripSide = 16

// popupStanding is one popup's row of the store.
type popupStanding struct {
	at      image.Point // the kept corner; a sized prompt's corner
	placed  bool        // at holds a corner it was given, so it does not open centred
	size    image.Point // the size it was dragged to; zero is its natural size
	minWide int         // the width it had when it was first sized
	key     string      // the window it was sized for
	slack   bool        // it was dragged, so it may hang past the window's edges
	short   bool        // it was shortened from a place past the bottom edge, down to its least rows
	shift   image.Point // an unsized prompt's offset from its own place
	fit     int         // the list rows a sized list popup fits
	page    int         // the first row its body shows where the layout pass pages it
	open    bool        // its centred window stands open
	offset  bool        // it opened while the list window stood, so it stands off it
	stood   image.Point // the corner its centred window last stood at
}

// popupReset names the parts of a row a policy drops.
type popupReset uint8

const (
	resetSize popupReset = 1 << iota
	resetShift
	resetSlack
	resetFit
	resetPage
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
	popList:   {key: keyByList, closed: resetSize | resetFit | resetPage},
	popObject: {key: keyByTitle, closed: resetSize | resetPage},
	popPrompt: {key: keyByPrompt, opened: resetShift | resetSlack | resetPage, closed: resetShift | resetSlack | resetSize | resetPage},
}

// openPlace is where a window kind first opens when it states its own place:
// its top at the editor window's height over Top, centred across. The zero
// place is the middle of the window.
type openPlace struct{ Top int }

// centreCorner is the corner a window of size first opens at: centred across
// the editor window and, unless place states its top, down it, never above
// the bounds' top.
func (e *MapEditor) centreCorner(size image.Point, place openPlace) image.Point {
	y := (e.window.Y - size.Y) / 2
	if place.Top > 0 {
		y = e.window.Y / place.Top
	}
	return image.Pt((e.window.X-size.X)/2, max(e.popupBounds().Min.Y, y))
}

// keptCorner is popup id's kept corner once it was given one, dragged or
// sized; else the bounds' top-left corner, which keepInRoom and openCentred
// replace with the middle.
func (e *MapEditor) keptCorner(id editorPopupID) image.Point {
	p := e.places[e.storeRow(id)]
	if p.placed || p.slack || p.size != (image.Point{}) {
		return p.at
	}
	return e.popupBounds().Min
}

// placeSpec gives a spec the slack, the size and, for a prompt, the corner of
// popup id's record: a sized prompt's kept corner, an unsized one moved by the
// drag of its title. It also gives the kept page.
func (e *MapEditor) placeSpec(id editorPopupID, s *popupSpec) {
	p := &e.places[e.storeRow(id)]
	s.Slack, s.Short = p.slack, p.short
	switch {
	case p.size != (image.Point{}):
		s.Size, s.MinWidth = p.size, p.minWide
		if id == popPrompt {
			s.At = p.at
		}
	case id == popPrompt:
		s.At = s.At.Add(p.shift)
	}
	e.pageSpec(id, s)
}

// keepInRoom stands spec s of popup id, while nobody dragged or sized it, in
// the middle of the editor window, or offset from the window it opened from
// (openCentred), then moves it left and up by what its laid out window hangs
// past the map area's right and bottom edges, never past the map area's left
// or top edge: a window wider or taller than the middle allows opens clear of
// the docked panels and strips at the right and below. An offset window that
// cannot stand at its offset inside the map area keeps the middle. Every list
// and object window passes here.
func (e *MapEditor) keepInRoom(id editorPopupID, s *popupSpec) {
	e.pageSpec(id, s)
	row := e.storeRow(id)
	p := e.places[row]
	if p.slack || p.size != (image.Point{}) {
		e.places[row].open = false
		return
	}
	middle := e.openCentred(id, s)
	room := e.mapArea()
	keep := func(at image.Point) image.Point {
		probe := *s
		probe.At = at
		r := layoutPopup(probe, e.font, e.bodyFont()).Rect
		d := image.Pt(min(0, room.Max.X-r.Max.X), min(0, room.Max.Y-r.Max.Y))
		at = r.Min.Add(d)
		return image.Pt(max(at.X, min(r.Min.X, room.Min.X)), max(at.Y, min(r.Min.Y, room.Min.Y)))
	}
	want := s.At
	s.At = keep(want)
	if want != middle && s.At != want {
		s.At = keep(middle)
	}
	e.places[row].stood = s.At
}

// openCentred stands spec s of popup id, while nobody dragged or sized it,
// in the middle of the editor window by the size it lays out to from the
// bounds' top-left corner, so the size never depends on the corner it had,
// its title never above the bar. A popup that opened while the centred
// list popup stood open stands one step down and right of the list's corner
// instead when it fits the room that way, the step being the title row's
// height and popupPad, so the earlier window's title bar and left edge stay
// clear. A dragged, sized or otherwise placed popup keeps its place. It
// returns the middle corner.
func (e *MapEditor) openCentred(id editorPopupID, s *popupSpec) image.Point {
	row := e.storeRow(id)
	p := &e.places[row]
	if p.slack || p.size != (image.Point{}) || p.placed {
		p.open = false
		return s.At
	}
	top := e.popupBounds().Min
	probe := *s
	probe.At = top
	l := layoutPopup(probe, e.font, e.bodyFont())
	size := l.Rect.Size()
	size.X, size.Y = max(size.X, s.CentreSize.X), max(size.Y, s.CentreSize.Y)
	middle := e.centreCorner(size, openPlace{})
	s.At = middle
	if !p.open {
		p.open, p.offset = true, row != popList && e.listStands()
	}
	if q := e.places[popList]; p.offset && q.open && e.listKind() != "" {
		step := l.Title.Dy() + popupPad
		at := q.stood.Add(image.Pt(step, step))
		if r := l.Rect.Add(at.Sub(top)); r.Intersect(e.popupRoom()) == r {
			s.At = at
		}
	}
	p.stood = s.At
	return middle
}

// floatPlace stands popup id's window at rectangle r: the corner and size of
// a floating side panel's or the toolbar's window, which may hang past the
// window's edges.
func (e *MapEditor) floatPlace(id editorPopupID, r image.Rectangle) {
	e.places[id] = popupStanding{at: r.Min, placed: true, size: r.Size(), minWide: 1, slack: true}
}

func (e *MapEditor) restoreBarPlaces(places [popCount]popupStanding) {
	for _, id := range []editorPopupID{popList, popObject, popPrompt, popProps} {
		e.places[id] = places[id]
	}
	e.places[popList].key = e.windowKey(popList, "")
	e.places[popPrompt].key = e.windowKey(popPrompt, "")
	if s, _, ok := e.objectSpecBase(); ok {
		e.places[e.storeRow(popObject)].key = s.Title
	}
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

// rowsFit is the rows a window that sets MaxRows shows when the layout pass
// fits it to its bounds, measured from the bounds' top, so the count does not
// hang on the corner the window stands at.
func (e *MapEditor) rowsFit(s popupSpec) int {
	s.At = s.Bounds.Min
	return layoutPopup(s, e.font, e.bodyFont()).RowsFit
}

// bodyPage is the kept first row of popup id's paged body.
func (e *MapEditor) bodyPage(id editorPopupID) *int {
	return &e.places[e.storeRow(id)].page
}

// pageSpec gives a window that sets no MaxRows its kept page, and holds the
// page where the layout pass shows it: zero when the window does not page.
func (e *MapEditor) pageSpec(id editorPopupID, s *popupSpec) {
	page := e.bodyPage(id)
	if s.MaxRows != 0 || *page == 0 {
		return
	}
	s.Page = *page
	if l := layoutPopup(*s, e.font, e.bodyFont()); l.RowBar.Empty() {
		*page = 0
	} else {
		*page = l.Window.Body.Top
	}
	s.Page = *page
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
// window it shows, centred at that size unless it was dragged or placed.
func (e *MapEditor) adoptKept(id editorPopupID, s *popupSpec) {
	p := &e.places[id]
	sz, ok := e.kept[e.keptWindow(id)]
	if !ok || p.size != (image.Point{}) {
		return
	}
	p.size, p.key = sz, e.windowKey(id, s.Title)
	p.minWide = layoutPopup(*s, e.font, e.bodyFont()).Rect.Dx()
	if !p.slack && !p.placed {
		probe := *s
		probe.At, probe.Size, probe.MinWidth = e.popupBounds().Min, p.size, p.minWide
		p.at = e.centreCorner(layoutPopup(probe, e.font, e.bodyFont()).Rect.Size(), openPlace{})
	}
}

// dropPopup applies the parts of a policy to popup id's row.
func (e *MapEditor) dropPopup(id editorPopupID, what popupReset) {
	p := &e.places[id]
	if what&resetSize != 0 {
		p.size, p.short = image.Point{}, false
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
	if what&resetPage != 0 {
		p.page = 0
	}
}

// popupOpened and popupClosed run the reset policy of popup id.
func (e *MapEditor) popupOpened(id editorPopupID) { e.dropPopup(id, popupRecords[id].opened) }
func (e *MapEditor) popupClosed(id editorPopupID) {
	e.dropPopup(id, popupRecords[id].closed)
	e.popupHidden(id)
}

// listStands reports whether the list popup stands open, laying it out first
// when its own window has not yet been, so the order the popups are laid out
// in does not decide which one opens off the other.
func (e *MapEditor) listStands() bool {
	if e.listKind() == "" {
		return false
	}
	if !e.places[popList].open {
		_, _ = e.listSpec()
	}
	return e.places[popList].open
}

// popupHidden notes that popup id's centred window no longer stands, so
// no later window opens offset from it.
func (e *MapEditor) popupHidden(id editorPopupID) { e.places[e.storeRow(id)].open = false }

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
	p.size, p.short = image.Point{}, false
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
		// The title row stays inside the bounds, so the bottom stays put.
		h = max(h, r.Max.Y-bounds.Max.Y+layoutPopup(s, e.font, e.bodyFont()).Title.Dy())
	}
	p := &e.places[e.storeRow(id)]
	if r.Max.Y > bounds.Max.Y && e.dragEdge&(edgeTop|edgeBottom) != 0 {
		// A window past the bottom edge stays where it stands and shortens
		// to its least rows.
		p.short, p.slack, s.Short, s.Slack = true, true, true, true
	}
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

// holdPopup places popup id at at and keeps it there.
func (e *MapEditor) holdPopup(id editorPopupID, at image.Point) {
	e.placePopup(id, at)
	e.places[e.storeRow(id)].placed = true
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
