package ui

import (
	"fmt"
	"image"
	"image/color"
)

// The triggers on the map: every place a trigger's conditions and actions
// name is outlined in that trigger's colour and labelled with its number and
// name. The selected trigger's areas are gold; a click on its label selects
// the trigger, a click inside an area does not. The Layers menu shows or
// hides them.

// areaInks tell overlapping triggers apart; gold is the selection's.
var areaInks = []color.RGBA{
	{0x4f, 0xd6, 0xe0, 255}, {0xe0, 0x6f, 0xd6, 255}, {0x8c, 0xe0, 0x4f, 255}, {0xf0, 0x9a, 0x3e, 255},
	{0x7a, 0x9c, 0xff, 255}, {0xff, 0x7a, 0x8a, 255}, {0xd8, 0xe0, 0x6a, 255}, {0xb0, 0x86, 0xff, 255},
}

func areaInk(trigger int) color.RGBA { return areaInks[trigger%len(areaInks)] }

// areasShown reports that the Layers menu shows the trigger areas.
func (e *MapEditor) areasShown() bool {
	return e.doc != nil && e.doc.Script != nil && !e.view.hideAreas
}

// selectedTrigger is the index of the trigger whose areas are highlighted:
// the selected trigger, or the one a node popup returns to; -1 for none.
func (e *MapEditor) selectedTrigger() int {
	if v, ok := e.scriptViewOf(e.selected); ok {
		if v.trigger != nil {
			return v.index
		}
		for i := len(e.objectBack) - 1; i >= 0; i-- {
			if p := e.objectBack[i]; p.kind == KindTrigger {
				return p.index
			}
		}
	}
	return -1
}

// TriggerAreas are the areas of trigger t, for a headless driver.
func (e *MapEditor) TriggerAreas(t int) []InspectionArea {
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	var out []InspectionArea
	for _, a := range e.doc.Script.Areas {
		if a.Trigger == t {
			out = append(out, a)
		}
	}
	return out
}

// corner is world corner c of the cell grid on the map frame.
func (e *MapEditor) corner(c image.Point) (float32, float32) {
	x, y := e.doc.Viewer.cellScreenCorner(c.X, c.Y)
	return float32(x), float32(y)
}

// areaOutline is the outline of the cells r covers, along the cell corners so
// it follows the ground; a long side is walked in steps.
func (e *MapEditor) areaOutline(r image.Rectangle, ink color.RGBA) []editorLine {
	var pts []image.Point
	step := max(1, (r.Dx()+r.Dy())/128)
	for x := r.Min.X; x < r.Max.X; x += step {
		pts = append(pts, image.Pt(x, r.Min.Y))
	}
	for y := r.Min.Y; y < r.Max.Y; y += step {
		pts = append(pts, image.Pt(r.Max.X, y))
	}
	for x := r.Max.X; x > r.Min.X; x -= step {
		pts = append(pts, image.Pt(x, r.Max.Y))
	}
	for y := r.Max.Y; y > r.Min.Y; y -= step {
		pts = append(pts, image.Pt(r.Min.X, y))
	}
	out := make([]editorLine, 0, len(pts))
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		x0, y0 := e.corner(p)
		x1, y1 := e.corner(q)
		out = append(out, editorLine{x0, y0, x1, y1, ink})
	}
	return out
}

// areaLines outline every trigger area, the highlighted trigger's in gold and
// doubled, a square's centre crossed.
func (e *MapEditor) areaLines() []editorLine {
	var out, top, handles []editorLine
	drag := e.areaDrag()
	if drag != nil && (drag.kind == handleDraw || drag.kind == handleDrawSquare) {
		for _, l := range e.areaOutline(e.areaPreview(drag), editorGold) {
			top = append(top, l, editorLine{l.x0 + 1, l.y0 + 1, l.x1 + 1, l.y1 + 1, editorGold})
		}
	}
	if !e.areasShown() {
		return top
	}
	sel := e.selectedTrigger()
	for i, a := range e.doc.Script.Areas {
		if drag != nil && drag.area == i {
			a.Rect, a.Radius = e.areaPreview(drag), drag.radius // preview only; no write until release
		}
		if a.Rect.Empty() {
			continue
		}
		if a.Trigger == sel && e.editable() && areaEditable(a) {
			handles = append(handles, e.areaHandleLines(a, a.Rect)...)
		}
		ink := areaInk(a.Trigger)
		lines := e.areaOutline(a.Rect, ink)
		if a.Shape == AreaSquare && a.Radius > 0 {
			x, y := e.cellCenter(a.Centre)
			sx, sy := e.doc.Viewer.cam.WorldToScreen(x, y)
			lines = append(lines, editorLine{float32(sx - 4), float32(sy - 4), float32(sx + 4), float32(sy + 4), ink},
				editorLine{float32(sx - 4), float32(sy + 4), float32(sx + 4), float32(sy - 4), ink})
		}
		if a.Trigger != sel {
			out = append(out, lines...)
			continue
		}
		for _, l := range lines {
			l.color = editorGold
			top = append(top, l, editorLine{l.x0 + 1, l.y0 + 1, l.x1 + 1, l.y1 + 1, editorGold})
		}
	}
	return append(append(out, top...), handles...)
}

