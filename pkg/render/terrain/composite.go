package terrain

import (
	"fmt"
	"image"
	"image/color"
)

// maxRenderPixels caps a composite so a hostile or mistaken width/height/scale
// cannot ask for an unbounded allocation. 2^28 pixels is a full 256x256 map at
// scale 2 (16384 x 16384), comfortably past anything the game ships.
const maxRenderPixels = 1 << 28

// PlaceholderColor fills a cell whose strip slot is absent or too short. It is
// deliberately a colour no terrain tile would hold, so a fallback is obvious on
// sight. Own engineering, not a decoded game fact.
var PlaceholderColor = color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff}

// Grid is a map's cell layers: Width*Height tile words in row-major order,
// exactly as alm.Map.Tiles holds them, and — optionally — the map's altitude and
// object grids alongside them. Passing them all as plain integers is what keeps
// the map format out of the render tier.
type Grid struct {
	Width  int
	Height int
	Tiles  []uint16

	// Altitudes is the map's Width*Height altitude grid in row-major order,
	// read SIGNED at the point of use, or nil for a caller that has none. It
	// rides here because it is part of the MAP, not part of a viewer's
	// configuration, and every construction site uses a keyed literal, so a
	// caller that does not set it is unaffected.
	//
	// The compositors below do NOT read it: they take the altitudes as their
	// own parameter, because a raster's geometry and a caller's map layers are
	// different arguments to them.
	Altitudes []uint8

	// Overlay is the map's Width*Height object grid in row-major order — one
	// PLACEMENT BYTE per cell, exactly as alm.Map.Overlay holds them, 0 meaning
	// *no object* — or nil for a caller that has none. It rides here for the
	// reason Altitudes does: it is part of the MAP rather than part of a
	// viewer's configuration, and every construction site uses a keyed literal,
	// so a caller that does not set it is unaffected.
	//
	// The compositors do not read it either, and for a stronger reason than
	// they have for the altitudes: what stands ON the terrain is a separate
	// layer drawn after all of it (0017 spec, "Draw order"). StaticPlacements
	// is this field's one reader, and it validates the length itself rather
	// than trusting the type — a Grid is a bag of layers a caller assembles,
	// not an invariant this package establishes.
	Overlay []uint8

	// Block is the map's Width*Height DERIVED block plane in row-major order
	// — one byte per cell, as pkg/mapload derives it for the simulation —
	// or nil for a caller that has none. It rides here for the reason Altitudes
	// and Overlay do: it is a fact about the MAP rather than part of a viewer's
	// configuration, and every construction site uses a keyed literal, so a
	// caller that does not set it is unaffected.
	//
	// It is DERIVED and not decoded, which is the one way it differs from its
	// two neighbours here: no ALM record carries it. That is why it is carried
	// as a plane at all — the rule that derives it lives at a tier this one
	// may not import, so the answer travels as bytes rather than as a constant.
	//
	// BorderCell is this field's one reader, and it validates the length itself
	// rather than trusting the type, exactly as Overlay's does.
	Block []uint8

	// Structures is the map's type-4 placement records — one per placed
	// structure, exactly as alm.Map.Objects holds them, carried as plain
	// unsigned integers so no format type crosses into this tier — or nil for
	// a caller that has none.
	//
	// It rides here for the reason Altitudes, Overlay and Block do: a placement
	// record is part of the MAP rather than part of a viewer's configuration,
	// and every construction site uses a keyed literal, so a caller that does
	// not set it is unaffected.
	//
	// It is a RECORD LIST and not a plane, which is the one way it differs from
	// its three neighbours: a structure covers a rectangle of cells from an
	// anchor its own record carries, so there is no per-cell layer for it to
	// arrive as. StructurePlacements is this field's one reader; it establishes
	// no invariant here, because a Grid is a bag of layers a caller assembles.
	Structures []StructureRecord
}

type Render struct {
	Image        *image.RGBA
	Placeholders int

	// OriginY is the native row — before scale — that the image's row 0
	// stands for. It is 0 for the flat raster, whose canvas starts at native
	// row 0 by construction, and the projection's own minV for a
	// height-displaced render, whose canvas is the half-open [minV, maxV) its
	// vertex mesh spans. A caller drawing on the un-displaced cell lattice —
	// the diagnostic overlays — must shift by it, and the tool reports it.
	OriginY int
}

// The two values validateComposite's needHeights parameter takes, so a call site
// says which grids it carries rather than passing a bare boolean.
const (
	withHeights    = true
	withoutHeights = false
)

func validateComposite(ts *Tileset, g Grid, heights []uint8, needHeights bool, scale int, canvasRows func() int) (wpx, hpx int64, err error) {
	if ts == nil {
		return 0, 0, fmt.Errorf("terrain: nil tileset")
	}
	if g.Width <= 0 || g.Height <= 0 {
		return 0, 0, fmt.Errorf("terrain: map is %dx%d, want positive dimensions", g.Width, g.Height)
	}
	cells := int64(g.Width) * int64(g.Height)
	if int64(len(g.Tiles)) != cells {
		return 0, 0, fmt.Errorf("terrain: grid holds %d cells, want %d (W*H)", len(g.Tiles), cells)
	}
	if needHeights && int64(len(heights)) != cells {
		return 0, 0, fmt.Errorf("terrain: height grid holds %d cells, want %d (W*H)", len(heights), cells)
	}
	if scale < 1 {
		return 0, 0, fmt.Errorf("terrain: scale %d, want at least 1", scale)
	}

	wpx = int64(g.Width) * int64(CellSize) * int64(scale)
	hpx = int64(canvasRows()) * int64(scale)
	if wpx*hpx > maxRenderPixels {
		return 0, 0, fmt.Errorf("terrain: composite would be %dx%d px, over the %d-pixel cap; try a smaller scale", wpx, hpx, maxRenderPixels)
	}
	return wpx, hpx, nil
}

