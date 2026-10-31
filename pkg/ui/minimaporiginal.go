package ui

import (
	"image"
	"image/color"
	"slices"

	"againrom/pkg/render/terrain"
)

// MinimapPaint is the installed data the first game's minimap is painted
// from. Colours is interface/MiniMapData.bmp, the 52x14 lookup a tile word
// selects its terrain colour from (MISSION-083). OwnerShades holds, for each
// of the sixteen owner shade tables, the colour word +0x1148 of that table:
// row 8, the neutral level, of palette entry 164 (MISSION-085,
// TERR-LIGHT-018, TERR-LIGHT-060). A viewer given no MinimapPaint draws the
// symbolic minimap.
type MinimapPaint struct {
	Colours     *image.RGBA
	OwnerShades [16]color.RGBA
}

// minimapOriginalState is the first game's paint data and what the minimap
// keeps between repaints: the terrain colours built for one brightness plane,
// and the last composed picture with the counter value it was painted at.
type minimapOriginalState struct {
	paint         *MinimapPaint
	playerColours []uint32

	terrain       []color.RGBA
	terrainLevels *uint8
	terrainBuilt  bool

	pic     *image.RGBA
	at      image.Point
	key     minimapRepaintKey
	count   uint32
	painted bool
}

// minimapRepaintKey is every engine input that repaints the minimap at once,
// whatever the counter says: the geometry, the outlined cells, the reveal and
// the loot option. The original's full repaint trigger set is open
// (MISSION-087); these are this build's own triggers (DIV-2998).
type minimapRepaintKey struct {
	box, content image.Rectangle
	view         image.Rectangle
	reveal       bool
	hideLoot     bool
	frame        *DialogFrame
	paint        *MinimapPaint
	held         bool
}

// minimapRepaintSteps is the counter difference below which the original
// keeps its last minimap picture (MISSION-087).
const minimapRepaintSteps = 11

// SetMinimapPaint gives the viewer the first game's minimap data. Nil keeps
// the symbolic minimap.
func (v *Viewer) SetMinimapPaint(p *MinimapPaint) {
	if v.minimapOriginal.paint != p {
		v.minimapOriginal = minimapOriginalState{paint: p, playerColours: v.minimapOriginal.playerColours}
	}
}

// SetPlayerColours gives authored maps their roster's colour slots.
func (v *Viewer) SetPlayerColours(colours []uint32) {
	v.minimapOriginal.playerColours = slices.Clone(colours)
	v.minimapOriginal.painted = false
}

// minimapOriginalActive reports whether the minimap is painted by the first
// game's rules: the paint data and the installed crystal background are both
// present.
func (v *Viewer) minimapOriginalActive() bool {
	p := v.minimapOriginal.paint
	return p != nil && p.Colours != nil && v.dialogFrame != nil && v.dialogFrame.Minimap != nil
}

// minimapOriginalTerrain is one colour per grid cell, row-major: the
// MiniMapData sample of the tile word, scaled by the cell's brightness
// (MISSION-083). The tile index t = word & 0x1fff samples (t>>4, t&15); a
// sample outside the lookup is terrain.PlaceholderColor (DIV-2995). The
// brightness factor is 96 minus the truncated mean of the cell's four corner
// levels, the same level the terrain shade table is indexed by; each channel
// becomes min(255, channel*factor>>5). Without a level plane the factor is 32,
// the identity. The colours are rebuilt when the level plane is replaced.
func (v *Viewer) minimapOriginalTerrain() []color.RGBA {
	st := &v.minimapOriginal
	var key *uint8
	if len(v.levels) > 0 {
		key = &v.levels[0]
	}
	if st.terrainBuilt && st.terrainLevels == key {
		return st.terrain
	}
	cols, rows := v.grid.Width, v.grid.Height
	out := make([]color.RGBA, cols*rows)
	lookup := st.paint.Colours
	lb := lookup.Bounds()
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			t := int(v.tileWord(col, row) & terrain.TileGroupBits)
			x, y := t>>4, t&15
			c := terrain.PlaceholderColor
			if x < lb.Dx() && y < lb.Dy() {
				c = lookup.RGBAAt(lb.Min.X+x, lb.Min.Y+y)
			}
			factor := 32
			if len(v.levels) >= cols*rows {
				lv := terrain.CornerLevels(v.levels, cols, rows, col, row)
				factor = 96 - (int(lv[0])+int(lv[1])+int(lv[2])+int(lv[3]))/4
			}
			out[row*cols+col] = color.RGBA{
				R: minimapGain(c.R, factor), G: minimapGain(c.G, factor), B: minimapGain(c.B, factor), A: 0xff,
			}
		}
	}
	st.terrain, st.terrainLevels, st.terrainBuilt = out, key, true
	return out
}

