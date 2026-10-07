package sim

// THE SIGHT PREDICATE: what a unit can see, and why it is not a circle.
//
// The region a unit's sight admits is NOT a radius and NOT a shape. It is a
// BUDGET, spent walking outward one cell at a time, and the term that dominates
// it is the observer's own altitude. A unit on a hill sees an order of magnitude
// further than the same unit in a ditch: over the shipped corpus at a scan range
// of 6 the visible count runs 29 at the worst position and 1 248 at the best,
// against 145 on flat ground (AI-LOS-089). Nothing about that is reproducible by
// scaling a disk.
//
// The walk carries a running remainder from a cell to the next cell OUTWARD,
// and the step it walks along is fixed in advance by two tables over a 41x41
// window: where each offset's predecessor is, and what the step to it costs.
// Both tables are built ONCE, from geometry and one shift parameter, and
// depend on the map for nothing at all — the routine that fills them runs
// from the sight object's init inside the world constructor, BEFORE the map
// is loaded (AI-LOS-087). So they are a property of the build and not of a
// world, which is why they are package-level values here and not a field on
// anything.
//
// The window is 41x41 and the ring walk stops at radius 19, so the window's own
// outermost ring is unreachable in principle (AI-LOS-089).

// sightShift is `k`, the fixed-point precision of the whole predicate: every
// budget, every step cost and every accumulator value is in units of 1/(1<<k) of
// a cell.
//
// It is a FILE-CARRIED customisation limit, and one with a trap in it
// (TERR-FOG-088). The law reads it from `[Scanning] ScanShift` in
// `World\Data\map.reg`, which ships as 7 in both roots, against a compiled
// default of the same 7 — so the two agree today only because the shipped value
// equals the code's. Editing that key moves THIS predicate's precision and
// leaves the drawn fog untouched, because the fog is a second implementation of
// the same algorithm with its own copy of the constant.
//
// This tree reads no registry, so it is written here. The seam is this constant.
const sightShift = 7

// sightStep is 1<<sightShift: what one axis step costs, and the unit the whole
// table is scaled in.
const sightStep = 1 << sightShift

// sightHalf is the half-width of the window the tables cover, and sightSpan its
// full width in cells.
const (
	sightHalf = 20
	sightSpan = 2*sightHalf + 1
	sightArea = sightSpan * sightSpan
)

// sightRings is the last ring the march ever walks.
//
// It is 19 and not sightHalf, and that difference is the law's own: the ring
// loop's bound is a compare against 20 taken as a strict less-than, so ring 20 —
// the window's outer edge — is never visited however much budget is left
// (AI-LOS-089). The tables still cover it, because the builder writes the whole
// window; nothing ever reads those cells.
const sightRings = sightHalf - 1

// sightOffset indexes the window tables by a cell offset from the observer.
//
// dx is the COLUMN delta and dy the ROW delta, and which is which is not a
// convention picked here: the law's grid is addressed with the column on the
// 0x80 stride and the row on the 2 stride, which is what makes the builder's two
// literal fix-up addresses land on the cells (+1,0) and (-1,0) that AI-LOS-087
// names them as. Getting it the other way round transposes the whole region
// wherever the two zone tests are asymmetric, which is everywhere off the
// diagonal.
func sightOffset(dx, dy int32) int {
	return int(dy+sightHalf)*sightSpan + int(dx+sightHalf)
}

// sightWindowTables is the pair of tables the march walks: for every offset in
// the window, the one step back toward the observer, and what that step costs.
//
// stepX/stepY are the law's step grid — a signed byte pair per offset, each in
// {-1,0,+1} (AI-LOS-088). cost is its cost grid, one signed 16-bit value per
// offset. Both are stored as one flat array per component rather than as a
// struct per cell, because the march reads the two at different moments and a
// struct would put the cost in cache on the many cells whose predecessor is
// blocked.
type sightWindowTables struct {
	stepX [sightArea]int8
	stepY [sightArea]int8
	cost  [sightArea]int32
}

// sightTables is the built pair, at sightShift. It is a package-level value
// and not a field of a world, because it depends on nothing a world carries:
// were it a field it would be hashed state no world could vary, and the
// field-set pin exists to refuse exactly that.
var sightTables = buildSightTables(sightShift)

