package ui

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// sackGridW/sackGridH is the bare grid every placement and exclusion test in
// this file shares: 4x4, no altitude layer, so Mode() is flat and lift/originY
// are both zero — the tests below are about WHICH sacks place and where, not
// about the displaced geometry (that is TestSackLayerLiftsByItsOwnCell's own,
// narrower, job).
const sackGridW, sackGridH = 4, 4

// sackFrame builds one synthetic drawable frame, mirroring static_draw_test.go's
// drawFrame: only its SIZE and its pointer identity reach anything under test
// here.
func sackFrame(w, h int) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: uint8(i % 251), Opaque: true}
	}
	return f
}

// sackIdentityViewer builds a bare viewer over the 4x4 flat grid — no bundle,
// no structures, so showStaticArt and showStructureArt are both false and
// nothing but a sack could ever appear in its band — whose camera is the
// identity transform, so a placement's TopLeft world coordinate IS its screen
// one (planeViewer's own precedent, structures_test.go, rebuilt here so this
// file owns its fixture end to end rather than reaching into another one).
func sackIdentityViewer(t *testing.T) *Viewer {
	t.Helper()
	v := newViewer(t, grid(sackGridW, sackGridH))
	layoutViewport(v, sackGridW*terrain.CellSize, sackGridH*terrain.CellSize)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the identity (0,0) at 1", cam.X, cam.Y, cam.Zoom)
	}
	if v.Mode() != ModeFlat {
		t.Fatalf("Mode() = %v, want flat; the fixture carries no altitude layer", v.Mode())
	}
	return v
}

func TestSetSackFramesAndSetSacksAdoptAndReplace(t *testing.T) {
	v := sackIdentityViewer(t)

	frames := []*terrain.StaticFrame{sackFrame(4, 4)}
	v.SetSackFrames(frames)
	if reflect.ValueOf(v.sackFrames).Pointer() != reflect.ValueOf(frames).Pointer() {
		t.Errorf("SetSackFrames copied the slice rather than adopting it")
	}

	first := []MapSack{{Cell: image.Pt(0, 0), FrameIndex: 0}}
	v.SetSacks(first)
	if reflect.ValueOf(v.sacks).Pointer() != reflect.ValueOf(first).Pointer() {
		t.Errorf("SetSacks copied the slice rather than adopting it")
	}

	second := []MapSack{{Cell: image.Pt(1, 1), FrameIndex: 0}, {Cell: image.Pt(2, 2), FrameIndex: 0}}
	v.SetSacks(second)
	if len(v.sacks) != 2 || v.sacks[0].Cell != (image.Point{X: 1, Y: 1}) || v.sacks[1].Cell != (image.Point{X: 2, Y: 2}) {
		t.Fatalf("a second SetSacks call did not REPLACE the first: %+v", v.sacks)
	}

	v.SetSacks(nil)
	if got := v.sackLayer(); got != nil {
		t.Errorf("sackLayer() = %v after SetSacks(nil), want nil", got)
	}
}

// TestSackLayerIsNilWhenNeverPushed pins the "unchanged from T1" half of this
// task's done-when: a viewer SetSacks is never called on holds a nil sack
// stream, exactly the nil DepthOrder's fifth argument already treated as an
// empty one before this task supplied a real builder, and drawPlane submits
// nothing over a fixture that carries no other content either.
func TestSackLayerIsNilWhenNeverPushed(t *testing.T) {
	v := sackIdentityViewer(t)
	if got := v.sackLayer(); got != nil {
		t.Errorf("sackLayer() = %v on a viewer never given SetSacks, want nil", got)
	}
	var rec staticRecorder
	v.drawPlane(&rec)
	if len(rec.imgs) != 0 {
		t.Errorf("drawPlane submitted %d draws with no sack ever pushed and no other content, want 0", len(rec.imgs))
	}
}

