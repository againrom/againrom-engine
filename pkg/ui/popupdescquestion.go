package ui

import (
	"fmt"
	"image"

	"againrom/pkg/render/text"
)

// questionPlace is where the editor stands question s in an editor window
// of size window: the corner in the middle and the bounds, as
// questionSpec stands it.
func questionPlace(s popupSpec, title, body *text.Font, window image.Point) (image.Point, image.Rectangle) {
	e := NewMapEditor(title, nil, nil)
	e.catalog = false
	e.SetBodyFont(body)
	e.Layout(window.X, window.Y)
	s.Bounds = e.popupBounds()
	return e.centreCorner(layoutPopup(s, title, body).Rect.Size(), openPlace{}), s.Bounds
}

// gridPictures gives grid g's cells the pictures names name, through
// picture; a cell whose name it does not find shows no picture.
func gridPictures(g *popupGrid, names []string, picture func(string) *image.RGBA) {
	if g == nil || picture == nil {
		return
	}
	for i, name := range names {
		if name != "" && i < len(g.Cells) {
			g.Cells[i].Icon = picture(name)
		}
	}
}

// cardPicture is the portrait of the test party's card n for the first
// kind of hero, a picture name "card:N"; nil for none.
func (e *MapEditor) cardPicture(name string) *image.RGBA {
	var n int
	if _, err := fmt.Sscanf(name, "card:%d", &n); err != nil || e.doc == nil {
		return nil
	}
	host, ok := e.doc.Edits.(InspectionTestPlayParty)
	if !ok {
		return nil
	}
	if cards := host.TestPlayCards(0); n >= 0 && n < len(cards) {
		return cards[n].Portrait
	}
	return nil
}