// buildSightTables fills the window from geometry and shift alone — the
// SAME shift the seed and the cost table read their units in, which is what
// lets one march serve two readers: shift is a builder argument rather than
// a literal inside it, so a caller building at a different precision gets
// tables that agree with that precision's own seed.
//
// The law's builder walks i = 0..20 by j = 0..20 and stores EIGHT bytes per
// iteration — the two bytes of one step word in each of four quadrant mirrors —
// which is the shape reproduced here (AI-LOS-087). The four mirrors are written
// in the order (+i,+j), (-i,+j), (+i,-j), (-i,-j), and that order is not
// cosmetic: on the j == 0 axis the +j and -j addresses COINCIDE, so the last
// store wins, and the cell (1,0) — the one axis cell whose slope falls in the
// diagonal zone — is left pointing diagonally at a neighbour no closer to the
// centre than itself. Its chain would never terminate.
//
// That is what the four trailing literal stores are for. The law ends the
// builder by writing (-1,0) into the cell (+1,0) and (+1,0) into the cell
// (-1,0), and re-executing the builder without them leaves EXACTLY TWO offsets
// whose predecessor is not closer — those two. They are reproduced at the same
// point, after the loop, so that a reader can see the same two-cell repair the
// original needed rather than a rule that quietly never had the problem.
func buildSightTables(shift int32) *sightWindowTables {
	t := new(sightWindowTables)
	for i := int32(0); i <= sightHalf; i++ {
		for j := int32(0); j <= sightHalf; j++ {
			sx, sy := sightZoneStep(i, j)
			c := sightStepCost(i, j, shift)
			for _, q := range [4][2]int32{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
				at := sightOffset(q[0]*i, q[1]*j)
				t.stepX[at], t.stepY[at] = int8(sx*q[0]), int8(sy*q[1])
				t.cost[at] = c
			}
		}
	}
	t.stepX[sightOffset(1, 0)], t.stepY[sightOffset(1, 0)] = -1, 0
	t.stepX[sightOffset(-1, 0)], t.stepY[sightOffset(-1, 0)] = 1, 0
	return t
}

// sightZoneStep is the step back toward the centre for the first-quadrant offset
// (i, j), before mirroring: a pure column step where the ray is shallow, a pure
// row step where it is steep, and a diagonal between them.
//
// The two comparisons are the law's own — `j < i>>1` and `j > 2i` — and the
// halving is an arithmetic shift, so the shallow zone's boundary is not the
// mirror image of the steep zone's. That asymmetry is why (1,0) falls in the
// diagonal band and (0,1) does not.
func sightZoneStep(i, j int32) (int32, int32) {
	switch {
	case j < i>>1:
		return -1, 0
	case j > 2*i:
		return 0, -1
	default:
		return -1, -1
	}
}

// sightStepCost is what one step along the ray through (i, j) costs, in
// units of 1/(1<<shift) cell: the ray's Euclidean length divided by the
// number of Bresenham steps it takes, which is max(i, j) — at whatever
// shift the table being built is precise to, not only sightShift.
//
// So an axis ray's step costs exactly 1<<shift and a diagonal ray's costs
// 1<<shift times sqrt(2) — 128 and 181 at the shipped shift, and nothing outside
// that band anywhere in the window (AI-LOS-088). The centre's cost is a division
// by zero in the law and is never read; it is zero here for the same reason.
//
// IT IS COMPUTED IN INTEGERS, and the law's is not: the published expression
// is an FPU sequence ending in a truncating conversion, and this package is
// held to no floating-point value at all by a source scan. The transcription
// is exact rather than approximate, because trunc(A*sqrt(s)/m) ==
// isqrt(A*A*s)/m for non-negative integers under integer division —
// floor(floor(a)/m) is floor(a/m) for a positive integer m. The two forms
// are compared at every one of the window's offsets by this package's own
// test, not argued for here.
func sightStepCost(i, j, shift int32) int32 {
	m := i
	if j > m {
		m = j
	}
	if m == 0 {
		return 0
	}
	step := int32(1) << shift
	return isqrt(step*step*(i*i+j*j)) / m
}

