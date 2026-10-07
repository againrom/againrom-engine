package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// The two bits a derived grid byte is allowed to set, spelled at the tier
// that WRITES one: bit 0 blocks a ground mover, bit 1 an air mover, and bits
// 2-7 are reserved — refused by the world constructor rather than masked
// away, so a grid that set one would build no world at all.
//
// They are named here rather than reached for in pkg/sim because that package
// keeps them unexported, and a bare 1 and 2 at the write site would leave the two
// bit meanings written down nowhere on this side of the seam.
const (
	blockGround byte = 1 << 0
	blockAir    byte = 1 << 1
)

// borderDepth is how deep the impassable ring round a map runs, in cells: a
// cell is a border cell when it lies within this many of any edge.
//
// It is also why a map of twice this or fewer cells on an axis is border
// throughout — the two sides of that axis meet — and so why a fixture a test
// walks a map-built world over has to be as large as it is.
const borderDepth = 8

const (
	waterLo = 512
	waterHi = 768
)

// blendMinimum is the level at or above which the strip group's PRIMARY
// terrain is taken; below it the secondary is. Mountain is the primary of
// group 7 and the secondary of none, which is why the block plane's mountain
// arm and the classifier's Mountain agree everywhere.
const blendMinimum = 3

var blendLevel = [4][14]uint8{
	{2, 3, 2, 4, 3, 4, 2, 2, 2, 2, 4, 4, 4, 4},
	{3, 5, 3, 3, 1, 3, 2, 4, 2, 2, 4, 2, 4, 4},
	{2, 3, 2, 4, 3, 4, 2, 4, 2, 2, 4, 2, 4, 4},
	{5, 5, 5, 5, 5, 5, 2, 2, 2, 2, 4, 4, 4, 4},
}

// The ten terrain classes, numbered as the classifier numbers them, and the
// REJECT that is not one of them.
//
// classReject is a value and not an error: the classifier is total, and both of
// its reject arms — a water word whose low nibble is 8 or more, and any word
// whose sub-cell is 14 or more — are arms no shipped cell reaches. So is the
// third, strip groups 13 to 15, which is ours: those name no terrain pair at all
// and the original reads uninitialised memory for them (0076 spec, Divergences).
const (
	classLand uint8 = iota + 1
	classGrass
	classFlowers
	classSand
	classCracked
	classStones
	classSavanna
	classMountain
	classWater
	classRoad

	classReject uint8 = 0xff
)

// classCost is each class's movement cost scalar, indexed by the class itself so
// that no second ordering exists to disagree with the numbering above. Entry 0
// is the reject's, and it is the value the ingest writes for any class of 13 or
// more.
//
// These are the shipped values used by the native fresh-map path. Original
// LOAD uses data.OriginalTerrainCostTable and classifyCosts with installed parameters.
var classCost = [11]uint8{
	classReject,
	classLand:     8,
	classGrass:    8,
	classFlowers:  8,
	classSand:     14,
	classCracked:  6,
	classStones:   12,
	classSavanna:  8,
	classMountain: 16,
	classWater:    8,
	classRoad:     6,
}

// stripPair is each strip group's (primary, secondary) terrain pair. Groups 8 to
// 11 are the water range and are left at the early-out before this is read;
// groups 13 to 15 name no pair and are a reject, which is the one place this
// derivation departs from the original and is disclosed as such.
var stripPair = [16][2]uint8{
	0:  {classGrass, classLand},
	1:  {classCracked, classLand},
	2:  {classSand, classLand},
	3:  {classSavanna, classLand},
	4:  {classStones, classLand},
	5:  {classCracked, classStones},
	6:  {classFlowers, classSavanna},
	7:  {classMountain, classStones},
	12: {classRoad, classLand},
}

// waterCost is what every cell of the water arm costs: the classifier's entry
// literal, taken whatever the class scalars say. It is 8 and so is CostWater, so
// no shipped map can tell the two readings apart — only the instruction can, and
// it never reaches the table.
const waterCost = 8

// classify is THE classifier: one tile word to a terrain class and a
// movement cost.
//
// It is one function with two consumers — the cost plane is its cost and the
// block plane's mountain arm is its class — because a second reading of the same
// tile-word split beside the first is exactly the failure cellArms' own note
// names. What is NOT folded into it is the block plane's water arm: that test is
// taken on the RAW tile index ahead of any classification, and the two select
// the same set by algebra rather than by construction, so folding one into the
// other would assert an identity the decode does not.
func classify(word uint16) (class, cost uint8) {
	return classifyCosts(word, classCost)
}

