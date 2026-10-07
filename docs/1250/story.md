# Cornered flee request and the seed route

## Intent and authority

A flee request for the mover's own cell is not one result. Public knowledge
`57ad4d61a8620f48ff86d69e72f6a2daf8627a16` binds four conditional contracts.
`MOVE-080` (High): extraction with endpoint equal to seed leaves an initially
empty list empty and retains an existing list. `MOVE-081` (High / Medium): the
static list is cleared before the requested-seed shortcut; a substitute equal to
the seed, or none, yields zero nodes from an empty list. `MOVE-082` (High /
Medium / Unknown): the zero-count static search tail resolves to the current
cell and sets the failure flag. `MOVE-083` (High / Unknown): a centred
zero-radius request equal to the current cell returns before search and
preserves lists and flags. Native actor and order continuation stays Unknown
(`DIV-1556`). Engine code and tests are not ROM1 evidence.

## As built

`fleeRefusedAt` separates the two seed requests. A request equal to the cell of
a centred mover (no transit outstanding) is not refused: no reacquisition runs,
the pending move ends by the arrival rule and the actor holds no victim until
the ordinary decision. A mover between cells, and a search that settles on the
own cell or returns no route, stay refused and take the DIV-1555 reacquisition.
The change covers both automatic withdrawal arms and explicit Retreat, which
share `answerRefusedFlee`. A cornered ranged actor at the playable edge no
longer keeps its pick across cycles (`DIV-1639`). No persisted field is added.

## Proof

Sim tests: `TestFleeRequestForTheOwnCellIsRefusedOnlyBetweenCells`,
`TestWithdrawalCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut`,
`TestExplicitRetreatCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut`
and `TestExplicitRetreatCorneredAtThePlayableEdgeStandsAfterTheCentredShortcut`.
Release witness `TestReleaseCorneredSlingerCentredSeedRequestSAVColdLoadAndNextAction`
places slinger 0 of mission 41 on a playable-edge cell with the hero two cells
inward, runs production ticks, saves through F2, cold LOADs and compares World
hashes through 48 further ticks. With the shortcut reverted to a refusal it
fails at tick 7 with the slinger holding an attack target.

## Open debt

`DIV-1639` names the continuation after the shortcut and the unmeasured native
reach of the request from the flee arm. `DIV-1556` keeps the engine's single
route list against the original's separate static and dynamic lists. The
release witness proves SAV round trip of the resulting state; it holds no
original-SAV oracle and no byte-level loss control for the shortcut.
