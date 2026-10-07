package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// groundTintPalette carries one entry per RGB channel near the top of the
// byte range — so a tint applied to it exercises AC-9's saturating clamp —
// plus one entry well inside the range for the ordinary, non-saturating add.
// Every entry is distinct, so a pixel painted the wrong index shows up as the
// wrong colour.
func groundTintPalette() color.Palette {
	return color.Palette{
		color.RGBA{R: 250, G: 10, B: 5, A: 0xff},    // R saturates under this file's tint
		color.RGBA{R: 5, G: 245, B: 10, A: 0xff},    // G saturates
		color.RGBA{R: 10, G: 5, B: 250, A: 0xff},    // B saturates
		color.RGBA{R: 100, G: 150, B: 200, A: 0xff}, // no channel saturates
	}
}

// groundTintCellPixels is a CellSize x CellSize index plane cycling through
// groundTintPalette's four entries, so every entry is painted at least once.
func groundTintCellPixels() *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, terrain.CellSize, terrain.CellSize), groundTintPalette())
	for i := range img.Pix {
		img.Pix[i] = uint8(i % len(img.Palette))
	}
	return img
}

// groundTintTileset is a Tileset with slot 0 populated by one sub-cell built
// over groundTintPalette; every other slot is absent. Word 0 resolves to slot
// 0 (light_test.go's slotWord documents the same mapping), so a grid built
// with grid()/altGrid() and no words draws this cell at every tile.
func groundTintTileset() *terrain.Tileset {
	var ts terrain.Tileset
	ts.Slots[0] = &terrain.Strip{SubCells: []*image.Paletted{groundTintCellPixels()}}
	return &ts
}

// groundTint is the tint this file exercises AC-9's clamp with: three values
// each chosen so its own channel's saturating palette entry crosses 255 and
// its non-saturating entry (100,150,200) does not.
var groundTint = [3]uint8{20, 30, 15}

// TestCellPixelsAddsTheTintSaturatingAt255 - AC-9: a tinted cell texture
// differs from the untinted one by exactly the tint per channel, saturating
// at 255 rather than wrapping, and alpha never moves.
func TestCellPixelsAddsTheTintSaturatingAt255(t *testing.T) {
	src := groundTintCellPixels()
	zero := cellPixels(src, [3]uint8{})
	tinted := cellPixels(src, groundTint)

	b := src.Bounds()
	sawSaturation := false
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			z := zero.RGBAAt(x, y)
			c := tinted.RGBAAt(x, y)
			want := color.RGBA{
				R: terrain.TintChannel(z.R, groundTint[0]),
				G: terrain.TintChannel(z.G, groundTint[1]),
				B: terrain.TintChannel(z.B, groundTint[2]),
				A: z.A,
			}
			if c != want {
				t.Fatalf("pixel (%d,%d) tinted = %v, want %v — the zero-tint colour plus this band's "+
					"tint, saturating at 255", x, y, c, want)
			}
			if c.A != z.A {
				t.Fatalf("pixel (%d,%d) alpha moved from %d to %d; the tint touches no channel but RGB", x, y, z.A, c.A)
			}
			if want.R == 255 || want.G == 255 || want.B == 255 {
				sawSaturation = true
			}
		}
	}
	if !sawSaturation {
		t.Fatal("fixture drift: no pixel reached the saturating clamp; AC-9's clamp half is untested")
	}
}

// TestCellPixelsZeroTintIsByteIdenticalToTheSource - AC-9's second half: under
// a zero tint cellPixels reproduces the source's own per-pixel colour exactly,
// with no channel moved by even one.
func TestCellPixelsZeroTintIsByteIdenticalToTheSource(t *testing.T) {
	src := groundTintCellPixels()
	got := cellPixels(src, [3]uint8{})

	b := src.Bounds()
	distinct := map[color.RGBA]bool{}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			want := paletteRGBA(src, b.Min.X+x, b.Min.Y+y)
			if got.RGBAAt(x, y) != want {
				t.Fatalf("pixel (%d,%d) = %v, want the untinted source's own %v", x, y, got.RGBAAt(x, y), want)
			}
			distinct[want] = true
		}
	}
	if len(distinct) < 2 {
		t.Fatal("fixture drift: fewer than two distinct colours painted; a wrong palette walk could not be caught here")
	}
}

