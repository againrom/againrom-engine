# The dead victim, the group join and the vanished sack (k323)

## Result

No production code changes. Thirteen divergence rows that leaned on Unknowns
of a victim's death, the script owner change, a pickup's sack and the strike's
virtual calls now cite the k323 claims: one row is closed, nine are narrowed,
three take the new citation with their Unknowns unchanged, and two new rows
record what k323 shows this build does differently or leaves Unknown. One new test pins the owner change against
`AI-643`.

## Intent

KNOWN-DEFECTS D2 lists rows whose Unknowns EXP-0588 answered, and the
story1449 review left three row notes. The story adopts k323 into those rows
and changes code only where a High claim differs from this build.

## Authority

Pin k323 (`a26ba068`). The claims are read with `go run ./tools/claim <ID>`.

- `AI-640` (High for the refusal tests and the read death routines, Medium
  that no other routine clears the owner): the strike path refuses a victim
  only for a null pointer, a zero owner `+0x14` or reach; a corpse keeps its
  owner and passes every test.
- `AI-641` (High for the stores, Medium for the corpse case): a strike on a
  victim at or below 0 health still runs the damage resolver and its stores.
- `AI-642` (High for the read bodies, Medium for the dying-victim
  composition): the AI call that stands state 3 down for a victim at action
  0x10 runs the acquisition in the same call; `ord+0x0c` changes only when a
  candidate is found; bodies below 1 are candidates only when no living one is
  seen.
- `AI-643` (High for the stores, Medium for the inference) and `AI-644` (High
  for the call sites, Medium that it is the only caller): the join puts the
  actor alone in a new Group of the new Player under Stand Ground and stores no
  progress, phase or victim; the script's owner-change actions call it.
- `AI-645` (High for the read bodies, Medium for the arrival route): a pickup
  at an emptied cell completes without a transfer at state `0xc`, then the
  next AI call acquires.
- `AI-646` (High for the vtables and bodies read, Medium for the facing): the
  five virtual calls on the strike path are read; none of the bodies read
  stores the facing.
- Amended `AI-633`, `AI-636`, `AI-637`, `AI-374`; `MOVE-174`, `MOVE-175` and
  `AI-634` for the story1449 notes.

## As built

The engine already does what the High clauses state on every route the rows
name, except where a row below records the difference:

- A victim at health 0 to -9 stays the attacker's victim; at -10 every
  attacker's active attack ends and a state-3 member stands down at its cell in
  state `0xc` with pending order 0 (`clearInvalidTargetReferences`,
  `stopTargetOrder`). The acquisition parks bodies below 1 behind living
  candidates (`candidates`). The original keeps the victim pointers and, at
  Medium, keeps striking the corpse until the next AI call. A unit that keeps
  striking a body is a known original bug a player sees, and the claim is
  Medium, so this build keeps the immediate stand-down.
- The script owner change (`handOver`) moves the actor into a one-member Stand
  Ground Group of the new owner at state `0xc`, its own cell as post and
  pending order 0, keeps a loaded strike and the held own cell, and the next
  order 3 evaluation reads the new owner's control word.
- A pickup whose sack is taken before the actor arrives is dropped by the game
  driver (`settlePickup`); the walk ends as a plain move. The original
  completes it as a pickup without a transfer. The pickup approach lives on
  the driver (DIV-337), so this stays a row.

## Rows

- DIV-2018 narrowed: `AI-640`..`AI-642` answer whether the strike refuses a
  corpse (no, at Medium) and place the acquisition in the stand-down call
  (`TestFirstAttackerCrossingMinusTenClearsEveryAttackerWithoutExtraRNG`,
  `TestTargetOrderDeathBoundaryPreservesFinishing`). Open: a native read of a
  dead actor's owner, the AI call cadence, other removal routes.
- DIV-1607 narrowed: the same claims. Open: active-writer continuation, the
  owner read, other terminal routes.
- DIV-1581 narrowed: `AI-643` and `AI-644` answer the join
  (`TestAHandOverKeepsTheLoadedStrikeAndTheEvaluationReadsTheNewOwner`). Open:
  the participant's order-5 no-target behaviour and the native heal branch.
- DIV-1582 narrowed, moved to
  `docs/divergences/combat-and-ai-attack-and-hold-orders.md` for the ledger
  size ceiling: the join keeps the loaded strike and the held destination
  (same test). Open: `AI-354`'s common tail during a running cycle.
- DIV-1509, DIV-1563, DIV-1577 narrowed: `AI-646` reads the five virtual
  calls. Open: `R0810` and the unread callees.
- DIV-1741 narrowed: the in-play corpse window and the stand-down's kept
  pointer bound the question. Open: the saved words of a loaded attack after
  LOAD.
- DIV-1440, DIV-1519, DIV-3310: source updated with `AI-642`; their Unknowns
  (nonterminal removal, destructor notification) are unchanged.
- DIV-1631 closed: `MOVE-175` answers the 2:1 strictness and the 16-bit range
  (`TestHeadingOfMatchesTheCentreBearingTableForEveryDelta`,
  `TestReacquisitionDirectionMatchesControlledCentres`).
- DIV-348 narrowed: names `MOVE-174`'s octant 1 and 5 x adjustment, which this
  build's stride step does not make.
- DIV-3469 new (UNKNOWN): `AI-634`'s sack dropped on a picked cell between
  two skipped-executor calls (`TestASecondTakeOnTheEmptiedCellTransfersNothing`).
- DIV-3470 new (FIDELITY-DEBT): `AI-645`'s pickup at an emptied cell
  (`TestAStandingPickupDisarmsWhenTheSackGoesAway`).

## Proof

- Focused `pkg/sim` and `pkg/game` tests named under Rows.
- Census: every shipped mission from a fresh start, 1500 ticks, World hash
  every 50 ticks and a per-entity hash at the end, on `94ab14f3` and on this
  story's head, with a lane instrument that is not committed: see the return.
- Original-save LOAD: the milestone-2 acceptance,
  `TestReleaseAttackOrderReleasesAnOriginalHoldMember` and
  `TestReleaseWideMoverRetreatsFromItsRawPosition` on EN.

## Open debt

- DIV-2018, DIV-1607: a code change for the corpse window waits for a High
  owner-retention claim and an owner decision, since the strikes would be
  visible.
- DIV-3470: the review found a smaller route than DIV-337: `settlePickup`
  waits for `sackPickupArrived` with the sack gone and completes the pickup
  without a transfer through `World.CompleteSackPickup`. It goes to a hotfix.
- The join's clear of the old and new owner's bits in `actor+0x18` (`AI-643`)
  is not checked against this build's SAV writer; the mask is the unmodelled
  `Token18` of DIV-969.
- The new test does not see the hand-over's state and post stores on the
  saved dialect (`standScriptedMembers` writes them too) and does not cover
  the native dialect of `handOver` (review note N2).
- DIV-1582 sits in `combat-and-ai-attack-and-hold-orders.md` although its
  subsystem is simulation; `simulation.md` is near its size ceiling and
  should split by subsystem (review note N3).
