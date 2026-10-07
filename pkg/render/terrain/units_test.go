package terrain_test

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// unitPlaceClass is AC-5's class, now carrying TWO frames of unequal size so
// which frame is DRAWN is observable: an anchor taken over either size alone,
// or over the wrong frame's, lands elsewhere.
//
//	frame 0, 20x24: anchorX = 30 - 64/2 + 20/2 =  8
//	                anchorY = 80 - 96/2 + 24/2 = 44
//	frame 1, 10x8:  anchorX = 30 - 64/2 + 10/2 =  3
//	                anchorY = 80 - 96/2 +  8/2 = 36
func unitPlaceClass() *terrain.UnitClass {
	return &terrain.UnitClass{
		Width: 64, Height: 96, CenterX: 30, CenterY: 80,
		Frames: []*terrain.StaticFrame{
			{Width: 20, Height: 24, Pixels: make([]terrain.StaticPixel, 20*24)},
			{Width: 10, Height: 8, Pixels: make([]terrain.StaticPixel, 10*8)},
		},
	}
}

func TestUnitPlaceAnchorRuleAtTheDrawnFrameSize(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal below is stated over 32-pixel cells", terrain.CellSize)
	}

	c := unitPlaceClass()
	// Cell (3,5) flat, frame 0: destX = 3*32+16-8 = 104, destY = 5*32+16-44 = 132.
	p, ok := terrain.UnitPlace(3, 5, c, c.Frames[0], false, 0, 0)
	if !ok {
		t.Fatal("UnitPlace answered false for a drawable class and frame")
	}
	if want := (image.Point{X: 3, Y: 5}); p.Cell != want {
		t.Errorf("Cell = %v, want %v", p.Cell, want)
	}
	if want := (image.Point{X: 104, Y: 132}); p.TopLeft != want {
		t.Errorf("TopLeft = %v, want %v — the anchor rule over a 64x96 canvas, centre (30,80), 20x24 frame", p.TopLeft, want)
	}
	if want := (image.Point{X: 8, Y: 44}); p.Anchor != want {
		t.Errorf("Anchor = %v, want %v", p.Anchor, want)
	}
	if p.Frame != c.Frames[0] {
		t.Error("Frame is not the handed frame's own pointer; the texture cache downstream keys on frame identity")
	}
	if p.Mirror {
		t.Error("Mirror = true on a placement asked for plain")
	}
	// The accessors ride along: the exact rect is the FRAME's 20x24, and the
	// flat ground point is the cell's centre (3*32+16, 5*32+16).
	if want := image.Rect(104, 132, 124, 156); p.Rect() != want {
		t.Errorf("Rect() = %v, want %v — the drawn frame's own size, never the canvas", p.Rect(), want)
	}
	if want := (image.Point{X: 112, Y: 176}); p.Ground() != want {
		t.Errorf("Ground() = %v, want the cell centre %v in flat geometry", p.Ground(), want)
	}

	// A LATER frame of the same sheet places at ITS own size: same cell, same
	// class, frame 1's 10x8 — destX = 3*32+16-3 = 109, destY = 5*32+16-36 = 140.
	p1, ok := terrain.UnitPlace(3, 5, c, c.Frames[1], false, 0, 0)
	if !ok {
		t.Fatal("UnitPlace answered false for the sheet's second frame")
	}
	if want := (image.Point{X: 109, Y: 140}); p1.TopLeft != want {
		t.Errorf("frame 1 TopLeft = %v, want %v — the DRAWN frame's size feeds the anchor, not frame 0's", p1.TopLeft, want)
	}
	if want := (image.Point{X: 3, Y: 36}); p1.Anchor != want {
		t.Errorf("frame 1 Anchor = %v, want %v", p1.Anchor, want)
	}
	if p1.Frame != c.Frames[1] {
		t.Error("frame 1's placement does not carry frame 1's own pointer")
	}
	// Both frames' ground points are the same cell centre: the anchor terms
	// cancel out of Ground(), whatever frame is drawn.
	if want := (image.Point{X: 112, Y: 176}); p1.Ground() != want {
		t.Errorf("frame 1 Ground() = %v, want the cell centre %v", p1.Ground(), want)
	}

	// A frame LARGER than its canvas, which an anchor clamped to the canvas
	// could not place: anchorX = 16-32/2+96/2 = 48, anchorY = 30-32/2+128/2 = 78;
	// cell (2,1): destX = 2*32+16-48 = 32, destY = 1*32+16-78 = -30.
	bigClass := &terrain.UnitClass{
		Width: 32, Height: 32, CenterX: 16, CenterY: 30,
		Frames: []*terrain.StaticFrame{
			{Width: 96, Height: 128, Pixels: make([]terrain.StaticPixel, 96*128)},
		},
	}
	big, ok := terrain.UnitPlace(2, 1, bigClass, bigClass.Frames[0], false, 0, 0)
	if !ok {
		t.Fatal("UnitPlace answered false for the larger-than-canvas class")
	}
	if want := (image.Point{X: 32, Y: -30}); big.TopLeft != want {
		t.Errorf("larger-than-canvas TopLeft = %v, want %v", big.TopLeft, want)
	}
	if want := (image.Point{X: 48, Y: 78}); big.Anchor != want {
		t.Errorf("larger-than-canvas Anchor = %v, want %v", big.Anchor, want)
	}
}

