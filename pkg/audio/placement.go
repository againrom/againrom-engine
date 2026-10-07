package audio

// GainUnit is the fixed-point unit every gain and pan term in this package is
// expressed over. It is a DECODED value and not a round number chosen for
// convenience: spec.md's Terms section ties it directly to the original —
// "the unit is 10000 because that is the clamp the original applies to both
// of its own positional terms" — so a Placement's Left and Right are integers
// in [0, GainUnit] by the same rule the original enforces on its own two
// values.
const GainUnit = 10000

// FalloffCells is the distance, in map cells, at which a sounding cell stops
// being heard at all: Place's gain reaches zero here and Place returns false
// at or beyond it, so nothing plays rather than playing at zero gain (AC-9).
//
// THIS NUMBER IS OURS.
const FalloffCells = 40

// PanCells is the horizontal offset, in map cells, at which the stereo pan
// reaches its full swing to one side: Place's pan term is clamped to
// ±GainUnit at or beyond this many cells left or right of the listener.
//
// THIS NUMBER IS OURS, for the reason FalloffCells is: no claim behind this
// story recovers the original's own pan curve, only the width its terms are
// clamped to. It is set to half of FalloffCells so a sound pans hard to one
// side well before it also fades to silence, rather than the two effects
// finishing together at the same cell.
const PanCells = FalloffCells / 2

// Placement is one play's left and right gains, each an integer in
// [0, GainUnit] (spec Terms). It carries no sample and no position of its
// own: Place is the one function that produces one from a relative cell
// offset, and Stereo is the one function that consumes one.
type Placement struct {
	Left, Right int
}

// Place computes the Placement for a sound at (dx, dy) map cells from the
// listener — the sounding cell's coordinates minus the listener's — or
// reports false when the sound lies at or beyond FalloffCells and should not
// play at all (AC-9).
//
// Pan is dx*GainUnit/PanCells, clamped to ±GainUnit: a NEGATIVE dx — the
// sounding cell to the LEFT of the listener — raises the left gain above the
// right, and a positive dx does the reverse, so two cells at the same range
// but on opposite sides of the listener are louder in the near ear and
// quieter in the far one by the same total gain, rather than merely louder
// overall. On the listener's own cell (dx == 0) pan is zero and Left equals
// Right exactly.
func Place(dx, dy int) (Placement, bool) {
	dist := isqrt(int64(dx)*int64(dx) + int64(dy)*int64(dy))
	if dist >= FalloffCells {
		return Placement{}, false
	}
	gain := GainUnit - dist*GainUnit/FalloffCells

	pan := int64(dx) * GainUnit / PanCells
	switch {
	case pan > GainUnit:
		pan = GainUnit
	case pan < -GainUnit:
		pan = -GainUnit
	}

	left := clampGain(gain * (GainUnit - pan) / GainUnit)
	right := clampGain(gain * (GainUnit + pan) / GainUnit)
	return Placement{Left: left, Right: right}, true
}

// clampGain folds a computed gain back into [0, GainUnit]. Place's own
// algebra can carry a term as high as 2*gain into this division — a fully
// panned nearby sound scales gain by up to 2*GainUnit on its near side — so
// the upper clamp is reached in that ordinary case and not only on malformed
// input.
func clampGain(g int64) int {
	switch {
	case g < 0:
		return 0
	case g > GainUnit:
		return GainUnit
	default:
		return int(g)
	}
}

// isqrt is the integer floor of the square root of n, for n >= 0: the largest
// integer x with x*x <= n. It is Newton's method starting from n itself,
// which converges monotonically down to the floor value in a bounded number
// of steps for every non-negative n a squared cell offset can produce, and it
// touches no float — see Place's own comment for why that is load-bearing
// here rather than incidental.
func isqrt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	x := n
	for {
		y := (x + n/x) / 2
		if y >= x {
			return x
		}
		x = y
	}
}
