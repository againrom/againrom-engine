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

// The strip groups by where their pictures come from: the installs ship
// groups 0 to InstallTileGroups-1 (ALM-TILE-336), and the tile loader holds
// TileGroupCount groups, one +0x18 mask bit each (TERR-TILE-291,
// TERR-TILE-292); the groups between are a mod's.
const (
	InstallTileGroups = 13
	TileGroupCount    = 32
)

// TileGroup is the strip group tile word w names (TERR-TILE-291).
func TileGroup(w uint16) int { return int(w&TileGroupBits) >> 6 }

// TileGroupMask is the type-0 +0x18 word of a tile plane holding the words
// given. The install groups take one bit each, 0xf00 added when a group above
// 7 is present, bounded at TileGroupBits (ALM-SAVE-238); each mod group in use
// adds its own bit above that bound, which the original editor's cap would
// drop (DIV-4412). A group past the loader's names no bit. The new-map
// builder and the editor's brush both write the mask through this function.
func TileGroupMask(words []uint16) uint32 {
	var install, mods uint32
	for _, w := range words {
		switch g := TileGroup(w); {
		case g < InstallTileGroups:
			install |= 1 << g
		case g < TileGroupCount:
			mods |= 1 << g
		}
	}
	if install > 0xff {
		install |= 0xf00
	}
	return min(install, TileGroupBits) | mods
}
