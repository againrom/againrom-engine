# 0163 — verification

Branch `0163-campaign-drive` from `041b773`. Research pin unchanged at
`e1b27fd5f682d8e6f29ba3e8683b4f7cb82e7e97`, verified with `git submodule status`
showing no leading character.

**Nothing was watched on a screen.** No windowed game was launched. Every result
below comes from `--headless` runs, from `cmd/missionrun`, and from `go test`.

## Gate

```
go build ./...                     clean
go vet ./...                       clean
gofmt -l $(git ls-files '*.go')    prints nothing
go test -trimpath -count=1 ./...   ok, 25 packages, 7.4s
scripts/check-no-game-assets.sh    PASS
scripts/check-doc-budget.sh        PASS
scripts/check-hotfix-ledger.sh     PASS
scripts/check-sdd-audit-selftest.sh PASS
scripts/check-sdd-audit.sh         no FAIL rows
```

`check-sdd-audit.sh` was run from a worktree, which has no `builds/`, so its note
and warning counts are not comparable and only the FAIL set is reported. There
are no `Co-Authored-By` trailers:

```
$ git log --format='%h [%(trailers:key=Co-Authored-By)]' 041b773..HEAD
1b00ac0 []
ecc7ed1 []
```

## The script-gap census, before and after

`pipeline/milestone-baseline.txt` carries 17 unrunnable arms for mission 10 and
11 for mission 20. Measured from this story's own build:

```
$ AGAINROM_ASSETS=<en> missionrun.exe -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
17
$ AGAINROM_ASSETS=<en> missionrun.exe -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
11
```

**Both are unchanged.** This story implements no script operation and was not
meant to move either number.

## The result someone can point at

The story's result is in `builds/0163-campaign-drive/`, not in the census. Three
things exist that did not before.

**A number nothing had measured.** `scripts/campaign-sweep.sh` drives all 28
campaign missions and reports **506 compiled script gaps and 10072 reached** over
the campaign. Before this story two campaign maps had a scenario; the other 26
had never been driven at all. The reached count is the new information: mission
151 alone reaches 3250 unrunnable arms in 2000 ticks, and mission 101 reaches
1509, so the arms this build skips are not rare corners but code the maps walk
into constantly.

**A generated character in a mission.** `scenarios/0163-chargen-mission10.json`
presses NEW GAME on the brooch, chooses the mission 10 row, and generates a
female mage named Aeryn through the production generation screen:

```
[9] assert_member  screen=map members=1 purse=100 docs=0
    hero Aeryn        entity=35 xp=1593 defense=23 absorption=0 skills=Air=10(1593)
```

**A crossing.** `scenarios/0163-mission-to-town.json` drives mission 20 to its
verdict, crosses into the town, and walks out of the gates into mission 30 with
the roster intact: purse 600 on the map and 1600 in the town, one document on
both sides, the hero at 1795 experience on both sides, and six party members in
the town where the map had one.

Both scenarios and the sweep run on **both** preserved roots.

## Requirements

**FR-1, FR-2, FR-3** — `create_character` in `pkg/game/headlesschargen.go`.
Witnessed by `TestCreateCharacterReachesTheRequestedCharacter`,
`TestCreateCharacterRefusals` and, on a lawful install, by
`scenarios/0163-chargen-mission10.json`. FR-1's "same code path" claim is the
subject of AC-2 below.

**FR-1a** — `App.headlessMenuButton`, reached through `activate` on the menu
screen. Witnessed by `scenarios/0163-chargen-mission10.json`, whose first two
steps are `activate "NEW GAME"` and `activate "Mission 10: 10.alm"`, and which
loads no save.

**FR-4** — `HeadlessState.Chargen`, filled in `headlessAppSnapshot`. Witnessed by
`TestChargenStateIsReportedOnlyOnTheGenerationScreen`.

**FR-5** — front-end `wait_until`, `headlessWaitFrontEnd`. Witnessed by
`TestFrontEndWaitUntil` and `TestUntilFormsAreStageSpecific`.

**FR-6** — met by `scenarios/0163-mission-to-town.json`, using commands that
already existed plus the menu arm DD-12a added. Recorded above.

**FR-7, FR-8, FR-9** — `scripts/campaign-sweep.sh`. Recorded under AC-9 to AC-11.