// isqrt is the integer square root of n: the largest v with v*v <= n, and 0 for
// a negative n.
//
// Newton's iteration on integers, which terminates because the sequence is
// strictly decreasing once it is above the root and the guard stops it the first
// time it is not. It is here rather than in a shared place because it is the
// only square root this package has ever needed, and pkg/sim may not reach for
// one that returns a float.
func isqrt(n int32) int32 {
	if n < 1 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// sightSeed is what an observer starts the march with, in units of
// 1/(1<<shift) cell: sight256 — a sight radius in units of 1/256 cell —
// shifted into the shift's own units, plus a half-cell rounding term. It
// replaces sightBudget: both readers call it, and neither names a radius of
// its own.
//
// A whole-cell radius r arrives as r<<8 — the AI reader's own call — and
// at every shift 1..8 that reproduces exactly the old whole-cell form,
// sightStep>>1 + r<<sightShift, because (r<<8)>>(8-shift) == r<<shift:
// shifting left by 8 and right by 8-shift is a net left shift of shift. That
// identity is asserted over the whole parameter square by this package's own
// test, not assumed (AC-4).
//
// The law writes it into the accumulator's own centre cell rather than into a
// field, and the address it writes to was published as a field for two
// experiments before it was recognised as `acc[20][20]` (AI-SIGHT-006 as
// amended). What it means is arithmetic on the table: the cheapest step in the
// window costs exactly 1<<shift, so a seed of `range` cells plus a half
// survives exactly `range` axis steps and no more.
func sightSeed(sight256, shift int32) int32 {
	return 1<<(shift-1) + sight256>>(8-shift)
}

// sightReader is what makes one march into two readers rather than two
// marches: the window tables, the shift they were built at — the same
// units the seed and the cost table read in — and which in-bounds
// rectangle a ring cell must lie inside to be walked at all. aiSight and
// fogSight are the only two values that exist; marchSight, marchCell and
// groupSight all take one. A second march function for the fog reader is
// exactly what the decoded law says a consumer must not do, and it is what
// produced two implementations that had to be proved equal after the fact.
type sightReader struct {
	tables *sightWindowTables
	shift  int32
	inset  func(w *World, i int, c cell) bool
}

// aiSight is the AI reader: the group-acquisition sweep's own march
// (candidates, groupSight's default caller), its answers unchanged by this
// story except through the seed identity sightSeed asserts.
var aiSight = sightReader{tables: sightTables, shift: sightShift, inset: aiInset}

// fogSight is the fog reader this story adds: the same tables and the same
// shift as aiSight — nothing here varies either — differing only in the
// rectangle a ring cell is tested against.
var fogSight = sightReader{tables: sightTables, shift: sightShift, inset: fogInset}

// aiInset is the AI reader's rectangle, read off the grid rather than
// computed from the bounds. The law tests each ring cell against a rectangle
// inset eight cells from every edge, written at map load; every map this
// tree loads marks that same region — and nothing else — with the bit
// that closes a cell to an air mover, so the two are the same set of cells
// wherever a map produced the world.
func aiInset(w *World, i int, c cell) bool {
	return w.grid[i]&blockAir == 0
}

// fogInset is the fog reader's rectangle: computed from the world's own
// bounds rather than read off a bit, because the fog's inset — 7 cells
// rather than the AI's 8 — has no bit marking it. A cell is admitted when
// its absolute column lies in 7..W-8 and its row in 7..H-8.
func fogInset(w *World, i int, c cell) bool {
	return c.x >= 7 && c.x <= w.bounds.Width-8 && c.y >= 7 && c.y <= w.bounds.Height-8
}

// marchSight walks one observer's sight into stamp, one byte per world cell,
// through reader — the window tables, the shift and the inset that
// together make it either the AI reader or the fog reader — and is the
// whole predicate.
//
// THE RECURRENCE IS FOUR TERMS AND THE ORDER OF THEM IS THE LAW'S
// (AI-LOS-081): the predecessor's remaining budget, minus this step's cost,
// minus this cell's height, plus the observer's own. A cell is visible while
// that is positive. Unrolled along a ray it says why the region is what it is —
// the observer's altitude is added at EVERY step while a cell's own height is
// charged ONCE, when the ray passes through it, so reach scales with where the
// unit is standing and barely with what is in the way (AI-LOS-089).
//
// A BLOCKING CELL IS NOT PRUNED. The law stores the failing value into that
// cell's own slot BEFORE it branches, and its caller consumes the blocked answer
// only to decide whether the whole ring was blocked; the ring runs to its end
// either way, and a later cell whose predecessor is the blocked one continues
// from the stored negative (AI-LOS-090). That is reproduced exactly, and it is a
// named seam rather than an implementation detail — see sightRelight.
//
// The height plane is read SIGNED, which is the law's own MOVSX and is the other
// half of that seam.
func (w *World) marchSight(reader sightReader, stamp []byte, from cell, scanRange int32) {
	var acc [sightArea]int32
	acc[sightOffset(0, 0)] = sightSeed(scanRange<<8, reader.shift)

	// The observer's own cell is marked before any ring and outside the inset
	// test: the law's walk marks it directly, and a unit that could not see the
	// cell it is standing in would not be found by its own group's sweep.
	if i, ok := w.cellIndex(from.x, from.y); ok {
		stamp[i] = 1
	}
	observer := int32(int8(w.heightAt(from)))

	for r := int32(1); r <= sightRings; r++ {
		lit := false
		for _, d := range sightRing(r) {
			if w.marchCell(reader, stamp, &acc, from, d[0], d[1], observer) {
				lit = true
			}
		}
		// The ring stops the walk only when NOTHING in it was visible. An
		// unvisited cell clears nothing, so a ring entirely outside the world or
		// entirely inside the border ends the march exactly as a ring of
		// mountains does.
		if !lit {
			return
		}
	}
}

// marchCell evaluates one window offset against reader's tables and inset,
// and reports whether it came out visible.
//
// The two ways not to be visited are here rather than at the caller because they
// share their consequence: nothing is written, so the offset's accumulator slot
// keeps the zero it was made with, and any later cell naming it as a predecessor
// marches on from zero — which, the cheapest step costing 1<<shift, is always
// blocked. That is what makes an unvisited cell a wall rather than a hole.
func (w *World) marchCell(reader sightReader, stamp []byte, acc *[sightArea]int32, from cell, dx, dy, observer int32) bool {
	c := cell{x: from.x + dx, y: from.y + dy}
	i, ok := w.cellIndex(c.x, c.y)
	if !ok {
		return false
	}
	// THE INSET: which rectangle a ring cell must lie inside to be walked at
	// all, and the one place the two readers differ in what they test —
	// aiInset and fogInset carry their own reasoning.
	if !reader.inset(w, i, c) {
		return false
	}
	at := sightOffset(dx, dy)
	pred := sightOffset(dx+int32(reader.tables.stepX[at]), dy+int32(reader.tables.stepY[at]))
	v := acc[pred] - reader.tables.cost[at] - int32(int8(w.height[i])) + observer
	acc[at] = v
	if v <= 0 {
		return false
	}
	stamp[i] = 1
	return true
}

// sightRing is the Chebyshev ring of radius r, as offsets, in the law's own
// four-edge order: the two full rows, then the two columns between them.
//
// The ORDER cannot change the result, and that is worth stating rather than
// relying on: every offset's predecessor is strictly closer to the centre
// (AI-LOS-088), so it lies in a ring already finished, and no cell of a ring
// ever reads another cell of the same ring.
func sightRing(r int32) [][2]int32 {
	out := make([][2]int32, 0, 8*r)
	for dx := -r; dx <= r; dx++ {
		out = append(out, [2]int32{dx, -r}, [2]int32{dx, r})
	}
	for dy := -r + 1; dy <= r-1; dy++ {
		out = append(out, [2]int32{-r, dy}, [2]int32{r, dy})
	}
	return out
}

// sightRelight is the margin between the cheapest step and the largest rise the
// recurrence can be handed, and it is THE CUSTOMISATION LIMIT of this file.
//
// Because a blocker is not pruned, whether the region is star-shaped is
// arithmetic and not a property of the code: a cell can rise above its
// predecessor only when `height(observer) - height(cell)` reaches the cheapest
// step cost, 1<<k = 128. The height plane is read signed, so the largest rise
// available in principle is 127 - (-128) = 255. The largest available on any
// SHIPPED map is 127, because every altitude byte of all 72 of them is 0..127,
// with none at 0x80 or above (AI-LOS-090) — one short.
//
// So an engine that pruned at the first blocker would be indistinguishable from
// this one on every shipped map, and would diverge on the first map that authors
// a single altitude byte of 128 or more. The limit's class is a CORPUS FACT with
// a one-unit margin, not a code property; lifting it changes no shipped file at
// all, only what a map is allowed to say. This build takes the faithful walk, so
// the limit is a measurement its own tests take rather than an assumption baked
// into the loop.
const sightRelight = sightStep

// groupSight is the union of one march per member at reader: the stamp the
// candidate sweep then reads.
//
// A GROUP SEES AS ONE ANIMAL (AI-GROUPSEE-068). The law clears one shared
// byte map, stamps every member's sight into it and sweeps the whole actor
// list against it, so a candidate visible to any member is a candidate for
// every member; the union here is that same set. It is rebuilt from nothing
// on every decision and stored nowhere — the law's array is cleared before
// every use for the same reason.
//
// EACH MEMBER MARCHES AT ITS OWN RANGE. A group of a scout and a heavy
// therefore lights two differently sized regions into one map, which is a
// shape a single range could not produce however it was chosen.
func (w *World) groupSight(reader sightReader, members []int) []byte {
	stamp := make([]byte, len(w.grid))
	for _, i := range members {
		e := w.entities[i]
		w.marchSight(reader, stamp, cellOf(e), int32(e.ScanRange))
	}
	return stamp
}

// Sight is the fog reader: owned bodies below the third decay stage and with
// a scan range stamp (TERR-TILE-079).
func (w *World) Sight(owner uint32) []byte {
	stamp := make([]byte, len(w.grid))
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner != owner || e.Decay >= decayDarkStage || e.ScanRange == 0 {
			continue
		}
		w.marchSight(fogSight, stamp, cellOf(e), int32(e.ScanRange))
	}
	return stamp
}

