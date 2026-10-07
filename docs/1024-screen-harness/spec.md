# Spec — 1024, the screen harness

Canonical at landing. Describes what this tree does now, not the original intention.

## Touched domain

**Client** (`docs/DOMAINS.md`, domain 8) only. `pkg/ui` and two new `cmd/`/`internal/` development
tools. No sim, no persistence, no campaign state, no shipped content.

## B1 — the screen registry fails closed

`pkg/ui/screenregistry.go` defines `screenRegistry`, a `map[Screen]screenEntry` naming, for every
`ui.Screen` value declared in `flow.go`, either the composer `composeScreen` selects for it and the
synthetic test that witnesses that selection, or a `Reason` it has neither.

`pkg/ui/screenregistry_test.go` parses `flow.go`'s own `Screen` const block with `go/parser` and
`go/ast` and requires every declared identifier to resolve through `screenConstName` to a
`screenRegistry` entry (`TestEveryScreenValueIsRegistered`), and requires every entry naming a
`Test` to name a function that actually exists in the package's test files, again by parsing source
rather than by a maintained list (`TestEveryRegisteredSyntheticTestExists`). An unregistered `Screen`
value or a `Test` name with no matching function fails these tests; nothing needs a game install to
run them.

Coverage, as shipped:

| Screen | Composer | Test |
|---|---|---|
| `ScreenMenu` | `menu.Assets.Compose`, selected by `(*App).composeScreen` | `TestComposeScreenSelectsMenuComposer` |
| `ScreenChargen` | `(*App).composeChargenScreen` -> `composeChargenPage` (pre-create) / `composeChargenDetailedPage` | `TestComposeScreenSelectsChargenComposer` |
| `ScreenTown` | `(*App).composeTownScreen` -> `(*App).composeTownRoom` -> `composeTownRoom` -> `ComposeTownSurface` / `ComposeShopScreen` / `ComposeTownSquare` / `ComposeWorldMap` | `TestComposeScreenSelectsTownComposer` |
| `ScreenGameMenu` | shares `ScreenTown`'s case in `composeScreen` | `TestComposeScreenSelectsTownComposer` |
| `ScreenPicker` | none (reason: `ebitenutil.DebugPrintAt` only) | — |
| `ScreenLoad` | none (reason: `ebitenutil.DebugPrintAt` only) | — |
| `ScreenMap` | none (reason: `Viewer.Draw` paints the ebiten canvas directly; out of scope) | — |

Fail-closed behaviour verified by mutation: removing `ScreenMap`'s entry made
`TestEveryScreenValueIsRegistered` fail with `Screen ScreenMap (map) has no screenRegistry entry`;
the entry was restored and the file diffed byte-identical against its pre-mutation state.

`ScreenPicker` and `ScreenLoad` were left as reason-carrying entries rather than given a CPU
composite. Both draw only through `ebitenutil.DebugPrintAt` (debug text, no image); giving either a
composite path would be new production behaviour with no corresponding owner ask, so it is left out
of this story's scope and the registry says why. This is a lane decision within the contract's own
"the lane's call, stated in spec.md either way" (contract, "The measured starting set").

## B2 — geometry witnesses, kept distinct from selection

`screenEntry` (`screenregistry.go`) names two different kinds of synthetic test, and they answer
different questions (round 2 adversarial review, finding 2). `Test` proves `composeScreen`'s switch
reaches this screen's composer: it compares `composeScreen`'s own output byte-for-byte against a
second call to the same composer, so it cannot fail when that composer's own destination rectangles
move, only when the switch stops reaching it at all. `GeometryTests` names every synthetic test that
pins one of a screen's own destination rectangles or text origins against a hand-transcribed literal,
independent of the constant or formula it tests, proved by a real +1px mutation and a byte-identical
revert — this is what satisfies this contract's own bar (B2, "shifting the screen's destination
rectangle by one pixel must fail at least one test").

Coverage, as shipped:

