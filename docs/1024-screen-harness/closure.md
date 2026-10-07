# Closure — 1024, the screen harness

As-built record. Review history belongs to git and `pipeline/LOG.md`, not here.

## Twelve-aspect matrix

| Aspect | Result | Note |
|---|---|---|
| Data | N/A | No data format, archive, or table touched. |
| Runtime state | N/A | No new `ui.App`/`ui.Flow` field. Reads existing state (`townCursor`, `hasTownCursor`, `shopDragArmed`, `shopDragMoved`, `shopDragIcon`) in a new test only. |
| Simulation | N/A | `pkg/sim` untouched. `internal/gatedtests` and `cmd/screencensus` import no sim package. |
| Player input | N/A | No new input path. The drag/hover state exercised in `TestAppComposeTownRoomThreadsLiveCursorAndDragState` is set directly on the fixture, not driven through an input event. |
| AI | N/A | Untouched. |
| UI/HUD | PASS | B1's registry, B2's geometry witnesses (kept structurally distinct from B1's selection witnesses since round 2), and B4's five-column census are UI/HUD's own coverage; see spec.md. |
| Triggers/scripts | N/A | Untouched. |
| Inventory/equipment | N/A | Untouched. The shop drag fixture used for `TestAppComposeTownRoomThreadsLiveCursorAndDragState` reuses `fakeShopTown`'s pre-existing shape; no inventory rule changed. |
| Persistence/save-load | N/A | Untouched. |
| Campaign/session | N/A | Untouched. |
| Shipped content | N/A | No shipped-content sweep applies: this story adds no content-dependent rule. |
| Interactions with existing mechanics | PASS | `screendispatch_test.go`'s dispatch tests exercise the existing `composeScreen`/`composeTownRoom`/`HeadlessFrame` call graph exactly as production reaches it; no interaction was found broken. The one known gap (surface-room hover, below) is disclosed rather than silently passed. |

No in-scope GAP against the twelve-aspect matrix. The chargen-detailed-page mutation site
(`chargen_page.go:742`) is explicitly out of scope this cycle (the 1022 boundary), not an in-scope
gap left open; spec.md and the contract both record it as deferred, not missed.

**One unmet contract bar, disclosed rather than denied (round 3).** Contract B4 states, twice, that
a disagreement between `cmd/screencensus`'s gated-test count and `check-release-tests.sh`'s own
run-time count "is a failure rather than a note". The shipped tool prints a note, for the
module-boundary reason spec.md's B4 section states (a binary in this module must not assume a
sibling checkout path exists). That reason is sound, and both counts agree at 35 on both roots at
this landing (Reconciliation, below); it does not make the contract's literal sentence true of the
shipped binary. This is not a twelve-aspect GAP — no aspect row above depends on the census
enforcing that comparison itself — and it is not silently met either: it is a recorded deviation
from the contract's own stated bar, and a future story could close it by having `cmd/screencensus`
accept the sibling count as a flag or environment input rather than only printing a note.

**Round 2 correction to this line.** The round 2 adversarial review (finding 1) found this
statement false as of the pushed-for-review sha: B4 names five columns and `cmd/screencensus`
printed two, with `screenEntry.Composer` and `screenEntry.Reason` produced but read only by
`screenregistry_test.go`'s own emptiness checks. That was an in-scope GAP the pass-1 closure denied
rather than recorded. `cmd/screencensus` now prints all five (spec.md, B4); the statement above is
re-verified against the current tree, not carried over from pass 1.

**The contract's headline number did not move.** 5 of 9 named screens compose (`ScreenMenu`,
`ScreenChargen`, `ScreenTown`, `ScreenGameMenu`'s shared case, and the chargen/town split inside
`cmd/screenshot`'s 9); `ScreenPicker`, `ScreenLoad`, and the mission screen (`ScreenMap`) still
compose nothing. Round 2 added geometry witnesses and a fuller census; it changed no composer and
closed no `Reason` row.

## Integration witness

