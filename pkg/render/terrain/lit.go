package terrain

import "image"

// CompositeLit draws a whole map's terrain with relief shading. It is the shaded
// sibling of Composite (which stays available, unshaded, for comparison): each
// cell is resolved and placed exactly as Composite does, but every terrain pixel
// is attenuated by the per-vertex relief level (TERR-LIGHT-011). The level grid is
// computed from the height field and the light (LevelGrid), and per pixel the row
// is the bilinear interpolation of the cell's four corner levels (InterpRow),
// applied through the shading transform (ShadeRGBA) with the light's sky tint.
//
// A cell's four corner levels are the grid vertices at (col,row), (col+1,row),
// (col,row+1), (col+1,row+1). W*H vertices only fully bound (W-1)*(H-1) cells, so
// the last cell row/column has no +1 vertex; those indices are clamped to the
// grid edge. The game's own far edge is decoded (TERR-EDGE-024..026) and has NO
// defined shading to reproduce: it reads the missing corner at a raw flat offset
// into the same unpadded grid (last column -> the next row's column 0, last row
// -> past the allocation), and the brightness those reads sample was never
// computed on the outer ring. The ring is drawn, but what it shows is undefined.
// Clamping is therefore a deliberate, defined choice — the safe treatment the
// research names for a port — not an approximation of a known engine behaviour.
//
// Arguments are validated up front and rejected atomically, exactly as
// Composite plus a height-grid length check: a nil tileset, a non-positive
// dimension, a tile or height grid whose length is not Width*Height, a scale
// below 1, or an output over the pixel cap yields a nil render and an error.
// The placeholder path for an absent/short slot is unchanged (and unshaded
// — it is a diagnostic fill, not terrain), so an off-corpus word still
// lands as a reported placeholder, never a crash.
func CompositeLit(ts *Tileset, g Grid, heights []uint8, lt Light, scale int) (*Render, error) {
	wpx, hpx, err := validateComposite(ts, g, heights, withHeights, scale, func() int { return g.Height * CellSize })
	if err != nil {
		return nil, err
	}

	levels := LevelGrid(heights, g.Width, g.Height, lt)

	img := image.NewRGBA(image.Rect(0, 0, int(wpx), int(hpx)))
	// OriginY stays 0: shading changes a pixel's colour, never its position.
	out := &Render{Image: img}

	cellpx := CellSize * scale
	for row := 0; row < g.Height; row++ {
		for col := 0; col < g.Width; col++ {
			word := g.Tiles[row*g.Width+col]
			ref := Resolve(word)

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
			// The far-edge +1 corner is clamped into the grid by CornerLevels -- the
			// read this function and compositeProjected wrote independently before it
			// was made the one place (see the file doc comment).
			cl := CornerLevels(levels, g.Width, g.Height, col, row)
			drawCellLit(img, col*cellpx, row*cellpx, scale, src, dirt, lt.SkyTint,
				cl[0], cl[1], cl[2], cl[3])
		}
	}
	return out, nil
}

// drawCellLit blits one 32x32 sub-cell as an axis-aligned rectangle whose
// top-left destination pixel is (originX, originY), with relief shading: each
// source pixel (terrain, or the dirt overlay where an impassable cell shows it)
// is attenuated by the bilinear-interpolated level of its position in the cell,
// then replicated scale x scale. The four corner levels are l00 top-left, l10
// top-right, l01 bottom-left, l11 bottom-right.
//
// It takes the destination origin for the same reason drawCell does: the cell's
// rectangle is the same shape wherever the geometry puts it.
func drawCellLit(dst *image.RGBA, originX, originY, scale int, src, dirt *image.Paletted, tint [3]uint8, l00, l10, l01, l11 uint8) {
	sb := src.Bounds()
	for y := 0; y < CellSize; y++ {
		for x := 0; x < CellSize; x++ {
			c := paletteColor(src, sb.Min.X+x, sb.Min.Y+y)
			if dirt != nil {
				db := dirt.Bounds()
				c = overlayPixel(c, dirt, db.Min.X+x, db.Min.Y+y)
			}
			level := InterpRow(l00, l10, l01, l11, x, y, CellSize)
			c = ShadeRGBA(c, tint, level)
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetRGBA(originX+x*scale+sx, originY+y*scale+sy, c)
				}
			}
		}
	}
}
