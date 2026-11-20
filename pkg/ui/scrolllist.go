package ui

import (
	"math"

	"againrom/pkg/ui/widgetkit"
)

// This file is the one list model: every scrolling list, picker, save page,
// notice window, help text, mod screen and editor popup steps through a
// scrollSpec row. A row holds the list's unit, its wheel step, its key step and
// its page step; nothing else in pkg/ui reads a wheel field to move a list.

const (
	// pickerWheelRows is how far one wheel notch moves the selection. One row
	// per notch feels broken on a 38-row list in a 25-row window; a whole page
	// loses the reader's place. Three is the usual compromise and is stated here
	// rather than at the dispatch, so the number and the window it moves through
	// sit together.
	pickerWheelRows = 3

	// modWheelLines is how many lines one wheel notch scrolls a text screen.
	modWheelLines = 3
)

// scrollUnit is what a list's steps count.
type scrollUnit uint8

const (
	scrollRows scrollUnit = iota
	scrollLines
)

// scrollSpec is one list's row of the model.
type scrollSpec struct {
	unit scrollUnit
	// wheel is the units one notch moves.
	wheel int
	// whole means the wheel moves by its whole notches (a fraction below one
	// notch moves nothing); otherwise it moves by its sign alone.
	whole bool
	// key is the arrow key's step, and a bar line request's.
	key int
	// lap is how many units a page down keeps in view; a page up keeps none.
	lap int
}

// The rows of the model, one per list.
var (
	// pickerList is the map list, the town list, the load window, the movie
	// list and every other Picker list.
	pickerList = scrollSpec{unit: scrollRows, wheel: pickerWheelRows, key: 1, lap: 1}
	// saveListFocused is the save page's list while it holds focus; saveListHover
	// is the same list turned by the wheel with the pointer over it.
	saveListFocused = scrollSpec{unit: scrollRows, wheel: 1, key: 1, lap: 1}
	saveListHover   = scrollSpec{unit: scrollRows, wheel: 1, whole: true, key: 1, lap: 1}
	// settingsPageList is a settings page's list: the music tracks, the keys.
	settingsPageList = scrollSpec{unit: scrollRows, wheel: pickerWheelRows, whole: true, key: 1, lap: 1}
	// worldMapList is the world map's destination list.
	worldMapList = scrollSpec{unit: scrollRows, wheel: 1, key: 1}
	// shopRegion is the shop's regions under the pointer, one row a notch.
	shopRegion = scrollSpec{unit: scrollRows, wheel: 1, key: 1}
	// packStrip is the viewer's pack strip, one element a notch.
	packStrip = scrollSpec{unit: scrollRows, wheel: 1, key: 1}
	// noticeList is the notice window's list and text.
	noticeList = scrollSpec{unit: scrollLines, wheel: 3, key: 1}
	// modScreenList is an open mod screen's text.
	modScreenList = scrollSpec{unit: scrollLines, wheel: modWheelLines, key: 1}
	// helpText is the help panel's text control.
	helpText = scrollSpec{unit: scrollLines, wheel: modWheelLines, key: 1, lap: helpPageDownLess}
	// questList is the quest objectives window's text.
	questList = scrollSpec{unit: scrollLines, wheel: 3, whole: true, key: 1}
	// endingList is the ending credits text.
	endingList = scrollSpec{unit: scrollLines, wheel: 3, whole: true}
	// confirmList is a save page confirmation's text.
	confirmList = scrollSpec{unit: scrollLines, wheel: 1, key: 1}
	// editorList is every editor popup list; its wheel keeps the remainder of a
	// fractional notch.
	editorList = scrollSpec{unit: scrollRows, wheel: 3, whole: true, key: 1}
)

// wheelSource is an input that carries a wheel reading.
type wheelSource interface {
	wheelNotches() float64
}

func (in appInput) wheelNotches() float64 { return in.WheelY }
func (in Input) wheelNotches() float64    { return in.WheelY }

// wheeled reports whether the wheel moved this tick.
func wheeled(in wheelSource) bool { return in.wheelNotches() != 0 }

// wheelSign is the wheel's direction: 1 for a turn away from the user, -1
// towards, 0 for none. Only pointer behaviours that are not lists read it.
func wheelSign(in wheelSource) int {
	switch w := in.wheelNotches(); {
	case w > 0:
		return 1
	case w < 0:
		return -1
	}
	return 0
}

// wheelStep is the signed step the wheel asks of the list, positive towards
// the end: a turn away from the user goes towards the first unit.
func (s scrollSpec) wheelStep(in wheelSource) int {
	return widgetkit.WheelStep(in.wheelNotches(), s.wheel, s.whole)
}

// wheelAccumulated is wheelStep for a high-resolution wheel: the fraction left
// over from each tick is kept in remainder rather than dropped.
func (s scrollSpec) wheelAccumulated(remainder *float64, in wheelSource) int {
	wheel := in.wheelNotches()
	if math.IsNaN(wheel) || math.IsInf(wheel, 0) {
		return 0
	}
	*remainder -= math.Max(-100000, math.Min(100000, wheel)) * float64(s.wheel)
	rows := int(*remainder)
	*remainder -= float64(rows)
	return rows
}

