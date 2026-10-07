# 0155 — verification

## Result someone can point at

`scenarios/0155-synthetic-melee.json`, `scenarios/0155-mission10-escort.json` and
`scenarios/0155-mission20-sweep.json`. The first runs with no game install and no
asset root configured at all; the other two run against either preserved root and
report a number this pipeline did not have before: how many unrunnable script
arms a **driven** mission actually reached.

Measured on this machine, `builds/0155-headless/againrom.exe`:

| Mission | Compiled unrunnable arms | Reached by the drive | Reached breakdown | Digest, en | Digest, ru |
|---|---|---|---|---|---|
| 10, driven to its verdict at tick 224 | 17 | 1 | instant op 2, once | `9d74f90c9238a7ce` | `9d74f90c9238a7ce` |
| 20, 600 unattended ticks | 12 | 1 | instant op 2, once | `1da002033aa16dbf` | `1da002033aa16dbf` |

Instant op 2 is the broadcast no-op. Neither drive walked into a mechanically
unrunnable node. The compiled figures 17 and 12 agree with
`pipeline/milestone-baseline.txt` for `en m10` and `en m20`.

The digest for each mission is identical over `gameversions/en` and
`gameversions/ru`. Both roots were run.

## The milestone census

`pipeline/check-milestone.sh`'s numbers are unchanged by this story, which is the
honest claim: it measures the compiled script gap, and this story decodes
nothing.

