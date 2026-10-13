# One order state: the automatic cast acts at the right moment and no unit fires on the move

## Intent

A mage on automatic casting no longer kills everything on the move. A unit
finishes its step, stands, turns to its target, strikes or casts, and walks on
only when the action has ended. The automatic cast is a layer over the order
machine: it acts only at the moment a player's cast would act at once, and it
starts the cast through the player's cast path. Every question "what is this
unit's order doing now" reads one order state. The story started at public main
`a2732033`, pin k275; the pin moved to k287, k288 and k289.

## Authority

Owner direction (story brief): no firing on the move; the automatic cast is
ours and acts only when a player's order would be right; one order state
(architecture audit `arch-audit-34a44c34`, row 13, owner decision 2); Swarm
stops, turns, fights and resumes.

Claims at k289:

- The order machine reads a new order only at progress 0; a running strike or
  cast keeps progress to its exit, recovery included, and a move order given
  meanwhile walks after the exit (`AI-487`, `AI-491`, `MAGIC-311`; High for the
  setters, Medium for the timing).
- The step bodies read no order; a stored order waits for the step to reach the
  cell centre (`AI-509`, Medium; `MOVE-148`, High). The setters of move,
  attack, stand, retreat and the two casts store no position, progress or
  `mover+0x98` (`AI-527`). Progress 1 and 2 (a strike, a cast) count to their
  exit before progress returns to 0 (`AI-534`, Medium).
- A Move group order acquires nothing before arrival (`AI-519`); a strike or
  shot a unit starts by itself begins only at progress 0 behind an exact-facing
  and reach gate (`AI-522`).
- The original has no armed automatic cast (`MAGIC-321`); its own unbidden casts
  (the heal arm `MAGIC-320`, the engage's spell roll `MAGIC-322`) are stored as
  pending kinds 8 or 9, the kinds a player's cast reaches.
- The player's Swarm command leaves group order 5, walk to the commanded cell
  and fight what is seen (`AI-SWARM-022` amended, `AI-520`).

Rows: `DIV-3191` (new, the automatic cast moment), `DIV-3009` (the support
cast inherits the moment), `DIV-028` and `DIV-030` (closed: the engine now holds
the body through the recovery as `MAGIC-311` states).

## As built

### One order state

`pkg/sim/orderstate.go` holds `orderState`, built by `World.orderState` from
every carrier of a unit's order: the book cast record (`castPhase`: approach,
waiting for mana, charging unpaid, charging paid, recovering), the one-shot
recovery `CastWait`, the scroll record (reserved, started), the player's cast
row queued in `PendingOrder`, the step (`Transit` or an imported motion) and
the strike cycle. `phase()` answers one `orderPhase`: none, step, queued,
approach, waiting, charging unpaid, charging paid, recovering, scroll,
striking. No tick ends in a releasing phase: the application happens inside
the tick the wind-up reaches zero, so a cast reads charging up to that tick
and recovering after it.

The former predicates read it. `bookCastInFlight` and `scrollCastPending` are
removed; `actorCastBusy` is `castBusy()`; `World.paidWindUp` is a reader of
`paidWindUp()`; `manualCastContinues` takes the record through `bookAt`. Sites
moved: `actionguard.go`, `actioncontinuation.go`, `currentactions.go`,
`facing.go`, `manualcast.go` (three), `nativescalars.go` (two),
`pendingorder.go` (two), `retreatcontinuation.go`, `savedmotion.go`,
`spell.go` (`inCombat`), `step.go` (three). `bookCastIndex` stays as the index
lookup where a site mutates or reads the record's fields (`activity.go`,
`creaturecast.go`, `spell.go` admission and release, `spelldelivery*.go`,
`world.go`, `actionguard.go` clear). No site disagreed with the others; step 1
changed no behaviour (`TestOrderStateAgreesWithEveryFormerPredicate`).

### The right moment

