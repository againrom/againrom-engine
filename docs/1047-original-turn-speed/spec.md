# Spec — story 1047 original turn speed (as built)

This document describes the shipped behaviour. Research provenance is in
`contract.md`, not here.

## Outcome

An actor pays a turn interval before movement, attack or an admitted book
cast can use a new facing. The interval uses the actor's `RotationSpeed` at
request time. A turn of at most one 32-unit direction step snaps to the
desired facing and costs one actor tick. A larger turn stores
`ceil(shortestArc / RotationSpeed)` as its total duration, destroys the
stored movement route, and requires a new route search after the turn
completes.

## Canonical turn state

`sim.Entity` carries four related fields:

- `Facing uint8` — the current facing, in 32-unit steps over 256 (eight
  directions), unchanged in meaning from before this story.
- `DesiredFacing uint8` — the facing a turn in progress is moving toward,
  same quantum.
- `TurnRemaining uint8` — ticks left before `Facing` becomes `DesiredFacing`.
- `TurnTotal uint8` — the duration fixed when that turn was requested.

All four enter the hash and the byte form.

An **inactive** turn has `TurnRemaining == 0`, `TurnTotal == 0` and
`DesiredFacing == Facing`. An **active** turn has `TurnRemaining` and
`TurnTotal` in `1..128`, `TurnRemaining <= TurnTotal`, and `DesiredFacing` a
multiple of 32. An active turn whose current and desired facings are equal is
valid only for the one-tick snap state. Any other combination is malformed
and refused by construction and by decode (see Persistence below).

`Entity.Turning() bool` reports `TurnRemaining != 0`.

## Requesting and advancing a turn

`pkg/sim/facing.go` holds the one turn-request function every producer
calls (`requestFacing`, taking the entity and a desired facing already
computed by the caller) and the one per-tick advance function.

Given a desired facing:

1. If a turn is already active toward the same desired facing, the request
   is a no-op.
2. The shortest arc is `min(abs(current-desired), 256-abs(current-desired))`.
   Zero arc requires no turn; `DesiredFacing`, `TurnRemaining` and
   `TurnTotal` are left inactive.
