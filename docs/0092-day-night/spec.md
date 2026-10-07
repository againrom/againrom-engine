# Spec — the day and night cycle, and the light that moves with it

**Intensity:** spec-anchored / static. **Terrain:** brownfield — a relief-lighting grid ships, is
built once, and is built from a sun that never moves — and greenfield for the sun model.

## Why

The original ships the day/night cycle **on**. Its switch is a persisted user option whose compiled
default is 1, and the arm it selects sweeps the sun angle across two of the day's four bands.

This build fixes the sun at the literal the original stores **when that switch is off**. So a fixed
sun here is not a missing feature, it is a **divergence this build has today**, and every relief
level it has ever drawn was drawn on the off arm. What this contract fixes is the sun: where it
stands at a given moment, how often that reaches the screen, and what a consumer is allowed to read
it through.

## Requirements

### The clock

**FR-1** A build carries a **lighting clock** measured in sub-ticks, and **it is the world's own
tick counter: one world tick is one sub-tick**, unsigned, handed to what draws **once per tick** and
never once per frame. Sixteen sub-ticks are one full tick and one full tick is one **in-game
minute**, so the clock's minute is `subTicks >> 4`. The sun's **time-of-day argument** is that minute
plus 360 and the hour is `(argument / 60) mod 24`, so a run beginning at sub-tick 0 begins at in-game
**06:00**.

### The angle

**FR-2** A build carries the cycle and a switch for it, and the switch is **on by default**. While
it is off the sun angle is the literal `+0.78539815` at every minute.

**FR-3** While it is on, the angle is chosen by the hour, in four bands that partition the day:

| Hours | Angle |
|---|---|
| 6…17 — day | `-0.78539815 + m * 0.0021816615277777777`, `m = (argument + 360) mod 720` |
| 18…21 — dusk | the literal `+0.78539815` |
| 22, 23, 0, 1 — night | `+0.78539815 - m * 0.0065449845833333332`, `m = (argument + 120) mod 240` |
| 2…5 — dawn | the literal `-0.78539815` |

Both phase terms cancel — `(argument + 360) mod 720` is `minute mod 720` and `(argument + 120) mod
240` is `minute mod 240` — so the whole of the `+360` lands in the **band boundaries** and none of it
in the phases. The unreduced form is the one the original computes and is written here so that the
two are known to be the same thing.

**FR-4** Both literals are **exactly `+-0.78539815`** — a quarter of a *truncated* pi, whose double
is `3fe921fb4d12d84a` (positive) and `bfe921fb4d12d84a` (negative), and **not** `pi/4`, whose double
is `3fe921fb54442d18`. The day step is **positive**, `0.0021816615277777777`; the original stores its
**negation**, whose double is `bf61df469d353918`. No angle constant is computed from `pi`, and none
is derived from another: `720 x` the day step is a truncated `pi/2` and the night step is the day
step's threefold, and both of those are **checks**, holding to within a part in `10^15` rather than
bit-exactly.

**FR-5** The sun's ambient intensity is `0x0e`, its directional range `0x20` and its sky tint
`(0, 0, 0)`, in **every band and with the switch off alike** — see D-1. With the switch off the whole
sun, not merely its angle, is the one this build already drew.

### What may read it

**FR-6** Every consumer in the **map view** reads the sun through **one cache on that view**, and
only a relight writes it. That includes the sprite layer, which must take its ramp row and its sky
tint from the cache and not from a constant beside it. What is drawn is therefore the sun as of the
**last relight**, never the sun of the current tick. A tool that renders a map outside the view — one
given a sun on its command line — keeps its own and is not bound by this.

### When it is rebuilt

**FR-7** A relight is **unforced** or **forced**. An unforced relight happens exactly on a sub-tick
that both begins a full tick and whose minute satisfies `(minute + 360) mod 20 == 0` — one per 20
in-game minutes, **72 per in-game day**. A forced relight happens whatever the clock reads.

**FR-8** A relight rebuilds the sun from the clock and the switch, and then rebuilds **the whole
per-vertex relief grid** from that sun. There is no partial path: nothing re-shades without the
rebuild.

