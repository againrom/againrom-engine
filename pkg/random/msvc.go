package random

// MSVC is the original's generator, the CRT rand() recurrence over one 32-bit
// seed (AI-RAND-058, MAGIC-279): state = state*214013 + 2531011, and a draw is
// bits 16..30 of the new state, 0..32767. A CRT thread block starts at seed 1.
type MSVC struct {
	State uint32
}

const (
	msvcMultiplier = 214013
	msvcIncrement  = 2531011
)

// RandMax is the largest value Rand answers.
const RandMax = 0x7fff

// OriginalStartSeed is the seed a CRT thread block starts at (MAGIC-279).
const OriginalStartSeed = 1

// Rand is one raw rand() draw, 0..RandMax.
func (r *MSVC) Rand() int32 {
	r.State = r.State*msvcMultiplier + msvcIncrement
	return int32(r.State>>16) & RandMax
}

// Range is the integer-range wrapper (MAGIC-284): 0 without a draw when n is
// 0, otherwise one draw scaled as rand()*(n+1) shifted right 15 with
// truncation toward zero over a wrapping 32-bit product. For 0 <= n <= 65537
// that is floor(rand()*(n+1)/32768); n = -1 draws and answers 0.
func (r *MSVC) Range(n int32) int32 {
	if n == 0 {
		return 0
	}
	return rangeOf(r.Rand(), n)
}

// rangeOf is the range wrapper's arithmetic over a value already drawn.
func rangeOf(raw, n int32) int32 { return int32(uint32(raw)*uint32(n+1)) / 32768 }

// RangeFrom1 is the wrapper that passes n-1 and adds 1 (MAGIC-284); n = 1
// answers 1 without a draw.
func (r *MSVC) RangeFrom1(n int32) int32 { return 1 + r.Range(n-1) }

// Float is the float wrapper (MAGIC-284): one draw, rand()/32767.0, 0.0..1.0.
func (r *MSVC) Float() float64 { return float64(r.Rand()) / RandMax }

// LCG is a linear congruential step with a masked shifted output, the form a
// data description names when it carries its own generator. MSVC is the
// instance with the original's constants.
type LCG struct {
	Multiplier, Increment uint32
	Shift                 uint
	Mask                  uint32
}

// Step advances state and answers the masked output.
func (g LCG) Step(state *uint32) int {
	*state = *state*g.Multiplier + g.Increment
	return int(*state>>g.Shift) & int(g.Mask)
}