| Screen | `Test` (selection) | `GeometryTests` (destination rectangles) |
|---|---|---|
| `ScreenMenu` | `TestComposeScreenSelectsMenuComposer` | `pkg/render/menu.TestSelectionAndCompose` (pre-existing; round 3 correction, below) |
| `ScreenChargen` (pre-create only) | `TestComposeScreenSelectsChargenComposer` | `TestPreCreateNameControlRegionIsPublished` (pass 1); `TestPreCreateNameTextDrawsAtItsOwnOrigin` (round 2, new) |
| `ScreenTown` | `TestComposeScreenSelectsTownComposer` | `TestSchoolSkillRectsMatchTheMeasuredLiterals`, `TestTownButtonTextRectsSplitTheWellIntoLabelThenValue`, `TestDrawNinePatchBorderTilesFourDistinctEdges` (pass 1); `TestTownSurfaceMessageIsDrawnAtItsOwnRect` (round 2, new) |
| `ScreenGameMenu` | `TestComposeScreenSelectsTownComposer` | `ScreenTown`'s own four (shared, `SharesGeometryWith`, round 3); `TestTheDecodedRectangles` (its own, round 3) |

Every registered composing screen now has at least one EFFECTIVE `GeometryTests` entry
(`ScreenCensusEntry.EffectiveGeometryTests`, `pkg/ui/screenregistry.go`) — own plus, where
`SharesGeometry` is set, the screen it shares a composer call with. The one disclosed gap is inside
`ScreenChargen`: the list above covers the pre-create page only. The chargen **detailed** page's own
destination rectangles (`composeChargenDetailedPage`) carry no geometry witness in this registry —
that page's geometry is story 1022's territory this cycle (contract, "A boundary with another open
lane"; do not touch `chargen_page.go`'s detailed-page code this round, per the coordinator's round 2
brief). This is recorded as registry DATA (`screenEntry.GeometryGapNote`, `ScreenChargen`'s own
entry), not a sentence baked into `cmd/screencensus`; that binary's "unwitnessed remainder" and
"known geometry gaps" sections both print from this data (B4, below).

### Round 2 — three geometry witnesses added

Two are the round 2 review's own acceptance bar; the third is the lane's own pick for the one
registered, composing screen (`ScreenMenu`) that previously had a `Test` but no `GeometryTests`.

- **`townSurfaceMessageRect()`** (`pkg/ui/townshell.go`, extracted from an inline literal in
  `ComposeTownSurface`) returns `image.Rect(12, 448, 468, 478)`. `TestTownSurfaceMessageIsDrawnAtItsOwnRect`
  (`townshell_test.go`) pins the function's return value against that literal and confirms
  `ComposeTownSurface`'s real render places a `townShellText`-coloured pixel inside it for a
  non-empty `Message`. A +1px shift of the rect's `Min.X` (`12` -> `13`) was applied and reverted;
  the test failed under the mutation and passed again after the byte-identical revert.
- **`preCreateNameTextOrigin()`** (`pkg/ui/chargen_page.go`, extracted from an inline literal in
  `composeChargenPage`) returns `image.Pt(452, 26)`. `TestPreCreateNameTextDrawsAtItsOwnOrigin`
  (`chargen_page_test.go`) pins the function's return value against that literal and confirms a
  non-empty name (`(*Chargen).EditName`) is drawn with a non-transparent pixel at that origin. A
  +1px shift (`452` -> `453`) was applied and reverted the same way. This is the pre-create page
  only; `composeChargenDetailedPage` was not touched.
- **`pkg/render/menu.TestNewGameButton`** (`menu_test.go`, pre-existing, not written this story)
  pins `menu.HoverRects[0]`/`PressedRects[0]` against hand literals, and was cited in round 2 as this
  screen's geometry witness on the strength of a mutation at `menu.go`'s own table declaration.

### Round 3 — the registry's geometry citation for `ScreenMenu` was wrong, and one screen's remainder was misreported

