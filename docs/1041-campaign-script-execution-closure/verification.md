# Story `1041` — verification

This file accounts for every FR and DD in `spec.md`. Commands were run from this story's worktree;
the final gate repetition is on the committed tree reported in the lane return.

## Contract-to-evidence map

| Requirement | Evidence | Verdict |
|---|---|---|
| FR-1, shipped denominator | `cmd/scriptcoverage` loads 28 maps/root, refuses the root-specific 680/678 checks, 759 instants and 398/397 triggers, and requires the 52 active operation keys. It does not reject an extra reachable unknown key (`DIV-387`). | PASS with typed witness debt |
| FR-2, complete observer | `TestEveryCheckDispositionIsRecordedAtItsOwnBoundary`, `TestEveryTriggerDecisionAndShortCircuitIsRecorded`, `TestInstantDispatchRecordsSubcommandAndStateOutcome`; legacy projections remain covered by the earlier trace tests. | PASS |
| FR-3, ordinary drives | The reviewed branch reports 180 exact instant nodes and the landing merge reports 181 on both roots, above the unattended 147; every compiled runtime check and accepted trigger has a natural pass record. | PASS |
| FR-4, controlled seam | `TestAControlledProgramRunsOnlyThroughTheOrdinaryScriptPass`, `TestAControlledProgramKeepsMissionStateOutsideTheProgram`, `TestControlledScriptWorldRequiresBothInputs`; all 545/543 check, 614 instant and 89 group exact-node rows cross `NewScript` plus `StepTraced`. | PASS |
| FR-5, effect oracles | Every exact runtime node runs the current state/no-op oracle before aggregation. Message 2 samples the production announcer; seeded and unmodified paths are distinct. Instant 30 is `DIV-385`; `DIV-387` enumerates the check, trigger, presence, duplicate-effect and group clauses those current oracles cannot discriminate. Instant 29 now cites both `TRIG-EFFECTTIME-034` and its active width replacement `TRIG-CELLEFFECT-045`. | PASS with typed mechanics and witness debts |
| FR-6, synthetic isolation | Nine synthetic rows/root: three exact instant-20 nodes, one exact group-1 node, separate 17 and 18 absences, and three unreachable parameter branches. The test refuses the seven named case classes and dispositions; the three group-field assertions remain bounded by `DIV-387`. | PASS with typed debts |
| FR-7, durable gate/disposition | `TestReleaseCampaignScriptExecutionClosure` is registered in `internal/gatedtests/testdata/population.txt`; both-root results and four RED mutations are below. Those mutations prove selected boundaries, not the seven missing discriminants in `DIV-387`. | PASS with typed witness debt |
| DD-1, return-only observation | `TestObservingAWorldMovesNoByteOfIt` compares traced and untraced bytes and hashes after every step; `TestScriptGroupObservationIsACopyOfTheNamedRecords` mutates a returned copy and rechecks both. | PASS |
| DD-2, real mission plus exact record | The command starts through `game.StartMission`, then verifies the returned slot/index/opcode/sub-command against the exact compiled record installed through `NewScript`. Observer-identity mutation M3 is RED. | PASS |
| DD-3, dispatch is not semantics | `ScriptInstantRun` reports identity/support/hash only; operation-specific functions inspect the state owner. M2 and M4 discriminate two selected effects; `DIV-387` names the still-coarse postconditions and mutations owed. | PASS with typed witness debt |
| DD-4, root isolation | EN and RU are separate command invocations and CSV root values. Their operation sets agree while their compiled check/trigger totals remain different. | PASS |
| DD-5, synthetic does not promote | CSV has distinct `node`, `operation`, `synthetic`, `present`, `accepted`, `reachable`, `natural` and `controlled` fields. Synthetic rows never enter `validateDenominator`. | PASS |
| DD-6, builder-only forms | 135 constants/root prove the preset register. The 28 drops/root prove runtime absence and `ScriptNone`, but do not inspect `DropCells` or its coordinate (`DIV-387`). | PASS with typed witness debt |
| DD-7, no byte-form state | No version or form layout changed. Existing `TestTheScriptAndItsStateCrossTheByteForm` covers program/register/latch/counter/outcome; the traced/untraced identity test proves the observer adds no bytes. | PASS |

## Both-root release result

Direct command results:

| Root | Maps | Checks | Instants | Triggers | Operation rows | Synthetic rows | Natural exact instants |
|---|---:|---:|---:|---:|---:|---:|---:|
| EN | 28 | 680 | 759 | 398 | 52 | 9 | 181 |
| RU | 28 | 678 | 759 | 397 | 52 | 9 | 181 |

The seat reran the command on the landing merge. Integration adds `41.alm` node 3, instant 2, to
the natural set on both roots; the other identities, the 52 rows and all dispositions remain the
reviewed result. The branch-local mutation proof below therefore truthfully retains its earlier 180.

`pipeline/check-release-tests.sh` measured 47 gated invocations: 44 selected through
`AGAINROM_ASSETS`, one through `AGAINROM_ORIGINAL_SAVES`, and two through `AGAINROM_SAVE_666`.
Both EN and RU ran 47 of 47 and passed, with 0 skipped.

The landing merge selected and passed 49 of 49 on each root, with 0 skipped; the larger population
includes install-gated tests already on master.

`pipeline/check-scenarios.sh` selected and passed 15 of 15 scenarios on EN and 15 of 15 on RU.

