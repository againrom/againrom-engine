package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
)

// RecordRef addresses one record of a document by its InspectionRecord Kind
// and Index.
type RecordRef struct {
	Kind  string
	Index int
}

// InspectionClip is a copied set of records as the host keeps them: their
// bytes and cells as they stood when copied. It outlives the document it came
// from, so a paste may land in another map opened in the same window.
type InspectionClip interface{ Len() int }

// SelectionEdit is the outcome of an edit of several records: the document
// (nil when nothing changed), the records left selected, the status note, in
// UTF-8, of what the edit did, and a note per record it refused. The status
// line names the first refusals and counts the rest.
type SelectionEdit struct {
	Doc      *InspectionDocument
	Selected []RecordRef
	Note     string
	Refusals []string
}

// InspectionSelection is the optional seam of a host that edits several
// records at once. Each call that changes the map is one edit: one undo step.
// Move shifts every record by offset cells; Paste places the clip so that its
// top-left record cell lands on cell, each record copied as Clone copies one.
type InspectionSelection interface {
	MoveRecords(refs []RecordRef, offset image.Point) (SelectionEdit, error)
	DeleteRecords(refs []RecordRef) (SelectionEdit, error)
	CopyRecords(refs []RecordRef) (InspectionClip, string, error)
	PasteRecords(clip InspectionClip, cell image.Point) (SelectionEdit, error)
}

// selectionState is the window's group selection: the records, the status
// note of the last group edit, the copied records and the rectangle a Shift
// drag is drawing.
type selectionState struct {
	refs     []RecordRef
	note     string
	refusals []string
	clip     InspectionClip
	marquee  *selectionMarquee
}

// selectionMarquee is a Shift press on the map: the frame point it began at
// and the cells of the rectangle's two corners.
type selectionMarquee struct {
	start    image.Point
	from, to image.Point
	moved    bool
}

// selectable reports a record kind the group selection takes: the kinds a
// drag moves.
func selectable(kind string) bool { return dragRank[kind] > 0 }

// selectionShown reports that several records are selected, so the group's
// keys and help apply.
func (e *MapEditor) selectionShown() bool {
	return e.doc != nil && !e.catalog && len(e.selection.refs) > 1
}

// inSelection reports whether document record i is in the group.
func (e *MapEditor) inSelection(i int) bool {
	if i < 0 || i >= len(e.doc.Records) || len(e.selection.refs) == 0 {
		return false
	}
	r := e.doc.Records[i]
	return slices.Contains(e.selection.refs, RecordRef{r.Kind, r.Index})
}

// clearSelection drops the group, its note and any rectangle; the copied
// records stay.
func (e *MapEditor) clearSelection() {
	e.selection.refs, e.selection.marquee = nil, nil
	e.noteSelection("", nil)
}

// setSelection makes refs the group and selects its first record. The
// record's panel stays closed while a group is selected, so the map shows the
// group.
func (e *MapEditor) setSelection(refs []RecordRef) {
	first := -1
	for i, r := range e.doc.Records {
		if slices.Contains(refs, RecordRef{r.Kind, r.Index}) {
			first = i
			break
		}
	}
	if first >= 0 {
		e.Select(first, false)
	} else {
		e.selected = -1
	}
	e.selection.refs = slices.Clone(refs)
	e.objectHidden = len(refs) > 1
}

// selectionCell is the map cell under frame point p, clamped into the map.
func (e *MapEditor) selectionCell(p image.Point) (image.Point, bool) {
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		return image.Point{}, false
	}
	return image.Pt(max(0, min(x, e.doc.Width-1)), max(0, min(y, e.doc.Height-1))), true
}

