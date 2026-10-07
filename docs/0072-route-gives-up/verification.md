# Verification — the route the hero does not take

Toolchain `go1.26.1 windows/amd64`, matching `go.mod`. Two lawful installs under the preserved
`gameversions/` roots, EN and RU. Every real-install run below sources its root from
`AGAINROM_ASSETS`; nothing was written outside the repository and no converted asset was produced.

## The gate

```
go build ./...                          EXIT=0
go vet ./...                            EXIT=0
gofmt -l $(git ls-files '*.go')         EXIT=0, 0 lines
go test -count=1 -trimpath ./...        EXIT=0
sh scripts/check-no-game-assets.sh      EXIT=0   check-no-game-assets: clean (tree scan)
sh scripts/check-doc-budget.sh          EXIT=0
sh scripts/check-sdd-audit.sh           EXIT=0
```

`-trimpath` is the local workaround for a test binary Windows Defender quarantines; it changes no
behaviour under test.

The deletion set against the branch point is **empty**:

```
git diff --diff-filter=D --name-only origin/master HEAD   -> (no output)
```

## The milestone — AC-7, SC-6

`missionrun` starts the tenth campaign mission and drives it with move orders alone. Both installs,
same build, same arguments:

```
$ AGAINROM_ASSETS=<en root> go run ./cmd/missionrun \
    -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (54,24), Chebyshev 3, after 1250 ticks
waypoint 2  p0 -> (66,16) r3 : reached (63,17), Chebyshev 3, after 2246 ticks
outcome won at tick 3520

$ AGAINROM_ASSETS=<ru root> ... (same arguments)
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (54,24), Chebyshev 3, after 1250 ticks
waypoint 2  p0 -> (66,16) r3 : reached (63,17), Chebyshev 3, after 2246 ticks
outcome won at tick 3520
```

Tick for tick identical on both roots, which is what a claim about this game has to be.

**The differential.** The same tool, the same install, with the near call site put back to the
all-or-nothing rule this story replaces and nothing else changed:

```
waypoint 2  p0 -> (66,16) r3 : STOPPED SHORT of (50,26), Chebyshev 16, after 40000 ticks
outcome undecided at tick 41314
```

So the win is this change's, and the cell it used to stop on is the one the defect was reported at.

The same drive runs inside the suite as `cmd/missionrun.TestTheTenthMissionIsDrivenToAWin`, which
**skips** when no asset root is configured — the suite is green with no game present, and that is the
state `go test ./...` above was run in.

## Acceptance criteria

| id | witness | result |
|---|---|---|
| AC-1 | `pkg/sim.TestAMoverWalksPastABodyStandingOnItsWaypoint`, both sub-cases | PASS |
| AC-2 | `pkg/sim.TestTheNearSearchLooksFurtherThanTheFarOneForASubstitute` | PASS |
| AC-3 | `pkg/sim.TestABoxedInOrderIsHeldFifteenTicksAndGivenUpOnTheSixteenth`, `TestEachTickHasExactlyOneOutcome` | PASS, unedited |
| AC-4 | `pkg/sim.TestACorpseFreesItsCellAndADownedUnitDoesNot`, `TestACorpseFreesItsCellInTheTickItDies` | PASS, re-pinned |
| AC-5 | `pkg/sim.TestTheTickHandsTheNearSearchTheScaledRule` | PASS, re-pinned |
| AC-6 | the whole suite, `EXIT=0` | PASS |
| AC-7 | the two milestone runs above and their differential | PASS |

**AC-3 is witnessed by tests this story did not touch**, which is the strongest form the criterion
could take: the boxed-in mover reaches the limit and gives up under the new rule for the reason the
contract states — it settles for the cell it already stands on, which yields an empty route and is not
an advance.

**AC-4's expectation moved by one cell and its subject did not.** A mover ordered onto a cell a downed
unit holds now walks up to it and stops adjacent, still holding its order, where it used to hold its
start cell; it still never stands on the held cell, and a one-row corridor sealed by a downed unit is
still not passable. `TestACorpseFreesItsCellInTheTickItDies`'s control was strengthened rather than
relaxed: one tick no longer separates a corpse from a body, so the control now runs six ticks and
measures the blocker's own cell.

**AC-5's discriminator changed shape.** The scaled rule no longer makes the mover hold its cell, so
the test now computes both rules' answers from the same world, asserts they differ in their first
step, and asserts the tick took the scaled one. The near call site still cannot be moved to the flat
budget unnoticed, which is the whole reason that pin exists.

**AC-6 — no pinned digest moved.** No file carrying one is in the diff:

```
git diff --name-only origin/master HEAD | grep -E 'hash|digest|preserved|gridform|form'  -> (no output)
```

So nothing had to be re-derived. R-2 is answered by measurement rather than by argument: no fixture in
the corpus puts a body on a near search's waypoint.

## Properties

| id | witness | result |
|---|---|---|
| P-1 | `internal/archtest.TestSimSourcesAreDeterministic`, `TestCheckSimTests`, `TestLiveTreeClean` | PASS |
| P-2 | the byte-form version is unmoved at 11; no field added; `pkg/sim` binary and hash tests unedited and PASS | PASS |
| P-3 | `pkg/sim` route, wall, channel, relation and waypoint suites; the sealed-corridor and boxed-in cases | PASS |

P-2 is checkable from the diff: `pkg/sim/binary.go`, `pkg/sim/world.go`, `pkg/sim/hash.go` and
`pkg/sim/scriptbinary.go` are untouched. The settle rule is a parameter of the search, chosen at the
two call sites in the tick and stored nowhere.

## Success criteria

| id | evidence | result |
|---|---|---|
| SC-1 | AC-1's two sub-cases: arrives when the way round is open, gives up at the cell before the body when it is sealed | met |
| SC-2 | AC-2: settled under the near bound, refused under the far one, on one world with every other parameter held equal | met |
| SC-3 | AC-3 and AC-4 | met |
| SC-4 | `go test -count=1 -trimpath ./...` `EXIT=0`; empty digest diff | met |
| SC-5 | `internal/archtest` all PASS: the import graph accepts `cmd/missionrun`'s row and the determinism scan over `pkg/sim` is clean | met |
| SC-6 | the milestone runs and the differential above | met |

## Limitations, stated rather than implied

**The oscillation risk (R-1) is answered by one map, not by a proof.** A settled step is chosen
cheapest-to-reach-from-the-mover, which makes a cell behind the mover a worse candidate than an equal
cell ahead of it, but nothing here proves a mover cannot cycle. What is measured is a 2246-tick walk
across a campaign map with thirty-five bodies on it, arriving.

**The tool's own drive is not a player's.** It aims at the open cell nearest the mover inside the
radius and re-aims when a walk ends short. A player clicks once. The differential above shows the
defect was not the re-aiming — the pre-change build stops at the same cell twice and the re-aim buys
it nothing.

**Nothing here measures the other twenty-seven campaign maps.** One mission reaching a win is the
milestone; a census of what the other maps now do is not in this story and was not run.

**The wait rule remains unimplemented**, so where the original may have a mover turn and stand still
in front of an adjacent body, this build steps around it. No evidence here bears on which the original
does, because the predicate that decides it is undecoded.
