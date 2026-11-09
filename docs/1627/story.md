# Composed order rules

## Intent

Make the ordinary second-game player order stages follow the composed table: move without automatic diversion, finish the crossing, acquire while standing, pursue an explicitly named moving target, and accept a replacement immediately. Keep one owner for target ranking, walk stops and attack execution. The brief's bounded fallback leaves the rest of the machine as named debt.

## Authority

Owner direction and `knowledge/formats/rom2-ai/orders.md` at k452 (`45c1ee8ef818ac942dba95deafa8965e43770e70`). Read R2-ENGINE-1104 through 1115, the table's cited cards and the additional inherited 1076/1077/986/362/503/199 cards. No original runtime was run.

The base census has 56 rows: same 23, differs 26, absent 7. Row numbers T01..T56 follow the data rows of the pinned page in file order. The complete census remains an owner artifact, `TABLE.md`; it is not a second specification in this repository. Grades remain local: High arithmetic/stores do not promote Medium caller, kind, population or timing clauses.

## As-built behaviour

Changed clauses: T07, T11, T12, T23, T25, T26, T27, T28, T34, T35, T38, T39, T44, T45, T48 and T52. T23/T28/T39/T48 keep the broader obligations below.

- `chooseTarget` replaces the independent minimum loops in `acquireStanding`, `savedDecision` and `reacquisitionVictim`. The first commit of this merge keeps their policies unchanged. The same builder then receives idle distance/turn/later-tie ranking as profile data; Group scoring keeps first-on-tie and re-pick keeps its existing turn policy.
- Standing selection uses nearest live edge distance, least turn and the later full tie. The existing score veto and flier adjustment apply to living candidates. A live-list miss can fall back to bodies without those two adjustments. `savedAcquire` writes prepared activity 6 without replacing member order 12. R2-ENGINE-501/1018/1043.
- `closedOn` supplies the shared live edge-distance/range admission to physical approach and strike; spell weapons retain their existing spell-range admission. `advanceAttack` owns the facing gate; a rejected facing requests a turn and waits even when that turn snaps. Activity 6 does not pursue a target that moves out of range. Physical HP application retains its separate existing rule. R2-ENGINE-1105/1106; DIV-4768 bounds application operands.
- The existing engage-refusal predicate receives the acquisition/engage context as data. A short-range player mage can idle its acquisition without disabling an explicit attack. R2-ENGINE-1106; the full mage helper stays open.
- `stopWalkOrder` extends the former refused-walk writer and absorbs the Group-arrival writes. The profile makes a centred second-game arrival retain programme 4 and store member order 12. Empty-route stop and arrival retain different latches. The move planner waits for transit/current motion before acquisition. R2-ENGINE-1108 and R2-ENGINE-1076. The route that decides when the original signals arrival is not claimed closed.
- `candidateCost` keeps one scorer body. Profile data supplies the real turn term, ground-only ranged candidate folding and the read stand-ground preference rule without a size/Mind gate. Existing preference data and kind binding remain bounded. R2-ENGINE-503/1111.

Kinds and owners: the unit decision still enters through `stepWorld`, `savedGroupPassTraced` and `savedActorDispatch`; target ranking is `chooseTarget` with `candidateCost`; candidate population is the existing `actorCandidates`/`candidates` and sight builder; approach/strike is `approach`/`closedOn`/`advanceAttack`; walk stop is `stopWalkOrder`; edge arithmetic is `edgeDistanceAt`, used by both centred and current-position callers; game differences are fields of `gameProfile.orders`, chosen only by `newGameProfile`. The archtest guards the three merged chooser callers with a source check. No second game executor, game-identity branch, persisted field or sidecar is introduced. The SAV reader and producer are unchanged. `autoCastTargetWithin` is unchanged.

## Proof

`TestIdleTargetRankUsesDistanceTurnAndLaterTie` first failed for all three ranking cases. `TestSavedIdlePreparationKeepsMemberOrder` first observed 12/5 instead of 12/6. `TestStandGroundScoreUsesTurnAndUnconditionalPreference` first observed the omitted turn term. The final tests include living/body fallback, separate admission/application distances, facing rejection, mage exception and different stop latches. Every changed rule's test cites its claim.

`TestFirstGameOrderCellsKeepTheirBytes` pins pre-change scene hashes for idle selection, scoring, strike admission and empty walk: 17586582087991034016, 14172829896077500475, 12156141118182599328 and 3291890885520310301. Existing two-game move/arrival and player-command regressions remain required.

`TestSecondGamePlayerOrderScenario` drives production `Step` through move, centred arrival with an enemy, idle attack, explicit attack on a moving target and replacement move. It checks immediate attachment and identical hashes after a synthetic binary round trip and subsequent actions. This is simulation continuation proof, not installed SAV or native reload proof. `TestSecondGameComputerChoiceUsesTheSharedMachine` drives the phase-6 computer assignment through the same chooser and executor. `TestOrderTargetSelectionHasOneBuilder` checks the three merged callers.

The existing build-path hashes and cast-request recordings change only for second-game runs. Their first-game rows remain byte-for-byte unchanged. The newer activity-6 exception replaces the older second-game mage expectations; the activity-cover fixture starts already facing its target so it measures the activity gate independently of the new facing wait. Weapon-spell range controls remain required and unchanged.

Final lane evidence: the focused sim set, the brief's focused archtest package, the focused storyguard check after lowering its comment counters, `scripts/check-fast.sh`, and `check-div-claims.sh`. Exact command results and times belong to the owner return. No release test, whole sim package, scenario set, chain, game window or install write is authorized here.

Next release: on ROM1 EN/RU run `TestReleaseAttackFacingMissionTrace`, `TestReleaseAWarriorKeepsItsOrderedVictim`, `TestReleaseAMageKeepsItsOrderedEnemyAndResumesAfterASelfCast`, `TestReleaseALoadedAttackCycleFinishesBeforeAMoveOrder` and `TestReleaseResavedAttackBoundaryKeepsItsCurrentAction`. On ROM2 EN/RU run `TestReleaseSecondMissionNamedSaveContinuation`, `TestReleaseSecondMissionTwentyNamedSaveContinuation` and `TestReleaseSecondMissionSaveEntryPoints`. Run them through the release gate only. A native ROM2 order-stage comparison remains separate evidence.

## Open debt

Unchanged differs/absent rows: T02, T03, T05, T14, T15, T20, T21, T22, T29, T31, T33, T36, T37, T47, T53, T54 and T56. Partial rows T23/T28/T39/T48 retain the remaining selector/programme, re-pick population and route-result obligations. No claim is made that all 56 rows or every player command kind is complete.

DIV-4758..4768 name the settling static reads/runs: complete writer census; occupied goal/blocked neighbours and arrival-to-reset; all pursuit endings; deferred/input writes; RU rate writer and delivered cadence; full EN/kind coverage; candidate/grid/list and mage helpers; unsupported raw primary/prepared cells; remaining programmes/support; posted hit readers/Darkness; application operands. Existing first-game rules remain fallbacks in those cells and are never cited as second-game evidence. DIV-4769..4777 are unused.

Seat verification still owes a fast-gate run: the sandbox mod-folder test reports `Access is denied`, and unrelated path tests disagree with the prescribed temporary-root spelling or executable lookup. The lane records that refusal and does not repeat the gate. Synthetic checks do not replace installed EN/RU or native acceptance.