func TestSackLayerPlacesEachSackAtItsOwnCell(t *testing.T) {
	v := sackIdentityViewer(t)
	frame := sackFrame(8, 8)
	v.SetSackFrames([]*terrain.StaticFrame{frame})
	cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 3}}
	v.SetSacks([]MapSack{{Cell: cells[0], FrameIndex: 0}, {Cell: cells[1], FrameIndex: 0}})

	got := v.sackLayer()
	if len(got) != len(cells) {
		t.Fatalf("sackLayer() returned %d placements, want %d", len(got), len(cells))
	}
	for i, cell := range cells {
		if got[i].Cell != cell {
			t.Errorf("placement %d: cell %v, want %v (list order)", i, got[i].Cell, cell)
		}
		if got[i].Frame != frame {
			t.Errorf("placement %d: frame pointer is not the sheet's own", i)
		}
	}
}

func TestSackLayerLiftsByItsOwnCell(t *testing.T) {
	v := newViewer(t, cliffGrid())
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced; the fixture's altitude layer is valid", v.Mode())
	}
	frame := sackFrame(4, 4)
	v.SetSackFrames([]*terrain.StaticFrame{frame})
	const col, row = 0, 1
	v.SetSacks([]MapSack{{Cell: image.Pt(col, row), FrameIndex: 0}})

	proj := cliffProjection()
	lift := proj.AnchorHeight(col, row)
	if lift == 0 {
		t.Fatal("fixture does not discriminate: AnchorHeight(0,1) is zero")
	}
	flatPlace, ok := terrain.SackPlace(col, row, frame, 0, 0)
	if !ok {
		t.Fatal("SackPlace refused a valid frame in the flat geometry")
	}
	dispPlace, ok := terrain.SackPlace(col, row, frame, lift, proj.MinV)
	if !ok {
		t.Fatal("SackPlace refused a valid frame in the displaced geometry")
	}

	got := v.sackLayer()
	if len(got) != 1 {
		t.Fatalf("sackLayer() returned %d placements, want 1", len(got))
	}
	if got[0] != dispPlace {
		t.Errorf("displaced placement = %+v, want %+v (AnchorHeight(%d,%d)=%d, MinV=%d)",
			got[0], dispPlace, col, row, lift, proj.MinV)
	}
	if got[0].TopLeft.X != flatPlace.TopLeft.X {
		t.Errorf("X moved between geometries: displaced %d, flat %d — P-2 requires zero X difference",
			got[0].TopLeft.X, flatPlace.TopLeft.X)
	}
	if d := flatPlace.TopLeft.Y - got[0].TopLeft.Y; d != lift+proj.MinV {
		t.Errorf("Y difference = %d, want lift+originY = %d", d, lift+proj.MinV)
	}
}

func TestSackLayerExcludesOutOfRangeIndexAndUngriddedCell(t *testing.T) {
	v := sackIdentityViewer(t)
	frame := sackFrame(4, 4)
	v.SetSackFrames([]*terrain.StaticFrame{frame})

	valid := image.Pt(1, 1)
	pushed := []MapSack{
		{Cell: valid, FrameIndex: 0},                  // the one survivor
		{Cell: valid, FrameIndex: 5},                  // frame index past the sheet
		{Cell: valid, FrameIndex: -1},                 // negative frame index
		{Cell: image.Pt(-1, 0), FrameIndex: 0},        // ungridded: X < 0
		{Cell: image.Pt(sackGridW, 0), FrameIndex: 0}, // ungridded: X == Width
		{Cell: image.Pt(0, -1), FrameIndex: 0},        // ungridded: Y < 0
		{Cell: image.Pt(0, sackGridH), FrameIndex: 0}, // ungridded: Y == Height
	}
	v.SetSacks(pushed)

	got := v.sackLayer()
	if len(got) != 1 {
		t.Fatalf("sackLayer() returned %d placements, want 1 (only the valid sack); got %+v", len(got), got)
	}
	if got[0].Cell != valid {
		t.Errorf("the surviving placement is cell %v, want %v", got[0].Cell, valid)
	}

	if len(v.sacks) != len(pushed) {
		t.Errorf("SetSacks's own list holds %d entries, want the pushed %d — DD-7 excludes only at the "+
			"build, never at the push, and none of it may be removed from what the world gave", len(v.sacks), len(pushed))
	}

	// The exclusion must not panic and must not disturb the rest of the band
	// either — drawPlane over this same viewer paints exactly the one survivor.
	var rec staticRecorder
	v.drawPlane(&rec)
	if len(rec.imgs) != 1 {
		t.Errorf("drawPlane submitted %d draws, want 1 (the excluded sacks draw nothing, silently)", len(rec.imgs))
	}
}