Two corrections, both found by mutating at the USE site rather than the declaration site
(`AGENTS.md`'s reachability-claim rule: read to the leaf).

- **`TestNewGameButton` does not witness composition.** Mutated at `Overlay`'s own return statement
  (`pkg/render/menu/state.go:98`, `return a.Hover[i], HoverRects[i], ...` shifted +1px), it stays
  green: it reads `HoverRects`/`PressedRects` directly and never calls `Compose`. The mutation that
  does move a rectangle `Compose` actually reads is invisible to it. `pkg/render/menu.TestSelectionAndCompose`
  (`state_test.go`), pixel-level, catches the same mutation: five of its subtests fail
  (`hover_composites_at_the_hover_rect`, `the_overlay_is_copied,_not_blended`,
  `never_two_overlays_at_once`, `compose_returns_a_fresh_image`, `AC-10_sequence`). The registry's
  `ScreenMenu.GeometryTests` now names `TestSelectionAndCompose`; `TestNewGameButton` is not a
  registered geometry witness. Both mutations were reverted byte-identical.
- **`ScreenGameMenu`'s remainder was a false gap.** `composeScreen`'s switch has one arm for
  `ScreenTown` and `ScreenGameMenu` (`case ScreenTown, ScreenGameMenu: return a.composeTownScreen()`),
  and `drawTown` (`app.go`) calls `a.composeTownScreen()` directly for both `Draw` arms too — the
  same production call, verified to the leaf rather than assumed from the switch alone. `ScreenTown`'s
  own four `GeometryTests` therefore witness `ScreenGameMenu`'s frame as well.
  `screenEntry.SharesGeometry`/`SharesGeometryWith` (new fields) record this, and `ScreenCensus`
  computes the union at read time — `cmd/screencensus`'s remainder section previously printed
  `ScreenGameMenu` as having no geometry witness at all; it now prints zero screens with no effective
  witness. `ScreenGameMenu` additionally carries its own `GeometryTests` entry,
  `TestTheDecodedRectangles` (`gamemenusurface_test.go`, pre-existing): it pins `gameMenuPanelRect`
  and `gameMenuRowRect` against hand literals for both the mission and town surfaces, which
  `composeScreen`'s shared call does not draw (the panel and dim paint through ebiten's vector
  package, `Draw`'s own `ScreenGameMenu` case, with no CPU composite either way) — this is a real,
  separate geometry witness the round 2 registry omitted entirely.

### The two zero-witness composers closed

- `ComposeChargenFrame` — exercised by `TestComposeScreenSelectsChargenComposer`
  (`pkg/ui/screendispatch_test.go`), which compares `composeScreen()`'s own output byte-for-byte
  against a direct `ComposeChargenFrame(c)` call with the same model.
- `RenderPanel` — exercised by `TestRenderPanelDrawsAOnePixelBorderOnAFixedBox`
  (`pkg/ui/screendispatch_test.go`), a fixed `image.Pt(40, 20)` box: all four corners are the border
  colour, three interior points are the fill colour, and `RenderPanel`'s returned bounds are exactly
  `(0,0)-(40,20)` (a fixed `Size` is not resized to content).

### The five destination-rect mutations from the contract's "measured starting set"

| site (contract's `4de5d8b` numbering) | current site | status |
|---|---|---|
| `pkg/ui/chargen_page.go:100`, `chargenName`'s control rect | `preControlRegion`'s `chargenName` case | **killed**. `TestPreCreateNameControlRegionIsPublished` (`chargen_page_test.go`) pins `image.Rect(448,20,628,46)` and checks the hit test agrees at its own corner. +1px on `Min.X` fails both assertions; reverted byte-identical. |
| `pkg/ui/chargen_page.go:742`, the plate copy's source rect | inside `composeChargenDetailedPage` | **deferred**. This is the chargen **detailed** page's own geometry, which story 1022 is re-laying this cycle. Witnessing it now would pin a layout about to change and cost both lanes. Left for the next story that touches this screen after 1022 lands. |
| `pkg/ui/tippanel.go:497`, the tip panel's left border tile | `drawNinePatchBorder` | **killed**. `TestDrawNinePatchBorderTilesFourDistinctEdges` (`ninepatchborder_test.go`, new) builds an 8-region synthetic border and checks every one of the four edge tiles and four corners lands at its own boundary pixels, plus one interior "no centre" pixel. +1px on the left tile's destination X fails two boundary assertions; reverted byte-identical. |
| `pkg/ui/townshell.go:348`, a shell rect `image.Rect(200,196,280,228)` | `schoolSkillRects[0][0]` | **killed**. `TestSchoolSkillRectsMatchTheMeasuredLiterals` (`townshell_test.go`, new) pins all ten `schoolSkillRects` entries against hand-transcribed literals. +1px on `[0][0].Min.X` fails; reverted byte-identical. |
| `pkg/ui/townshell.go:869`, the button value's own text rect | `townButtonValueRect` (new, extracted from an inline literal in `ComposeTownSurface`'s button loop) | **killed**. `TestTownButtonTextRectsSplitTheWellIntoLabelThenValue` (`townshell_test.go`, new) pins both `townButtonLabelRect` and `townButtonValueRect` against a hand-picked well and hand-computed output rects. +1px on the value rect's `Min.X` fails; reverted byte-identical. |

Four of five are closed. The fifth is explicitly deferred to the 1022 boundary and is not a miss:
witnessing it now would have required pinning geometry both lanes know is about to move.

### The seven dispatch-layer symbols (review W-1)

`pkg/ui/screendispatch_test.go` witnesses selection (which composer a given App/flow state reaches),
never a screen's own destination rectangles — B2's geometry witnesses above own the rectangles.

| symbol | test |
|---|---|
| `(*App).composeScreen` | `TestComposeScreenSelectsMenuComposer`, `TestComposeScreenSelectsChargenComposer`, `TestComposeScreenSelectsTownComposer` (one per switch arm reached) |
| `dialogueOrigin` | `TestDialogueOriginCentersDialogueInTheFrame` |
| `composeTownRoom` (package-level, including hover/drag) | `TestAppComposeTownRoomThreadsLiveCursorAndDragState` (drag branch fully; the surface hover/press branch is a disclosed gap, below) |
| `(*App).composeTownRoom` | same test |
| `(*App).composeTownScreen` | `TestComposeScreenSelectsTownComposer` |
| `(*App).composeChargenScreen` | `TestComposeScreenSelectsChargenComposer` |
| `HeadlessFrame`'s `note` branch | `TestHeadlessFrameNoteNamesTheGameMenuOverlay` |

`overlayTownDialogue` is not on this list; it was witnessed before this story
(`towndialogueoverlay_test.go`, `TestTheWorldMapTakesNoDialogueOverlay`).

**Disclosed gap.** `composeTownRoom`'s hover/press branch for a general-purpose or tavern surface
room was investigated and found to have no visible effect through the CPU composite: `HoverCell` is
only visually consumed inside the school room's skill-slot rendering path (`townshell.go`, the
`artSchool`-gated block at `:813`). A search of every non-test reference to `HoverCell` across the
module finds 7 occurrences in 5 files (round 2 review, "correct one count"; verified by `grep`
here): the struct field declaration itself (`pkg/ui/townshell.go:176`); the one read, behind the
`artSchool` gate (`pkg/ui/townshell.go:833`, current line — `:824` before this round's
`townSurfaceMessageRect` extraction moved it); one runtime write (`pkg/ui/app.go:2887`,
`surface.HoverCell = c.Index`); and four more construction-site writes of a static or computed
value, none of them runtime hover state — `pkg/game/townshell.go:149`, `cmd/plaquescreens/main.go:99`
and `:114`, `cmd/schoolcheck/main.go:516`. The load-bearing half of the original claim stands: the
single read is the `artSchool`-gated one, `v.Press` is read separately at `townshell.go:867`, and the
no-art arm at `townshell.go:874` passes a literal `false`. A surface button's own press/hover
appearance only differs from its rest state when real button art
(`pic[0]`/`pic[1]`) is loaded — with no art, `drawTownShellBox` always draws the same "false" state
regardless of hover or press. `TestAppComposeTownRoomThreadsLiveCursorAndDragState` therefore
witnesses the App-level cursor/drag wiring through the shop room's drag icon, which is visible with
no art, rather than through a general surface room's hover state, which is not. The school room's
own hover-driven skill-slot highlight, if any exists, is untouched by this story and unwitnessed by
this test; `TestSchoolPaintsEachClassIconAtItsOwnRectangle` (pre-existing) covers the skill icon
placement itself, not hover.