func classifyCosts(word uint16, costs [11]uint8) (class, cost uint8) {
	i := int(alm.TileIndex(word))

	// Water first, and taken whole. Its cost is the entry literal and the water
	// scalar is never consulted.
	if i&0x300 == 0x200 {
		switch {
		case i&0xf >= 8:
			return classReject, costs[0]
		case i&0xf == 4 && i&0x30 == 0x10:
			return classLand, waterCost
		default:
			return classWater, waterCost
		}
	}

	s := i & 0xf
	b := (i >> 4) & 3
	g := (i >> 6) & 0xf
	if s >= 14 {
		return classReject, costs[0]
	}
	pri, sec := stripPair[g][0], stripPair[g][1]
	if pri == 0 {
		// Strip groups 13 to 15, which the original never wrote a pair for.
		return classReject, costs[0]
	}

	cp, cs := uint16(costs[pri]), uint16(costs[sec])
	switch blendLevel[b][s] {
	case 1:
		return sec, uint8(cs)
	case 2:
		return sec, uint8((3*cs + cp) >> 2)
	case 3:
		return pri, uint8((cs + cp) >> 1)
	case 4:
		return pri, uint8((3*cp + cs) >> 2)
	default:
		return pri, uint8(cp)
	}
}

// arms is which of the five block arms hold for one cell, EACH AS IT STANDS WITH
// THE OTHER FOUR ABSENT.
//
// The five are a union and no ordering among them is asserted, so the grid
// byte needs none of this detail — a single bool would carry it. The
// detail exists because the census does: a union plane cannot say which arm
// blocked a cell, so an instrument reporting per-arm counts would otherwise
// have to ask a second copy of the rule and would then be measuring that
// copy.
type arms struct {
	impassable bool //
	water      bool //
	mountain   bool //
	scenery    bool //
	border     bool //
}

// block is the grid byte those arms make: bit 0 set when any of the five
// blocks ground, bit 1 set when the border covers the cell and by nothing
// else, and every other bit clear.
//
// Written as one OR of five terms there is no assignment order to get wrong, and
// so no precedence for a criterion to have to pin.
func (a arms) block() byte {
	var c byte
	if a.impassable || a.water || a.mountain || a.scenery || a.border {
		c |= blockGround
	}
	if a.border {
		c |= blockAir
	}
	return c
}

// cellArms is THE classifier, and the only one: both exported entry points below
// walk their cells through this function, so the census counts the rule the
// plane is built from rather than a second reading of it written from the first.
//
// The word is split by this tier's own shifts and not by the render tier's. The
// two partitions are different functions above bit 9 — that one keeps bits 10-12
// in its group — and the DAG forbids the import in any case, so a shared split
// would be a third reading that is neither tier's.
//
// x, y, w and h are compared in int, so the border test on a map narrower than
// the ring underflows nothing and the whole-map case falls out of the arithmetic
// rather than being special-cased.
func cellArms(word uint16, overlay uint8, x, y, w, h int) arms {
	i := int(alm.TileIndex(word))
	class, _ := classify(word)
	return arms{
		impassable: alm.Impassable(word),
		water:      i >= waterLo && i < waterHi,
		mountain:   class == classMountain,
		scenery:    overlay != 0,
		border:     inBorderRing(x, y, w, h),
	}
}

// extent is how many cells the derivation describes, and it is the world's own
// answer rather than a second one: FromALM hands sim.NewWorld int32(m.Width) and
// int32(m.Height), so the plane is sized from exactly that narrowing and the two
// cannot come to disagree about a decoded extent that does not fit an int32.
//
// A nil map is an extent of nothing, as is one whose narrowed width or height is
// not positive — which is gridCells' own rule, restated in this one place.
func extent(m *alm.Map) (w, h int, ok bool) {
	if m == nil {
		return 0, 0, false
	}
	nw, nh := int32(m.Width), int32(m.Height)
	if nw <= 0 || nh <= 0 {
		return 0, 0, false
	}
	return int(nw), int(nh), true
}

// planeAt reads index i of a plane that may be shorter than the extent, and
// answers zero past its end.
//
// That is what makes the derivation total: a decoded map always carries W*H of
// each plane, so this arm is reached only by a map built by hand, and answering
// zero there is the loader declining to fail on one. A plane LONGER than the
// extent is never indexed past it, because the walk is over the extent.
func planeAt[T uint16 | uint8](plane []T, i int) T {
	if i < len(plane) {
		return plane[i]
	}
	var zero T
	return zero
}

