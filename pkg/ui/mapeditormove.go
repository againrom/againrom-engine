package ui

import (
	"image"
	"image/color"
	"math"
)

type editorModalRow struct {
	text  string
	y     int
	color color.RGBA
}

// Both long paths and errors have a bounded viewport. Keep the editable tail
// and caret visible; reserve separate rows for refusal text even at 640x480.
func (e *MapEditor) modalLayout(h int) []editorModalRow {
	height := 16
	if e.font != nil {
		height = max(height, e.font.Height()+2)
	}
	y := e.listBottom() + 40
	var rows []editorModalRow
	add := func(s string, c color.RGBA) {
		rows = append(rows, editorModalRow{editorClip(e.font, s, 300), y, c})
		y += height
	}
	if e.pathMode {
		title := "ALM path / Enter: open"
		if e.pathSave {
			title = "New .alm path / Enter: Save As"
		}
		add(title, editorGold)
	}
	available := max(1, (h-88-y)/height)
	var errors []string
	if e.message != "" {
		errors = editorWrap(e.font, e.editorHostText(e.message), 300)
	}
	if e.pathMode {
		reserved := min(len(errors), max(1, available/2))
		paths := editorWrap(e.font, e.editorHostText(e.path)+"_", 300)
		n := max(1, min(6, available-reserved))
		if len(paths) > n {
			paths = paths[len(paths)-n:]
			paths[0] = "..." + paths[0]
			for e.font != nil && len(paths[0]) > 3 && e.font.Advance(paths[0]) > 300 {
				paths[0] = "..." + paths[0][4:]
			}
		}
		for _, s := range paths {
			add(s, editorLight)
		}
		available -= len(paths)
	}
	for i := 0; i < min(available, len(errors)); i++ {
		s := errors[i]
		if i == available-1 && i+1 < len(errors) {
			s = editorClip(e.font, s, 280) + "..."
		}
		add(s, editorWarning)
	}
	return rows
}

func (e *MapEditor) editable() bool { return e.doc != nil && e.doc.Edits != nil }
func (e *MapEditor) Dirty() bool    { return e.editable() && e.doc.Edits.Dirty() }

// TextInput is shared by the native path field and deterministic input drives.
func (e *MapEditor) TextInput(s string) {
	if e.pathMode {
		e.path += s
	}
}

func (e *MapEditor) saveAs(path string) error {
	if !e.editable() {
		return nil
	}
	source, err := e.doc.Edits.SaveAs(path)
	if err != nil {
		e.message = err.Error()
		return err
	}
	e.doc.Source = source
	e.pathMode, e.pathSave = false, false
	e.message = ""
	return nil
}

func (e *MapEditor) adoptEdit(next *InspectionDocument, err error) error {
	if err != nil {
		e.message = err.Error()
		return err
	}
	if next == nil {
		return nil // unavailable history or same-cell move
	}
	// Unit movement does not change terrain, structures or sacks. Retain the
	// live viewer (camera, GPU caches and render settings) and refresh entities.
	old := e.doc.Viewer
	old.SetEntities(next.Viewer.entities)
	next.Viewer = old
	var kind string
	index := -1
	if e.selected >= 0 && e.selected < len(e.doc.Records) {
		r := e.doc.Records[e.selected]
		kind, index = r.Kind, r.Index
	}
	e.doc = next
	e.selected = -1
	for i, r := range next.Records {
		if r.Kind == kind && r.Index == index {
			e.selected = i
			break
		}
	}
	e.message = ""
	return nil
}

type editorUnitDrag struct {
	record, index int
	press         image.Point
	grab, cell    image.Point
	moved         bool
}

func (e *MapEditor) cancelUnitDrag() {
	if e.unitDrag != nil {
		e.unitDrag = nil
		e.cancelledDrag = e.down
	}
	if e.doc != nil {
		e.doc.Viewer.dragging = false
	}
}

func (e *MapEditor) hitsAt(p image.Point) []int {
	cx, cy, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		return nil
	}
	var hits []int
	for i, r := range e.doc.Records {
		if r.Spatial && image.Pt(cx, cy).In(image.Rectangle{Min: r.Cell, Max: r.Cell.Add(r.Size)}) {
			hits = append(hits, i)
		}
	}
	return hits
}

func (e *MapEditor) selectAt(p image.Point) {
	hits := e.hitsAt(p)
	if len(hits) == 0 {
		e.selected = -1
		return
	}
	chosen := hits[0]
	for i, id := range hits {
		if id == e.selected {
			chosen = hits[(i+1)%len(hits)]
			break
		}
	}
	e.Select(chosen, false)
}

// Capture units before Viewer.step: otherwise the same primary drag pans the
// ground. One release commits one history entry; previews never touch bytes.
func (e *MapEditor) dragUnit(in Input, p image.Point, rail int) bool {
	if e.cancelledDrag {
		e.down = in.PrimaryDown
		if !in.PrimaryDown {
			e.cancelledDrag = false
		}
		return true
	}
	inside := p.In(image.Rect(0, 0, rail, e.window.Y))
	if e.unitDrag == nil && !e.down && in.PrimaryDown && inside && e.editable() && !e.catalog && !e.pathMode {
		hit := -1
		for _, i := range e.hitsAt(p) {
			if e.doc.Records[i].Kind == "Unit" && (hit < 0 || i == e.selected) {
				hit = i
			}
		}
		if hit >= 0 {
			r := e.doc.Records[hit]
			x, y, _ := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
			e.unitDrag = &editorUnitDrag{record: hit, index: r.Index, press: p, grab: image.Pt(x, y).Sub(r.Cell), cell: r.Cell}
			e.doc.Viewer.dragging = false
		}
	}
	d := e.unitDrag
	if d == nil {
		return false
	}
	e.down = in.PrimaryDown
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	cell := image.Pt(x, y).Sub(d.grab)
	if !inside || !ok || !cell.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) {
		e.cancelUnitDrag()
		return true
	}
	if math.Abs(float64(p.X-d.press.X)) > 4 || math.Abs(float64(p.Y-d.press.Y)) > 4 {
		if !d.moved && e.selected != d.record {
			e.Select(d.record, false)
		}
		d.moved = true
	}
	d.cell = cell
	if !in.PrimaryDown {
		e.unitDrag = nil
		if d.moved {
			next, err := e.doc.Edits.MoveUnit(d.index, cell)
			_ = e.adoptEdit(next, err)
		} else {
			e.selectAt(p)
		}
	}
	return true
}