**FR-9** Moving the switch forces a relight. In the **dusk** band that relight moves nothing, and
that is faithful rather than missed: the dusk arm stores the same literal the switch-off arm does.

**FR-10** A view that is never given a clock **and whose switch and offset are never moved** never
relights and keeps the switch-off sun it was built with, so a build with no world behind it draws
exactly what it drew before this contract. A view whose switch IS moved relights from whatever clock
it holds, which for a view that was never given one is zero.

### Inspecting it

**FR-11** A view carries a **lighting-clock offset**, an unsigned count of sub-ticks added to every
clock it is given. It moves **forward only and by exactly one in-game hour** — 960 sub-ticks — at a
time, is never reduced, and forces a relight when moved. It reaches what is drawn and nothing else:
no world, no tick and no simulation state is addressable through it, and because an hour is three
whole relight periods it produces the same relight instants FR-7 fixes.

### What it is not

**FR-12** None of the clock, the switch or the offset is canonical state: **none is a field of any
simulation record**, no byte form carries any of them, and no digest covers any of them. The clock is
*read from* the world's tick, which the form already carries and the digest already covers, so two
worlds at different ticks draw different suns and are still described by the same form.

### Moving it

**FR-13** On the map screen, one key **toggles the switch** and one key **steps the offset**, each on
a press edge and each doing nothing on any other screen. The switch's key is the original's own; the
offset's is not, and is drawn from this front-end's diagnostic register.

## Acceptance criteria

- **AC-1** With the switch on, the angle at minute 0 is `-0.78539815`, at minute 719 is the day
  arm's last step, at minutes 720 and 959 is `+0.78539815`, at minute 960 is `+0.78539815`, at
  minute 1199 is the night arm's last step, and at minutes 1200 and 1439 is `-0.78539815`.
- **AC-2** Over one in-game day the four bands take 720, 240, 240 and 240 minutes, **960** of the
  1440 minutes carry a computed angle and 480 a literal one, and no minute is unassigned.
- **AC-3** The angle never leaves `[-0.78539815, +0.78539815]` at any minute of any day, and the day
  arm's angle is strictly increasing while the night arm's is strictly decreasing.
- **AC-4** The dusk literal's double is `3fe921fb4d12d84a` and the dawn literal's `bfe921fb4d12d84a`;
  `pi/4`'s is `3fe921fb54442d18` and is neither. The negated day step's double is
  `bf61df469d353918`. `720 x` the day step is `1.5707963` and is **not** `pi/2`; the night step is
  three times the day step. The last two hold to a part in `10^15`, not bit-exactly.
- **AC-5** With the switch off the whole sun — angle, ambient, range and tint — is the one the build
  already drew, at every one of the day's 1440 minutes.
- **AC-6** A clock advanced between two relight instants does not move the drawn sun; the sun moves
  exactly on the relight instants and by nothing else.
- **AC-7** Over one in-game day of sub-ticks, exactly **72** unforced relights fire, on minutes 0,
  20, 40 … and on the sub-tick that begins each of those minutes. Neither half of the test is the
  test: a sub-tick part-way through a relight minute fires nothing, and so does the start of a minute
  that is not a multiple of 20.
- **AC-8** A forced relight applies the sun of the sub-tick it fires on, on a sub-tick the cadence
  would not have fired on — witnessed in the day band, where the angle is computed, and in the dawn
  band, whose literal is the other sign. In the dusk band it moves nothing, per FR-9.
- **AC-9** A view given no clock has the same relief grid as one built from the switch-off sun,
  vertex for vertex, and the same sun.
- **AC-10** On relief the grid drawn at minute 0 differs from the one drawn at minute 720 — the two
  suns are mirror images and the lateral shear's direction follows the angle's sign, so a mirrored
  sun is not the same light — and both differ from noon's. On ground of one height every vertex is 46
  at all three, and at every other minute.
- **AC-11** A world's byte form, its decoded field set and its digest are what they were: the byte
  form's existing offset partition and the existing literal field-set table both hold unchanged, a
  world round-trips unchanged, and two worlds built alike hash alike.
