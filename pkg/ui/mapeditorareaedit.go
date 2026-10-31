package ui

import (
	"fmt"
	"image"
	"math"
	"strings"

	"againrom/pkg/intmath"
)

// Trigger areas edited on the map. With the areas shown, the selected
// trigger's areas carry handles: a box's corners and edges, a square's rim
// and a cell's body. A press on a handle starts the record drag carrying it;
// the release writes the parameters the area comes from as one edit, after a
// notice when other triggers or pairs share what it changes. The draw tool
// drags a new box and asks which trigger takes it.

// InspectionAreaAuthor is implemented by edits whose host adds an area drawn
// on the map: a box condition over box's cells (Max exclusive) joined to
// trigger as one more pair, or to a new trigger when trigger is -1, as one
// edit.
type InspectionAreaAuthor interface {
	AddBoxArea(box image.Rectangle, trigger int) (*InspectionDocument, error)
}

type areaHandleKind uint8

const (
	handleCorner     areaHandleKind = iota + 1 // a box corner: two sides
	handleEdge                                 // a box edge: one side
	handleRim                                  // a square's rim: its radius
	handleCell                                 // a cell: the whole cell
	handleDraw                                 // a new box from the draw tool
	handleDrawSquare                           // a new square for the wizard: centre and radius
)

// The sides of a box a corner or an edge moves, as bits.
const (
	sideLeft uint8 = 1 << iota
	sideTop
	sideRight
	sideBottom
)

// handleReach is how far from a handle, in frame pixels, a press takes it.
const handleReach = 5

// maxAreaRadius is the widest square radius the area model draws.
const maxAreaRadius = 255

// areaHandle is what an area drag holds: its handle, its area and the shape
// the drag gives the area so far. box is a box's X, Y, X2, Y2 inclusive or a
// cell's X, Y; from and fromRadius are the shape before the drag.
type areaHandle struct {
	kind               areaHandleKind
	area               int // the index in the script model's areas; -1 drawing
	sides              uint8
	box, from          [4]int
	radius, fromRadius int
}

// areaPrompt is what an area prompt holds: the drawn box with the trigger
// drop-down, or the edit a shared-node notice holds back.
type areaPrompt struct {
	box    image.Rectangle
	choice *Picker // a new trigger first, then each trigger
	open   bool
	note   string
	kind   string
	index  int
	values []InspectionFieldValue
	notes  []string
}

// areaEditable reports that a drag can rewrite area a: its cells come from
// parameter slots, and a square's radius from one.
func areaEditable(a InspectionArea) bool {
	cells := a.Slots[0] >= 0 && a.Slots[1] == a.Slots[0]+1 && a.Slots[1] < 10
	switch a.Shape {
	case AreaBox:
		return cells && a.Slots[2] >= 0 && a.Slots[3] == a.Slots[2]+1 && a.Slots[3] < 10
	case AreaSquare:
		return cells && a.RadiusNode >= 0 && a.RadiusSlot >= 0 && a.RadiusSlot < 10
	case AreaCell:
		return cells
	}
	return false
}

// areaNode is the node of document record rec.
func (e *MapEditor) areaNode(rec int) (*InspectionScriptNode, bool) {
	v, ok := e.scriptViewOf(rec)
	if !ok || v.node == nil {
		return nil, false
	}
	return v.node, true
}

// areaHandleFrom is a handle on area i holding the area as its parameters
// give it.
func (e *MapEditor) areaHandleFrom(i int) (*areaHandle, bool) {
	a := e.doc.Script.Areas[i]
	n, ok := e.areaNode(a.Node)
	if !ok {
		return nil, false
	}
	h := &areaHandle{area: i, fromRadius: a.Radius}
	for k, s := range a.Slots {
		if s >= 0 && s < 10 {
			h.from[k] = int(min(n.Values[s], 1<<16))
		}
	}
	if a.Shape == AreaSquare {
		if _, ok := e.areaNode(a.RadiusNode); !ok {
			return nil, false
		}
	}
	h.box, h.radius = h.from, h.fromRadius
	return h, true
}

// vertexDist is the distance in frame pixels from p to grid corner c.
func (e *MapEditor) vertexDist(p image.Point, c image.Point) float64 {
	x, y := e.corner(c)
	return math.Hypot(float64(p.X)-float64(x), float64(p.Y)-float64(y))
}

