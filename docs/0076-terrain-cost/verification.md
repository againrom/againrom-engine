# Verification — the ground carries a cost, and the mover pays it

Worktree `wt-0076`, branch `impl/0076-terrain-cost`. Research submodule at `20921e2`,
`git submodule status` showing no leading character. Go toolchain as pinned in `go.mod`. Both
installs preserved read-only under `gameversions/{en,ru}`.

Written from master `71924d6` and **rebased onto `ba2d560`** before pushing, which brought 0077 and
0078 in underneath it. Every figure below was **re-measured after the rebase**, against a base
worktree at `ba2d560` rather than against the numbers taken at `71924d6`, and every one of them
reproduced: the two stories underneath moved neither mission 10 nor any block plane. The rebase cost
two edits, both stale literals — `pkg/sim/pursuit_test.go` and `pkg/mapload/hero_test.go` each
asserted the form version was 12 to say *their* story bumped nothing, and a later story bumping it
makes that literal false while the claim it was making stays true.

## The change

```
$ git diff --stat ba2d560 HEAD | tail -3
 pkg/sim/step.go     |  28 ++--
 pkg/sim/world.go    | 160 +++++++++++++++++++-
 36 files changed, 2820 insertions(+), 316 deletions(-)

$ git diff --diff-filter=D --name-only ba2d560 HEAD
                                      (empty — nothing was deleted)

$ git log --format='%h %s / %(trailers:key=SDD-Task,valueonly)' ba2d560..HEAD
f6b33b5 0076: verification /
4d88432 0076: every world is built over all three planes / 0076-terrain-cost/T5
f98a105 0076: a map describes a cost plane and a height plane / 0076-terrain-cost/T4
2beadcf 0076: the rate is composed from the two cells the transit joins / 0076-terrain-cost/T3
64052b6 0076: the search charges a ground mover the ground it enters / 0076-terrain-cost/T2
1a0a980 0076 revision: cost changes labels, not reach … /
d7a711e 0076: a world carries a cost plane and a height plane / 0076-terrain-cost/T1
b095950 0076 revision: the wave does not lengthen for a cheaper route … /
aabc801 0076: analysis, provenance, spec, plan and tasks … /
```

Five trailered commits for five tasks, each ID once; the other four are artifact stages and carry
none. (This file's own commit is the ninth and is not listed above by it.)

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')
EXIT=0                                (no output from any of the three)

$ go test -count=1 -trimpath ./...
EXIT=0                                33 packages: 30 ok, 3 with no test files, 0 FAIL

$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)                                       EXIT=0

$ bash scripts/check-doc-budget.sh                                            EXIT=0
docs/0076-terrain-cost/analysis.md               5503 /   7168 bytes  ok (76%)
docs/0076-terrain-cost/provenance.md            14103 /  16384 bytes  ok (86%)
docs/0076-terrain-cost/spec.md                  13305 /  13312 bytes  ok (99%)
docs/0076-terrain-cost/plan.md                  13109 /  13312 bytes  ok (98%)
0076-terrain-cost: plan <= 1.2 x spec           13109 <=  15966 bytes  ok
0076-terrain-cost: tasks <= 1.2 x plan           6057 <=  15730 bytes  ok

$ bash scripts/check-sdd-audit.sh                                             EXIT=0
check-sdd-audit: 297 trailered commit(s) checked, 0076-terrain-cost 5/5 landed
```

## The version bump

**Required, and it is 13.** Both planes are read while a world is advanced and both enter the
digest, so a version-12 form accepted here would decode to a world whose units route differently and
cross at different speeds from the world the bytes were cut from. It is structural besides: the
entity records move by twice the cell count, so a version-12 buffer read against these offsets is a
misparse of every record and not a wrong plane.

**One declared cell count serves all three planes.** No header field was added, nothing before the
records moved, and the entity record did not change by a byte — the pinned offset table needed its
version byte changed and nothing else.

## The plane's production rule: published, not AMBER

Fully published and used verbatim: `TERR-PASS-050` for the classifier — the water early-out, both
reject arms, the tile-word split, the strip-group pairs and the 4×14 blend-level table — and
`ALM-TERR-043` for the ten class scalars, with `TERR-COST-052` settling that the shipped file's
values win over the executable's compiled defaults (they differ at Flowers and Savanna and reach
128 694 of 880 704 cells). **No research question was opened and none is owed.**

One arm is **ours and disclosed**: strip groups 13–15 name no terrain pair, and the original reads
uninitialised heap for them. This build rejects them. No shipped cell of either root reaches it.

## The corpus, against the ledger's own figures

The cost derivation was written from the contract table and never from these numbers, then measured:

```
$ probe -planes -assets <en>
COST HISTOGRAM over 880704 cell(s):
  cost   6 :    49737 (5.65%)      cost  12 :    32798 (3.72%)
  cost   7 :    39969 (4.54%)      cost  13 :    15395 (1.75%)
  cost   8 :   518835 (58.91%)     cost  14 :    42870 (4.87%)
  cost   9 :    18542 (2.11%)      cost  15 :    16297 (1.85%)
  cost  10 :    28871 (3.28%)      cost  16 :    97703 (11.09%)
  cost  11 :    19687 (2.24%)
