package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// spriteLightPalette is the ONE palette both fixture frames carry. 3*i+1 is
// injective over the whole uint8 domain, so a frame resolving the wrong index
// paints a colour appearing nowhere in the correct output; entry
// spriteLightAlphaZero is left at alpha 0, which the ramp must shade rather than
// read as transparency.
func spriteLightPalette() [256]color.RGBA {
	var pal [256]color.RGBA
	for i := range pal {
		pal[i] = color.RGBA{R: uint8(3*i + 1), G: uint8(200 - i), B: uint8(7*i + 5), A: 0xff}
	}
	pal[spriteLightAlphaZero].A = 0
	return pal
}

const (
	spriteLightAlphaZero = 9
	spriteLightHole      = 77 // carried by transparent pixels only
)

// spriteLightIndices are the indices the fixture frames paint, index 0 among
// them: transparency is structural, so 0 is an ordinary colour here.
var spriteLightIndices = []uint8{0, 1, spriteLightAlphaZero, 200}

// spriteLightFrame builds a w x h frame over the shared palette. Every fourth
// pixel is a transparent hole, and the opaque ones cycle through
// spriteLightIndices, so two frames of DIFFERENT SIZES still carry the same
// index set — which is what makes "equal indices shade to equal colours" a
// statement about the ramp rather than about the frames being copies.
func spriteLightFrame(w, h int) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h), Palette: spriteLightPalette()}
	for i := range f.Pixels {
		if i%4 == 3 {
			f.Pixels[i] = terrain.StaticPixel{Index: spriteLightHole}
			continue
		}
		f.Pixels[i] = terrain.StaticPixel{Index: spriteLightIndices[i%len(spriteLightIndices)], Opaque: true}
	}
	return f
}

// spriteLightBundle is staticsBundle's geometry — the same canvases and centres,
// so statics_test.go's hand-worked placement table still applies — over frames
// that carry a real palette.
func spriteLightBundle() *terrain.StaticSet {
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: spriteLightFrame(10, 6)}
	set.Classes[2] = &terrain.StaticClass{Width: 31, Height: 31, CenterX: 15, CenterY: 29, Frame: spriteLightFrame(4, 4)}
	return set
}

// spriteLightUnitArt is the unit class the entity pass draws, at a size no
// object frame in the bundle has and over the identical palette.
func spriteLightUnitArt() *terrain.UnitClass {
	return &terrain.UnitClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60,
		Frames: []*terrain.StaticFrame{spriteLightFrame(7, 11)}}
}

// spriteLightViewer is a drawable viewer holding both layers: the object bundle
// above over the cliff grid's overlay, and one entity carrying the unit art, on
// a cell the cull keeps.
//
// The two frames are returned as THE VIEWER'S OWN POINTERS. A second call to the
// builders would produce equal frames at different addresses, and the cache is
// keyed on identity — an assertion against a rebuilt frame would look up nothing
// and pass or fail for the wrong reason.
func spriteLightViewer(t *testing.T) (v *Viewer, object, unit *terrain.StaticFrame) {
	t.Helper()
	bundle := spriteLightBundle()
	v = newStaticsViewer(t, bundle, true, false)
	layoutViewport(v, cliffW*terrain.CellSize, cliffCanvasH)
	art := spriteLightUnitArt()
	v.SetEntities([]MapEntity{{Cell: image.Pt(0, 1), Art: art, Frame: art.Frames[0]}})
	return v, bundle.Classes[1].Frame, art.Frames[0]
}

const spriteLightWantRow = 3

func TestWindowSpriteRowIsTheFixedSunsAndNothingElse(t *testing.T) {
	v, _, _ := spriteLightViewer(t)
	if got := v.spriteRow(); got != spriteLightWantRow {
		t.Fatalf("spriteRow() = %d, want %d — the fixed daytime sun's ambient 0x0e", got, spriteLightWantRow)
	}

	v.SetFlat(true)
	if got := v.spriteRow(); got != spriteLightWantRow {
		t.Errorf("the flat diagnostic moved the row to %d", got)
	}
	v.SetFlat(false)

	bare := newViewer(t, grid(cliffW, cliffH))
	if bare.Lit() || bare.levels != nil {
		t.Fatalf("the fixture is not the unlit-terrain case: Lit()=%v levels=%v", bare.Lit(), bare.levels != nil)
	}
	if got := bare.spriteRow(); got != spriteLightWantRow {
		t.Errorf("a map with no usable altitudes gives sprite row %d, want %d — "+
			"a sprite's row has no relief term", got, spriteLightWantRow)
	}
	f := spriteLightFrame(5, 5)
	if equalBytes(bare.spritePixels(f).Pix, f.RGBA().Pix) {
		t.Error("a map with no usable altitudes drew its sprites raw")
	}
}

