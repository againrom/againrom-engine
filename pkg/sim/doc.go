// Package sim holds the deterministic simulation core.
//
// Determinism wall: sim advances only via a pure Step(state, commands) -> state
// on a fixed integer tick — no IO, no wall-clock, no floats. CHANCE IS NOT AN
// EXCEPTION TO IT: the one source of it is the world's own integer generator,
// whose whole state the byte form carries and the digest covers, so a draw is
// part of the state a replay resumes rather than something reaching in from
// outside. It imports only the
// Go standard library and no other againrom tier, which internal/archtest's
// import check enforces; that check permits every standard-library import, so it
// says nothing about os, time, math/rand or floats, which need no import at all.
//
// Those are held out by a second check beside it, internal/archtest's source
// scan over this package's non-test files: it fails on an import of os, time or
// math/rand — or of any package under one of them — on a float32, float64,
// complex64 or complex128 identifier, and on a floating-point or imaginary
// literal. It reads parsed syntax, not file text, so the same names in a comment
// or a string are not findings.
//
// What that proves is lexical: no such import, type name or literal occurs in
// these files. It does not prove that nondeterminism cannot arrive some other
// way — a float reached through another package's untyped constant (math.Pi) is
// invisible to it. See docs/ARCHITECTURE.md.
package sim
