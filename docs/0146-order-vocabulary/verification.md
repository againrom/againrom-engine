# 0146 — verification

## Gates

Run in `wt-0146` on a clean tree at branch head, `go test` with `-trimpath` (Windows Defender
quarantines one test binary otherwise). The code gates were also run at `a2018e3`, the last commit
that touched code:

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test -count=1 -trimpath ./...` | every package ok |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-doc-budget.sh` | ok; this story's four artifacts at 50 %, 35 %, 30 %, and the prose ceiling |
| `bash scripts/check-sdd-audit.sh` | **FAIL set empty** |
| `bash scripts/check-hotfix-ledger.sh` | ok |

The audit's note and warning **count** is not quoted: a worktree has no `builds/`, so that number is
meaningless from here in both directions. Only the FAIL set is comparable, and it is empty.

`git log --format='%h %(trailers:key=Co-Authored-By)' 626ed7c..HEAD` prints each hash with no
trailer text beside it. No commit on this branch carries an `SDD-Task:` trailer either, and none
should: there is no `tasks.md` — one lane implemented its own slice — so the FR/DD accounting is the
table below.

## The result someone can point at

**The census did not move, and that is the honest answer.** `pipeline/check-milestone.sh`'s argv,
run from this worktree against the EN root:

```
go build -o /tmp/mr ./cmd/missionrun
for m in 10 20; do AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

| | master before (`pipeline/milestone-baseline.txt`) | after this story |
|---|---|---|
| mission 10 | 17 (1×groupcmd 11, 2×groupcmd 15, 13×instant op 2, 1×instant op 20) | **17** |
| mission 20 | 13 (1×instant op 12, 11×instant op 2, 1×instant op 23) | **13** |

Unchanged, and expected to be: this story adds no script opcode and no group sub-command. What it
adds is a way for the PLAYER to reach six arms the script already had.

**The result is in the game**, and it was driven on a shipped map rather than asserted. `cmd/
missionrun` gained a `-order` flag in the second commit for exactly this. Mission 10, EN root, unit
21 (the map script's own naming), standing at (36,51):

| Order | Command | What the drive printed |
|---|---|---|
| patrol between (36,51) and (39,51) | `-order patrol:u21:39:51 -tail 900` | ended at (38,51), **68 cell changes**, 4 distinct cells `(36,51) (37,51) (38,51) (39,51)` |
| march to (39,51) | `-order march:u21:39:51 -tail 900` | ended at (39,51), **2 cell changes**, 3 distinct cells |
| guard | `-order guard:u21 -tail 400` | ended at (36,51), **0 cell changes** |
| stand ground (unit 22) | `-order stand:u22 -tail 400` | ended at (9,18), **0 cell changes** |

The patrol/march contrast is the whole vocabulary in two lines: the same gesture and the same cell,
one walking there and stopping, the other walking there and back for as long as the world runs.

**FR-5 was driven on the same map.** `-order patrol:u21:39:51 -order guard:u21 -tail 900` — both in
one advance, patrol first — printed **0 cell changes** where the patrol alone printed 68. Before
this story the guard would have been applied and then undone: the actor pass re-issues a
patroller's ring one phase later, so the order was written and overwritten inside a single advance.

## What was NOT witnessed

**The four keys were not pressed on screen.** This is the owner's own desktop and it is in use; the
lane's rule is to stop rather than send synthetic input at a window that may not be foreground, and
that is what was done. No key was sent anywhere.

What was driven instead is the shipped dispatch minus the window: `pkg/ui/vocabulary_test.go` parks
a whole `App` on the map screen over a recording loader and drives `App.step` with real key frames
and real button edges, so which key acts on which selection and which seam a press leaves by are
observed on the same path the window would take. **That is not the same as having seen it**, and it
is not written up as if it were. What remains unwitnessed is the binding itself — that
Ebitengine reports `KeyU` for the U key — and the readout row drawn on a real frame.

## Every id, and what witnesses it

| Id | Witness |
|---|---|
| FR-1 Guard | `pkg/sim` `TestGuardPutsTheSelectionInOneGroupAndAnchorsItsPosts`; `pkg/game` `TestTheStanceSeamConvertsTheBoolToAnOrderByte`; `pkg/ui` `TestTheStanceKeysOrderTheWholeSelection`; the drive's guard row above |
| FR-2 Stand Ground | `TestStandGroundIsItsOwnOrderOverTheSameShape` and the same three siblings |
| FR-3 March | `TestMarchIsMoveWithTheOtherOrderByte`, `TestTheMarchSeamConvertsTheBoolToAKind`, `TestAnArmedAimedOrderSpendsOnTheNextSecondaryPress`; the drive's march row |
| FR-4 Patrol | `TestPatrolBuildsARingFromWhereEachMemberStands`, `TestPatrolClampsTheFarCellIntoTheMap`, and the same two siblings; the drive's patrol row |
| FR-5 a player order ends a patrol | `TestAnyPlayerOrderEndsAPatrol` (five arms) and `TestAMoveOverAPatrollerReachesTheOrderedCell`; the drive's patrol-then-guard run |
| FR-6 an order names a group | `TestAGroupOrderCollectsByKindAndTag`, `TestTwoDifferentOrdersNeverShareATag` |
| FR-7 the arms are exclusive | `TestTheArmsAreExclusive`, all three sub-cases |
| FR-8 no simulation type in `pkg/ui` | the seam signatures compile against a `pkg/ui` that may not import `pkg/sim` — `internal/archtest`'s allow-map is what enforces it, and it is green |
| FR-9 Swarm is cut | nothing built; `pkg/sim`'s script arm for sub-command 2 is untouched and its own 0096 tests still pass |
| FR-10 the readout row | `TestTheReadoutStatesWhichOrderIsArmed`; the layout's own `TestAuthoredPanelLayoutStatesExactlyTheEnumeratedFields` covers the number space |
| AC-1 | `TestGuardPutsTheSelectionInOneGroupAndAnchorsItsPosts` |
| AC-2 | `TestStandGroundIsItsOwnOrderOverTheSameShape` |
| AC-3 | `TestMarchIsMoveWithTheOtherOrderByte` and `TestAnAimedOrderIsSpentByThePressWhateverItProduced`'s outside-the-extent case |
| AC-4 | `TestPatrolBuildsARingFromWhereEachMemberStands` |
| AC-5 | `TestAnyPlayerOrderEndsAPatrol` + `TestAMoveOverAPatrollerReachesTheOrderedCell` |
| AC-6 | `TestTheArmsAreExclusive` |
| AC-7 | `TestTheFourKeysAreBoundOnceEach` — a source scan of `app.go` and `viewer.go`'s parsed syntax |
| AC-8 | `TestAMapWithNoStanceSeamTakesEveryKey` |
| P-1 | `vocPending` asserts the digest is unmoved by issuing, per seam |
| P-2 | the two seam types carry two booleans and no third value type |
| P-3 | no byte-form field, version or digest layout changed; `pkg/sim`'s whole binary suite is unchanged and green |
| DD-1 three kinds | `TestEveryGroupKindIsDispatched` |
| DD-2 the generalisation in `groupOrder` | same test; a kind `isGroupKind` admits with no arm builds no command group and fails it |
| DD-3 posts beside the write | `TestGuardPutsTheSelectionInOneGroupAndAnchorsItsPosts`'s post assertion |
| DD-4 `clearPatrol` in `commandGroup` | `TestAnyPlayerOrderEndsAPatrol`, which covers every caller |
| DD-5 two seams | `pkg/game`'s two conversion tests, one per seam |
| DD-6 one armed byte | `TestAnAimedKeyTogglesAndReplaces` |
| DD-7 `decide`'s branch | `TestAnUnarmedPressIsStillThePlainMove` beside `TestAnArmedAimedOrderSpendsOnTheNextSecondaryPress` |
| DD-8 the tag scan asks the kind | `TestTwoDifferentOrdersNeverShareATag` |

## What was checked by reverting it

The rule is that a line is witnessed only if removing it fails something.

- **`clearPatrol` in `commandGroup`** (FR-5) — removed: six failures,
  `TestAnyPlayerOrderEndsAPatrol`'s five sub-cases and
  `TestAMoveOverAPatrollerReachesTheOrderedCell`. Restored, green.
- **The March binding** — moved from `KeyR` to `KeyM`, which is already the minimap toggle:
  `TestTheFourKeysAreBoundOnceEach` failed with *"ebiten.KeyR is named 0 times, want exactly 1"*.
  Restored, green.

The first revert also found the test that did NOT witness: an earlier draft of
`TestAnyPlayerOrderEndsAPatrol` advanced one tick after the order and passed with the clear removed,
because the actor pass runs on one tick in sixteen. It advances two whole decision phases now, and
that is why.

## What was checked and did not hold up

- **A premise handed to this lane was wrong and is corrected here.** `PIPELINE-STATUS.md` names the
  six as *"stance, patrol, escort"*. There is no escort arm. The six with an arm in this build are
  Guard, Swarm, Stand Ground, Move, Swarm 2 and Patrol; the four PUBLISHED sub-commands with no arm
  are `10 Attack`, `11 Defend`, `15 Follow` and `17 Roam` (`AI-GROUPCMD-020`, `AI-ORDER-010`), which
  is where "four this build does not implement" comes from and why it is a different set.
- **The line numbers in that status row are stale**, as handed. `pkg/sim/script.go:897-911` is a
  relation check now; the dispatch is at 1165.
- **`Viewer.command` already existed as a method**, so the armed-order field is named `aimed`. The
  clash was found by the compiler, not by review.
- **`PanelField` numbers are one shared space between the readout and the unit panel**, and 28 and
  30 were both taken. The story's row is 53. `TestAuthoredPanelLayoutStatesExactlyTheEnumeratedFields`
  is what found it.

## Not done

- No panel button for any of the four. What the original's control panel binds is not decoded, and
  binding a guess would be inventing a placement claim.
- Swarm has no key (FR-9).
- `builds/` — a worktree has none; `builds/current/` is rebuilt from master at the landing.
