# Loaded-cycle order routes and Retreat geometry

## Intent and authority

A unit that holds a loaded attack cycle finishes its blow before an external
order takes hold, and the withdrawal and Retreat flee cell is measured from the
movers' fine positions. The owner directed both results as known defects D1 and
D2. Pinned k120.

Authority for the cycle: `AI-ORDER-039`, `AI-RETREAT-272`, `AI-RETREAT-271`,
`HERO-CADENCE-112`, `AI-351` through `AI-357`. For the flee cell:
`AI-WITHDRAW-028`, `AI-RETREAT-273`, `AI-RETREAT-274`, `AI-ORDER-039` (a unit on
a cell centre has fine position `0x80` on both axes). Each divergence row keeps
its own Unknown; nothing here claims more than the claims state.

## As built

State-only order writers: `commandDefend`, the player and script Patrol
(`commandPatrol`, `setSavedPatrol`, `cmdGroupPatrol`) and the script group stops
(`stopGroupMembers`, `stopSavedGroupMembers`, `stopRoamMember`). Each stores a
state and pending order 0 and no progress. Before this story they cleared the
attack record only between cycles and left the victim of a loaded cycle with no
wait, so a member whose cycle ended before the next decision pass (sixteen
ticks) loaded one more cycle on the old victim. They now call
`retainCycleForState`: the victim stays, the member's own cell waits beside it
as the destination, and the attack pass drops both when the cycle returns to
ready. No persisted field is added; a world holding the pair is the shape
`standDown` already gives Hold.

`fleeCell` measures the actor and the hostile list at the movers' fine
positions: cell centre when standing, retained current motion for an actor
loaded from a SAV, paid stride steps between cells. A centred actor's minor
axis changes: for the unchanged fixture of actor (20,20) and hostiles (18,18)
and (18,24) the flee cell is (23,19) where the whole-cell model gave (23,18).

## Rows verified, not changed

- DIV-574, DIV-576: both need native action-progress and whole-session cadence
  evidence the claims do not state; idle healing and turn fallback of
  `AI-ACQUIRE-002` remain absent. Open.
- DIV-575: `AI-RETREAT-273/274` give the divisor as the low byte of the list
  count. The engine divides by the full count; the two agree for 1 through 255
  hostiles. A count of 256 would divide by zero in the original. Owner
  decision, not a claim gap. Open.
- DIV-350: `AI-WITHDRAW-026` names no Stone exemption; the engine freezes the
  actor deliberately. Needs a claim or an owner decision. Open.
- DIV-352: signed WORD compare is unreachable on shipped content; fixing it for
  one consumer would split health handling. Open.
- DIV-1509, DIV-1563: held destination behind the cycle matches
  `AI-ORDER-039`; the first walk tick after the cycle is not stated. Open.
- DIV-1582, DIV-1663, DIV-1664, DIV-1601, DIV-1581, DIV-1607: each rests on a
  fact no claim states (see open debt). Open, text unchanged.

## Proof

Sim: `TestAStateOnlyOrderLoadsNoSecondCycleBehindAFinishingOne` gives Defend and
Follow in every tick of a cycle for a player unit, a script group and a saved
group. Loss control: with the wait removed the archer loads one more cycle for
orders given 70 through 78 ticks after the cycle loaded. The byte-form test
`TestAStateOnlyOrderHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically`
decodes the held world and runs 150 ticks to equal hashes.
`TestFleeCellMeasuresAMoverBetweenCellsAtItsFinePosition`, the updated
mean-geometry tests and `TestRetreatFromFinePositionsRunsManyTicksAndEveryTenthTickReloadsIdentically`
(300 ticks, decode every tenth tick) cover the flee cell.

Release: `TestReleaseOriginalSaveLoadedCycleFinishesBeforeStateAndMoveOrders`
resumes three original SAVs of the corpus whose participant units hold a loaded
cycle, steps to twelve offsets, gives a move and a Defend of a companion through
production ticks, and requires the cycle kept, the blow resolved, no second
cycle, and a decoded copy of the world to run on to equal hashes. Loss control:
with the wait removed, `projectiles-original-en/game0022.sav` loads a second
cycle after Defend at offsets 0 and 2. `TestReleaseADefendOrderGivenAsACycleEndsLoadsNoSecondCycle`
and `TestReleaseADefendOrderWaitingBehindABlowSurvivesSaveAndLoad` play the
installed warrior through App input and a SAVE and cold LOAD; the first passes
on the old route too, because the warrior's cycle ends inside one decision
interval, so the sim sweep carries the loss control.

## Open debt

Questions for research, each falsifiable and carrying no expected answer:

1. Which routine does the script group sub-command Stand Ground call, or does it
   store group order 3 without `R0060` and `R0063` (DIV-1581)?
2. Within one tick, does the executor pass of a member run before or after a
   Stand Ground setter, and between pickup row 7's first and completion pass
   can a setter run (DIV-1663)?
3. Which routines read `ord+0x50` after a completed pickup (DIV-1664)?
4. Does the group dispatcher read an effect bit of the member before the
   withdrawal tail's health compares, and does the tail write pending-order
   fields for a member carrying the Stone effect (DIV-350)?
5. On which tick after a retained cycle's recovery zero does a pending walk
   write its first step (DIV-1509, DIV-1563, DIV-1577)?
6. What does a mover with a footprint wider than one cell store as its fine
   position (DIV-348)?