**FR-10** — `HeadlessScenarioVersion` is 3 and the version is recorded per stage.
Witnessed by `TestCreateCharacterStepValidation` and by the existing
`TestScenarioVocabularyIsGatedByVersionAndStage`, which gained two cases and had
one rewritten (see *Two test changes* below).

## Acceptance criteria

**AC-1** — `scenarios/0163-chargen-mission10.json`, output above. The hero is
named Aeryn, is a mage, holds Air at level 10, and stands in mission 10.

**AC-2** — `TestCreateCharacterMatchesTheHandDrivenScreen`. The same choices are
made twice, once by `create_character` and once by a written-out sequence of
up/down/enter/typed presses, and the two `ui.ChargenResult` values are compared.
**This test failed when first written**, and it failed on the hand-written
sequence rather than on the step: `Spirit`'s up control is focus 12 and the
sequence pressed focus 11, its down control, so the hand drive reached 22 where
the step reached 28. The test is doing the job it exists for.

**AC-3** — `TestCreateCharacterRefusals`, cases *an unaffordable statistic*, *a
statistic above its ceiling* and *a statistic below its floor*. Each case also
asserts that nothing was handed over and that the generation screen is still
showing.

**AC-4** — `TestCreateCharacterNamesTheOfferedLabels` checks that a refusal lists
all four pictures.

**AC-5** — `TestChargenStateIsReportedOnlyOnTheGenerationScreen`.

**AC-6** — `TestFrontEndWaitUntil`. The ceiling case asserts that the error names
the condition, the ceiling and the screen it stopped on. The `notice` form is
also witnessed on the install: step 3 of `0163-mission-to-town.json` stops on the
tick the notice opens, which is what replaced 0152's fixed `wait_ticks 64`.

**AC-7, AC-8** — `scenarios/0163-mission-to-town.json`, recorded above. Step 25
enters mission 30 and step 27 asserts the hero is the same character the town
held.

**AC-9** — the sweep prints 28 mission rows and a totals row; every row carries
an outcome word and a digest. Full table in `builds/0163-campaign-drive/`.

**AC-10** — `bash campaign-sweep.sh --check` printed:

```
deterministic: two sweeps over <seat>\gameversions\en printed the same table
```

**AC-11** — with `--missions "10 999"`:

```
    999  DID NOT DRIVE: againrom: headless: read scenario/999.alm: readfile scenario/999.alm: file does not exist
```

The sweep printed mission 10's row, kept going, and exited 1. With
`--missions "10 20"`, where both missions end `undecided`, it exited 0.

**AC-12** — every scenario written before this story validates and runs
unchanged; none was edited. `TestCreateCharacterStepValidation` covers the
refusal of a version-2 file naming `create_character`.

## Properties

**P-1** — `pkg/sim` is untouched by this branch. `ui.headlessNow` is a fixed
`time.Unix(0, 0)`, the same timestamp `HeadlessKey` and `HeadlessStep` already
used, so no dispatch added here reads a clock. `internal/archtest`'s source scan
over `pkg/sim` passes.

**P-2** — `formatVersion` is unchanged at 46. No save format and no hashed
simulation state was touched, and the sweep's digests are identical across two
runs.

**P-3** — `cmd/missionrun` has no change on this branch, so
`pipeline/check-milestone.sh` measures what it measured.

**P-4** — `pkg/game` calls no method of `ui.Chargen`. Every state change in
`headlesschargen.go` goes through `headlessChargenPress`, which is
`App.HeadlessKey`, which is `App.step`. AC-2 is what tests this rather than
asserting it.

**P-5** — `pkg/game` gained no import. `pkg/ui` gained `againrom/pkg/render/menu`
in `headless.go`, which `pkg/ui` already imported elsewhere for `menu.Assets`, so
the import graph is unchanged in direction and `internal/archtest` passes.

## Design decisions

**DD-1, DD-2, DD-3, DD-4** — `pkg/ui/chargen_headless.go`. `preChoiceParts` is
lifted out of `Chargen.Forward` and used by both, so the picture labels and the
identity `Forward` commits are one statement. Witnessed by
`TestChargenSkillLabelsFollowTheChosenClass`, which shows the detailed page's
skill labels following the class the pictures committed.

**DD-5** — `App.HeadlessType` dispatches one character per `App.step` call.
`headlessChargenName` re-reads the field and fails when what was typed is not
what the field holds.