// actorSight is the current perception of one actor, rebuilt from canonical
// terrain and actor state for every admission decision. Actor-directed spells
// use the AI reader rather than a participant's union: a hidden patient or
// enemy another unit can see is not thereby visible to this caster. It is not
// persisted because it is a pure projection of hashed state.
func (w *World) actorSight(i int) []byte {
	stamp := make([]byte, len(w.grid))
	if i < 0 || i >= len(w.entities) || !w.entities[i].Alive() {
		return stamp
	}
	e := w.entities[i]
	w.marchSight(aiSight, stamp, cellOf(e), int32(e.ScanRange))
	w.stampAttackNotices(stamp, []int{i}, false)
	return stamp
}

func (w *World) actorSees(i int, at cell) bool {
	return w.sightShows(w.actorSight(i), at)
}

// actorSeesEntity is the entity-target perception seam. Terrain sight is the
// first half; an active Invisibility effect is the second. An actor always sees
// itself, which keeps self-target restorative spells valid. An invisible actor
// survives the second half only inside the observer's own SeeInvisible radius.
func (w *World) actorSeesEntity(i, target int) bool {
	if i < 0 || i >= len(w.entities) || target < 0 || target >= len(w.entities) {
		return false
	}
	if w.entities[i].ID == w.entities[target].ID {
		return true
	}
	observer, candidate := w.entities[i], w.entities[target]
	if !w.actorSees(i, cellOf(candidate)) {
		return false
	}
	return !w.hasAttachedSpell(candidate.ID, 15) ||
		cellOf(observer).chebyshevTo(cellOf(candidate)) <= int64(observer.SeeInvisible)
}

// invisibleToActor narrows the continuing-order gate to Invisibility itself.
// Ordinary attack orders retain their established acquisition rules; only a
// target carrying spell 15 is dropped when the observer lacks detection.
func (w *World) invisibleToActor(i, target int) bool {
	return target >= 0 && target < len(w.entities) &&
		w.hasAttachedSpell(w.entities[target].ID, 15) && !w.actorSeesEntity(i, target)
}

// sightShows reports whether c is a cell the stamp lit.
func (w *World) sightShows(stamp []byte, c cell) bool {
	i, ok := w.cellIndex(c.x, c.y)
	return ok && stamp[i] != 0
}
