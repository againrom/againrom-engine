# Contract — story 1047 original turn speed

This story starts from implementation
`5e5cc97ab2568da22fc768cecb575dce77c4c303` with research pin
`23daf74f6ee83e5fee5474981faf7c276680cc9f`. Story 1045 owns the action-cadence
state and the next byte-form change. Story 1047 may define its result before 1045
lands, but it does not edit production simulation state, the byte form or the
action schedule until the reviewed 1045 tip is on `master` and merged normally
into this branch.

## Outcome

An actor pays a turn interval before movement, attack or an admitted book cast
can use a new facing. The interval uses the actor's live `RotationSpeed`. A turn
of at most one 32-unit direction step snaps to the desired facing and costs one
actor tick. A larger turn costs `ceil(shortestArc / RotationSpeed)` actor ticks,
destroys the stored movement route and requires a new route after the turn.

The player-visible discriminator is a withdrawing ranged actor. The actor first
turns away from its pursuer, then runs. After it reacquires an attack, it turns
back before its next attack interval begins. The pursuer continues advancing
during both turn intervals. A headless lawful-root witness reports both
intervals and the pursuer's position changes without launching a game window.

## Research baseline

- `MOVE-TURN-031` is High for the movement step gate, desired-facing store,
  inclusive 32-unit snap boundary, one-tick snap cost, large-turn route
  destruction, shortest-arc fold and rounded-up division by `RotationSpeed`.
- `AI-FACE-066` is High for facing as the approach/order-machine precondition
  for an attack and for the absence of a facing test or write in the swing and
  strike routines. Its broader every-route statement is Medium and is not used
  to add an indirect producer.
- `AI-FACE-067` is High for the turn routine and its complete direct caller
  population. It establishes that the turn belongs to mover and order-machine
  producers rather than to the strike.
- `MOVE-CLOCK-032` is High for one actor advance per simulation sub-tick and for
  simulation and presentation consuming the same paced iteration. Turn duration
  is therefore measured in actor ticks, not milliseconds or frames.
- `MOVE-DIR-034` is High for the eight clockwise directions from north and the
  direction-to-facing quantum used by the movement producer.
- `TERR-SPR-047` is High for original action state 5 selecting the standing
  frame. It supports the presentation rule that a turning actor is not drawn as
  moving or swinging. It does not establish gradual facing interpolation.
- `DIV-429` is the allocated FIDELITY-DEBT row. This story closes it. No other
  divergence id is allocated.
- `DIV-022` remains the owner-authored cross-producer cast ordering and facing
  rule. This story changes its instantaneous facing write into a desired-facing
  request and turn gate. It does not change the owner-authored producer
  priority, visibility snapshot or projectile rule.

No research item is required for the contracted result. The live AI target
scorer's `turnCost` placeholder is an adjacent consumer, not a direct turn
producer. `AI-COST-071` establishes where the term is used but does not publish
the body of `R0115`. This story does not infer that body or change target
selection. The seat decides whether that separate undecoded term receives a
narrow research item.

## Domains and interfaces

The story touches six domains:

- **Assets** supplies the existing Units and Humans `RotationSpeed` columns and
  equipment-effect modifiers through typed data.
- **Sim Core** owns desired facing, turn progress, tick order, movement gate,
  stored-route invalidation, hash and binary form.
- **Combat & Magic** owns the attack-facing and book-cast gates. The strike and
  spell application routines do not turn an actor.
- **AI & Orders** owns attack approach, withdrawal, reacquisition and the common
  order paths used by player and AI producers.
- **Client** reads the canonical current facing and turn state. It presents a
  turn as standing, not walking or swinging.
- **Persistence** owns exact round-trip, malformed-state refusal and migration
  from readable forms that predate turn progress.

This slice combines hashed simulation state with more than three domains and
names five behaviour groups below. It remains one story because every producer
must converge on one canonical gate. Splitting persistence, presentation or one
producer from the gate would permit a saved, drawn or source-specific turn to
bypass the same interval. The owner also assigned withdrawal and reacquisition
as one decisive interaction.

## Behaviour

### 1. Canonical turn state and rate

1. `Facing` remains the current canonical facing byte. Each actor additionally
   carries a desired facing byte and an unsigned remaining-turn countdown. The
   desired facing uses the same 32-unit direction quantum. These values enter
   the hash and byte form.
2. An inactive turn has remaining zero and desired facing equal to current
   facing. An active turn has a remaining count in 1..128. A desired facing not
   divisible by 32, a remaining count above 128, or inactive residue is
   malformed.
3. The shortest arc is `min(abs(current-desired), 256-abs(current-desired))`.
   An arc of zero requires no turn. An arc in 1..32 snaps current facing to the
   desired facing immediately and stores one remaining tick. An arc above 32
   keeps the current facing until its countdown completes and stores
   `ceil(arc / RotationSpeed)`.
