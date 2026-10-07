# Verification — 0038, dim the non-playable map border

Run on the story's own branch, submodule pinned at research `f35be34`, over commits `be5b494`
(Stage 1–3), `e5eecb9` (T1), `0ea7f3e` (T2), `a6165b6` (T3). No game install was present for any run
below; every fixture is a byte literal or a synthetic stream built in test code.

## SC-1 — the gate

```text
$ go build ./... && go vet ./...
(clean)

$ go list ./... | wc -l
28

$ go test -trimpath -count=1 ./...
25 packages ok, 3 with no test files, 0 FAIL

$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)

$ sh scripts/check-doc-budget.sh docs/0038-dim-map-border
docs/0038-dim-map-border/analysis.md               5279 /   7168 bytes  ok (73%)
docs/0038-dim-map-border/provenance.md             5405 /   7168 bytes  ok (75%)
docs/0038-dim-map-border/spec.md                   9736 /  13312 bytes  ok (73%)
docs/0038-dim-map-border/plan.md                   8055 /  13312 bytes  ok (60%)
docs/0038-dim-map-border/tasks.md T1                880 /   1400 bytes  ok (62%)
docs/0038-dim-map-border/tasks.md T2               1063 /   1400 bytes  ok (75%)
docs/0038-dim-map-border/tasks.md T3               1146 /   1400 bytes  ok (81%)
docs/0038-dim-map-border/tasks.md (legend+trace)    773 /   1200 bytes  ok (64%)
0038-dim-map-border: plan <= 1.2 x spec            8055 <=  11683 bytes  ok
0038-dim-map-border: tasks <= 1.2 x plan           3862 <=   9666 bytes  ok

$ gofmt -l $(git ls-files '*.go')
(nothing)
```

`-trimpath` is not decoration here: without it one test binary is quarantined on this machine and the
suite reports a failure that is not the tree's.

## SC-2 — mutants

Six, one per design decision a passing suite could otherwise be blind to. Each was applied to a line
the entry that owns it actually wrote, run, and reverted there; the tree was confirmed clean after the
battery. Every one was killed.

| # | Decision | Mutation | Killed by |
|---|---|---|---|
| M1 | DD-6 | the predicate masks bit 0 instead of bit 1 | `TestBorderCellIsBitOne`, `TestBorderCellReadsRowMajor`, and downstream `TestLoadedViewerCarriesTheMargin`, `TestViewerMarginIsTheSimulationMargin` |
| M2 | DD-1 | the short-plane guard deleted from the predicate | `TestBorderCellIsTotal` — index out of range, which is the point: the guard is what makes an absent layer a non-event |
| M3 | DD-4 | the dim assigns the constant instead of multiplying by it | `TestCornerScalesDimsLitMarginAndComposesWithTheLight`, `TestCornerScalesPlaceholderInTheMarginIsUndimmed` |
| M4 | DD-5 | the dim applied to the placeholder arm as well | `TestCornerScalesPlaceholderInTheMarginIsUndimmed` |
| M5 | DD-2 | the load path wired but delivering nothing (`Passability(nil)`) | `TestLoadedViewerCarriesTheMargin`, `TestLoadedNarrowMapIsMarginThroughout`, `TestViewerMarginIsTheSimulationMargin` |
| M6 | DD-7 | the census reads the plane's bytes its own way instead of asking the predicate | `TestLoadedViewerCarriesTheMargin`, `TestViewerMarginIsTheSimulationMargin` |

**M6 survived its first run, and that is recorded rather than smoothed over.** The census mutant read
the plane for *any* block bit, and on the original fixture — an all-zero tile grid — every blocked cell
was a margin cell, so the two readings could not disagree and the mutant was equivalent. The fixture
was strengthened rather than the claim weakened: `marginMap` now carries one interior **water** cell,
which the derivation blocks to a ground mover and not to an air one, and the test asserts that the map
has strictly more ground-blocked cells than margin cells so the discriminator cannot silently vanish
again. M5 likewise first died at compile time only — an unused import — which is a compiler catch and
not a kill, so it was reshaped into a mutant that builds.

## SC-3 — nothing that held before this story stopped holding

No pre-existing test file was opened. The story's whole diff is five new documents, four new files and
three modified production files:

```text
$ git diff --stat be5b494^..HEAD -- pkg cmd internal
 pkg/game/border_test.go           | 160 ++++++++++++
 pkg/game/mapload.go               |  15 ++
 pkg/render/terrain/border.go      |  51 +++++
 pkg/render/terrain/border_test.go | 147 ++++++++++++
 pkg/render/terrain/composite.go   |  17 ++
 pkg/ui/border_test.go             | 258 +++++++++++++++++++
 pkg/ui/viewer.go                  |  78 ++++++-
```

So the corner-scale suite, the placeholder rule and the front-end summary composition are asserted by
the same code that asserted them before, unedited, and they pass. The mechanism that makes this
possible is the layer's absence being answerable: every existing fixture builds its grid without a
block plane, the predicate answers false, and the picture is the one it always was. That is also what
AC-7 checks directly.