**DD-6, DD-7** — one step, in the screen's own order. The order is load-bearing
and DD-3's test above is what shows it: the skill options do not exist until the
class is committed.

**DD-8** — `headlessChargenWhy` reads `App.HeadlessMessage`, so a refusal reports
the screen's own sentence. `TestCreateCharacterRefusals`'s *a reserved name* case
asserts the setup's own `ReservedName` wording comes back.

**DD-9, DD-10** — `headlessCommandSpec.stages`. Witnessed by
`TestCreateCharacterStepValidation` and the two new cases in
`TestScenarioVocabularyIsGatedByVersionAndStage`.

**DD-11** — `HeadlessUntil.validate(stage)`. `TestUntilFormsAreStageSpecific`
covers all four cross-stage refusals plus both arity refusals and an unknown
screen name. `TestScreenNamesRoundTrip` checks every screen name resolves.

**DD-12** — `headlessWaitFrontEnd` calls only `App.HeadlessStep`. That the notice
survives the wait is what `0163-mission-to-town.json` depends on: its
`activate "notice"` steps would have nothing to dismiss otherwise.

**DD-12a** — `App.headlessMenuButton`. The point is found by scanning window
positions against `App.buttonAt`, the production hit test, and an install whose
brooch has no region for a button gives an error rather than a press onto the
background.

**DD-13** — the sweep runs `cmd/againrom --headless` over a generated mission
scenario per mission, so it exercises the scenario language over all 28 maps.

**DD-14** — `--ticks`, default 2000. Measured cost: 20 seconds for 28 missions.
At 20000 ticks mission 10 costs 0.5 seconds, so the default is not chosen for
speed alone; it is chosen because nothing decides either way.

**DD-15** — the mission list is the same 28 `check-milestone.sh` walks, repeated
in the script because a lane worktree has no `pipeline/` beside it. `--missions`
overrides it.

**DD-16** — exit codes, witnessed under AC-11.

**DD-17** — `--check`, witnessed under AC-10.

## Scope cuts

**SC-1** — the sweep carries no roster between missions and drives each in its
own process. Campaign continuity is witnessed by `0163-mission-to-town.json`
instead.

**SC-2** — the sweep issues no orders. Every mission ends `undecided`, which is
recorded and is not asserted.

**SC-3** — `create_character` refuses the legacy one-stage generator by name.
Reachable only from a hand-built setup with no `PreCreate` block; no scenario in
this repository drives one.

**SC-4** — no mouse dispatch was added to the generation screen. The menu brooch
did need one (DD-12a), because it has no keyboard path at all.

## Two test changes, and why

`TestScenarioVocabularyIsGatedByVersionAndStage` had a case *a play command in an
older file*: a version-1 front-end file naming `report`, expecting the refusal to
name the version. Under a per-stage version table it is refused for its **stage**
first, and it can never be refused for its version, because a version-1 file
cannot declare the mission stage at all. The case was kept with its true
expectation and two new cases were added that reach the version gate on a stage
that has one. The gate is tested on a fixture where the thing it tests is
reachable.

## Measured beside the story

**Three of 28 campaign maps produce a different digest on the two preserved
roots** — missions 100, 120 and 140. Every other column of the sweep table —
tick, outcome, compiled and reached gaps, alive and fallen — is identical on all
28 rows across both roots.

This does not contradict 0155's statement that its two mission scenarios produce
the same digest over both roots: missions 10 and 20 still agree, and the
population has grown from 2 to 28.

It is also not evidence of nondeterminism. `scenario.res` is **not byte-identical
between the two roots**:

```
$ md5sum gameversions/{en,ru}/scenario.res     # differ
```

A digest is meaningful only against byte-identical assets, so a cross-root digest
comparison is not a well-posed check, and the interesting fact is that 25 of 28
agree anyway rather than that 3 do not. Why those three differ is not established
here and no cause is claimed.

## Not done

- The sweep does not win missions and issues no orders. A sweep that plays would
  need per-map knowledge this story does not have.
- The generation screen's mouse controls are still unreachable headlessly. Every
  control has a focus path, so nothing is unreachable, but a scenario cannot
  witness the pointer arms.
- `create_character` reaches the two-stage generator only. The legacy one-stage
  model is refused rather than driven.
- No windowed session was run and nothing was observed on a screen.