Command from the brief, run from this worktree against `gameversions/en`:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   ->  17
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   ->  12
```

Master before this story carried the same two figures: `milestone-baseline.txt`
sums `en m10` to 1 + 2 + 13 + 1 = 17 and `en m20` to 1 + 11 = 12. Unchanged.

## Gate

Run from the worktree on a clean tree.

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test -trimpath -count=1 ./...` | green, no game install read |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-doc-budget.sh` | ok; spec 12498/24576, plan 6973/24576, plan ≤ 1.2 × spec |
| `bash scripts/check-sdd-audit.sh` | FAIL set empty |
| `bash scripts/check-hotfix-ledger.sh` | ok |

The note and warning counts from `check-sdd-audit.sh` are not comparable from a
worktree, which has no `builds/`. Only the FAIL set is reported here.

## Requirements

**FR-1** — the stage split. `StageFrontEnd`/`StageMission` in `pkg/game/headless.go`;
the `stage` field of each row in `headlessCommands`; `Validate` rejects a
cross-stage command. Witnessed by
`TestScenarioVocabularyIsGatedByVersionAndStage`.

**FR-2** — a mission starts with no save file. `StartScenarioMission` in
`pkg/game/scenario.go` and `runMissionStage` in `cmd/againrom/main.go`. Witnessed
by the two install runs in the table above, neither of which names a save.

**FR-3** — unit orders. `HeadlessStep.validateOrder` and `HeadlessStep.commands`.
Witnessed by `TestAScenarioCanWaitForAUnitToFall` (attack) and
`TestAMissionScenarioRunsEndToEndWithNoInstallPresent` (move), and by the two
`order` steps of the mission-10 scenario.

**FR-4** — `wait_until`. `HeadlessUntil` and `waitUntil`. Witnessed by
`TestAWaitThatNeverHoldsFailsAndNamesTheCondition`,
`TestAScenarioCanWaitForAUnitToFall`, and the mission-10 run, whose
`outcome decided after 222 tick(s)` is the third condition form.

**FR-5** — unit and world assertions. `HeadlessUnitAssertion.check` and
`HeadlessWorldAssertion.check`. Witnessed by
`TestAWorldAssertionReadsTheScriptGapCensus` and the assert steps of all three
shipped scenarios.

**FR-6** — the output is evidence. `HeadlessWorldState`, `HeadlessUnitState`,
`RunPlayScenario`'s two writers. Witnessed by
`TestAMissionScenarioRunsEndToEndWithNoInstallPresent`, which reads the ordered
unit's movement out of the JSON stream rather than out of the world.

**FR-7** — versioning. `HeadlessScenarioVersion`, the `version` field of each
command row, the version clause of `Validate`. Witnessed by
`TestScenarioVocabularyIsGatedByVersionAndStage` and by
`TestHeadlessScenarioValidationRefusesMalformedOrAmbiguousSteps`, whose
unreadable-version case is written against `HeadlessScenarioVersion + 1` and so
spells no version number.

**FR-8** — the mission stage runs with no install. `PlayWorld`'s exported fields.
Witnessed by `TestAMissionScenarioRunsEndToEndWithNoInstallPresent`, which writes
a scenario file, reads it through `ReadHeadlessScenario` and runs it through
`RunPlayScenario` with nothing on disk but the scenario.

**FR-9** — the reference grammar. `PlayWorld.Resolve` and `headlessRefParts`.
Witnessed by `TestUnitReferencesResolveThroughScriptPartyAndEntity`.

**FR-10** — the asset root. `runMissionStage` takes the root `run` already
resolved through `game.ResolveAssetRoot`, not `o.assets`. Witnessed by the
install runs above, every one of which supplied the root through
`AGAINROM_ASSETS` and no `-assets` flag.

**FR-11** — the two asset providers. `AssetsInstall`/`AssetsSynthetic`,
`AssetSource`, `HeadlessWorldSpec.Build`, and the synthetic branch of `run`, which
sits above the asset-root resolution. Witnessed by
`TestASyntheticWorldIsBuiltFromTheScenarioItself` and by SC-6 below.

**FR-12** — the reached census. `playCensus`, `PlayWorld.step`, the `Reached` and
`Census` fields, `reached_unsupported_at_most`. Witnessed by
`TestTheReachedCensusCountsWhatTheDriveWalkedInto` and by the reached column of
the result table.

**FR-13** — the digest. `HeadlessWorldState.Hash` from `sim.World.Hash()`.
Witnessed by the last clause of `TestTheReachedCensusCountsWhatTheDriveWalkedInto`
and by the two digest columns of the result table.

**FR-14** — an outcome is recorded, not asserted. The mission-10 scenario ends
with `report`, not with an outcome assertion. Reverting that to
`assert_world: {"outcome": "lost"}` would make the file assert a property of the
drive. The synthetic scenario does assert, because its own order is what fells
the victim.

## Acceptance criteria

**AC-1** `TestAMissionScenarioRunsEndToEndWithNoInstallPresent`.

**AC-2** `TestUnitReferencesResolveThroughScriptPartyAndEntity`.

**AC-3** `TestAWaitThatNeverHoldsFailsAndNamesTheCondition`.

**AC-4** `TestAWorldAssertionReadsTheScriptGapCensus`.

**AC-5** `TestScenarioVocabularyIsGatedByVersionAndStage`, thirteen refusal cases
plus the two runners refusing each other's file.

**AC-6** `TestAScenarioCanWaitForAUnitToFall`.

**AC-7** The mission-10 scenario over both roots: `lost` at tick 224 with one
fallen, which is what `pipeline/milestone-baseline.txt` records for the
`missionrun` drive over the same mission on both roots.

**AC-8** `TestTheReachedCensusCountsWhatTheDriveWalkedInto`: compiled 1 before and
after, reached 0 before and non-zero after, and the per-opcode row accounting for
every arrival.

**AC-9** `TestASyntheticWorldIsBuiltFromTheScenarioItself`, with five refusal
cases; and SC-6.

**AC-10** `TestEveryShippedScenarioParsesAndValidates`, which logs
`shipped scenarios: 1 ran on synthetic assets, 3 skipped for want of an install`.

## Design decisions

**DD-1** Held. `Validate` checks the stage once for the whole file and each
runner refuses the other's scenario. `TestScenarioVocabularyIsGatedByVersionAndStage`.

**DD-2** Held. The mission branch sits above `frontEnd` in `run`.
`TestAMissionScenarioTakesTheMissionRouteAndNotTheFrontEnd` distinguishes the two
routes by their error prefix over the same failing asset root; moving the branch
back below `frontEnd` makes the two cases print the same message.

**DD-3** Held. `headlessCommands` states stage, version, accepted and required
parameters once each; `HeadlessStep.present()` is the only place a parameter name
is mapped to a field.

**DD-4** Held. Every field of `PlayWorld` is exported except the census, and
three tests build one directly.

**DD-5** Held, and mutation-checked: SC-5.

**DD-6** Held. `worldState` takes a `units` argument that only a `report` step
sets true. `TestAMissionScenarioRunsEndToEndWithNoInstallPresent` checks that an
assert step emits no unit rows and a report step emits two.

**DD-7** Held. `headlessRefParts` reads `u`, `p` and `e` and nothing else;
`scenarios/0155-mission10-escort.json` names `u21` and `p0`, which are the same
two strings the milestone drive's argv names.

**DD-8** Held. The refusal message is built from `spec.version`, so no version
list is maintained anywhere.

**DD-9** Held. The unreadable-version case is `HeadlessScenarioVersion + 1`; no
test in this story spells a version number.

**DD-10** Held. `assets` is a scenario field and there is no provider flag. The
synthetic branch of `run` sits above the asset-root resolution, which is why
SC-6 runs with nothing configured.

**DD-11** Held. `HeadlessWorldSpec.Build` constructs entities from the
declaration alone and reads no file. `check-no-game-assets.sh` is clean over the
tree, including the three new scenario files.

**DD-12** Held. `PlayWorld.step` always calls `sim.StepTraced`; there is no
tracing flag. The digests in the result table are the digests of a traced run,
and `pkg/sim/scripttrace.go` states that a trace enters neither the world nor the
digest.

**DD-13** Held. `HeadlessWorldState` carries `Unsupported` and `Reached` as
separate fields and `assert_world` bounds each separately.
`TestTheReachedCensusCountsWhatTheDriveWalkedInto` moves one and not the other.

**DD-14** Held. Neither install scenario asserts an outcome. The synthetic one
asserts, because its own `order` step is what fells the victim.

## Properties

**P-1** No wall clock and no seed of its own. `RunPlayScenario` and `waitUntil`
step fixed counts; the only time source in `pkg/game/scenario.go` would be an
`import "time"`, and there is none. `internal/archtest`'s source scan holds
`pkg/sim` to the same rule beneath it.

**P-2** No file written. `pkg/game/scenario.go` imports no `os` and opens
nothing; the mission and synthetic routes in `cmd/againrom/main.go` set no save
store. The mission stage is also refused `saves` and `original_saves` at
validation, so it cannot be handed a directory to write into. Nothing was written
into either preserved root; `bash pipeline/check-preserved-installs.sh` is the
seat's own measurement of that.

**P-3** Additive across versions. `scenarios/0152-save666.json` is unchanged and
still validates, which `TestEveryShippedScenarioParsesAndValidates` checks on
every run.

## Success criteria

**SC-1** `TestEveryShippedScenarioParsesAndValidates`: four files, both stages
covered, both providers covered.

**SC-2** `go test -trimpath -count=1 ./...` green with no install read. The
install-tier scenarios are skipped and counted, never run, under `go test`.

**SC-3** Both mission scenarios exit zero against `gameversions/en` and
`gameversions/ru`, with the root supplied through `AGAINROM_ASSETS`.

**SC-4** The gate table above.

**SC-5** Mutation check, run and reverted: replacing `waitUntil`'s ceiling arm
with `return "", nil` makes
`TestAWaitThatNeverHoldsFailsAndNamesTheCondition` fail with
`RunPlayScenario = <nil>, want the unmet condition and its ceiling`. The arm was
restored and the suite is green.

**SC-6** `againrom.exe --headless scenarios/0155-synthetic-melee.json` with no
`-assets` flag and no `AGAINROM_ASSETS` set exits zero and reports
`againrom: synthetic world 24x24, 2 unit(s), party of 1`.

**SC-7** The digest columns of the result table: identical per mission across
both roots.

## What was not done

Named in `spec.md`'s scope section: seeding a party to stated statistics, mission
chaining, the 10 to 150 sweep, keying a digest to an install fingerprint, and
authoring a script inside a synthetic world. Two missions were driven, 10 and 20.
`cmd/missionrun` and `pipeline/check-milestone.sh` are unchanged.
