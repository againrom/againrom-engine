package ui

import (
	"fmt"
	"image"
	"math"
)

// The drop-zone tool. A drop zone is a drop cell action the shipped maps
// wrap in a trigger "Deploy Location"; the game picks one zone at random at
// mission start (MISSION-DROP-002). While the tool is armed a click on an
// empty cell adds a zone, a click on a zone selects it, a drag moves it and
// Del deletes the selected zone; each zone shows the cell the hero lands on
// and the square the other units land in (MISSION-SCATTER-053).

func (e *MapEditor) dropZones() (InspectionDropZones, bool) {
	if !e.editable() {
		return nil, false
	}
	z, ok := e.doc.Edits.(InspectionDropZones)
	return z, ok
}

// dropToolOn reports the drop-zone tool armed.
func (e *MapEditor) dropToolOn() bool { return e.pick == pickDrop && e.doc != nil }

// dropZoneAtCell is the zone on cell, the selected one first.
func (e *MapEditor) dropZoneAtCell(cell image.Point) (int, bool) {
	if e.doc == nil {
		return 0, false
	}
	if z := e.dropSel - 1; z >= 0 && z < len(e.doc.Drops) && e.doc.Drops[z] == cell {
		return z, true
	}
	for z, c := range e.doc.Drops {
		if c == cell {
			return z, true
		}
	}
	return 0, false
}

// dropZoneAt is the zone under frame point p.
func (e *MapEditor) dropZoneAt(p image.Point) (int, bool) {
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		return 0, false
	}
	return e.dropZoneAtCell(image.Pt(x, y))
}

// dropAt is a click of the drop tool on cell: a zone there is selected, an
// empty cell takes a new zone. A host without zones moves the one drop cell.
func (e *MapEditor) dropAt(cell image.Point) {
	z, ok := e.dropZones()
	if !ok {
		if s, ok := e.script(); ok {
			next, err := s.SetDropCell(cell)
			_ = e.adoptEdit(next, err)
		}
		return
	}
	e.keepPick(pickDrop) // the tool stays armed until Esc, D or a right click
	if zone, ok := e.dropZoneAtCell(cell); ok {
		e.dropSel = zone + 1
		return
	}
	next, err := z.AddDropZone(cell)
	if e.adoptEdit(next, err) == nil && next != nil {
		e.dropSel = len(e.doc.Drops)
	}
	e.keepPick(pickDrop)
}

// dropPress starts a drag of the zone under p while the tool is armed.
func (e *MapEditor) dropPress(p image.Point) (*editorRecordDrag, bool) {
	if !e.dropToolOn() || e.blocksCanvas() {
		return nil, false
	}
	if _, ok := e.dropZones(); !ok {
		return nil, false
	}
	zone, ok := e.dropZoneAt(p)
	if !ok {
		return nil, false
	}
	cell := e.doc.Drops[zone]
	return &editorRecordDrag{record: -1, index: -1, press: p, cell: cell, origin: cell, drop: zone + 1}, true
}

// dragDrop follows a zone drag to frame point p; the release moves the zone,
// or, without a move, selects it.
func (e *MapEditor) dragDrop(d *editorRecordDrag, in Input, p image.Point, inside bool) bool {
	e.down = in.PrimaryDown
	if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok && inside {
		d.cell = e.clampAreaCell(image.Pt(x, y))
	}
	if math.Abs(float64(p.X-d.press.X)) > 4 || math.Abs(float64(p.Y-d.press.Y)) > 4 {
		d.moved = true
	}
	if in.PrimaryDown {
		return true
	}
	e.recordDrag = nil
	zone := d.drop - 1
	e.dropSel = d.drop
	if !d.moved || d.cell == d.origin {
		return true
	}
	if z, ok := e.dropZones(); ok {
		next, err := z.MoveDropZone(zone, d.cell)
		_ = e.adoptEdit(next, err)
		e.dropSel = d.drop
	}
	return true
}

// deleteDropZone deletes the selected zone; the tool stays armed.
func (e *MapEditor) deleteDropZone() bool {
	z, ok := e.dropZones()
	if !ok || !e.dropToolOn() || e.dropSel <= 0 || e.dropSel > len(e.doc.Drops) {
		return false
	}
	next, err := z.DeleteDropZone(e.dropSel - 1)
	if e.adoptEdit(next, err) == nil && next != nil {
		e.dropSel = 0
	}
	e.keepPick(pickDrop)
	return true
}

// dropScatterRadius is the radius the other units land within around a
// drop cell for n units: trunc(max(5, sqrt(n)+4)), its low byte
// (MISSION-SCATTER-053).
func dropScatterRadius(n int) int {
	return int(uint8(max(5, int(math.Sqrt(float64(max(0, n))))+4)))
}

// dropScatter is the cells a unit other than the hero may land on around
// cell: each axis offset is -r/2 plus a draw of 0..r (MISSION-SCATTER-053).
func dropScatter(cell image.Point, r int) image.Rectangle {
	return image.Rect(cell.X-r/2, cell.Y-r/2, cell.X-r/2+r+1, cell.Y-r/2+r+1)
}

// dropPopulation is the unit count the tool's square is drawn for: the
// units placed for player 1 plus the hero.
func (e *MapEditor) dropPopulation() int {
	n := 1
	for _, r := range e.doc.Records {
		if r.Kind == "Unit" && r.Player == 1 {
			n++
		}
	}
	return n
}

// DropToolRadius is the radius of the square the drop tool shows.
func (e *MapEditor) DropToolRadius() int {
	if e.doc == nil {
		return 0
	}
	return dropScatterRadius(e.dropPopulation())
}

// DropToolOn reports the drop-zone tool armed, and its selected zone, -1 for
// none.
func (e *MapEditor) DropToolOn() (bool, int) {
	return e.dropToolOn(), e.dropSel - 1
}

// dropToolLines draw, while the tool is armed, each zone's hero cell and the
// square the others land in, the selected zone in gold.
func (e *MapEditor) dropToolLines() []editorLine {
	if !e.dropToolOn() || e.doc.Viewer == nil {
		return nil
	}
	r := e.DropToolRadius()
	bounds := image.Rect(0, 0, e.doc.Width, e.doc.Height)
	var out, top []editorLine
	for z, cell := range e.doc.Drops {
		if d := e.recordDrag; d != nil && d.drop == z+1 {
			cell = d.cell // preview only; the release writes it
		}
		ink := editorCondition
		if z == e.dropSel-1 {
			ink = editorGold
		}
		hero := image.Rect(cell.X, cell.Y, cell.X+1, cell.Y+1).Intersect(bounds)
		lines := e.areaOutline(dropScatter(cell, r).Intersect(bounds), ink)
		if !hero.Empty() {
			lines = append(lines, e.areaOutline(hero, ink)...)
		}
		if ink == editorGold {
			for _, l := range lines {
				top = append(top, l, editorLine{l.x0 + 1, l.y0 + 1, l.x1 + 1, l.y1 + 1, ink})
			}
			continue
		}
		out = append(out, lines...)
	}
	return append(out, top...)
}

// dropToolStatus is the status line of the armed tool.
func (e *MapEditor) dropToolStatus() string {
	if _, ok := e.dropZones(); !ok {
		return e.word("editor.keys.dropcell")
	}
	return fmt.Sprintf(e.word("editor.drop.tool"), len(e.doc.Drops), e.DropToolRadius())
}
