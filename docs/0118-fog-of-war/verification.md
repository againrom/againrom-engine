# Verification — 0118 fog of war

Commits `5a8c29d` (T1), `9b894e9` (T2), `4f00349` (T3), `d753aed` (T4), off
master `67d2e57`, plus one untrailered repair `5856ef1`. Gate run on the clean
committed tree.

## The gate

`go build ./...`, `go vet ./...`, `gofmt -l $(git ls-files '*.go')` (prints
nothing) and `go test -trimpath -count=1 ./...` are all clean with **no game
install visible to the tests**. `scripts/check-no-game-assets.sh` reports
`clean (tree scan)`. `scripts/check-doc-budget.sh` exits 0.
`scripts/check-sdd-audit.sh` reports no FAIL for this story; its note and
warning counts are not comparable from a lane, because a worktree has no
`builds/`.

`git diff --diff-filter=D --name-only 67d2e57..HEAD` is **empty** — this story
deletes nothing. Every commit's trailers were read with
`git log --format='%h %(trailers:key=Co-Authored-By)'`: **none carries one**.
The four trailered commits carry `0118-fog-of-war/T1..T4`, one each, and the
two untrailered commits are the SDD artifacts and the gofmt repair.

33 tests were added. The four executors were each asked to witness their
arithmetic by **reverting the code and watching the test go red**, not by
reading the assertion; each reported the red output, and those reports are
summarised under the criteria below.

## Acceptance

**AC-1 — green with no game present.** `go test -trimpath -count=1 ./...`, all
34 packages `ok`. Every fixture in the 33 new tests is built in test code.

**AC-2 — flat-ground cell counts through the exported reader.**
`TestTheExportedSightMatchesTheFlatGroundDiscThroughAnEntity` measures **9, 21,
45, 69, 105, 145** for ranges 1..6 on a flat synthetic world with the observer
far from every edge — the law's own figures, reproduced rather than adjusted to.

**AC-3 — the two rectangles disagree only near an edge.**
`TestTheTwoRectanglesAgreeAwayFromTheEdgeAndDisagreeInsideIt`. Witnessed by
reverting: setting the fog inset to the AI's 8-cell rectangle turned it red at
exactly the boundary it exists to check — *"an observer 26 cells from the left
edge agrees between the two readers, want a difference"*.

**AC-4 — the seed identity.** `TestTheSeedIdentityHoldsAtEveryShift` checks the
whole square, radius 0..20 by shift 1..8. Witnessed by reverting: seeding
`1<<shift` instead of `1<<(shift-1)` produced `sightSeed(0<<8, 1) = 2, want 1`
and failures across the square.

**AC-5 — nothing entered the hashed world.** `formatVersion` is still **26** at
`pkg/sim/binary.go:403`; the allocated version 28 is **returned unused**. The
three existing pins — the version test, the canonical field-set pin and the
fixed-world digest — all pass untouched. `pkg/sim`'s exported method-set pin
required listing the new `Sight`, which is the pin doing its job.

`TestTheAIReaderStampIsUnchangedByThisStory` pins the AI reader itself: a
sha256 of `groupSight`'s output on a fixed synthetic world (relief ridge, one
hill, an 8-cell border, three members at ranges 6/3/9) was captured from the
**pre-change** single-argument form, and the parameterised form reproduces it
byte for byte — `9e0b5778d752c4c0d87253c13c06088f60c71969e01498f2a34dd5624ba66bb9`,
435 lit cells.

**AC-6 — a fresh mission opens closed.** `TestAMissionOpensClosedAtTickZero`
and `TestOneRefreshLightsSomeCellsAndLeavesOthersUnseen`. Witnessed by
reverting: removing the construction-time refresh gave *"a freshly constructed
mapWorld shows nothing visible at tick 0 (AC-6)"*.

Also witnessed **on real data**, which is the check that matters here. A
throwaway tool (built, run, deleted, never committed) started campaign mission
10 through `game.StartMission` and censused `World.Sight(sim.SelfSlot)` at tick
0. Both roots give the identical answer:

    map 80x80 = 6400 cells; entities 36; owner-1 entities 1, scan ranges map[5:1]
    Sight(SelfSlot) at tick 0: 103 lit, 6297 unseen (98.4% of the map closed)

**FR-3 is witnessed by that same line.** The party's only entity has
`ScanRange` **5**, and 5 is what the hero's own sheet derives — Mind 15 plus
Reaction 26 is 41, `41/25 + 4 = 5`. No constant in this tree names a sight
radius; the value came from the install's data through `Hero.Sight()`. The 103
against a flat-ground 105 is the terrain taking two cells off, which is the
predicate behaving as `AI-LOS-089` says it must.

