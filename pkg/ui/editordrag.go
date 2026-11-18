package ui

import (
	"image"
	"image/draw"
)

// popupDrop is one drag and drop rule of a window: a press on a part of
// kind From, a grid cell or a canvas box, carried onto a part of kind To.
// A grid target is the whole grid; a canvas box target is a box that takes
// a press other than the one carried, and with Empty only a box marked
// Empty. Word names the rule to the window that acts on the drop.
type popupDrop struct {
	From, To popupPart
	Empty    bool
	Word     string
}

// popupCarry is a press on a part a drop rule takes from, held while the
// pointer moves: the part and its index, where the press fell and where
// the pointer is. It is moved once the pointer has gone more than
// carryStart pixels from the press; a release before that is a click.
type popupCarry struct {
	from      popupPart
	index     int
	start, at image.Point
	moved     bool
}

// carryStart is how far the pointer goes from a press before it carries.
const carryStart = 4

// carries reports whether a drop rule of s takes a carry from index n of
// part: a grid cell that takes a press, or a canvas box that is not Empty.
func (s popupSpec) carries(part popupPart, n int) bool {
	for _, d := range s.Drops {
		switch {
		case d.From != part:
		case part == popupGridPart && s.Grid != nil && n >= 0 && n < len(s.Grid.Cells):
			return !s.Grid.Cells[n].Inert
		case part == popupCanvasBox && s.Canvas != nil && n >= 0 && n < len(s.Canvas.Boxes):
			return !s.Canvas.Boxes[n].Empty
		}
	}
	return false
}

// dropAt is the rule of s that takes carry c onto p in layout l, the
// index of the box it lands on (-1 on the grid) and the rectangle a drop
// there outlines; false where no rule takes it. A drop lands where a press
// at p would: on a grid cell that takes a press or the grid's bar, the
// whole grid outlined, or on a canvas box.
func (s popupSpec) dropAt(l popupLayout, c popupCarry, p image.Point) (popupDrop, int, image.Rectangle, bool) {
	part, i := l.hit(p)
	for _, d := range s.Drops {
		if d.From != c.from || d.To != part && !(d.To == popupGridPart && part == popupGridBar) {
			continue
		}
		if part != popupCanvasBox {
			return d, -1, l.Grid, true
		}
		if c.from == popupCanvasBox && i == c.index || d.Empty && !s.Canvas.Boxes[i].Empty {
			continue
		}
		return d, i, l.CanvasBoxes[i].rect, true
	}
	return popupDrop{}, 0, image.Rectangle{}, false
}

// carried is the picture a carry from index n of part takes, and the text
// a cell without a picture shows in its place.
func (s popupSpec) carried(part popupPart, n int) (*image.RGBA, string) {
	switch {
	case part == popupGridPart && s.Grid != nil && n >= 0 && n < len(s.Grid.Cells):
		return s.Grid.Cells[n].Icon, s.Grid.Cells[n].Label
	case part == popupCanvasBox && s.Canvas != nil && n >= 0 && n < len(s.Canvas.Boxes):
		return s.Canvas.Boxes[n].Held, ""
	}
	return nil, ""
}

// carryPress holds a press at p on index n of part, laid out as l from s,
// when a drop rule of s takes a carry from it: the press acts on release.
func (e *MapEditor) carryPress(s popupSpec, part popupPart, n int, p image.Point) bool {
	if !s.carries(part, n) {
		return false
	}
	e.carry = &popupCarry{from: part, index: n, start: p, at: p}
	return true
}

// carryMove follows the held carry to p while the button is down; on the
// release it is a click of the part pressed, or the drop of the rule that
// takes it where it stands, and none where no rule does.
func (e *MapEditor) carryMove(p image.Point, down bool) {
	c := e.carry
	if c == nil {
		return
	}
	c.at = p
	if d := p.Sub(c.start); d.X > carryStart || d.X < -carryStart || d.Y > carryStart || d.Y < -carryStart {
		c.moved = true
	}
	if down {
		return
	}
	e.carry = nil
	s, _, ok := e.objectSpec()
	if !ok {
		return
	}
	if !c.moved {
		e.modCarryClick(s, c.from, c.index)
		return
	}
	if d, to, _, ok := s.dropAt(layoutPopup(s, e.font, e.bodyFont()), *c, p); ok {
		e.modCarryDrop(s, d, *c, to)
	}
}

// drawCarry draws the carry over window s: the outline of the place a
// drop there takes, and the carried picture under the pointer.
func (e *MapEditor) drawCarry(dst *image.RGBA, s popupSpec) {
	c := e.carry
	if c == nil || !c.moved {
		return
	}
	ink := editorPopupInk
	if _, _, r, ok := s.dropAt(layoutPopup(s, e.font, e.bodyFont()), *c, c.at); ok {
		drawFrame(dst, frameSpec{Kind: frameOutline, Rect: r, Border: ink.Title, Width: 2})
	}
	pic, label := s.carried(c.from, c.index)
	side := bookCellSize
	if pic != nil {
		side = max(pic.Bounds().Dx(), pic.Bounds().Dy())
	}
	box := image.Rectangle{Min: c.at.Add(image.Pt(2, 2)), Max: c.at.Add(image.Pt(2+side, 2+side))}
	if pic == nil {
		draw.Draw(dst, box, &image.Uniform{C: ink.TitleGround}, image.Point{}, draw.Src)
		if body := e.bodyFont(); body != nil {
			at := box.Min.Add(image.Pt((side-body.Advance(label))/2, (side-fontHeight(body, 15))/2))
			body.Draw(dst.SubImage(box).(*image.RGBA), label, at.X, at.Y, ink.Label)
		}
	} else {
		draw.Draw(dst, box, pic, pic.Bounds().Min, draw.Over)
	}
	drawFrame(dst, frameSpec{Kind: frameOutline, Rect: box, Border: ink.Title})
}
