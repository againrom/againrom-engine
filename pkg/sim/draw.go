package sim

import "againrom/pkg/random"

// Draws is a standalone sequence over this package's generator, for a caller
// that must choose before a world exists.
//
// A LOADER IS EXACTLY SUCH A CALLER, and it is why this type exists at all. The
// world owns its own generator and keeps it private, which is the right shape
// for anything the simulation does while it runs; but where a unit starts is
// decided while the world is still being assembled, and a loader that reached
// for a generator of its own would put a second one in the tree — process-global,
// or duplicated, or both.
//
// The generator is the world's: SplitMix64 over one uint64 of state, OUR OWN
// engineering choice and not the original engine's. The original seeds from a
// clock, which is the one thing a reproducible loader must not do, so no value
// drawn here may be read as game-derived.
//
// A sequence is a value with no hidden state: two Draws built from one seed
// answer alike for as long as they are asked alike, and nothing about the
// process, the machine or the order of goroutines can enter. It is not safe for
// concurrent use, which is the same statement.
type Draws struct {
	r rng
}

// NewDraws is the sequence that starts at seed.
func NewDraws(seed uint64) *Draws {
	return &Draws{r: rng{state: seed}}
}

// NewOriginalDraws is the sequence that continues an original stream at
// state; Upto is then the original's range wrapper.
func NewOriginalDraws(state uint32) *Draws {
	return &Draws{r: rng{state: uint64(state), original: true}}
}

// Original reports whether the sequence runs on the original's generator.
func (d *Draws) Original() bool { return d.r.original }

// State answers the sequence's state after the draws taken so far.
func (d *Draws) State() uint64 { return d.r.state }

// Upto is a uniform value in [0, n] — INCLUSIVE OF n — and 0 for any n below 1.
//
// The inclusive bound is the interface's, not an accident of the arithmetic: the
// original's own helper scales its generator by n+1, so the value it names as a
// bound is reachable, and a consumer that wrote a half-open range would be one
// short at the top and differently distributed throughout. That is a clause the
// research corrected once already; spelling it here means it is spelled once.
//
// The distribution is the modulo of a 64-bit draw, so it is not perfectly
// uniform for an n+1 that does not divide 2^64. At the bounds this project uses —
// a list of drop cells, a coordinate under 128 — the excess is around one part in
// 2^57, which no test could distinguish from uniform and which no shipped map
// exercises at all. It is written down rather than corrected because rejection
// sampling would make the number of draws depend on their values, and the draw
// COUNT is part of what makes a load reproducible.
func (d *Draws) Upto(n int32) int32 {
	if n < 1 {
		return 0
	}
	if d.r.original {
		return d.r.uniform(n)
	}
	g := random.SplitMix64{State: d.r.state}
	v := g.Upto(n)
	d.r.state = g.State
	return v
}
