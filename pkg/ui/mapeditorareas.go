package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

// The triggers on the map: every place a trigger's conditions and actions
// name is outlined in that trigger's colour and labelled with its number and
// name. The selected trigger's areas are gold; a click on an area or its
// label selects the trigger. The View menu shows or hides them.

// areaInks tell overlapping triggers apart; gold is the selection's.
var areaInks = []color.RGBA{
	{0x4f, 0xd6, 0xe0, 255}, {0xe0, 0x6f, 0xd6, 255}, {0x8c, 0xe0, 0x4f, 255}, {0xf0, 0x9a, 0x3e, 255},
	{0x7a, 0x9c, 0xff, 255}, {0xff, 0x7a, 0x8a, 255}, {0xd8, 0xe0, 0x6a, 255}, {0xb0, 0x86, 0xff, 255},
}

func areaInk(trigger int) color.RGBA { return areaInks[trigger%len(areaInks)] }

// areasShown reports that the View menu shows the trigger areas.
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
		return e.backTrigger
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
	if !e.areasShown() {
		return nil
	}
	sel := e.selectedTrigger()
	var out, top []editorLine
	for _, a := range e.doc.Script.Areas {
		if a.Rect.Empty() {
			continue
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
	return append(out, top...)
}

// areaLabel is one trigger's label on the map: where it stands in the
// window, its text and its trigger.
type areaLabel struct {
	trigger int
	rect    image.Rectangle
	text    string
}

// areaLabels place one label per trigger at the top left corner of its first
// area, moved down while it would cover a label already placed and left of
// an open popup it would run under; a label a popup still covers is left
// out.
func (e *MapEditor) areaLabels() []areaLabel {
	if !e.areasShown() {
		return nil
	}
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
	f := e.bodyFont()
	h := fontHeight(f, 15) + 2
	area := e.mapArea()
	ox, oy := e.doc.Viewer.place.Origin()
	var out []areaLabel
	done := map[int]bool{}
	for _, a := range e.doc.Script.Areas {
		if done[a.Trigger] || a.Rect.Empty() || a.Trigger >= len(e.doc.Script.Triggers) {
			continue
		}
		done[a.Trigger] = true
		t := e.doc.Script.Triggers[a.Trigger]
		text := editorClip(f, fmt.Sprintf("T%d %s", a.Trigger+1, e.editorHostText(t.Name)), 160)
		w := 8
		if f != nil {
			w += f.Advance(text)
		}
		x, y := e.corner(a.Rect.Min)
		r := image.Rect(int(float64(x)+ox), int(float64(y)+oy)-h, int(float64(x)+ox)+w, int(float64(y)+oy))
		for _, p := range popups {
			if p.Overlaps(r) && r.Min.X < p.Min.X {
				r = r.Sub(image.Pt(r.Max.X-p.Min.X, 0)) // to the popup's left, beside its area
			}
		}
		for moved := 0; moved < 8; moved++ {
			clash := false
			for _, l := range out {
				clash = clash || l.rect.Overlaps(r)
			}
			if !clash {
				break
			}
			r = r.Add(image.Pt(0, h))
		}
		covered := false
		for _, p := range popups {
			covered = covered || p.Overlaps(r)
		}
		if r.In(area) && !covered {
			out = append(out, areaLabel{trigger: a.Trigger, rect: r, text: text})
		}
	}
	return out
}

// drawAreaLabels paints the labels on the chrome picture, under the popups.
func (e *MapEditor) drawAreaLabels(pic *image.RGBA) {
	sel := e.selectedTrigger()
	f := e.bodyFont()
	for _, l := range e.areaLabels() {
		ink := areaInk(l.trigger)
		if l.trigger == sel {
			ink = editorGold
		}
		draw.Draw(pic, l.rect, image.NewUniform(editorInk), image.Point{}, draw.Src)
		drawFrame(pic, panelFrame(l.rect, editorInk, ink))
		if f != nil {
			f.Draw(pic.SubImage(l.rect).(*image.RGBA), l.text, l.rect.Min.X+4, l.rect.Min.Y+1, ink)
		}
	}
}

// areaLabelAt is the trigger whose label is under the window point p.
func (e *MapEditor) areaLabelAt(p image.Point) (int, bool) {
	labels := e.areaLabels()
	for i := len(labels) - 1; i >= 0; i-- {
		if p.In(labels[i].rect) {
			return labels[i].trigger, true
		}
	}
	return 0, false
}

// AreaLabelRect is the window rectangle of trigger t's label on the map.
func (e *MapEditor) AreaLabelRect(t int) (image.Rectangle, bool) {
	for _, l := range e.areaLabels() {
		if l.trigger == t {
			return l.rect, true
		}
	}
	return image.Rectangle{}, false
}

// areaAt is the trigger of the smallest area covering cell, false when no
// area does.
func (e *MapEditor) areaAt(cell image.Point) (int, bool) {
	if !e.areasShown() {
		return 0, false
	}
	best, size := -1, 0
	for _, a := range e.doc.Script.Areas {
		if cell.In(a.Rect) && (best < 0 || a.Rect.Dx()*a.Rect.Dy() < size) {
			best, size = a.Trigger, a.Rect.Dx()*a.Rect.Dy()
		}
	}
	return best, best >= 0
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
