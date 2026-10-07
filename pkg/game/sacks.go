package game

import (
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

const sackSheetPath = "backpack/sprites.256"

// LoadSackFrames decodes the sack sheet's every frame, in the sheet's own
// order, into the render tier's drawable frame type — or answers no frames
// at all for an archive entry that is absent, a stream spr256 refuses, or a
// sheet with no usable palette.
//
// IT RETURNS NO ERROR, on LoadHeroBody's own precedent (heroart.go) and NOT
// LoadStatics's: a missing sheet here is a missing COSMETIC, not a missing
// piece of an install's own content the game cannot run without, so nothing
// here can fail a map or a mission open — the caller has nothing to check
// and nothing to propagate.
//
// It builds a sheetCache of its own, with both maps allocated exactly as
// LoadHeroBody's does, and calls sheetCache.frames — the one function that
// already converts a whole sheet in order and already answers nil for every
// exclusion this contract names (an absent entry, a stream the decoder
// refuses, a palette-less sheet), so nothing here reimplements that rule.
func LoadSackFrames(src terrain.EntrySource) []*terrain.StaticFrame {
	if src == nil {
		return nil
	}
	sheets := sheetCache{
		src:       src,
		decoded:   make(map[string]*spr256.Sprite),
		converted: make(map[string][]*terrain.StaticFrame),
	}
	return sheets.frames(sackSheetPath)
}

// sackFrameCount is the sack sheet's own frame population and the ladder's
// ceiling. ROM1 clamps the computed index to a signed maximum of 5
// (ITEM-136), and `backpack/sprites.256` holds exactly six 32x32 frames, so
// the clamp and the sheet agree. This build does not read the sheet to find
// the bound: a sheet that decoded fewer frames would be excluded by the
// window tier's own range check (pkg/ui/statics.go), not by silently picking
// a different rule here.
const sackFrameCount = 6

// sackValue is the sack's own Token value slot, `Sack+0x1c`: gold plus the
// sum of each contained item's own value, recomputed rather than stored
// (ITEM-SACK-010). Not weight, and not a contained-item count — ITEM-137
// read the producing routine whole and neither is touched anywhere in it.
//
// AN ITEM THIS BUILD CANNOT PRICE CONTRIBUTES ZERO, which is the same answer
// the shop's own valuation gives it (shoproom.go, SHOP-SELL-010's
// `elem+0x1c != 0` test). That is a pre-existing gap in what this tree
// decodes, not a rule about sacks, and it can only push a sack's frame DOWN
// a band, never up.
//
// A world's own sack is valued by sim.World.SackValue, over the item records
// its container writes: a stack counts its unit price once (SHOP-PRICE-011),
// not once per unit. This record-free form treats each instance as its own
// record.
func sackValue(sack sim.Sack) int64 {
	value := int64(sack.Gold)
	for _, item := range sack.ItemInstances {
		value += int64(item.Price)
	}
	return value
}

// sackFrameIndex answers which of the sack sheet's six frames a ground sack
// draws.
//
// IT IS NOW DECODED, NOT AUTHORED. ROM1's own rule is a ladder at the powers
// of ten over the sack's own value: the generic notify routine writes
// `_ftol(log10(Sack+0x1c))` into the message byte (ITEM-137), the client
// dispatcher's opcode-0x7a case stores that byte to `CBackPack+0x20` and
// clamps it to a signed maximum of 5 (ITEM-136), and the sheet's painter
// pushes that same field in the frame-argument position at both of its draw
// calls (SPR256-077). Six bands:
//
//	[1, 10)         -> 0
//	[10, 100)       -> 1
//	[100, 1000)     -> 2
//	[1000, 10000)   -> 3
//	[10000, 100000) -> 4
//	>= 100000       -> 5
//
// The mapping ASCENDS — log10 is monotonic over the positive domain and
// nothing between the `fyl2x` result and the clamped store flips it, so a
// richer sack never draws a lower index (ITEM-138).
//
// THE BANDS ARE WALKED ON INTEGERS, not through math.Log10. ROM1 reaches
// them through floating point, but its thresholds are exact powers of ten
// and a binary float cannot represent every one of them exactly; multiplying
// an int64 bound by ten hits each threshold on the nose, so a sack worth
// exactly 1000 lands in band 3 here with no rounding to argue about.
//
// VALUE 0 AND BELOW ARE NOT DECODED, and this build answers frame 0 there as
// an AUTHORED choice (DIV-1358). ROM1 hands the value to `log10`, whose
// zero-input branch discards the operand and returns the extended-precision
// negative-infinity pattern with error code 2; its negative branch returns a
// quiet NaN with error code 1, and that branch is reachable at the
// instruction level because the load is an integer load of a SIGNED
// 32-bit value. What `_ftol` then makes of either pattern was not traced
// (formats/item/sacks.md). Frame 0 is the band immediately above and the
// frame this whole population drew before this rule landed, so the undecoded
// corner changes nothing that was already right.
func sackFrameIndex(sack sim.Sack) int {
	return sackFrameForValue(sackValue(sack))
}

// sackFrameForValue is sackFrameIndex's ladder over a value already summed.
func sackFrameForValue(value int64) int {
	if value < 1 {
		return 0
	}
	index := 0
	for bound := int64(10); index < sackFrameCount-1 && value >= bound; bound *= 10 {
		index++
	}
	return index
}
