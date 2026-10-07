# 0161-spell-art — spec

A cast draws the game's own projectile art. The sprite sheet a spell names is
read out of the player's install, flown from the caster to the target for the
length the original gives that picture, and the burst sheet is drawn where the
original puts one. The coloured square and expanding ring 0154 drew instead are
removed.

The simulation is untouched. Which spells exist, what they cost, what they do
and when they land are exactly what they were; this story changes only the
picture.

## What the player sees

The build ships two castable spells: **Fire Arrow** and **Heal**. Both fly in
the original, and this is what each looks like after this story.

**Fire Arrow.** A fire bolt sprite leaves the caster and crosses to the target.
It is one of only two spell sheets in the game that turn: the sheet holds nine
stored directions and the bolt is drawn in the one nearest its flight, mirrored
for the other seven. It runs a four-frame cycle at one frame per two ticks. The
crossing takes the distance in cells, times 256, divided by 200 ticks — about six
ticks over five cells. **On arrival it shows nothing further.** Fire Arrow's
burst picture is id 11, and the game ships no art at that id, so the original
draws nothing at the moment of impact either. The damage numeral and the ring on
the struck unit, both already in the build, remain the whole of what marks the
hit.

**Heal.** The heal sprite is a small twelve-pixel glow. Its flight length is 1
tick: it appears on the target and is gone on the next tick, which is what the
original's own table gives that picture. It does not turn — the sheet holds one
direction. **On arrival it shows nothing further**, for Fire Arrow's reason: heal's
burst picture is id 21 and the game ships no art at that id.

So neither of the two spells the player can cast today produces a burst. That is
the original's behaviour and not a gap in this build: a burst sheet exists for
eight pictures, none of which either of these two spells names.

Five further spells fly, and this story builds them although nothing in this
build casts them yet: Fire Ball, Drain Life, Lightning, Prismatic Spray and
Teleport. Every other spell in the game spawns nothing that flies, and after this
story this build spawns nothing that flies for them either.

## Requirements

**FR-1 — the projectile registry loads.** `graphics/projectiles/projectiles.reg`
is parsed into one record per row, keyed by that row's own `ID` and not by its
section number. Each record carries `File`, `Phases`, `RotationPhases`, `Width`,
`Height`, `Palette`, `Homing`, `Flip`, `SFX` and `A16`. A key a row omits takes
the loader's own default: `ID` -1, `Phases` -1, `RotationPhases` 16, `Width` 64,
`Height` 64, and 0 for the rest. An unreadable or unparseable registry yields an
error and no records; a row whose `ID` is absent is refused.

**FR-2 — a spell names two pictures.** The flying object's picture is
`2*spellId + 8` and the burst's is `2*spellId + 9`. A picture naming no record
draws nothing at all, and that is a normal outcome rather than a failure: seven
of the 28 spells name no record at either parity.

**FR-3 — exactly seven pictures fly, each for its own length.** The flight length
in ticks is `dist/200` for picture 10, `dist/384` for 12, `1` for 20 and 30, `13`
for 34 and 36, `21` for 60, and `0` for every other picture. `dist` is the
caster-to-target distance with one cell measuring 256. A picture whose length is
0 spawns no flying object. A length that computes to 0 for a nonzero arm is
raised to 1, so a cast at point-blank range still shows its sprite for one tick.

**FR-4 — the object crosses in its own length.** A flying object of length `N` is
drawn on `N` consecutive ticks and stands at `(age + 1) / N` of the way from the
caster's cell to the target's on the tick numbered `age`, so it reaches the
target on its last drawn tick and is gone after it. That is the driver's own
motion, which closes the remaining gap over the remaining ticks and therefore
runs at a constant rate. Its span is the picture's own, and it is not the
caster's attack charge: a fast caster and a slow caster throw the same picture
for the same number of ticks.

**FR-5 — the sheets come from the install.** A record's sheet is the archive
entry `graphics/projectiles/<File>.16a`, with `File`'s backslashes read as
separators. It is decoded through the 16-level alpha format and each pixel
resolved to a premultiplied colour at coverage `(level+1)/16`. Nothing is added
to the repository and no sheet is read at start-up that a cast cannot reach. A
record whose `A16` is 0 names a sheet in the other format and is skipped; on the
shipped registry that is three arrow rows no cast can name.

**FR-6 — the frame drawn.** Where a record's `RotationPhases` is 1 the frame is
the phase alone. Otherwise the flight direction is folded to a 16-way facing
biased so that 0 is south; where `Flip` is set, facings 9 to 15 are mirrored onto
7 to 1, leaving nine stored, and the frame is `Phases * facing + phase`, drawn
reflected for a mirrored facing. A frame index outside the sheet draws nothing.

**FR-7 — the phase clock is one frame per two ticks.** `phase = ((age + 1) / 2) %
Phases` for every picture but two: picture 60 takes `age` with no modulus, and
picture 51 takes `age + 1`. A record whose `Phases` is not positive has no phase
and draws nothing.

The `+1` is not an adjustment. The engine's counter is incremented before the
phase is read, so its first drawn tick is counter 1, and `age` here counts drawn
ticks from 0. Picture 60 is the check: its counter form is `counter - 1`, its
life is 21 ticks and its sheet holds 21 phases, so the two agree exactly only
under this alignment.