`pipeline/check-scenarios.sh` (the seat's own headless-scenario gate, 13 scenarios driving real
campaign missions and a save through production `pkg/game`/`pkg/ui` code) ran clean on both asset
roots, re-run at round 2's end with `AGAINROM_IMPL` explicitly set to this worktree (round 2
mechanics instruction — an unset run silently checks the seat's own `implementation/` checkout,
which happens to print identical figures for this story and cannot discriminate): 13 of 13 on `en`,
13 of 13 on `ru`. This story adds no scenario of its own — B1-B4 are synthetic, package-level tests
reached by `go test ./pkg/ui/...` and `go test ./internal/gatedtests/...`, not scenario-shaped — so
the integration witness for this landing is that the existing scenario suite is unaffected by the
new registry, tests, and tools: nothing in `pkg/ui`'s production code changed the game's own drawn
output (every extracted-literal function this story added, including round 2's
`townSurfaceMessageRect`/`preCreateNameTextOrigin`, returns the same value the inline literal it
replaced already computed, confirmed by the full `go test ./...` pass and by `check-scenarios.sh`'s
unchanged 13/13).

`pipeline/check-release-tests.sh`, the seat's install-gated test-population gate, likewise re-run
with `AGAINROM_IMPL` set to this worktree at round 2's end: 35 of 35 selected tests ran and passed,
0 skipped,
on both `en` and `ru`. This is the "real campaign mission" witness for B3 specifically, since B3's
own subject is the population of install-driven tests, not a screen's pixels.

## Reconciliation

- **Screen count.** `cmd/screencensus`'s screen section is built by running
  `go run ./cmd/screenshot -list` as a subprocess and passing its output through; it printed
  "cmd/screenshot's own 9 known screen(s)" when run at this landing, matching the contract's
  stated 9 by construction (same source, not a second count).
- **Gated-test count.** `internal/gatedtests.Scan` (used by both `cmd/screencensus` and its own
  `TestScanMatchesTheCheckedInPopulationList`) found 35 tests at this landing.
  `pipeline/check-release-tests.sh`, run independently on both `en` and `ru`, measured its own
  population the same way it always has (grepping `--- SKIP` lines with all three `AGAINROM_`
  variables unset) and printed "selected 35 install-gated tests" on both roots. The two counts
  agree: 35 = 35, on both roots. The contract's own baseline ("35 on both roots at `4de5d8b`") is
  unchanged at this landing.
