package mapedit

import (
	"fmt"

	"againrom/pkg/formats/alm"
)

// A mod's tile group is painted in whole cells. Its words name a group past
// the install's (alm.InstallTileGroups), which the transition tables do not
// hold, so a stroke writes each covered cell one of the group's words and no
// corner rule reads or rewrites a cell of such a group.

// groupBrushBase is added to a mod group's number to make its brush value,
// so the value never equals a terrain class.
const groupBrushBase = 0x100

// GroupBrush is the brush value that paints tile group g's whole cells.
func GroupBrush(g int) int { return groupBrushBase + g }

// BrushGroup is the tile group a brush value paints whole cells of; false
// for a terrain class or a value no mod group can have.
func BrushGroup(v int) (int, bool) {
	g := v - groupBrushBase
	return g, g >= alm.InstallTileGroups && g < alm.TileGroupCount
}

// opaqueWord reports whether w names a group the corner rule has no table
// for: such a cell is neither read for its corners nor rewritten.
func opaqueWord(w uint16) bool { return alm.TileGroup(w) >= alm.InstallTileGroups }

// groupStroke is a tile stroke of a mod group: every covered cell takes one
// of b.Words, chosen by the fill (the position pattern, a draw or a variant),
// without bit 13. b.Value names the group, and every word must be of it.
func (e *Editor) groupStroke(f frame, b Brush, at []Cell, w, h int) (edit, error) {
	g, ok := BrushGroup(b.Value)
	if !ok {
		return nil, fmt.Errorf("mapedit: %d is not a mod tile group's brush", b.Value)
	}
	for _, word := range b.Words {
		if alm.TileGroup(word) != g || word&alm.TileImpassable != 0 {
			return nil, fmt.Errorf("mapedit: word %#04x is not a word of tile group %d", word, g)
		}
	}
	span := f.required(alm.TypeTiles)
	old := e.tileWords(span, w*h)
	next := append([]uint16(nil), old...)
	painted := make([]bool, len(old))
	for _, cell := range at {
		b.walk(cell, w, h, func(x, y, _ int) {
			i := y*w + x
			if painted[i] {
				return
			}
			painted[i] = true
			n := len(b.Words)
			k := patternIndex(n, x, y)
			switch b.Fill.Mode {
			case FillVariant:
				k = max(0, min(b.Fill.Variant, n-1))
			case FillRandom:
				if b.Fill.Draw != nil {
					k = max(0, min(b.Fill.Draw(n), n-1))
				}
			}
			next[i] = b.Words[k]
		})
	}
	return e.tileEdit(f, w, old, next, painted), nil
}