## B3 — the gated layer is visible from the ungated chain

`internal/gatedtests` (new package) statically finds every top-level `func TestXxx(t *testing.T)` in
this module that reaches a `t.Skip`/`t.Skipf` call whose string argument contains `AGAINROM_`,
either in its own body or in one same-package helper function it calls by a bare identifier (a
"one-level indirection" — most `TestRelease*` tests in `pkg/game` call a shared `releaseFront(t)`
helper that carries the skip, rather than skipping in their own body). The search is by
`go/parser`/`go/ast`, walks the whole module tree (skipping `research/` and `builds/`), and needs no
install.

`internal/gatedtests/scan_test.go`'s `TestScanMatchesTheCheckedInPopulationList` requires the live
scan to match `internal/gatedtests/testdata/population.txt` exactly, in both directions: a gated
test added without a line in that file fails, and a stale line the scan no longer finds also fails.
Verified by mutation both directions: removing a real line from the manifest produced "Scan found
..., not listed"; adding a fictitious line produced "...lists ..., Scan did not find it"; both
reverted byte-identical.

**Depth is one call level, disclosed and not exhaustive.** A test reaching a gated skip through two
or more levels of same-package helper calls would not be found; the observed population (35 tests
across `pkg/game` and `cmd/missionrun`) needs at most one level, and no such deeper chain was found
while enumerating it. A future test built with a deeper helper chain would silently read as
ungated by this scanner; `TestScanMatchesTheCheckedInPopulationList`'s reconciliation against
`pipeline/check-release-tests.sh`'s own run-time count (below) is what would catch that case, not
the scanner alone.

