# Story `1044` — verification

This record accounts for every behaviour and design decision in `spec.md`. Commands ran from the
story worktree. The production result is the read-only original-save drive below; the campaign
support census is deliberately unchanged.

## Contract-to-evidence map

| Requirement | Evidence | Verdict |
|---|---|---|
| B1, complete typed import | `TestCampaignProjectionDecodesEveryTypedAxis` and `TestCampaignProjectionAxesVaryIndependently` exercise the record in serializer order. | PASS |
| B2, bounds and atomicity | `TestCampaignProjectionRejectsMalformedSetsWithoutAValue`, `TestCampaignProjectionIsDetached` and the parser fuzz seed population cover count ceilings, announcement/hire/first-point booleans, document kinds, marker length/NUL classes including zero count and an exactly one-NUL-byte empty picture, exact exhaustion and copied nested slices. | PASS |
| B3, replacement before consumers | `TestRestoredCampaignReplacesFreshDefaults`, `TestEveryRestoredPreTownMainStaysClosedUntilTheCampaignBoundary`, `TestCampaignProjectionRejectsEveryDeadCandidateAtomically`, `TestActiveCampaignMissionMustBeTheSelectedLiveRecordOnBothRestorePaths`, `TestPreTownCampaignCannotEnterThroughATownOnlySave` and `TestEmptyOriginalCampaignMarkerPictureIsRejectedAtomically` cover replacement, every pre-town main, defeat/save/boundary entry, complete record joins, malformed original-marker refusal and assignment-once on both restore paths. The EN/RU drives below prove the importer reads unequal shipped values. | PASS |
| B4, restored progression | `TestCompletingRestoredSideReturnsToMainWithoutLowerReplay`, `TestCompletingRestoredMainRebuildsTheNextRegistryRecord`, `TestRestoredMainGuardRunsBeforeMapIO` and the transition-reward regression test cover side removal, main advancement, child aging, registry reload, the lower-main guard and authored payment. | PASS |
| B5, destructive consumers | `TestRestoredAcceptanceConsumesCandidateBeforeAnnouncement`, `TestRestoredSchoolAndShopCandidatesFlowThroughProductionScreens` and `TestRestoredOneShotGrantsDocumentsAndSelectionMutateProjection` cover paired inn mutation, production school/shop removal, announcement order, one-shot hero grants, pair-deduplicated documents and selection. | PASS |
| B6, world-map position | `TestRestoredMapPointAndImportedMarkerReachTheProductionWorldMap` covers first and selected points plus an imported marker through `enterWorldMap` and `WorldMapView`. `TestSelectionAfterZeroMarkerImportRoundTripsToTheProductionWorldMap` covers selection producer, envelope and restored screen. `DIV-418` bounds the unnamed original marker dwords. | PASS with typed evidence limit |
| B7, againrom persistence | `TestRestoredCampaignRoundTripsThroughAgainromEnvelope`, `TestSelectionAfterZeroMarkerImportRoundTripsToTheProductionWorldMap`, the cross-field atomic tests and `TestReleasedEnvelopeAtTheCurrentSimulationFormFixture` cover all three additive gob fields, legacy zero value, atomic refusal and the changed outer fixture. Simulation form 61 stays unchanged. | PASS |
| B8, accurate disclosure | `TestTheOriginalSaveCaveatIsSayableOnOneLine` checks the load-row disclosure and its 104-column bound. The three production drives print the same corrected note. The between-mission branch reports restored main/selected state or the absence of a campaign record. | PASS |
| DD1, copied format boundary | Parser detachment plus game-layer mutation tests show no original byte or parser-owned slice becomes live state. | PASS |
| DD2, complete candidate | Parser, candidate-array, active-location and pre-town town-only refusal tests retain the prior front end on both original and `.ags` paths. | PASS |
| DD3, immutable definition | Main-rebuild and replacement tests keep `Campaign` as install definition and mutate only `campaignProgress`. | PASS |
| DD4, registry-derived reload | The restored-main test reloads the next record and candidates from a synthetic registry with values unequal to fresh defaults. | PASS |
| DD5, early lower guard | `TestRestoredMainGuardRunsBeforeMapIO` supplies an opener that fails if archive I/O occurs. | PASS |
| DD6, separate latches and candidates | The acceptance-order test checks candidate removal before record announcement and paired inn removal. | PASS |
| DD7, unnamed values stay unnamed | `TestUnnamedCampaignScalarsNeverEnterProjection` varies both framing scalars without changing canonical projection; marker dwords have no game-layer meaning. | PASS |
| DD8, headless production seam | `savecheck -campaign-progress` calls the production restore and `FinishMission` seams. It opens no GUI and invents no world-half outcome. | PASS |

