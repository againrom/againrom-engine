# Order admission and raw primary cells

## Intent

Start stage D of the second game's order machine: one admission through which cast, movement, strike and a pending order pass, then the raw executor cells as rows of it. First-game orders, bytes and hashes stay as they were.

## Authority

Owner direction (BACKLOG item 8) and `knowledge/formats/rom2-ai/orders.md` at k452. The census of the owed cells at base `2d86e1491` is the owner artifact `TABLE.md` of this story's notes. Grades are clause-local.

## As-built behaviour

`admission` (`pkg/sim/orderadmission.go`) is the one answer to which arm owns an actor on an executor call, numbered as the original's primary byte: hold, step, cast, strike, or zero for a prepared action, plus the order's progress byte. `executorProgress` and `pendingLogicalProgress` are its views. The book and scroll cast steps, the move loop, the strike loop and `takePendingOrders` ask `waits` instead of testing the hold and the imported step themselves. The first commit changes no behaviour.

Raw primary cells (second game, profile field `orders.primaryCells`; R2-ENGINE-553 High, all 255 cells in EN and RU): `primaryCell` maps +9 of 1..4 to their arms, 5..254 to the clear arm and 255 to the keep arm. A saved byte no typed carrier holds takes its cell in `admission`. The clear cell makes every executor wait; `runPrimaryCell` runs on the actor's call in the move loop, stores +9 = 0 and holds the strike loop for that call, so the call dispatches nothing (R2-ENGINE-542 High, R2-ENGINE-556 High local). The keep cell is the admission's answer for byte 255 and, as at base `0040adcf2`, holds only the pending order: the byte is kept, and movement, the strike, casts and the player's orders run. The hold arm clears +9 once its effect is gone, so an imported 4 without a hold clears as well. A queued order then starts on the next call. Unit state 27 is not written: the World projects the action word from its typed carriers, and the meaning of 27 is Medium (R2-ENGINE-547).

Not built, row T22: the hold of every executor at 255 (R2-ENGINE-553, High) and its exit (R2-ENGINE-554: both clears belong to planner L2.02864 with unread conditions, writer census Medium). A hold without the exit would freeze the unit for the mission, so the engine keeps the byte and holds only the pending order (DIV-4799). The hold arm stays bound to Stone Curse, which holds at once, also mid-strike, while the original installs the hold only at +9 = 0 and the effect bit is unidentified (DIV-4798).

Kinds and builders: order admission is `admission` with `waits`; target choice is unchanged in `chooseTarget`/`candidateCost`; route failure is unchanged in `refusedWalkArm`/`answerRefusedRoute`; game differences are the new `gameProfile.orders.primaryCells` field; SAV is the existing reader and producer, and the cleared byte is the existing persisted `Raw[9]`. No new persisted field, no sidecar, no owner test. `TestExecutorGatesReadTheAdmission` holds the two cast steps and `takePendingOrders` (no direct hold or motion test), the two progress views (they read the admission), and `stepWorld` (it asks `waits` for the move and the strike executors and tests no `motionActive`; it keeps `stoneCursed` for other purposes).

## Proof

Step 1: the focused set of 319 tests passes on the admission commit, including the 16 scenes of `TestFirstGameOrderCellsKeepTheirBytes`, the second-game scenarios of stories 1627 and 1628 and the paid regressions listed in the brief.

Rows: `TestAnImportedPrimaryFiveToTwoFiftyFourClearsAtTheFirstCall`, and `TestAnImportedHoldByteWithoutItsEffectClears` failed before the row and name R2-ENGINE-553/554. After the correction pass, `TestAnImportedPrimaryTwoFiftyFiveKeepsItsByteAndLetsTheActorWalk`, `...AnswersAnAttackOrder`, `...FinishesAStrikeInProgress` and `...HoldsOnlyAPendingOrder` fail on the reviewed candidate and pass on the corrected one. `TestSecondGameImportedPrimaryScenario` steps a cleared and a kept (walking) actor through `Step` and compares every step with a synthetic binary round trip. `TestFirstGameOrderCellsKeepTheirBytes` gains first-game scenes for imported primaries 4, 7 and 255, with hashes measured before the row. Synthetic worlds only; no installed save shows a primary of 4..255.

Release tests owed at the landing, EN and RU of each game: ROM1 `TestReleaseAttackFacingMissionTrace`, `TestReleaseAWarriorKeepsItsOrderedVictim`, `TestReleaseAMageKeepsItsOrderedEnemyAndResumesAfterASelfCast`, `TestReleaseALoadedAttackCycleFinishesBeforeAMoveOrder`, `TestReleaseResavedAttackBoundaryKeepsItsCurrentAction`; ROM2 `TestReleaseSecondMissionNamedSaveContinuation`, `TestReleaseSecondMissionTwentyNamedSaveContinuation`, `TestReleaseSecondMissionSaveEntryPoints`.

## Open debt

- T15: prepared 3/13/14 start nothing because the engine has no prepared-byte dispatch at all; the admission has no prepared table yet (DIV-4800).
- T02: the selector keeps 8 of its 28 arms; the default planner L2.01733 and orders 2, 4, 0xd, 0xe, 0xf, 0x18, 0x19, 0x1a, 0x1b are absent (DIV-4800, R2-ENGINE-555 per-order closure Medium).
- T23/T48 raw 0x18 arm and the raw-primary tail: the empty-route tail runs only after a refused route, not after every arm, and writes no 255 for order 0x18 (DIV-4800). Not started: the tail needs the +0x98 flag as admission state, and its coexistence with a new order is Unknown (R2-ENGINE-556).
- T20: hold binding and install timing (DIV-4798). T22: not built, the hold at 255 and its exit (DIV-4799).
- `castHeldNextTick` (`pkg/sim/spell.go`) still reads `stoneCursed` and `motionActive` itself to predict the book-cast gate.
- `advanceTurns` (`pkg/sim/facing.go`) still reads `stoneCursed` and `motionActive` itself to freeze a turn.
- Outside rows T03, T05, T31, T33, T36, T37, T47, T53, T54, T56 and outer T39 keep their 1628 verdicts.
