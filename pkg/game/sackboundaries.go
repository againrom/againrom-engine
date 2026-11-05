package game

import (
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
)

// TOWN-SMOOTH-460 / SPR256-OVL-014: the sparse sibling uses the base palette.
// A missing or mismatched pair leaves the ordinary sack drawable.
func LoadSackBoundaries(src terrain.EntrySource) []*terrain.StaticFrame {
	if src == nil {
		return nil
	}
	c := sheetCache{src: src, decoded: make(map[string]*spr256.Sprite), converted: make(map[string][]*terrain.StaticFrame)}
	return c.boundaryFrames(c.frames(sackSheetPath), sackSheetPath, "backpack/spritesb.256")
}
