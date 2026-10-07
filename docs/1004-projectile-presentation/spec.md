# 1004 — projectile presentation, as built

Self-contained. Claim provenance is in `contract.md`.

## FR-1 A path picture stands still and draws no sprite at its own position

Picture 34 (Lightning) and picture 36 (Prismatic Spray) put an object on the map for 13 ticks. The
object's position is never written: it holds the cell the cast was observed from for its whole
life, and its 13 ticks are a countdown.

The draw for these two pictures is the generated figure alone. No interpolated sprite is submitted
at the object's own point, and nothing is created when the object ends.

`data.CastDrawsPath` names the two pictures. `data.CastTravels` names the two that do move, 10 and
12; the other five with a non-zero flight length stand still.

## FR-2 The figure is a bounded random walk spanning the whole segment

On every drawn tick the whole point list is regenerated. It runs from the caster's own departure
point (FR-12) to the target's cell, both in `ui.ShotScale` units, and it is never partial.

The walk is generated in a canonical horizontal frame and rotated onto the segment by the cosine
`dx/length` and the sine `dy/length`. Two accumulators, both integer:

- the **abscissa** advances by `rand()%50` per step, carried in hundredths, and the walk ends on
  the step that reaches 70. Abscissa 70 is the far endpoint, so the figure spans the segment;
- the **perpendicular** advances by `rand()%7` with a random sign and is clamped to ±3. Its unit is
  **three hundredths of the segment length per clamp unit**, so the widest excursion the clamp
  allows is 9 per cent of the segment and the figure's raggedness is proportional to the cast
  distance. An increment reaching 6 against a clamp of 3 is what makes the figure a zigzag rather
  than a drift.

The last point is forced to abscissa 70 with a perpendicular of 0, so the figure ends on the
target.

**Rejection.** Before rotation, a raw walk with any ordinate further than 0.15 of the segment length
from the straight line is thrown away whole and the walk re-entered with the same generator, at
most four times. Nothing is clamped to the bound. The bound and the perpendicular are both
fractions of the segment length, so the test bites at the same raggedness at every cast distance:
the clamp's 3 against the ordinate scale is 9 per cent of the segment against a bound of 15. The
test reads the unsmoothed walk, because smoothing (below) runs after rotation, in world-space,
where the clamp's small integers no longer apply.

**Smoothing.** After rotation onto the segment, the point list — the caster's own departure point
(FR-12), the rotated walk, and the target — is passed through two passes of Chaikin corner-cutting
in `ui.ShotScale` (world-space) units: each interior point is replaced by two points, one a quarter
of the way from it toward its predecessor and one a quarter of the way toward its successor, and the
two ends are carried through unchanged. Each pass at most doubles the interior point count. The pass
runs after rotation rather than on the canonical abscissa/ordinate pair the walk is generated in,
because the ordinate is clamped to ±3: a quarter-cut of an integer that small truncates to zero under
Go's integer division and would leave the walk unchanged. Rotated into world-space the same points
carry two to three digits, where the same integer cut moves them a visible amount. Because rotation
is an affine map, corner-cutting the rotated points is the same construction as corner-cutting the
canonical ones and then rotating the result; only the arithmetic precision differs. Chaikin's cut is
a convex combination of a point and each of its neighbours, so no cut point ever lies outside the
bounding box of the raw walk it was built from, and the rejection test above stays meaningful: a walk
that passed the 0.15 bound before smoothing cannot be moved outside a comparable bound by smoothing
alone.

**Spacing.** The walk ends after three to five points, which at a long cast leaves the stamped
sprites detached. Each gap is then subdivided evenly until no gap exceeds **a third of one stamp's
frame width**, at most 72 stamps per gap. Subdivision moves no existing point. The third is the art
and not a margin: the shipped `lightnin` glyph's opaque core is about 8 by 6 pixels inside its
16 by 16 frame, so stamps a full frame apart leave visible gaps between their cores.

**The generator** is MSVC's linear congruential generator, `state = state*214013 + 2531011` with
the result `(state>>16) & 0x7fff`. Its seed is a function of the cast observation — caster, target
and spell — mixed with the object's age and picture, so the figure is fresh on every tick, the same
observation redraws the same figure, and nothing here reads a clock or `math/rand`.

A zero-length segment answers one point.

## FR-3 The sheet is stamped at every point

Each point takes one stamp of the picture's own sheet, centred on the point by the sheet's
centring halves. Both sheets state 16x16 in the registry, so those halves are the engine's own
hard-coded 8.

The phase is a ramp over the object's 13 ticks: `age * ramp / 13`, clamped into the sheet, where
`ramp` is **five** for both pictures.

