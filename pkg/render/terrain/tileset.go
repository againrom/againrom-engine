package terrain

import (
	"fmt"
	"image"
)

const (
	// CellSize is the edge of one terrain cell in pixels. Every terrain tile
	// file is a 32 px wide vertical strip of 32x32 sub-cells: 448 px tall gives
	// 14 of them (land and road), 256 px gives 8 (water), 128 px gives 4 (dirt).
	CellSize = 32

	// SlotCount is the size of the tile array the game loads into,
	// tiles[(G-1)*16 + V] for file group G = 1..8 and variant V = 0..15.
	SlotCount = 128

	slotGroups   = 8  // G = 1..8
	slotVariants = 16 // V = 0..15

	// TilePathPrefix is the address prefix the tile strips live under:
	// graphics.res's own identity segment, then the node inside it, in the folded
	// form every address is compared in (lower case, '/' separators).
	// TERR-FAMILY-187 establishes the ordinary source table. The optional
	// terrain.3d precomposition pass has separate temporary sources.
	//
	// The identity segment is part of the prefix BECAUSE it is part of the
	// name: one address identifies an entry across every container an install
	// ships, so this package's constants say which container the tiles come
	// from instead of leaving the caller to know it out of band on the
	// package's behalf.
	TilePathPrefix = "graphics/terrain/"

	// DirtPath is the impassable-tile composite overlay strip (32x128). It is
	// DERIVED from the prefix and never spelt out, so it cannot come to name a
	// container the tiles do not.
	DirtPath = TilePathPrefix + "dirt.bmp"
)

// Strip is one decoded tile file: a vertical run of 32x32 sub-cells in top-down
// order. Sub-cell k is the block at rows [k*CellSize, (k+1)*CellSize) — the
// same ordering the game reaches through its loaded-buffer offset 8 + k*0x400
// (0x400 = 32x32 at 8 bpp), less that buffer's own 8-byte prefix.
type Strip struct {
	SubCells []*image.Paletted
}

// SubCell returns sub-cell k, or nil when the strip is absent or too short
// to hold k. A null or short slot is a reported condition, never a crash.
func (s *Strip) SubCell(k int) *image.Paletted {
	if s == nil || k < 0 || k >= len(s.SubCells) {
		return nil
	}
	return s.SubCells[k]
}

// EntrySource supplies raw entry bytes by name. It is the whole of this
// package's dependency on the container layer, and it DID NOT MOVE when the
// tile constants became addresses: *vfs.FS satisfies this one method
// unchanged, so the seam is the same shape and no container or formats type
// crosses into the render tier (see docs/ARCHITECTURE.md).
type EntrySource interface {
	ReadFile(name string) ([]byte, error)
}

// Tileset is the loaded terrain tile model: the 128 tile slots plus the dirt
// overlay used to mark impassable ground.
type Tileset struct {
	Slots   [SlotCount]*Strip // tiles[(G-1)*16 + V]; nil means absent
	Dirt    *Strip            // DirtPath's strip
	Loaded  int               // populated tile slots (Dirt not counted)
	Missing []string          // entries that were absent or would not decode
}

// Slot returns the strip in tiles[i], or nil when i is outside the array or
// the slot is absent. Callers resolve tile words to slot indices, so an
// off-corpus word must not be able to index out of range.
func (t *Tileset) Slot(i int) *Strip {
	if t == nil || i < 0 || i >= SlotCount {
		return nil
	}
	return t.Slots[i]
}

// TilePath returns the address of tile file group g (1..8), variant v (0..15):
// TilePathPrefix + tileG-VV.bmp with the variant zero-padded to two digits.
func TilePath(g, v int) string {
	return fmt.Sprintf("%stile%d-%02d.bmp", TilePathPrefix, g, v)
}

// SlotIndex returns the tiles[] slot that file group g, variant v loads into:
// (g-1)*16 + v (TERR-LOAD-002).
func SlotIndex(g, v int) int { return (g-1)*slotVariants + v }

// SliceStrip cuts a decoded tile strip into its 32x32 sub-cells in top-down
// order. A strip must be exactly CellSize wide and a positive whole number
// of cells tall; anything else is rejected so a slot can never hold a
// partial cell.
func SliceStrip(img *image.Paletted) (*Strip, error) {
	if img == nil {
		return nil, fmt.Errorf("terrain: nil strip image")
	}
	b := img.Bounds()
	if b.Dx() != CellSize {
		return nil, fmt.Errorf("terrain: strip is %d px wide, want %d", b.Dx(), CellSize)
	}
	if b.Dy() <= 0 || b.Dy()%CellSize != 0 {
		return nil, fmt.Errorf("terrain: strip is %d px tall, want a positive multiple of %d", b.Dy(), CellSize)
	}

	// Every sub-cell shares the strip's palette; only the index plane is cut, so
	// slicing preserves the palette indices the dirt overlay and the shading
	// table address pixels by.
	cells := make([]*image.Paletted, b.Dy()/CellSize)
	for k := range cells {
		cell := image.NewPaletted(image.Rect(0, 0, CellSize, CellSize), img.Palette)
		for y := 0; y < CellSize; y++ {
			src := img.Pix[(b.Min.Y+k*CellSize+y-img.Rect.Min.Y)*img.Stride+(b.Min.X-img.Rect.Min.X):]
			copy(cell.Pix[y*cell.Stride:y*cell.Stride+CellSize], src[:CellSize])
		}
		cells[k] = cell
	}
	return &Strip{SubCells: cells}, nil
}

// LoadTileset reads the tile strips at this package's own addresses out of
// src and builds the 128-slot model: tiles[(G-1)*16 + V] <- tileG-VV.bmp for
// G = 1..8, V = 0..15, plus dirt.bmp in its own slot (TERR-LOAD-002). Each
// entry is decoded and sliced into its 32x32 sub-cells.
//
// src is asked for ADDRESSES, not for bare entry paths: every name this
// function builds carries graphics.res's identity segment, so it resolves
// against the container filesystem over an install's archives and states
// which container the tiles come from.
//
// Absence is the normal case, not an error: a shipped install carries only
// tile1/2/3 (16 variants each), tile4-00..03 and dirt.bmp, so groups 5-8 and
// tile4 V >= 4 stay null exactly as the claim describes. An entry that is
// missing, is not a BMP in the accepted subset, or whose geometry is not a whole
// number of cells leaves its slot nil and is recorded in Missing. Loading itself
// never fails and never panics (AC-7).
func LoadTileset(src EntrySource) *Tileset {
	ts := &Tileset{}
	if src == nil {
		ts.Missing = append(ts.Missing, "<no entry source>")
		return ts
	}

	load := func(path string) *Strip {
		data, err := src.ReadFile(path)
		if err != nil {
			ts.Missing = append(ts.Missing, path)
			return nil
		}
		img, err := DecodeBMP8(data)
		if err != nil {
			ts.Missing = append(ts.Missing, path)
			return nil
		}
		strip, err := SliceStrip(img)
		if err != nil {
			ts.Missing = append(ts.Missing, path)
			return nil
		}
		return strip
	}

	for g := 1; g <= slotGroups; g++ {
		for v := 0; v < slotVariants; v++ {
			if strip := load(TilePath(g, v)); strip != nil {
				ts.Slots[SlotIndex(g, v)] = strip
				ts.Loaded++
			}
		}
	}
	ts.Dirt = load(DirtPath)
	return ts
}
