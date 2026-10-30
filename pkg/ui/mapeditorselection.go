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
// from, so a paste may land in another map opened in the same window. Patch
// is the size in cells of the clip's terrain patch, zero when it holds none;
// a paste puts the patch's top-left cell on the paste's cell.
type InspectionClip interface {
	Len() int
	Patch() image.Point
}

// SelectionEdit is the outcome of an edit of several records: the document
// (nil when nothing changed), the records left selected, the status note, in
// UTF-8, of what the edit did, and a note per record it refused. The status
// line names the first refusals and counts the rest. Refused is each refused
// record.
type SelectionEdit struct {
	Doc      *InspectionDocument
	Selected []RecordRef
	Note     string
	Refusals []string
	Refused  []SelectionRefusal
}

// SelectionRefusal is a record a group edit refused: its address before the
// edit (in the clip for a paste) and its footprint where the edit would have
// put it.
type SelectionRefusal struct {
	Ref       RecordRef
	Footprint image.Rectangle
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
	box      image.Rectangle // the cells the last selection box covered; empty when none
	hidden   bool            // the selection's popup is closed
}

// selectionMarquee is a Shift press on the map: the frame point it began at
// and the cells of the rectangle's two corners.
type selectionMarquee struct {
	start    image.Point
	from, to image.Point
	moved    bool
	// box is a plain press: its drag selects the open tab's kinds and its
	// click selects as a click on the map does.
	box bool
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

// replaceGroup makes refs the selected group and drops the selection box: a box
// is the selection only until the selection changes, so the group a click, a
// list pick or a toggle makes is copied as records, not as the box's terrain.
func (e *MapEditor) replaceGroup(refs []RecordRef) {
	e.selection.refs, e.selection.box = refs, image.Rectangle{}
}

// clearSelection drops the group, its note and any rectangle; the copied
// records stay.
func (e *MapEditor) clearSelection() {
	e.replaceGroup(nil)
	e.selection.marquee = nil
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
	e.replaceGroup(slices.Clone(refs))
	e.selection.hidden = false
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
		switch {
		case m.box && m.moved:
			e.selectCellsOf(m.from, m.to, e.boxKinds())
			e.selection.box = cellBox(m.from, m.to)
		case m.box:
			e.selectAt(m.start)
		case m.moved:
			e.selectCells(m.from, m.to)
		default:
			e.toggleAt(m.start)
		}
	}
	return true
}

// boxPointer starts a selection box: a plain press on the map that no tool,
// pick or record drag took. Its drag draws the box; a press that does not
// move is a click on the map. The map pans with the middle button and the
// arrow keys.
func (e *MapEditor) boxPointer(in Input, p image.Point, rail int) bool {
	if in.Shift || !in.PrimaryDown || e.down || e.recordDrag != nil || e.cancelledDrag || !e.free() || e.blocksCanvas() || e.brushActive() {
		return false
	}
	if !p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
		return false
	}
	cell, ok := e.selectionCell(p)
	if !ok {
		return false
	}
	e.selection.marquee = &selectionMarquee{start: p, from: cell, to: cell, box: true}
	e.down = true
	e.doc.Viewer.dragging = false
	return true
}

// boxKinds are the record kinds a selection box takes: the open tab's kind
// on the Units, Structures and Sacks tabs, units on the Groups tab, else
// every kind a selection takes.
func (e *MapEditor) boxKinds() []string {
	if e.listShown && e.listKind() == "records" {
		switch e.filter {
		case 1, editorGroupsFilter:
			return []string{"Unit"}
		case 2:
			return []string{"Structure"}
		case 3:
			return []string{"Sack"}
		}
	}
	return nil
}

// cellBox is the cell rectangle with corners a and b, both inside it.
func cellBox(a, b image.Point) image.Rectangle {
	return image.Rect(min(a.X, b.X), min(a.Y, b.Y), max(a.X, b.X)+1, max(a.Y, b.Y)+1)
}

// selectCells makes the group every unit, structure and sack whose cell lies
// in the rectangle with corners a and b.
func (e *MapEditor) selectCells(a, b image.Point) { e.selectCellsOf(a, b, nil) }

// selectCellsOf is selectCells over kinds only; no kinds is every kind a
// selection takes.
func (e *MapEditor) selectCellsOf(a, b image.Point, kinds []string) {
	e.noteSelection("", nil)
	e.setSelection(e.cellRecords(cellBox(a, b), kinds))
}