// marqueePointer drives a Shift press on the map: a drag draws a rectangle
// that selects the records whose cell lies inside it, and a click adds the
// record under the pointer to the group or takes it out. It reports whether
// it took the input, so the map does not pan under it.
func (e *MapEditor) marqueePointer(in Input, p image.Point, rail int) bool {
	m := e.selection.marquee
	if m == nil {
		if !in.Shift || !in.PrimaryDown || e.down || e.recordDrag != nil || e.cancelledDrag || !e.free() || e.blocksCanvas() {
			return false
		}
		if !p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
			return false
		}
		cell, ok := e.selectionCell(p)
		if !ok {
			return false
		}
		e.selection.marquee = &selectionMarquee{start: p, from: cell, to: cell}
		e.down = true
		e.doc.Viewer.dragging = false
		return true
	}
	e.down = in.PrimaryDown
	if cell, ok := e.selectionCell(p); ok {
		m.to = cell
	}
	if absInt(p.X-m.start.X) > 4 || absInt(p.Y-m.start.Y) > 4 {
		m.moved = true
	}
	if !in.PrimaryDown {
		e.selection.marquee = nil
		if m.moved {
			e.selectCells(m.from, m.to)
		} else {
			e.toggleAt(m.start)
		}
	}
	return true
}

// cellBox is the cell rectangle with corners a and b, both inside it.
func cellBox(a, b image.Point) image.Rectangle {
	return image.Rect(min(a.X, b.X), min(a.Y, b.Y), max(a.X, b.X)+1, max(a.Y, b.Y)+1)
}

// selectCells makes the group every unit, structure and sack whose cell lies
// in the rectangle with corners a and b.
func (e *MapEditor) selectCells(a, b image.Point) {
	box := cellBox(a, b)
	var refs []RecordRef
	for _, r := range e.doc.Records {
		if selectable(r.Kind) && r.Spatial && r.Cell.In(box) {
			refs = append(refs, RecordRef{r.Kind, r.Index})
		}
	}
	e.noteSelection("", nil)
	e.setSelection(refs)
}

// toggleAt adds the record under frame point p to the group, or takes it out;
// a group starts from the selected record.
func (e *MapEditor) toggleAt(p image.Point) {
	hit := -1
	for _, i := range e.hitsAt(p) {
		if k := e.doc.Records[i].Kind; selectable(k) && (hit < 0 || dragRank[k] < dragRank[e.doc.Records[hit].Kind]) {
			hit = i
		}
	}
	if hit < 0 {
		return
	}
	refs := slices.Clone(e.selection.refs)
	if len(refs) == 0 && e.selected >= 0 && e.selected < len(e.doc.Records) && selectable(e.doc.Records[e.selected].Kind) {
		r := e.doc.Records[e.selected]
		refs = []RecordRef{{r.Kind, r.Index}}
	}
	r := e.doc.Records[hit]
	ref := RecordRef{r.Kind, r.Index}
	if i := slices.Index(refs, ref); i >= 0 {
		refs = slices.Delete(refs, i, i+1)
	} else {
		refs = append(refs, ref)
	}
	e.noteSelection("", nil)
	if slices.Contains(refs, ref) {
		e.Select(hit, false)
		e.selection.refs = refs
		e.objectHidden = len(refs) > 1
		return
	}
	e.setSelection(refs)
}

// selectionRefs is the records a copy takes: the group, else the selected
// record when a group can hold it.
func (e *MapEditor) selectionRefs() []RecordRef {
	if len(e.selection.refs) > 0 {
		return e.selection.refs
	}
	if e.doc != nil && e.selected >= 0 && e.selected < len(e.doc.Records) {
		if r := e.doc.Records[e.selected]; selectable(r.Kind) && r.Spatial {
			return []RecordRef{{r.Kind, r.Index}}
		}
	}
	return nil
}

func (e *MapEditor) selectionHost() (InspectionSelection, bool) {
	if !e.editable() {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionSelection)
	return h, ok
}

