package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
)

// The Units and Structures tabs show a picture grid of what the Add window
// adds: every unit class or structure kind the host offers, each with the
// host's own picture of it. Choosing one opens the Add window on its
// settings at once, with the cell pick armed; each click on the map then
// places one copy, and a press held over the map places one on every cell
// it crosses. Each copy is one Add, one history entry.

// The object brush's one-kind mode shows the same grid over the structure
// kinds of the Objects palette, and a press on its cell places copies the
// same way.

// placeGrid is the tabs' grid state: its first shown row, and the object
// brush's, the pictures the host gave, by kind and value, and the choices
// of the document they were read for with the palette of each.
type placeGrid struct {
	top, brushTop int
	ruined        bool
	icons         map[string]*image.RGBA
	doc           *InspectionDocument
	kind          RecordKind
	choices       []InspectionChoice
	groups        []InspectionChoiceGroup
	palettes      map[int64]Palette
}

// InspectionChoiceGroup places choice values under a block and row heading.
// Headings are in the font's bytes; values keep their field meanings.
type InspectionChoiceGroup struct {
	Block, Head string
	Values      []int64
}

var placeGridSides = map[RecordKind]int{KindStructure: 2*(gridCellSide-8) + 8}

// placeGridShell is kind's grid with no cells: it shows every row the room
// holds.
func placeGridShell(kind RecordKind, top int) *popupGrid {
	return &popupGrid{Top: top, Side: placeGridSides[kind]}
}

// placeGridKind is the record kind and the Add step the shown tab's grid
// chooses: unit classes on the Units tab, structure kinds on the Structures
// tab.
func (e *MapEditor) placeGridKind() (kind RecordKind, key string, ok bool) {
	if _, host := e.addHost(); !host || e.listKind() != "records" {
		return "", "", false
	}
	switch e.filter {
	case 1:
		return KindUnit, "identity", true
	case 2:
		return KindStructure, "kind", true
	}
	return "", "", false
}

// placeGridOf is the record kind, the Add step and the palette of the grid
// the list popup shows: on the Units tab every unit class, on the
// Structures tab the structure kinds of the Structures palette, and in the
// object brush's one-kind mode the structure kinds of the Objects palette.
func (e *MapEditor) placeGridOf() (kind RecordKind, key string, palette Palette, ok bool) {
	if kind, key, ok := e.placeGridKind(); ok {
		if kind == KindStructure {
			return kind, key, PaletteStructures, true
		}
		return kind, key, PaletteOwn, true
	}
	if _, host := e.addHost(); host && e.listKind() == "brush" && e.brush.mode == brushObjects && !e.brush.forest.on && !e.brush.forest.trees {
		return KindStructure, "kind", PaletteObjects, true
	}
	return "", "", PaletteOwn, false
}

// placeChoices are the choices of the grid's step, as the host offers them
// for the open document.
func (e *MapEditor) placeChoices(kind RecordKind, key string) []InspectionChoice {
	host, ok := e.addHost()
	if !ok {
		return nil
	}
	if e.place.doc != e.doc || e.place.kind != kind {
		step := host.NewRecordStep(InspectionNewRecord{Kind: kind}, key)
		e.place.doc, e.place.kind = e.doc, kind
		e.place.choices, e.place.palettes = nil, step.Palettes
		e.place.groups = step.Groups
		for _, c := range step.Choices {
			if !slices.Contains(step.Shut, c.Value) {
				e.place.choices = append(e.place.choices, c)
			}
		}
	}
	return e.place.choices
}

// placeShown are the choices the shown grid holds: those of its step that
// list in its palette, every one for a palette of PaletteOwn.
func (e *MapEditor) placeShown() []InspectionChoice {
	kind, key, palette, ok := e.placeGridOf()
	if !ok {
		return nil
	}
	return e.placeIn(kind, key, palette)
}

// placeIn are the choices of kind's step key that list in palette.
func (e *MapEditor) placeIn(kind RecordKind, key string, palette Palette) []InspectionChoice {
	all := e.placeChoices(kind, key)
	if palette == PaletteOwn {
		return all
	}
	var out []InspectionChoice
	for _, c := range all {
		if e.place.palettes[c.Value].in(PaletteStructures) == palette {
			out = append(out, c)
		}
	}
	return out
}

// placeTop is the first shown row of the shown grid.
func (e *MapEditor) placeTop() *int {
	if e.listKind() == "brush" {
		return &e.place.brushTop
	}
	return &e.place.top
}

// brushPlaceGridHeight is the room the brush window's list leaves the
// Objects palette's grid of structure kinds: its least rows and the hint.
func (e *MapEditor) brushPlaceGridHeight() int {
	if _, _, palette, ok := e.placeGridOf(); !ok || palette != PaletteObjects || len(e.placeShown()) == 0 {
		return 0
	}
	return (&popupGrid{Rows: gridLeastRows, Side: placeGridSides[KindStructure]}).height() + fontHeight(e.bodyFont(), 15) + popupRowExtra
}

// placeIcon is the host's picture of choice value of kind's step key, read
// once.
func (e *MapEditor) placeIcon(kind RecordKind, key string, value int64) *image.RGBA {
	host, ok := e.addHost()
	if !ok {
		return nil
	}
	k := fmt.Sprintf("%s/%s/%d", kind, key, value)
	if pic, ok := e.place.icons[k]; ok {
		return pic
	}
	if e.place.icons == nil {
		e.place.icons = map[string]*image.RGBA{}
	}
	previewKey := key
	if kind == KindStructure && key == "kind" {
		previewKey = "kind-grid"
	}
	pic := host.NewRecordPreview(kind, previewKey, value)
	e.place.icons[k] = pic
	return pic
}