func TestSackLayerWithNoFramesDrawsNothing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		frames []*terrain.StaticFrame
	}{
		{"SetSackFrames never called", nil},
		{"SetSackFrames called with an empty slice", []*terrain.StaticFrame{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := sackIdentityViewer(t)
			if tc.frames != nil {
				v.SetSackFrames(tc.frames)
			}
			v.SetSacks([]MapSack{{Cell: image.Pt(0, 0), FrameIndex: 0}})

			// A nil sheet and an empty one both fail the range test inside the
			// loop, so the result is an EMPTY slice here rather than the nil
			// early-return a never-pushed v.sacks gives (TestSackLayerIsNilWhenNeverPushed) —
			// both answer "no frame resolves" and neither is an error.
			if got := v.sackLayer(); len(got) != 0 {
				t.Errorf("sackLayer() = %v, want none", got)
			}
			var rec staticRecorder
			v.drawPlane(&rec)
			if len(rec.imgs) != 0 {
				t.Errorf("drawPlane submitted %d draws with no sheet, want 0", len(rec.imgs))
			}
		})
	}
}

func TestSacksAreNotGatedByEitherArtSwitch(t *testing.T) {
	v := sackIdentityViewer(t)
	if v.showStaticArt || v.showStructureArt {
		t.Fatalf("fixture does not discriminate: an art switch is already on")
	}
	frame := sackFrame(4, 4)
	v.SetSackFrames([]*terrain.StaticFrame{frame})
	v.SetSacks([]MapSack{{Cell: image.Pt(1, 1), FrameIndex: 0}})

	var rec staticRecorder
	v.drawPlane(&rec)
	if len(rec.imgs) != 1 {
		t.Fatalf("got %d draw calls with both art switches off and no bundle, want 1 — "+
			"FR-3: no art switch gates a sack", len(rec.imgs))
	}
}

// sackLightBundle is one object class whose canvas and centre put its sprite
// on its own cell, over the given frame — spritelight_test.go's own bundle
// shape, rebuilt here so this file does not reach into that one's fixture.
func sackLightBundle(f *terrain.StaticFrame) *terrain.StaticSet {
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{
		Width: terrain.CellSize, Height: terrain.CellSize,
		CenterX: terrain.CellSize / 2, CenterY: terrain.CellSize / 2, Frame: f,
	}
	return set
}