// sideDist is the distance in frame pixels from p to side s of the cells r
// covers, along the cell corners the outline follows.
func (e *MapEditor) sideDist(p image.Point, r image.Rectangle, s uint8) float64 {
	var a, b image.Point
	switch s {
	case sideLeft:
		a, b = r.Min, image.Pt(r.Min.X, r.Max.Y)
	case sideTop:
		a, b = r.Min, image.Pt(r.Max.X, r.Min.Y)
	case sideRight:
		a, b = image.Pt(r.Max.X, r.Min.Y), r.Max
	default:
		a, b = image.Pt(r.Min.X, r.Max.Y), r.Max
	}
	step := image.Pt(sign(b.X-a.X), sign(b.Y-a.Y))
	best := math.Inf(1)
	x0, y0 := e.corner(a)
	for c := a; c != b; {
		c = c.Add(step)
		x1, y1 := e.corner(c)
		best = math.Min(best, segmentDist(float64(p.X), float64(p.Y), float64(x0), float64(y0), float64(x1), float64(y1)))
		x0, y0 = x1, y1
	}
	return best
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// segmentDist is the distance from (px, py) to the segment (ax, ay)-(bx, by).
func segmentDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l))
	}
	return math.Hypot(px-ax-t*dx, py-ay-t*dy)
}

var boxSides = []uint8{sideLeft, sideTop, sideRight, sideBottom}

// areaHandleAt is the handle of the selected trigger's areas under frame
// point p: a corner before an edge or a rim, either before a cell's body,
// the nearest of a kind.
func (e *MapEditor) areaHandleAt(p image.Point) (*areaHandle, bool) {
	sel := e.selectedTrigger()
	if !e.areasShown() || sel < 0 {
		return nil, false
	}
	cx, cy, cellOK := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	var best *areaHandle
	bestRank, bestDist := 3, math.Inf(1)
	consider := func(h areaHandle, kind areaHandleKind, sides uint8, rank int, d float64) {
		if rank < bestRank || rank == bestRank && d < bestDist {
			h.kind, h.sides = kind, sides
			best, bestRank, bestDist = &h, rank, d
		}
	}
	for i, a := range e.doc.Script.Areas {
		if a.Trigger != sel || a.Rect.Empty() || !areaEditable(a) {
			continue
		}
		h, ok := e.areaHandleFrom(i)
		if !ok {
			continue
		}
		r := a.Rect
		switch a.Shape {
		case AreaBox:
			for _, c := range []struct {
				at    image.Point
				sides uint8
			}{{r.Min, sideLeft | sideTop}, {image.Pt(r.Max.X, r.Min.Y), sideRight | sideTop}, {r.Max, sideRight | sideBottom}, {image.Pt(r.Min.X, r.Max.Y), sideLeft | sideBottom}} {
				if d := e.vertexDist(p, c.at); d <= handleReach+2 {
					consider(*h, handleCorner, c.sides, 0, d)
				}
			}
			for _, s := range boxSides {
				if d := e.sideDist(p, r, s); d <= handleReach {
					consider(*h, handleEdge, s, 1, d)
				}
			}
		case AreaSquare:
			for _, s := range boxSides {
				if d := e.sideDist(p, r, s); d <= handleReach {
					consider(*h, handleRim, 0, 1, d)
				}
			}
		case AreaCell:
			if cellOK && image.Pt(cx, cy) == r.Min {
				consider(*h, handleCell, 0, 2, 0)
			}
		}
	}
	return best, best != nil
}

// areaPress starts an area drag at frame point p: the draw tool's box, or a
// handle of the selected trigger's areas. A trigger popup with unapplied
// changes refuses a handle.
func (e *MapEditor) areaPress(p image.Point) (*areaHandle, bool) {
	if e.pick == pickArea {
		x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
		if !ok || e.blocksCanvas() {
			return nil, false
		}
		c := e.clampAreaCell(image.Pt(x, y))
		from := [4]int{c.X, c.Y, c.X, c.Y}
		if w, ok := e.wizardDrawing(); ok && w.square {
			return &areaHandle{kind: handleDrawSquare, area: -1, box: from, from: from}, true
		}
		return &areaHandle{kind: handleDraw, area: -1, box: from, from: from}, true
	}
	if !e.free() {
		return nil, false
	}
	h, ok := e.areaHandleAt(p)
	if !ok {
		return nil, false
	}
	if e.objectDirty() {
		e.message = e.word("editor.trigger.applyfirst")
		return nil, false
	}
	return h, true
}