- **Destination-rect mutations.** 4 of the contract's 5 surviving sites are now killed by a new or
  extended test, each proved by a real +1px edit, a failing test, and a byte-identical revert
  (spec.md's table). The 5th is deferred to the 1022 boundary.
- **Geometry witnesses per registered composing screen (round 2, B2; corrected round 3).** Every
  registered composing screen has at least one EFFECTIVE geometry witness: `ScreenMenu`
  (`pkg/render/menu.TestSelectionAndCompose`), `ScreenChargen` pre-create
  (`TestPreCreateNameControlRegionIsPublished`, pass 1; `TestPreCreateNameTextDrawsAtItsOwnOrigin`,
  round 2), `ScreenTown` (four tests, the fourth — `TestTownSurfaceMessageIsDrawnAtItsOwnRect` —
  added round 2), and `ScreenGameMenu` (`ScreenTown`'s own four, shared by construction, plus its
  own `TestTheDecodedRectangles`). `ScreenChargen`'s list still covers the pre-create page only; the
  detailed page is story 1022's territory and is named by `screenEntry.GeometryGapNote`, read by
  `cmd/screencensus`'s "known geometry gaps" section, not by the unwitnessed-remainder section (that
  section's own count is 0 at this landing).
- **Round 3 correction: `ScreenMenu`'s cited geometry test did not witness composition, and
  `ScreenGameMenu`'s remainder was a false gap.** Round 2's closure (this document, previous
  revision) named `pkg/render/menu.TestNewGameButton` as `ScreenMenu`'s geometry witness on the
  strength of a mutation at `menu.go`'s table declaration. Mutated instead at the USE site
  (`pkg/render/menu/state.go:98`, `Overlay`'s own return of `HoverRects[i]`), `TestNewGameButton`
  stays green — it reads the table directly and never calls `Compose`.
  `pkg/render/menu.TestSelectionAndCompose` catches the same mutation at the pixel level (five
  subtests fail); it is now the registered witness. Separately, `composeScreen`'s switch and
  `drawTown`'s own direct call both route `ScreenGameMenu` through the identical
  `a.composeTownScreen()` call `ScreenTown` uses — verified to the leaf, not assumed from the switch
  alone — so `ScreenTown`'s own `GeometryTests` witness `ScreenGameMenu`'s frame too.
  `screenEntry.SharesGeometry`/`SharesGeometryWith` record this and `ScreenCensus` computes the
  union; `cmd/screencensus`'s unwitnessed-remainder count moved from 1 (`ScreenGameMenu`, a false
  positive) to 0. `ScreenGameMenu` also gained its own `GeometryTests` entry,
  `TestTheDecodedRectangles` (pre-existing, `gamemenusurface_test.go`), witnessing the panel/row
  rectangles the shared composer call does not draw. Both mutations (state.go:98, and a prior
  screenregistry.go read confirming the shared composer call) were reverted byte-identical; see
  spec.md's "Round 3" subsection for the full mutation table.
- **`cmd/screencensus`'s two other round 2 hardcoded facts, made computed (round 3).** Section 1's
  header and section 2's header printed the literals "9" and "7"; both are now `%d` against the same
  values their own `selected:` lines already computed. Verified by mutation: adding a 10th row to
  `cmd/screenshot`'s `knownScreens` moved section 1's header and `selected:` line from 9 to 10
  together; reverted byte-identical. The hardcoded three-line chargen-detailed note is now
  `geometryGaps`, printed from `screenEntry.GeometryGapNote` with a computed count ("known geometry
  gaps recorded in the registry: 1").
- **`internal/gatedtests`'s own doc-comment/test-name drift (round 2, finding 3).** `scan.go`'s
  package doc named a test, `TestEveryGatedTestIsNamed`, that did not exist in the module; the real
  test is `TestScanMatchesTheCheckedInPopulationList`. Fixed, and given the same fail-closed guard
  `pkg/ui`'s registry already has: `SelfTestName` (a new exported constant) names the test, and a new
  `TestSelfTestNameIsDeclared` parses this package's own `_test.go` files with `go/ast` and fails if
  no function of that name is declared. Mutation-proved: `SelfTestName` temporarily corrupted to a
  nonexistent name, `TestSelfTestNameIsDeclared` failed with the expected message, reverted
  byte-identical.
- **The silent-return blind spot (round 2, finding 4) is disclosed, not fixed.** `internal/gatedtests`
  and `pipeline/check-release-tests.sh` both define a gated test by a `t.Skip`/`t.Skipf` call naming
  an `AGAINROM_` variable; a test that instead reads such a variable and returns silently is invisible
  to both and reports `ok`. Reproduced directly with a throwaway probe added to and removed from
  `pkg/game`: `internal/gatedtests.Scan`'s count stayed at 35 with a silent return, and moved to 36
  with the same probe rewritten to call `t.Skip`. `scan.go`'s package doc now states this as a
  disclosed limit; no test of this shape is known to exist in the module. `pkg/game/save_safety_test.go:443-446`
  reads an `AGAINROM_` variable and returns silently, but is a subprocess helper rather than an
  install gate and stays correctly excluded from the gated-test population; it is the one example in
  the current module of the shape this disclosure names, without itself being a case that should be
  counted.
- **`HoverCell`'s non-test reference count, corrected (round 2, "correct one count").** spec.md's
  disclosed gap for the surface-room hover branch previously stated "two call sites total". The
  actual count is 7 occurrences across 5 files (spec.md's B2 disclosure lists each). The load-bearing
  half of the original claim — the single read is behind the `artSchool` gate, `v.Press` is read
  separately, and the no-art arm passes a literal `false` — is unchanged and was re-verified to the
  leaf this round.
- **Milestone census (mission 10 / mission 20 UNSUPPORTED trace nodes).** `go build -o mr
  ./cmd/missionrun` then `mr -mission N -trace -ticks 1 | grep -c UNSUPPORTED`, both roots:
  mission 10 = 0, mission 20 = 0, unchanged on both `en` and `ru`. `pipeline/milestone-baseline.txt`
  carries no `cannot run` line for `m10` or `m20` on either root, which is the same reading in that
  instrument's own vocabulary. This story does not touch `pkg/sim`, script execution, or any
  simulation-facing package, so an unchanged census is the expected result, not merely the observed
  one.

## Open items

1. **`chargen_page.go:742`'s destination-rect mutation is deferred**, not fixed. Whichever story
   next touches the chargen detailed page after 1022 lands should extend
   `composeChargenDetailedPage`'s own composition test to close it; that story now has B1's registry
   entry for `ScreenChargen` to extend rather than a symbol to find from scratch.
2. **`composeTownRoom`'s surface-room hover/press branch has no geometry witness.** Investigated
   and found to have no visible effect through the CPU composite with no game art loaded: `HoverCell`
   is read at exactly one production site (`townshell.go:833`, current line; `:824` before round 2's
   `townSurfaceMessageRect` extraction moved it), gated on
   `!statistics && artSchool && v.SchoolClass >= 0 && v.SchoolClass < 2`, and a surface button's
   press/hover appearance differs from its rest state only when real `pic[0]`/`pic[1]` art is
   present. A fixture built with synthetic school-room art (`SchoolArt.Faces`/`SchoolArt.Skills`
   populated) would let this branch be witnessed; it was not built here, scoped out to keep this
   story inside its own three-adversarial-pass ceiling on a Client-only domain.
3. **The process-rule addendum landed only in this worktree's gitignored `SDD/WORKFLOW.md` copy.**
   `AGENTS.md`'s coverage rule 6, reworded round 2 to name `Test` and `GeometryTests` separately, is
   committed and pushed. The matching correction in `SDD/WORKFLOW.md` (the section, renamed "Screen
   geometry tests") is local to this worktree
   (`git check-ignore -v SDD/WORKFLOW.md` confirms `SDD/` is gitignored) and is not part of the
   pushed commit. The canonical `SDD/WORKFLOW.md` outside any lane worktree needs the same sentence
   applied by whoever holds that copy.
4. **`ScreenPicker` and `ScreenLoad` were left as reason-carrying registry entries**, not given a
   CPU composite. Both currently draw only through `ebitenutil.DebugPrintAt`. Giving either an
   image-producing composite is new production behaviour with no owner ask behind it this cycle;
   the registry records the reason rather than the story inventing a composite to fill the row.
5. **`internal/gatedtests`'s scanner resolves one call level.** A test reaching a gated skip through
   two or more levels of same-package helper calls would read as ungated. No such test was found
   while enumerating the current population (35, matching `check-release-tests.sh`'s own count on
   both roots); a future one would be caught only by that reconciliation disagreeing, not by the
   scanner itself. Disclosed in spec.md's B3 section.
6. **The silent-return blind spot is disclosed and unmitigated.** `internal/gatedtests` and
   `pipeline/check-release-tests.sh` are both blind to a test that reads an `AGAINROM_` variable and
   returns silently rather than calling `t.Skip`; the depth-reconciliation in item 5 does not catch
   this shape, because both sides of that reconciliation are blind to it together. No such test is
   known to exist in the module. Building a third instrument that would catch it was out of round 2's
   ordered scope; disclosed in `scan.go`'s package doc and spec.md's B3 section.

## Research reconciliation

None applicable. This story makes no claim about ROM1 behaviour and cites no research claim
(contract, "Research claims": none cited). No divergence row was opened or closed.