The repository's fail-closed registries caught the two additions before this result was accepted:
`cmd/scriptcoverage` is registered with only `pkg/formats/alm`, `pkg/game`, `pkg/mapload` and
`pkg/sim`, and the release test is in the gated-test population manifest.

## Mutation proof

All four mutations were made at production sites, one at a time, against the EN lawful root. The
command was always:

```text
AGAINROM_ASSETS=../gameversions/en go test ./cmd/scriptcoverage \
  -run '^TestReleaseCampaignScriptExecutionClosure$' -count=1 -v
```

| ID | Mutation | Release failure | Restore proof |
|---|---|---|---|
| M1 | Remove instant 4 from `scriptInstantSupported` | RED: `operation instant:4 ... trace marks exact opcode 4 unsupported`, naming three authored nodes | `script.go` SHA-256 restored to `2332FCC8F21DB57F32B2B086B965E4F27A05E59130923EDEAE75350B456259B5` |
| M2 | Leave `ScriptInstantLose`'s dispatch case empty | RED: `operation instant:5 ... loss counter increments alone`, naming three authored nodes | same `script.go` SHA-256 |
| M3 | Make `instantDone` report `in.Op + 1` | RED: exact group-2 probes reported opcode 7, wanted 6 | `scripttrace.go` SHA-256 restored to `F7696C832FF66880D811BB2145C0A0AFE5E7B2025ABE0433A7377192265E3E4A` |
| M4 | Suppress `takeOffMap`'s `OffMap = true` store | RED: `operation instant:16 ... loses map presence`, naming three authored nodes | `presence.go` SHA-256 restored to `36E8F5AF67700844693FC66BEA51F8BE053E76B9BC1D583E063DAAD36792E84E` |

After the fourth restore, the same EN release test passed and printed
`maps=28 checks=680 instants=759 triggers=398 rows=52 synthetic=9 natural_exact_instants=180`.
No mutation byte remains in the branch. These four mutations do not claim to cover the seven
full-population discriminants now listed in `DIV-387`.

## Repository and implementation gates

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')`: no output.
- `go test -trimpath -count=1 ./...`: exit 0; every package passed.
- `scripts/check-claim-citations.sh`: exit 0; 1287 distinct citations resolve against 1479 claims
  and 220 experiments under 787 prefixes.
- `scripts/check-no-game-assets.sh`: exit 0, clean tree scan.
- `scripts/campaign-sweep.sh --check`, once per root: exit 0; 28 of 28 missions drove,
  `unsupported=0`, `reached=0`, and two sweeps of each root printed byte-identical tables.

## Seat gates

- `pipeline/check-preserved-installs.sh`: 162 files, both roots as recorded.
- `pipeline/check-div-claims.sh` with `AGAINROM_IMPL` naming this worktree: exit 0; 235 of 235 live
  rows have nine cells and cite 302 distinct ids. It selected `DIV-385` and `DIV-386` from this tree
  alongside `DIV-387`, at research pin `d7ee0c6`.
- `pipeline/check-release-tests.sh`: 47 of 47 ran and passed, 0 skipped, on each root.
- `pipeline/check-scenarios.sh`: 15 of 15 on each root.
- `pipeline/check-milestone.sh`, pointed at a `missionrun` built from this worktree: unchanged from
  `pipeline/milestone-baseline.txt` over all 28 maps and both roots. The mission-10 drive still loses
  at tick 240 by design and still reports 4 of 36 moved, 1 fell.

The lane instruction's direct smoke count agrees with the census:

| Root | Mission 10 `UNSUPPORTED` | Mission 20 `UNSUPPORTED` | Baseline | Change |
|---|---:|---:|---:|---:|
| EN | 0 | 0 | 0 / 0 | unchanged |
| RU | 0 | 0 | 0 / 0 | unchanged |

This story was not meant to reduce the support-gap counter: master already carried zero unsupported
nodes. Its outside-test result is the new 52-row execution/effect matrix, backed by 1,411 exact
execution rows on EN and 1,409 on RU, with 51 current-oracle PASS labels and one typed shipped
divergence per root, plus the separate nine-row synthetic section. `DIV-387` bounds the semantic
strength of those PASS labels. The old support census is unchanged and remains labelled support.

## Research and divergence gates

Every claim used by a postcondition was read through the pinned claim tool. The partial retractions
that matter are dispositioned in `closure.md`; `TRIG-CELLEFFECT-045` replaces both superseded
instant-29 width clauses while `TRIG-EFFECTTIME-034` retains the all-match/duration clauses.
`check-claim-citations.sh` and `check-div-claims.sh` both pass against this worktree.

`DIV-385` carries the complete four-observation shipped population: two exact nodes in `90.alm` on
each root. `DIV-386` carries a zero-node shipped population and one synthetic literal-17 witness per
root. Catalogue literal 18 remains its own PASS row. `DIV-387` carries one repeated W class over all
seven weak-oracle populations; independent inspection found their mechanics correct and no item
produced a P. No id beyond the reserved range was used, and no `formatVersion` was allocated.

## Review stopping surface

Fresh pass 1 ended with P=0, one repeated W class and one D correction. The D is applied to instant
29's metadata and prose; the W is the full-population `DIV-387` row and corrected completeness
wording. Its review remaining-surface list is empty, so the chain stops without pass 2. The earlier
`DIV-385` and `DIV-386` mechanics findings remain bounded follow-ups, not new pass-1 findings.