// clampAreaCell keeps c on the map and inside the 0..255 an X or Y
// parameter holds.
func (e *MapEditor) clampAreaCell(c image.Point) image.Point {
	return image.Pt(max(0, min(c.X, min(e.doc.Width, 256)-1)), max(0, min(c.Y, min(e.doc.Height, 256)-1)))
}

// areaDragTo gives the dragged area the shape its handle at cell c makes,
// clamped to the map, a box's corners kept in order and a radius to
// 0..255.
func (e *MapEditor) areaDragTo(h *areaHandle, c image.Point) {
	c = e.clampAreaCell(c)
	switch h.kind {
	case handleCorner, handleEdge:
		b := h.from
		if h.sides&sideLeft != 0 {
			b[0] = min(c.X, b[2])
		}
		if h.sides&sideRight != 0 {
			b[2] = max(c.X, b[0])
		}
		if h.sides&sideTop != 0 {
			b[1] = min(c.Y, b[3])
		}
		if h.sides&sideBottom != 0 {
			b[3] = max(c.Y, b[1])
		}
		h.box = b
	case handleRim:
		ctr := e.doc.Script.Areas[h.area].Centre
		h.radius = min(maxAreaRadius, max(intmath.Abs(c.X-ctr.X), intmath.Abs(c.Y-ctr.Y)))
	case handleCell:
		h.box = [4]int{c.X, c.Y, -1, -1}
	case handleDraw:
		h.box = [4]int{min(h.from[0], c.X), min(h.from[1], c.Y), max(h.from[0], c.X), max(h.from[1], c.Y)}
	case handleDrawSquare:
		h.radius = min(maxAreaRadius, max(intmath.Abs(c.X-h.from[0]), intmath.Abs(c.Y-h.from[1])))
	}
}

// areaPreview is the cells the dragged area covers now, clipped to the map.
func (e *MapEditor) areaPreview(h *areaHandle) image.Rectangle {
	var r image.Rectangle
	switch h.kind {
	case handleRim:
		c := e.doc.Script.Areas[h.area].Centre
		r = image.Rect(c.X-h.radius, c.Y-h.radius, c.X+h.radius+1, c.Y+h.radius+1)
	case handleCell:
		r = image.Rect(h.box[0], h.box[1], h.box[0]+1, h.box[1]+1)
	case handleDrawSquare:
		c := image.Pt(h.from[0], h.from[1])
		r = image.Rect(c.X-h.radius, c.Y-h.radius, c.X+h.radius+1, c.Y+h.radius+1)
	default:
		r = image.Rect(h.box[0], h.box[1], h.box[2]+1, h.box[3]+1)
	}
	return r.Intersect(image.Rect(0, 0, e.doc.Width, e.doc.Height))
}

// areaDrag is the area drag in progress, nil for none.
func (e *MapEditor) areaDrag() *areaHandle {
	if e.recordDrag == nil {
		return nil
	}
	return e.recordDrag.area
}

// dragArea follows an area drag to frame point p: the cell under the
// pointer reshapes the area, off the map the shape holds, and the release
// writes it or, without a move, is a click.
func (e *MapEditor) dragArea(d *editorRecordDrag, in Input, p image.Point, inside bool) bool {
	h := d.area
	e.down = in.PrimaryDown
	if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok && inside {
		e.areaDragTo(h, image.Pt(x, y))
	}
	if math.Abs(float64(p.X-d.press.X)) > 4 || math.Abs(float64(p.Y-d.press.Y)) > 4 {
		d.moved = true
	}
	if in.PrimaryDown {
		return true
	}
	e.recordDrag = nil
	switch {
	case h.kind == handleDrawSquare:
		e.wizardAreaDrawn(h)
	case h.kind == handleDraw:
		if _, ok := e.wizardDrawing(); ok {
			e.wizardAreaDrawn(h)
			return true
		}
		e.areaDrawn(h)
	case d.moved:
		e.areaRelease(h)
	default:
		e.selectAt(p)
	}
	return true
}

