# 0161-spell-art — plan

## Where each piece lands

| Tier | File | Carries |
|---|---|---|
| `pkg/data` | `projectile.go` (new) | FR-1's registry records, FR-2's picture arithmetic, FR-3's flight table, FR-9's burst lifetimes, FR-5's sheet address |
| `pkg/render/terrain` | `effect.go` (new) | FR-5's decoded frame type, FR-6's frame selection, FR-7's phase clock, FR-8's centring, the 16-way facing |
| `pkg/game` | `projectiles.go` (new) | the loader: registry to bundle, `.16a` to frames |
| `pkg/game` | `spellbolt.go` | FR-3, FR-4, FR-9, FR-10 — which objects exist and how far along they stand |
| `pkg/game` | `frontend.go`, `world.go` | the bundle reaches a mission |
| `pkg/ui` | `spellbolt.go` | FR-11, FR-12 — the seam and the sprite pass |

## Decisions

**DD-1 — a new frame type in the render tier, not `StaticFrame`.** `StaticFrame`
is an 8-bit index plus a palette and one boolean per pixel. A `.16a` pixel
carries a 4-bit coverage as well, and the sheets this story draws are translucent
glows whose whole appearance is that coverage. Resolving them into `StaticFrame`
would discard it. `terrain.EffectFrame` therefore holds premultiplied
`color.RGBA` per pixel — resolved once, at load, by the tier that may read a
sprite stream — and the drawing tier uploads it unchanged. *Rejected:* widening
`StaticFrame` with an alpha field, which would put a byte on every pixel of every
static object and unit sheet in the game for a use only 23 sheets have.

**DD-2 — the blend is `cursorPixel`'s, moved rather than copied.** `pkg/game`
already resolves a `.16a` pixel to a premultiplied colour, in `cursor.go`, and
two `.16a` readers in one package that disagreed on what a level means would be
invisible on screen. The function stays where it is and the projectile loader
calls it. This is the statics loader's own precedent: one palette walk, not two
(FR-5).

**DD-3 — the bundle is keyed by picture id and built once.** `REG-PROJ-086`'s
array is indexed by `ID`, so a lookup by picture is the registry's own shape
(FR-1). It is built beside `LoadUnits` in `NewFrontEnd` and assigned onto the
mission driver in `openMission`, which is how `figures` and `spellNames` already
reach it — the constructor's parameter list is at seven and an eighth would be a
field every test fixture has to state. A driver holding no bundle draws no spell
art, which is every hand-assembled front end in the tree (P-4).

**DD-4 — the sheet is decoded at load, not at first draw.** The original loads a
projectile sheet lazily, on its first draw. This build loads all of them with the
registry, for the reason `LoadUnits` does: the load is headless and GPU-free, so
a `-check` run reaches every sheet with no graphics context, and a decode on the
first frame of a cast would be the one frame where a spell stutters. Cost is
bounded — 24 sheets, of which the seven flying and eight burst rows are what a
cast can reach. *Rejected:* loading only the 23 spell-reachable rows, which would
put the spell-to-picture join inside the loader and make the bundle's contents
depend on a table the loader does not read.

**DD-5 — a row this build cannot draw is a skip, never an error (FR-5).** An
absent entry, a stream the decoder refuses, a row with `A16` at 0 and a row whose
sheet declares no palette each leave that picture without a sheet. Only the
registry itself fails the load, exactly as `LoadUnits` treats a class it cannot
draw. On the shipped registry the `A16` skip is three rows and each is an arrow
no cast can name.

**DD-6 — the palette is the sheet's own, and a row that has none is skipped.**
Every one of the 23 spell-reachable rows sets `Palette`, so the shared
`projectiles.pal` arm is unreachable from a cast. Building it would be a code
path no shipped data takes and no test could witness against a real install.
Disclosed rather than built (FR-5).

**DD-7 — the flight length is the picture's, and the swing is the caster's.**
0154 gave a bolt the caster's own attack charge as its span so that one
projectile covered one swing. The picture's length is decoded and the swing's is
not, so the two come apart here: the bolt takes FR-3's length and
`startCastRun` keeps handing the swing the charge it always did. That is the
smaller change of the two available — the alternative is scaling the caster's
animation to the projectile, which would alter a picture the owner has already
accepted and which no claim asks for. A Fire Arrow across a long room therefore
outlives its caster's swing, and that is the original's behaviour.

**DD-8 — the distance is an integer square root of the cell delta scaled by
256.** The flight table divides a distance term the engine produces with `ftol`,
whose rounding is not published. `dist = 256 * isqrt(dx*dx + dy*dy)` truncates,
so a diagonal crossing is at most one tick shorter than a real-valued term would
give. `pkg/game` is outside the determinism wall and could use a float; an
integer root is used anyway, so that the one arithmetic a player can see is
reproducible on any machine.

