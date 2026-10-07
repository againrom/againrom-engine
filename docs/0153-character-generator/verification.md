# 0153 character generator verification

Verification used Go 1.26.1 on Windows amd64. Synthetic tests ran without a game install. Release
checks used the preserved EN and RU roots separately. The implementation submodule remained pinned
at `35337edeb2b31547e168e00de0d5bdb7be5a8411`.

## Automated results

The complete implementation gate passed:

```text
go build ./...                         PASS
go vet ./...                           PASS
gofmt -l $(git ls-files '*.go')        no output
go test -trimpath -count=1 ./...       PASS (all packages)
scripts/check-no-game-assets.sh        clean (tree scan)
scripts/check-doc-budget.sh            PASS
scripts/check-hotfix-ledger.sh         PASS
```

All seven `research/scripts/check-*.sh` scripts passed against the pinned research checkout. The
implementation deletion diff from the story baseline was empty.

The EN and RU roots both passed the generated-character release drive:

```text
AGAINROM_ASSETS=.../gameversions/en go test -trimpath ./pkg/game \
  -run TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity -count=1
ok againrom/pkg/game

AGAINROM_ASSETS=.../gameversions/ru go test -trimpath ./pkg/game \
  -run TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity -count=1
ok againrom/pkg/game
```

The drive launches mission 10 through the captured generator action, selects the first live player
through the Viewer, saves, loads in a fresh front end, and enters the campaign successor. Name,
class, sex, spread, sole level-10 skill, weapon and worn set remained equal. The production sheet
contained 20 rows in each root. The first 19 preview/live rows were byte-identical. The final row
changed only from the honest unplaced `CELL -, -` preview to the live `CELL 17, 66` value.

## Contract evidence

| Contract | Evidence and outcome |
|---|---|
| FR-1; AC-1, AC-12 | `TestPreCreateControlBoundsAndNativePixels`, `TestChargenSourcePlacementsMasksAndFrames`, `TestDetailedStatTextUsesDecodedPlateBoxes` and `TestDetailedSkillUsesSelectedHoverAndPressedSourceStates` passed. They cover one native source placement per control, exact mask hits, keyed black pixels, stat baselines, source button states, hover highlights, the centre crop, framed regions and absence of a yellow overlay. Both lawful roots loaded their own generator payload. |
| FR-2; AC-2; P-1 | `TestChargenAC2StepCostIsTheCostDifference`, `TestChargenAC3ReachableStatesAreLegal` and `TestDetailedPointBuyRefusalsKeepTheWholeProjection` passed. Bounds, cumulative cost, remainder and refusal non-mutation held. |
| FR-3; AC-3; P-2 | `TestChargenPreviewCardCoversEveryGeneratedIdentity` covered all 20 class/sex/skill results. `TestDetailedSkillClicksMoveTheOneSelectedSourceState` exercised sword through bow. Every frame retained one selected source state, one level-10 slot and the declared weapon. |
| FR-4; AC-4, AC-5; P-3, P-6 | `TestChargenPreviewUsesTheConfirmedPartyProjection`, `TestDetailedMessageDoesNotAlterNativeCard`, `TestDetailedControlBoundsAndDollReplacement` and the release drive passed. Generator and Viewer use the shared native 300-by-273 character-panel renderer. Optional Doll layer loss preserved later layers; base loss cleared the old Doll. |
| FR-5; AC-6, AC-7, AC-8, AC-9; P-4, P-5 | `TestEncodeRune`, `TestChargenNamePreservesCaseAndStopsAtTenBytes`, `TestChargenPreCreateOwnsIdentityAndStoredName`, `TestChargenResetPreservesSkillAndReplacesPreview`, `TestPreCreateFlow`, the pre-create double-click tests and `TestDetailedPlayUsesSourceRefusalsAndLaunchesOnce` passed. Reset was idempotent, refusal preserved transient state, Back rebuilt the declared stage, and valid Play launched once. |
| AC-10 | The EN/RU release drive preserved generated identity through launch, save/load and campaign transition. The existing difficulty tests remained green in the full suite. |
| AC-11 | `TestChargenPreviewLeavesPersistentFrontEndStateAlone` and point-buy refusal tests passed. Preview and refusal left world, party, purse, documents, difficulty, saves and the random source unchanged. |

The separate-context input regression initially failed because two rapid Enter activations shared
the pointer double-click latch. `TestPreCreateKeyboardChoiceDoesNotBecomePointerDoubleClick` now
passes: only completed pointer releases participate in the 500 ms gesture.

## Mutation evidence

Two temporary mutations were applied separately and reverted before the final gate.

```text
Mutation: disable the Draft selected-index read in detailedSkillState
TestDetailedSkillUsesSelectedHoverAndPressedSourceStates
FAIL: selected source state pixel was rest state 0, want selected state 2

Mutation: replace the generator RenderCharacterPanel call with a blank native-size image
TestDetailedMessageDoesNotAlterNativeCard
FAIL: generator Card pixel (0,0) differed from the shared production pixel
```

These failures witness the selected-index dependency and the shared Card call required by AC-3 and
AC-4. The restored tests passed before the full gate.

## Plan coverage

Design decisions D-1 through D-12 are exercised by the contract tests above. Success criteria 1
through 10, 12 and 13 passed through their named tests. Success criterion 11 is the visible result
and the milestone census below. Risks R-1 through R-6 have no remaining observed instance: source
geometry, stored-byte conversion, preview purity, shared Card rendering, stale preview replacement
and input replay each have a discriminating regression.

## Visible and milestone result

The 640-by-480 EN frame was rendered headlessly from the lawful install at pre-create and at each of
the five fighter skill selections. The pre-create frame showed each of the four choices once. The
five detailed frames showed readable complete Card rows, five separate skill states and the matching
weapon/Doll. The owner inspected the running game after these corrections and accepted the visual
result. Temporary render files were removed after inspection.

The story changes front-end presentation rather than mission scripts. The missionrun census is
unchanged from `pipeline/milestone-baseline.txt`:

```text
mission 10 unsupported nodes: 17 (baseline 17)
mission 20 unsupported nodes: 12 (baseline 12)
```

## Conclusion

FR-1 through FR-5, AC-1 through AC-12, P-1 through P-6, D-1 through D-12 and success criteria 1
through 13 have executable or manual evidence. No residual product failure was observed. The
character generator is merge-ready.