// editorWheelRows is the editor popups' wheel step; a high-resolution wheel may
// report less than one notch per update, so the remainder is preserved.
func editorWheelRows(remainder *float64, in wheelSource) int {
	return editorList.wheelAccumulated(remainder, in)
}

// keyStep is the signed step of the arrow keys, Up first.
func (s scrollSpec) keyStep(up, down bool) int {
	switch {
	case up:
		return -s.key
	case down:
		return s.key
	}
	return 0
}

// pageUp and pageDown are the units a page moves over visible units.
func (s scrollSpec) pageUp(visible int) int   { return visible }
func (s scrollSpec) pageDown(visible int) int { return visible - s.lap }

// barStep is the signed step a bar's line or page request asks of the list; ok
// is false for any other request.
func (s scrollSpec) barStep(req barRequest, visible int) (step int, ok bool) {
	switch req {
	case barLineUp:
		return -s.key, true
	case barLineDown:
		return s.key, true
	case barPageUp:
		return -s.pageUp(visible), true
	case barPageDown:
		return s.pageDown(visible), true
	}
	return 0, false
}

// listKey is the shared list's four focused keys (MENU-120). Page Up first
// selects the top row, then a page above it; Page Down first selects the last
// visible row, then a page below it. It reports whether a key acted.
func listKey(m *Picker, up, down, pageUp, pageDown bool) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	s := pickerList
	top, _ := m.Visible()
	visible := m.visibleRows()
	sel := m.Selection()
	switch {
	case up:
		m.Select(clampIndex(sel+s.keyStep(true, false), m.Len()))
	case down:
		m.Select(clampIndex(sel+s.keyStep(false, true), m.Len()))
	case pageUp:
		target := top - s.pageUp(visible)
		if sel != top {
			target = top
		}
		m.Select(clampIndex(target, m.Len()))
	case pageDown:
		target := top + visible + s.pageDown(visible)
		if sel != top+s.pageDown(visible) {
			target = top + s.pageDown(visible)
		}
		m.Select(clampIndex(target, m.Len()))
	default:
		return false
	}
	return true
}

// scrollOwner is the one value a scrolled region gives its bar: the bar with
// its position and count, the row of the model that sizes its line and page
// steps over Visible positions, and the apply. scrollBarInput.drive reads it,
// runs the step and updates it.
type scrollOwner struct {
	Bar     vScrollBar
	Spec    scrollSpec
	Visible int
	// Set writes the first shown position; Pos is already clamped.
	Set func(pos int)
	// Keys, when set, takes a line or page request as the four keys the
	// region's own routine answers.
	Keys func(up, down, pageUp, pageDown bool)
	// List is the list whose highlight a request moves instead of Set.
	List *Picker
	// Before runs once a request is made, ahead of its apply; Landed runs
	// after a list request acted, with the selection before it.
	Before func()
	Landed func(old int)
	// Grab is the pointer's offset in the thumb when Grabbed: a drag keeps
	// that offset.
	Grab    int
	Grabbed bool
}

// keys reports a line or page request as the key that makes it.
func (r barRequest) keys() (up, down, pageUp, pageDown bool) {
	return r == barLineUp, r == barLineDown, r == barPageUp, r == barPageDown
}

// moveTo clamps pos to the bar's range and writes it.
func (o *scrollOwner) moveTo(pos int) {
	o.Bar.Pos = max(0, min(pos, o.Bar.Count-1))
	if o.Set != nil {
		o.Set(o.Bar.Pos)
	}
}

// apply runs request req, with the dragged position pos for barSetPos, on
// the owner: a list moves its highlight, a region with Keys answers line and
// page requests as keys, any other moves its position by the spec's step. It
// reports whether a list request acted.
func (o *scrollOwner) apply(req barRequest, pos int) bool {
	switch {
	case req == barNone:
		return false
	case o.List != nil:
		old := o.List.Selection()
		acted := listBarRequest(o.List, req, pos)
		if acted && o.Landed != nil {
			o.Landed(old)
		}
		return acted
	case req == barSetPos:
		o.moveTo(pos)
	case o.Keys != nil:
		o.Keys(req.keys())
	default:
		step, _ := o.Spec.barStep(req, o.Visible)
		o.moveTo(o.Bar.Pos + step)
	}
	return true
}

// listBarRequest applies a bar request to its list: line and page requests
// act as the arrow and page keys, and a dragged position selects that item.
func listBarRequest(m *Picker, req barRequest, pos int) bool {
	switch req {
	case barLineUp:
		return listKey(m, true, false, false, false)
	case barLineDown:
		return listKey(m, false, true, false, false)
	case barPageUp:
		return listKey(m, false, false, true, false)
	case barPageDown:
		return listKey(m, false, false, false, true)
	case barSetPos:
		return m != nil && m.Select(clampIndex(pos, m.Len()))
	}
	return false
}