func TestUnitPlaceCarriesTheMirrorBitAndReadsItNowhere(t *testing.T) {
	c := unitPlaceClass()
	for _, d := range []struct{ lift, originY int }{{0, 0}, {7, 19}} {
		plain, okP := terrain.UnitPlace(3, 5, c, c.Frames[0], false, d.lift, d.originY)
		mirrored, okM := terrain.UnitPlace(3, 5, c, c.Frames[0], true, d.lift, d.originY)
		if !okP || !okM {
			t.Fatalf("lift=%d originY=%d: UnitPlace answered (plain %t, mirrored %t), want both true", d.lift, d.originY, okP, okM)
		}
		if !mirrored.Mirror || plain.Mirror {
			t.Errorf("lift=%d originY=%d: Mirror bits = (plain %t, mirrored %t), want (false, true)",
				d.lift, d.originY, plain.Mirror, mirrored.Mirror)
		}
		mirrored.Mirror = false
		if mirrored != plain {
			t.Errorf("lift=%d originY=%d: mirrored placement %+v differs from plain %+v beyond the bit — "+
				"placement arithmetic must not read it", d.lift, d.originY, mirrored, plain)
		}
		mirrored.Mirror = true
		if mirrored.Rect() != plain.Rect() {
			t.Errorf("lift=%d originY=%d: Rect() differs under mirror: %v vs %v", d.lift, d.originY, mirrored.Rect(), plain.Rect())
		}
		if mirrored.Ground() != plain.Ground() {
			t.Errorf("lift=%d originY=%d: Ground() differs under mirror: %v vs %v", d.lift, d.originY, mirrored.Ground(), plain.Ground())
		}
	}
}

// TestUnitPlaceDisplacesVerticallyOnly covers SC-4's displaced half (0022
// AC-5; 0024 AC-6's lift clause): against the same cell's flat placement, a
// lift and an originY move the top-left by exactly -(lift + originY) in Y, by
// zero in X, and leave the frame's own anchor pixel alone.
//
// The (1, -1) pair sums to zero, so an implementation applying only one term —
// or the two with different signs — lands one pixel off a case a same-sign
// pair would let through.
func TestUnitPlaceDisplacesVerticallyOnly(t *testing.T) {
	c := unitPlaceClass()
	flat, ok := terrain.UnitPlace(3, 5, c, c.Frames[0], false, 0, 0)
	if !ok {
		t.Fatal("UnitPlace answered false for a drawable class")
	}

	for _, d := range []struct{ lift, originY int }{
		{7, 0}, {0, 19}, {9, -4}, {-13, 29}, {1, -1},
	} {
		p, ok := terrain.UnitPlace(3, 5, c, c.Frames[0], false, d.lift, d.originY)
		if !ok {
			t.Fatalf("lift=%d originY=%d: UnitPlace answered false", d.lift, d.originY)
		}
		if p.TopLeft.X != flat.TopLeft.X {
			t.Errorf("lift=%d originY=%d: TopLeft.X = %d, want the flat %d — the displacement is vertical only",
				d.lift, d.originY, p.TopLeft.X, flat.TopLeft.X)
		}
		if want := flat.TopLeft.Y - (d.lift + d.originY); p.TopLeft.Y != want {
			t.Errorf("lift=%d originY=%d: TopLeft.Y = %d, want %d (the flat %d shifted by exactly -(lift+originY))",
				d.lift, d.originY, p.TopLeft.Y, want, flat.TopLeft.Y)
		}
		if p.Anchor != flat.Anchor {
			t.Errorf("lift=%d originY=%d: Anchor = %v, want the flat %v — the anchor pixel is geometry-independent",
				d.lift, d.originY, p.Anchor, flat.Anchor)
		}
	}
}