4. A positive `RotationSpeed` is consumed exactly as carried. The maximum arc
   is 128, so every positive-rate countdown fits in the canonical range.
   `RotationSpeed <= 0` keeps the existing authored compatibility arm: it snaps
   without a turn interval or route destruction. Shipped definitions use
   positive values, and this round's wiring (`cmd/rotationcensus`) carries
   them to every live actor built from a shipped placement: 2361 of 2361
   entities across the 28-mission campaign census, both lawful roots, none
   at `RotationSpeed <= 0`. Synthetic and legacy worlds that omit the field
   keep their pre-story compatibility schedule.
5. Equipment effects and `SetDerived` continue to replace the live
   `RotationSpeed`. A new turn reads the value after the latest recomputation.
   A turn already in progress keeps its stored countdown; rearming does not
   rescale elapsed work.
6. The first tick starts the interval. At the start of each later eligible
   actor tick the countdown falls by one. A positive remainder consumes that
   actor's movement and attack opportunity. When it reaches zero, current
   facing becomes desired facing and the actor may re-evaluate its producer on
   that tick.

The delayed large-turn facing update and the nonpositive compatibility arm are
againrom representation choices. The decoded observable is the interval, gate
and route destruction. The presentation does not interpolate a value the
research does not establish.

### 2. Complete implementation producer population

7. The production source census at the branch point has four runtime facing
   producers: the next movement step in `pkg/sim/step.go`, the closed-on attack
   approach in `pkg/sim/combat.go`, unit-target book admission in
   `pkg/sim/spell.go`, and cell-target book admission in the same file. All four
   use one turn request and gate. No source-specific direct `Facing` assignment
   remains.
8. Constructor literals, binary decode, summoned-entity inheritance and
   `SetDerived` are state feeds, not turn producers. Binary decode installs the
   canonical pair. A summoned entity inherits the source's current facing and
   starts with no turn. `SetDerived` changes only the rate input.
9. The producer census is pinned mechanically. A new runtime write of current
   facing, desired facing or remaining progress fails until it is classified as
   construction, restoration, turn advancement or a new direct producer using
   the common gate.
10. Player commands, group/AI decisions, withdrawal and retained action
    rearming converge after their source-specific selection. They do not carry
    separate turn arithmetic.

### 3. Movement gate, route destruction and interruption

11. A movement producer derives desired facing from the next adjacent cell
    returned by the production near search. It requests a turn before committing
    that cell. Current and desired equality permits the existing step and
    transit-rate path.
12. A small positive-rate turn preserves the stored route. A large
    positive-rate turn destroys the whole stored route when the turn begins.
    The actor takes no cell on that tick. After the countdown completes, the
    ordinary route path searches again from the actor's unchanged cell and may
    request another turn if the new first step differs.
13. A player or AI order that replaces its destination or victim cancels active
    turn progress and preserves current facing. The replacement producer then
    requests its own turn. Reissuing the same retained order does not restart
    progress. Withdrawal is an order replacement and therefore starts its
    outward turn from the actor's current facing.
14. Death clears desired-facing and progress residue while preserving the
    current facing of the body. Stone Curse freezes current facing, desired
    facing and progress exactly. An off-map actor also freezes the active turn
    and resumes it after return, matching its preserved movement and attack
    orders.

### 4. Attack, cast and withdrawal gates

15. Attack approach requests the turn only after the existing stop-distance
    predicate succeeds. Out of reach, movement supplies facing one adjacent
    step at a time. On the same cell, the zero delta requests no turn.
16. The attack cycle does not load charge, count down or apply while turn
    progress remains or current facing differs from the current desired facing.
    The approach rechecks the victim each tick after completion. Victim death,
    removal, invisibility or order replacement cancels the turn with the attack
    order. The strike routine remains free of facing writes and turn arithmetic.
17. Unit-target and cell-target book admission fix the desired facing from the
    admitted target or cell. Their wind-up does not advance until the turn gate
    opens. A zero delta requires no turn. Retained player and AI cast producers
    use the same admitted record and gate after their existing priority rules.
18. `DIV-028` remains unchanged. A move attached during a pending book cast
    neither cancels the cast nor replaces its admitted desired facing. The cast
    completes its turn and action first; movement then requests any different
    facing it needs.
19. In the decisive interaction, the ranged actor pays the outward turn before
    its first withdrawal step. Reacquisition replaces the move with an attack,
    and the actor pays the return turn before attack charge begins. The pursuer
    remains independently scheduled and may advance during both intervals.

### 5. Persistence, migration, presentation and observation

20. The form change appends desired facing and remaining progress to the final
    entity record produced by the landed 1045 form. No existing offset moves.
    This contract does not allocate a version number. After the reviewed 1045
    merge, the lane measures the landed current version and record width. It may
    use the next version only if the seat confirms no other branch consumed it.
