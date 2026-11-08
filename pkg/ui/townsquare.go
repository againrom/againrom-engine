package ui

import (
	"fmt"
	"image"
)

// TownSquareScene is the composed town square a game supplies: it paints
// itself and answers what a point of it does. The ui draws the message line
// and the tip panel over it and owns no fact of the square.
type TownSquareScene = RoomScene

// TownSquareControlKind is what a click on the square's picture resolved to:
// one of the four doors Choose(i) already accepts, or the statue, which
// opens the save/load mini-menu rather than a door (0143's own seam).
type TownSquareControlKind uint8

const (
	TownSquareControlNone TownSquareControlKind = iota
	TownSquareControlDoor
	TownSquareControlMenu
	// TownSquareControlSelect selects row Door without choosing it: a seat
	// on a composed page, whose own button then acts on the selection.
	TownSquareControlSelect
	// TownSquareControlButton invokes the composed room's button by index.
	TownSquareControlButton
)

// TownSquareControl is one resolved click. Door is the row index Choose(i)
// expects; it is valid only when Kind is TownSquareControlDoor or
// TownSquareControlSelect. With TownSquareControlButton it is a button index.
type TownSquareControl struct {
	Kind TownSquareControlKind
	Door int
}

// TownSquareSelector is the optional seam a composed scene's selection
// crosses: SelectTownSquare selects row i and changes no row.
type TownSquareSelector interface {
	SelectTownSquare(i int)
}

// TownSquareButtonScreen handles a room button independent of its row action.
type TownSquareButtonScreen interface {
	TownSquareButton(i int) TownAction
}

// roomDollHover describes the room's i'th worn figure with its tip.
func (v TownRoomView) roomDollHover(i int) DollHover {
	d := v.Dolls[i]
	d.Tip = DollTip{Kind: tooltipText, Key: fmt.Sprintf("room-item/%d", i)}
	return d
}

// heroHoverLines is the item text of the pane's worn slot under p.
func (v TownRoomView) heroHoverLines(p image.Point) ([]string, bool) {
	if !v.Pane {
		return nil, false
	}
	return v.heroDollHover().LinesAt(p)
}

// TownSquarePaneScreen is the optional seam a click on a room's character
// pane corner crosses.
type TownSquarePaneScreen interface {
	TownSquarePane(c CharacterPaneCorner)
}

// townSquarePaneCorner is the pane corner a click at p acts on.
func townSquarePaneCorner(v TownSquareView, p image.Point) (CharacterPaneCorner, bool) {
	c, hit := v.ControlAt(p)
	return c.Corner, hit && c.Pane
}

// ComposeTownSquare paints the scene, its character pane, then the message
// line and the tip panel over it. A view with no scene answers a blank
// canvas; app.go never calls this without a scene (see townSquareView).
func ComposeTownSquare(v TownSquareView) *image.RGBA {
	return ComposeTownRoom(v)
}