// selectionKey is Ctrl+C, which copies the group or the selected record, and
// Ctrl+V, which pastes the copied records at the cell under the pointer.
func (e *MapEditor) selectionKey(key string) error {
	if e.doc == nil || e.catalog {
		return nil
	}
	h, ok := e.selectionHost()
	if !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	e.leave(leaveDrag | leavePick)
	if key == "copy" {
		refs := e.selectionRefs()
		if len(refs) == 0 {
			e.noteSelection(e.word("editor.selection.none"), nil)
			return nil
		}
		clip, note, err := h.CopyRecords(refs)
		if err != nil {
			e.message = err.Error()
			return err
		}
		e.selection.clip = clip
		e.noteSelection(e.editorHostText(note), nil)
		return nil
	}
	if e.selection.clip == nil || e.selection.clip.Len() == 0 {
		e.noteSelection(e.word("editor.selection.noclip"), nil)
		return nil
	}
	if !e.hoverOK {
		e.noteSelection(e.word("editor.selection.nocell"), nil)
		return nil
	}
	return e.adoptSelection(h.PasteRecords(e.selection.clip, e.hoverCell))
}

// deleteSelection removes the group as one edit.
func (e *MapEditor) deleteSelection() error {
	h, ok := e.selectionHost()
	if !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	e.leave(leaveDrag | leavePick)
	return e.adoptSelection(h.DeleteRecords(e.selection.refs))
}

// moveSelection shifts the group by offset cells as one edit.
func (e *MapEditor) moveSelection(offset image.Point) error {
	h, ok := e.selectionHost()
	if !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	return e.adoptSelection(h.MoveRecords(e.selection.refs, offset))
}

// adoptSelection takes a group edit: its document, the records it leaves
// selected and its note. An edit that changed nothing keeps the group.
func (e *MapEditor) adoptSelection(r SelectionEdit, err error) error {
	if err != nil {
		e.message = err.Error()
		return err
	}
	if r.Doc != nil {
		if err := e.adoptEdit(r.Doc, nil); err != nil {
			return err
		}
		e.setSelection(r.Selected)
	}
	e.noteSelection(e.editorHostText(r.Note), r.Refusals)
	return nil
}

// noteSelection sets the note of the last group edit and its refusals, both
// in the font's text.
func (e *MapEditor) noteSelection(note string, refusals []string) {
	e.selection.note, e.selection.refusals = note, nil
	for _, r := range refusals {
		e.selection.refusals = append(e.selection.refusals, e.editorHostText(r))
	}
}

// selectionShownRefusals is how many refusals the whole status line names.
const selectionShownRefusals = 2

// selectionNote is the note of the last group edit with named, the refusals
// it names, and the count of the refusals it does not.
func (e *MapEditor) selectionNote(named []string) string {
	var parts []string
	if e.selection.note != "" {
		parts = append(parts, e.selection.note)
	}
	parts = append(parts, named...)
	if n := len(e.selection.refusals) - len(named); n > 0 {
		parts = append(parts, fmt.Sprintf(e.word("editor.selection.more"), n))
	}
	return strings.Join(parts, "; ")
}

// selectionStatus is the status line's part for the group: how many records
// it holds and note, the note of its last edit.
func (e *MapEditor) selectionStatus(note string) []string {
	var out []string
	if e.selectionShown() {
		out = append(out, fmt.Sprintf(e.word("editor.selection.count"), len(e.selection.refs)))
	}
	if note != "" {
		out = append(out, note)
	}
	return out
}

// marqueeLines outlines the rectangle a Shift drag is drawing, cell edge by
// cell edge, so it follows the ground's heights.
func (e *MapEditor) marqueeLines() []editorLine {
	m := e.selection.marquee
	if m == nil || !m.moved {
		return nil
	}
	box := cellBox(m.from, m.to)
	v := e.doc.Viewer
	c := editorCondition
	var out []editorLine
	edge := func(x, y, a, b int) {
		verts := v.cellSpace().vertices(v.cam, x, y)
		out = append(out, editorLine{verts[a].DstX, verts[a].DstY, verts[b].DstX, verts[b].DstY, c})
	}
	for x := box.Min.X; x < box.Max.X; x++ {
		edge(x, box.Min.Y, 0, 1)
		edge(x, box.Max.Y-1, 2, 3)
	}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		edge(box.Min.X, y, 0, 2)
		edge(box.Max.X-1, y, 1, 3)
	}
	return out
}