**DD-9 — the 16-way facing is an angle split, not the eight-way sign table.** The
simulation's own facing is eight-way and derived from the sign of a one-cell
step. A bolt crosses an arbitrary delta — five cells east and one north is an
ordinary cast — and the sign table would call that north-east and draw the sprite
45 degrees off. The split compares the along and across components against the
four tangent boundaries at 11.25, 33.75, 56.25 and 78.75 degrees, scaled by
10000, in integers. The wheel it lands on is not ours: it is `sheetOctant`'s
eight-way sheet ordering doubled, so facing 0 is south and the ordering runs
clockwise, which is the space `Flip`'s fold and `Phases * facing + phase` are
defined in (FR-6).

**DD-10 — the selection functions live beside `SelectUnitFrame`.** FR-6's index,
FR-7's clock and DD-9's facing are pure integer functions over a descriptor, an
age and a delta. They go in `pkg/render/terrain` beside the unit selectors for
that tier's own reason: a selector that reads nothing but its arguments is
reachable from a test with no archive, no window and no world (P-1, P-3).

**DD-11 — a refused selection answers "no frame" rather than frame 0.**
`SelectUnitFrame` answers a refused index as frame 0 unmirrored, because a unit
that draws nothing is worse than a unit drawing its first frame. A projectile is
the opposite: the original's own draw refuses a picture past the array and an
empty slot by drawing nothing, and FR-2 makes drawing nothing the normal outcome
for 21 of the 28 spells. So the selector returns a third value and every refusal
— no sheet, no frames, `Phases` not positive, an index outside the sheet —
reaches it (AC-10).

**DD-12 — the burst is a second kind on the same memory, not a second list.** A
burst and a bolt differ in three values: whether the position moves, which
picture, and how long. `spellBolt` therefore keeps one struct with a picture, a
life and a from/to pair whose two ends are equal for a burst, and
`advanceBolts` ages both. A second slice would need a second advance, a second
compaction and a second push, all to express "the two ends are the same cell"
(FR-9).

**DD-13 — the burst is spawned by the cast that lands, and its life is the
sender's.** This build has no area delivery, so the only place a burst can come
from is the observation that a cast was applied. The object it builds is the
decoded one: stationary, at the target's cell, 16 ticks, 18 for picture 27, 22
for picture 13. The trigger is authored and the object is not (FR-9).

**DD-14 — the seam carries a sheet, an index, a mirror bit and a point.**
`ui.SpellBolt` keeps `Cell`, `Pos` and `Owner` — the relief key, the fixed-point
position and the fog key it already had — loses `School` and `Burst`, and gains
`Sheet *terrain.EffectSheet`, `Frame int` and `Mirror bool`. `School` goes because
a sheet carries its own colours; `Burst` goes because the ring it stepped is gone.
The drawing tier still derives nothing (P-2, FR-12).

**DD-15 — the sprite pass rides `placeArm`.** A bolt's rectangle is its frame's
size at the object's point less FR-8's halves, and `placeArm` already applies the
displaced mode's per-cell lift, the camera and the view cull to a world rectangle
on a cell. Reusing it is what keeps a bolt on the same relief as the caster's own
mark rather than on a second geometry. The texture is cached under the frame
pointer, `staticImage`'s own shape, so a sheet frame drawn on twenty ticks
uploads once (FR-12, P-4).

**DD-16 — the pass is drawn immediately after the content band.** Spell art is
game content, so it goes on with the terrain, the objects and the units and under
every instrument — the same place `drawArt` sits, one call later, so a bolt
crosses in front of the units it flies over and under the health bars and
numerals that measure them (FR-12).

**DD-17 — teleport's second object is a second entry, spawned once.** FR-10's
copy is built at the same observation as the first, with both ends of its
from/to pair at the caster's cell, so it stands still and runs the same clock
through the same code. Nothing special-cases it downstream.

**DD-18 — nothing in `pkg/sim` is touched.** No field, no rule, no constant and
no byte-form version. `formatVersion` stays at 46, and this story's whole effect
is between an observation the step already reports and a texture (P-5).

## Traceability

| FR | Decisions | Witnessed by |
|---|---|---|
| FR-1 | DD-3 | AC-1 |
| FR-2 | DD-3 | AC-2 |
| FR-3 | DD-7, DD-8 | AC-3, AC-4 |
| FR-4 | DD-7, DD-8 | AC-5 |
| FR-5 | DD-1, DD-2, DD-4, DD-5, DD-6 | AC-10, P-3 |
| FR-6 | DD-9, DD-10, DD-11 | AC-6 |
| FR-7 | DD-10, DD-11 | AC-7 |
| FR-8 | DD-15 | AC-11 |
| FR-9 | DD-12, DD-13 | AC-8 |
| FR-10 | DD-17 | AC-9 |
| FR-11 | DD-14 | AC-4, P-4 |
| FR-12 | DD-14, DD-15, DD-16 | AC-11, P-2 |

The cut list reaches no decision above, because each entry is something not
built: `SC-1` the smoke trail, `SC-2` the burst sound, `SC-3` the ramp two
pictures take their phase from, `SC-4` the rows in the other sprite format,
`SC-5` the muzzle table, `SC-6` area delivery, `SC-7` the unit shot's use of the
same art. `P-1` is a property of DD-10's functions and `P-5` of DD-18.