// areaRelease writes the dragged area's parameters: a box's or a cell's
// coordinates in its node, a radius in the node or constant that holds it.
// A change that reaches other triggers or pairs waits for the notice.
func (e *MapEditor) areaRelease(h *areaHandle) {
	a := e.doc.Script.Areas[h.area]
	rec := a.Node
	var values []InspectionFieldValue
	if h.kind == handleRim {
		if h.radius == h.fromRadius {
			return
		}
		rec = a.RadiusNode
		values = append(values, InspectionFieldValue{Key: fmt.Sprintf("value%d", a.RadiusSlot), Value: int64(h.radius + a.RadiusBias)})
	} else {
		for k, s := range a.Slots {
			if s >= 0 && h.box[k] != h.from[k] {
				values = append(values, InspectionFieldValue{Key: fmt.Sprintf("value%d", s), Value: int64(h.box[k])})
			}
		}
	}
	if len(values) == 0 {
		return
	}
	r := e.doc.Records[rec]
	if note, shared := e.areaShared(rec, h.kind == handleRim && rec != a.Node, a.RadiusSlot); shared {
		e.openPrompt(editorPrompt{kind: promptAreaShared, area: &areaPrompt{kind: r.Kind, index: r.Index, values: values, notes: []string{note}}})
		return
	}
	e.writeArea(r.Kind, r.Index, values)
}

// writeArea writes an area's parameters through the host's one-edit seam.
func (e *MapEditor) writeArea(kind string, index int, values []InspectionFieldValue) {
	if _, err := e.commandSetFields(recordFields{Kind: kind, Index: index, Values: values}); err != nil {
		e.message = err.Error()
	}
}

// areaShared is the notice a change to node rec needs: a node more than one
// trigger names, or a constant (its value in slot) more than one pair
// compares with; false when none is needed.
func (e *MapEditor) areaShared(rec int, constant bool, slot int) (string, bool) {
	v, ok := e.scriptViewOf(rec)
	if !ok || v.node == nil {
		return "", false
	}
	id := v.node.ID
	var triggers []string
	uses := 0
	for t, tr := range e.doc.Script.Triggers {
		n := 0
		if v.condition {
			for k := range tr.Left {
				n += refCount(tr.Left[k] == id) + refCount(tr.Right[k] == id)
			}
		} else {
			for _, act := range tr.Acts {
				n += refCount(act == id)
			}
		}
		if n > 0 {
			triggers = append(triggers, fmt.Sprintf("T%d", t+1))
			uses += n
		}
	}
	list := strings.Join(triggers, ", ")
	if constant {
		if uses <= 1 || slot < 0 || slot >= 10 {
			return "", false
		}
		return fmt.Sprintf(e.word("editor.area.sharedconst"), int32(v.node.Values[slot]), list, uses), true
	}
	if len(triggers) <= 1 {
		return "", false
	}
	return fmt.Sprintf(e.word("editor.area.sharednode"), list), true
}

// refCount is 1 for a reference, 0 for none.
func refCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// The shared-node notice's buttons, in order.
const (
	areaSharedGo = iota
	areaSharedStay
)

// areaSharedSpec is the notice before a change reaches other triggers or
// pairs.
func (e *MapEditor) areaSharedSpec() popupSpec {
	return e.questionSpec(e.word("editor.area.sharedtitle"), e.prompt.area.notes,
		[]string{e.word("editor.area.change"), e.word("editor.button.cancel")})
}

// areaSharedPress is a press on the shared-node notice.
func (e *MapEditor) areaSharedPress(p image.Point) {
	s, _ := e.promptSpec()
	part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == areaSharedStay:
		e.cancelPopup()
	case part == popupButton && n == areaSharedGo:
		e.areaSharedContinue()
	}
}

// areaSharedContinue leaves the notice and writes the edit it held.
func (e *MapEditor) areaSharedContinue() {
	a := e.prompt.area
	e.leave(leavePrompt)
	e.writeArea(a.kind, a.index, a.values)
}

// keyDrawArea arms the draw tool, or disarms it; the areas show while it is
// armed.
func (e *MapEditor) keyDrawArea(string) error {
	if e.obj.wiz != nil {
		e.obj.wiz.draw = -1
	}
	if e.pick == pickArea {
		e.leave(leavePick)
		return nil
	}
	if e.doc == nil {
		return nil
	}
	if _, ok := e.doc.Edits.(InspectionAreaAuthor); !ok || !e.editable() || e.doc.Script == nil {
		e.message = e.word("editor.area.noscript")
		return nil
	}
	if p, ok := e.mapPanel(); ok && p.MapFacts().Mode > 1 {
		e.message = e.word("editor.area.multiplayer")
		return nil
	}
	e.leave(leaveDrag | leavePrompt | leaveReport | leaveBrush)
	e.view.hideAreas = false
	e.armPick(pickArea)
	return nil
}

