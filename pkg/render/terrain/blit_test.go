package terrain_test

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

const (
	blitFrameW, blitFrameH = 5, 4
	blitDstW, blitDstH     = 16, 12

	// The index whose palette entry carries a ZERO ALPHA. An opaque pixel holding
	// it must still paint: an entry is a colour at full opacity and its alpha is
	// not a transparency channel, so a blit reading that alpha would write
	// invisible pixels here.
	blitAlphaZeroIndex = 9

	// The index every TRANSPARENT pixel of the fixture carries. It is non-zero and
	// no opaque pixel uses it, so a blit that ignored StaticPixel.Opaque paints a
	// colour that appears nowhere in the correct output.
	blitHoleIndex = 77
)

// blitArt is AC-4's frame, one rune per pixel in row-major order. '.' is a
// transparent hole; every other rune is an opaque pixel at the index blitIndex
// gives it.
//
// The pattern is asymmetric in both axes and repeats in neither, so a blit that
// transposed the frame, shifted it by a row or walked its rows in the wrong order
// lands different colours in different places rather than the same ones. The
// frame is deliberately not square for the same reason.
var blitArt = [blitFrameH]string{
	"z.0.h",
	".hz..",
	"0..z0",
	".h0..",
}

// blitIndex maps a rune of the art to its palette index. ZERO IS AMONG THEM and
// it carries opaque pixels ('0'): transparency is structural, so index 0 is an
// ordinary colour and a blit treating it as a hole must fail here (spec, "Class
// to sprite frame").
var blitIndex = map[rune]uint8{'0': 0, 'z': 200, 'h': blitAlphaZeroIndex}

// blitPalette gives every index a distinct colour derived from the index itself
// — 3*i+1 is injective over the whole uint8 domain — so a blit resolving the
// wrong index writes a wrong colour rather than a plausible one.
func blitPalette() [256]color.RGBA {
	var pal [256]color.RGBA
	for i := range pal {
		pal[i] = color.RGBA{R: uint8(3*i + 1), G: uint8(200 - i), B: uint8(7*i + 5), A: 0xff}
	}
	pal[blitAlphaZeroIndex].A = 0
	return pal
}

// blitTestFrame builds the fixture fresh on every call: a frame mixing
// transparent pixels with palette-index ones, its own palette riding on it.
func blitTestFrame(t *testing.T) *terrain.StaticFrame {
	t.Helper()

	f := &terrain.StaticFrame{
		Width:   blitFrameW,
		Height:  blitFrameH,
		Pixels:  make([]terrain.StaticPixel, blitFrameW*blitFrameH),
		Palette: blitPalette(),
	}
	for y, art := range blitArt {
		if len(art) != blitFrameW {
			t.Fatalf("row %d of the art is %d runes, want %d", y, len(art), blitFrameW)
		}
		for x, r := range art {
			if r == '.' {
				f.Pixels[y*blitFrameW+x] = terrain.StaticPixel{Index: blitHoleIndex}
				continue
			}
			idx, ok := blitIndex[r]
			if !ok {
				t.Fatalf("row %d column %d holds %q, which blitIndex does not name", y, x, r)
			}
			f.Pixels[y*blitFrameW+x] = terrain.StaticPixel{Index: idx, Opaque: true}
		}
	}
	return f
}

// blitBackground varies in BOTH axes, so a blit landing one row or one column
// off overwrites a colour that differs from the one it should have — and every
// pixel it must not touch is checkable against the same rule.
func blitBackground(x, y int) color.RGBA {
	return color.RGBA{R: uint8(11*x + 3), G: uint8(29 + 5*y), B: 0x44, A: 0xff}
}

func blitDest() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, blitDstW, blitDstH))
	for y := 0; y < blitDstH; y++ {
		for x := 0; x < blitDstW; x++ {
			dst.SetRGBA(x, y, blitBackground(x, y))
		}
	}
	return dst
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	out := image.NewRGBA(src.Bounds())
	copy(out.Pix, src.Pix)
	return out
}

