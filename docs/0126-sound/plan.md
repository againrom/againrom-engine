# Plan — 0126

Three tasks, one per tier, in dependency order. Nothing here touches `pkg/sim`, and the serialized
byte form is untouched: no version number is allocated to this story.

## Shape

```
pkg/audio      NEW leaf, stdlib only      decode, resample, placement, mix, settings, Player
                                          (no ebiten, no vfs, no game knowledge)
pkg/ui         + pkg/audio                the grunt rule, the throttle, the listener,
                                          the ebiten-backed device
pkg/game       already may import pkg/    the archive, the slot registry, the class slot
                                          table, the swing emission, the wiring
cmd/againrom                              two flags
```

## T1 — `pkg/audio`, the leaf

New package `pkg/audio`, stdlib only, no intra-module imports. Registered in `internal/archtest`'s
allow-map as an empty row and added to `docs/ARCHITECTURE.md`'s tier table — a leaf with no outgoing
edge, the same shape `pkg/render/text` already has. `externalAllowed` needs no change: it already
denies every external import outside the formats, ui and cmd tiers, so the stdlib-only rule is
mechanical rather than a promise.

Contents:

- `Sample{Rate int; PCM []int16}` — mono. `DecodeWAV(b []byte, rate int) (Sample, error)`: walk the
  RIFF chunk list, require `WAVE`, read the format chunk, refuse anything but format 1, 1-2
  channels, 8 or 16 bits; read the data chunk; downmix stereo by averaging the pair; widen 8-bit
  unsigned to signed 16-bit; then resample to `rate` (FR-14, FR-15). Chunk walking respects the
  odd-size pad byte. A truncated or absent chunk is an error, never a panic.
- `resample(in []int16, from, to int) []int16` — nearest neighbour, index `i*from/to`, computed in
  int64 so a long sample cannot overflow. Identity when the rates match, and identity by *returning
  the input*, so the common case allocates nothing (P-3).
- `Placement{Left, Right int}` and `GainUnit = 10000`.
- `Place(dx, dy int) (Placement, bool)` — the sounding cell minus the listener cell. Distance is an
  integer square root. Gain falls linearly from `GainUnit` at zero to zero at `FalloffCells`; beyond
  it the second return is false and nothing plays (FR-7, AC-9). Pan is `dx * GainUnit / PanCells`
  clamped to one unit either way. Then left is `gain*(GainUnit-pan)/GainUnit` and right is
  `gain*(GainUnit+pan)/GainUnit`, each clamped to `GainUnit`. Named constants `FalloffCells = 40`
  and `PanCells = 20`, with a comment saying both are ours (DD-11).
- `Settings{Master int; Muted bool}`, `MasterUnit = 100`, `DefaultSettings`. A muted or zero master
  yields no gain at all.
- `Stereo(s Sample, p Placement, st Settings) []byte` — interleaved 16-bit little-endian stereo,
  each side scaled by its gain over `GainUnit` and by the master over `MasterUnit`, accumulated in
  int32 and clamped to the int16 range. Returns nil when both scaled gains are zero (FR-8, AC-10).
- `Player interface { Play(Sample, Placement) }`. Settings are held by the caller and not by the
  device, because a mute must silence a device that is not there either.
- `DeviceRate = 22050` (DD-4), documented with the corpus measurement that chose it.

**DD-1 note for the executor:** floats are permitted here — this is the front-end side of the
determinism wall — but the whole of the above is integer arithmetic anyway, which is what makes
AC-9, AC-10 and AC-13 exact assertions rather than tolerances.

## T2 — `pkg/ui`, the rule and the device

Allow-map: `pkg/ui` gains `pkg/audio`.

`ui.MapEntity` gains one field, `Sound []int` — the class's sound slots, carried whole and read by
index alone, documented in the shape `Speed` and `GroupSpeed` already use: this tier interprets
nothing, it indexes. Plain `int`, so the package still names no other tier's type.

New `pkg/ui/sound.go`:

- Viewer fields: the player, the bank, the settings, and `soundAt map[uint32]time.Time`.
- `SoundBank interface { Sample(slot int) (audio.Sample, bool) }` — declared here, implemented in
  `pkg/game`, so the drawing tier never learns what an archive is.
- `SetAudio(p audio.Player, b SoundBank, st audio.Settings)` — one setter, the shape `SetFont` has.
- `GruntThrottle = 1500 * time.Millisecond` (FR-3), `GruntFloor = -10`, and the four slot indices
  named as constants with `spec.md`'s index table beside them. The throttle map is keyed on the
  victim, so FR-3 is per victim rather than per frame.
- `gruntSlot(hp, maxHP int) (int, bool)` — pure: false at or below `GruntFloor`, index 3 below
  `maxHP/2`, else index 2 (FR-1, FR-2).
- `listenerCell() (image.Point, bool)` — the midpoint of the camera's visible tile range; false when
  there is no camera or no grid.
- `playSlotAt(slot int, cell image.Point)` — the ONE path to the device: refuse a slot of 0 or less,
  refuse a nil device or a nil bank, compute the placement against the listener, ask the bank, play.
  Every FR-7, FR-8, FR-10, FR-11 and AC-11 guard lives here and nowhere else.
