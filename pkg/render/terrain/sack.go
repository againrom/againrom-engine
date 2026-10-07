package terrain

import "image"

// SackPlace returns one sack's placement: StaticAnchor called with the
// canvas equal to the drawn frame's own size and the centre equal to that
// canvas's centre, so the frame's centre pixel lands on the cell's ground
// point.
//
// With canvas == frame size and centre == canvas/2, StaticAnchor's own
// arithmetic collapses to anchorX = frameW/2, anchorY = frameH/2: the
// frame's own centre pixel, and nothing else. A synthetic UnitClass carrying
// only a canvas and a centre was rejected for the reason UnitPlace's own
// precedent already was: every other field of it would be a lie, so this
// function takes the two numbers it needs and no class-shaped thing to carry
// them in.
//
// lift is this cell's own height (0 in the flat geometry) and originY the
// render's vertical origin, both handed straight to StaticAnchor exactly as
// UnitPlace hands them over: one cell's flat and displaced placements differ
// by exactly -(lift+originY) in Y and by zero in X.
//
// The returned placement carries no Class and no Mirror. A sack neither
// animates nor reflects, so AnimateStatics' re-selection arm — which reads
// Class and leaves a class-less placement exactly as built — has nothing
// to do with one, and the shadow pass, which reads only the three
// class-driven lists, never reaches it either.
func SackPlace(col, row int, f *StaticFrame, lift, originY int) (StaticPlacement, bool) {
	if f == nil {
		return StaticPlacement{}, false
	}
	destX, destY, anchorX, anchorY := StaticAnchor(
		col, row, f.Width, f.Height, f.Width/2, f.Height/2, f.Width, f.Height, lift, originY)
	return StaticPlacement{
		Cell:    image.Point{X: col, Y: row},
		TopLeft: image.Point{X: destX, Y: destY},
		Anchor:  image.Point{X: anchorX, Y: anchorY},
		Frame:   f,
	}, true
}
