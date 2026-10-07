package sim

// The rate law, term by term, as integers.
//
// Every constant below is one term of the decoded movement-rate law, named
// at the one place it is used so that a reader can set the arithmetic beside
// the law rather than beside a number nobody can source.
//
// speedMultiplier is a MAP PARAMETER in the original, read from a registry key
// this tree has no reader for. The shipped file carries the code default in both
// releases, so the constant and the file agree today; when a reader arrives it
// replaces this one name and nothing else.
//
// slopeLimit and slopeShift are the height tilt: the two cells' height
// difference saturates at plus or minus slopeLimit, and the correction is that
// difference times the rate, shifted right ARITHMETICALLY by slopeShift — so
// uphill reduces and downhill increases. A shift and not a divide: the two
// differ on negatives, and the negative half is the uphill one.
//
// fallbackMeanCost is what the law substitutes when the two cost bytes average
// to zero. A world here carried no cost plane until 0076 and so always took it;
// now it is taken on a mean that is genuinely zero or has WRAPPED to zero, and
// on a transit with an endpoint outside the world's bounds, which the planes
// describe no cell for. It is also the byte an absent cost plane materialises at
// — see defaultCost, which is derived from this name so the two cannot drift —
// and it is the value most of the original's own cells carry.
//
// rateFloor and rateCeil clamp the result. They are hard-coded immediates in the
// original, carried by no shipped file, so they are a customisation limit rather
// than a parameter.
//
// subCell is the sub-cell grid: a mover advances the rate's worth of 1/subCell
// of a cell per tick along its dominant axis, so a cell takes ceil(subCell/step)
// of them.
//
// diagNumerator and diagDenominator are the DIAGONAL constant as a ratio. The
// original multiplies by a double whose shipped bytes read as a value a shade
// below 707/1000, truncating the product. This package may hold no float at all,
// and the ratio reproduces that truncation EXACTLY for every rate the law admits
// — see diagonalStep.
const (
	speedMultiplier = 8

	slopeLimit = 32
	slopeShift = 6

	fallbackMeanCost = 8

	rateFloor = 1
	rateCeil  = 63

	subCell = 256

	diagNumerator   = 707
	diagDenominator = 1000
)

// maxTransit is the longest transit the law can produce: the sub-cell grid over
// the smallest step the clamp permits. It is derived from the two constants it
// depends on rather than written as a number, so the byte form's refusal of a
// longer one cannot come to disagree with what transitOf can return.
const maxTransit = subCell / rateFloor

// rateOf is the mover's per-tick displacement, in 1/subCell of a cell along
// its dominant axis, for one cell transit.
//
// It takes the two cells' COST and HEIGHT bytes as parameters rather than
// reading them from a world, and that is the whole shape of this file: a world
// in the signature would make the law untestable except through a world. When
// the planes arrived in 0076 that shape was worth exactly what it was written
// for — the change was to this function's ONE CALLER and not a line of this
// function.
//
// The two arms are the original's own fork on the movement domain, and they are
// not two spellings of one rule:
//
//   - the GROUND domain takes the multiplier, the slope tilt and the divide by
//     the two cells' mean cost;
//   - EVERY OTHER domain takes the raw speed — no multiplier, no tilt, and no
//     cost read at all.
//
// They agree exactly at a multiplier of 8 and a mean cost of 8, which is what
// the original ships and what an absent cost plane still produces, so a world
// over flat default ground computes one number through either arm and only a
// test at another mean cost separates them. That is a fact about those inputs,
// not about the law, and it is why the fork is written out rather than
// collapsed — a derived plane runs from 6 to 16 and separates them everywhere.
//
// The arithmetic is done in 64 bits and clamped at the end. The original's speed
// source is a 16-bit field, so within its own range the width changes nothing;
// what it buys is that a customised speed far outside that range cannot wrap
// into a small plausible rate on the way to a clamp that would then accept it.
//
// Both byte-width behaviours the law depends on are the PARAMETER TYPES' own
// arithmetic rather than masking written out: the cost add wraps in a uint8, and
// the height difference is a uint8 subtraction read back as a signed byte.
func rateOf(d Domain, speed int32, costSrc, costDst, hSrc, hDst uint8) int32 {
	v := int64(speed)
	if d == DomainGround {
		v *= speedMultiplier

		// The difference is taken in the plane's own byte width and read back
		// signed, so a drop of more than 127 wraps exactly as it does there, and
		// it then saturates rather than scaling without bound.
		rise := int64(int8(hSrc - hDst))
		if rise > slopeLimit {
			rise = slopeLimit
		}
		if rise < -slopeLimit {
			rise = -slopeLimit
		}
		v += (v * rise) >> slopeShift

		// A byte-wide add: two costly cells CAN sum past 255 and wrap, and the
		// law does not guard it. The mean is that wrapped byte halved, and a
		// mean of zero — which is every cell in this tree, and in the original
		// only a wrap that lands on 0 or 1 — takes the substitute.
		c := int64((costSrc + costDst) >> 1)
		if c == 0 {
			c = fallbackMeanCost
		}
		v /= c
	}
	if v < rateFloor {
		return rateFloor
	}
	if v > rateCeil {
		return rateCeil
	}
	return int32(v)
}

// diagonalStep is the per-axis step of a DIAGONAL move at rate v: the
// original's truncated product of v with a constant a shade below 707/1000.
//
// THE INTEGER RATIO IS EXACT, and the argument is arithmetic rather than
// empirical. 707 and 1000 are coprime, so 707v/1000 is an integer only when 1000
// divides v; for every other v the exact product sits at least 1/1000 from an
// integer, while the shipped constant differs from 707/1000 by under 5e-17 and
// one rounding of the product adds under 1e-14 — twelve orders of magnitude
// inside that margin. So the two truncate to the same value for every v below
// 1000, and rateOf clamps v to rateCeil, sixteen times below that.
//
// The equality is not left as an argument: rate_test.go re-executes both forms,
// the original's own shipped bytes included, for every v in 0..999.
//
// The bound is 999 and not larger, deliberately. At exactly 1000 the product IS
// an integer and the margin the argument rests on vanishes, so which way the
// product lands is a question about one rounding rather than about the law.
// Nothing here can reach it; asserting anything there would be asserting what
// the argument does not carry.
func diagonalStep(v int32) int32 {
	return int32(int64(v) * diagNumerator / diagDenominator)
}

// transitOf is how many ticks one cell transit takes at rate v: the sub-cell
// grid divided by the step, rounded UP, since the surplus travel of the last
// tick is discarded when the mover lands centred.
//
// A DIAGONAL costs more than a straight move and not the other way round, which
// is the one thing to get right here: the step shrinks by the diagonal constant
// while the cell to cross does not, so the same rate takes longer. It is not the
// square root of two, and how far it departs from it is a customisation limit
// the story records rather than a rounding to tidy away.
//
// WHERE THE LAW HAS NO VALUE, we take a step of one. A rate of 1 stepped
// diagonally truncates to a step of 0, and the original divides the grid by that
// — an expression with no value at all. Taking one gives the longest transit the
// grid allows, maxTransit. It is ours by necessity and it is unreachable from
// shipped data, whose smallest rate is 2; it is reachable by CUSTOMISING a speed
// low enough, which is the reason it is answered here rather than left to
// whatever a division would do.
func transitOf(v int32, diagonal bool) int32 {
	step := v
	if diagonal {
		step = diagonalStep(v)
	}
	if step < rateFloor {
		step = rateFloor
	}
	return (subCell + step - 1) / step
}
