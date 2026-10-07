# Tasks — 0126

Trailer: `SDD-Task: 0126-sound/T<n>`.

## T1 — the audio leaf

Build `pkg/audio` per plan.md's T1 section: `Sample`, `DecodeWAV`, the resampler, `Placement`,
`Place`, `GainUnit`, `FalloffCells`, `PanCells`, `Settings`, `MasterUnit`, `DefaultSettings`,
`Stereo`, `Player`, `DeviceRate`. Standard library only — no ebiten, no other package of this
module.

Register the package in `internal/archtest`'s allow-map with an empty import set and add its row to
`docs/ARCHITECTURE.md`'s tier table, as a leaf beside `pkg/render/text`.

Tests, all synthetic, all built in Go: AC-12 (16-bit mono, 8-bit mono, 16-bit stereo downmix, a
truncated header, a non-PCM format tag, a missing data chunk), AC-13 (a sample at twice the device
rate, and one already at it — assert the identity case returns the same backing array), AC-9 over
`Place` (left of the listener, right of it, on it, and past `FalloffCells`), AC-10 over `Stereo`
(muted, master zero, master halved against master full, asserting exactly half the amplitude).

A twenty-line helper that builds a RIFF/WAVE byte stream is the whole fixture layer. **No game file
is read and none is committed.**

Do not touch `pkg/ui`, `pkg/game`, `cmd/` or `pkg/sim`.

## T2 — the grunt, the throttle and the device

Per plan.md's T2 section: add `pkg/audio` to `pkg/ui`'s allow-map row, add `Sound []int` to
`ui.MapEntity`, and write `pkg/ui/sound.go` and `pkg/ui/sounddev.go`.

`stepSound` is called from `stepNumerals` as its first statement. It reads the viewer's per-entity
health memory and must not write it — `ingestDamage` keeps sole ownership of that map and runs after.

Tests in `pkg/ui`, against a recording `audio.Player` written in the test file: AC-1, AC-2, AC-3,
AC-4, AC-5, AC-6, AC-7, AC-8, AC-11. Use whatever the package's existing viewer tests use to push
entities and step a frame; the clock reaches the step already, so AC-5 needs no new seam.

`OpenAudio` is **never called from a test** — opening a device is what a headless machine cannot do.
Its correctness in this task is that it compiles, that a nil player is a lawful viewer state, and
that every guard is on the `playSlotAt` path where AC-11 can see it.

Do not touch `pkg/game`, `cmd/` or `pkg/sim`. Do not change `numeral.go` beyond the one call.

## T3 — the archive, the class table and the swing

Per plan.md's T3 section: `pkg/game/sound.go` with the archive constants, `OpenSounds`, the lazy
`Sample`, `LoadUnitSounds` and `UnitSound`; the `FrontEnd` fields and their wiring at the two sites
that already hand the viewer its font; the entity-seam fill and the swing emission in
`pkg/game/world.go`; the two flags in `cmd/againrom`.

The swing emission is a callback field on the world, nil for every path that has no viewer.

Tests in `pkg/game`, synthetic: a registry built in test code resolving three slots including a
sparse id, a slot with no entry, and a value with backslashes; `OpenSounds` on a missing and on a
malformed archive returning nil and no error surface (P-2); `LoadUnitSounds` over a synthetic
`units.reg` carrying a five-element array and a class carrying none; AC-14 driving the swing counter
past the class's attack delay and asserting exactly one emission per run, at the right tick, with
the attacker's own cell. `cmd/againrom`: the two flags' defaults, in the existing flag test.

Do not touch `pkg/sim`, `pkg/audio` or `pkg/ui`.

## Traceability

| Spec | Task |
|---|---|
| FR-7, FR-8, FR-14, FR-15 | T1 |
| FR-1, FR-2, FR-3, FR-4, FR-6, FR-11 | T2 |
| FR-5, FR-9, FR-10, FR-12, FR-13, FR-16 | T3 |
| AC-9, AC-10, AC-12, AC-13 | T1 |
| AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-11 | T2 |
| AC-14, AC-15 | T3 |
| DD-3, DD-4, DD-11 | T1 |
| DD-2, DD-5, DD-9 | T2 |
| DD-1, DD-6, DD-7, DD-8, DD-10 | T3 |
| P-3 | T1 |
| P-1 | T2 |
| P-2 | T3 |
| SC-3 | T2 |
| SC-1, SC-2, SC-4 | T3 |
