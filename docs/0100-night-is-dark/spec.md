# 0100 — night is dark: specification

The day/night cycle already moves the sun's **angle** with the in-game minute. It moves nothing
else: the same ambient, the same slope range and the same all-zero sky tint are handed to every
band, so the relief's contrast turns over the day and the map never dims. This story puts the whole
per-band light on the same clock, and carries it to the ground and to every sprite.

## The schedule

Let `minute` be the in-game minute, `hour = ((minute mod 1440) + 360) / 60 mod 24`, and

    m    = minute mod 120                 the ramp phase
    q(k) = (k * m) / 120                  integer division, truncating toward zero

Seven arms. Each names five values — the three sky-tint bytes R, G, B, the **ambient** byte and the
**range** byte — and the two shroud indices FR-5 carries.

| arm | hours | R | G | B | ambient | range | shroud obj / unit |
|---|---|---|---|---|---|---|---|
| cycle off | — | 0 | 0 | 0 | 14 | 32 | 4 / 2 |
| day | 6..17 | 0 | 0 | 0 | 14 | 32 | 4 / 2 |
| dusk 1 | 18,19 | `q(24)` | 0 | `q(8)` | `14 + q(10)` | `32 - q(12)` | 6 / 3 |
| dusk 2 | 20,21 | `24 - q(24)` | `q(12)` | `8 + q(40)` | `24 + q(8)` | `20 - q(12)` | 6 / 3 |
| night | 22,23,0,1 | 0 | 12 | 48 | 32 | 8 | 8 / 4 |
| dawn 1 | 2,3 | `q(24)` | 12 | `48 - q(48)` | `32 - q(4)` | `8 + q(12)` | 6 / 3 |
| dawn 2 | 4,5 | `24 - q(24)` | `12 - q(12)` | 0 | `28 - q(14)` | `20 + q(12)` | 6 / 3 |

Each two-hour half-band spans exactly 120 minutes and opens at `m = 0`, so every ramp runs `q` from
`0` to its `k*119/120` and no ramp is entered part-way.

## Functional requirements

**FR-1 — the band light.** `terrain.SkyLight(minute uint64, cycle bool) Light` returns the arm above
that `minute` and `cycle` select: the three tint bytes as `SkyTint`, the ambient and range bytes as
`Ambient` and `Range`, the two shroud indices as FR-5's fields. With `cycle` false it returns the
cycle-off arm at every minute. `Theta` is not this function's and it returns the zero angle.

**FR-2 — the ramp.** The four ramp arms compute `q(k) = (k*m)/120` with `m = minute mod 120` and
integer division truncating toward zero. Every `k` and every additive term in the table above is
written as its own decimal literal; no constant is derived from another, and no ramp is expressed
through another ramp.

**FR-3 — day is the cycle-off arm.** The two arms carry the same five values. They are written as two
separate arms with their own literals, and their agreement is asserted rather than assumed.

**FR-4 — the sun carries them.** `terrain.SunAt(minute, cycle)` returns `SkyLight(minute, cycle)`
with `Theta` set to `SunAngle(minute, cycle)` and nothing else changed. It no longer reads
`DefaultDaytime`.

**FR-5 — the shroud indices are carried and nothing reads them.** `Light` gains two `uint8` fields
holding the object-path and unit-path shroud/silhouette indices. `DefaultDaytime` carries 4 and 2.
No drawing and no exported behaviour outside `SkyLight` and `SunAt` reads either field in this
story; they exist so the shadow pass inherits them.

**FR-6 — the ground takes the tint and the level.** Terrain drawn through the viewer's GPU path
carries the sky tint. The texture a cell is drawn from holds `clamp(channel + tint, 0, 255)` per
channel, and the per-corner vertex multiplier stays exactly what it is today. The texture cache is
keyed by the tint as well as by the cell it resolves to.

**FR-7 — sprites take the tint at the band they are drawn in.** The sprite texture cache is keyed by
the sky tint as well as by the frame and the ramp row, so a band's textures are never served into a
later band.

**FR-8 — nothing in the tint darkens.** The tint is an unsigned addend applied before the level
multiply, in both FR-6's and FR-7's paths. No channel is ever reduced by a tint byte.

**FR-9 — the diagnostics are unlit, not tinted.** Under the unshaded diagnostic, and for a
placeholder cell with no source image, the tint is zero and the shading multiplier is 1.

**FR-10 — the cycle is on.** The viewer's day/night switch defaults on, its relight cadence is
unchanged, and a relight rebuilds the sun, the level grid, and whatever the two caches must rebuild
for FR-6 and FR-7.

