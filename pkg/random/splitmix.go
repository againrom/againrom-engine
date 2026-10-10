// Package random is the engine's one source of random values: the session's
// random service, its named streams and the generators they run on.
//
// It reads no clock and no operating-system state. A caller that wants a
// clock-chosen session seed reads the clock itself and hands the value in, so
// every value this package answers follows from the seeds it was given.
package random

import "math/bits"

// SplitMix64 is the engine's own 64-bit generator over one uint64 of state. It
// is not the original's generator (DIV-027); a value pinned for it is a pin on
// engine code, not on ROM1.
type SplitMix64 struct {
	State uint64
}

// splitMixGamma is the odd additive step, so no seed is a fixed point and the
// state after k draws is seed + k*gamma.
const splitMixGamma = 0x9E3779B97F4A7C15

// Next advances the state by gamma and answers the finalizer's mix of it.
func (r *SplitMix64) Next() uint64 {
	r.State += splitMixGamma
	return mix64(r.State)
}

// mix64 is SplitMix64's finalizer, a bijection of uint64.
func mix64(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Uniform answers a value in [0, n] and always draws once, so the draw count
// never depends on a bound or a drawn value. The reduction is Lemire's high
// half of the 128-bit product; n at or below zero answers 0 after the draw.
func (r *SplitMix64) Uniform(n int32) int32 {
	v := r.Next()
	if n <= 0 {
		return 0
	}
	hi, _ := bits.Mul64(v, uint64(n)+1)
	return int32(hi)
}

// Upto answers a value in [0, n] as the modulo of one draw, and 0 without a
// draw for n below 1. It is the pre-world placement form.
func (r *SplitMix64) Upto(n int32) int32 {
	if n < 1 {
		return 0
	}
	return int32(r.Next() % uint64(n+1))
}
