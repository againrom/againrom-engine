package data

// The spell book's icons: one pre-composited strip, a fixed grid over it, and
// the table that says which spell each cell belongs to (`MAGIC-ICON-024`).
//
// AN ICON IS A POSITION, NOT AN INDEXED CELL. The engine blits the atlas WHOLE
// and ONCE, with no source rectangle and no cell index anywhere in the draw;
// what is per-spell is the MASKING — where a spell is not known, a 36x36
// backing tile is pasted over its position. So there is no "icon for spell N"
// to load: there is a place in one picture, and this file is the arithmetic for
// finding it.
//
// TWENTY-FOUR CELLS SERVE TWENTY-EIGHT SPELLS. Four ids are in no slot at all
// and therefore have no icon of their own — a fact about the shipped game, not
// a gap in this decode, and the reason SpellIconSlot answers a bool.

const (
	// SpellIconAtlasPath is the strip every icon is cut from and
	// SpellIconBackPath the tile pasted over a position whose spell is not
	// offered. Both are 24-bit Windows bitmaps under the interface tree; the
	// archive's lookup is case-insensitive, so this spelling follows the
	// claim's rather than asserting how the index writes it.
	SpellIconAtlasPath = "interface/SpellBook.bmp"
	SpellIconBackPath  = "interface/SpellBack.bmp"

	// SpellIconAtlasW and SpellIconAtlasH are the strip's own extent —
	// identical bytes on both roots. They are carried so a consumer can refuse
	// an atlas that is not the one this grid was measured against, rather than
	// cutting cells out of the wrong picture.
	SpellIconAtlasW, SpellIconAtlasH = 480, 85

	// SpellIconSide is one cell's side, which is exactly the backing tile's own
	// size — that identity is what makes the tile "exactly the cell".
	SpellIconSide = 36

	// spellIconOrigin and spellIconPitch are the grid: x = 6 + 38*(i%12),
	// y = 6 + 38*(i/12). The engine computes both with its own shifts and
	// LEA chains; the arithmetic here is the same numbers written plainly.
	spellIconOrigin = 6
	spellIconPitch  = 38

	// SpellIconColumns is where the grid wraps and SpellIconSlots how many
	// positions there are. Twenty-four is an ENGINE limit stated twice — the
	// masking loop's own bound and the length of the slot table below — and
	// lifting it changes no shipped file's bytes but needs both, plus a wider
	// atlas (`UNIT-PICT-038`'s sibling G2 note in `MAGIC-ICON-024`).
	//
	// The grid's bounding box is 6 + 12*38 = 462 <= 480 across and
	// 6 + 2*38 = 82 <= 85 down, i.e. it fits the atlas with the margin the
	// origin implies. TestSpellIconGridFitsItsAtlas holds that.
	SpellIconColumns = 12
	SpellIconSlots   = 24
)

// spellSlotOrder is the ONLY slot-to-spell mapping the engine computes: a
// 24-entry table with exactly one reader, indexed by slot and yielding the
// spell id that slot's cell belongs to (`MAGIC-ICON-024`).
//
// IT IS NOT SORTED AND IT IS NOT A RANGE. Slot 5 is spell 23, slot 7 is 16,
// slot 17 is 25 — the order is the book's own layout and nothing derives it.
// Ids 11, 17, 27 and 28 appear nowhere in it, which is what makes 24 cells
// serve 28 spells.
var spellSlotOrder = [SpellIconSlots]int32{
	1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12,
	6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18,
}

// SpellIconCell is where slot's cell stands in the atlas, in atlas pixels, and
// whether the slot exists at all.
//
// It answers the top-left corner alone; the extent is SpellIconSide on both
// axes for every slot, which is what makes the grid a grid.
func SpellIconCell(slot int) (x, y int, ok bool) {
	if slot < 0 || slot >= SpellIconSlots {
		return 0, 0, false
	}
	return spellIconOrigin + spellIconPitch*(slot%SpellIconColumns),
		spellIconOrigin + spellIconPitch*(slot/SpellIconColumns), true
}

// SpellIconSlot is the slot a spell id occupies, or false for one that occupies
// none.
//
// FOUR SHIPPED SPELLS ANSWER false, and a caller must have something to do with
// that answer other than treat it as a failure: the original draws those four
// no icon either, because there is no cell for them. It is a walk rather than a
// reversed table, because the table is twenty-four entries and a second array
// that had to be kept in step with it is a second thing to get wrong.
func SpellIconSlot(spellID int32) (int, bool) {
	for slot, id := range spellSlotOrder {
		if id == spellID {
			return slot, true
		}
	}
	return 0, false
}