**FR-8 — the art is centred by the registry's halves.** The frame's top-left
corner sits at the object's point minus `Width/2` and `Height/2` as the registry
states them, which are the draw's centring offsets and not the art's dimensions —
on the shipped rows the two disagree, and the art is not clipped or scaled to fit.

**FR-9 — the burst is a stationary object with a decoded life.** When a cast is
applied and its burst picture names a record, that sheet is drawn at the target's
cell, at the target's cell for every tick of its life, for 16 ticks — 18 for
picture 27 and 22 for picture 13. It runs FR-7's clock. It does not move, and it
does not track the target: it stands where the cast landed.

**FR-10 — teleport draws two.** Picture 60 puts a second, identical object at the
caster's own cell, drawn for the same life and running the same clock.

**FR-11 — the authored marks are gone.** The coloured square drawn for a bolt in
flight and the expanding ring drawn on arrival are removed, together with the
four-tick burst constant behind the ring. Where the original draws nothing this
build draws nothing.

**FR-12 — the picture is gated and placed exactly as the mark it replaces.** It
is hidden wherever the caster is hidden by fog, it takes the same relief lift on
displaced terrain, and it is drawn in the map's content band — over the terrain
and the units, under every diagnostic marker, the health bars, the routes, the
damage numerals and every window.

## Acceptance criteria

**AC-1** A registry stream carrying two rows, one stating every key and one
stating only `ID` and `File`, loads two records; the second holds the six
documented defaults and the first holds its own values. A stream whose `[Global]
Count` names more sections than exist loads the sections that are there.

**AC-2** Over spell ids 0 to 27, the cast picture is `2*id + 8` and the burst
picture is `2*id + 9`; the seven ids 12, 15, 17, 20, 24, 25 and 28 resolve a
record at neither parity against the shipped registry's id set.

**AC-3** The flight length of pictures 10, 12, 20, 30, 34, 36 and 60 is nonzero
at a distance of five cells and the flight length of the other 44 reachable
pictures is 0. At picture 10 and five cells the length is 6.

**AC-4** A cast of a spell whose picture does not fly produces no drawable at
all, for every tick of the cast run.

**AC-5** A cast of Fire Arrow at five cells produces one drawable for six
consecutive ticks and none afterwards. Its position at age 5 is the target's
cell, at age 2 the halfway point, and at age 0 one sixth of the way out.

**AC-6** Against a synthetic record with `Phases = 4`, `RotationPhases = 16` and
`Flip = 1`, the frame index and mirror bit are the documented function of facing
and phase over all 16 facings: facings 0 to 8 unmirrored at `4*facing + phase`,
facings 9 to 15 mirrored at `4*(16-facing) + phase`. Against a record with
`RotationPhases = 1` the index is the phase at every facing.

**AC-7** The phase of a `Phases = 4` record over ages 0 to 9 is
0,1,1,2,2,3,3,0,0,1. Picture 60 at `Phases = 21` gives the age itself over its
whole 21-tick life, and picture 51 gives `age + 1`.

**AC-8** A cast of a spell whose burst picture names a record produces a burst
drawable standing on the target's cell for exactly 16 ticks, 18 for picture 27
and 22 for picture 13, and none afterwards.

**AC-9** A teleport cast produces two drawables from the tick it is applied, one
on the caster's cell and one crossing to the target's.

**AC-10** A frame index past the sheet's frame count, a record with no frames, a
record with `Phases` at 0 and a picture with no record each produce no drawable
and no error.

**AC-11** A viewer holding spell drawables submits one draw call per drawable
whose caster is visible and none for one whose caster is fogged, in the content
band and before the first overlay pass.

## Properties

**P-1** Every function that selects a frame, a phase, a facing or a flight length
is a pure integer function of its arguments. None reads a clock, a world or a
sheet, none panics on any input including negative ages and zero deltas, and equal
inputs give equal answers.

**P-2** The drawing tier receives a sheet, a frame index, a mirror bit and a
point. It derives no picture id, no phase and no direction, and it names no
simulation, format or data type.

**P-3** `go test` stays green with no game install present. Every fixture in this
story is a byte stream built in test code from the documented registry and sprite
contracts.

**P-4** A viewer that is never handed a spell drawable composes exactly the frame
it composed before this story, and builds no texture.

**P-5** The simulation's hashed state, its byte form and `formatVersion` are
untouched. A save written before this story loads after it and digests the same.

## Cut

**SC-1** The smoke trail pictures 10 and 12 draw behind themselves. It is one of
the sheets the registry does not name and its evidence is decompiled rather than
instruction-level.

**SC-2** The sound a burst plays at id `500 + picture`.

**SC-3** The ramp pictures 34 and 36 take their phase from. Its per-index
constants are not published, so those two run FR-7's default clock instead. The
values it yields are known to be 4, 3, 2 and 1 over a 13-entry table, so the
divergence is bounded: the two spells play the low frames of their sheets in a
different order from the original.

**SC-4** The `.256` projectile rows. No cast can name one; they are reachable
only through a unit class's own `Projectile` key, which is the archer's arrow and
not a spell.

**SC-5** The muzzle table a cast object is spawned from. The bolt leaves the
caster's cell, which is where this build already put it.

**SC-6** The area-effect delivery that puts a burst on the ground in a ring. This
build models no area effect, so a burst is drawn only where a cast lands.

**SC-7** The unit-borne shot's use of the same art. An archer's arrow keeps the
mark it has.
