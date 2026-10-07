# 1001 — Spell effects: canonical specification

## Result and owners

An installed spell row is immutable data. Simulation owns cast admission, action timing, facing,
mana, training, ordinary application, actor effects, area effects, passability and semantic cast
events. The client consumes those facts for animation, spell art, the spellbook, bars and character
dolls. It does not maintain a second combat result, cooldown, effect duration or spell-power
formula.

One ordinary application function owns damage, healing and every decoded non-damage arm. A direct
cast calls it once. An area effect calls it once for each decoded unit visit. Script casts and an
equipped weapon's spell enter the same application logic under their own source gates; they do not
gain a private damage or status implementation.

## Installed spell data

The spell collection supplies, per row, id, mana cost, Sphere, maximum range, damage interval,
target kind, defensive and restorative flags, distribution, radius, area duration, spell duration
and the first `Effects` record. The first effect record carries its kind, magnitude, mode and
duration. Stone Curse's second textual record is retained in the source row but is not applied.
Singular rules select by installed spell id only where the original arm itself does so.

The runtime table is a copy in installed order. Duplicate or zero ids, negative mana and negative
loaded damage are refused. School, range, distribution, radius and effect widths remain explicit at
the data-to-simulation seam. No shipped row, sprite, save or map is embedded in source or tests.

Spell power is `clamp(skill[Sphere] + Mind - 30, 0, 100)`. Ordinary range is installed maximum
range plus `power / 30`; a maximum of zero receives no bonus. Teleport alone adds `power / 3`.
Range uses whole-cell Chebyshev distance. The popup and cast admission use these same functions.

Damage first scales both installed endpoints by `(power + 30) / 30`. The simulation rolls once
inside the resulting interval. Elemental protection reduces a spell of schools one through five
with the decoded rounded percentage expression. Spell damage does not roll to hit and does not read
physical absorption. Heal uses the same scaled roll as a positive health delta, clamped to maximum
health. Drain Life heals the caster by the positive resisted amount removed from the target, up to
the caster's maximum.

## Actor action and cast admission

An actor owns at most one action timeline. Physical attack, weapon-spell attack, manual book cast,
offensive autocast and idle Heal all ask one busy predicate. Approaching an admitted attack,
physical charge and recovery, a pending book cast, and book recovery are busy. A refused competitor
does not change the actor's target or order, turn it, spend mana, train, create an event, create a
projectile or start recovery. No refused action is queued; all candidates are recomputed from live
state when the actor is idle again.

A book cast has these phases:

1. Admission validates the actor, installed and known spell, target kind, spell-specific target
   condition, current actor perception, range, action idleness and affordable mana. It records one
   pending cast but pays nothing.
2. The actor turns once toward the admitted target cell. Unit spells use the target's cell at
   admission; place spells use the requested cell. A same-cell or self cast retains the previous
   facing. The same octant feeds directional area geometry.
3. Wind-up lasts the actor's equipped attack-charge value, floored at eight ticks and narrowed to
   the canonical byte. Repeated commands and autocast scans cannot create another pending cast.
4. Release revalidates target existence and liveness, useful Heal, current perception, current
   range, place validity and affordability. Failure removes the pending cast with no cost or
   recovery. Success pays once, applies once, emits one semantic cast event and enters the actor's
   equipped recovery interval.
5. A long-lived client projectile does not extend actor recovery. It also cannot restart the
   actor's animation or repeat release. A later action may begin only after simulation recovery.

A row's own target condition is asked at admission and again before the cost: Control Spirit
requires a corpse at the bones stage and a loaded `Ghost` template, Teleport requires an open
destination cell, and any timed row requires a computed duration above zero. `BookSpellRefusal`
answers with the same predicate, so the read-only instrument and the cast agree.