**A second blind spot is shared with `check-release-tests.sh` itself, and the depth-reconciliation
above does not catch it (round 2 adversarial review, finding 4).** Both this scanner and that script
define a gated test by a `t.Skip`/`t.Skipf` call naming an `AGAINROM_` variable. A test that instead
reads such a variable and returns silently, with no `Skip` call, is reported `ok` by `go test`
exactly like an ordinary pass, and is invisible to both instruments: this scanner finds no `Skip`
call to match, and the script's own `grep` for `--- SKIP` finds nothing either. Reconciling this
scanner's count against that script's run-time count — the mitigation named above for the depth
limit — does not catch this shape, because both sides of that reconciliation are blind to it
together. Measured directly, not inferred: a throwaway probe test of this shape (an early silent
`return` keyed on an unset `AGAINROM_` variable, no `Skip` call) was added to `pkg/game`, and
`Scan`'s own result stayed at 35 with every other instrument green; the same probe rewritten to call
`t.Skip` instead moved `Scan` to 36 and made `check-release-tests.sh` report a skip. The probe was
then deleted; no test of this shape is known to exist in this module at this writing. This is a
disclosed limit of what "gated" means here, not a claim that such a test has been found.

Population at this landing: **35** (30 in `pkg/game` reached through `TestRelease*` naming or a
direct skip, 3 more direct-skip tests in `pkg/game` outside that naming
(`TestInstallWordsOverALawfulInstall`, `TestPatrolWalksTheMissionTensPatrollers`,
`TestTheShippedCensusMatchesTheWearRule`), and 2 in `cmd/missionrun`). `TestReleasedEnvelopeOneSimulationFormFiftyFiveFixture`
(`pkg/game/save_test.go`) matches the `TestRelease*` naming convention but is not gated — it neither
skips nor calls `releaseFront` — and the scanner correctly excludes it; a name-based population count
would have included it in error.