// minimapGain is one channel times factor/32, saturated (MISSION-083).
func minimapGain(c uint8, factor int) uint8 {
	return uint8(min(255, max(0, int(c)*factor>>5)))
}

// minimapFogWord is a cell's fog state as the tile word's two fog bits:
// visible 0xc000, explored 0x8000, unseen 0 (TERR-TILE-079, TERR-FOG-082).
func (v *Viewer) minimapFogWord(col, row int) uint16 {
	if v.minimapHeld {
		return 0
	}
	switch v.fogAt(col, row) {
	case FogVisible:
		return 0xc000
	case FogExplored:
		return 0x8000
	}
	return 0
}

// minimapBlockFog is the masked OR of the four fog words the original's block
// loop reads for map cell (col, row): i, i+1, i+W+1 and i+W+2, the lower pair
// one cell right of an ordinary quad (MISSION-084).
func (v *Viewer) minimapBlockFog(col, row int) uint16 {
	return v.minimapFogWord(col, row) | v.minimapFogWord(col+1, row) |
		v.minimapFogWord(col+1, row+1) | v.minimapFogWord(col+2, row+1)
}

// minimapMarkFog is the masked OR of the ordinary four-cell quad at an
// object's cell, the gate of every mark (MISSION-085).
func (v *Viewer) minimapMarkFog(col, row int) uint16 {
	return v.minimapFogWord(col, row) | v.minimapFogWord(col+1, row) |
		v.minimapFogWord(col, row+1) | v.minimapFogWord(col+1, row+1)
}

// minimapOwnerColour is the mark colour of an owner roster slot: its shade
// table's word +0x1148 plus the sky tint the unit sprites carry, saturated.
// Campaign maps use the slot's low nibble; authored maps supply roster colours.
// Slot 0 names no owner and has no colour. DIV-2996. A game whose mission
// screen names its own local mark colour gives the local owner that colour
// (DIV-4118).
func (v *Viewer) minimapOwnerColour(owner uint32) (color.RGBA, bool) {
	if owner == 0 || v.minimapOriginal.paint == nil {
		return color.RGBA{}, false
	}
	if c := v.missionScreen.MinimapLocal; owner == v.localOwner && c.A != 0 {
		return c, true
	}
	shade := owner & 15
	if uint64(owner) <= uint64(len(v.minimapOriginal.playerColours)) {
		shade = v.minimapOriginal.playerColours[owner-1] & 15
	}
	c := v.minimapOriginal.paint.OwnerShades[shade]
	tint := v.lightTint()
	return color.RGBA{
		R: uint8(min(255, int(c.R)+int(tint[0]))),
		G: uint8(min(255, int(c.G)+int(tint[1]))),
		B: uint8(min(255, int(c.B)+int(tint[2]))),
		A: 0xff,
	}, true
}

// minimapOriginalMarks is the first game's marks in grid cells: loot dots
// (DIV-2954), structures drawn under them, and units (MISSION-085). A structure's extents
// are its class's TileWidth and TileHeight, or its own Size for a
// VariableSize class (the wooden bridges). A unit or air unit is one cell and
// needs a corpse stage of at most 1. Every mark needs its quad's fog OR to be
// 0xc000 and an owner with a colour; another owner's unit also needs its own
// cell visible, the owner's fog rule for every unit indicator (DIV-2997).
func (v *Viewer) minimapOriginalMarks() []minimapMark {
	if v.minimapHeld {
		return nil
	}
	out := v.minimapLootMarks()
	for _, s := range v.structureStates {
		c, ok := v.minimapOwnerColour(s.Owner)
		if !ok || v.minimapMarkFog(s.Cell.X, s.Cell.Y) != 0xc000 {
			continue
		}
		var size image.Point
		if v.structureSet != nil && int(s.Kind) < len(v.structureSet.Classes) {
			if cls := v.structureSet.Classes[s.Kind]; cls != nil {
				size = image.Pt(cls.TileWidth, cls.TileHeight)
				if cls.VariableSize {
					size = s.Size
				}
			}
		}
		if size.X <= 0 || size.Y <= 0 {
			continue
		}
		out = append(out, minimapMark{Cell: s.Cell, Size: size, Colour: c, Local: s.Owner == v.localOwner, Under: true})
	}
	for _, e := range v.entities {
		c, ok := v.minimapOwnerColour(e.Owner)
		if !ok || e.CorpseStage > 1 || v.minimapMarkFog(e.Cell.X, e.Cell.Y) != 0xc000 || !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		out = append(out, minimapMark{Cell: e.Cell, Size: image.Pt(1, 1), Colour: c, Local: e.Owner == v.localOwner})
	}
	return out
}