**FR-11 — simulation is untouched.** No file under `pkg/sim` changes, no byte form gains a version,
and nothing here enters any digest.

## Acceptance criteria

**AC-1** `SkyLight` returns the tabled values at the first and last minute of every one of the seven
arms, each expected value written as a literal in the test.

**AC-2** `SkyLight(minute, false)` is the cycle-off arm for `minute` 0, 1, 359, 360, 719, 720, 1439,
1440 and 1000000, and is the same value at every minute of a day.

**AC-3** The day arm and the cycle-off arm are equal in all five values, and both agree with
`DefaultDaytime`'s `Ambient`, `Range` and `SkyTint`.

**AC-4 — continuity.** Over all 1440 minutes of a day with the cycle on: the largest step in any of
R, G, B, ambient and range between one minute and the next is **1**; every one of the five stays
within `[0, 48]`; and the two joins night to dawn 1 and day to dusk 1 are exact.

**AC-5 — the level a pixel gets.** The flat-ground relief level computed by `LevelGrid` from
`SunAt`'s light is **46** at noon and **64** at midnight, and the shading multiplier a ground pixel
takes at those levels is **1.5625** and **1.0** — night ground is 64% of noon's brightness. Asserted
on the level and on the multiplier, not on the ambient byte.

**AC-6 — the tint cannot darken.** For every minute of a day, every 8-bit channel value and every
level, the shaded output with the band's tint is greater than or equal to the output with a zero
tint at the same level. This fails if any tint byte is read as a signed offset or subtracted.

**AC-7 — the sprite row moves.** `SpriteRow(SunAt(minute, true))` is 3 at noon and 8 at midnight, and
over a day takes exactly the values 3, 4, 5, 6, 7, 8.

**AC-8 — the caches discriminate.** Two lights differing only in their sky tint produce different
cache keys on both the ground path and the sprite path, and identical lights produce equal keys.

**AC-9 — the ground texture carries the tint.** A cell texture built under a band with a non-zero
tint differs, per channel, from the same cell's texture under a zero tint by exactly that band's
tint, saturating at 255; under a zero tint it is byte-identical to the untinted source.

**AC-10** Under the unshaded diagnostic the ground texture is byte-identical to the untinted source
and every corner multiplier is 1.

**AC-11** `missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3` ends
`lost at tick 272` with 2 movers on both the ru and the en root, unchanged from before this story.

## Properties

**P-1 — total over `uint64`.** `SkyLight` is defined at every `uint64` minute with no panic and no
out-of-range value; the reduction modulo 1440 is exact because 120 and 60 both divide 1440, so a
reduced minute and a raw one select the same band and the same ramp phase.

**P-2 — the band values never leave a byte.** No arm produces a value below 0 or above 48, so no
subtraction underflows and no store wraps.

**P-3 — a relight is idempotent.** Relighting twice at the same clock and switch leaves the sun, the
level grid and both caches in the state one relight leaves them in.

**P-4 — one sun.** Every drawn pixel of one frame — ground and sprite — takes its tint and its
intensity from the single cached light the last relight wrote. There is no second light and no call
site that names a tint or a row of its own.

## Divergences

**D-1 — 0092's D-1 is closed.** That story disclosed "the ambient, range and tint do not move with
the band" as a deliberate hold. This story removes the hold and the divergence with it. Nothing in
`docs/0092-day-night/` is amended; that contract stands as the record of what it shipped.

**D-2 — the ground tint saturates one step early.** The engine bakes a table and computes
`clamp(((chan + tint) * (96 - level)) / 32, 0, 255)` with no clamp before the multiply. This tree
carries the level as a GPU vertex multiply, so the tinted channel must survive an 8-bit texture and
is clamped at 255 before it is multiplied. The two agree wherever the multiplier is at least 1, i.e.
at every level up to 64, which covers all flat ground at every band. Above level 64 — steep slopes
away from the sun — a channel whose palette value exceeds `255 - tint` comes out dimmer than the
engine's by at most 2 of 255, falling with the multiplier. Ours, stated, and not visible by eye.

**D-3 — the sprite ramp and the terrain ladder are two ladders.** They already are, and this story
does not join them. The sprite row is `ambient>>2` and takes six values across a day; the ground
level is continuous in the ambient byte. They dim together, not identically.

## Out of scope

The shadow and silhouette pass (FR-5 carries its inputs and draws nothing). Any change to `pkg/sim`,
to a byte form, or to a digest. The map-stored light fields, which the engine overwrites. RGB565
packing.