// cellRecords are the records of kinds whose cell lies in box, in document
// order; no kinds is every kind a selection takes.
func (e *MapEditor) cellRecords(box image.Rectangle, kinds []string) []RecordRef {
	var refs []RecordRef
	for _, r := range e.doc.Records {
		if selectable(r.Kind) && r.Spatial && r.Cell.In(box) && (len(kinds) == 0 || slices.Contains(kinds, r.Kind)) {
			refs = append(refs, RecordRef{r.Kind, r.Index})
		}
	}
	return refs
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
		e.replaceGroup(refs)
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

// SelectedRecords are the records the selection holds, else the selected
// record when a selection can hold it: what a group action takes.
func (e *MapEditor) SelectedRecords() []RecordRef { return slices.Clone(e.selectionRefs()) }

func (e *MapEditor) selectionHost() (InspectionSelection, bool) {
	if !e.editable() {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionSelection)
	return h, ok
}

// selectionKey is Ctrl+C, which copies the group or the selected record, and
// Ctrl+V, which arms the paste brush: the next click on the map pastes the
// copied records there. A copy of a selection box takes every unit,
// structure and sack whose cell the box holds, whichever tab chose the box's
// records.
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
		_, ok := h.(InspectionAreaCopy)
		box := e.selection.box
		if !ok || box.Empty() {
			box = image.Rectangle{}
		} else {
			refs = e.cellRecords(box, nil)
		}
		if len(refs) == 0 && box.Empty() {
			e.noteSelection(e.word("editor.selection.none"), nil)
			return nil
		}
		clip, note, err := e.copyOf(refs, box)
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
	e.leave(leaveBrush)
	e.armPick(pickPaste)
	return nil
}

// pasteAt is the armed paste brush's click: the clip pasted with its origin
// on the ground cell under p, as one edit. The clip stays for the next Ctrl+V.
func (e *MapEditor) pasteAt(p image.Point) error {
	if _, ok := e.selectionHost(); !ok || e.selection.clip == nil {
		return nil
	}
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		e.noteSelection(e.word("editor.selection.nocell"), nil)
		return nil
	}
	_, err := e.commandPaste(recordsPaste{Clip: e.selection.clip, Cell: image.Pt(x, y)})
	return err
}

// deleteSelection removes the group as one edit, after asking when it empties
// a group a trigger names.
func (e *MapEditor) deleteSelection() error {
	if _, ok := e.selectionHost(); ok && e.askDelete(e.selection.refs, false) {
		return nil
	}
	return e.deleteRecords()
}

// deleteRecords removes the group as one edit.
func (e *MapEditor) deleteRecords() error {
	if _, ok := e.selectionHost(); !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	e.leave(leaveDrag | leavePick)
	_, err := e.commandDeleteRecords(e.selection.refs)
	return err
}

// moveSelection shifts the group by offset cells as one edit.
func (e *MapEditor) moveSelection(offset image.Point) error {
	if _, ok := e.selectionHost(); !ok {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	_, err := e.commandMoveRecords(recordsMove{Refs: e.selection.refs, Offset: offset})
	return err
}

// boxPress is a press inside the selection box on no record, or on one the
// group holds, of a host that moves an area: its drag moves the group with
// the objects under the box. hit is the record the press would drag, or -1.
func (e *MapEditor) boxPress(p image.Point, hit int) (*editorRecordDrag, bool) {
	box := e.selection.box
	_, area := e.doc.Edits.(InspectionAreaCopy)
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	cell := image.Pt(x, y)
	if box.Empty() || !area || !ok || !cell.In(box) || hit >= 0 && !e.inSelection(hit) {
		return nil, false
	}
	return &editorRecordDrag{record: -1, index: -1, press: p, cell: cell, origin: cell, group: true, box: true}, true
}

// boxOffset is offset cut so the selection box stays on the map.
func (e *MapEditor) boxOffset(offset image.Point) image.Point {
	b := e.selection.box
	return image.Pt(max(-b.Min.X, min(offset.X, e.doc.Width-b.Max.X)), max(-b.Min.Y, min(offset.Y, e.doc.Height-b.Max.Y)))
}

// moveArea shifts the group and the objects under the selection box by
// offset cells as one edit; the box follows what it moved.
func (e *MapEditor) moveArea(offset image.Point) error {
	h, ok := e.selectionHost()
	if _, isArea := h.(InspectionAreaCopy); !ok || !isArea {
		e.noteSelection(e.word("editor.selection.nohost"), nil)
		return nil
	}
	box := e.selection.box
	ch, err := e.commandMoveArea(areaMove{Refs: e.selection.refs, Box: box, Offset: offset})
	if err != nil {
		return err
	}
	if ch.doc != nil {
		e.selection.box = box.Add(offset)
	}
	return nil
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

// marqueeLines outlines the rectangle a drag is drawing, else the cells the
// last selection box covered, which Ctrl+C copies, cell edge by cell edge,
// so it follows the ground's heights.
func (e *MapEditor) marqueeLines() []editorLine {
	m := e.selection.marquee
	box := e.selection.box
	if m != nil && m.moved {
		box = cellBox(m.from, m.to)
	} else if _, area := e.doc.Edits.(InspectionAreaCopy); m != nil || box.Empty() || !area {
		return nil
	} else if d := e.recordDrag; d != nil && d.box && d.moved {
		box = box.Add(d.cell.Sub(d.origin)) // preview only, as the records' marks
	}
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