Current perception is actor-local and is rebuilt from the simulation sight predicate. Entity casts
require the target unit's current cell to be visible. Place and area casts require the target cell
to be visible. The check runs at admission and at release. A target that becomes hidden before
release cancels cleanly. Release commits the projectile/effect once; loss of visibility afterwards
does not cancel, refund, retarget or reapply it. While the target stays hidden it is absent from new
manual, automatic, AI and weapon-spell admissions. Script-created temporary casts are not
actor-directed and retain their script contract. Client drawing remains fog-gated even when the
committed simulation result continues unseen.

An attack order on a weapon-spell carrier is an ordinary attack order: the victim is written and
the approach begins on the same terms as for a plain fighter. Actor-local perception gates the
weapon-borne release, not the order. A release whose target the actor cannot currently see applies
nothing, spends nothing and leaves the attack cycle to turn over.

A weapon-borne release applies through the same ordinary and area application a book cast reaches,
selected by the row's own shape rather than by whether it deals damage. A non-damaging row lands its
effect, and a row whose target kind is not a unit is not refused on that account. Two rows are
refused outright at a weapon or item release: Prismatic Spray, which does not travel as a weapon
spell, and Control Spirit, whose own arm removes the target entity and appends a new one while a
release hands its caller live indices that the caller resumes using (`DIV-056`). An Area row released
from a weapon lands at the struck target's own cell with the releasing actor's current facing fixed,
which is the unit-target book-cast convention carried over rather than separately decoded
(`DIV-057`).

A carrier with no mana pool does not replace its attack with a release. Its weapon's spell is a
RIDER on the ordinary blow: it fires from the blow's tail after damage has landed, so an absorbed or
refused blow fires nothing, and it asks no liveness of the victim. The two arms are exact negations
of one another on the same predicate, so a single strike can never take both. A rider trains
nothing, as no weapon or item spell does.

A successful admission turns toward one of the established eight facing octants before animation.
Target movement during wind-up does not repeatedly turn the caster. Release still validates the
target's live cell for visibility and range; the admitted facing remains fixed for that action.

## Target selection and autocast

Right-clicking a known spell's live spellbook cell submits the autocast command for that row.
Right-clicking it again clears the selection. A left click continues to use manual casting. A wrong
button, empty cell, unknown spell or modal path outside the active book submits no toggle. The
selected cell draws a dashed border whose phase advances with client time and stays keyed to the
spell cell across ordinary redraw and popup interaction.

Automatic selection first filters by the row's target kind and spell condition, then current
perception, then range. It considers every valid candidate rather than stopping at the first invalid
or full-health unit. Within a priority tier it selects the shortest decoded distance, then stable
entity id. Offensive spells use permitted living targets. Area rows target a visible permitted cell
through the row's place path.

Heal admits only a living, non-hostile target with maximum health above zero and current health
strictly below it. A full-health, dead, incapacitated, hostile, hidden or out-of-range target causes
no action side effect. The condition is checked again at release. A target healed to full during
wind-up therefore costs nothing.

An idle mage who knows a restorative row may scan without an armed autocast only when it has no
active hostile engagement, attack, cast, movement or script order. Candidate tiers are the caster's
own owner, allied owners, then neutral owners. Distance and entity id order each tier. This idle
Heal keeps one quarter of the caster's maximum mana in reserve. An explicitly armed Heal is the
player's order and does not use that reserve. An active scripted or player order remains attached;
idle healing never consumes it.

At one decision point explicit armed Heal is considered first. Out of combat, unarmed idle Heal is
next. The remaining armed row is last.

The actor-action admission guard governs the start of a spell. A manual cast, an armed autocast, an
idle Heal, an AI cast and a weapon-borne release cannot begin while another cast owns the actor or
while a physical or weapon attack cycle is loaded. A competing attack order is refused only while a
cast owns the actor; against another attack order the established cycle rule applies, so a
different victim resets the cycle and the same victim keeps its timeline.

A movement order is never refused for busyness. A move from the player, from a group command, from
a script or from the AI ends whatever fight the ordered unit was in and attaches its destination,
whether the unit is approaching, charging, relaxing or carrying a book cast. A pending cast is not
interruptible: the body does not advance while the cast winds up, the cast then applies or refuses
on its own terms, and the walk begins. The move spends nothing and refunds nothing.