**AC-7 — explored only grows.**
`TestExploredNeverShrinksAndVacatedCellsStayExplored` moves an entity and
refreshes repeatedly, asserting the explored count is monotonically
non-decreasing and that vacated cells hold `FogExplored`. Witnessed by
reverting: clearing explored on an unlit cell gave *"the vacated cell fell to
unseen, want it to stay FogExplored (AC-7)"* and *"the explored count never
grew past its first refresh's 69 over 5 further refreshes"*.

**AC-8 — the shroud factor and where it multiplies.** `TestFogScale` pins 1,
0.5, 0; `TestFogComposesOntoCornerScales` pins that all four of a cell's corner
scales are multiplied by it. Witnessed by reverting: stripping the multiply
from `cornerScales` gave *"unseen cornerScales = [1.25 1 1.25 1.8125], want
all-0"*.

**AC-9 — the drawable gates.** `TestFogGatesEntitiesByOwnerAndVisibility`,
`TestFogGatesSacksByVisibility`,
`TestFogGatesStaticAndStructurePlacementsByAnythingButUnseen`. Witnessed by
reverting: forcing the entity gate true gave *"a non-local entity on an unseen
cell reached the drawn set"*.

**AC-10 — the minimap composer is pure.**
`TestComposeMinimapIsPureAndTakesNoWindow` calls it as a free function with no
`Viewer`, no window and no engine, twice, and compares the bytes.

**AC-11 — the reveal does not write the plane.**
`TestFogRevealAnswersVisibleWithoutWritingThePlane` asserts the plane's bytes
are identical before and after revealing, so restoring is exact by
construction rather than by a restore path.

**AC-12 — the minimap is inert.** Three tests:
`TestMinimapFunctionsTakeNoCursorPosition`,
`TestCommandFileNamesNoMinimapSymbol` and `TestMinimapConsumesNoClick`. No
click surface was built.

## Success criteria

**SC-1** and **SC-2** — the gate above, including `internal/archtest`, which
scans `pkg/sim`'s sources for float identifiers and literals and for `os`,
`time` and `math/rand`, and holds the import DAG. `pkg/ui` still does not
import `pkg/sim`: the plane crosses as `[]byte` and two ints.

**SC-3** — the AI stamp pin under AC-5.

**SC-4** — the version and field-set pins under AC-5.

**SC-5** — `fogAt` is the plane's only reader. Every consumer goes through it:
the terrain shading via `cornerScales`, the four layer builders' gates, and the
minimap. The gate helpers live in `fog.go` beside it.

**SC-6 — witnessed in mechanism, NOT as a rendered frame.** This is the one
criterion this lane did not fully discharge and it is stated plainly. What was
run: the binary builds and starts mission 10 on both roots
(`-check -mission 10` reports *"mission 10 at scenario/10.alm, 80x80, 36
entities, party at (17, 66), 11 raise(s)"*); the plane census above shows
98.4 % of that map closed at tick 0; `TestFogRevealKeyTogglesOncePerPress` and
`TestMinimapKeyTogglesOncePerPress` drive `App.step` with the input struct and
observe both toggles flip once per press; and the wiring was read end to end —
`mw.view.SetFog(...)` in `push()`, `F4` and `M` bound in `readAppInput` and
consumed in `step`, the minimap drawn in `Draw` after the readout, shown by
default because its field stores hidden inverted. **No windowed frame was
looked at from this seat.** Whether the shroud, the minimap panel and the
culled units read correctly to a human eye is unverified.

## What was approximated, and against what

- **One fog value per cell, not four** (D-1). The original projects fog
  per-vertex and rebuilds it every frame; `TERR-FOG-083` states outright that a
  per-cell consumer cannot reproduce any frontier cell. Ours cannot. The
  frontier is a stair.
- **Whole-cell sight** (D-2). `AI-SIGHT-094` gives a 40-cell gap for a hero
  between the fog's `u16` seed and the simulation's byte. Our entity carries
  whole cells, so our fog and our AI agree exactly — the original's own
  behaviour for every monster and human, and different only for a hero. The
  seed already takes 1/256 units, so the seam is one entity field wide.
- **Structures and statics on explored ground** (D-5), against `TERR-FOG-086`,
  which hides every drawable whose cell is not currently visible.
- **The plane does not survive a save** (D-6): it is not in the byte form.
- **The minimap is authored** (D-4). `AI-MINIMAP-062`'s own confidence cell
  says the second window's identity as a minimap is inferred from art names and
  not established, so nothing here claims to reproduce one.
- The corpus figures quoted from `AI-LOS-089`, `TERR-FOG-118` and
  `AI-SIGHT-094` are graded **Medium** by research and were used as
  expectations, never asserted in a test.

## What one executor's report got wrong

T2 reported `gofmt -l` clean; it was not — `pkg/ui/fog_test.go` carried two
comment-alignment defects. Caught by re-running the gate in this seat rather
than taking the report. Repaired in `5856ef1`, separately, so the
one-task-one-commit mapping stays exact.
