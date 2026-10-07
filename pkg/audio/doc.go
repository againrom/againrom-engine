// Package audio is the entirely synthetic playback leaf: decode a WAV
// sample, compute a stereo placement, mix the two into a device buffer, and
// mute or scale that mix through one named place. Nothing here knows what a
// "grunt" or a "swing" is, what an archive or a registry is, or that the
// game has a listener cell at all — those are pkg/ui's and pkg/game's
// concerns (0126 plan, Shape). This package is the leaf under both: stdlib
// only, no import of another package in this module, so every function below
// is testable with no window, no install and no audio device at all.
//
// # What this package is not
//
// It is not a decoder for one of the GAME's own formats. A WAV file is a
// public, documented container — RIFF chunks, a format tag, PCM samples
// — and nothing about reading one is reverse-engineered from the original
// engine. Everything else below — the falloff radius, the pan radius, the
// resampling algorithm, the mixing formula, the master-volume unit — is
// this tree's OWN choice, spelt out as a named constant rather than a
// literal at a call site, and each such constant says in its own comment
// that it is ours.
//
// # Why the cheap algorithm and not a nicer one
//
// resample and Place both pick the cheapest correct approach on purpose:
// this subsystem's own spec puts breadth before fidelity (spec "Why"), and
// an interpolating resampler or a curved falloff would be exactly the
// fidelity work this story defers.
package audio