A move or group-move order that changes an entity's target also drops that entity's stored route,
when the new target differs from the one already held. A non-empty stored route's last cell equals
the entity's target at every tick boundary, which the byte form requires. An ordinary mover's own
walk rebuilds an equivalent route on the same tick, but an actor whose walk does not advance that
tick — a pending cast's wind-up, or an owed transit crossing — would otherwise end the tick holding
a route that still ends at the previous target beside the newly written one, a state the byte form
refuses. A re-issued order naming the same target keeps its route.

A movement order does not suppress unbidden casting on the actor it orders. An armed autocast is
the player's own instruction and reaches a unit that is walking, on the tick the order arrives and
on every tick after it. The unbidden, unarmed Heal is the one row a movement order does keep off
the actor, because a unit holding a destination and no victim is under command.

A group order addresses every living member the command names. The centroid, the formation test,
the per-member offsets and the group rate term are computed over that addressed set. A member that
is fighting, approaching, charging, relaxing or casting is neither dropped from the set nor
excluded from the computation. The same rule holds for the script's own group march.

Automatic selection never picks Teleport, and a stored autocast naming it is cleared. A
player-issued Teleport is unaffected.

A member that begins a cast on a decision pass stays in its group's member list. The group's shared
sight stamp, its guard centroid and its notice circle are computed over every member the group
holds, and a group in which every member casts is still decided for.

A map-owned mage may begin one cast per decision pass. The candidate rows are those it knows, that
are applicable, that are not defensive and whose mana cost it can pay; one is chosen uniformly and
aimed by the ordinary automatic target selection. A caster of Mind above 59 hands the decision back
to the ordinary group arm on about three passes in ten. The cast is admitted through the same guard
as every other, so it cannot overlap an attack cycle or another cast. The player's own and unowned
actors do not use this arm.

## Ordinary point effects

Lasting effects are canonical records sorted by target and spell. One target carries at most one
record of one spell id. Reapplying a duration effect unapplies its old magnitude, replaces its
magnitude and duration, and applies the new value. Reapplying a continuous effect refreshes only its
duration. Bless and Curse annihilate: applying one to a target carrying the other removes the old
effect and drops the new one. Untimed effects apply once and are not retained.

Duration and continuous records apply once on attachment. Continuous health effects apply again
when remaining duration reaches an eighth-tick boundary. A retained duration above 9,600 does not
count down. Ordinary expiry unapplies a non-continuous magnitude and removes presentation feedback.
The canonical record stores target, optional caster, spell, kind, mode, magnitude and remaining
ticks.

Expiry reverses exactly what the application landed. Where a clamp meant less than the nominal
magnitude was applied, the stored magnitude is what was applied, so no effect can rewrite an
actor's base protection or scan range. A reversal that takes an actor below zero health fells it
through the ordinary path.

Where more than one effect of a kind stands on an actor and a clamp keeps one record's own removal
from delivering its full reversal, the undelivered remainder moves onto a still-active record of the
same kind rather than being discarded, so the group's combined total stays exactly recoverable. A
fresh recompute — the derived speed and scan-range lines, and a recompute's protection block —
applies the same rule to its own clamp against the fresh base it is handed. Health is excluded: its
ceiling is maximum health, reachable by healing unrelated to any effect, so there is no effect-free
base to redistribute a clamped health reversal against.

An effect is untimed when none of the duration, continuous and charges bits is set. An untimed
effect is applied once and stored nowhere; every other mode is stored and counted down.

An effect may not take an actor's speed below 1.

The installed arms are:

- four Protections add `power / 2` to their elemental protection;
- Shield adds `power / 10 + 3` absorption;
- Haste adds and Slow subtracts `power / 15 + 1` speed;
- Freezing Cloud subtracts the same quantity when its ordinary inner effect is applied;
- Poison Cloud scales the installed health magnitude by `(power + 30) / 30` and repeats as the
  installed continuous effect requires;