The ramp is five for picture 34 because its record declares five phases. It is five for picture 36
against a record that declares **thirty-five**: that record's sheet is seven colour blocks of five
frames each, and the declared count is the whole sheet rather than one block. Picture 36 adds
`tag * 5` to the ramp's phase, where `tag` is the object's own record tag in 0..6, so the tag
selects the block and the ramp walks the five frames inside it. Taking the ramp over the declared
35 instead walks a single figure through all seven colours across its own 13 ticks, and names frame
indices past the end of the sheet for every tag above zero.

This build's tag is the victim's own index in the cast's victim set (FR-10) where the cast reports
one, the cast observation's index inside its tick for any other book cast, and FR-9's for a
weapon-borne release.

## FR-4 A path picture leaves no trail

Neither path picture stamps a trail sheet.

## FR-5 A travelling picture leaves six past positions

Pictures 10 (Fire Arrow) and 12 (Fire Ball) stamp a trail sheet at up to six positions the object
previously held, behind its current one. The newest entry is where the object stood one tick ago,
the oldest where it stood six ticks ago; before the object has moved six times the queue is
shorter. The positions are recomputed from the object's own two ends and its age, not stored.

The frame is the entry's own age: 0 for the entry appended this tick and 5 for the one about to
expire. Each trail sheet holds exactly six frames against a queue bounded at exactly six, and the
art runs from a small dense puff at frame 0 to a large pale one at frame 5.

The trail is submitted after the object's own sprite, which is where the engine's default arm
walks it.

**The art** is `graphics/projectiles/smoke0/sprites.16a` and `smoke1/sprites.16a`, loaded by
constructed path. `projectiles.reg` names neither, so neither has a registry row: the phase count
is the sheet's own frame count, the rotation count is one, and the centring halves are the art's
own. Slot 0 belongs to picture 10 and slot 1 to picture 12. An install missing either entry draws
no trail for that picture and opens normally.

## FR-6 A burst waits for the cast object it belongs to

A burst spawned for a book cast whose cast picture travels is held back by that picture's own
drawn flight length: it is not drawn during the wait, and it then runs its own lifetime. A burst
whose cast picture does not travel, and a caster's replacement release, wait nothing. A rider
release's burst does wait (FR-8).

On the shipped book exactly one spell has both a travelling cast picture and a burst sheet: Fire
Ball, picture 12 flying and picture 13 exploding for 22 ticks.

## FR-7 A caster's replacement release draws what a book cast of the same picture draws

A spell released by a weapon is not an object in the client's cast list. It has no life of its own:
it is rebuilt from the attacker's attack cycle on every frame, from the attacker's cell to the
attack target's cell, for as long as `AttackPhase` is casting.

It is nevertheless drawn by the same two producers as a book cast of the same picture, and by no
others. Picture selection, the figure, the trail and the sheet phase all come out the same, so a
picture cannot be drawn one way from a book and another way from a staff. What differs is only
where the stamp comes from: the attacker's swing position within its attack charge, in place of the
object's own age within its 13 ticks. This arm's burst waits nothing (FR-6): the picture is
already at the victim on the release tick.

The seed is the attacker id, the attack target id and the weapon's spell id, through the same
mixing a book cast uses, so a staff redraws a fresh figure on each swing tick and the same swing
redraws the same figure.

Each install root carries 46 staff rows with a `castSpell`, over 8 distinct staff names, naming
four spells: Fire Arrow (22 rows), Lightning (14), Prismatic Spray (8) and Stone Curse (2). Three of
the four pictures this specification changes are reached from a staff. The fourth, Fire Ball, is not
carried on any staff: its two `castSpell` rows are unit templates, and they reach the other arm
(FR-8).

## FR-8 A rider release draws like a book cast

A weapon-borne release has two arms and they are chosen by the carrier, not by the item.

- The **caster's replacement release** belongs to a carrier with a mana pool. Its attack cycle
  enters the casting phase, so the release has a wind-up, and FR-7 draws it live off that wind-up.
- The **fighter's rider** belongs to a carrier without one. Its attack cycle never enters the
  casting phase, so there is no wind-up to draw across; the release applies at the blow.

A cast observation reports which arm it came from. Both arms report `Weapon`; only the rider
reports `Rider`. The flag is derived at the recorder from the same mana-pool predicate that chose
the arm, so the two cannot come apart.

A rider release spawns an ordinary cast object, exactly as a book cast does: it travels from the
carrier's cell to the target's cell over the picture's own flight length, leaves a trail if the
picture leaves one, and draws the figure if the picture draws one. Its burst waits for it (FR-6).

A rider release starts **no cast run**. A cast run is the caster's own animation, and a rider's
carrier is swinging a weapon: it is already playing its attack run, and replacing that with a cast
run would animate a blow as a spell.

