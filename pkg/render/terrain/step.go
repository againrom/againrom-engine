package terrain

import "sort"

// The step table and the two edge walks: the pure integer arithmetic a
// projected sloped cell's top and bottom edges are drawn from. Nothing in
// this file reads a map, allocates an image or writes a pixel.
//
// The recipe and both walks are decoded game facts (research TERR-GEOM-034):
// a 14-instruction table builder, the forward and mirrored counts that read
// it, and the T[d][d] == 32 sentinel that terminates a row. The engine's own
// table holds 127 usable rows and defines nothing past them; extending the
// one recipe over every step count the format can produce, rather than
// rejecting or clamping the input, is this project's choice (spec
// Constraints, R-3).
//
// See docs/0012-height-displaced-terrain/spec.md.

const (
	// stepFracBits is the fixed-point shift the recipe accumulates in: 16.16.
	stepFracBits = 16

	// stepSpread is the distance a whole edge is spread over, in 16.16 — the
	// cell's 32 destination columns. The per-step increment is this divided by
	// the number of steps plus one, truncating.
	stepSpread = CellSize << stepFracBits // 0x200000

	// stepSeed is the accumulator's starting value: half a fixed-point unit
	// (0x8000), the recipe's rounding bias.
	stepSeed = 1 << (stepFracBits - 1)

	// maxStepCount is the largest step count an edge can carry, so the largest
	// row the table needs. Both vertices of a cell edge sit in the same map row,
	// so the r*32 term of V(c,r) = r*32 - h(c,r) cancels and the step count is
	// the altitude difference alone; altitudes are signed 8-bit, so the widest
	// difference is 127 - (-128) = 255.
	maxStepCount = 255
)

// stepTable[d] is the row for step count d, for d in 1..maxStepCount; entry
// stepTable[0] is nil, d = 0 being the constant-edge rule that never consults the
// table. Row d holds d+1 entries, so the whole table is about 33 KB of uint8 —
// every value it can hold is in [0, 32], which is why uint8 suffices.
//
// It is built once at initialisation and never written again: it depends on
// no map, so rebuilding it per composite would repeat one map-independent
// loop per image, and deferring it behind a sync.Once would only move the
// same loop to the first draw.
var stepTable = buildStepTable()

func buildStepTable() [][]uint8 {
	table := make([][]uint8, maxStepCount+1)
	for d := 1; d <= maxStepCount; d++ {
		s := stepSpread / (d + 1)
		acc := stepSeed
		row := make([]uint8, d+1)
		for k := 0; k <= d; k++ {
			acc += s
			row[k] = uint8(acc >> stepFracBits)
		}
		table[d] = row
	}
	return table
}

// stepRow returns the table's own row for step count d, or nil when d has no
// row. The nil is deliberate: a step count outside 1..maxStepCount is
// answered rather than indexed blind.
func stepRow(d int) []uint8 {
	if d < 1 || d > maxStepCount {
		return nil
	}
	return stepTable[d]
}

// StepRow returns the step-table row for step count d — the d+1 destination
// columns at which an edge of d steps advances, ending in the CellSize sentinel —
// or nil when d is outside the built range 1..255.
//
// The result is a copy: the table itself is immutable, and handing out a row that
// aliases it would let one caller change every later render's geometry.
func StepRow(d int) []uint8 {
	row := stepRow(d)
	if row == nil {
		return nil
	}
	out := make([]uint8, len(row))
	copy(out, row)
	return out
}

// edgeStep reduces an edge's two vertices to the step count and step direction
// the walks are defined in terms of: d = |yFar - yNear|, dir = sign(yFar - yNear).
func edgeStep(yNear, yFar int) (d, dir int) {
	switch {
	case yFar > yNear:
		return yFar - yNear, 1
	case yFar < yNear:
		return yNear - yFar, -1
	default:
		return 0, 0
	}
}

func forwardSteps(d, i int) int {
	row := stepRow(d)
	if row == nil {
		return 0
	}
	return sort.Search(d, func(k int) bool { return int(row[k]) > i })
}

func mirroredSteps(d, i int) int {
	row := stepRow(d)
	if row == nil {
		return 0
	}
	return d - sort.Search(d, func(k int) bool { return int(row[k]) >= CellSize-i })
}

// EdgeForward returns the destination row a cell edge occupies at
// destination column i when the edge walks forward from its near vertex
// yNear to its far vertex yFar:
//
//	d   = |yFar - yNear|
//	dir = sign(yFar - yNear)
//	EdgeForward(i) = yNear + dir * #{ k in [0,d) : T[d][k] <= i }
//
// Equal vertices leave the edge constant at yNear. The result never passes yFar.
func EdgeForward(yNear, yFar, i int) int {
	d, dir := edgeStep(yNear, yFar)
	return yNear + dir*forwardSteps(d, i)
}

// EdgeMirrored returns the same edge walked mirrored — the row read from
// the far end of the table row rather than the near one:
//
//	EdgeMirrored(i) = yNear + dir * #{ k in [0,d) : 32 - T[d][k] <= i }
//
// The two walks agree at nearly every (d, i), and where they do not, one
// edge of a shared seam lands one row from the other: the disagreement is a
// property of the recipe, and it is drawn as it falls rather than smoothed.
func EdgeMirrored(yNear, yFar, i int) int {
	d, dir := edgeStep(yNear, yFar)
	return yNear + dir*mirroredSteps(d, i)
}
