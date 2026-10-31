package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// chromeRectStates are the editor states the chrome rectangles are read in:
// the list, a unit's, a structure's and a trigger's window, the players
// list, the F1 list, a prompt, and three windows at their own places.
var chromeRectStates = []struct {
	name string
	act  func(e *MapEditor)
}{
	{"list", func(*MapEditor) {}},
	{"unit", func(e *MapEditor) { e.Select(0, false) }},
	{"structure", func(e *MapEditor) { e.Select(5, false) }},
	{"trigger", func(e *MapEditor) { e.Select(7, false) }},
	{"players", func(e *MapEditor) { _ = e.Key("players") }},
	{"help", func(e *MapEditor) { _ = e.Key("help") }},
	{"saveas", func(e *MapEditor) { _ = e.Key("saveas") }},
	// The windows at the places the editor gives them.
	{"unit-own", func(e *MapEditor) { e.objPlaced, e.listPlaced = false, false; e.Select(0, false) }},
	{"trigger-own", func(e *MapEditor) { e.objPlaced, e.listPlaced = false, false; e.Select(7, false) }},
	{"players-own", func(e *MapEditor) { e.objPlaced, e.listPlaced = false, false; _ = e.Key("players") }},
}

// chromeRectSizes are the window sizes the chrome rectangles are read at.
var chromeRectSizes = []image.Point{{1280, 800}, {1024, 768}, {640, 480}}

// chromeRectsText is the groups map's chrome in every state at every size:
// the map area, the status strip, the bar entries, every popup rectangle,
// where unit 0's cell shows and the map frame's size and origin.
func chromeRectsText(t *testing.T, prepare func(*MapEditor)) string {
	t.Helper()
	var out strings.Builder
	for _, size := range chromeRectSizes {
		for _, st := range chromeRectStates {
			e, _ := groupsEditor(t)
			if prepare != nil {
				prepare(e)
			}
			e.Layout(size.X, size.Y)
			st.act(e)
			var bar []image.Rectangle
			for _, b := range e.barEntries() {
				bar = append(bar, b.rect)
			}
			v := e.doc.Viewer
			ox, oy := v.place.Origin()
			fmt.Fprintf(&out, "%dx%d %s map=%v status=%v bar=%v popups=%v cell=%v frame=%dx%d origin=%.1f,%.1f\n",
				size.X, size.Y, st.name, e.mapArea(), e.statusRect(), bar, e.popupRects(), e.CellPoint(image.Pt(10, 10)), v.frameW, v.frameH, ox, oy)
		}
	}
	return out.String()
}