// placing is the open Add window that places copies, and the value of its
// step key: the grid cell shown pressed.
func (e *MapEditor) placing(key string) (*addRun, int64, bool) {
	r, ok := e.addRunning()
	if !ok || !r.repeat {
		return nil, 0, false
	}
	for i, st := range r.steps {
		if st.Key == key {
			v, err := strconv.ParseInt(strings.TrimSpace(e.addText(r, i)), 10, 64)
			return r, v, err == nil
		}
	}
	return r, 0, false
}

// placeGridSpec is the shown tab's grid as data, and the note naming the
// chosen cell.
func (e *MapEditor) placeGridSpec() (*popupGrid, string, bool) {
	kind, key, _, ok := e.placeGridOf()
	if !ok {
		return nil, "", false
	}
	choices := e.placeShown()
	if len(choices) == 0 {
		return nil, "", false
	}
	g := placeGridShell(kind, *e.placeTop())
	r, chosen, placing := e.placing(key)
	placing = placing && r.kind == kind
	note := e.word("editor.place.hint")
	for _, c := range choices {
		pressed := placing && c.Value == chosen
		if pressed {
			note = fmt.Sprintf(e.word("editor.place.placing"), c.Label)
		}
		g.Cells = append(g.Cells, popupGridCell{Icon: e.placeIcon(kind, key, c.Value), Pressed: pressed, Tip: c.Label})
	}
	if len(e.place.groups) > 0 {
		indices := make(map[int64]int, len(choices))
		for i, c := range choices {
			indices[c.Value] = i
		}
		for _, group := range e.place.groups {
			row := popupGridGroup{Block: group.Block, Head: group.Head}
			for _, value := range group.Values {
				if n, ok := indices[value]; ok {
					row.Cells = append(row.Cells, n)
				}
			}
			if len(row.Cells) > 0 {
				g.Groups = append(g.Groups, row)
			}
		}
	}
	return g, note, true
}

// placeGridWheel scrolls the grid by the wheel's rows.
func (e *MapEditor) placeGridWheel(l popupLayout, rows int) {
	if l.GridCols <= 0 {
		return
	}
	g, _, ok := e.placeGridSpec()
	if !ok {
		return
	}
	total := g.totalRows(l.GridCols)
	top := e.placeTop()
	*top = max(0, min(*top+rows, total-l.GridRows))
}

// placeChoose is a press on grid cell n: it opens the Add window on that
// choice's settings, with the cell pick armed, to place copies of it. A
// press on the chosen cell ends the placing. A press in the object brush
// leaves the brush, as every Add does, and shows no records list while
// the copies are placed; Esc goes back to the brush.
func (e *MapEditor) placeChoose(n int) {
	kind, key, _, ok := e.placeGridOf()
	choices := e.placeShown()
	if !ok || n < 0 || n >= len(choices) {
		return
	}
	fromBrush := e.listKind() == "brush"
	if r, v, ok := e.placing(key); ok && r.kind == kind && v == choices[n].Value {
		e.leave(leavePick | leavePrompt)
		return
	}
	e.openAdd(kind)
	r, ok := e.addRunning()
	if !ok {
		return
	}
	if fromBrush {
		e.listShown, r.brush = false, true
	}
	r.repeat = true
	e.prompt.path = strconv.FormatInt(choices[n].Value, 10)
	e.addFollow()
	e.addNext()
	e.armPick(pickAddCell)
}

// PlaceGridCellRect is the window rectangle of cell n of the list popup's
// grid, the tab's or the forest's, when it is shown: a headless driver
// presses it.
func (e *MapEditor) PlaceGridCellRect(n int) (image.Rectangle, bool) {
	s, ok := e.listSpec()
	if !ok || s.Grid == nil {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, rect := range l.GridCells {
		if l.gridIndex(i) == n {
			return rect, true
		}
	}
	return image.Rectangle{}, false
}

// PlaceGridLen is how many cells the shown tab's grid has.
func (e *MapEditor) PlaceGridLen() int {
	if g, _, ok := e.placeGridSpec(); ok {
		return len(g.Cells)
	}
	return 0
}

// placeStroke is a press held over the map while copies are placed: the
// cells it has placed on.
type placeStroke struct{ cells map[image.Point]bool }

// placePointer drives the map while copies are placed: a press places one
// copy on the cell under the pointer, and while the button is held each
// further cell the pointer enters gets one. It reports whether it took the
// input, so the map does not pan under it.
func (e *MapEditor) placePointer(in Input, p image.Point, rail int) bool {
	r, ok := e.addRunning()
	if !ok || !r.repeat {
		return false
	}
	if r.stroke == nil {
		if e.pick != pickAddCell || !in.PrimaryDown || e.down || e.recordDrag != nil || e.cancelledDrag {
			return false
		}
		if !p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
			return false
		}
		r.stroke = &placeStroke{cells: map[image.Point]bool{}}
		e.doc.Viewer.dragging = false
	}
	e.down = in.PrimaryDown
	if x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y)); ok && e.pick == pickAddCell && p.In(image.Rect(0, 0, rail, e.doc.Viewer.cam.ViewH)) {
		if cell := image.Pt(x, y); cell.In(image.Rect(0, 0, e.doc.Width, e.doc.Height)) && !r.stroke.cells[cell] {
			r.stroke.cells[cell] = true
			e.pickAt(p)
		}
	}
	if !in.PrimaryDown {
		r.stroke = nil
	}
	return true
}

// placedNote is the Add window's note after a copy is placed.
func (e *MapEditor) placedNote(r *addRun) {
	r.placed++
	r.note = fmt.Sprintf(e.word("editor.place.placed"), r.placed)
}