## Fixed original-save discriminator

The worktree-built `savecheck` loaded the owner's read-only EN `game0020.sav`, labelled
`we have brian !!`, through the production original-save importer. Before completing mission 41:

```text
restored=true main=50 selected=41 children=[41 51] announced=[41]
inn=[0 50 51] shop=[] school=[] documents=4 markers=[] mission-time=1789
```

After the live completion seam:

```text
next=0 offered=50
restored=true main=50 selected=50 children=[51] announced=[] inn=[0 50 51] available=[]
```

The party listing contains one `stable=player:brian` row, with the twelve worn slots
`[8486 0 0 0 0 9770 10032 10291 10550 10810 0 11326]`. Neither mission 30 nor mission 40 is
offered, and the production lower-main guard refuses 40 before map I/O. No Brian-specific rule is
present.

Two independent controls exclude fixture constants:

| Root/save | Before | Completion | After |
|---|---|---|---|
| EN `game0014.sav` | main 20, selected 20, no children, announced 20, documents 3, time 217 | authored transition reward `+500`, offered 30 | main 30, child 31, inn 30 |
| RU `game0001.sav` | main 10, selected 10, no children, no announcement, documents 3, time 0 | next 20 | main and selected 20 |

The first EN scenario run exposed a player-visible regression: restored mission 20 omitted its
authored transition reward. Commit `d8db7d57` restores that payment and the exact final EN and RU
scenario populations pass.

All three current drives print the corrected production disclosure:

```text
ORIGINAL SAVE: party, positions, stats, items, explored map and campaign load -- mission world restarts
```

## Repository and release gates

- `go build ./...`, `go vet ./...`, gofmt and
  `go test -p 1 -trimpath -count=1 ./...`: pass. The package suite ran serially to stay within the
  host's memory limit.
- `scripts/check-claim-citations.sh`: 1,343 distinct citations resolve against 1,580 claims and 233
  experiments under 829 prefixes.
- `scripts/check-no-game-assets.sh`: clean.
- Research at exact pin `23daf74f`: build, vet, gofmt and full tests pass;
  `check-claim-ids` reads 1,580 ids, `check-regen-out` checks 24 scripts, and
  `check-retraction-status` reads 251 retraction entries.
- `pipeline/check-pin-forward.sh`: story pin `23daf74f` is 12 research commits ahead of master's
  `40ac9aff` pin.
- `pipeline/check-div-claims.sh` against this worktree: 245 live rows have nine cells; `DIV-417`
  and `DIV-418` are selected from this ledger.
- `pipeline/check-scenarios.sh`: 15 of 15 pass on EN and 15 of 15 on RU.
- `pipeline/check-release-tests.sh`: 60 of 60 pass, zero skipped, on each root.
- `pipeline/check-preserved-installs.sh`: expected external RED from the owner's known EN save-set
  drift. EN `game0000.sav`, `game0001.sav` and `game9999.sav` differ in size from the recorded
  standard, and `game0003.sav` through `game0021.sav` are additional. No install was mutated or
  re-recorded by this story.

## Milestone and outside-test result

`pipeline/check-milestone.sh`, pointed at `missionrun.exe` built from this worktree, matches
`pipeline/milestone-baseline.txt` across all 28 maps and both roots. The unattended mission-10 drive
still loses at tick 240 by design and still reports 4 of 36 units moved, 1 fallen.

| Root | Mission 10 `UNSUPPORTED` | Mission 20 `UNSUPPORTED` | Master baseline | Change |
|---|---:|---:|---:|---:|
| EN | 0 | 0 | 0 / 0 | unchanged |
| RU | 0 | 0 | 0 / 0 | unchanged |

This story was not intended to move the script-support census. Its external result is the fixed EN
save drive: campaign state that formerly defaulted to chapter 30 now reads main 50 and side 41,
completes 41 into main 50, retains child 51 and exactly one Brian, and offers neither lower mission.

## Remaining surface

Parser framing covers every malformed field class, including zero-count and sole-NUL marker
pictures. Complete cross-field validation, every restored pre-town main and town consumer,
original import and disclosure, mission completion, the first and selected MapPoint plus marker
producer-to-envelope-to-screen route, and againrom persistence are covered. The only known
evidence-limited surface is the meaning of the three dwords in a non-empty original marker record,
recorded as `DIV-418`; the preserved saves have no such record. World-half continuity and the
independently active token-load, cell-rebind and dead-actor-projection SAV wave remain outside this
story and are neither consumed nor cited.