- **AC-12** Moving the offset by one in-game hour moves the drawn sun; moving it by 24 returns the
  drawn sun and the relief grid to exactly what they were; the offset moves no relight instant; and a
  world advanced with the offset moved hashes identically to the same world advanced without.
- **AC-13** No sprite's shading moves with the sun, and it is disconnected from neither: the ramp row
  and the sky tint are taken from the same cache the terrain is, and both are the same at every
  minute of the day because FR-5 holds their two inputs still.
- **AC-14** A view whose altitude layer was unusable at construction — too short, too long, or absent
  — still has no relief grid after any number of relights, and is still not lit.
- **AC-15** Each key acts once per press and not once per frame held, and neither does anything on
  any screen but the map.

## Properties

- **P-1** *Determinism.* Nothing in this contract reaches the simulation. Two worlds advanced against
  equal commands stay equal and hash equal whether the switch is on, off, or moved mid-run, and
  whatever the offset holds.
- **P-2** *Idempotence.* Two relights at the same clock and switch produce the same sun and the same
  relief grid.
- **P-3** *Totality.* Every hour lands in exactly one band and every minute has an angle; no clock
  value, however large, is undefined.
- **P-4** *Boundedness.* Because `|theta| <= 0.78539815`, the relief's horizontal step never exceeds
  `32/cos(0.78539815) = 45.26` and its lateral shear never exceeds `1.0`. A sweep twice this wide lets
  both diverge.

## Divergence from the original

- **D-1** *The per-band intensity and sky tint are held at the daytime configuration.* The original
  grades a sky tint and two intensity bytes per band; the day band's and the switch-off arm's values
  are established and agree, and the dawn, dusk and night arms compute theirs through sequences
  nobody has read. FR-5 therefore ships the established values in every band. The visible
  consequence is exact and is stated rather than hidden: **the sun moves but nothing darkens.** The
  angle changes the relief's contrast and reverses its shear at noon; it does not dim the map at
  night, because the term that would is the undecoded one. Closing it is a table of numbers behind
  FR-5 **plus one structural cost that is named here so that it is not discovered later**: the sprite
  layer's texture cache is keyed by frame and ramp row, and the sky tint is baked into the cached
  pixels without being in the key. A per-band tint therefore needs that key widened, or the first
  band's textures serve the whole session.
- **D-2** *The switch is not persisted.* The original keeps it in the registry and in the savegame's
  options section. This build reads no registry and its byte form is the world, so the switch is
  held where it is consumed and is lost at exit; every run starts at the compiled default, which is
  on.
- **D-3** *The lighting-clock offset is ours and has no counterpart.* It is a diagnostic, FR-11
  bounds what it can touch, and the original has nothing like it.
- **D-4** *No shadow is drawn, so no shadow swings.* The original derives every shadow's lateral
  shear from this same cached angle, so a shadow pass built later must take it from the FR-6 cache
  and not from the current tick.

## Out of scope

The shadow passes themselves — unit, structure and object — which this build does not have at all,
and the shear they take from the angle. **When one is built it must reproduce a live asymmetry:
the shadow shears and the body does not.** The original's unit body computes the identical shear
value and discards it one instruction later, so a faithful build leans the shadow while leaving the
body upright; that reads like a bug in our code and is not one, and it must not be "fixed".

Also out of scope: the per-band tint and intensity schedule (D-1); persisting the switch (D-2); any
display of the in-game time of day; and the wall-clock length of an in-game hour, which is the
pacer's and is unchanged by anything here.

## Traceability

| FR | AC | Property |
|---|---|---|
| FR-1 | AC-1, AC-7 | P-3 |
| FR-2 | AC-5, AC-9 | P-3 |
| FR-3 | AC-1, AC-2, AC-3 | P-3, P-4 |
| FR-4 | AC-4 | P-4 |
| FR-5 | AC-5, AC-13 | — |
| FR-6 | AC-6, AC-13 | P-2 |
| FR-7 | AC-7, AC-8 | — |
| FR-8 | AC-10, AC-14 | P-2 |
| FR-9 | AC-8 | — |
| FR-10 | AC-9, AC-14 | — |
| FR-11 | AC-12 | P-1 |
| FR-12 | AC-11, AC-12 | P-1 |
| FR-13 | AC-15 | — |