21. Every readable legacy form migrates to no active turn: desired facing equals
    its migrated current facing and remaining progress is zero. A legacy form
    cannot contain progress from behaviour that did not exist, so migration
    does not infer an interrupted turn from a destination, route, victim or
    facing mismatch.
22. Hash and round-trip witnesses cover inactive state, the one-tick snap, each
    large-turn boundary and the last remaining tick. Malformed desired-facing,
    count and residue classes are refused. Save migration uses the shared
    validated upgrader rather than byte patches.
23. The Client treats positive turn progress as a standing action. A turning
    attacker is not selected as swinging, and a turning mover is not selected
    as walking. Drawing uses current `Facing`; a small snap therefore shows the
    new direction during its one-tick hold, while a large turn keeps the old
    direction until completion.
24. The headless trace reports current facing, desired facing and remaining
    progress from the production world. It does not recompute the turn duration.

## Witnesses

- A literal arc/rate matrix covers zero, 31, 32, 33 and 128 units, both wrap
  directions, exact and rounded-up division, rates 1, 8 and 23, and the
  nonpositive compatibility arm.
- Movement witnesses prove step withholding, the inclusive snap boundary,
  small-route preservation, large-route destruction, replan after completion,
  transit starting only with the cell commit and all eight desired directions.
- Attack and cast witnesses cover unit/cell targets, player and AI producers,
  target motion during a large turn, zero delta, one-tick and large turns, and
  prove no strike, cast release or action countdown occurs early.
- Interruption witnesses cover replacement destination, replacement victim,
  retained same order, death, downed state, Stone Curse, off-map return,
  rearming during progress and actor removal.
- Hash, form and migration witnesses vary desired facing and remaining progress
  independently, resume at every boundary, refuse every malformed class and
  cover every readable legacy version through the shared upgrader.
- A presentation test observes the production current facing and turn state at
  the map-world seam. It proves a turning attacker neither walks nor swings and
  that no viewer state changes the simulation hash.
- The lawful-root withdrawal witness runs on EN and RU. It reports outward and
  return turn lengths, the first movement and attack ticks, and pursuer position
  deltas during both intervals. It launches no window.

## Twelve-aspect scope

| Aspect | Contract scope |
|---|---|
| Data | Existing `RotationSpeed` columns, constructor defaults and equipment modifiers reach the live actor unchanged, through `mapload`'s `blockFor` and `PartyLoadout`/`RotationSpeedBase` on every arm (`cmd/rotationcensus`: 2361 of 2361 live entities across the campaign census carry a positive value). One construction path is excluded: a generated, non-hired hero always takes the flat constructor default regardless of the chosen archetype's own row (`DIV-437`). |
| Runtime state | Current facing, desired facing and remaining turn progress form one canonical lifecycle. |
| Simulation | Arc arithmetic, countdown, movement and attack/cast gates, route destruction and interruption. |
| Player input | Move, attack and unit/cell cast commands use the same gate as AI decisions. |
| AI | Approach, withdrawal and reacquisition pay turn intervals; target-scoring `turnCost` remains explicitly undecoded. |
| UI/HUD | No HUD control changes. The map presentation suppresses walk/swing selection during a turn and uses canonical facing. |
| Triggers/scripts | Script-produced movement, attack and cast orders converge on the same simulation producers; no opcode changes. |
| Inventory/equipment | Rotation-speed effects reach new turns through existing rearm; active progress is not rescaled. |
| Persistence/save-load | New state hashes, round-trips, validates and migrates through the shared upgrader. |
| Campaign/session | Turn duration uses simulation sub-ticks and pauses with simulation; no session clock is added. |
| Shipped content | EN and RU lawful-root witnesses exercise placed positive-rate actors and the decisive withdrawal interaction (`missionrun -mission 100 -withdrawal`: entity 109 carries `RotationSpeed=21`, byte-identical on both roots). |
| Interactions with existing mechanics | Action cadence, route state, transit, withdrawal, visibility, death, Stone Curse, off-map return, effects and rendering are covered. |

## Exclusions

- The undecoded AI target-scoring `turnCost` body is not inferred or changed.
- The original idle random-turn and struck-turn behaviour is not added. Those
  are separate AI behaviours with their own random and retaliation contracts;
  neither is a runtime facing producer in the current implementation.
- No gradual turn interpolation, new turn sprite block, sound or GUI control is
  added. The decoded turn action draws the standing frame.
- No research experiment, new divergence id, byte-form number or GUI/game/ROM
  launch is authorized by this story.

## Review boundary

The adversarial-review ceiling is three passes. The chain stops at the first
pass with no class-P finding and an empty remaining-surface list. A class-P
finding returns the story. Class-W and class-D findings are corrected or
recorded without reopening the whole lane.
