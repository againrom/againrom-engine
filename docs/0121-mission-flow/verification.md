# 0121 — mission flow: verification

Branch `0121-mission-flow` at `94b074c`, off master `67d2e57`. Two trailered commits, `37cac15`
(T1) and `94b074c` (T2), plus the untrailered document commit `3ad2a0e`. Every gate below was run
on the committed tree with `git status --porcelain` empty.

## Acceptance

| Id | Witness |
|---|---|
| AC-1 | `TestWithMissions/mission rows lead, ascending by number, followed by the whole list unchanged` — and the install run below, where 28 mission rows lead 38 unchanged map rows in the order 10, 20, 30, 31 … 150, 151. Numeric ordering is load-bearing there: mission 100 follows mission 91, which a text sort would have reversed. |
| AC-2 | `TestWithMissions/a non-numeric stem yields no mission row`. |
| AC-3 | `TestWithMissions/row text names the mission for a mission row, and a map row's text does not change`. |
| AC-4 | `TestLoadMapRoutesAMissionRowToTheMissionDoor` — a mission row's `loadMap` hands back a non-nil `ui.MapAdvance`, which only `MissionOpener` returns. Reverted below (SC-4). |
| AC-5 | That same test's other half: a map row hands back nil there. `TestAMapOpenedAsAMapOpensNoNoticeAfterManySteps` is the consequence — the world `openMapWorld` builds reaches no outcome and opens no notice however long it is stepped. |
| AC-6 | `TestWithMissions/a mission row over an undecodable entry is present and unchoosable`. |
| AC-7 | `TestMissionWithTheHeroRemovedFromTheWorldReportsALoss`. |
| AC-8 | `TestMissionWithTheHeroDownedButNotDeadReportsALoss`. |
| AC-9 | `TestMissionWithTheHeroAliveAndTheWinningCounterSatisfiedIsWon`. |
| AC-10 | `TestMissionWithTheHeroDeadAndAWinAlreadyReachedReportsTheLoss` — the one that measures the **order** rather than the rule, and the test whose revert output below flips from a loss to `outcome=1`. |
| AC-11 | `TestMissionWithNoPartyReportsNothingEvenWithADeadEntityZero` — entity id 0 is dead in that world and the mission is not lost, which is why the flag and not the id is what the rule reads. |
| AC-12 | `TestAMapOpenedAsAMapOpensNoNoticeAfterManySteps`. |
| AC-13 | `TestTheHeroLossBannerOpensOnceAndDismissingGoesToTheMenu`. |

## Properties

- **P-1** No file under `pkg/sim` is in the diff, and the only world access either half adds is
  `mw.entity`, the driver's existing copy-handing read over `mw.world.Entities()`. The cadence,
  order and draw invariance suites in `pkg/game` are unchanged and green.
- **P-2** `pkg/sim/binary.go`'s `formatVersion` is `26` at both `67d2e57` and `94b074c`, and no
  serialised record gained, lost or widened a field:

```
$ git diff --name-only 67d2e57..HEAD | grep '^pkg/sim/'
(no output)
```

- **P-3** No mission number, map name or campaign extent is written as a constant. `WithMissions`
  derives every row from what the container holds, through `strconv.Atoi` on the entry's own stem —
  `MissionMap`'s `strconv.Itoa` inverted. The two negative subtests fix the boundary from the other
  side, and the install run below produced 28 rows off the shipped container with nothing
  enumerated in source.
- **P-4** `loadMap`'s mission arm is `return f.MissionOpener(e.Mission)()` and nothing else, so the
  picker and the `-mission` flag reach a mission through one closure. Deleting that one statement
  is what SC-4 measures.

## Success criteria

**SC-1** — clean tree, no game install read by any test:

```
$ go build ./...                      # exit 0, silent
$ go vet ./...                        # exit 0, silent
$ gofmt -l $(git ls-files '*.go')     # silent
$ go test -trimpath -count=1 ./...    # every package ok; pkg/game 2.8s, pkg/sim 4.7s, pkg/ui 3.6s
```

**SC-2** — the repo gates:

```
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)                          EXIT=0
$ bash scripts/check-doc-budget.sh
0121-mission-flow: plan <= 1.2 x spec    6332 <=  8002 bytes  ok
0121-mission-flow: tasks <= 1.2 x plan   2773 <=  7598 bytes  ok   EXIT=0
$ bash scripts/check-sdd-audit.sh                                FAIL set empty
```

`check-sdd-audit.sh` was red before this file existed, on exactly one line — `FAIL
0121-mission-flow: every task in tasks.md has landed and there is no verification.md` — and that
was its whole FAIL set. The note and warning counts a lane prints are meaningless here: `builds/`
does not exist in a worktree, so only the FAIL set was read.

Trailer bijection, checked directly rather than taken from a report:

