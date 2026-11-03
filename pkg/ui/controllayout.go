package ui

import "image"

// controlLayout is a screen's control geometry: its controls and their
// rectangles, as rows. The draw reads a control's rectangle from it and the
// hit test asks it which rectangle holds a point, so a control has one
// rectangle for both.
type controlLayout []controlRow

// controlRow is one control: its identity on its screen and its rectangle in
// the 640x480 frame.
type controlRow struct {
	id   int
	rect image.Rectangle
}

// rect is control id's rectangle, empty when the layout has no such row.
func (l controlLayout) rect(id int) image.Rectangle {
	for _, r := range l {
		if r.id == id {
			return r.rect
		}
	}
	return image.Rectangle{}
}

// holds reports whether p is inside control id.
func (l controlLayout) holds(id int, p image.Point) bool { return p.In(l.rect(id)) }

// The controls of a SavePage's layout: the three push buttons by their
// latch ids, then the name box.
const pageNameControl = loadCancelButton + 1

// layout is the page's control geometry, read from its button and name box
// rectangles.
func (p SavePage) layout() controlLayout {
	l := make(controlLayout, 0, len(p.Buttons)+1)
	for i, b := range p.Buttons {
		l = append(l, controlRow{id: i, rect: b.Rect})
	}
	return append(l, controlRow{id: pageNameControl, rect: p.NameBox})
}

func (p SavePage) saveLayout() controlLayout {
	l := controlLayout{
		{int(saveDirectoryControl), p.DirectoryBox},
		{int(saveOpenControl), p.Navigation[0].Rect},
		{int(saveUpControl), p.Navigation[1].Rect},
		{int(saveListControl), p.List},
		{int(saveNameControl), p.NameBox},
	}
	if p.directories {
		l[3].rect.Max.X += 24
	}
	for i, c := range []saveControl{saveWriteControl, saveDeleteControl, saveCancelControl} {
		l = append(l, controlRow{int(c), p.Buttons[i].Rect})
	}
	return l
}