// minimapOriginalPlan is the first game's picture for geometry g. Each pixel
// shows its cell's terrain where the block fog OR has bit 14, the average of
// terrain and the CrystalR background pixel under it where the OR is exactly
// 0x8000, and the background pixel where it is zero (MISSION-084). The marks
// follow, then the white outline of the camera's cells (MISSION-086).
func (v *Viewer) minimapOriginalPlan(g minimapGeom) minimapPlan {
	colours := v.minimapOriginalTerrain()
	back := v.dialogFrame.Minimap
	offset := g.Content.Min.Sub(g.Box.Min)
	cols := v.grid.Width
	var marks []minimapMark
	for _, m := range v.minimapOriginalMarks() {
		m.Cell = m.Cell.Sub(g.Origin)
		if m.Size == (image.Point{}) && !m.Cell.In(image.Rect(0, 0, g.Cols, g.Rows)) {
			continue
		}
		marks = append(marks, m)
	}
	return minimapPlan{
		cols: g.Cols, rows: g.Rows,
		pixel: func(col, row int, at image.Point) color.RGBA {
			gc, gr := col+g.Origin.X, row+g.Origin.Y
			t := colours[gr*cols+gc]
			fog := v.minimapBlockFog(gc, gr)
			if fog&0x4000 != 0 {
				return t
			}
			p := offset.Add(at)
			bg := color.RGBA{A: 0xff}
			if p.In(back.Bounds()) {
				bg = back.RGBAAt(p.X, p.Y)
				bg.A = 0xff
			}
			if fog == 0 {
				return bg
			}
			return color.RGBA{R: bg.R>>1 + t.R>>1, G: bg.G>>1 + t.G>>1, B: bg.B>>1 + t.B>>1, A: 0xff}
		},
		marks:           marks,
		view:            v.minimapViewCells().Sub(g.Origin),
		viewColour:      color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		separateOutline: true,
	}
}

// strokeMinimapView draws the camera's cell range as four inclusive edges:
// left and top are the scaled origin, right and bottom add the separately
// scaled span less one (MISSION-086). Scaling truncates toward zero. A pixel
// outside the picture is dropped; an edge is never moved onto the picture's
// border.
func strokeMinimapView(img *image.RGBA, view image.Rectangle, num, den int, c color.RGBA) {
	if view.Empty() || den <= 0 {
		return
	}
	x0, y0 := minimapCellPixel(view.Min.X, num, den), minimapCellPixel(view.Min.Y, num, den)
	x1 := x0 + max(1, minimapCellPixel(view.Dx(), num, den)) - 1
	y1 := y0 + max(1, minimapCellPixel(view.Dy(), num, den)) - 1
	w, h := x1-x0+1, y1-y0+1
	fillBlock(img, x0, y0, w, 1, c)
	fillBlock(img, x0, y1, w, 1, c)
	fillBlock(img, x0, y0, 1, h, c)
	fillBlock(img, x1, y0, 1, h, c)
}

// minimapRepaint is the picture Draw shows for the first game: the last
// composed one while the presentation counter has moved fewer than
// minimapRepaintSteps since it was painted and no repaint key changed, and a
// fresh one otherwise, which records the counter (MISSION-087).
func (v *Viewer) minimapRepaint(g minimapGeom) (*image.RGBA, image.Point, bool) {
	st := &v.minimapOriginal
	key := minimapRepaintKey{
		box: g.Box, content: g.Content, view: v.minimapViewCells(),
		reveal: v.fogReveal, hideLoot: v.graphics.HideLootDots,
		frame: v.dialogFrame, paint: st.paint, held: v.minimapHeld,
	}
	count := v.anim.Count()
	diff := int32(count - st.count)
	if diff < 0 {
		diff = -diff
	}
	if st.painted && st.key == key && diff < minimapRepaintSteps {
		return st.pic, st.at, st.pic != nil
	}
	pic, at, ok := v.minimapCompose(g)
	st.pic, st.at, st.key, st.count, st.painted = pic, at, key, count, true
	if !ok {
		st.pic = nil
	}
	return pic, at, ok
}

// minimapMarkColours is the local participant's mark colour and the set of
// every other owner's mark colour now on the map, for the headless mark
// witness. A colour another owner shares with the local one counts as local.
func (v *Viewer) minimapMarkColours() (color.RGBA, map[color.RGBA]bool) {
	if !v.minimapOriginalActive() {
		return minimapLocalColor, map[color.RGBA]bool{minimapOtherColor: true}
	}
	local, _ := v.minimapOwnerColour(v.localOwner)
	others := map[color.RGBA]bool{}
	add := func(owner uint32) {
		if c, ok := v.minimapOwnerColour(owner); ok && owner != v.localOwner && c != local {
			others[c] = true
		}
	}
	for _, e := range v.entities {
		add(e.Owner)
	}
	for _, s := range v.structureStates {
		add(s.Owner)
	}
	return local, others
}
