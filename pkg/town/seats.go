package town

import "image"

// SeatSpec is a page's seat grid: Columns seats in a row, Rows rows, the
// first seat at First, each next seat of a row offset by Step and each next
// row by RowStep. Seat i is in row i/Columns and column i%Columns. Back is
// the page art drawn in a populated seat under its actor, Actor the page art
// of a seated actor's frames. A selection requests the page scene's Select
// slot; the selected actor alone advances one frame after FrameMS, compared
// as Compare.
type SeatSpec struct {
	First   Rect     `json:"first"`
	Step    Point    `json:"step"`
	RowStep Point    `json:"row-step"`
	Columns int      `json:"columns"`
	Rows    int      `json:"rows"`
	Back    string   `json:"back"`
	Actor   string   `json:"actor"`
	Select  string   `json:"select"`
	FrameMS int      `json:"frame-ms"`
	Compare string   `json:"compare"`
	Cite    []string `json:"cite"`
}

// Count is the number of seats.
func (s *SeatSpec) Count() int { return s.Columns * s.Rows }

// Seat answers seat i's rectangle, empty outside the grid.
func (s *SeatSpec) Seat(i int) image.Rectangle {
	if i < 0 || i >= s.Count() {
		return image.Rectangle{}
	}
	at := image.Pt(s.Step[0]*(i%s.Columns)+s.RowStep[0]*(i/s.Columns),
		s.Step[1]*(i%s.Columns)+s.RowStep[1]*(i/s.Columns))
	return s.First.Rectangle().Add(at)
}

// SeatAt answers the first of the first n seats that holds p, -1 for none.
func (s *SeatSpec) SeatAt(p image.Point, n int) int {
	for i := 0; i < n && i < s.Count(); i++ {
		if p.In(s.Seat(i)) {
			return i
		}
	}
	return -1
}

// ButtonSpec is one page button: the action its page binds by Name, its hit
// rectangle, the page art it shows released and pressed, drawn at the
// rectangle's corner, and the main text slot of its caption, if it has one.
// A button with If exists only while the host's condition holds: it is not
// drawn, hit or chosen otherwise.
type ButtonSpec struct {
	Name    string   `json:"name"`
	Rect    Rect     `json:"rect"`
	Off     string   `json:"off"`
	On      string   `json:"on"`
	Caption *int     `json:"caption"`
	If      string   `json:"if"`
	Cite    []string `json:"cite"`
}

// ButtonShown reports whether button b exists now: it has no condition, or
// the host's condition holds.
func (s *Scene) ButtonShown(b ButtonSpec) bool { return b.If == "" || s.host.Condition(b.If) }

// ShownButtons answers the buttons that exist now, in index order.
func (s *Scene) ShownButtons(buttons []ButtonSpec) []ButtonSpec {
	var out []ButtonSpec
	for _, b := range buttons {
		if s.ButtonShown(b) {
			out = append(out, b)
		}
	}
	return out
}

// ButtonAt answers the index of the first button of buttons whose rectangle
// holds p, -1 for none.
func ButtonAt(buttons []ButtonSpec, p image.Point) int {
	for i, b := range buttons {
		if p.In(b.Rect.Rectangle()) {
			return i
		}
	}
	return -1
}

// Sound requests a slot's own sound, as a "sound <slot>" step does.
func (s *Scene) Sound(slot string) { s.sound(slot, "") }

// GridSpec is one of a page's item grids: Count cells, the first at First
// and each next one Step further, showing the page's list named List from
// its first entry; Back is the page art drawn behind each cell.
type GridSpec struct {
	List  string   `json:"list"`
	First Rect     `json:"first"`
	Step  Point    `json:"step"`
	Count int      `json:"count"`
	Back  string   `json:"back"`
	Cite  []string `json:"cite"`
}

// Cell answers cell i's rectangle, empty outside the grid.
func (g *GridSpec) Cell(i int) image.Rectangle {
	if i < 0 || i >= g.Count {
		return image.Rectangle{}
	}
	return g.First.Rectangle().Add(image.Pt(g.Step[0]*i, g.Step[1]*i))
}

// CellAt answers the first cell that holds p, -1 for none.
func (g *GridSpec) CellAt(p image.Point) int {
	for i := 0; i < g.Count; i++ {
		if p.In(g.Cell(i)) {
			return i
		}
	}
	return -1
}
