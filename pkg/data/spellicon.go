package data

import "againrom/pkg/rules"

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

// SpellIconSlot is a spell's first-game atlas slot (`MAGIC-ICON-024`).
func SpellIconSlot(spellID int32) (int, bool) {
	if s, ok := rules.GameSpellOf(rules.SpellGameFirst, int(spellID)); ok && s.Slot != 0 {
		return int(s.Slot) - 1, true
	}
	return 0, false
}