// areaDrawn asks which trigger takes the box the draw tool drew: a new
// trigger first, then each trigger by number and name.
func (e *MapEditor) areaDrawn(h *areaHandle) {
	e.leave(leavePick)
	rows := []PickerRow{{Text: e.word("editor.area.newtrigger"), Choosable: true}}
	for i, t := range e.doc.Script.Triggers {
		rows = append(rows, PickerRow{Text: fmt.Sprintf("T%d %s", i+1, e.editorHostText(t.Name)), Choosable: true})
	}
	choice := NewPicker(rows).SetWindow(min(10, len(rows)))
	choice.Select(0)
	box := image.Rect(h.box[0], h.box[1], h.box[2]+1, h.box[3]+1)
	e.openPrompt(editorPrompt{kind: promptAreaTrigger, area: &areaPrompt{box: box, choice: choice}})
}

// The trigger choice's buttons, in order.
const (
	areaCreateGo = iota
	areaCreateStay
)

// areaTriggerSpec is the trigger choice: what the box becomes, the trigger
// drop-down and Create and Cancel.
func (e *MapEditor) areaTriggerSpec() popupSpec {
	a := e.prompt.area
	b := a.box
	notes := []string{fmt.Sprintf(e.word("editor.area.newtext"), b.Min.X, b.Min.Y, b.Max.X-1, b.Max.Y-1)}
	if a.note != "" {
		notes = append(notes, a.note)
	}
	s := e.questionSpec(e.word("editor.area.newtitle"), notes, []string{e.word("editor.area.create"), e.word("editor.button.cancel")})
	label := e.word("editor.area.trigger")
	s.LabelWidth = 120
	if f := e.bodyFont(); f != nil {
		s.LabelWidth = f.Advance(label) + 24
	}
	value := ""
	if rows := a.choice.Rows(); a.choice.Selection() >= 0 && a.choice.Selection() < len(rows) {
		value = editorClip(e.bodyFont(), rows[a.choice.Selection()].Text, s.Width-s.LabelWidth-60)
	}
	s.Rows = []popupRow{{Kind: popupChoice, Label: label, Value: value, Focus: true}}
	l := layoutPopup(s, e.font, e.bodyFont())
	s.At = e.promptPlace(image.Pt((e.window.X-l.Rect.Dx())/2, max(e.barHeight()+8, (e.window.Y-l.Rect.Dy())/3)))
	return s
}

// areaDropdown is the trigger choice's drop-down on its row.
func (e *MapEditor) areaDropdown() (dropdown, bool) {
	if e.prompt.kind != promptAreaTrigger || e.prompt.area == nil {
		return dropdown{}, false
	}
	a := e.prompt.area
	s := e.sizedPrompt(e.areaTriggerSpec())
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if rr.Index == 0 {
			return dropdown{Rect: rr.Control, Text: s.Rows[0].Value, Open: a.open, List: a.choice, Rows: min(10, a.choice.Len()), Focus: true, Fill: editorPanel, Ink: editorLight}, true
		}
	}
	return dropdown{}, false
}

// drawAreaDropdown paints the trigger choice's open list over the prompt.
func (e *MapEditor) drawAreaDropdown(pic *image.RGBA) {
	d, ok := e.areaDropdown()
	if !ok || !d.Open {
		return
	}
	rows := d.List.Rows()
	drawDropdown(pic, e.bodyFont(), nil, d, func(row, width int) string { return editorClip(e.bodyFont(), rows[row].Text, width) }, e.pointer, e.pointerOK)
}

// areaDropdownHit reports that p lies on the trigger choice's open list.
func (e *MapEditor) areaDropdownHit(p image.Point) bool {
	d, ok := e.areaDropdown()
	return ok && d.Open && p.In(d.Bounds(e.bodyFont()))
}

// areaTriggerPress is a press on the trigger choice: the drop-down opens,
// chooses and closes; Create adds the area, Cancel and the close button
// leave.
func (e *MapEditor) areaTriggerPress(p image.Point) {
	a := e.prompt.area
	if d, ok := e.areaDropdown(); ok {
		switch part, row := d.hit(e.bodyFont(), p); part {
		case dropdownToggle:
			a.open = !a.open
			return
		case dropdownRow:
			a.choice.Select(row)
			a.open, a.note = false, ""
			return
		case dropdownBar:
			return
		}
		a.open = false
	}
	s, _ := e.promptSpec()
	part, n := layoutPopup(s, e.font, e.bodyFont()).hit(p)
	switch {
	case part == popupClose || part == popupButton && n == areaCreateStay:
		e.cancelPopup()
	case part == popupButton && n == areaCreateGo:
		e.areaCreate()
	}
}

