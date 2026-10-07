# Analysis — story 1047 original turn speed

## Fixed baseline

The worktree branch is `story/1047-original-turn-speed`. Its branch point is
`5e5cc97ab2568da22fc768cecb575dce77c4c303`, the assigned `origin/master` tip.
The remote has no branch with the same name. The implementation tree carries
research gitlink `23daf74f6ee83e5fee5474981faf7c276680cc9f`.

The branch-point byte form is version 61 with a 290-byte entity record. Story
1045's correction branch currently proposes version 62 and a 291-byte record,
but those values are not a story-1047 allocation. They are measured again only
after the reviewed 1045 tip lands and is merged.

## Claim reconciliation

`MOVE-TURN-031`, `AI-FACE-066`, `AI-FACE-067`, `MOVE-CLOCK-032` and
`MOVE-DIR-034` supply the contracted rules at sufficient confidence. The claims
establish these separable facts:

- The movement step is gated on current and desired facing equality.
- Desired facing uses a 32-unit direction quantum over one byte.
- An arc of at most 32 snaps with a one-tick cost.
- A larger arc destroys the route and costs the rounded-up shortest arc divided
  by `RotationSpeed`.
- Facing is an attack approach/order-machine precondition. The strike neither
  tests nor changes facing.
- Direct calls to the original aim routine are completely enumerated.
- The relevant duration unit is the per-actor simulation sub-tick.

`TERR-SPR-047` independently establishes that original action state 5 selects a
standing frame. It does not establish gradual current-facing updates during a
large turn. The contract therefore stores desired facing and progress, retains
current facing until large-turn completion, and exposes a standing presentation
state.

`AI-COST-071` establishes a separate use of facing in target selection. The
target score is `(distance << 8) + turnCost`. The implementation has retained
this term as `turnCost(member, candidate) int32 { return 0 }` because no promoted
claim publishes `R0115`'s body. This is not a turn producer or a gate on
movement, attack or cast execution. It remains excluded pending the seat's
decision on a narrow research item.

## Current runtime facing producers

The census used production Go files under `pkg/sim` and excluded tests. Three
assignment statements currently change an existing actor's facing at runtime:

| Assignment | Runtime producer sites | Source paths |
|---|---:|---|
| `Entity.face` writes `e.Facing` | 2 | movement cell commit in `step.go`; closed-on attack approach in `combat.go` |
| Unit-target book admission writes `w.entities[ci].Facing` | 1 | `beginBookSpell` in `spell.go` |
| Cell-target book admission writes `w.entities[ci].Facing` | 1 | `beginBookSpellAt` in `spell.go` |

This is four direct runtime producer sites. The first two already share one
assignment helper. The two cast forms independently repeat the same
facing-toward calculation and direct assignment.

The following writes are not runtime turn producers:

- `binary.go` restores `Facing` from entity-record byte `+91`.
- `spell.go` copies a source actor's facing into a newly created entity.
- Entity composite literals in loaders and tests construct state.
- `castevent.go` copies current facing into an observation event. It does not
  write the actor.

The future producer pin must distinguish construction, restoration, turn
advancement and direct requests. Counting only assignments would incorrectly
classify binary restore and summon construction as bypasses.

## Upstream order population

All current attack-order sources converge on `orderAttack` before approach:

- the player `KindAttack` command in `step.go`;
- three group-decision arms in `engage.go`;
- two escort arms in `escort.go`;
- a script group attack in `script.go`;
- an applied control effect in `celleffect.go`.

The unit-target book producer population converges on `beginBookSpell` from the
player command, AI cast and autocast paths. The cell-target player command
converges on `beginBookSpellAt`. Retained retry remains inside the book-cast
lifecycle that story 1045 is changing. The exact post-1045 population must be
recounted after merge because the cadence correction owns these functions.

Movement destination writers are broader, but they converge on the one movement
loop before any cell commit. They include player move, group formation, patrol,
escort, script orders, attack approach and withdrawal. The turn gate belongs at
the selected next adjacent cell, after route search, so none requires separate
arc arithmetic.

## Current facing consumers

The source census found these production consumer classes:

- `FacingDir` maps the canonical byte to an eight-direction index.
- `pkg/game/world.go` maps that direction to the unit sheet and selects live,
  swing and death presentation.
- book and weapon spell application reads current facing for fixed-facing area
  effects and writes it into `CastEvent`.
- `pkg/game/spellpath.go` uses the event facing for projectile departure.
- `cmd/missionrun` prints the current direction through `FacingDir`.
- binary encoding and the world hash preserve current facing.

The game seam currently defines `swinging` as a living actor holding a victim or
cast run while not moving. A turning attacker is stationary and still holds its
victim, so it would be drawn swinging without an explicit turn exclusion. That
is the presentation edit this story owes. Movement presentation already derives
walking from cell displacement, so a withheld step is stationary.

## RotationSpeed flow

**Round-1 finding (P1, `pipeline/reviews/1047-adversarial-pass1-return.md`): this section's
opening claim was the story's planning premise and was never measured. It was false as written.**
`RotationSpeed` reached `sim.Entity` through `recompute.go`'s `DerivedBlock` and stopped there
unconsumed, but two of `mapload.blockFor`'s three arms (the creature arm and the unresolved-loadout
arm) never populated `data.Loadout.RotationSpeed` in the first place, so a shipped placement taking
either arm spawned at `RotationSpeed 0`. Measured before this round's fix: 1968 of 1968 live
entities at 0 across the reviewer's own 24-mission census, both roots.

