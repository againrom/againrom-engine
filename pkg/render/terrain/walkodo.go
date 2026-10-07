package terrain

const (
	// subCellSpan is one cell along one axis on the sub-cell grid the mover's
	// displacement is measured on: 256 of them to a cell, so a straight cell
	// crossing is a delta of exactly 256 on one axis and a diagonal one 256 on
	// both (MOVE-STEP-010; the direction tables shifted left 8).
	subCellSpan = 256

	// walkPhaseShift is how far one Move-timeline step is: a shift of four,
	// i.e. 16 sub-cell units, a sixteenth of a cell — which is why a straight
	// cell is exactly 16 steps whatever the mover's speed (ANIM-WALK-013,
	// ANIM-WALK-014).
	//
	// It is DECODED AND NOT TUNED: the engine derives the step from the clock
	// with one arithmetic shift by four, and this is that instruction. It is
	// written as the shift and not as the sixteen because the shift is what the
	// engine does and what WalkPhase must do to stay total on negatives.
	walkPhaseShift = 4

	// maxWalkDelta bounds the cell delta the arithmetic will square. It exists
	// only so that a nonsense input cannot overflow the product below; every
	// delta this tree produces is one cell.
	maxWalkDelta = 1 << 20
)

// WalkPhase is the Move-timeline step a distance walked stands on: the count
// divided by sixteen, ROUNDED DOWN.
//
// Down and not toward zero. The engine's is an arithmetic shift, Go's `/` is a
// truncation, and the two differ on exactly the negatives that make this
// function total — a count no walk can produce, but one the selector must still
// answer for rather than refuse. Go's `>>` on a signed value IS the arithmetic
// shift, so this is the engine's instruction and not an imitation of it.
func WalkPhase(odo int) int { return odo >> walkPhaseShift }

// WalkStepOdometer is WalkPhase's inverse: the least distance walked that
// stands on timeline step step.
//
// It exists for a caller ENUMERATING the timeline rather than driving it — the
// corpus audit, which must reach every step of a class's Move track to sweep
// the whole selection domain and has no distance to reach it with. Without it
// that caller would have to hold a copy of the sixteen, which is the one number
// this file exists to keep in one place.
func WalkStepOdometer(step int) int { return step << walkPhaseShift }

// WalkAdvance is how far a mover travels on ONE tick of a cell crossing, in
// sub-cell units: the euclidean length of that tick's own share of the
// crossing, taken to its whole part.
//
//	dx, dy — the crossing's cell delta, so (1,0) straight and (1,1) diagonal
//	span   — how many ticks the whole crossing takes
//	tick   — which tick of it this is, counted from 0
//
// THE SHARE IS THE ENGINE'S OWN SPLIT, not a proportion of the crossing. The
// engine divides the delta still owed by the ticks still owed, truncating, and
// subtracts what it took; over span ticks that lands each axis on exactly its
// whole delta, with the LARGER shares last. walkShare is that sequence in
// closed form — see there for why the two are the same sequence and not merely
// the same total.
//
// The length is then the integer square root of the two shares squared. That
// is the engine's square root followed by its truncating float-to-int of the
// sum with an integer clock: the clock is a whole number and the length is
// never negative, so truncating their sum adds the length's whole part,
// which is the integer root.
//
// A zero delta travels nothing, whatever the span. A span below one is read as
// one, and a tick outside the crossing is brought inside it: this is a drawing
// path and it answers rather than refuses. No input panics and no input divides
// by zero.
func WalkAdvance(dx, dy, span, tick int) int {
	if dx == 0 && dy == 0 {
		return 0
	}
	if span < 1 {
		span = 1
	}
	if tick < 0 {
		tick = 0
	}
	if tick > span-1 {
		tick = span - 1
	}
	sx := int64(walkShare(dx, span, tick))
	sy := int64(walkShare(dy, span, tick))
	return isqrt(sx*sx + sy*sy)
}

// walkShare is one axis's share of one tick of a crossing: the whole delta
// abs(d)*subCellSpan split over span ticks the way the engine's own recurrence
// splits it.
//
// THE RECURRENCE AND THIS FORM ARE THE SAME SEQUENCE. The engine holds the
// delta still owed and takes `owed / ticksOwed` each tick, truncating, then
// subtracts it. Write q = total/span and m = total mod span. By induction the
// owed amount before tick k is (span-k)*q + min(m, span-k), so the division
// gives q while m < span-k and q+1 from k = span-m on: the first span-m ticks
// take q and the last m take q+1. That is O(1) where the recurrence is O(span)
// per mover per tick, and it carries no state — the caller need not remember
// how much of a cell is still owed, which is the whole reason this tree can
// reconstruct the split from a tick count alone.
//
// It is stated here as an argument and PROVED IN THE TEST, which re-executes
// the recurrence itself for every span the rate law admits and requires the two
// to agree share for share. An argument in a comment is not evidence.
//
// The SIGN is dropped: the odometer accumulates a distance, and the engine's
// negative arm is the mirror of its positive one — its divide and Go's both
// truncate toward zero, so the shares of a negative delta are the negatives of
// the shares of its magnitude. The magnitude is clamped so that the caller's
// square of it cannot overflow.
func walkShare(d, span, tick int) int {
	if d < 0 {
		d = -d
	}
	if d > maxWalkDelta {
		d = maxWalkDelta
	}
	total := d * subCellSpan
	q, m := total/span, total%span
	if tick >= span-m {
		return q + 1
	}
	return q
}

// isqrt is the whole part of the square root of n, by Newton's method on
// integers: the largest r with r*r <= n, for every n >= 0, and 0 for a negative
// n so that the function is total.
//
// The iteration decreases strictly until it reaches that r and then stops, so
// it terminates for every input and needs no bound on the number of rounds. The
// first guess is n itself, which is at or above the root for every n >= 1.
func isqrt(n int64) int {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return int(x)
}
