package town

import "image"

// SeatSpec is a page's seat grid: Columns seats in a row, Rows rows, the
// first seat at First, each next seat of a row offset by Step and each next
// row by RowStep. Seat i is in row i/Columns and column i%Columns. Back is
// the page art drawn in a populated seat under its actor, Actor the page art
// of a seated actor's frames. A selection requests the page scene's Select
// slot; the selected actor alone advances one frame after FrameMS, compared
// as Compare. A FrameMS of zero runs no seat clock: the page names each
// seat's frame.
type SeatSpec struct {
	First   Rect   `json:"first"`
	Step    Point  `json:"step"`
	RowStep Point  `json:"row-step"`
	Columns int    `json:"columns"`
	Rows    int    `json:"rows"`
	Back    string `json:"back"`
	// HireBack is the back a hire actor's seat draws; empty draws Back.
	HireBack string `json:"hire-back"`
	Actor    string `json:"actor"`
	// Patron is the bound page art of a seated person's own frames, its key
	// formatted with the person's type id; a person whose frames do not load
	// draws Actor. Empty draws Actor for every seat.
	Patron  string `json:"patron"`
	Select  string `json:"select"`
	FrameMS int    `json:"frame-ms"`
	Compare string `json:"compare"`
	// Clock names a process clock of the page scene whose stamp the seat step
	// reads and writes, so the step outlives every entry; empty stamps the
	// step at each entry.
	Clock string   `json:"clock"`
	Cite  []string `json:"cite"`
	// Card and Figure are where the selected seat's person shows its
	// statistics card and its full figure; nil shows neither.
	Card   *SeatCard   `json:"card"`
	Figure *SeatFigure `json:"figure"`
}

// SeatCard is the selected person's statistics card: drawn in Rect over the
// page art Art, its canvas moved right by Offset, the page art Seam
// completing the moved board's right edge.
type SeatCard struct {
	Rect   Rect     `json:"rect"`
	Art    string   `json:"art"`
	Seam   string   `json:"seam"`
	Offset Point    `json:"offset"`
	Cite   []string `json:"cite"`
}

// SeatFigure is the selected person's full figure, drawn with its top-left
// corner at Rect's corner, or centred in Rect when Centre is set, moved by
// Offset and clipped to Rect.
type SeatFigure struct {
	Rect   Rect     `json:"rect"`
	Offset Point    `json:"offset"`
	Centre bool     `json:"centre"`
	Cite   []string `json:"cite"`
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

// Sound requests a slot's own sound, as a "sound <slot>" step does.
func (s *Scene) Sound(slot string) { s.sound(slot, "") }
