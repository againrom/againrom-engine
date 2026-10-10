package random

import "math/rand"

// Go is the standard library's seeded generator. The presentation streams and
// the shop stock ran on it before this package existed; they keep it in seeded
// mode so every output pinned for them stays the same.
type Go struct {
	r *rand.Rand
}

// NewGo is the generator that starts at seed.
func NewGo(seed int64) *Go { return &Go{r: rand.New(rand.NewSource(seed))} }

// Intn answers a value in [0, n); n must be positive.
func (g *Go) Intn(n int) int { return g.r.Intn(n) }

// Raw answers one 15-bit value, 0..RandMax, the form the presentation
// consumers scale inline.
func (g *Go) Raw() int { return g.r.Intn(RandMax + 1) }
