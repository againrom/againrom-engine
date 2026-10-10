package sim

import "testing"

// gamma is SplitMix64's additive step; a seeded stream's state after k draws
// is seed + k*gamma.
const gamma = 0x9E3779B97F4A7C15

// seed0 pins the first four values our generator produces from seed 0. It is a
// change detector for OUR OWN generator: SplitMix64 is a fixed published
// function, so anyone holding no code of ours can recompute these, and a change
// to the constants or to the finalizer fails here instead of silently moving
// every world's RNG state.
//
// Nothing about these numbers is game-derived (see rng.go).
var seed0 = []uint64{
	0xe220a8397b1dcdaf,
	0x6e789e6aa1b965f4,
	0x06c45d188009454f,
	0xf88bb8a8724c81ec,
}

func draw(seed uint64, n int) []uint64 {
	r := rng{state: seed}
	out := make([]uint64, n)
	for i := range out {
		out[i] = r.next()
	}
	return out
}

func TestRNGPinnedSequence(t *testing.T) {
	got := draw(0, len(seed0))
	for i := range seed0 {
		if got[i] != seed0[i] {
			t.Errorf("draw %d from seed 0 is %#016x, pinned as %#016x", i, got[i], seed0[i])
		}
	}
}

// TestRNGOneSeedOneSequence: a seed determines the whole sequence, and two seeds
// give different values at the same index. The comparison is element-wise at the
// same index on purpose — SplitMix64's state is seed + k*gamma, so two seeds an
// exact multiple of gamma apart produce the SAME sequence shifted, which no
// pairwise-different claim would survive and this test does not make.
func TestRNGOneSeedOneSequence(t *testing.T) {
	const n = 64
	for _, seed := range []uint64{0, 1, 0x0123456789ABCDEF} {
		a, b := draw(seed, n), draw(seed, n)
		for i := range a {
			if a[i] != b[i] {
				t.Fatalf("seed %#x: two runs disagree at draw %d: %#x vs %#x", seed, i, a[i], b[i])
			}
		}
	}
	one, two := draw(1, n), draw(2, n)
	same := 0
	for i := range one {
		if one[i] == two[i] {
			same++
		}
	}
	if same != 0 {
		t.Errorf("seeds 1 and 2 agree at %d of %d draws", same, n)
	}
}

// TestRNGStateFollowsFromSeedAndDrawCount is the property that makes eight bytes
// enough to carry the generator: the state advances by gamma per draw and the
// mixed output is never fed back into it.
func TestRNGStateFollowsFromSeedAndDrawCount(t *testing.T) {
	for _, seed := range []uint64{0, 1, 0xFFFFFFFFFFFFFFFF} {
		r := rng{state: seed}
		for k := 1; k <= 32; k++ {
			r.next()
			if want := seed + uint64(k)*gamma; r.state != want {
				t.Fatalf("seed %#x after %d draws: state %#016x, want %#016x", seed, k, r.state, want)
			}
		}
	}
}

// TestRNGZeroSeedIsNotAFixedPoint: zero is an ordinary seed here, which is the
// reason this generator was chosen over an xorshift whose state zero absorbs.
func TestRNGZeroSeedIsNotAFixedPoint(t *testing.T) {
	r := rng{state: 0}
	for k := 0; k < 16; k++ {
		if v := r.next(); v == 0 {
			t.Errorf("draw %d from seed 0 is zero", k)
		}
		if r.state == 0 {
			t.Fatalf("state returned to zero after %d draws", k+1)
		}
	}
}

// An original stream is the MSVC recurrence drawn through the range wrapper,
// and its mode and state survive the byte form.
func TestOriginalStreamRoundTripsAndDrawsAsTheRangeWrapper(t *testing.T) {
	w, err := NewWorld(7, Bounds{Width: 4, Height: 4}, ModeCanonical, make([]byte, 16), nil)
	if err != nil {
		t.Fatal(err)
	}
	seeded, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w.SetRandom(originalMode, 1)
	probe := crtProbe{state: 1}
	if got, want := w.rng.uniform(0), int32(0); got != want || w.RandomState() != 1 {
		t.Fatalf("uniform(0) = %d at state %d, want no draw", got, w.RandomState())
	}
	for _, n := range []int32{5, 100, 0x7fff} {
		if got, want := w.rng.uniform(n), probe.rangeOf(n); got != want {
			t.Fatalf("uniform(%d) = %d, want %d", n, got, want)
		}
	}
	if got, want := w.OriginalRand(), probe.rand(); got != want {
		t.Fatalf("OriginalRand = %d, want %d", got, want)
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if b[29]&0x40 == 0 || seeded[29]&0x40 != 0 {
		t.Fatalf("mode bit seeded %#x original %#x", seeded[29], b[29])
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back.RandomMode() != originalMode || back.RandomState() != uint64(probe.state) {
		t.Fatalf("decoded mode %d state %d, want original %d", back.RandomMode(), back.RandomState(), probe.state)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("round trip changed the digest")
	}
}

func TestOriginalDrawsAreTheRangeWrapper(t *testing.T) {
	d := NewOriginalDraws(1)
	probe := crtProbe{state: 1}
	for _, n := range []int32{0, 3, 70, 32767} {
		if got, want := d.Upto(n), probe.rangeOf(n); got != want {
			t.Fatalf("Upto(%d) = %d, want %d", n, got, want)
		}
	}
	if !d.Original() || d.State() != uint64(probe.state) {
		t.Fatal("original draws lost their state")
	}
}

// originalMode is random.Original; sim tests import only the standard library.
const originalMode = 1

// crtProbe is an independent CRT rand() oracle: the recurrence and the range
// wrapper written out from AI-RAND-058 and MAGIC-284.
type crtProbe struct{ state uint32 }

func (p *crtProbe) rand() int32 {
	p.state = p.state*214013 + 2531011
	return int32(p.state>>16) & 0x7fff
}

func (p *crtProbe) rangeOf(n int32) int32 {
	if n == 0 {
		return 0
	}
	return int32(int64(p.rand()) * int64(n+1) / 32768)
}
