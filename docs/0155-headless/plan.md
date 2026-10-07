# 0155 — plan

## Shape

`pkg/game/headless.go` keeps the scenario document: the struct, the reader, the
command table and the front-end runner. `pkg/game/scenario.go` is new and holds
the whole mission stage. `cmd/againrom/main.go` gains one branch and one helper.
No other package changes.

The mission stage is defined over `*sim.World` plus two reference tables and
nothing else. That is what makes it constructible in a test.

## Design decisions

**DD-1 — Two stages, never mixed in one file.** A front-end step's meaning
depends on the menu path above it; a live world exists only where the controller
put one there. Allowing a play step in a front-end file would make every such
step's meaning depend on menu history, and the failure mode is a scenario that
orders a unit in a world other than the one its author had in mind. The stage is
therefore a property of the whole file, checked once at validation, and each
runner refuses the other's scenario outright rather than per step.

**DD-2 — The mission branch returns before the front end is built.** A mission
scenario reaches no menu, no font, no sprite sheet and no save directory.
Building the front end for it would load assets the run never reads and would
make a play scenario fail on an install whose only defect is in its menu
graphics. The branch sits immediately after the scenario is read and before
`frontEnd`. The two routes are distinguishable from outside by their error
prefix, which is what witnesses the ordering.

**DD-3 — The command grammar is a table, not a switch.** The predecessor spelled
each command's accepted parameter set twice: once as a switch arm and once as a
list of field names passed to a helper. A parameter added to one and not the
other is accepted by a validator that then reports it as unexpected. The table
states stage, minimum version, accepted parameters and required parameters once
each, and `present()` is the single place a field name is mapped to a struct
field.

**DD-4 — `PlayWorld` is a plain struct with exported fields.** A constructor that
only a mission start could satisfy would make the language untestable without an
install. `NewPlayWorld` exists for the mission path; a test builds the struct
directly over a hand-made world, which is FR-8.

**DD-5 — A spent ceiling is a failure.** A `wait_until` that returns quietly
after its ceiling leaves every later assertion to report a world that simply did
not move, and the reported failure then points at the assertion rather than at
the walk that never finished. It is also why a verdict reached during a unit
condition is its own failure: after a verdict the world accepts no further work,
so waiting on can only spend the ceiling.

**DD-6 — Unit rows only on `report`.** A run of four thousand ticks past a
hundred units would otherwise write four hundred thousand rows. The summary is on
every event; the per-entity detail is on the step that asked for it.

**DD-7 — The reference grammar is taken over verbatim, not redesigned.** `uNN`
and `pN` are `cmd/missionrun`'s. A drive written as flags transcribes into a
scenario without re-deriving what a reference means. `eNN` is added because a
world with neither a script table nor a party — the one a test builds — can be
addressed no other way.

**DD-8 — The version gate is a floor per command, not a list per version.** Each
table row carries the version that introduced it, so adding a command is one
field and the refusal message for an older file is derived rather than
maintained.

**DD-9 — The version test names no version.** The case that refuses an unreadable
version is written against `HeadlessScenarioVersion + 1`. A case spelling the
next number has to be re-spelled at every bump, and a stale one passes over
exactly the version it was written to refuse. This project has repaired that
class of test three times in its byte-form version.

**DD-10 — The provider is declared in the file, not chosen by the runner.** A
scenario says what happens; where the bytes come from is a run parameter. Putting
it in the file rather than in a flag means one scenario set, one runner, and a
file that cannot be run against the wrong provider by mistake. It is also what
lets the command skip the asset root entirely for a synthetic scenario: the read
of the scenario moves above the root resolution.

**DD-11 — A synthetic world is authored, never sampled.** Cutting a subset out of
a real archive would be game data in the repository at every commit including
history. The world spec is composed from this project's own contracts, and its
builder fills both reference tables from the refs the units declare, so the same
`uNN`/`pN` grammar addresses an authored world and an installed map alike.