// sackLightViewer builds a viewer holding BOTH a static object and a sack,
// over two frames of DIFFERENT SIZES that share spritelight_test.go's own
// palette (spriteLightFrame) — so "equal indices shade to equal colours"
// is a statement about the ramp, not about the two frames being copies, on
// TestBothWindowPassesTakeOneRowAndOneRule's own precedent applied to a sack
// instead of a unit.
//
// THE GRID CARRIES AN ALL-ZERO ALTITUDE LAYER, deliberately, and not none at
// all: lightTint() returns the zero tint whenever Lit() is false, and Lit()
// needs v.levels != nil, which only a VALID altitude layer builds — an
// altitude-less grid would make every tint below zero regardless of v.sun and
// the fixture would not discriminate. All-zero is still valid (mode_test.go's
// own "an all-zero valid layer" case) and keeps every lift and origin zero, so
// it adds no geometry this test is not already about.
func sackLightViewer(t *testing.T) (v *Viewer, sack, sibling *terrain.StaticFrame) {
	t.Helper()
	sack = spriteLightFrame(5, 5)
	sibling = spriteLightFrame(9, 9)

	overlay := make([]uint8, sackGridW*sackGridH)
	overlay[1*sackGridW+1] = 1 // one object at cell (1,1)
	g := grid(sackGridW, sackGridH)
	g.Altitudes = make([]uint8, sackGridW*sackGridH)
	g.Overlay = overlay

	var err error
	v, err = NewViewerWithStatics("m", g, &terrain.Tileset{}, sackLightBundle(sibling), true, false, terrain.AnimGateTiles, nil, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	layoutViewport(v, sackGridW*terrain.CellSize, sackGridH*terrain.CellSize)
	v.SetSackFrames([]*terrain.StaticFrame{sack})
	v.SetSacks([]MapSack{{Cell: image.Pt(2, 2), FrameIndex: 0}})
	return v, sack, sibling
}

func TestSackAndSiblingSpriteShareLightRowAndTint(t *testing.T) {
	for _, tc := range []struct {
		name string
		sun  terrain.Light
	}{
		{"daytime", terrain.DefaultDaytime},
		{"a differently-lit hour", terrain.Light{Ambient: 0x28, Range: 0x20, SkyTint: [3]uint8{9, 0, 5}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, sack, sibling := sackLightViewer(t)
			v.sun = tc.sun

			var rec staticRecorder
			v.drawPlane(&rec)
			if len(rec.imgs) != 2 {
				t.Fatalf("drawPlane submitted %d textures, want 2 (the object and the sack)", len(rec.imgs))
			}

			sackKey, sibKey := v.spriteKey(sack), v.spriteKey(sibling)
			if sackKey.row != sibKey.row || sackKey.tint != sibKey.tint {
				t.Fatalf("sack key %+v, sibling key %+v — the two must share row and tint (AC-8)", sackKey, sibKey)
			}

			sackPix, sibPix := v.spritePixels(sack), v.spritePixels(sibling)
			seen := map[uint8]color.RGBA{}
			for _, f := range []*terrain.StaticFrame{sack, sibling} {
				pix := sackPix
				if f == sibling {
					pix = sibPix
				}
				for i, p := range f.Pixels {
					if !p.Opaque {
						continue
					}
					got := pix.RGBAAt(i%f.Width, i/f.Width)
					want := terrain.SpriteRGBA(f.Palette[p.Index], tc.sun.SkyTint, sackKey.row)
					if got != want {
						t.Fatalf("%dx%d frame index %d shaded to %v, want %v (AC-9: the decoded frame's own, "+
							"changed by nothing but the light row and tint)", f.Width, f.Height, p.Index, got, want)
					}
					if prev, ok := seen[p.Index]; ok && prev != got {
						t.Fatalf("index %d shades to %v in one frame and %v in the other", p.Index, prev, got)
					}
					seen[p.Index] = got
				}
			}
			if len(seen) < 2 {
				t.Fatalf("only %d distinct indices painted; fixture does not discriminate", len(seen))
			}
		})
	}
}

func TestSacksAddNoShadowEntry(t *testing.T) {
	v, sack, _ := sackLightViewer(t)
	withSack := v.shadowDraws()

	v.SetSacks(nil)
	withoutSack := v.shadowDraws()

	if len(withSack) != len(withoutSack) {
		t.Fatalf("shadowDraws() length changed when a sack was added: %d -> %d — DD-8 adds no shadow entry",
			len(withoutSack), len(withSack))
	}
	for i := range withSack {
		if withSack[i] != withoutSack[i] {
			t.Fatalf("shadowDraws()[%d] changed when a sack was added: %+v -> %+v", i, withoutSack[i], withSack[i])
		}
	}
	for _, d := range withSack {
		if d.Frame == sack {
			t.Fatalf("the sack's own frame appeared among the shadow casters")
		}
	}
}
