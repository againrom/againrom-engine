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
	base, boundary := c.sheet(sackSheetPath), c.sheet("backpack/spritesb.256")
	if base == nil || boundary == nil || !base.HasPalette || len(base.Frames) != len(boundary.Frames) {
		return nil
	}
	copy := *boundary
	copy.Palette, copy.HasPalette = base.Palette, true
	c.decoded["backpack/spritesb.256"] = &copy
	return c.frames("backpack/spritesb.256")
}