// checkBlit asserts every pixel of after against the per-pixel oracle described
// at the top of this file, over after's own bounds.
//
// before is the destination as it stood before the blit, in the same coordinate
// frame. A pixel the frame does not cover, and a pixel whose source is a hole,
// must be byte-identical to it.
func checkBlit(t *testing.T, name string, before, after *image.RGBA, f *terrain.StaticFrame, destX, destY int) {
	t.Helper()

	bad := 0
	b := after.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sx, sy := x-destX, y-destY
			want, why := before.RGBAAt(x, y), "the frame does not cover this pixel, so it keeps the byte it had"
			if sx >= 0 && sy >= 0 && sx < f.Width && sy < f.Height {
				if px := f.Pixels[sy*f.Width+sx]; px.Opaque {
					c := f.Palette[px.Index]
					want = color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
					why = "an opaque pixel takes its palette colour at FULL opacity, whatever alpha the entry carries"
				} else {
					why = "a transparent pixel leaves what is beneath untouched — no blend, no clear, no write"
				}
			}
			if got := after.RGBAAt(x, y); got != want {
				bad++
				if bad <= 5 {
					t.Errorf("%s: pixel (%d,%d) = %v, want %v — %s", name, x, y, got, want, why)
				}
			}
		}
	}
	if bad > 5 {
		t.Errorf("%s: %d pixels wrong in total", name, bad)
	}
}

// TestBlitStaticClipsOnEveryEdge covers SC-4's first half (AC-4): the frame
// blitted wholly inside, over each of the four edges, into two corners at
// once, flush against the canvas, and wholly off the image in five
// directions.
//
// Every case is checked over the WHOLE destination, so "clipped without an
// out-of-bounds write" is not merely the absence of a panic: a write one pixel
// past an edge would either panic or land somewhere this oracle is watching. A
// read outside the frame cannot be silent either — Go's own bounds check makes it
// a panic, and this test would report it as one.
func TestBlitStaticClipsOnEveryEdge(t *testing.T) {
	f := blitTestFrame(t)

	cases := []struct {
		name         string
		destX, destY int
		offImage     bool
		why          string
	}{
		{name: "wholly inside", destX: 5, destY: 4,
			why: "the ordinary case: ten opaque pixels at their offsets, ten holes leaving the background"},
		{name: "flush at the origin", destX: 0, destY: 0,
			why: "touching two edges without crossing either"},
		{name: "flush at the far corner", destX: blitDstW - blitFrameW, destY: blitDstH - blitFrameH,
			why: "the last pixel of the frame is the last pixel of the canvas"},
		{name: "over the left edge", destX: -2, destY: 4,
			why: "two source columns fall off; the third column must land at x = 0"},
		{name: "over the right edge", destX: blitDstW - 2, destY: 4,
			why: "three source columns fall off the far side"},
		{name: "over the top edge", destX: 5, destY: -3,
			why: "three source rows fall off; the fourth must land at y = 0, which is where an off-by-one row offset shows"},
		{name: "over the bottom edge", destX: 5, destY: blitDstH - 1,
			why: "only the frame's first row survives"},
		{name: "into the top-left corner", destX: -3, destY: -2,
			why: "both axes clipped at once, which a clip written per axis in sequence can still get wrong"},
		{name: "into the bottom-right corner", destX: blitDstW - 1, destY: blitDstH - 3,
			why: "one column by three rows survive at the opposite corner"},
		{name: "one column short of the left edge", destX: -blitFrameW, destY: 4, offImage: true,
			why: "the frame ends exactly where the canvas begins: half-open rectangles, so nothing survives"},
		{name: "one column past the right edge", destX: blitDstW, destY: 4, offImage: true,
			why: "the frame begins exactly where the canvas ends"},
		{name: "one row above the top edge", destX: 5, destY: -blitFrameH, offImage: true,
			why: "the same argument on the other axis"},
		{name: "one row below the bottom edge", destX: 5, destY: blitDstH, offImage: true,
			why: "the same again, below"},
		{name: "far off in both axes", destX: 1 << 20, destY: -(1 << 20), offImage: true,
			why: "wholly off-image at a magnitude no map produces: the no-op must fall out of the rect arithmetic, not out of a special case"},
	}

	for _, c := range cases {
		dst := blitDest()
		before := cloneRGBA(dst)

		terrain.BlitStatic(dst, f, c.destX, c.destY)

		checkBlit(t, c.name, before, dst, f, c.destX, c.destY)

		identical := bytes.Equal(before.Pix, dst.Pix)
		if c.offImage && !identical {
			t.Errorf("%s: the destination changed, want a byte-identical no-op — %s", c.name, c.why)
		}
		if !c.offImage && identical {
			t.Errorf("%s: the destination is byte-identical, so nothing was drawn at all — %s", c.name, c.why)
		}
	}
}