func (e *MapEditor) triggerLabelRows() []mapLabel {
	if !e.areasShown() {
		return nil
	}
	ox, oy := e.mapOrigin()
	var out []mapLabel
	done := map[int]bool{}
	sel := e.selectedTrigger()
	for _, a := range e.doc.Script.Areas {
		if done[a.Trigger] || a.Rect.Empty() || a.Trigger >= len(e.doc.Script.Triggers) {
			continue
		}
		done[a.Trigger] = true
		t := e.doc.Script.Triggers[a.Trigger]
		x, y := e.corner(a.Rect.Min)
		ink := areaInk(a.Trigger)
		if a.Trigger == sel {
			ink = editorGold
		}
		out = append(out, mapLabel{
			source: labelTrigger, index: a.Trigger,
			anchor: image.Pt(int(float64(x)+ox), int(float64(y)+oy)), above: true,
			text: fmt.Sprintf("T%d %s", a.Trigger+1, e.editorHostText(t.Name)), maxTextWidth: 160,
			padding: image.Pt(4, 1), ink: ink, border: ink,
			popup: labelPopupLeft, edge: labelInside,
			collides: labelTrigger, moveDown: true, stopAtBottom: true, layer: 2,
		})
	}
	return out
}

// popupRects are the window rectangles of the open list, object and prompt
// popups: a label on the map is left out where one covers it.
func (e *MapEditor) popupRects() []image.Rectangle {
	var popups []image.Rectangle
	if s, ok := e.listSpec(); ok {
		popups = append(popups, layoutPopup(s, e.font, e.bodyFont()).Rect)
	}
	if s, _, ok := e.objectSpec(); ok {
		popups = append(popups, layoutPopup(s, e.font, e.bodyFont()).Rect)
	}
	if s, ok := e.promptSpec(); ok {
		popups = append(popups, layoutPopup(s, e.font, e.bodyFont()).Rect)
	}
	return append(popups, e.panelRects()...)
}

// areaLabelAt is the trigger whose label is under the window point p.
func (e *MapEditor) areaLabelAt(p image.Point) (int, bool) {
	labels := e.mapLabelsOf(labelTrigger)
	for i := len(labels) - 1; i >= 0; i-- {
		if p.In(labels[i].rect) {
			return labels[i].index, true
		}
	}
	return 0, false
}

// controlAreaLabelRect is the window rectangle of trigger t's label on the map.
func (e *MapEditor) controlAreaLabelRect(t int) (image.Rectangle, bool) {
	for _, l := range e.mapLabelsOf(labelTrigger) {
		if l.index == t {
			return l.rect, true
		}
	}
	return image.Rectangle{}, false
}

// areaUnder is the smallest area covering cell, false when no shown area
// does.
func (e *MapEditor) areaUnder(cell image.Point) (InspectionArea, bool) {
	if !e.areasShown() {
		return InspectionArea{}, false
	}
	var best InspectionArea
	found := false
	for _, a := range e.doc.Script.Areas {
		if cell.In(a.Rect) && (!found || a.Rect.Dx()*a.Rect.Dy() < best.Rect.Dx()*best.Rect.Dy()) {
			best, found = a, true
		}
	}
	return best, found
}

// selectTrigger selects trigger t and opens its popup.
func (e *MapEditor) selectTrigger(t int, jump bool) {
	if e.doc.Script != nil && t >= 0 && t < len(e.doc.Script.Triggers) {
		e.Select(e.doc.Script.Triggers[t].Record, jump)
	}
}

// triggerCentre is where the map centres for trigger record id: the middle of
// the box around its areas.
func (e *MapEditor) triggerCentre(id int) (image.Point, bool) {
	v, ok := e.scriptViewOf(id)
	if !ok || v.trigger == nil {
		return image.Point{}, false
	}
	var box image.Rectangle
	for _, a := range e.TriggerAreas(v.index) {
		box = box.Union(a.Rect)
	}
	if box.Empty() {
		return image.Point{}, false
	}
	return image.Pt((box.Min.X+box.Max.X-1)/2, (box.Min.Y+box.Max.Y-1)/2), true
}