// TestBothWindowPassesTakeOneRowAndOneRule — AC-7, SC-5.
//
// Two frames of DIFFERENT SIZES over one palette, one placed as a static
// object and one as a unit sprite, drawn in one rendered frame.
func TestBothWindowPassesTakeOneRowAndOneRule(t *testing.T) {
	v, object, unit := spriteLightViewer(t)
	if object.Width == unit.Width && object.Height == unit.Height {
		t.Fatal("the fixture does not discriminate: the two frames are the same size")
	}
	if object.Palette != unit.Palette {
		t.Fatal("the fixture does not discriminate: the two frames carry different palettes")
	}

	// Both passes exist in this one rendered frame, and both submit sprites.
	var rec staticRecorder
	v.drawPlane(&rec)
	if len(rec.imgs) < 2 {
		t.Fatalf("the content band submitted %d frames, want the object sprite and the entity one", len(rec.imgs))
	}

	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)
	v.Draw(screen)

	// Every cached texture is at the one row, whichever pass built it.
	if len(v.staticImages) == 0 {
		t.Fatal("the draw built no textures")
	}
	sawUnit := false
	for key := range v.staticImages {
		if key.row != spriteLightWantRow {
			t.Errorf("a texture was cached at row %d, want %d — every sprite of one frame takes one row",
				key.row, spriteLightWantRow)
		}
		if key.frame == unit {
			sawUnit = true
		}
	}
	if !sawUnit {
		t.Error("the unit frame has no texture; the entity pass did not reach the builder")
	}

	objectPix := v.spritePixels(object)
	unitPix := v.spritePixels(unit)
	seen := map[uint8]color.RGBA{}
	for _, f := range []*terrain.StaticFrame{object, unit} {
		pix := objectPix
		if f == unit {
			pix = unitPix
		}
		for i, p := range f.Pixels {
			if !p.Opaque {
				continue
			}
			got := pix.RGBAAt(i%f.Width, i/f.Width)
			want := terrain.SpriteRGBA(f.Palette[p.Index], terrain.DefaultDaytime.SkyTint, spriteLightWantRow)
			if got != want {
				t.Fatalf("%dx%d frame, index %d shaded to %v, want %v", f.Width, f.Height, p.Index, got, want)
			}
			if prev, ok := seen[p.Index]; ok && prev != got {
				t.Fatalf("index %d shades to %v in one frame and %v in the other", p.Index, prev, got)
			}
			seen[p.Index] = got
		}
	}
	if len(seen) < 2 {
		t.Fatalf("only %d distinct indices were painted; the fixture does not discriminate", len(seen))
	}
}

// TestWindowUnshadedToggleReturnsTheLitPixelsAndTheLitTexture — AC-9,
// SC-7.
//
// Lit, raw, and lit again. The third state's pixels are compared with the
// FIRST's byte for byte, which is the assertion a stale texture fails and an
// equality-free "they differ" would not; and the same is asked of the cache,
// where the row in the key is what lets the toggle back hand the original
// texture over instead of uploading a third.
func TestWindowUnshadedToggleReturnsTheLitPixelsAndTheLitTexture(t *testing.T) {
	v, f, _ := spriteLightViewer(t)

	lit := append([]uint8(nil), v.spritePixels(f).Pix...)
	raw := f.RGBA().Pix

	v.SetUnshaded(true)
	if got := v.spriteRow(); got != spriteUnlit {
		t.Errorf("the unshaded switch gives row %d, want the unlit slot %d", got, spriteUnlit)
	}
	unshaded := append([]uint8(nil), v.spritePixels(f).Pix...)

	v.SetUnshaded(false)
	again := v.spritePixels(f).Pix

	if !equalBytes(unshaded, raw) {
		t.Error("the unshaded state's pixels are not the raw palette's, byte for byte")
	}
	if equalBytes(lit, raw) {
		t.Fatalf("the fixture does not discriminate: the lit pixels equal the raw ones at row %d", spriteLightWantRow)
	}
	if !equalBytes(again, lit) {
		t.Error("turning the diagnostic off again did not restore the lit pixels")
	}

	// The cache: three draws, two distinct rows, and the third draw's texture is
	// the first draw's own object rather than a fresh upload.
	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)
	v.Draw(screen)
	litTexture := v.staticImages[v.spriteKey(f)]
	if litTexture == nil {
		t.Fatal("the lit draw cached no texture for the fixture frame")
	}
	litCount := len(v.staticImages)

	v.SetUnshaded(true)
	v.Draw(screen)
	if len(v.staticImages) <= litCount {
		t.Errorf("the unshaded draw reused the lit textures: cache went %d -> %d", litCount, len(v.staticImages))
	}
	if v.staticImages[v.spriteKey(f)] == litTexture {
		t.Error("the unshaded draw handed back the LIT texture; the row is not in the key")
	}
	unshadedCount := len(v.staticImages)

	v.SetUnshaded(false)
	v.Draw(screen)
	if len(v.staticImages) != unshadedCount {
		t.Errorf("toggling back uploaded again: cache went %d -> %d, want no growth",
			unshadedCount, len(v.staticImages))
	}
	if v.staticImages[v.spriteKey(f)] != litTexture {
		t.Error("toggling back did not hand the original lit texture over")
	}
}

// equalBytes is bytes.Equal without the import, so this file's dependency list
// stays the render tier and the engine.
func equalBytes(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