- Light adds `power / 30 + 1` scan range and Darkness subtracts `power / 30 + 1`;
- Stone Curse adds the absorption its own installed effect record names, which ships as five. Its
  duration is first power-scaled, then shortened by the target's Earth protection percentage, with
  a one-tick floor;
- Bless and Curse retain probability `(power * 4) / 5 + 20`. On a physical damage roll an inclusive
  draw from 0 through 100 selects the maximum for Bless or minimum for Curse when the probability is
  greater than the draw;
- Invisibility uses its singular fast duration, removes the affected other actor from new entity
  target selection, and is removed by its owner's next cast at another actor or by the owner's
  physical attack approach. A currently visible invisible target survives entity filtering only
  when the observing actor is within that actor's installed whole-cell Chebyshev See-invisible
  radius. Group acquisition accepts the target when any member is within its own detector radius;
- Prismatic Spray performs a complete ordinary application on every living actor hostile to the
  caster, in radius `min(power / 20 + 2, 7)`, and does not travel as a weapon spell: a weapon or
  item release refuses it outright (`MAGIC-SING-019` (g)). Hostility is the same relation an
  offensive row's autocast target selection already asks: the caster himself, his allies and any
  neutral actor in radius take no application. This is a deliberate difference from the original,
  whose own multi-target applier is undiscriminating (`MAGIC-TARGET-017`, `DIV-072`). A
  script-triggered cast, which has no caster, keeps the unfiltered radius apply it always had. The
  ordinary application refuses a target that is not alive, on every arm and not only this one;
- Fire Sacrifice derives its inner fire damage from caster health, mana and power, then leaves the
  caster at one health and zero mana;
- Teleport moves the caster to an admitted open target cell and clears its route;
- Control Spirit consumes a bones-stage corpse and creates one owned actor built from the installed
  `Units` row named `Ghost`. That row supplies the class key, movement domain, speed, sight, reach,
  token size, dying time, experience value, protections and combat block, with the mission's
  difficulty applied. Four values come off the corpse: `corpse.Reaction / 2 + 1`, the corpse's Mind
  and Spirit verbatim, and half the corpse's maximum health. Reaction, Mind and Spirit are
  canonical source state; no value is reconstructed from Speed or protection. A world carrying no
  template refuses the cast at admission, so nothing is spent; and
- Wall of Earth applies no unit effect.

Health, speed, scan range, absorption and protection changes are immediately visible to their
ordinary consumers. Bless and Curse are consumed by the physical damage roll. Attached effect state
also drives the client mark for its real remaining duration.

## Area effects

An area record stores anchor cell, spell, optional caster, power, mode, stage phase, direction,
singular damage and the accepted cell list. It is ordered canonically by anchor and arrival. At most
six records may share one anchor. Script instant 21 creates through the same landing path; instant 29
retimes every matching record at its addressed cell.

An area row's per-tick mode comes from its own columns. `Distribution system = 5` selects staged
mode and leaves the duration counter at zero, so a staged row's `Area Effect Duaration` has no
effect; otherwise a positive `Area Effect Duaration` selects cloud mode and a zero one selects
blast. A staged row's whole life is its own cadence, `(stages - 1) * 3 + 1` ticks, and that is also
the duration the spellbook popup reports for it.

Blast applies once over the `(2r + 1)` square and retains no area record. Fire Ball alone divides
each scaled damage endpoint by the target footprint side squared, then performs the ordinary direct
application once for every covered cell that finds that actor. Partial coverage therefore gives a
partial total. A covered cell's occupants are keyed by movement domain into a
ground-and-spirit group and a flyer group, plus a third, structure group that this build's entities
never occupy. The decoded record caps each group at one actor, but that cap depends on an invariant
(`TERR-FOOTPRINT-147`) this build's occupancy plane does not maintain: movement occupancy is
anchor-only and a Teleport landing tests only terrain, so two ground actors can cover one cell.
Application reads every actor of a group and not the first one found (`DIV-054`, `DIV-055`). Blast
and staged modes read all three groups in that order; a cloud pulse reads the ground-and-spirit
group alone. A body occupies its group while it lies where it fell and none once it has begun to
decay.