The two cells a rider's object runs between are the carrier's own weapon reach, because the release
applies at the blow. A melee carrier's object crosses one cell; a siege carrier's crosses its
firing range.

Every campaign mission on both install roots places 79 actors carrying a weapon spell: 76 on the
caster's replacement arm and 3 on the rider arm. All three rider carriers carry Fire Ball, in
missions 90, 111 and 140.

## FR-9 A weapon-borne release carries a record tag

The second path picture adds a per-point phase of `tag * 5`, and its sheet holds seven blocks of
five phases. A book cast's tag is the victim's index in the cast's victim set where it reports one
(FR-10), and the observation's index inside its tick otherwise. A weapon-borne live draw holds no
observation, so its tag is the attacker's own entity id modulo seven: stable across one swing, and
different between two carriers on the map.

## FR-10 A multi-target cast draws one figure per victim

A cast that reaches more than one actor reports the cell of every actor it reached, and one figure
is drawn to each. Prismatic Spray is the shipped case: `applyPrismatic` returns the cell of every
LIVING actor its radius-and-hostility walk visited, in visit order, including one that resisted,
each cell read before the effect is applied. A corpse in the radius is skipped by the walk itself
and never enters the list: `docs/DIVERGENCES.md` DIV-072 records the exclusion, which matches the
sibling area collector a few lines above in the same file.

The list travels as `sim.CastEvent.Victims`, which is observation-only in the sense FR-P-1 states:
no tick, no byte form and no digest reads it. A script-triggered cast has no observation and reports
no list.

One figure is spawned per entry. Each runs from the caster's own departure point (FR-12) to its own
victim's cell and carries that victim's loop index as its `tag`, so the seven colour blocks of
picture 36's sheet are handed out in visit order and no two figures of one spray draw the same
colour until the eighth victim. Each figure's generator seed is offset by its index, so two victims
at similar bearings do not draw congruent walks.

A cast reporting an empty list draws exactly what it drew before: one figure to the directed target.

## FR-11 A figure's two ends take their own cells' relief

In the displaced projection every glyph carries a vertical lift for the cell it stands on. A figure
spans two cells, so it carries two: the caster cell's lift at its near end, the target cell's at its
far end, and a linear interpolation between them for every stamp in between, taken on the stamp's
projection onto the segment rather than on its own offset. A figure whose two cells are equal takes
one lift and the interpolation is skipped.

The interpolation is of the **lift** and not of the ground. The figure is a straight segment between
two points that already carry their height terms, so a bolt crosses a valley in a straight line
rather than dipping into it, and a stamp's height is a function of how far along the segment it
stands and of nothing else.

Lifting every stamp by the caster's cell instead puts the far end of a bolt as far above its target
as the caster stands above him, which on a cliff is the whole figure passing over the enemy's head.

## FR-12 A cast's departure point follows the caster's own facing

A figure's near end is not the caster's cell centre. It is offset from the centre by one of eight
fixed vectors, one per compass octant, selected by `sim.FacingDir(a.Facing)` at the tick the cast
was observed (a book cast) or drawn (a weapon-borne release). The offset is `ui.ShotScale / 4` along
the facing's own axis and `3/4` of that on each perpendicular axis for a diagonal facing, so the
departure point sits toward the caster's forward side rather than at his feet. The magnitude is
authored, not measured: DIV-083 records it as UNKNOWN against ROM1 and names what would settle it.

The offset is computed once, at the single seam where a cast's origin becomes a screen point
(`castOrigin`, consumed by `castShotPoint` and `boltPath`), so every picture that reads a figure's
or a burst's origin through that seam carries it, not only the two path pictures of FR-2. A
same-cell object — a burst or an area-paint cell, where `from == to` — is exempted explicitly: the
offset only ever applies to a figure that actually departs from the caster.

`sim.CastEvent` carries the caster's `Facing` at observation for a book cast; a weapon-borne live
draw (FR-7, FR-8) reads the attacker entity's own `Facing` directly, since it holds no observation.
Both of a bolt figure's two producers — the book-cast path (`spawnCastSet`/`spawnCast`, driven by
the tick's observed casts) and the weapon-borne live-draw path (`weaponBoltDraws`, rebuilt every
frame from the attack cycle) — carry a `facing` field through to `castOrigin` and `castShotPoint`.

## Non-functional

- **P-1** Nothing here reaches simulation state. The figure is not simulation state, is not
  serialized, and no tick, hash or digest reads it. `formatVersion` is unchanged. FR-10's victim
  list and FR-12's `Facing` are fields of `sim.CastEvent`, which is the observation record `pkg/sim`
  already keeps for this tier and which `Step` never reads.
- **P-2** The figure is deterministic in the observation and the age. Two runs of one replay draw
  one picture.
- **P-3** Every test fixture is synthetic; no test reads a game install.