`autoCastMoment` (`pkg/sim/spell.go`) gates `stepAutoCasts`: the automatic cast
(the armed row, the idle heal and the support cast) runs only when the order
phase is none, no move order is in force (`HasTarget` without an attack
target) and no attack order waits. A pursuit under an attack order is not a
move order: the unit casts at a cell centre between steps, standing. The cast
starts through `castOrder` (`pkg/sim/manualcast.go`), the entry a player's cast
reaches once nothing stands before it, so the turn, the wind-up, the release
and the recovery are the player's (`TestTheAutomaticCastIsThePlayersCastAtTheSameMoment`).

### The body through the recovery

The mover stands down while `orderState.ownsBody()`: a book cast's wind-up and
recovery, the one-shot `CastWait`, and a started scroll. A move order given in
the recovery stays attached and walks after it. The earlier defect, a
commanded autocaster that re-armed on the tick its recovery ended and never
walked, cannot recur: the automatic cast never acts while a move order is in
force (`TestAMoveOrderInTheRecoveryWaitsThenWalks`, armed and unarmed;
`TestACommandedAutocasterWalksAndShoots`).

### Swarm and attackers

No change to the Swarm arms. A Swarm member walks under a destination, so the
automatic cast waits; `armSwarm` gives it an attack order on what it sees, and
the strike or cast happens standing; `resumeSwarm` walks it on. `advanceAttack`
already loads a strike only at ready, standing and facing, and resets a cycle
in transit; `TestAttackersNeverReleaseInAStep` proves it for a melee fighter
and an archer.

### SAV

No new persisted field. `CastWait`, the book record, the move order and the
automatic cast setting already persist; the moment is computed from them. A
SAVE in a recovery with a move order behind it and a SAVE mid-step with an
armed automatic cast continue after a LOAD in-process and in a fresh process
as the uninterrupted game does (`TestReleaseOrderStateSurvivesSAVE`).

### Second game

The automatic cast and the body rule are shared by both games. The support
cast keeps `R2-ENGINE-397` (support while attacking with an attack range below
2) and gains the moment of `DIV-3191`: a second-game mage no longer starts a
support cast mid-step or in its recovery. On mission 102 the fight after
the Blizzard SAVE of `TestReleaseSecondGameBlizzardSaveContinuation` now ends
with a unit whose death loses the mission dead within the 120 ticks the test
runs; the test keeps those units healed after the SAVE as it did before it, in
the continuing world, the in-process LOAD and the fresh-process LOAD.

## Choices where the claims leave one

- The automatic cast keeps the unit's attack target and group order; a
  player's cast at the same moment replaces them. The cast itself is the same.
- A pursuit under an attack order is not a move order; the unit may cast from
  a cell centre between steps.
- A retained creature cast also holds its body through recovery, by the same
  rule as a player's cast (`MAGIC-311` states it for the order machine as a
  whole).

## Proof

Focused tests in `pkg/sim`, each failing on the step-1 code (`e399ef73`) except
where noted: `TestAnArmedMageCastsOnlyAfterItsWalkEnds` (tick 0 cast under a
move order), `TestAMoveOrderInTheRecoveryWaitsThenWalks` (stepped during the
recovery), `TestSwarmStopsStrikesCastsAndWalksOn` (tick 53 cast under the Swarm
walk), `TestAttackersNeverReleaseInAStep` (passes on the old code: proof of the
existing rule), `TestOrderStateAgreesWithEveryFormerPredicate`,
`TestAnArmedAutocastWaitsForTheMoveOrderToEnd` and
`TestAMoveOrderWaitsForARetainedRecovery` (replacing two tests that pinned the
old rules).

Release tests on EN and RU (`pkg/game/orderstate_release_test.go`, mission
route, `f.live.tick`): `TestReleaseArmedMageCastsOnlyAfterItsWalk` and
`TestReleaseOrderStateSurvivesSAVE`; all three fail on the step-1 code.
`TestReleaseAreaLayerCostContinuesAcrossSaveAndLoad` walked its mage during
the recovery of its cloud cast; it now waits for the recovery before the walk.

## Open debt

- The original's own unbidden casts (`MAGIC-320` heal arm, `MAGIC-322` engage
  spell roll) are not built as such; the automatic cast stays the owner's
  affordance (`DIV-040`, `DIV-3191`).
- EXP-0548 (remaining setters, group-pass helpers, the length of a hold) lands
  later; the rules here do not depend on it.