TOTAL maps=38 cells=880704
```

`TERR-COST-052` publishes *"the cost byte takes only 6..16 over 880 704 cells (58.9 % are exactly
8)"*. Both the range and the share reproduce, and the map and cell counts reproduce independently.
The RU root gives 34 maps / 677 696 cells at 58.50 % — its own corpus, and `AI-ROOT-049`'s counts.
**Value 255 occurs 0 times**, so no shipped cell reaches a reject arm and that arm is witnessed only
by the suite.

## Mission 10 — the measurement asked for, both ways

```
$ missionrun -assets <root> -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3
```

Before is a base worktree at `ba2d560`, after is this branch, both against the same installs.

| | EN before | EN after | RU before | RU after |
|---|---|---|---|---|
| waypoint 1 reached after | 1250 | **1293** | 1250 | **1293** |
| waypoint 2 reached after | 2246 | **2198** | 2246 | **2198** |
| **won at tick** | **3520** | **3504** | **3520** | **3504** |

**It moved, and the two legs moved in opposite directions** — one crosses dearer or rising ground and
takes longer, the other cheaper or falling ground and takes less. Both units end on the same cells as
before (`(54,24)` and `(63,17)`), so what changed is the crossing and not the destination. The two
roots agree exactly, as they did before.

Measured after T1–T3 as well, with the planes carried and read but the loaders not yet wired: **3520,
1250, 2246 on both roots** — unmoved, which is AC-3's end-to-end half.

## Acceptance criteria

| AC | Evidence | Result |
|---|---|---|
| AC-1 | `pkg/sim/planeform_test.go` — the form of a world naming neither plane carries an all-8 cost plane and an all-zero height plane of the block plane's length, and that world equals one naming both | pass |
| AC-1a | same file — a plane one byte short and one byte long, refused at construction and on decode | pass |
| AC-2 | `pkg/sim/costsearch_test.go` — two equal-length corridors, the cost plane MIRRORED between runs; the ground mover follows the cheap one whichever it is, so no tie-break can be what chose | pass |
| AC-2a | same file — one world, a uniform plane and a varied one: identical touched sets, differing labels | pass |
| AC-3 | mission 10 unmoved at 3520/1250/2246 after T1–T3; and every re-taken digest cross-checked backwards (below) | pass |
| AC-4 | `pkg/sim/costsearch_test.go` — a ghost and a flyer take the same route over both mirrored planes, and it is the uniform-plane route | pass |
| AC-5 | `pkg/sim/route_test.go` — both arms over ten (domain, cost) rows and all eight deltas, including 1→1/1, 0→0/0, 255→255/382 | pass |
| AC-6 | `pkg/sim/costsearch_test.go` — one row of five distinct costs; the canonical plane reads 0,2,5,9,14 and the optimised 14,12,9,5,0, which is the entered cell in each | pass |
| AC-6a | same file — an all-zero cost plane returns no route instead of cycling | pass |
| AC-7 | `pkg/sim/costrate_test.go` — transits over cost 6, 8 and 16 strictly increasing, and the 8 equal to a world naming no plane | pass |
| AC-8 | same file — uphill slower than downhill, and a difference of 99 equal to one of 32 in both directions | pass |
| AC-8a | same file — a transit onto the map from outside it gives one number over cost planes of 6, 16 and none | pass |
| AC-9 | `pkg/mapload/costplane_test.go` — the seven words answer 8, 16, 13, 9, 255, 255, 255 | pass |
| AC-10 | same file — short altitude and tile planes, and a long one, all at the extent's length | pass |
| AC-11 | probe run over both roots before and after: `diff` empty, 38 EN and 34 RU maps, block digest and all five arm counts identical | pass |
| AC-11a | `pkg/mapload/costplane_test.go` — **all 65 536 tile words**, the pre-story mountain predicate transcribed into the test against the new class: 0 disagreements, and non-vacuous | pass |
| AC-12 | `pkg/sim/binary_test.go` — every version byte but 13 refused; the pinned form round-trips to an equal world and digest | pass |
| AC-13 | `pkg/sim/planeform_test.go` — one byte at each end of each plane moves the digest | pass |
| AC-14 | `pkg/mapload/costwiring_test.go` — six construction paths × two planes, each carried; plus the positive half on the one path whose entity set is reproducible | pass |
| AC-15 | the table above | pass |
| AC-16 | probe route sweep: **9 of 38 EN maps and 8 of 34 RU** route a ground mover differently over the derived plane than over a uniform one | pass |

### Success criteria

| SC | Discharged by | Result |
|---|---|---|
| SC-1 | AC-1 and AC-1a | met |
| SC-2 | AC-12 and AC-13 | met |
| SC-3 | AC-2, AC-2a and AC-4 | met |
| SC-4 | AC-5, AC-6 and AC-6a | met |
| SC-5 | AC-7, AC-8 and AC-8a | met |
| SC-6 | AC-9 and AC-10 | met |
| SC-7 | AC-11 and AC-11a — the corpus diff empty over both roots, and the exhaustive sweep at 0 of 65 536 disagreeing | met |
| SC-8 | AC-3, AC-14, AC-15 and AC-16 — the no-plane world unmoved, every path carrying both planes, the mission tick reported, and real routes moving on shipped maps | met |

### The backward derivations

The check a new pin and a new transcription cannot make between them, since both move together:

```
strippedOfThePlanes(form, cells)   cut the two sections, restore version 12
  pkg/sim   pinned world   -> 0x140dc9a829f7e3dd   (the pre-story literal)  ok
  pkg/sim   routed world   -> 0xc3ecca188ad60e2d   (the pre-story literal)  ok
  pkg/mapload gridform     -> 0x6ef0e7ef03f32172   (the pre-story literal)  ok