Cloud mode paints its decoded cells and retains the record for
`(AreaDuration << 4) + (power << 4) / 10 + 1` ticks when the base duration is positive. Its counter
starts at `V0 = (AreaDuration << 4) + (power << 4) / 10`, falls by one each tick, and pulses when
the new value is a positive multiple of 16, so a cloud whose `V0` is not a multiple of 16 pulses
off the sixteenth tick. Each pulse reads the covered cell's ground-and-spirit group alone, so it
reaches every ground or spirit actor covering the cell and no flyer, and it applies to friends as
well as hostiles. Each pulse calls the ordinary inner effect; continuous and same-id non-stacking remain
properties of that ordinary owner.

Wall of Fire and Wall of Earth are the two `Distribution system = 4` rows and use the same two cell
tables, selected by the cast's bearing truncated to one of eight. The four axis bearings take the
10-cell table, a 5-by-2 block; the four diagonal bearings take the 9-cell table, an anti-diagonal
of five plus the same line shifted by one cell in x, spanning two cells either side of the target.
A wall's cell count does not come from its radius column.

The painted shape is the row's `Distribution system` column and not its spell id: 4 takes the wall
tables and 3 takes the diamond walk. The diamond walk runs both offsets over `[-r, +r]` and keeps a
cell when `|dx| + |dy| <= r + 1`, so the four axis cells at distance `r + 1` are outside the walk
and are not painted: 9 cells at radius 1, 21 at radius 2, 57 at radius 4.

Ring mode executes stage zero at landing and later stages at ticks 3, 6, 9 and so on.

Every ring row's stage offsets are relative to its own anchor cell, and the anchor is the aimed
cell for every row, Acid Stream included. Fire Sacrifice's own row carries a maximum range of 0,
the column Shield also carries 0 on, so the admission gate accepts only an aim at the caster's own
cell, and the anchor and the aimed cell coincide there regardless. Acid Stream's own row carries a
maximum range of 3, a real reach, and is a cone the caster throws: as built, its anchor is the
aimed cell, matching every other row, and the aimed cell also supplies orientation, through the
same delta from the caster (`DIV-063`). Meteor Storm's anchor is the aimed cell.

Fire Sacrifice has two fixed orientation-independent stages. Relative to the target, stage zero visits
`(-1,1) (-1,0) (-1,-1) (0,1) (0,-1) (1,1) (1,0) (1,-1)` and stage one visits
`(-2,1) (-2,0) (-2,-1) (-1,2) (0,2) (1,2) (-1,-2) (0,-2) (1,-2) (2,1) (2,0) (2,-1)`.

Acid Stream has six stages and orientation `o = facing >> 5`. For even orientations, stage `s`
starts from `(x,s)` for `x = -s..s` at stages zero through four and stage five is empty. For odd
orientations, stage `s` starts from `(s-i,i)` for `i = 0..s`. Orientation zero and one transform
to `(+x,-y)`, two to `(+y,+x)`, three to `(+x,+y)`, four and five to `(-x,+y)`, six to
`(-y,+x)`, and seven to `(-x,-y)`. The empty even final stage is still terminal. Meteor Storm has
32 stages, each consuming two world RNG draws in x-then-y order for independent offsets in
`[-2, 3]`; cells may repeat. Ring coordinates narrow to bytes and are accepted only when both lie
between eight and the corresponding map dimension minus nine, inclusive. Each accepted cell walks
its occupant groups in group order, applying every actor a group holds. Cells are applied in the generator's own visit order; the record
stores the same cells in ascending key order, which is the byte form's own order. The terminal
stage removes the area record on the same tick; staged effects register no retained map layer.

The six retained map layers belong to Wall of Fire, Freezing Cloud, Poison Cloud, Wall of Earth,
Light and Darkness. Same-layer replacement does not stack. Fire Ball and Wall of Fire remove
overlapping Poison and Freezing layers. Freezing removes Fire. Poison placed in an existing Fire
cell is omitted there. Light and Darkness replace either light layer. Removing or expiring a layer
does not revoke actor effects already attached by earlier pulses; those expire under their own
duration.