3. `RotationSpeed <= 0` is the compatibility arm: `Facing` snaps to
   `desired` immediately, `DesiredFacing` is left equal to it, and
   `TurnRemaining` and `TurnTotal` stay 0. No interval and no route
   destruction occur.
   `mapload`'s `RotationSpeedBase` (`pkg/mapload/loadout.go:136`) resolves a
   positive rotation base for every construction arm this build reaches —
   map placement (resolved, creature and unresolved-loadout), fresh party
   spawn and restored party members. **This arm IS reached by measured
   shipped content, and round 3 corrects two places round 2 reached it in
   error**: a raised Control Spirit ghost dropped `RotationSpeed` to 0
   regardless of its Units row (P1, `pkg/mapload/ghost.go`), and a hired
   mercenary resolved a Humans row other than the one hired from for 36 of
   the 52 shipped `NPC%02d_%d` rows, some of which land on `RotationSpeed=0`
   rows (P2, `pkg/mapload/loadout.go`'s `RotationSpeedBase`). Both are fixed;
   see Persistence and the divergence rows below.

   Measured through the production mission door, not the `.alm` file
   (`cmd/rotationcensus`, both lawful roots, 28-mission campaign): 2361 of
   2361 live entities carry a positive `RotationSpeed`, 16 distinct values
   in 8..23, none at 0. `Bat_Sonic.2` (mission 100), the actor an earlier
   revision of this section named as reaching the compatibility arm, now
   loads at `RotationSpeed=21` and pays the interval below instead. **That
   census's own population is placed actors opened with the game's default,
   never-hired starting party** (`front.NextParty()`, one member, unhired) —
   it contains no hired actor and reads the world once at open, so a
   Control Spirit raise cannot appear in it at all. It does not corroborate
   the hired arm or the raise path, which round 2's closure had read it as
   doing.

   Two release-gated tests supply that population instead, both passing on
   both lawful roots: `TestReleaseEveryReachableMercenaryCarriesItsOwnRotationSpeed`
   (`pkg/game/rotationcensus_release_test.go`) hires every one of the 33
   Humans templates the installed campaign's own tavern progression can
   actually construct and checks each against its own row, independently
   parsed by name rather than by `RotationSpeedBase`'s own lookup; measured,
   18 of the 33 have a `RotationSpeed` a by-TypeID lookup alone would answer
   wrong. `TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed`
   opens a real mission and reads the live world's ghost template against
   the installed Units table's own `Ghost` row.

   One construction path the placed/hired population does not cover is
   disclosed as a divergence rather than measured here: a newly generated,
   non-hired hero's own construction (`data.ChargenBase`) does not read the
   chosen archetype's own row and takes the flat constructor default
   regardless of which archetype was picked (`DIV-437`).
4. With a positive `RotationSpeed`, an arc in `1..32` snaps `Facing` to
   `desired` immediately and sets `TurnRemaining = TurnTotal = 1` — the
   actor holds a one-tick "just turned" state before its next opportunity.
5. An arc above 32 keeps the current `Facing`, sets
   `DesiredFacing = desired`, and sets `TurnRemaining = TurnTotal =
   ceil(arc / RotationSpeed)`. The maximum arc is 128, so every positive-rate
   duration fits in the `1..128` canonical range.

Each later eligible actor tick, the advance function decrements
`TurnRemaining` by one. While it is positive the actor spends that tick's
movement and attack opportunity on the turn. When it reaches zero,
`Facing` is set to `DesiredFacing` (a no-op for the already-snapped
one-tick case), `TurnTotal` is cleared, and the actor may re-evaluate its
producer that same tick.

`SetDerived` and equipment-effect recomputation continue to replace the
live `RotationSpeed` at their existing call sites. A new turn reads the
value at request time. A turn already in progress keeps its stored
countdown and request-time total; a later positive `RotationSpeed` change
does not rescale either value or its drawn progress. A later non-positive
rate snaps the turn to its desired facing and clears both values, matching
the compatibility arm for a newly requested turn.

## Producer population

Four runtime call sites request a turn, all through the one function above:

- **Movement** (`pkg/sim/step.go`): after the production near-search
  selects the next adjacent cell, the mover derives the desired facing from
  that cell and requests a turn before committing `X`/`Y`.
- **Attack approach** (`pkg/sim/combat.go`): once the existing stop-distance
  predicate finds the actor in reach of its victim, the approach requests
  the turn toward the victim in place of its former direct `Facing` write.
- **Unit-target book admission** (`pkg/sim/spell.go`): fixes the desired
  facing from the admitted target at admission.
- **Cell-target book admission** (`pkg/sim/spell.go`): fixes the desired
  facing from the admitted cell at admission.

No other production call site assigns `Facing`, `DesiredFacing`,
`TurnRemaining` or `TurnTotal` directly. The strike routine, the spell-apply
routines and the group/AI order-selection routines carry no turn arithmetic;
they all converge on the four sites above through their existing dispatch.

Three further sites construct or restore entity state and are not runtime
turn producers:

- `pkg/sim/world.go`'s `newWorld()` normalizes a caller-supplied entity
  whose turn is inactive: if `TurnRemaining == 0`, `DesiredFacing` is set
  equal to `Facing` and `TurnTotal` to 0, mirroring the existing
  `Reach`/`HealthHundredths` normalization precedent. An entity supplied with
  a genuinely active turn is left as given and validated by the same check
  the decoder uses.
- `pkg/sim/binary.go`'s `UnmarshalBinary` reads `DesiredFacing` and
  `TurnRemaining` and `TurnTotal` from the byte form's appended tail into the
  decoded entity.
- `pkg/sim/spell.go`'s `raisedGhost` (Control Spirit) sets the new ghost's
  `Facing` and `DesiredFacing` to the source corpse's current facing,
  starting the ghost with no turn in progress.

## Movement gate and route destruction

A movement producer derives the desired facing from the next adjacent cell
the near search returns and requests a turn before committing that cell.
Current and desired facing already equal falls through unchanged to the
existing cell-commit, transit-rate and route-consumption logic.

A snap-cost (arc `<=32`) turn preserves the stored route: the actor holds
for its one tick, then proceeds. A larger turn destroys the whole stored
route the tick it begins; the actor takes no cell that tick. Once its
countdown completes, the ordinary route path searches again from the
actor's unchanged cell and may request a further turn if the new first step
differs.

A player or AI order that replaces its destination or victim cancels active
turn progress and preserves the current facing; the replacement producer
then requests its own turn. Reissuing the same retained order does not
restart progress. Withdrawal is an order replacement, so an actor turning
to retreat starts its outward turn from its current facing at the moment
the withdrawal order is issued.

Death clears `DesiredFacing`/`TurnRemaining`/`TurnTotal` residue on the body
while preserving its current facing. Stone Curse freezes `Facing`,
`DesiredFacing`, `TurnRemaining` and `TurnTotal` exactly, as it freezes the
rest of the actor. An off-map actor freezes its active turn and resumes it
after return, matching the existing preserved movement and attack orders.

## Attack gate

Attack approach requests its turn only after the existing stop-distance
predicate succeeds; out of reach, movement continues to supply facing one
adjacent step at a time. On the same cell (zero delta), no turn is
requested.

The attack cycle does not load charge, count down, or apply while
`TurnRemaining` is positive or current facing differs from the desired
facing recorded for the approach. The approach rechecks its victim every
tick after the turn completes; victim death, removal, invisibility, or
order replacement cancels the turn along with the attack order. The strike
routine itself performs no facing write and no turn arithmetic.

## Cast gate

Unit-target and cell-target book admission fix the desired facing from the
admitted target or cell at the moment of admission. Wind-up does not
advance until the turn gate opens (`TurnRemaining == 0` and
`Facing == DesiredFacing`). A zero delta requires no turn. Retained player
and AI cast producers reuse the same admitted record and gate after their
existing priority rules.

A move order attached while a book cast is pending neither cancels the cast
nor replaces its admitted desired facing (`DIV-028`, unchanged by this
story): the cast completes its own turn and action first, and movement then
requests whatever different facing it needs afterward.

## Persistence, migration and validation

The story's first byte-form change appended `DesiredFacing` then
`TurnRemaining` to each entity record. The pass-3 correction appends
`TurnTotal` after them. The current form is version 64 with `entityLen` 294;
no existing field offset moved.

`turnFault(e Entity) error` validates the malformed classes described under
Canonical turn state: an active remainder or total outside `1..128`, a
remainder above its total, an active desired facing outside the direction
quantum, an overlong equal-facing interval, inactive desired/total residue,
a non-living entity holding progress, or an active turn with a non-positive
rate. It is checked in
`pkg/sim/itembinary.go`'s `decodeItemState`, immediately after
`RotationSpeed` is read from the same decode pass — not in `binary.go`'s
fixed-record decode, where an entity's `RotationSpeed` still holds its zero
value and a real active turn would read as fault.

Version 63 already stores genuine `DesiredFacing` and `TurnRemaining`. Its
renderer derived the missing duration from the live rate. Migration recovers
a `TurnTotal` that reproduces the facing that renderer could show at the save
instant, leaves the canonical remainder unchanged, and reports no loss. Every
readable form below 63 migrates to an inactive turn: `DesiredFacing` is set to
the entity's migrated current `Facing`, and both progress values to 0. Those
forms predate genuine turn progress, so migration never infers an interrupted
turn from a destination, route, victim or facing mismatch. Their upgrade
records a `lostTurnProgress` notice ("no saved turn in progress for any
character... everyone resumes facing the saved direction before their next
action").

**`mapload.PartyMember.HiredRotationSpeed` (round 3, P2) is a separate,
additive `Snapshot.Party` gob field, not part of the hashed byte form
above.** A save written before this field existed decodes it at its Go zero
value. At the resume ownership boundary, before either a town adopts the
party or a mission derives entities from it, `mapload.RepairLegacyParty`
resolves the saved member's exact named definition row — `Units` for siege
types and `Humans` for the remaining mercenary types — and restores that
row's positive `RotationSpeed`. Current-form nonzero values remain
authoritative; a missing, invalid or non-positive row is not guessed; and a
second repair pass is inert. `TestRepairLegacyPartyCoversTownItemsAndExactHiredRows`
covers both row families, an unresolved name and idempotence. Compatibility
hotfix `51f3051e03f6a02e9d8e45f98af8a5a9ea9fa4b6` therefore closes `DIV-436`;
no valid-save rotation-speed residue remains.

## Presentation

`pkg/game/world.go` reads `Entity.Turning()` at the map-world seam. A
turning actor is excluded from swing selection; movement selection
continues to depend on cell displacement as before.

**Standing-frame selection reads `Entity.DrawnFacing()`, not `Entity.Facing`
(story 1047 round 3, P3).** An earlier revision of this section described
standing-frame selection as reading the current `Facing` directly and
holding the pre-turn octant for the whole interval; that was this build's
own behaviour through round 2 and is a player-visible defect this round
fixes, not the shipped design. A large turn now visibly rotates through
intermediate octants at the sprite's existing eight-way resolution instead
of standing still and then snapping.

`DrawnFacing()` (`pkg/sim/facing.go`) is a pure read over the same four
already-hashed fields (`Facing`, `DesiredFacing`, `TurnRemaining`,
`TurnTotal`): linear progress toward `DesiredFacing`, `arc * elapsed / total`
where `total` is the duration stored at request time. It does not read the
mutable live `RotationSpeed`. It writes nothing back and is called zero, one
or many times per tick with an identical result each time; client-side reads
do not affect the simulation hash (`TestTheSwingClockReachesNoWorld`,
`pkg/game/swing_test.go`, already covers a render call reaching no world for
the swing seam, and nothing in `DrawnFacing` differs in that respect). This is
a linear approximation of progress, not a reproduction of the original's own
per-tick step formula
(`ANIM-DIR-006`'s `(target-current)/ticksRemaining` sixteenths per tick,
sixteen-way): `TurnRemaining` carries how many ticks are left, not how the
original would have divided them. The output resolution is also unchanged
at eight-way (`sheetOctant`), not the original's sixteen-way; that gap is
`DIV-438`, a divergence row and not a fix owed by this story.

## Observation

A headless trace reports current facing, desired facing and remaining turn
progress read directly from the production `sim.World`. It performs no
independent recomputation of turn duration.

## Exclusions

- The undecoded AI target-scoring `turnCost` term (`AI-COST-071`) is neither
  inferred nor changed; the live scorer's placeholder is untouched.
- The original idle random-turn and struck-turn behaviours are not
  reproduced; neither is a runtime facing producer in this implementation.
- No new turn sprite block, sound, or GUI control is added. A turn draws
  the standing frame throughout, at whichever octant `DrawnFacing()` names
  that tick (Presentation above) — gradual octant progress is shipped;
  sixteen-way resolution is not (`DIV-438`).