// Passability is the block plane a decoded map describes: exactly one byte
// per in-bounds cell, row-major from (0,0), and no bytes at all for a map
// with no cells.
//
// It is a function of the extent, the tile plane and the overlay plane ALONE, so
// two maps agreeing on those three yield equal planes however else they differ.
// It is integer-only, reads no clock, file or generator, and cannot fail: a short
// plane reads as zero past its end, a long one is ignored past the extent, and no
// map produces an error, a panic or a plane of any other length.
//
// It is exported because a world has no grid reader: this is the only way a
// caller can see what a map's cells came out as, and the census below is the only
// way to see which arm made them that.
func Passability(m *alm.Map) []byte {
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			out[i] = cellArms(planeAt(m.Tiles, i), planeAt(m.Overlay, i), x, y, w, h).block()
		}
	}
	return out
}

// Counts is the census of one derived plane: how many cells hold each byte
// value, how many block a ground and how many an air mover, and how many each of
// the five arms would block WITH THE OTHER FOUR ABSENT.
//
// The five arm counts overlap and do not sum to either blocked count — a border
// cell carrying a water word is counted by both arms and is one blocked cell —
// which is the whole reason they are reported separately.
type Counts struct {
	Value [4]int

	// Ground and Air are how many cells block a ground and an air mover.
	Ground, Air int

	// The five arms, each counted as it stands alone.
	Impassable, Water, Mountain, Scenery, Border int
}

// Census counts one decoded map's derived plane, arm by arm. It walks the same
// cells through the same classifier Passability does, so what it reports is the
// production rule and not a copy of it.
//
// A map with no cells censuses to the zero value, which is the honest reading:
// every count over no cells is zero.
func Census(m *alm.Map) Counts {
	var c Counts
	w, h, ok := extent(m)
	if !ok {
		return c
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			a := cellArms(planeAt(m.Tiles, i), planeAt(m.Overlay, i), x, y, w, h)
			b := a.block()

			c.Value[b]++
			if b&blockGround != 0 {
				c.Ground++
			}
			if b&blockAir != 0 {
				c.Air++
			}
			if a.impassable {
				c.Impassable++
			}
			if a.water {
				c.Water++
			}
			if a.mountain {
				c.Mountain++
			}
			if a.scenery {
				c.Scenery++
			}
			if a.border {
				c.Border++
			}
		}
	}
	return c
}

// Cost is the movement-cost plane a decoded map describes: exactly one byte
// per in-bounds cell, row-major from (0,0), and no bytes at all for a map
// with no cells.
//
// It is a function of the extent and the TILE plane alone — not the overlay, not
// the altitudes, not the records and not the structures — so two maps agreeing
// on those two yield equal planes however else they differ. It is integer-only,
// reads no clock, file or generator, and cannot fail: a short tile plane reads as
// the word zero past its end, a long one is ignored past the extent, and no map
// produces an error, a panic or a plane of any other length.
//
// A structure does NOT reach it. A placed building rebuilds the block bytes of
// the cells it covers and puts their cost baseline back unchanged, so there is
// no second stage here for a table to feed — which is why this takes a map where
// the block plane's entry point takes a map and a table.
func Cost(m *alm.Map) []byte {
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]byte, w*h)
	for i := range out {
		_, c := classify(planeAt(m.Tiles, i))
		out[i] = c
	}
	return out
}

// Height is the altitude plane a decoded map describes: the map's own
// altitudes, byte for byte and cell for cell, unscaled and unshifted.
//
// It is a COPY at the extent's own length rather than the decoded slice itself,
// which is what makes it total on the same terms Cost is — a map built by hand
// with a short altitude plane reads as zero past its end, and one with a long
// plane is ignored past the extent. It also keeps the map's slice the map's: the
// world copies again on the way in, but a plane handed out of here is nobody
// else's to mutate.
//
// There is no derivation to speak of and that is the finding rather than a gap:
// the ingest's height arm is a per-cell copy, so anything cleverer here would be
// an invention.
func Height(m *alm.Map) []byte {
	w, h, ok := extent(m)
	if !ok {
		return nil
	}
	out := make([]byte, w*h)
	for i := range out {
		out[i] = planeAt(m.Altitudes, i)
	}
	return out
}

// Planes is the three per-cell planes a world over m stands on, with m's
// placed structures applied to the block plane when t names them.
//
// It exists so that every world-building path takes all three from ONE place.
// Two of those paths rebuild a world from an earlier one, and a path assembling
// its own bundle is how a rebuilt world comes to stand on different ground from
// the world it was rebuilt from.
func Planes(m *alm.Map, t *Table) sim.Terrain {
	return sim.Terrain{Block: PassabilityWith(m, t), Cost: Cost(m), Height: Height(m)}
}