// TestGroundCacheKeyDiscriminatesOnTintAlone - AC-8, the ground half: an
// unchanged light hands the same texture back and grows the cache by nothing;
// a light differing ONLY in its sky tint uploads a second, distinct texture;
// and returning to the first tint hands the FIRST texture back rather than
// uploading a third — the whole of "the caches discriminate" on this path.
func TestGroundCacheKeyDiscriminatesOnTintAlone(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), groundTintTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}
	if v.sun.SkyTint != ([3]uint8{}) {
		t.Fatalf("fixture drift: a fresh viewer's tint is %v, want the seeded zero", v.sun.SkyTint)
	}

	img1 := v.cellImage(v.tileWord(0, 0), 0, 0)
	if len(v.cache) != 1 {
		t.Fatalf("cache holds %d entries after one draw, want 1", len(v.cache))
	}

	// Identical light, called again: no growth and the same texture.
	if again := v.cellImage(v.tileWord(0, 0), 0, 0); again != img1 || len(v.cache) != 1 {
		t.Fatalf("an unchanged light re-uploaded: cache now %d entries, same pointer %v", len(v.cache), again == img1)
	}

	// A tint-only difference: ambient, range and theta are untouched.
	v.sun.SkyTint = [3]uint8{9, 0, 5}
	img2 := v.cellImage(v.tileWord(0, 0), 0, 0)
	if img2 == img1 {
		t.Fatal("a tint-only difference handed back the SAME texture; the tint is not in the key")
	}
	if len(v.cache) != 2 {
		t.Fatalf("cache holds %d entries after a tint-only change, want 2", len(v.cache))
	}

	// Back to the original tint: the original texture, not a third upload.
	v.sun.SkyTint = [3]uint8{}
	img3 := v.cellImage(v.tileWord(0, 0), 0, 0)
	if img3 != img1 {
		t.Fatal("returning to the original tint did not reuse the original texture")
	}
	if len(v.cache) != 2 {
		t.Fatalf("returning to the original tint grew the cache to %d, want 2 (no third upload)", len(v.cache))
	}
}

// TestSpriteCacheKeyDiscriminatesOnTintAlone - AC-8, the sprite half: with the
// row held fixed (ambient is untouched), a tint-only difference in v.sun gives
// spriteKey a different value, and returning to the first tint gives the
// original value back.
func TestSpriteCacheKeyDiscriminatesOnTintAlone(t *testing.T) {
	v, f, _ := spriteLightViewer(t)

	key1 := v.spriteKey(f)
	v.sun.SkyTint = [3]uint8{9, 0, 5}
	key2 := v.spriteKey(f)
	if key2.row != key1.row {
		t.Fatalf("fixture drift: changing SkyTint alone moved the row from %d to %d", key1.row, key2.row)
	}
	if key1 == key2 {
		t.Fatal("a tint-only difference produced equal spriteTextureKeys; the tint is not in the key")
	}

	v.sun.SkyTint = [3]uint8{}
	key3 := v.spriteKey(f)
	if key3 != key1 {
		t.Fatalf("returning to the original tint gave %+v, want the original key %+v", key3, key1)
	}
}

func TestGroundPlaceholderKeyIgnoresTint(t *testing.T) {
	g := grid(1, 1, slotWord(1)) // slot 4: absent in groundTintTileset
	g.Altitudes = []uint8{0}

	v, err := NewViewer("m", g, groundTintTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	img1 := v.cellImage(v.tileWord(0, 0), 0, 0)
	if img1 != v.placeholder {
		t.Fatal("an absent slot's cached image is not the shared placeholder")
	}

	v.sun.SkyTint = [3]uint8{9, 0, 5}
	img2 := v.cellImage(v.tileWord(0, 0), 0, 0)
	if img2 != img1 {
		t.Fatal("a placeholder cell's texture moved when only the tint changed")
	}
	if len(v.cache) != 1 {
		t.Fatalf("cache holds %d entries after a tint-only change over a placeholder cell, want 1", len(v.cache))
	}
}

// TestGroundTextureUnshadedIsUntintedAndCornersAreOne - AC-10: under the
// unshaded diagnostic, over a light no band at DefaultDaytime can produce (as
// TestTheSpriteWireIsTheCacheAndNotAConstant writes directly for the sprite
// path), the ground texture is byte-identical to the untinted source and
// every tile's corner multiplier is 1.
func TestGroundTextureUnshadedIsUntintedAndCornersAreOne(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), groundTintTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.sun.SkyTint = [3]uint8{9, 0, 5}
	v.SetUnshaded(true)
	if v.Lit() {
		t.Fatal("SetUnshaded(true) left the viewer lit")
	}

	if got := v.lightTint(); got != ([3]uint8{}) {
		t.Fatalf("lightTint() = %v under the unshaded diagnostic, want zero", got)
	}

	ref := v.resolveCell(v.tileWord(0, 0), 0, 0)
	src := v.set.Slot(ref.Slot).SubCell(ref.Sub)
	if src == nil {
		t.Fatal("fixture drift: the fixed word must resolve to the populated slot")
	}
	got := cellPixels(src, v.lightTint())
	want := cellPixels(src, [3]uint8{})
	if !equalBytes(got.Pix, want.Pix) {
		t.Fatal("the unshaded ground texture is not byte-identical to the untinted source")
	}

	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			if got := v.cornerScales(tx, ty); got != ([4]float32{1, 1, 1, 1}) {
				t.Fatalf("tile (%d,%d) cornerScales = %v under the unshaded diagnostic, want all-1", tx, ty, got)
			}
		}
	}
}