Darkness and Light use opposite scan-range magnitudes. Darkness additionally provokes its affected
victim toward the caster; Light does not. Neither changes the world's lighting plane.

Wall of Earth refuses a cell whose ground-and-spirit group is non-empty — any cell of a ground or
spirit actor's footprint, including a body lying where it fell — harms no unit, and marks every accepted
cell in the ordinary passability plane. A stored route crossing a cell the wall has just closed is
discarded at the landing, and its mover routes again on its next tick. Ground and spirit movement read the derived block; air movement does
not. Removal, conflict removal and script retiming to expiry clear the same cells and restore route
availability. Route search contains no spell-id branch.

Where a cast is refused, nothing on the anchor changes: no cell is painted, no overlapping record is
shortened, and no record is removed. At a full cell this refuses a row whose own older record would
otherwise have been displaced to make room.

The original's adjacent per-layer multiplication of the mutable movement-cost byte is not
implemented. It is not used to emulate Wall of Earth passability and remains one explicit fidelity
debt.

## Training and immediate recomputation

A successful use is a spellbook application that reaches the ordinary apply path. Admission alone
is not use. A refusal, a release-time cancellation and an effect that cannot apply spend no mana,
start no recovery and award no training. Item and weapon spells do not train. Non-hero actors do not
train.

That covers the cell form as well as the unit form. An area landing is refused before its cost when
the row selects a staged program its spell id has none of, when its computed lifetime is zero, or
when the target cell already holds the six records one cell carries. The admission pre-check and
the landing test the same cell for every row, Acid Stream included, since every row's anchor is
the aimed cell (`DIV-064`, closed). The same three refusals are
asked at admission, at release, and by the read-only instrument, so a cast the instrument calls
admissible is one the release will land. A blast is not held by a full cell, because it applies once
and retains no record. Fire Sacrifice's payment of the caster's health and mana happens after those
refusals, not before them.

Each successful book application awards `(manaCost + 1) / 2` to the installed row's Sphere. The
existing skill sink applies its human, mage, level, participant and cap gates. A book cast supplies
no source entity, so the sink's same-owner and locked-relation refusals do not apply to it: an idle
mage that heals an ally of its own team trains its school for doing so. That is the owner's default
(2026-08-15) and it becomes an in-game "auto-heal" setting in a later story.
Area spells award once per complete ordinary unit application, including later pulses and Fire Ball
footprint visits. A scan tick with no application awards nothing.

Per-skill experience, stored skill level and total experience are distinct. The total is the sum of
the six stored experience slots and is maintained by the skill sink. One award raises at most one
level and no slot grows beyond its decoded cap. On a rise, the full hero sheet is recomputed in the
same release: health and mana maxima, sight, damage, to-hit, defence, absorption, protections,
resistances, attack timings and spell power. Current pools are preserved and clamped to changed
maxima; a level does not refill them. Subsequent decisions, hashing, persistence and presentation
observe the recomputed values in that same simulation state.

## Presentation

One successful release emits one cast event. The caster's Attack animation starts once at projectile
start. A client projectile has its own flight clock and continues without restarting the actor.
Instant and area casts do not borrow an inappropriate repeated swing. Cast and effect presentation
are gated by the current player's fog without changing simulation.

Lasting point feedback reads attached records and remaining duration. Area feedback reads active
area records, painted cells, phase and layer. Removing or expiring the underlying record removes the
feedback. No decorative client timer substitutes for simulation state.

A positive Heal emits one semantic feedback event anchored at the healed target's application
cell. The client resolves picture 20 through the installed projectile registry to
`projectiles/healing/sprites.16a`. Its ordinary travelling-projectile arm remains suppressed. One
event creates one burst of seven sprite instances. Instances use the seven registry-accessible
phases of the eight-frame 12-by-12 sheet, preserve embedded alpha, palette and 16-by-16 centring,
advance on the two-tick sheet clock with deterministic phase offsets, rise from fixed offsets around
the target anchor and expire. Arrangement and rise are client-authored. No simulation RNG, hash or
save byte is consumed. A full-health or refused Heal emits no burst. Fog, camera and viewport
clipping apply to each instance; removal of the target cannot leave a live pointer.