```

Every older derivation now runs on its output — the owner word, the group word, the eight combat
fields — each stripping one story at a time, newest first, and every one of them still reproduces its
own literal. **So the two planes are the only bytes this story wrote into the form.**

### FR-1a's premise, read rather than assumed

All 27 uses of the label plane across `route.go` and `optimised.go` were read and classified. Every
one is a write, a comparison against the unlabelled marker `0`, the ±1 packing, or one label against
another. The only term ever *added* to a label is the step cost, which scales with the plane; and no
label outlives the search that wrote it, since the plane is reset at the head of each. The three
shapes that would break the scaling argument are absent. Its visible consequence: the whole suite
came through T2 with every route, outcome, digest and mission tick unchanged and **only label numbers
moved, every one by exactly four.**

## What the review gates changed

The **peer-prediction read** (spec alone, zero context) reconstructed the build correctly and found
five underspecifications, all folded in before the gate closed: the water arm's reject, that the
blended cost is not the named class's scalar, a plane of the wrong length, FR-1a's invariance
premise, and two criteria that could not fail.

The **adversarial plan read** found two defects that would have shipped. A cost byte of zero makes
every candidate tie in the route extraction and a three-cell region cycles for ever — a hang, now
bounded by the cell count (ours; the decoded 1000-step discard is not reproduced, and the spec says
so). And an off-map endpoint would have halved the rate's cost mean, breaking FR-1a in exactly that
corner. Four smaller ones with them.

**Twice the contract was wrong about this search, in opposite directions.** The first revision said
cost steers the route more than it does; the second said less. What settled it was writing the
criterion down and running it: AC-2a's first fixture failed against correct code, and the true
invariant turned out to be narrower and provable — the wave advances one ring per generation whatever
the ground costs, so the labelled *set* is a property of the map and the order alone.

## Limitations

- **AC-3, AC-11, AC-15 and AC-16 are measured against a lawful install**, not in the suite. Their
  figures ship here; the assets do not.
- **The class scalars are compiled in.** A map customising `data/map.reg`'s `Terrain` section is not
  honoured. Building that reader changes no output today, since the values would be the same.
- **The decoded 1000-step route-walk discard is not implemented.** This build's bound is the cell
  count, chosen because it is provably unreachable while the walk's costs are positive.
- **Strip groups 13–15** get a defined answer here and an undefined one in the original.
- The **separate-context test author** gate was not run: the tests here were written by the same
  context that wrote the code, against the spec. The two independent gates that were run are the ones
  above.

## One thing worth knowing beyond this story

A Go package under `builds/` **breaks `internal/archtest`'s DAG test**, which walks the live tree
rather than the tracked one — `builds/` being gitignored does not exempt it. This story's corpus
probe hit it twice: once as `builds/0076-terrain-cost/probe/`, and again when its source was copied
back beside the binaries for the owner. It ships as `main.go.txt`, and the convention that
`builds/<story>/` holds *binaries* rather than sources now has a mechanism behind it.