## B4 — one command prints the census, all five columns

`cmd/screencensus` prints five numbered sections, each stating what it selected (round 2 adversarial
review, finding 1: two of five was not the contract). Sections 2 and 3 read `pkg/ui.ScreenCensus()`,
an exported function returning `screenRegistry`'s own rows — the only exported read of that
otherwise-unexported map, so the census reads the same source of truth
`screenregistry_test.go` polices, not a second hand-maintained table.

1. **Screens.** Built by running `go run ./cmd/screenshot -list` as a subprocess and passing its own
   output through verbatim. This is not a second copy of `cmd/screenshot`'s own `knownScreens` table
   kept in sync by convention: the census's screen count is `cmd/screenshot`'s count, read at
   request time, so the two cannot disagree. `-list` touches no asset root. Printed as
   "cmd/screenshot's own 9 known screen(s)" — matches the contract's "9" figure by construction, not
   by re-derivation.
2. **Composers.** One line per registered `ui.Screen` value: the composer it dispatches to, or its
   `Reason` where it has none. 7 values, `pkg/ui.ScreenCensus()`'s own count. This is the column
   pass 1 left inside `cmd/screenshot`'s free-text description only. This section also prints a
   fixed note (round 3) that its `gamemenu` row and section 1's `game-menu` row name different
   reachability frames — see B4's own "the game-menu cell" paragraph below.
3. **Synthetic witnesses.** Per composing screen, its `Test` (selection) printed on its own line,
   then its `EffectiveGeometryTests` (round 3: own `GeometryTests` plus, when `SharesGeometry` is
   set, the screen it shares a composer call with — `ScreenCensusEntry.SharedWith` names that
   screen and each inherited line is suffixed `(from <name>)`), labelled `selection:` and
   `geometry:` respectively so the two are never read as one column. A screen with an empty
   `EffectiveGeometryTests` prints `geometry: none`.
4. **Gated tests.** `internal/gatedtests.Scan`'s live result, printed as "N test function(s)
   reaching an AGAINROM_-gated t.Skip/t.Skipf within one call level", one row per test naming its
   package, whether it is direct or via a helper, and a subject phrase derived mechanically from the
   test's own name (`gatedtests.Subject`: strip the leading "Test", lower-case, space CamelCase
   boundaries). Unchanged from pass 1.
