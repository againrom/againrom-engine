package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// The hover box's line pitch and margins (MENU-128).
const (
	hoverPitch    = 14
	hoverExtraW   = 11
	hoverExtraH   = 5
	hoverTextX    = 5
	hoverTextY    = 4
	hoverBallSize = 4
	// hoverBesideGap is how far a box beside the pointer stands from it, on
	// both axes.
	hoverBesideGap = 14
)

// hoverAnchorKind is how a box is placed against its anchor point.
type hoverAnchorKind uint8

const (
	// hoverAbove puts the lower-left corner at the point and keeps the box
	// inside bounds on all sides.
	hoverAbove hoverAnchorKind = iota
	// hoverBeside puts the top-left corner hoverBesideGap right and down of
	// the point and pulls the box back from the right and bottom edges only.
	hoverBeside
)

// hoverAnchor is where a hover box stands: a point in bounds.
type hoverAnchor struct {
	kind   hoverAnchorKind
	at     image.Point
	bounds image.Rectangle
}

// hoverDescription is one hover box: its lines, its font, its style and its
// anchor. placeHover is the one thing that turns it into a picture.
type hoverDescription struct {
	lines  []string
	font   *text.Font
	style  *hoverStyle
	anchor hoverAnchor
}

// hoverLinePitch is the distance between two lines of a box in shape s.
func (s *hoverShape) linePitch(f *text.Font) int {
	if s.pitch != 0 {
		return s.pitch
	}
	return f.Height() + s.pitchPad
}

// hoverBoxSize is the box for lines in shape s: the widest line plus the
// shape's width, the pitch per line plus its height.
func hoverBoxSize(lines []string, f *text.Font, s *hoverShape) image.Point {
	width := 0
	for _, ln := range lines {
		w, _ := f.Measure(ln)
		width = max(width, w)
		if s.advance {
			width = max(width, f.Advance(ln))
		}
	}
	return image.Pt(width+s.padW, s.linePitch(f)*len(lines)+s.padH)
}

// composeHoverBox is the one hover box painter (MENU-128). The bevelled box
// is the green fill, the two-colour gold bevel, Ball.bmp at the four corners
// and each line at (L+5,T+4+14i); its right edge R is its last column and
// its bottom B one past its last row, so every side keeps a one-pixel margin
// outside the bevel. Without a ball picture the corners stay unpainted. The
// outlined box is the fill under a one-pixel outline. The text draws in
// style s, or popupTextColor without a shadow when s is nil; a style's box
// names the colours instead and its shape the geometry.
func composeHoverBox(lines []string, f *text.Font, ball image.Image, s *hoverStyle) *image.RGBA {
	if f == nil || len(lines) == 0 {
		return nil
	}
	shape := s.shapeOf()
	if !shape.outlined && f.Height() <= 0 {
		return nil
	}
	size := hoverBoxSize(lines, f, shape)
	if !shape.outlined && size.X <= shape.padW {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	recordWidget(widgetHoverBox, img.Bounds(), lines)
	box := hoverSharedBox
	if s != nil && s.box != nil {
		box = *s.box
	}
	if shape.outlined {
		drawFrame(img, panelFrame(img.Bounds(), box.fill, box.light))
	} else {
		paintHoverBevel(img, box, ball, shape)
	}
	ink := popupTextColor
	if s != nil {
		ink = s.ink
	}
	pitch := shape.linePitch(f)
	for i, ln := range lines {
		x, y := shape.textX, shape.textY+i*pitch
		if s != nil && s.shadow != nil {
			f.DrawFlat(img, ln, x+1, y+1, *s.shadow)
		}
		f.Draw(img, ln, x, y, ink)
	}
	return img
}

// paintHoverBevel is the bevelled frame: fill, bevel and corner pictures.
func paintHoverBevel(img *image.RGBA, box hoverBoxInk, ball image.Image, shape *hoverShape) {
	size := img.Bounds().Size()
	r, b := size.X-1, size.Y
	draw.Draw(img, image.Rect(1, 1, r-1, b-2), &image.Uniform{C: box.fill}, image.Point{}, draw.Src)
	hline := func(y int, c color.RGBA) {
		for x := 3; x <= r-3; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	vline := func(x int, c color.RGBA) {
		for y := 3; y <= b-4; y++ {
			img.SetRGBA(x, y, c)
		}
	}
	hline(1, box.light)
	hline(2, box.dark)
	hline(b-3, box.light)
	hline(b-2, box.dark)
	vline(1, box.light)
	vline(2, box.dark)
	vline(r-2, box.light)
	vline(r-1, box.dark)
	if ball != nil && shape.corners {
		for _, at := range []image.Point{{0, 0}, {r - 3, 0}, {0, b - hoverBallSize}, {r - 3, b - hoverBallSize}} {
			src := ball.Bounds()
			dst := image.Rect(at.X, at.Y, at.X+hoverBallSize, at.Y+hoverBallSize).Intersect(img.Bounds())
			draw.Draw(img, dst, ball, src.Min, draw.Over)
		}
	}
}

// placeHover draws description d and returns its picture and the top-left
// corner the anchor gives it. ball is the corner picture, nil for none.
func placeHover(d hoverDescription, ball image.Image) (*image.RGBA, image.Point, bool) {
	pic := composeHoverBox(d.lines, d.font, ball, d.style)
	if pic == nil {
		return nil, image.Point{}, false
	}
	size, a := pic.Bounds().Size(), d.anchor
	switch a.kind {
	case hoverBeside:
		at := a.at.Add(image.Pt(hoverBesideGap, hoverBesideGap))
		if at.X+size.X > a.bounds.Max.X {
			at.X = a.bounds.Max.X - size.X
		}
		if at.Y+size.Y > a.bounds.Max.Y {
			at.Y = a.bounds.Max.Y - size.Y
		}
		return pic, at, true
	}
	// The cursor anchors the lower-left corner (TEXT-HOVERPAINT-053).
	// Growing text moves the top upward, leaving the hovered item visible.
	at := image.Pt(a.at.X, a.at.Y-size.Y)
	if at.X+size.X > a.bounds.Max.X {
		at.X = a.bounds.Max.X - size.X
	}
	if at.Y+size.Y > a.bounds.Max.Y {
		at.Y = a.bounds.Max.Y - size.Y
	}
	at.X, at.Y = max(a.bounds.Min.X, at.X), max(a.bounds.Min.Y, at.Y)
	return pic, at, true
}

// drawHover paints description d onto dst, as a picture over dst, with the
// text capture of its glyphs moved to where the box stands.
func drawHover(dst *image.RGBA, d hoverDescription) {
	start := markCapture()
	pic, at, ok := placeHover(d, nil)
	if !ok {
		return
	}
	shiftCapture(start, at)
	draw.Draw(dst, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Over)
}
