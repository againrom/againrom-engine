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
// names the colours instead and its shape the geometry; its picture draws at
// its own size under the lines.
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
	var picture image.Image
	if s != nil && s.picture != nil && !s.picture.Bounds().Empty() {
		picture = s.picture
		size.X = max(size.X, picture.Bounds().Dx()+shape.padW)
		size.Y += picture.Bounds().Dy() + shape.textY
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
	if picture != nil {
		at := image.Pt(shape.textX, shape.textY+len(lines)*pitch)
		draw.Draw(img, picture.Bounds().Sub(picture.Bounds().Min).Add(at), picture, picture.Bounds().Min, draw.Over)
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
	box := image.Rectangle{Max: size}
	switch a.kind {
	case hoverBeside:
		box = box.Add(a.at.Add(image.Pt(hoverBesideGap, hoverBesideGap)))
		return pic, placeTip(tipPlace{Box: box, PullRight: true, PullBottom: true}, a.bounds).Min, true
	}
	// The cursor anchors the lower-left corner (TEXT-HOVERPAINT-053).
	// Growing text moves the top upward, leaving the hovered item visible.
	box = box.Add(image.Pt(a.at.X, a.at.Y-size.Y))
	return pic, placeTip(tipPlace{Box: box, PullRight: true, PullBottom: true, PushTopLeft: true}, a.bounds).Min, true
}

// tipPlace is one tip box to stand in a page: where it wants to be and what
// it does at the page's edges.
type tipPlace struct {
	Box image.Rectangle
	// PullRight moves a box past the right edge back to it.
	PullRight bool
	// PullBottom moves a box past the bottom edge back to it.
	PullBottom bool
	// FlipBy moves a box past the bottom edge up by this much instead,
	// over its anchor; zero for none.
	FlipBy int
	// PushTopLeft moves a box past the left or top edge back to it.
	PushTopLeft bool
	// Clip cuts what still stands outside the page.
	Clip bool
}

// placeTip is the one placement of a tip in its page: every tip box takes its
// position from here, right and bottom edges first, then left and top.
func placeTip(t tipPlace, bounds image.Rectangle) image.Rectangle {
	r := t.Box
	if t.PullRight && r.Max.X > bounds.Max.X {
		r = r.Sub(image.Pt(r.Max.X-bounds.Max.X, 0))
	}
	if r.Max.Y > bounds.Max.Y {
		switch {
		case t.FlipBy != 0:
			r = r.Sub(image.Pt(0, t.FlipBy))
		case t.PullBottom:
			r = r.Sub(image.Pt(0, r.Max.Y-bounds.Max.Y))
		}
	}
	if t.PushTopLeft {
		r = r.Add(image.Pt(max(0, bounds.Min.X-r.Min.X), max(0, bounds.Min.Y-r.Min.Y)))
	}
	if t.Clip {
		r = r.Intersect(bounds)
	}
	return r
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