// TestBlitStaticWritesOnlyInsideASubImage pins that the clip is taken against
// dst.Bounds() and the store addresses dst through PixOffset.
//
// A destination that does not begin at the origin is the case that separates
// those from the plausible alternatives — a clip against image.Rect(0, 0, Dx,
// Dy), or a store indexing Pix as y*Stride + 4*x. Both agree with the contract on
// every origin-anchored canvas in the tree and both write outside a sub-image
// here.
func TestBlitStaticWritesOnlyInsideASubImage(t *testing.T) {
	f := blitTestFrame(t)

	parent := blitDest()
	before := cloneRGBA(parent)

	window := image.Rect(4, 3, 11, 9)
	sub, ok := parent.SubImage(window).(*image.RGBA)
	if !ok {
		t.Fatal("SubImage of an *image.RGBA is not an *image.RGBA")
	}

	// The frame straddles the window's left and top edges, so a clip that used
	// the parent's extent would write into the region asserted untouched below.
	const destX, destY = 3, 2
	terrain.BlitStatic(sub, f, destX, destY)

	checkBlit(t, "sub-image", before, sub, f, destX, destY)

	for y := 0; y < blitDstH; y++ {
		for x := 0; x < blitDstW; x++ {
			if (image.Point{X: x, Y: y}).In(window) {
				continue
			}
			if got := parent.RGBAAt(x, y); got != before.RGBAAt(x, y) {
				t.Fatalf("pixel (%d,%d) outside the sub-image changed to %v; a blit clips to the destination it was given, not to the memory behind it", x, y, got)
			}
		}
	}
}

// TestBlitStaticDrawsNothingItCannotWalk covers the refusals: each leaves the
// destination byte-identical and none panics.
//
// The pixel-less and short-Pixels frames are the ones that matter, because
// they are REACHABLE — the placement builder places any non-nil frame and
// reads no pixels, and one of its own tests builds exactly such a frame. A
// blit that trusted the header would index past the slice on the first row.
func TestBlitStaticDrawsNothingItCannotWalk(t *testing.T) {
	full := blitTestFrame(t)

	cases := []struct {
		name  string
		frame *terrain.StaticFrame
		why   string
	}{
		{name: "nil frame", frame: nil,
			why: "a class with no art places nothing, but a caller may still hand the frame over"},
		{name: "a frame of no area", frame: &terrain.StaticFrame{},
			why: "zero by zero: nothing to walk, and image.Rect would give an empty rectangle anyway"},
		{name: "a size with no pixels behind it", frame: &terrain.StaticFrame{Width: 3, Height: 2},
			why: "reachable from a placement, and the first indexed row would be out of range"},
		{name: "one pixel short of the size claimed", frame: &terrain.StaticFrame{
			Width: blitFrameW, Height: blitFrameH, Pixels: full.Pixels[:blitFrameW*blitFrameH-1], Palette: full.Palette},
			why: "the last row is short, so the frame draws nothing rather than all but one pixel of something"},
		{name: "negative width", frame: &terrain.StaticFrame{Width: -blitFrameW, Height: blitFrameH, Pixels: full.Pixels, Palette: full.Palette},
			why: "image.Rect would normalise the rectangle to the LEFT of destX and the row index would go negative"},
		{name: "negative height", frame: &terrain.StaticFrame{Width: blitFrameW, Height: -blitFrameH, Pixels: full.Pixels, Palette: full.Palette},
			why: "the same on the other axis"},
	}

	for _, c := range cases {
		dst := blitDest()
		before := cloneRGBA(dst)
		terrain.BlitStatic(dst, c.frame, 2, 3)
		if !bytes.Equal(before.Pix, dst.Pix) {
			t.Errorf("%s: the destination changed, want nothing drawn — %s", c.name, c.why)
		}
	}

	// A nil destination is the caller's mistake and still not a panic.
	terrain.BlitStatic(nil, full, 0, 0)
}