This round wires the missing arms (`pkg/mapload/loadout.go`'s `RotationSpeedBase`, threaded through
`blockFor`, `PartyLoadout`, `PartySpawn`, `recomputeRaisedSkills` and every call site of the
package-level `Rearm`). Measured after the fix (`cmd/rotationcensus`, the committed campaign
census, 28 missions both roots): 2361 of 2361 live entities carry a positive value, 16 distinct
values in 8..23. `RotationSpeed` is now carried end to end and has a simulation tick consumer
(`pkg/sim/facing.go`'s `requestFacing`/`turnToward`):

- `pkg/data` reads Units slot 9 and Humans slot 7. Both table constructors use
  their established default when the cell is empty.
- `pkg/mapload` carries the resolved definition into map placements, fresh
  party members and restored party members, through `RotationSpeedBase`. One
  exception is disclosed at `DIV-437`: a newly generated, non-hired hero
  (`data.ChargenBase`'s own construction path, `pkg/game/hero.go`) does not
  read the chosen archetype's own row and takes the flat constructor default
  regardless of which of the four archetypes was picked.
- equipment effect kind 18 adds its scalar to the derived modifier.
- `pkg/game/rearm.go` sends recomputed derived state through
  `sim.DerivedBlock`.
- `pkg/sim/rearm.go` replaces the entity's live `RotationSpeed`, and (this
  round) snaps a turn in progress closed if the new rate is `<= 0`.
- `pkg/sim/itembinary.go` writes and restores the value as part of derived
  actor state, so it already participates in the world form and hash;
  `pkg/sim/binary.go`'s `MarshalBinary` (this round) runs the same `turnFault`
  check the decoder already ran.

The value was inert before this story, so many synthetic entities name zero.
The contract preserves that authored absence as an immediate compatibility arm.
Lawful-root actors now take positive definition values and use the decoded
rule, with the one disclosed exception above.

## Route and scheduler placement

The movement loop currently performs work in this order:

1. refuse dead, off-map and Stone-cursed actors;
2. refuse book-cast wind-up;
3. pay transit;
4. update attack approach;
5. obtain or validate the stored far route;
6. obtain one near-search step;
7. commit the cell;
8. write facing from the completed step;
9. start transit and consume the route.

The turn lifecycle belongs after step 1 and before the book-cast, transit and
approach gates when progress is already active. A new movement request belongs
between steps 6 and 7. This placement freezes turn progress for Stone and
off-map actors, clears it through the death normalization path, and prevents a
cell commit before facing permits it.

Attack advancement is a later loop. It currently checks life, off-map, Stone,
victim, book-cast and cast-recovery state before `advanceAttack`. It must also
refuse active turn progress and a pending approach-facing mismatch. Arc
arithmetic stays in the common turn helper, not in the strike.

Book casts are scheduled before the movement loop. Story 1045 changes their
pending/retry lifecycle. The story-1047 merge must therefore identify the final
1045 admission and per-tick advancement seams before inserting the desired-
facing request and progress gate.

## Canonical form implications

The minimum new canonical data is one desired-facing byte and one remaining-
progress byte per entity. The maximum positive-rate duration is 128 ticks:
shortest arc is at most 128 and the least positive integer rate is 1. A wider
count is unnecessary.

The fields append to the landed current entity record. Existing record offsets
remain unchanged. Encoding invariants are:

- remaining zero requires desired facing equal to current facing;
- desired facing is a multiple of 32;
- remaining is at most 128;
- current equal to desired with positive progress is legal only for the
  one-tick snapped state.

Every readable older form migrates to desired equal to its migrated current
facing and remaining zero. No older implementation form could contain active
turn progress, so no route, destination or attack residue is used to guess one.

## Conflict boundary with stories 1045 and 1046

Story 1045 currently owns `pkg/sim/world.go`, `binary.go`, `castbinary.go`,
`step.go`, `combat.go`, `spell.go`, `turn_test.go`, form tests and related
mapload/game save tests. Story 1047 does not edit those files before the reviewed
1045 SHA lands. The eventual merge is a normal merge at a clean boundary.

Story 1046 owns joined-hero state and constructors. Story 1047 neither rebases
nor rewrites its work. After 1046 lands, the producer census is rerun so a new
hero constructor is classified as construction rather than as a runtime turn
producer.

New tests and witness code use story-specific files where possible:

- `pkg/sim/turn1047_test.go` for turn arithmetic, movement, attack, cast and
  interruption matrices;
- `pkg/sim/turnform1047_test.go` for hash, form and migration boundaries;
- `pkg/mapload/turn1047_test.go` for definition/effect/rearm flow;
- `pkg/game/turnpresentation1047_test.go` for the map presentation seam;
- `cmd/missionrun/turn1047_test.go` for the lawful-root witness formatter;
- `internal/archtest/facing.go` and `facing_test.go` for the producer pin.

Production edits remain in the owning source files after the merge. The
story-specific tests reduce avoidable merge conflicts but do not replace tests
at the production boundary.

## Open decision sent to the seat

The only newly exposed research gap is `turnCost` in AI target scoring. The seat
was asked whether to keep it explicitly out of scope or assign a narrow research
question. No production or document claims that the placeholder is original
turn duration, and no divergence id was selected.
