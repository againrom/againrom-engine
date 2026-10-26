package ui

import (
	"image"

	"againrom/pkg/render/text"
)

// TownSquareScene is the composed town square a game supplies: it paints
// itself and answers what a point of it does. The ui draws the message line
// and the tip panel over it and owns no fact of the square.
type TownSquareScene interface {
	// Size is the scene's size; Paint draws it onto dst at the origin.
	Size() image.Point
	Paint(dst *image.RGBA)
	// ControlAt answers the door or the menu a click at p reaches.
	ControlAt(p image.Point) (TownSquareControl, bool)
	// TipAt answers the main text slot of the tooltip at p.
	TipAt(p image.Point) (int, bool)
}

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

	// Dolls are the room's other worn figures whose items answer a hover.
	Dolls []TownSquareDoll
}

// TownSquareDoll is a worn figure a room shows: its slot map with the map's
// origin at At, answered inside Clip, and each slot's worn item text.
type TownSquareDoll struct {
	At   image.Point
	Clip image.Rectangle
	Mask *SlotMask
	Info [12][]string
}

// LinesAt is the item text of the worn slot whose figure pixel is under p.
func (d TownSquareDoll) LinesAt(p image.Point) ([]string, bool) {
	if d.Mask == nil || !p.In(d.Clip) {
		return nil, false
	}
	n, ok := d.Mask.At(p.X-d.At.X, p.Y-d.At.Y)
	if !ok || n < 1 || n > len(d.Info) || len(d.Info[n-1]) == 0 {
		return nil, false
	}
	return d.Info[n-1], true
}

// heroHoverLines is the item text of the pane's worn slot under p.
func (v TownSquareView) heroHoverLines(p image.Point) ([]string, bool) {
	if !v.Pane {
		return nil, false
	}
	return TownHeroHoverLines(TownSurfaceView{Hero: v.Hero, HeroSlotMask: v.HeroSlotMask, HeroSlotInfo: v.HeroSlotInfo}, p)
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