// TestBlitStaticReadsExactlyTheSizeItWasGiven pins the permissive half of the
// walkability test. The blit refuses a Pixels slice SHORTER than Width*Height,
// which is the condition that could index out of range, and draws a longer one to
// its header's size rather than refusing that too.
//
// Both readings are defensible — a StaticFrame documents Pixels as exactly
// Width*Height — so this states which one is the contract. Without it a guard
// tightened to != passes every other test in this file while silently drawing
// nothing at all for a frame it could have drawn.
func TestBlitStaticReadsExactlyTheSizeItWasGiven(t *testing.T) {
	f := blitTestFrame(t)
	long := &terrain.StaticFrame{
		Width: f.Width, Height: f.Height, Palette: f.Palette,
		Pixels: append(append([]terrain.StaticPixel(nil), f.Pixels...),
			terrain.StaticPixel{Index: blitHoleIndex, Opaque: true},
			terrain.StaticPixel{Index: blitHoleIndex, Opaque: true},
			terrain.StaticPixel{Index: blitHoleIndex, Opaque: true}),
	}

	dst := blitDest()
	before := cloneRGBA(dst)
	terrain.BlitStatic(dst, long, 5, 4)

	// The oracle reads the first Width*Height pixels alone, so a trailing pixel
	// that reached the canvas would land on one the frame does not cover.
	checkBlit(t, "a Pixels slice longer than the size claimed", before, dst, long, 5, 4)
	if bytes.Equal(before.Pix, dst.Pix) {
		t.Error("nothing was drawn: a frame carrying MORE pixels than its header claims is drawn to its header's size, not refused")
	}
}

// TestBlitStaticLeavesTheFrameAlone pins that the blit reads its source and
// writes only its destination. The window blits one frame at every cell that
// holds its byte, through the very same pointer, so a blit that wrote
// through it would corrupt every later placement of that class.
func TestBlitStaticLeavesTheFrameAlone(t *testing.T) {
	f := blitTestFrame(t)

	dst := blitDest()
	terrain.BlitStatic(dst, f, 5, 4)
	terrain.BlitStatic(dst, f, -1, -1)

	if pristine := blitTestFrame(t); !reflect.DeepEqual(f, pristine) {
		t.Error("the frame changed while it was being blitted; a blit reads its source and writes only its destination")
	}
}