```
$ git log --format='%h %(trailers:key=SDD-Task,valueonly)' 67d2e57..HEAD
94b074c 0121-mission-flow/T2
37cac15 0121-mission-flow/T1
3ad2a0e
$ git log --format='%h [%(trailers:key=Co-Authored-By)]' 67d2e57..HEAD
94b074c []   37cac15 []   3ad2a0e []
```

**SC-3** — witnessed by reverting, not by reading the assertion. The hero branch in
`settleNotices` was deleted and the tests re-run:

```
--- FAIL: TestMissionWithTheHeroRemovedFromTheWorldReportsALoss
    world_test.go:3358: announced=false outcome=0, want true/OutcomeLost - the hero's entity is gone
--- FAIL: TestMissionWithTheHeroDownedButNotDeadReportsALoss
    world_test.go:3371: announced=false outcome=0, want true/OutcomeLost - a downed hero is not living
--- FAIL: TestMissionWithTheHeroDeadAndAWinAlreadyReachedReportsTheLoss
    world_test.go:3412: announced=true outcome=1, want true/OutcomeLost - the hero test runs before Outcome() is read
--- FAIL: TestTheHeroLossBannerOpensOnceAndDismissingGoesToTheMenu
    world_test.go:3464: notice = ""/0/false, want "MISSION FAILED" as an outcome
```

The third line is the interesting one: without the branch that world still decides, and decides
**won**. So the tests measure the rule's position ahead of `Outcome()` and not merely its presence.
Branch restored, all four green.

**SC-4** — the routing arm deleted from `loadMap`:

```
--- FAIL: TestLoadMapRoutesAMissionRowToTheMissionDoor
    frontend_test.go:565: a mission row's loadMap returned a nil MapAdvance; DD-5's delegation to MissionOpener is missing
```

Restored, green.

**SC-5** — the deletion set:

```
$ git diff --diff-filter=D --name-only 67d2e57..HEAD
(no output)
```

## Against a real install

Not a test — a developer run of `cmd/againrom` against the preserved EN root, which no test reads.

```
$ go run ./cmd/againrom -check -assets <en root>
againrom: 66 map rows, 8 of 8 buttons have a mask region; hero Body 43, ...
$ go run ./cmd/againrom -check -mission 10 -assets <en root>
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)
```

66 = 38 map rows + **28** mission rows, which is the campaign's extent measured off the shipped
container by this build. The rows themselves, printed through a throwaway test that was then
removed:

```
 0  Mission 10: 10.alm            14  Mission 90: 90.alm       28  Beast.ALM - Beast  Land
 1  Mission 20: 20.alm            15  Mission 91: 91.alm       29  Cross.ALM - Crossroads of Mystery
 2  Mission 30: 30.alm            16  Mission 100: 100.alm     ..
 3  Mission 31: 31.alm            ..                           37  Waters.alm - Waters
13  Mission 81: 81.alm - Panic    27  Mission 151: 151.alm     38  10.alm  (the map-view row)
```

Row 0 is what the picker selects when it opens.

## Divergences and open threads

- **D-1 — the loss is the front end's, not the world's.** `sim.World.Outcome()` still reports
  `OutcomeUndecided` for a mission whose hero is dead, so `-check -mission N`, `almtool` and the
  world's byte form do not carry the rule. `MISSION-END-013` puts it inside the reporter; putting
  it there means a hero id on `sim.World`, which the byte form must round-trip, and no byte-form
  version is allocated to this story. Named in `provenance.md` as the seam a byte-form story
  should close.
- **D-2 — a mission row carries `FromArchive` false.** `mapBytes` is the only reader of that field
  and a mission row never reaches it — the routing arm returns first — so false is exactly what
  `frontend.go`'s own comment says the field now means: which address `mapBytes` would compose.
  The arm is what keeps that true, and SC-4 is what would catch the arm going away.
- **D-3 — the picker now lists 66 rows on a stock install.** Accepted in `plan.md` R-1: the mission
  rows lead, so nothing previously reachable moved out of reach and the row a player lands on is
  mission 10.
- **D-4 — the outcome still latches permanently.** `MISSION-END-013`'s win-then-lose re-report is
  not built; `pkg/sim/script.go` still returns on a decided outcome, and this story's hero test
  sits under the same latch through `m.announced`.
- **D-5 — the team-mates half of the owner's testimony is not built**, because `MissionParty`
  returns a party of one. `Start.IDs[0]` is the hero; the rest of `Start.IDs` is not read.

## What this makes cheaper, and what it does not

`openDifficulty` and `startColumns` were left exactly as they are, as scoped. The routing change
does make the first cheaper: both doors into a mission now spell `openDifficulty` and
`MissionParty(f.StartWeapon, f.Bodies)` at `frontend.go:647` and `:720` and nowhere else, so a
character-generation story replaces two argument lists and inherits the picker for free. It does
nothing at all for `startColumns`, which is a viewport question this story never touches.
