package ui

import "image"

// fakeSquareScene is a TownSquareScene fixture: it fills the canvas with one
// byte and answers fixed controls and tips at exact points.
type fakeSquareScene struct {
	fill     uint8
	controls map[image.Point]TownSquareControl
	tips     map[image.Point]int
}

func (s *fakeSquareScene) Size() image.Point { return image.Pt(640, 480) }

func (s *fakeSquareScene) Paint(dst *image.RGBA) {
	for i := range dst.Pix {
		dst.Pix[i] = s.fill
	}
}

func (s *fakeSquareScene) ControlAt(p image.Point) (TownSquareControl, bool) {
	c, ok := s.controls[p]
	return c, ok
}

func (s *fakeSquareScene) TipAt(p image.Point) (int, bool) {
	n, ok := s.tips[p]
	return n, ok
}

// squareDoor and squareMenu are the two control kinds a scene answers.
func squareDoor(i int) TownSquareControl {
	return TownSquareControl{Kind: TownSquareControlDoor, Door: i}
}

var squareMenu = TownSquareControl{Kind: TownSquareControlMenu}
