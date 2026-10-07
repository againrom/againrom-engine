package sim

import "math/bits"

// rng is the world's single pseudo-random source: SplitMix64 over one uint64 of
// state. It is the world's only generator, it is owned by the world rather than
// by the process, and its whole state is those eight bytes.
//
// This generator is OUR OWN ENGINEERING CHOICE and is not claimed to be the
// original engine's, and that is now a DISCLOSED DIVERGENCE rather than an
// unrecovered question. The original's generator is MSVC's CRT rand():
// seed = seed*214013 + 2531011, returning (seed>>16) & 0x7fff — so its
// RAND_MAX is 0x7fff (AI-RAND-058) — and the AI module's own uniform-range
// idiom divides that returned value by 0x8000, RAND_MAX plus one, rather than
// by RAND_MAX itself (AI-RANGE-102). SplitMix64 shares none of that: not the
// state width, not the step, not the range a raw draw lands in. A value
// pinned for this generator anywhere in the tree is a pin on our own code and
// must not be read as game-derived.
//
// Its FIRST consumer is the attack cycle: a damage roll, a to-hit roll and the
// jitter on a relax. Everything else about a world is still arithmetic.
//
// math/rand is not an option here — it is banned in this package, and it is
// process-global besides, which is the same nondeterminism under another name.
type rng struct {
	state uint64
}

// RandomState returns the complete current native generator state without
// consuming a draw. It is not the original engine's RNG or a SAV field.
func (w *World) RandomState() uint64 { return w.rng.state }

// RestoreRandomState installs the complete native stream during a cold LOAD.
// It consumes no draw and changes no other simulation state.
func (w *World) RestoreRandomState(state uint64) { w.rng.state = state }

// gamma is SplitMix64's additive step, the odd 64-bit approximation of the
// golden ratio. It is odd, so adding it repeatedly walks the whole 2^64 state
// space before repeating and no seed — zero included — is a fixed point.
const gamma = 0x9E3779B97F4A7C15

// next advances the state by gamma and returns the finalizer's mix of the new
// state. The mixed value is never fed back, so the state after k draws follows
// from the seed and k alone.
func (r *rng) next() uint64 {
	r.state += gamma
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// uniform returns a value in [0, n], both ends included, and is the ONE place a
// bounded draw is taken.
//
// IT ALWAYS DRAWS. The generator advances even where the answer is fixed — at an
// n of zero, and at an n below zero, where the answer is zero as well — so how
// many draws an advance makes follows from the world's state and its commands
// and never from a value a draw produced. A helper that returned early on a
// degenerate bound would make the draw count depend on a damage spread, and a
// replay of a recorded stream would then diverge for a reason no recorded state
// explains.
//
// The reduction is Lemire's: multiply the drawn word by n+1 in 128 bits and take
// the high half, which lands in [0, n] and costs no division. Its bias is one
// part in 2^64 per outcome and it draws exactly once, where rejection sampling
// would draw a number of times that depends on the values drawn — the same fault
// the paragraph above refuses. A modulo is worse on the first count and no better
// on the second.
//
// n+1 is taken in uint64 after the sign test, so the widening cannot wrap: the
// largest n this reaches is the top of int32, and 2^31 is a uint64.
func (r *rng) uniform(n int32) int32 {
	v := r.next()
	if n <= 0 {
		return 0
	}
	hi, _ := bits.Mul64(v, uint64(n)+1)
	return int32(hi)
}