// TestStaticFrameRGBAIsTheBlitOntoATransparentCanvas covers SC-4's second half.
//
// The assertion is against THIS PACKAGE'S OWN BLIT onto a fresh transparent
// canvas and deliberately not against a second expected image. The two paths
// exist so that the raster and the window cannot disagree per pixel, which
// nothing on either screen could reveal; a hand-built expectation here would
// be a third derivation of the palette walk and would agree with both while
// all three were wrong.
//
// What is stated INDEPENDENTLY of the blit is structural, not chromatic: the
// canvas is wholly transparent before the blit runs, every hole comes out at the
// canvas's own zero, and every opaque pixel comes out at full alpha.
func TestStaticFrameRGBAIsTheBlitOntoATransparentCanvas(t *testing.T) {
	f := blitTestFrame(t)

	want := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i, b := range want.Pix {
		if b != 0 {
			t.Fatalf("the comparison canvas is not transparent at byte %d (%d); the premise of this test is a fresh, wholly transparent image", i, b)
		}
	}
	terrain.BlitStatic(want, f, 0, 0)

	got := f.RGBA()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("RGBA() bounds = %v, want the frame's own extent %v", got.Bounds(), want.Bounds())
	}
	if got.Stride != want.Stride {
		t.Errorf("RGBA() stride = %d, want %d", got.Stride, want.Stride)
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Error("RGBA() differs from this package's own blit onto a transparent canvas; the two must be one implementation of the palette walk, because nothing on a screen could tell them apart (DD-5)")
	}

	holes, painted := 0, 0
	for i, px := range f.Pixels {
		x, y := i%f.Width, i/f.Width
		c := got.RGBAAt(x, y)
		if px.Opaque {
			painted++
			if c.A != 0xff {
				t.Errorf("the opaque pixel at (%d,%d) came out %v; an opaque pixel is painted at FULL opacity whatever alpha its palette entry carries", x, y, c)
			}
			continue
		}
		holes++
		if c != (color.RGBA{}) {
			t.Errorf("the hole at (%d,%d) came out %v, want the canvas's own zero — a see-through pixel must reach a texture as transparency, not as black at full alpha", x, y, c)
		}
	}
	if holes == 0 || painted == 0 {
		t.Fatalf("the fixture has %d holes and %d painted pixels; it must mix both for any of this to discriminate", holes, painted)
	}

	// No shared canvas: the window caches by frame identity, and a cached image
	// handed out twice would let one caller's mutation reach another's texture.
	second := f.RGBA()
	if second == got {
		t.Error("RGBA() returned the same image twice; every call allocates its own canvas")
	}
	if !bytes.Equal(second.Pix, got.Pix) {
		t.Error("two calls to RGBA() produced different pixels")
	}
	if pristine := blitTestFrame(t); !reflect.DeepEqual(f, pristine) {
		t.Error("RGBA() changed the frame it converted")
	}
}

// TestStaticFrameRGBAIsTotal covers the accessor on frames the loader never
// builds. The window asks a placement's frame for its texture, so this may not
// panic on anything a placement can carry.
func TestStaticFrameRGBAIsTotal(t *testing.T) {
	var nilFrame *terrain.StaticFrame
	if img := nilFrame.RGBA(); img == nil || !img.Bounds().Empty() {
		t.Errorf("a nil frame's RGBA() = %v, want a non-nil empty image", img)
	}

	for _, c := range []struct {
		name  string
		frame *terrain.StaticFrame
	}{
		{"a frame of no area", &terrain.StaticFrame{}},
		{"a frame of no width", &terrain.StaticFrame{Height: 4}},
		{"a frame of negative size", &terrain.StaticFrame{Width: -3, Height: -2}},
	} {
		img := c.frame.RGBA()
		if img == nil || !img.Bounds().Empty() {
			t.Errorf("%s: RGBA() = %v, want a non-nil empty image", c.name, img)
		}
	}

	// A frame the blit will not walk still gets an image of the size it claims,
	// wholly transparent: the canvas is the frame's own extent, and what the blit
	// declines to draw simply is not drawn.
	short := &terrain.StaticFrame{Width: 3, Height: 2, Palette: blitPalette()}
	img := short.RGBA()
	if want := image.Rect(0, 0, 3, 2); img.Bounds() != want {
		t.Errorf("a pixel-less frame's RGBA() bounds = %v, want %v", img.Bounds(), want)
	}
	for i, b := range img.Pix {
		if b != 0 {
			t.Fatalf("a pixel-less frame's RGBA() is not transparent at byte %d (%d)", i, b)
		}
	}
}
