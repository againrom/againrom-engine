package terrain

import "slices"

// RenderTileWords builds the light parser's private tile plane. ALM-TILEVIEW-122
// clears bit13 on load; the main parser keeps it for collision. Runtime fire
// updates this separate plane later (TERR-RLEBIT-168).
func RenderTileWords(stored []uint16) []uint16 {
	words := slices.Clone(stored)
	for i := range words {
		words[i] &^= ImpassableBit
	}
	return words
}

const (
	tileGroupMask  = 0x1fff // masked before the group shift
	tileGroupShift = 6      // g   = (w & 0x1fff) >> 6
	tileBlendShift = 4      // b   = (w >> 4) & 3
	tileBlendMask  = 3
	tileSubMask    = 0xf // sub = w & 0xf

	// ImpassableBit is bit 13 of a type1 tile word. An impassable non-water
	// cell is composited over dirt.bmp before blit (TERR-SEM-004); the flag has
	// no effect on which image or sub-cell is selected.
	ImpassableBit = 0x2000

	// Water occupies strip groups 8..11 (tile3, the animated group). The game's
	// renderer replaces the group with 8 + phase; the game's animation-disable
	// switch forces the phase to 0 rather than freezing the current one
	// (TERR-ANIM-009), so waterGroupBase is both the phase-0 group and the
	// disabled-animation group. Cycling lives in water.go.
	waterGroupLo   = 8
	waterGroupHi   = 11
	waterGroupBase = 8

	blendColumns = 4 // slot = g*4 + b
)

// TileRef is the graphic a tile word selects: the tiles[] slot holding the
// strip, the sub-cell within that strip, and whether the stored group was
// water (which suppresses the impassable dirt composite).
type TileRef struct {
	Slot  int  // g*4 + b, i.e. tiles[(G-1)*16 + V] with G = (g>>2)+1, V = (g&3)*4 + b
	Sub   int  // w & 0xf
	Water bool // stored group was 8..11 (tile3), so the slot carries an animation phase
}

// splitTileWord applies the TERR-IDX-003 split shared by the static and animated
// resolutions, so the two cannot drift apart.
func splitTileWord(word uint16) (g, b, sub int) {
	g = int(word&tileGroupMask) >> tileGroupShift
	b = int(word>>tileBlendShift) & tileBlendMask
	sub = int(word) & tileSubMask
	return g, b, sub
}

// Resolve maps a type1 tile word to the terrain graphic it draws, exactly as
// TERR-IDX-003 specifies:
//
//	g    = (w & 0x1fff) >> 6     strip group
//	b    = (w >> 4) & 3          blend column
//	sub  = w & 0xf               sub-cell
//	slot = g*4 + b               == tileG-VV.bmp, G = (g>>2)+1, V = (g&3)*4 + b
//
// Water (g 8..11, tile3) is animated in the game by replacing the group with
// 8 + phase. Resolve is the **animation-off** view: the game's disable switch
// forces phase 0 (TERR-ANIM-009), so water draws its base cell here. Use
// ResolveAnimated to draw a running cycle; the two agree for every non-water
// word, and for water Resolve equals ResolveAnimated at phase 0.
//
// Bit 13 (impassable) is deliberately not consulted — it selects the dirt
// composite at blit time, not the image. Resolve is total: it is defined for
// all 65536 words and never rejects, so an off-corpus word simply names a
// slot the tileset may not hold, which the compositor turns into a reported
// placeholder rather than an error.
func Resolve(word uint16) TileRef {
	g, b, sub := splitTileWord(word)

	water := IsWaterGroup(g)
	if water {
		g = waterGroupBase // phase 0 — animation disabled
	}
	return TileRef{Slot: g*blendColumns + b, Sub: sub, Water: water}
}

// IsImpassable reports the type1 impassable flag (bit 13, 0x2000). An
// impassable non-water cell composites over dirt before blit (TERR-SEM-004).
func IsImpassable(word uint16) bool { return word&ImpassableBit != 0 }

// DirtSubCell returns the dirt.bmp sub-cell that an impassable non-water cell at
// map position (col, row) composites over: (col + row*5) & 3 (TERR-SEM-004).
// The mask keeps the result inside dirt.bmp's four cells for any position.
func DirtSubCell(col, row int) int { return (col + row*5) & 3 }
