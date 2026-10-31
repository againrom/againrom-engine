package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// The Units and Structures tabs show a picture grid of what the Add window
// adds: every unit class or structure kind the host offers, each with the
// host's own picture of it. Choosing one opens the Add window on its
// settings at once, with the cell pick armed; each click on the map then
// places one copy, and a press held over the map places one on every cell
// it crosses. Each copy is one Add, one history entry.

// placeGrid is the tabs' grid state: its first shown row, the pictures the
// host gave, by kind and value, and the choices of the document they were
// read for.
type placeGrid struct {
	top     int
	icons   map[string]*image.RGBA
	doc     *InspectionDocument
	kind    string
	choices []InspectionChoice
}

// placeGridRows is how many rows of the grid show.
const placeGridRows = 3

var placeGridSides = map[string]int{"Structure": 2*(gridCellSide-8) + 8}

func placeGridShell(kind string, top int) *popupGrid {
	return &popupGrid{Rows: placeGridRows, Top: top, Side: placeGridSides[kind]}
}

// placeGridKind is the record kind and the Add step the shown tab's grid
// chooses: unit classes on the Units tab, structure kinds on the Structures
// tab.
func (e *MapEditor) placeGridKind() (kind, key string, ok bool) {
	if _, host := e.addHost(); !host || e.listKind() != "records" {
		return "", "", false
	}
	switch e.filter {
	case 1:
		return "Unit", "class", true
	case 2:
		return "Structure", "kind", true
	}
	return "", "", false
}

// placeChoices are the choices of the grid's step, as the host offers them
// for the open document.
func (e *MapEditor) placeChoices(kind, key string) []InspectionChoice {
	host, ok := e.addHost()
	if !ok {
		return nil
	}
	if e.place.doc != e.doc || e.place.kind != kind {
		e.place.doc, e.place.kind = e.doc, kind
		e.place.choices = host.NewRecordStep(InspectionNewRecord{Kind: kind}, key).Choices
	}
	return e.place.choices
}

// placeIcon is the host's picture of choice value of kind's step key, read
// once.
func (e *MapEditor) placeIcon(kind, key string, value int64) *image.RGBA {
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
	pic := host.NewRecordPreview(kind, key, value)
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
	kind, key, ok := e.placeGridKind()
	if !ok {
		return nil, "", false
	}
	choices := e.placeChoices(kind, key)
	if len(choices) == 0 {
		return nil, "", false
	}
	g := placeGridShell(kind, e.place.top)
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
	return g, note, true
}

// placeGridHeight is the room the grid takes in the list popup, 0 with none.
func (e *MapEditor) placeGridHeight() int {
	kind, _, ok := e.placeGridKind()
	if !ok {
		return 0
	}
	return placeGridShell(kind, 0).height() + fontHeight(e.bodyFont(), 15) + popupRowExtra
}

// placeGridWheel scrolls the grid by the wheel's rows.
func (e *MapEditor) placeGridWheel(l popupLayout, rows int) {
	if l.GridCols <= 0 {
		return
	}
	total := (len(e.place.choices) + l.GridCols - 1) / l.GridCols
	e.place.top = max(0, min(e.place.top+rows, total-placeGridRows))
}

// placeChoose is a press on grid cell n: it opens the Add window on that
// choice's settings, with the cell pick armed, to place copies of it. A
// press on the chosen cell ends the placing.
func (e *MapEditor) placeChoose(n int) {
	kind, key, ok := e.placeGridKind()
	choices := e.placeChoices(kind, key)
	if !ok || n < 0 || n >= len(choices) {
		return
	}
	if r, v, ok := e.placing(key); ok && r.kind == kind && v == choices[n].Value {
		e.leave(leavePick | leavePrompt)
		return
	}
	e.openAdd(kind)
	r, ok := e.addRunning()
	if !ok {
		return
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
	if n < l.GridFirst || n >= l.GridFirst+len(l.GridCells) {
		return image.Rectangle{}, false
	}
	return l.GridCells[n-l.GridFirst], true
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