// areaStep moves the trigger choice by delta: the arrow keys.
func (e *MapEditor) areaStep(delta int) {
	a := e.prompt.area
	editorListStep(a.choice, delta)
	a.note = ""
}

// areaWheel scrolls the trigger choice's open list.
func (e *MapEditor) areaWheel(wheel float64) bool {
	if !e.areaDropdownHit(e.pointer) {
		return false
	}
	e.prompt.area.choice.Move(editorWheelRows(&e.listWheel, wheel))
	return true
}

// areaCreate adds the drawn box to the chosen trigger, or to a new one, as
// one edit, then selects that trigger. A trigger with no free pair is
// refused in the prompt.
func (e *MapEditor) areaCreate() {
	a := e.prompt.area
	if _, ok := e.doc.Edits.(InspectionAreaAuthor); !ok {
		return
	}
	trigger := a.choice.Selection() - 1
	if trigger >= 0 && trigger < len(e.doc.Script.Triggers) && !freePair(e.doc.Script.Triggers[trigger]) {
		a.note = fmt.Sprintf(e.word("editor.area.full"), trigger+1)
		return
	}
	ch, err := e.commandAddBoxArea(boxArea{Box: a.box, Trigger: trigger})
	if err != nil {
		a.note = e.editorHostText(err.Error())
		return
	}
	e.leave(leavePrompt)
	if ch.record != nil {
		e.selectTrigger(ch.record.Index, false)
	}
}

// freePair reports that trigger t has an empty condition pair.
func freePair(t InspectionTrigger) bool {
	for k := range t.Left {
		if t.Left[k] == 0 && t.Right[k] == 0 {
			return true
		}
	}
	return false
}

// areaHandleLines mark the handles of area a: a box's corners and edge
// middles, a square's rim middles, a cell's centre.
func (e *MapEditor) areaHandleLines(a InspectionArea, r image.Rectangle) []editorLine {
	var pts [][2]float32
	vertex := func(c image.Point) {
		x, y := e.corner(c)
		pts = append(pts, [2]float32{x, y})
	}
	mid := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	switch a.Shape {
	case AreaBox:
		for _, c := range []image.Point{r.Min, {r.Max.X, r.Min.Y}, r.Max, {r.Min.X, r.Max.Y}} {
			vertex(c)
		}
		fallthrough
	case AreaSquare:
		for _, c := range []image.Point{{mid.X, r.Min.Y}, {r.Max.X, mid.Y}, {mid.X, r.Max.Y}, {r.Min.X, mid.Y}} {
			vertex(c)
		}
	case AreaCell:
		x, y := e.cellCenter(r.Min)
		sx, sy := e.doc.Viewer.cam.WorldToScreen(x, y)
		pts = append(pts, [2]float32{float32(sx), float32(sy)})
	}
	var out []editorLine
	for _, p := range pts {
		x, y := p[0], p[1]
		out = append(out, editorLine{x - 3, y - 3, x + 3, y - 3, editorPaper}, editorLine{x + 3, y - 3, x + 3, y + 3, editorPaper},
			editorLine{x + 3, y + 3, x - 3, y + 3, editorPaper}, editorLine{x - 3, y + 3, x - 3, y - 3, editorPaper})
	}
	return out
}

// AreaCornerPoint is the window point of grid corner c, for a headless
// driver.
func (e *MapEditor) AreaCornerPoint(c image.Point) image.Point {
	x, y := e.corner(c)
	ox, oy := e.mapOrigin()
	return image.Pt(int(math.Round(float64(x)+ox)), int(math.Round(float64(y)+oy)))
}

// AreaTriggerRect is the window rectangle of the trigger choice's box.
func (e *MapEditor) AreaTriggerRect() (image.Rectangle, bool) {
	d, ok := e.areaDropdown()
	return d.Rect, ok
}

// AreaDrawArmed reports that the draw tool waits for a drag on the map.
func (e *MapEditor) AreaDrawArmed() bool { return e.pick == pickArea }

// openAreaDropdown is the trigger choice's drop-down while its list is open.
func (e *MapEditor) openAreaDropdown() (dropdown, bool) {
	d, ok := e.areaDropdown()
	return d, ok && d.Open
}