**DD-12 — Tracing is unconditional on the mission stage.** `sim.StepTraced`
returns a value nothing stores on the world and nothing enters in the digest, so
a traced run and an untraced one are the same run. Making the census a mode would
have made it a mode nobody remembers to turn on.

**DD-13 — The reached count is separate from the compiled count, not a
replacement.** They answer different questions and a story that reported only one
would lose the other. The compiled count is what `pipeline/check-milestone.sh`
tracks and can be compared to its baseline; the reached count is what a
playthrough actually blocks on.

**DD-14 — The shipped install scenarios record the outcome and do not assert it.**
Asserting a verdict on an unattended drive measures the drive, not the game. This
project asserted mission 10's outcome as a milestone for eight days on exactly
that mistake. A scenario whose own orders make the verdict deterministic may
assert it; these do not, so they report it.


## Where each requirement lands

| Requirement | Code |
|---|---|
| FR-1 | the stage constants and the `stage` column of `headlessCommands`, `pkg/game/headless.go` |
| FR-2 | `StartScenarioMission`, `pkg/game/scenario.go`; `runMissionStage`, `cmd/againrom/main.go` |
| FR-3 | `HeadlessStep.validateOrder` and `HeadlessStep.commands` |
| FR-4 | `HeadlessUntil` and `waitUntil` |
| FR-5 | `HeadlessUnitAssertion.check` and `HeadlessWorldAssertion.check` |
| FR-6 | `HeadlessWorldState`, `HeadlessUnitState`, `RunPlayScenario`'s two writers |
| FR-7 | `HeadlessScenarioVersion` and the `version` column of `headlessCommands` |
| FR-8 | `PlayWorld`'s exported fields |
| FR-9 | `PlayWorld.Resolve` and `headlessRefParts` |
| FR-10 | `runMissionStage`'s root argument, resolved by `run` |
| FR-11 | `AssetSource`, `HeadlessWorldSpec.Build`, the synthetic branch of `run` |
| FR-12 | `playCensus` and `PlayWorld.step` |
| FR-13 | `HeadlessWorldState.Hash` |
| FR-14 | the shipped install scenarios, which end in `report` |

## Steps

1. `pkg/game/headless.go`: the stage constants, `HeadlessScenarioVersion`, the
   new scenario and step fields, the command table, `StageName`, the rewritten
   `Validate` and `validate`, the `World` field on `HeadlessEvent`, and the
   front-end runner's stage guard.
2. `pkg/game/scenario.go`: `PlayWorld` and reference resolution; the condition,
   unit-assertion and world-assertion types with their validators and checks; the
   order grammar; the world state types; `RunPlayScenario`; `StartScenarioMission`.
3. `cmd/againrom/main.go`: the mission branch and `runMissionStage`.
4. `scenarios/0155-mission10-escort.json` and the rewritten `scenarios/README.md`.
5. Tests: `pkg/game/scenario_test.go` for the language, the updated cases in
   `pkg/game/headless_test.go`, and the two new cases in `cmd/againrom/main_test.go`.

## Success criteria

**SC-1** Every file under `scenarios/` parses and validates, and both stages have
a worked example there.

**SC-2** `go test ./...` is green with no game install present.

**SC-3** The shipped mission scenarios, run the way the owner runs the game —
asset root from `AGAINROM_ASSETS` — exit zero against `gameversions/en` and
against `gameversions/ru`.

**SC-6** The shipped synthetic scenario exits zero with no `-assets` flag and no
`AGAINROM_ASSETS` set.

**SC-7** Missions 10 and 20 report a reached count and a digest, and the digest
for each mission is the same over both preserved roots.

**SC-4** The gate is green: `go build`, `go vet`, `gofmt`, `go test -trimpath
-count=1 ./...`, `check-no-game-assets.sh`, `check-doc-budget.sh`,
`check-sdd-audit.sh`, `check-hotfix-ledger.sh`.

**SC-5** Reverting the ceiling arm of `waitUntil` to a quiet return fails
`TestAWaitThatNeverHoldsFailsAndNamesTheCondition`, so DD-5 is witnessed rather
than asserted.
