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
)

// TownSquareControl is one resolved click. Door is the row index Choose(i)
// expects; it is valid only when Kind is TownSquareControlDoor.
type TownSquareControl struct {
	Kind TownSquareControlKind
	Door int
}

// TownSquareView is what the town square draws: the game's composed scene,
// the font its message line uses, the message line every composed room
// shares, and the square's own tip panel, drawn last.
type TownSquareView struct {
	Scene   TownSquareScene
	Font    *text.Font
	Message string
	Tip     TipPanelView
}

// ComposeTownSquare paints the scene, then the message line and the tip
// panel over it. A view with no scene answers a blank canvas; app.go never
// calls this without a scene (see townSquareView).
func ComposeTownSquare(v TownSquareView) *image.RGBA {
	if v.Scene == nil {
		return image.NewRGBA(image.Rect(0, 0, 640, 480))
	}
	size := v.Scene.Size()
	dst := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	v.Scene.Paint(dst)
	if v.Message != "" {
		drawTownShellText(dst, v.Font, v.Message, image.Rect(12, 448, 468, 478), townShellText)
	}
	ComposeTipPanel(dst, v.Tip)
	return dst
}
