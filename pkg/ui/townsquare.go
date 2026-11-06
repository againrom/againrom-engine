package ui

import (
	"fmt"
	"image"

	"againrom/pkg/render/text"
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
)

// TownSquareControl is one resolved click. Door is the row index Choose(i)
// expects; it is valid only when Kind is TownSquareControlDoor or
// TownSquareControlSelect.
type TownSquareControl struct {
	Kind TownSquareControlKind
	Door int
}

// TownSquareSelector is the optional seam a composed scene's selection
// crosses: SelectTownSquare selects row i and changes no row.
type TownSquareSelector interface {
	SelectTownSquare(i int)
}

// TownSquareView is what the town square draws: the game's composed scene,
// the font its message line uses, the message line every composed room
// shares, and the square's own tip panel, drawn last.
type TownSquareView struct {
	Scene   TownSquareScene
	Font    *text.Font
	Message string
	Tip     TipPanelView

	// Pane shows Hero's character pane over the scene, its corners hit
	// before the scene's controls. HeroSlotMask and HeroSlotInfo are its
	// doll's slot map and worn item text, as on TownSurfaceView.
	Pane         bool
	Hero         TownCharacterView
	HeroSlotMask *SlotMask
	HeroSlotInfo [12][]string

	// Dolls are the room's other worn figures whose items answer a hover,
	// each described by its slot map placed at At and answered inside Clip.
	Dolls []TownSquareDoll
}

// heroDollHover describes the pane's figure, or no doll while the room shows
// no pane.
func (v TownSquareView) heroDollHover() DollHover {
	if !v.Pane {
		return DollHover{}
	}
	return townHeroDollHover(v.Hero, v.HeroSlotMask, v.HeroSlotInfo)
}

// roomDollHover describes the room's i'th worn figure with its tip.
func (v TownSquareView) roomDollHover(i int) DollHover {
	d := v.Dolls[i]
	d.Tip = DollTip{Kind: tooltipText, Key: fmt.Sprintf("room-item/%d", i)}
	return d
}

// heroHoverLines is the item text of the pane's worn slot under p.
func (v TownSquareView) heroHoverLines(p image.Point) ([]string, bool) {
	return v.heroDollHover().LinesAt(p)
}

// TownSquarePaneScreen is the optional seam a click on a room's character
// pane corner crosses.
type TownSquarePaneScreen interface {
	TownSquarePane(c CharacterPaneCorner)
}

// townSquarePaneCorner is the pane corner a click at p acts on.
func townSquarePaneCorner(v TownSquareView, p image.Point) (CharacterPaneCorner, bool) {
	if !v.Pane {
		return 0, false
	}
	c, hit, acts := characterPaneCornerControl(v.Hero, p)
	return c, hit && acts
}

// ComposeTownSquare paints the scene, its character pane, then the message
// line and the tip panel over it. A view with no scene answers a blank
// canvas; app.go never calls this without a scene (see townSquareView).
func ComposeTownSquare(v TownSquareView) *image.RGBA {
	if v.Scene == nil {
		return image.NewRGBA(image.Rect(0, 0, 640, 480))
	}
	size := v.Scene.Size()
	dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	v.Scene.Paint(dst)
	if v.Pane {
		DrawTownCharacterRegion(dst, v.Hero)
	}
	if v.Message != "" {
		drawTownShellText(dst, v.Font, v.Message, image.Rect(12, 448, 468, 478), townShellText)
	}
	ComposeTipPanel(dst, v.Tip)
	return dst
}