5. **Unwitnessed remainder.** Computed, not asserted: every composing screen whose
   `EffectiveGeometryTests` is empty (`unwitnessedRemainder`, `cmd/screencensus/main.go`). At this
   landing that count is **0** — round 2 left `ScreenGameMenu` printed here as a false gap (its
   frame is `ScreenTown`'s own, witnessed by construction; round 3, above). A second, separate list,
   **known geometry gaps**, prints every entry carrying a non-empty `screenEntry.GeometryGapNote`
   (`geometryGaps`, `cmd/screencensus/main.go`) — at this landing, one: `ScreenChargen`'s detailed
   page. This replaces round 2's hardcoded three-line paragraph naming the same fact: the note's own
   text and the count above it both come from registry data now, so a later commit that gives the
   detailed page its own `GeometryTests` entry and clears `GeometryGapNote` in the same edit changes
   both numbers together. Verified by mutation (round 3): adding a 10th row to
   `cmd/screenshot`'s `knownScreens` moves section 1's count from 9 to 10 in both its header and its
   `selected:` line; reverted byte-identical.

**Two granularities are printed side by side, neither collapsed into the other.**
`cmd/screenshot -list` names 9 screens (chargen splits pre-create/detailed; town splits
square/shop). `pkg/ui.ScreenCensus()` is indexed at `composeScreen`'s own 7-value dispatch
granularity, where `ScreenChargen` and `ScreenTown` each cover two of the 9. Sections 2 and 3
print 7 rows for this reason; force-fitting them into 9 rows would either duplicate a row or hide
that `ScreenChargen`'s `GeometryTests` covers only one of its two pages — exactly the gap section 5
exists to name.

`cmd/screencensus` does not itself shell out to `pipeline/check-release-tests.sh`: that script lives
in the `againrom/` orchestrator tree, one level above this repository and outside its module, and a
binary built from this module must not assume a fixed relative path to a sibling checkout. The
census prints a note directing the reader to reconcile its gated count by hand against that script's
own run-time count. At this landing that reconciliation was performed directly (not through the
census binary) and both counts agree at 35 on both the `en` and `ru` asset roots (verification.md's
successor section below, and the story's return report).

**This is a recorded deviation from the contract's own literal wording, not a design choice quietly
substituted for it.** The contract states, twice (B4), that a disagreement between the census's
gated-test count and `check-release-tests.sh`'s own run-time count "is a failure rather than a
note". The shipped tool prints a note. The module-boundary reason above is sound — a binary in this
module cannot invoke a script outside it — but it does not make the contract's sentence true of the
shipped tool; `closure.md` records this explicitly rather than reporting "no in-scope GAP" over it.

**The game-menu cell.** `cmd/screenshot -list` (section 1) names `game-menu` as reached over a
running mission and refusing; `pkg/ui.ScreenCensus()` (section 2) names `gamemenu` as composing
(`(*App).composeTownScreen`). Both are correct, about different reachability frames: `game-menu` is
the map-showing case, which `composeScreen`'s own switch never reaches at all (`Draw`'s
`mapShowing()` branch returns first); `gamemenu` is the `ScreenGameMenu` dispatch value, reached only
when the town, not a mission, is beneath the menu (`Draw`'s own comment: "the only surface that
reaches here is the town"), and even then it composes only the town frame beneath the menu, not the
menu's own dim/panel (ebiten vector, no CPU composite either way). Section 2's own printed output
(round 3) states this rather than requiring the two tables to be read side by side to reconcile.

**Two contract numbers have no census column: the five destination-rect mutation sites (above) and
the seven dispatch-layer symbols (below).** B4 says "every number quoted in this contract must come
out of it afterwards"; these two numbers come out of `spec.md`'s own reconciliation tables, which
name the test that reproduces each figure, rather than out of `cmd/screencensus`'s five sections.
B4's own five sections are screens, composers, synthetic witnesses, gated witnesses and the
unwitnessed remainder — a mutation-site count and a dispatch-symbol count are neither. This is
recorded here as the reading applied, not silently: a future story is free to give either its own
census column, or to read B4's sentence as scoped to the five printed sections and amend it to say so.

`internal/archtest`'s allow-map entry for `cmd/screencensus` gained a second edge this round,
`"pkg/ui"`, alongside the pass-1 `"internal/gatedtests"` edge, with a comment stating why the
composers/synthetic-witnesses sections need it.

## The process-rule addendum

`AGENTS.md`'s coverage-and-witness list carries rule 6, reworded this round (round 2 adversarial
review, finding 2: the pass-1 wording pointed at the registry's `Test` column, which cannot fail on
a geometry change). The rule now names the two columns separately: a story that changes what a
registered screen draws extends a `GeometryTests` entry, never `Test`, before the production change,
and the extension is expected to fail against the pre-change code. `SDD/WORKFLOW.md`'s "Screen
geometry tests" section (renamed from "Screen composition tests", immediately before "Definition of
Done") carries the same correction. `SDD/` is gitignored in this worktree (confirmed via
`git check-ignore -v SDD/WORKFLOW.md`), so this local copy's edit is not part of the pushed commit;
the canonical `SDD/WORKFLOW.md` outside this worktree needs the same correction applied separately.
This is stated as an open item in `closure.md`.

## Out of scope, unchanged from the contract

The mission screen's CPU composite; rewriting or renaming the 30 pre-existing `TestRelease*`
functions; deleting any test; game assets in a test. None of these were touched.

## Divergence rows

None opened. This story changes no shipped behaviour and states nothing about ROM1 (contract,
"Divergence rows"). No row was added to `docs/DIVERGENCES.md`.
