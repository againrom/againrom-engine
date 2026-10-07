package ui

import (
	"image"
	"reflect"
	"testing"
	"time"
)

// shopPaneRects are the pane's six rectangles in the order the release
// handler tests them.
var shopPaneRects = []struct {
	name   string
	corner CharacterPaneCorner
}{
	{"backpack", CharacterPaneBackpack},
	{"book", CharacterPaneBook},
	{"mode", CharacterPaneMode},
	{"previous", CharacterPanePrev},
	{"next", CharacterPaneNext},
	{"menu", CharacterPaneMenu},
}

// shopPaneOrigins are the three cell grids an item is dragged from.
var shopPaneOrigins = []struct {
	name   string
	at     image.Point
	origin ShopControl
}{
	{"pack", ShopPackCellRect(1).Min.Add(image.Pt(4, 4)), ShopControl{Kind: ShopControlPackCell, Index: 1}},
	{"shelf", ShopShelfCellRect(0).Min.Add(image.Pt(4, 4)), ShopControl{Kind: ShopControlShelfCell, Index: 0}},
	{"table", ShopTableCellRect(0).Min.Add(image.Pt(4, 4)), ShopControl{Kind: ShopControlTableCell, Index: 0}},
}

// shopPaneRectPoints are three points of one rectangle: its first pixel, its
// last and its middle. The first pixel of the book and mode rectangles is in
// the pane's two top rows, above the figure.
func shopPaneRectPoints(c CharacterPaneCorner) []image.Point {
	r := CharacterPaneCornerRect(TownCharacterRegion, c)
	return []image.Point{r.Min, r.Max.Sub(image.Pt(1, 1)), r.Min.Add(r.Size().Div(2))}
}

// An item dragged from a shelf, pack or table cell and released over any of the
// pane's six rectangles is taken by the pane as an item released on its
// figure is, in either presentation mode: the release names the shown
// character as the destination and presses no rectangle.
func TestTheApplicationTakesAnItemReleasedOnEachPaneRectangleOfTheShop(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, statistics := range []bool{false, true} {
		mode := map[bool]string{false: "figure", true: "statistics"}[statistics]
		for _, from := range shopPaneOrigins {
			for _, rect := range shopPaneRects {
				a, town := shopDragTestApp(t)
				town.stats = statistics
				for _, at := range shopPaneRectPoints(rect.corner) {
					town.dragged, town.clicked, town.controls = nil, nil, nil
					a.step(appInput{CursorX: from.at.X, CursorY: from.at.Y, PrimaryPressed: true}, now)
					a.step(appInput{CursorX: at.X, CursorY: at.Y}, now)
					a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryReleased: true}, now)

					if len(town.dragged) != 1 || town.dragged[0][0] != from.origin || town.dragged[0][1].Kind != ShopControlDoll {
						t.Errorf("%s: %s cell released on %s at %v: dragged = %+v, want one drag from %+v to the doll",
							mode, from.name, rect.name, at, town.dragged, from.origin)
					}
					if len(town.controls) != 0 {
						t.Errorf("%s: %s cell released on %s at %v pressed %+v, want no rectangle pressed",
							mode, from.name, rect.name, at, town.controls)
					}
				}
			}
		}
	}
}

// An item picked up off the figure and released over one of the pane's
// rectangles goes back where it was: no move and no rectangle is pressed.
func TestTheApplicationDollItemReleasedOnAPaneRectangleStaysWorn(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, rect := range shopPaneRects {
		a, town := shopDragTestApp(t)
		for _, at := range shopPaneRectPoints(rect.corner) {
			town.dragged, town.clicked, town.controls = nil, nil, nil
			a.step(appInput{CursorX: shopDragDollPoint.X, CursorY: shopDragDollPoint.Y, PrimaryPressed: true}, now)
			a.step(appInput{CursorX: at.X, CursorY: at.Y}, now)
			a.step(appInput{CursorX: at.X, CursorY: at.Y, PrimaryReleased: true}, now)
			if len(town.dragged) != 0 || len(town.controls) != 0 {
				t.Errorf("doll item released on %s at %v: dragged = %+v, pressed = %+v, want neither",
					rect.name, at, town.dragged, town.controls)
			}
		}
	}
}

// A press and release on a pane rectangle with no item in flight still presses
// the rectangles the shop answers for, and the two it does not answer for stay
// silent. Each point lies in one rectangle only: the backpack overlaps the
// previous arrow and the next arrow overlaps the menu.
func TestTheApplicationTapOnAPaneRectangleOfTheShopStillPressesIt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name string
		at   image.Point
		want []ShopControlKind
	}{
		{"backpack", image.Pt(490, 478), nil},
		{"book", image.Pt(490, 250), []ShopControlKind{ShopControlBook}},
		{"mode", image.Pt(620, 250), []ShopControlKind{ShopControlCharacterMode}},
		{"previous", image.Pt(511, 460), []ShopControlKind{ShopControlPickerPrev}},
		{"next", image.Pt(602, 460), []ShopControlKind{ShopControlPickerNext}},
		{"menu", image.Pt(634, 460), nil},
	} {
		a, town := shopDragTestApp(t)
		a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: tc.at.X, CursorY: tc.at.Y, PrimaryReleased: true}, now)

		var got []ShopControlKind
		for _, c := range town.controls {
			got = append(got, c.Kind)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tap on %s at %v pressed %v, want %v", tc.name, tc.at, got, tc.want)
		}
		if len(town.dragged) != 0 {
			t.Errorf("tap on %s dragged %+v, want none", tc.name, town.dragged)
		}
	}
}