The character sheet's five protection rows state the live value. They are overlaid from the entity
of the tick, beside the Experience and Skills rows already taken from it, in the Fire, Water, Air,
Earth, Astral column order both arrays use. A Protection effect is therefore visible on the sheet
while it stands and gone when it expires.

Area-effect sprites are placed by the same fixed-point scale as every other spell object: one cell
is `ShotScale` units on each axis. A cell's sprite stands on that cell.

World-unit health is a clamped bar above the unit's rendered selection square. A mage with a positive
maximum mana has a blue mana bar immediately below health while both bars remain above the square.
Non-mages draw no empty mana bar. The two bars use the unit's camera transform, terrain lift,
footprint and visibility, track movement, clip at the viewport and do not replace selection or
status overlays.

Drawing the bars above a unit's own cell places them inside the cell of the unit to the north. They
stay clear of every glyph of both cells except the northern cell's selection rim, which the mana bar
crosses, and they stay far enough from the northern unit's own bars that neither can be read as the
other's.

The spellbook popup obtains one current `SpellCharacteristics` projection from the selected actor
and immutable row each time it paints. It may cache immutable icon and row art, never power, range,
scaled damage or healing, duration, radius, skill level or skill experience. A skill raise, Mind or
effect change, equipment recompute, original-save load or selected-member switch is visible without
closing the popup. Flat mana cost stays flat.

## Original saves, mission 40 and dolls

Original-save restoration is generic. It decodes every supported persistent party member before
party filtering, binds a stable identity, restores pools, class and stat sheet, six skill levels,
six per-skill experience values and their summed total, spellbook state where supported, worn slots
and carried container items. It then constructs entities through the same equipment and derived-stat
owners used by a fresh mission. Duplicate stable references do not create duplicate members; a
record outside the decoded persistence rule is not injected.

The mission-40 Reniesta binding is stable across original-save resume and fresh construction. Her
equipped staff resolves its live weapon spell and scaled damage before both simulation and tooltip
projection. Her death through the ordinary live death path sets mission loss. A controlled unrelated
party death does not.

Brian is restored as the non-primary persistent companion named by the save identity, without a
name-based exception. His displayed total comes from the decoded six experience slots. Every worn
and carried item present in the supported participant subtree is retained. The owner save contains
seven worn codes and no backpack codes for Brian; the implementation does not invent absent items.

Character dolls read the live worn array. Weapon and shield are resolved independently through the
same figure-layer compositor as ordinary party, dialogue and shop figures. A class name or mercenary
label never paints equipment that is absent from simulation. Equipment change and original-save
construction invalidate the composed result through the existing live projection.

## Persistence and customisation

Canonical format version 53 carries pending book casts, attached actor effects and area records,
including phase, direction, source, magnitude and accepted cells. Entity records carry recovery,
facing, skills, per-skill experience, Reaction, Mind, Spirit, protection, footprint and the
See-invisible detector radius. Total experience remains the sum of the six canonical experience
slots. The `Ghost` template is install-derived input carried on the world and is **not** in the byte
form: a decode keeps the receiving world's own template, so a resumed mission raises from the
template its start loaded and a decode into a fresh world refuses the raise at admission. Heal
particles and client projectile frames are presentation and are not serialized.
Round-trip bytes and hash are identical at mid-wind-up, mid-recovery, attached-effect and
area-phase states.

Installed rows remain the customisation seam for spell count and all row values. The fixed six area
layers, six records per anchor, byte spell/stage/direction coordinates, word duration, eight
orientations, fixed ring-program capacities and canonical serialized widths are engine limits.
Changing the installed rows changes no shipped file; lifting a compiled or serialized width changes
engine representation but does not require rewriting a shipped asset. Replacement ring geometry
may be externalized later without altering the preserved installed files.
