package alm

// MaxHeight is the top of the height range the format holds: every installed
// map's height plane lies in 0..127 and the original editor's height tool
// clamps to it (TERR-245). The new-map dialog's slider stops lower, at
// NewMapMaxAltitude.
const MaxHeight = 127

// TileGroupBits is the part of a tile word that names its strip group
// (g = (w & TileGroupBits) >> 6), and the largest value of the type-0 +0x18
// group mask (ALM-SAVE-238).
const TileGroupBits = 0x1fff

// TileGroupMask is the type-0 +0x18 word of a tile plane holding the words
// given: one bit per strip group in use, 0xf00 added when a group above 7 is
// present, and the result bounded at TileGroupBits (ALM-SAVE-238). The new-map
// builder and the editor's brush both write the mask through this function.
func TileGroupMask(words []uint16) uint32 {
	var mask uint32
	for _, w := range words {
		mask |= 1 << ((uint32(w) & tileIndexMask) >> 6)
	}
	if mask > 0xff {
		mask |= 0xf00
	}
	return min(mask, TileGroupBits)
}