// TestUnitPlaceNilClassOrNilFramePlacesNothing covers SC-4's negative half:
// a nil class and a nil frame both place nothing, at either mirror value,
// with the zero placement beside the false — never a frameless placement a
// caller might cull or draw.
func TestUnitPlaceNilClassOrNilFramePlacesNothing(t *testing.T) {
	c := unitPlaceClass()
	for _, mirror := range []bool{false, true} {
		if p, ok := terrain.UnitPlace(3, 5, nil, c.Frames[0], mirror, 7, 19); ok || p != (terrain.StaticPlacement{}) {
			t.Errorf("UnitPlace(nil class, mirror %t) = %+v, %t; want the zero placement and false", mirror, p, ok)
		}
		if p, ok := terrain.UnitPlace(3, 5, c, nil, mirror, 7, 19); ok || p != (terrain.StaticPlacement{}) {
			t.Errorf("UnitPlace(nil frame, mirror %t) = %+v, %t; want the zero placement and false", mirror, p, ok)
		}
	}
	frameless := &terrain.UnitClass{Width: 64, Height: 96, CenterX: 30, CenterY: 80}
	if p, ok := terrain.UnitPlace(3, 5, frameless, nil, false, 7, 19); ok || p != (terrain.StaticPlacement{}) {
		t.Errorf("UnitPlace(frameless class, nil frame) = %+v, %t; want the zero placement and false", p, ok)
	}
}

func TestUnitSetSparseSignedKeys(t *testing.T) {
	var zero terrain.UnitSet
	if c, ok := zero.Classes[1]; ok || c != nil {
		t.Errorf("a zero UnitSet answered (%v, %t) at id 1; it must be a legal empty bundle", c, ok)
	}

	set := terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		-3: unitPlaceClass(),
		7:  {Width: 8, Height: 8, CenterX: 4, CenterY: 6}, // resolved, artless
	}}

	// A NEGATIVE id is an ordinary key: the record's field is sign-extended,
	// so the map must answer there, not fold it into the positive range.
	c, ok := set.Classes[-3]
	if !ok || c == nil || len(c.Frames) == 0 {
		t.Fatalf("Classes[-3] = (%v, %t), want the drawable class", c, ok)
	}
	if _, ok := terrain.UnitPlace(0, 0, c, c.Frames[0], false, 0, 0); !ok {
		t.Error("the drawable entry at id -3 did not place")
	}
	if f, ok := set.Classes[7]; !ok || f == nil || f.Frames != nil {
		t.Fatalf("Classes[7] = (%v, %t), want a present, frameless entry", f, ok)
	}
	if _, ok := terrain.UnitPlace(0, 0, set.Classes[7], nil, false, 0, 0); ok {
		t.Error("the frameless entry at id 7 placed; an excluded class draws the square")
	}
	// The misses, either side of the entries: no entry at all, at any sign.
	for _, id := range []int32{-4, 0, 5, 80} {
		if c, ok := set.Classes[id]; ok || c != nil {
			t.Errorf("Classes[%d] = (%v, %t), want a clean miss — an id naming no class is not a frameless entry", id, c, ok)
		}
	}
}
