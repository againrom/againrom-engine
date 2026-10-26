package ui

import (
	"image"
	"math"
	"strings"

	"againrom/pkg/render/text"
)

func editorParagraphs(f *text.Font, s string, width int) []string {
	var out []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		out = append(out, editorWrap(f, paragraph, width)...)
	}
	return out
}

func (e *MapEditor) editable() bool { return e.doc != nil && e.doc.Edits != nil }
func (e *MapEditor) Dirty() bool    { return e.editable() && e.doc.Edits.Dirty() }

// TextInput is shared by the native path field and deterministic input drives.
func (e *MapEditor) TextInput(s string) {
	e.insertText(s)
}

func (e *MapEditor) saveAs(path string) error {
	return e.saveWith(func() (string, error) { return e.doc.Edits.SaveAs(path) })
}

// saveWith runs one save and the steps after it; write is the save itself.
func (e *MapEditor) saveWith(write func() (string, error)) error {
	then, errs := e.saveThen, e.saveErrors
	e.saveThen, e.saveErrors = nil, 0
	if !e.editable() {
		return nil
	}
	source, err := write()
	if err != nil {
		if r, ok := refusal(err); ok {
			if r.Ask {
				e.askSaveWithErrors(r, func() error {
					e.saveThen = then
					return e.saveWith(write)
				})
				return err
			}
			e.refuseSave(r)
			return err
		}
		e.message = err.Error()
		return err
	}
	e.doc.Source = source
	e.leave(leavePrompt)
	e.cancelRestore = nil
	e.message = ""
	e.afterSave()
	e.noteSave(errs)
	return e.leaveSaved(then)
}

// refreshGroup rebuilds the open group popup's group from the current document,
// keyed by player and group number, and closes the popup when no member is left.
func (e *MapEditor) refreshGroup() {
	if !e.groupSel.set {
		return
	}
	for _, g := range e.groups() {
		if g.player == e.groupSel.player && g.group == e.groupSel.group {
			e.groupSel = g
			return
		}
	}
	e.groupSel = editorGroup{}
}

func (e *MapEditor) adoptEdit(next *InspectionDocument, err error) error {
	if err != nil {
		e.message = err.Error()
		return err
	}
	if next == nil {
		return nil // unavailable history or same-cell move
	}
	e.saveNote = ""
	if e.terrainDiffers(next) {
		return e.adoptTerrainEdit(next, nil)
	}
	e.clearSelection()
	// A record edit does not change terrain. Retain the live viewer (camera,
	// GPU caches and render settings) and refresh entities, structures and sacks.
	old := e.doc.Viewer
	old.SetEntities(next.Viewer.entities)
	old.adoptPlacements(next.Viewer)
	next.Viewer = old
	var kind string
	index := -1
	if e.selected >= 0 && e.selected < len(e.doc.Records) {
		r := e.doc.Records[e.selected]
		kind, index = r.Kind, r.Index
	}
	e.doc = next
	e.cancelRestore = nil
	e.listRestore = nil
	e.closeReport()
	e.selected = -1
	for i, r := range next.Records {
		if r.Kind == kind && r.Index == index {
			e.selected = i
			break
		}
	}
	e.refreshGroup()
	e.message = ""
	return nil
}

type editorRecordDrag struct {
	kind          string
	record, index int
	press         image.Point
	grab, cell    image.Point
	origin        image.Point // the record's cell at the press
	moved         bool
	// group is set when the record is one of several selected: the drag
	// moves them all by the record's offset.
	group bool
	// area is the trigger area handle the drag carries; nil for a record.
	area *areaHandle
	// drop is the drop zone, plus one, the drag moves; 0 for none.
	drop int
}

func (e *MapEditor) cancelRecordDrag() {
	if e.recordDrag != nil {
		e.recordDrag = nil
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
		e.clearSelection()
		if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok {
			if t, ok := e.areaAt(image.Pt(x, y)); ok {
				e.selectTrigger(t, false)
			}
		}
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

// Capture records before Viewer.step: otherwise the same primary drag pans the
// ground. One release commits one history entry; previews never touch bytes.
func (e *MapEditor) dragRecord(in Input, p image.Point, rail int) bool {
	if e.cancelledDrag {
		e.down = in.PrimaryDown
		if !in.PrimaryDown {
			e.cancelledDrag = false
		}
		return true
	}
	inside := p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH))
	if e.recordDrag == nil && !e.down && in.PrimaryDown && inside && e.editable() {
		if d, ok := e.dropPress(p); ok {
			e.recordDrag = d
			e.doc.Viewer.dragging = false
		}
	}
	if e.recordDrag == nil && !e.down && in.PrimaryDown && inside && e.editable() {
		if h, ok := e.areaPress(p); ok {
			e.recordDrag = &editorRecordDrag{record: -1, index: -1, press: p, area: h}
			e.doc.Viewer.dragging = false
		}
	}
	if e.recordDrag == nil && !e.down && in.PrimaryDown && inside && e.editable() && e.free() {
		hit := -1
		for _, i := range e.hitsAt(p) {
			rank := dragRank[e.doc.Records[i].Kind]
			if rank == 0 || hit >= 0 && hit == e.selected {
				continue // the selected record wins
			}
			if hit < 0 || i == e.selected || rank < dragRank[e.doc.Records[hit].Kind] {
				hit = i
			}
		}
		if hit >= 0 {
			r := e.doc.Records[hit]
			x, y, _ := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
			e.recordDrag = &editorRecordDrag{kind: r.Kind, record: hit, index: r.Index, press: p, grab: image.Pt(x, y).Sub(r.Cell), cell: r.Cell, origin: r.Cell,
				group: e.selectionShown() && e.inSelection(hit)}
			e.doc.Viewer.dragging = false
		}
	}
	d := e.recordDrag
	if d == nil {
		return false
	}
	if d.area != nil {
		return e.dragArea(d, in, p, inside)
	}
	if d.drop > 0 {
		return e.dragDrop(d, in, p, inside)
	}
	e.down = in.PrimaryDown
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	cell := image.Pt(x, y).Sub(d.grab)
	if !inside || !ok || !cell.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) {
		e.cancelRecordDrag()
		return true
	}
	if math.Abs(float64(p.X-d.press.X)) > 4 || math.Abs(float64(p.Y-d.press.Y)) > 4 {
		if !d.moved && e.selected != d.record && !d.group {
			e.Select(d.record, false)
		}
		d.moved = true
	}
	d.cell = cell
	if !in.PrimaryDown {
		e.recordDrag = nil
		if d.moved && d.group {
			_ = e.moveSelection(cell.Sub(d.origin))
		} else if d.moved {
			next, err := e.doc.Edits.Move(d.kind, d.index, cell)
			_ = e.adoptEdit(next, err)
		} else {
			e.selectAt(p)
		}
	}
	return true
}

// adoptPlacements takes the structure and sack layers of a viewer built over
// the edited map, keeping this viewer's camera, caches and settings.
func (v *Viewer) adoptPlacements(n *Viewer) {
	v.grid.Structures = n.grid.Structures
	v.structuresFlat, v.structuresDisplaced, v.structureCounts = n.structuresFlat, n.structuresDisplaced, n.structureCounts
	v.structureAnimFlat, v.structureAnimDisplaced = n.structureAnimFlat, n.structureAnimDisplaced
	v.structureInfo, v.planeOrder = n.structureInfo, n.planeOrder
	v.sacks = n.sacks
}