## SC-4 — the grant that shaped the design is unchanged

```text
$ go list -f '{{join .Imports "\n"}}' ./pkg/ui | grep againrom
againrom/pkg/render/camera
againrom/pkg/render/frame
againrom/pkg/render/menu
againrom/pkg/render/terrain
```

The drawing tier reaches the render packages and nothing else — no new edge, and in particular none to
the tier that owns the depth or to the simulation. The import-graph check and the test that pins that
tier's allow-map to exactly its two prefixes both pass inside the suite above.

## SC-5 — nothing near the wall moved

The diff in SC-3 is the whole of it: no file under `pkg/sim`, no decoded format, no digest and no byte
form is touched, and no float, clock or generator is added anywhere near them.

## Criteria

| # | Witness | Result |
|---|---|---|
| AC-1 | `TestBorderCellIsBitOne`, `TestBorderCellWholeMapMargin` | Exactly the bit-1 cells answer true over a 20x19 ring; the interior cell carrying bit 0 alone answers false; a 12x30 plane whose rings meet answers true on all 360 cells |
| AC-2 | `TestBorderCellIsTotal` | Eleven rows — absent, empty and short planes, negative and past-the-edge cells on both axes, a zero width, a negative height, the zero grid — all false, no panic |
| AC-3 | `TestCornerScalesDimsLitMarginAndComposesWithTheLight` | Each corner equals its own undimmed value x `0.5`, strictly smaller and strictly above 0; the test refuses to pass unless some margin cell still has four unequal corners after dimming |
| AC-4 | `TestCornerScalesLeavesPlayableCellsAlone` | Every cell outside the margin answers its pre-story value on the same viewer whose margin is being dimmed |
| AC-5 | `TestCornerScalesDimsTheUnlitMargin` | Both ways of turning the light off — no usable altitude layer, and the unshaded diagnostic — give `0.5` on all four corners of a margin cell and `1` outside |
| AC-6 | `TestCornerScalesPlaceholderInTheMarginIsUndimmed` | The placeholder answers all-1 inside the margin; its margin neighbour answers `ShadeScale(46) x 0.5`, so the dim is demonstrably running |
| AC-7 | `TestCornerScalesWithoutAMarginPlaneIsThePreStoryPicture` | Both a nil plane and an all-clear plane give the pre-story value on every cell of the relief fixture |
| AC-8 | `TestLoadedViewerCarriesTheMargin`, `TestViewerMarginIsTheSimulationMargin` | 20x20 → 384, 24x19 → 432, 19x24 → 432, each the closed form for a ring of the depth the test recovers from the derivation itself; and equal to the simulation plane's own margin arm over the same decoded map |
| AC-9 | `TestLoadedNarrowMapIsMarginThroughout` | 16x40, 40x16, 3x3 and 1x1 are margin on every cell |
| P-1 | the whole `pkg/ui` and `pkg/render/terrain` suite above | No test opens a window or a graphics context; the predicate and the corner-scale source are called as plain functions, and AC-7's two rows show the dim reading nothing but the cell position and the plane |
| P-2 | SC-5 | No simulation, determinism, digest or format surface in the diff |
| P-3 | `TestLoadedViewerCarriesTheMargin` via `derivedDepth`, and M1 | The depth is written once. AC-8 recovers it from the derivation rather than restating it, so no test here carries a copy either; a search of the tree finds the number in one constant |

AC-8's three shapes include an oblong and its transpose on purpose: a derivation that swapped width and
height, or ringed two sides only, answers a different number for at least one of them.

## Not run, and what that leaves open

- **Whether it looks right.** Nothing here shows a picture. That a margin at `0.5` reads as *dimmed
  rather than broken*, that the hard boundary reads as a deliberate line rather than a seam artefact,
  and whether `0.5` is the right amount, are all questions only the owner with a window can answer, on
  a real install. This is the one criterion of the story with no automated witness and it is not
  claimed to have one.
- **No corpus run.** Nothing was measured over shipped maps, because nothing in this story needed it:
  the margin's depth and its shipped count were measured by the research being cited, and re-measuring
  them from a viewer would be checking our own derivation against itself.
- **The cross-cell brightness ordering is not claimed and not tested.** It is false on this renderer's
  scale range, which is why the contract asserts a per-cell property instead — see `analysis.md`.
- **The other margin bits.** The game's stamp sets bits 0 to 4; our plane sets 0 and 1. Bits 3 and 4
  are the cited claim's own Unknown, so there is nothing to verify against yet.
- **The audit's standing notes.** The full unscoped sweep at push time reported an **empty FAIL set**
  — the single FAIL seen earlier was this story's own missing `verification.md`, and it cleared when
  this file landed. The sweep also prints advisory notes for stories predating the witness rule; those
  are not enforced, are not this story's, and no id from them is repeated here, since quoting one
  would make it a claim of this file.