- `PlaySlotAt(slot int, cell image.Point)` — the exported form, for `pkg/game`'s swing (FR-5).
- `stepSound(now time.Time)` — walks the frame's entities, compares each against the health memory
  **before** `ingestDamage` rewrites it, applies `gruntSlot`, applies the throttle, plays. Called
  from `stepNumerals` as its first statement, ahead of the drift.

`stepSound` reads the health memory and must not write it; `ingestDamage` keeps sole ownership of
that map. An entity absent from it plays nothing (FR-4, AC-6). The numeral toggle is not consulted
(FR-6, AC-8).

New `pkg/ui/sounddev.go` — the ebiten-backed player. `OpenAudio() (audio.Player, error)` guards the
ebiten audio context with a `sync.Once` (it panics if constructed twice), creates it at
`audio.DeviceRate`, and returns a player whose `Play` builds the stereo buffer with `audio.Stereo`
and hands it to a byte-backed ebiten player. A nil buffer plays nothing. A failure returns an error
and no player; **no test in this package may call it**, because opening a device is what a headless
machine cannot do — AC-11's nil-device path is the tested one.

## T3 — `pkg/game`, the assets and the swing

New `pkg/game/sound.go`:

- `SfxArchive = "sfx.res"` and the address prefix beside it, the shape `graphicsPrefix` already
  states its reason in, plus the registry address `sfx/sfx.reg`.
- `OpenSounds(root string) *SoundBank` — takes the **already resolved** asset root, the one the
  existing flag and environment variable produce, so this subsystem names no install path of its own
  (FR-9). It opens the sound archive alone through `OpenContainers`,
  parses the registry, and builds a slot table from the `[Sfx]` section's `Sfx<n>` keys, folding
  backslashes to forward slashes and appending the `.wav` extension the values omit (FR-12).
  **Returns nil on every failure** and reports none: a nil bank is the silent state (FR-10, P-2).
- `Sample(slot int) (audio.Sample, bool)` on that bank — lazily reads the entry, decodes at the
  device rate, caches the result, and caches the *failure* too, so a bad leaf is decoded once
  (FR-13, P-3).
- `LoadUnitSounds(src terrain.EntrySource) map[int32]UnitSound` with
  `UnitSound{Slots []int32; AttackDelay int32}` — parses `units.reg` through the same two calls
  `LoadUnits` makes and keeps the two fields that loader drops. It is a second parse of one registry
  at startup and that is the accepted cost: widening `LoadUnits`'s signature would touch ten call
  sites across three commands for two fields none of them wants.
- `FrontEnd` gains `Sound SoundOptions{Enabled bool; Volume int}` and the resolved player, bank and
  class table, filled in `NewFrontEnd` after the required archives open. `OpenAudio`'s error is
  **swallowed into a nil player** (FR-11). The viewer is handed all three at the two sites that
  already hand it the font.

`pkg/game/world.go`:

- `push` fills the new entity field from the class slot table, widened to `[]int`. The lookup is by
  the entity's own class id — the *placed* class, not the substituted art class, because a sound is
  the unit's and not the drawing's.
- `advanceSwings` gains the swing emission: while charging, on the tick the swing counter reaches
  that class's attack delay, play slot 0 at the attacker's cell (FR-5, AC-14). It is emitted in the
  same walk that owns the counter, so there is no second traversal and no memory of "has this run
  already sounded" — the counter passes each value once per run. The emission is a callback field on
  the world, nil for every developer front-end, so the tier stays testable with no viewer.

`cmd/againrom`: `-sound` (bool, default true) and `-volume` (int 0..100, default 100), set beside
`front.Markers`, with a usage string each (FR-16).

## Where each decision and each scope claim lands

The contract states these; this is the tier each one is honoured in, so a reader looking for the
code behind a decision has one place to go.

| Spec | Where |
|---|---|
| DD-1 (outside the determinism wall) | nowhere — T1/T2/T3 all stay out of `pkg/sim`, and no version is taken |
| DD-2 (decision separate from device) | T1's `Player` interface, T2's `playSlotAt` |
| DD-3 (pan synthesised) | T1's `Stereo` |
| DD-4 (22050 Hz) | T1's `DeviceRate`, spent by T2's context and T3's decode |
| DD-5 (grunt where the health memory is) | T2's `stepSound`, T3's fill of the entity seam |
| DD-6 (swing where the swing clock is) | T3's `advanceSwings` |
| DD-7 (the attack-delay tick is disclosed) | T3, in the comment at the emission site |
| DD-8 (unchanged-health grunt refused) | T2's `gruntSlot`, which has no such arm |
| DD-9 (throttle on the wall clock) | T2's `GruntThrottle` and `soundAt` |
| DD-10 (sound archive optional) | T3's `OpenSounds`, opened apart from the required four |
| DD-11 (every number in one named place) | T1's constants and `Settings`, T2's `GruntThrottle`/`GruntFloor` |
| SC-1 (blow sounds only) | nothing loads ambience, music or interface sound |
| SC-2 (no channel policy) | T2's device plays and forgets |
| SC-3 (element 4 unused) | T2's slot constants name it and read it nowhere |
| SC-4 (no volume surface) | T3's two flags are the whole seam |

## Traceability

Every FR, AC, DD, P and SC of `spec.md` is landed by exactly one of T1, T2, T3; `tasks.md` carries
the table. `verification.md` and the build are stages, not tasks, and carry no trailer.
