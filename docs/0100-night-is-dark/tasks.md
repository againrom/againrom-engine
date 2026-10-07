# 0100 — night is dark: tasks

## T1 the per-band schedule

Add `pkg/render/terrain/skylight.go` with `SkyLight(minute uint64, cycle bool) Light` (FR-1), the
seven arms of spec.md's table as a `switch` on the hour, each arm's values its own decimal literal,
and one `ramp(k, m int) int` helper for `q` (FR-2, DD-1, DD-2). Phase is `minute % 120` after the
modulo-1440 reduction; the hour is `SunAngle`'s own expression (P-1).

Add `ShroudObject` and `ShroudUnit` `uint8` fields to `Light` in `light.go` and give
`DefaultDaytime` 4 and 2 (FR-5, DD-3). Rewrite `SunAt` in `sun.go` to return `SkyLight(minute,
cycle)` with `Theta` from `SunAngle`, no longer copying `DefaultDaytime`, and replace its "THE SUN
MOVES BUT NOTHING DARKENS" comment with what it now does (FR-4, DD-6).

Tests in `skylight_test.go`: every arm at its first and last minute against literals (AC-1); the
cycle-off arm at the listed minutes (AC-2); day equals cycle-off equals `DefaultDaytime`'s three
parameters (AC-3, FR-3); the continuity sweep over 1440 minutes: max step 1 in each of the five,
all in [0,48], the two exact joins (AC-4, P-2); the tint never darkens, over every minute,
channel and level (AC-6, FR-8); flat-ground level 46 at noon and 64 at midnight through `LevelGrid`
with multipliers 1.5625 and 1.0 (AC-5, DD-7); `SpriteRow` runs 3..8 and takes all six (AC-7).

Touch no file under `pkg/sim` and add no byte-form version (FR-11).

## T2 the tint reaches the screen

Widen `spriteTextureKey` in `pkg/ui/statics.go` with `tint [3]uint8` and fill it from the same
light `spritePixels` bakes in, so a band's textures cannot be served into another band (FR-7,
DD-5).

Widen `cacheKey` in `pkg/ui/viewer.go` the same way and build the cell texture through the tint:
per channel `clamp(chan + tint, 0, 255)`, the vertex multiplier in `withScales` unchanged (FR-6,
DD-4). Put the per-channel add in `pkg/render/terrain` beside the shading transform, expressed at
its identity level 64 rather than as a new arithmetic, and call it from `cellImage`.

The tint both keys carry is zero whenever `Lit()` is false, and the placeholder fill is untinted
(FR-9). Nothing about the relight cadence or the switch's default changes (FR-10).

Tests: equal lights give equal keys and a tint-only difference gives different keys, on both paths
(AC-8); a tinted cell texture differs from the untinted one by exactly the tint per channel,
saturating at 255, and is byte-identical under a zero tint (AC-9); under the unshaded diagnostic the
texture is the untinted source and every corner multiplier is 1 (AC-10). Keep them free of a
graphics context, as the package's existing pixel tests are.