// Composite draws a whole map's terrain into one image. Each cell's tile
// word is resolved to a strip slot and sub-cell and the resulting 32x32
// sub-cell is placed at map position (col*32, row*32), every source pixel
// replicated scale x scale.
//
// An impassable non-water cell (bit 13) is composited over dirt.bmp sub-cell
// (col + row*5) & 3 before it is drawn (TERR-SEM-004); if the dirt strip is
// absent the tile is drawn alone. A cell whose strip slot is absent, out of
// range, or too short for the resolved sub-cell is filled with
// PlaceholderColor and counted — never a crash and never an out-of-range
// index.
//
// Arguments are validated up front and rejected atomically: a bad tileset, a
// non-positive dimension, a grid whose length is not Width*Height, a scale below
// 1, or an output larger than maxRenderPixels yields a nil render and an error.
func Composite(ts *Tileset, g Grid, scale int) (*Render, error) {
	// The flat raster's canvas is Height cells tall whatever the altitudes are,
	// which is the whole of what makes it flat — a multiplication, not a
	// measurement, but taken through the same parameter so both geometries reach
	// the budget check by one path.
	wpx, hpx, err := validateComposite(ts, g, nil, withoutHeights, scale, func() int { return g.Height * CellSize })
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, int(wpx), int(hpx)))
	// OriginY stays 0: this canvas begins at native row 0.
	out := &Render{Image: img}

	cellpx := CellSize * scale
	for row := 0; row < g.Height; row++ {
		for col := 0; col < g.Width; col++ {
			word := g.Tiles[row*g.Width+col]
			ref := Resolve(word)

			// Slot and SubCell are both bounds-checked and nil-safe, so an
			// off-corpus word lands here as a nil sub-cell, not a panic.
			src := ts.Slot(ref.Slot).SubCell(ref.Sub)
			if src == nil {
				fillCell(img, col*cellpx, row*cellpx, cellpx, PlaceholderColor)
				out.Placeholders++
				continue
			}

			var dirt *image.Paletted
			if IsImpassable(word) && !ref.Water {
				dirt = ts.Dirt.SubCell(DirtSubCell(col, row))
			}
			drawCell(img, col*cellpx, row*cellpx, scale, src, dirt)
		}
	}
	return out, nil
}

// drawCell blits one 32x32 sub-cell as an axis-aligned rectangle whose top-left
// destination pixel is (originX, originY), replicating each source pixel
// scale x scale. A non-nil dirt sub-cell makes this the impassable composite
// (see overlayPixel).
//
// The origin is the caller's, not (col, row): the flat raster puts a cell at
// (col*32, row*32)*scale, while a height-displaced render puts a flat cell
// at its own projected top edge, in the same rectangle at a destination row
// the cell grid alone does not give.
func drawCell(dst *image.RGBA, originX, originY, scale int, src, dirt *image.Paletted) {
	sb := src.Bounds()
	for y := 0; y < CellSize; y++ {
		for x := 0; x < CellSize; x++ {
			c := paletteColor(src, sb.Min.X+x, sb.Min.Y+y)
			if dirt != nil {
				db := dirt.Bounds()
				c = overlayPixel(c, dirt, db.Min.X+x, db.Min.Y+y)
			}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetRGBA(originX+x*scale+sx, originY+y*scale+sy, c)
				}
			}
		}
	}
}

// paletteColor resolves a paletted pixel to RGBA. An index past the palette
// yields the placeholder rather than panicking, keeping the compositor
// total; bmp.DecodePaletted already rejects such files, so this is a belt-and-braces
// bound, not an expected path.
func paletteColor(img *image.Paletted, x, y int) color.RGBA {
	idx := int(img.ColorIndexAt(x, y))
	if idx >= len(img.Palette) {
		return PlaceholderColor
	}
	c, ok := img.Palette[idx].(color.RGBA)
	if !ok {
		r, g, b, a := img.Palette[idx].RGBA()
		return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	}
	return c
}

// fillCell paints one scaled cell — the square of side cellpx whose top-left
// destination pixel is (originX, originY) — a single colour: the placeholder
// path. Like drawCell it takes the destination origin rather than a cell
// position, so a placeholder can be placed wherever the cell it replaces is.
func fillCell(dst *image.RGBA, originX, originY, cellpx int, c color.RGBA) {
	for y := 0; y < cellpx; y++ {
		for x := 0; x < cellpx; x++ {
			dst.SetRGBA(originX+x, originY+y, c)
		}
	}
}

// DirtTransparentIndex is the dirt palette index that leaves the terrain pixel
// showing. Index 0 is the transparent key of the game's 8-bpp graphics
// (SPR256-PAL-013), and the dirt overlay honours it (TERR-DIRT-017).
const DirtTransparentIndex = 0

// overlayPixel composites the dirt overlay onto an impassable tile pixel.
//
// This is a decoded operation, not a chosen one (TERR-DIRT-017): the renderer
// copies the terrain sub-cell and then overlays a dirt sub-cell byte by byte,
// where a **non-zero** dirt palette index replaces the terrain pixel and a
// **zero** index leaves the terrain showing. There is no arithmetic and no
// blend — the earlier per-channel average here was a stand-in from before the
// operation was decoded, and it lightened every dirt pixel by mixing in terrain
// that the game replaces outright.
//
// Keying on the index, not the colour, is the whole point: index 0 is
// transparent whatever colour the palette happens to store there.
func overlayPixel(tile color.RGBA, dirt *image.Paletted, x, y int) color.RGBA {
	if dirt.ColorIndexAt(x, y) == DirtTransparentIndex {
		return tile
	}
	return paletteColor(dirt, x, y)
}
