package sim

import "againrom/pkg/random"

// rng is the world's single pseudo-random source, owned by the world and
// carried whole in its byte form. A seeded stream is SplitMix64 over all eight
// bytes, the engine's own choice (DIV-027); a value pinned for it is a pin on
// engine code. An original stream is the original's CRT rand() recurrence
// (AI-RAND-058) over the low four bytes, drawn through the original's range
// wrapper (MAGIC-284). Both generators live in pkg/random.
type rng struct {
	state uint64
	// original runs the stream on the original's MSVC recurrence over the low
	// 32 bits of state, with the original's range wrapper as uniform.
	original bool
}

// RandomState returns the complete current native generator state without
// consuming a draw. It is not the original engine's RNG or a SAV field.
func (w *World) RandomState() uint64 { return w.rng.state }

// RestoreRandomState installs the complete native stream during a cold LOAD.
// It consumes no draw and changes no other simulation state.
func (w *World) RestoreRandomState(state uint64) { w.rng.state = state }

// RandomMode answers which generator the world's stream runs on.
func (w *World) RandomMode() random.Mode {
	if w.rng.original {
		return random.Original
	}
	return random.Seeded
}

// SetRandom installs a stream in mode at state. An original stream keeps the
// low 32 bits.
func (w *World) SetRandom(mode random.Mode, state uint64) {
	w.rng.original = mode == random.Original
	if w.rng.original {
		state = uint64(uint32(state))
	}
	w.rng.state = state
}

// OriginalRand is one raw rand() draw of an original stream, 0..32767. A
// seeded stream answers the top 15 bits of one draw.
func (w *World) OriginalRand() int32 {
	if !w.rng.original {
		return int32(w.rng.next() >> 49)
	}
	m := random.MSVC{State: uint32(w.rng.state)}
	v := m.Rand()
	w.rng.state = uint64(m.State)
	return v
}

// next advances a seeded stream and returns its next 64-bit value.
func (r *rng) next() uint64 {
	g := random.SplitMix64{State: r.state}
	v := g.Next()
	r.state = g.State
	return v
}

// uniform returns a value in [0, n], both ends included, and is the ONE place a
// bounded draw is taken. A seeded stream always draws, whatever n is, so the
// draw count never follows a value (random.SplitMix64.Uniform). An original
// stream is the original's range wrapper, which draws nothing at n = 0
// (MAGIC-284).
func (r *rng) uniform(n int32) int32 {
	if r.original {
		m := random.MSVC{State: uint32(r.state)}
		v := m.Range(n)
		r.state = uint64(m.State)
		return v
	}
	g := random.SplitMix64{State: r.state}
	v := g.Uniform(n)
	r.state = g.State
	return v
}
