# Verification — main menu with map picker and NEW GAME

Evidence about the implementation as landed. Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` →
`plan.md` §Success criteria; `DDx` → `plan.md` §Design decisions; `R-x` → `plan.md` §Risks; `Tn` →
`tasks.md`.

Environment: Go 1.26.1 (the toolchain pinned in `go.mod`), Windows 11. Automated evidence was produced
with **no game install visible to the test suite** — every fixture is bytes built in test code or
written to a temp directory. The developer-run evidence used a lawful GOG install with the asset root
supplied through `-assets` and through `AGAINROM_ASSETS`; **no game byte, bitmap or screenshot is
committed**. The `research/` submodule stayed pinned at `da54e6d` throughout and was not touched.

**This is the second revision, after the owner's defect pass.** The story first landed at `de7432a`;
the owner ran that build against their install and reported four defects, then added two more items
while the fixes were in flight. `F1`…`F5` are the commits that answer them. Everything below is
re-run evidence about the tree as it stands now — the earlier revision's numbers are in git, not
restated here as if they still described the code.

**The defects, and what each turned out to be:**

| Reported | What it was |
|---|---|
| "exit does not work" — the EXIT *button* | Real. Button 8 was unbound, on a stated reason ("no command id is decoded") that was false for that one button. **F1**. Esc was traced end to end and is correct; nothing was changed on that path. |
| "no map names in NEW GAME's map picker" | The list order, not the decoder. 28 nameless campaign maps sorted ahead of 10 named loose ones, and only 25 rows fit. **F2**. |
| "it does not look like all 38 maps are there" | The same cause, seen from the other side, plus no sign on screen that the list continued. **F2** + **F3**. |
| the run note has no ready-to-run command | Fixed in `builds/0010-main-menu/README.md`, which is untracked and therefore appears in no commit. |
| the wheel does nothing in the picker | It was never wired. **F4**. |
| the window is too small | It opened at 1× the frame. **F5**, at 2×. |

## Gate results

Run from the repo root at `45c4a10`, the head of the defect pass. The gate ran clean before every one
of `F1`…`F5` as well; this is the final run.

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `go test -count=1 ./...` | **all packages ok** — `cmd/againrom`, `cmd/mapview`, `cmd/terraintool`, `internal/archtest`, `internal/notices`, `internal/synth`, `pkg/formats/alm`, `pkg/formats/res`, `pkg/formats/spr256`, `pkg/game`, `pkg/render/camera`, `pkg/render/frame`, `pkg/render/menu`, `pkg/render/terrain`, `pkg/ui` |
| `gofmt -l $(git ls-files '*.go')` | empty |
| `internal/archtest` import DAG | green (runs inside `go test`, and separately) — no package gained or lost an import in this pass |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-no-game-assets.sh --history` | `clean (history scan)` |
| `SDD-Task` bijection over `8ce1ea0..de7432a` | 14 IDs (`T1`…`T14`), 0 duplicates |
| `SDD-Task` bijection over `de7432a..HEAD` | 5 IDs (`F1`…`F5`), 0 duplicates; the two docs commits carry none, as workflow artifacts |
| `Co-Authored-By` trailers | 0 |
| `research/` submodule | `da54e6d`, unchanged — read for `MENU-STATE-007`'s wording, never bumped |

## Automated criteria

| SC | Method | Result |
|---|---|---|
| SC-1 | `TestBuildMapList`, `TestDirMapsScan`, `TestArchiveMapsScan` (`pkg/game`) | **pass** — one list in the FR-2a order over the same 12-row fixture, re-derived by hand under the revised rule: **every loose row precedes every archive row**, asserted both by the whole-slice comparison and by its own subtest that finds the group boundary and checks it falls after exactly the loose inputs; within each group, case-insensitive by source text with the case-sensitive tiebreak. The list is compared as a whole slice, not by spot checks; every input contributes exactly one row; a source text present in **both** sources yields two rows with the loose one first and each keeping its own recorded name; empty-name rows listed and choosable; undecodable rows listed, not choosable, distinguished in `Text()`; a well-framed map with a broken grid is listed **and choosable**; determinism under three permutations of the sources' `Names()` order. `DirMaps` keeps `.alm` in any case, drops `KIDS.LM`/`x.al`/`y.almx`/`noext`, does not scan a subdirectory, and does not list a *directory* named `dir.alm` |
| SC-2 | `TestFitAndMapping` (`pkg/render/frame`) | **pass** — containment and centring asserted on the exact rational at 13 window shapes including 700×500, 644×494 and 642×481; letterbox and out-of-window positions map to no frame pixel; `WindowToFrame ∘ FrameToWindow` is the identity wherever `ok`, and `ok` is exactly membership in the forward map's image, verified by construction on downscaled windows; 1×1 contains the whole frame; a non-positive window is invalid with every mapping refusing and no division executed. All exact, no tolerance |
| SC-3 | `TestFlow` (`pkg/ui`) | **pass** — a fresh flow starts on the **menu**; NEW GAME moves menu→picker; a loadable choice moves picker→map; a **failing** loader stays in the picker with a non-empty message and that row demoted, without exiting; Esc moves map→picker and picker→menu and returns exit only from the menu; the same 8-event sequence from two fresh flows gives identical results |
| SC-4 | `TestPicker` (`pkg/ui`) | **pass** — at 38 rows: `Move` clamps at both ends; the scroll window follows minimally, checked against an independently transcribed model at all 74 steps of a walk down and back; `Select` is absolute and refuses out-of-range; `RowAt` agrees with `Visible()` at the first row, the last visible row, one line above, one line past, and past a short list, before and after scrolling, and ignores x; the click path `RowAt → Select → Choose` reaches the row under the cursor; `SetUnusable` demotes while leaving the row listed |
| SC-5 | the nine pre-existing `cmd/mapview` tests + `TestFlagSet` | **pass** — `git diff` against the pre-story baseline is **empty** for `cmd/mapview/main_test.go`; the summary text, the cadence clause and the failure paths are unchanged; `TestFlagSet` pins the exact usage line and all eight flag names, with an undefined flag still rejected so the check cannot pass vacuously |
| SC-6 | `TestViewerStep`, `TestMapScreenRoutesThroughStep` (`pkg/ui`) | **pass** — each pan key moves by `PanSpeed`; edge-scroll fires with the right axis and sign at all four edges and at both sides of the margin boundary, and **not** when the cursor has left the window; a wheel notch zooms about the cursor with the anchor preserved; the zoom stays in range over 200 notches; the camera stays clamped over a mixed 60-step sequence; the water counter advances from the **injected** timestamp with the first call only taking the baseline. The structural half drives an `App` on the map screen and a standalone `Viewer` with identical input and requires identical camera state |
| SC-7 | `TestButtonAt` (`pkg/render/menu`) | **pass** — the eight hot indices select 1…8 in order; index 0, the edge ramp `0x10`…`0x1e` and strays `0x01/0x7f/0x81/0x8f/0xff` select none; ten out-of-frame points select none; the same `Pix` under an **all-black palette** gives identical answers; a **vertically mirrored** mask changes which button a fixed point selects |
| SC-8 | `TestSelectionAndCompose` (`pkg/render/menu`) | **pass** — hover composites at the hover rect, pressed at the **pressed** rect, both 1:1; dragged off while held the frame equals the base and no other button lights; releasing off activates nothing, releasing on activates and reports **which** button; a press on no button latches nothing while hover keeps tracking; `Overlay` never yields two images; with nothing selected `Compose` is pixel-identical to the base; the differing-pixel set over all 18 reachable states is contained in exactly one rectangle |
| SC-9 | `TestNewGameButton` (`pkg/render/menu`) | **pass** — the computed rect is `(112,64)…(324,200)`; it is the smallest-`y` **left-column** entry and not the bottom-left one; a smallest-`y`-over-the-whole-table rule computed independently in the test picks **button 5**, and the two are asserted to differ, with the four-pixel gap (64 vs 60) asserted explicitly; both placement tables match the decoded values entry by entry |
| SC-10 | `TestLoadRejects` (`pkg/render/menu`), `TestOpenArchives` (`pkg/game`), `TestStartupFailures` (`cmd/againrom`) | **pass** — a missing entry, a base or mask not 640×480, an overlay disagreeing with its own table row, a non-bitmap entry and each missing archive all give a non-nil error and a **nil** result, never a partial one, naming the offending entry or path; with two defects the **first** in the fixed order is reported and the second is not mentioned |
| SC-11 | `TestCheck`, `TestStartupFailures` (`cmd/againrom`) | **pass** — over a synthetic install the summary line is exactly `againrom: 5 map rows, 8 of 8 buttons have a mask region` with exit 0 and nothing on stderr; the root works from `-assets` and from `AGAINROM_ASSETS`, and the flag wins when both are set; a mask with one empty region prints `7 of 8` and still exits **0**; no root configured exits non-zero with nothing on stdout; each of the four FR-10 failure kinds exits non-zero with the message on **stderr** in **both** the windowed and the headless mode |
| SC-12 | `internal/archtest` + `docs/ARCHITECTURE.md` | **pass (automated half)** — the fail-closed DAG check is green with both new packages registered stdlib-only and `pkg/ui` gaining no import outside `pkg/render*`/Ebitengine. The doc half is a **reviewed edit, not a test**: nothing checks `ARCHITECTURE.md` against the allow-map, and both new packages were added to both of its tables by hand |
| SC-16 | `TestPickerHeaderText` (`pkg/ui`) | **pass** — at 38 rows the header carries the total and the **1-based** visible range, read back out of the header text and compared against `Visible()` rather than against a literal, at the top of the list, mid-scroll, at the bottom and back up; a 22-row list reports `maps 1-22 of 22`; an empty list reports `no maps` and no range at all; the header never exceeds `pickerCols`. Mutation check: changing the first number from `top+1` to `top` fails two subtests |
| SC-17 | `TestExitButton` (`pkg/render/menu`), `TestAppDispatch` (`pkg/ui`) | **pass** — `ExitButton` is 8, is inside `[1, ButtonCount]`, is not `NewGameButton`, and its hover rect is placement entry 8, which the test independently computes to be the bottom entry of the right column; press-and-release on it reports exit, a press on it released elsewhere does not and changes nothing, and each of the **six** buttons that is neither NEW GAME nor EXIT is pressed and released with no exit and no screen change (the count of six is itself asserted). Mutation check: removing the dispatch arm fails it |
| SC-18 | `TestAppDispatch` (`pkg/ui`) | **pass** — on the picker one notch towards the user moves the selection by `pickerWheelRows` and one away moves it back; rolling 38 notches each way clamps at the last and the first row with `Visible()` following and **nothing chosen**; the same input on the **menu** and on the **map** screen leaves the picker's selection and window untouched |
| F5's own criterion | `TestDefaultWindowIsIntegerScaled` (`pkg/ui`) | **pass** — the default window is an exact `MenuWindowScale` multiple of the frame, a freshly built `App` reports `Scale() == 2` and `Origin() == (0,0)` before `Layout` is ever called, and both frame corners round-trip through the mapping. The number 2 is a judgement about desktops; what the test pins is that it is an **integer**, because the blit is nearest-neighbour |

**The marker revision (T17) — AC-16, P-6, SC-19.** Gate re-run from the repo root at `a8af4db`, the
head of that task: `go build ./...` clean, `go vet ./...` clean, `gofmt -l $(git ls-files '*.go')`
empty, `go test -trimpath -count=1 ./...` all packages **ok**, `scripts/check-no-game-assets.sh`
`clean (tree scan)`, `scripts/check-doc-budget.sh` exit 0, `scripts/check-sdd-audit.sh` exit 0 with 19
trailered commits and 0 duplicates. `git diff` for `cmd/mapview/main_test.go` and
`cmd/mapview/flagset_test.go` against their pre-change state is **empty**, and no `pkg/ui` or
`pkg/render/terrain` test was edited.

| SC | Method | Result |
|---|---|---|
| SC-19 (AC-16, P-6) | `TestMarkerCells`, `TestLoadMapViewerMarkers` (`pkg/game`), `TestFrontEndMarkers` (`pkg/game`, internal), `TestMarkerFlag` (`cmd/againrom`) | **pass** — the derivation answers `(2,1)`, `(0,4)`, `(5,0)`, `(1,3)`, `(2,0)` for anchors that are deliberately **not** cell-aligned (`0x0280` is 2.5 cells, `0x01c0` is 1.75), so a centre offset, a border inset or a rounding rule would have shown; asked for both, the loaded viewer reports both overlays on holding exactly one cell per record (2 and 3); asked for one, only that one; asked for neither, both off with no cells. The front-end's own loader carries the field through to the map screen in both states, and the flag's parsed default is **on**, `-markers=false` parses, both values reach the front-end, `NewFrontEnd`'s default is on, and the `-check` line is byte-identical with the flag and without it |

**Each of those was checked to bite, and the checks were reverted.** Deleting the `SetObjects` call
from the load path failed `TestLoadMapViewerMarkers` and `TestFrontEndMarkers`; adding `+1` to a marker
column failed `TestMarkerCells` at both object cells; flipping `defaultMarkers` to `false` failed
`TestMarkerFlag`. All three edits were reverted before the task commit, which is the state the gate
above was run in.

**What SC-19 does not witness, stated plainly.** That the markers are *visible on the map screen* is
not automated here — it is the same R-9 gap the rest of the front-end carries, and it is not claimed.
What is claimed is the wiring: the overlays are on, holding the right number of cells derived from the
right records. Their geometry, clipping, height lift and draw order are 0008's, 0009's and 0015's,
verified in those stories and unchanged by this revision — `pkg/render/terrain` was not touched and
`pkg/ui` gained two read-only accessors and nothing else.

**Separate-context discipline.** The acceptance tests for `pkg/render/frame`, `pkg/render/menu` (both
halves), the picker and the flow in `pkg/ui`, and the map list in `pkg/game` were authored by contexts
that had read `spec.md` and the relevant `plan.md` sections but **not** the implementation. Each ran a
mutation check on its own expectations and confirmed the relevant subtest fails before restoring.

**The defect pass's tests were not authored that way, and that is a recorded deviation.** SC-16, SC-17,
SC-18 and F5's criterion were written in the same context as the fixes they cover, under a rigor
calibration the owner set for this pass: one good test per behaviour actually changed plus its
boundary, no separate-context authors, no adversarial reads. Each of the three behavioural ones
carries a mutation check instead, recorded in its row above. The gated criteria SC-1…SC-15 keep the
separate-context authorship they were written with; SC-1's *expectation* was re-derived by hand for
the new ordering rule, in this context, which is the one place the original independence is diluted.

**One defect was found this way, and it was real.** The `pkg/render/frame` author's first run showed
`Origin()` returning a few-ulp **negative** value at 700×500 and 642×481. The cause was computing the
origin as `(win − extent·Scale())/2`, which rounds the scale and then subtracts two nearly equal
quantities; measured over the 923 601 window sizes in 640×480…1920×1200 that form goes negative on
**16 629** of them (1.8 %). It is now a single correctly-rounded division of the exact integer
numerator and denominator and is non-negative everywhere. The test author declined to weaken the
assertion, which is why the defect surfaced rather than being absorbed.

**One criterion was found unachievable and was revised rather than met.** The same run showed
`Origin + Scale()·extent ≤ win` failing by 5.68e-14 px at 700×500 **even with the origin at exactly
zero** — the overshoot lives entirely in the product, so no formulation of the origin rescues it.
`plan.md` DD5/SC-2 were revised to assert containment on the **exact rational**, where it holds by
construction, and to hold the float draw transform only to being within one ulp. That changes how the
plan verifies the contract, not the contract: `spec.md` P-2 speaks of the *mapping*, and the exact
integer mapping satisfies it with no rounding. **No spec revision was made and none is implied.**

**One assertion the plan asked for was deliberately not written.** SC-9 originally required a test
that permuting the mask's index→button table leaves `NewGameButton` unchanged. `newGameButton` is a
pure function of the placement table; no mask value is in scope where it is computed, so such a test
would pass by construction whatever the code did. The separate-context author refused to write it and
was right; DD8/SC-9 now record that guarantee as **structural rather than asserted**.

## Developer-run evidence (T15)

Install: a lawful GOG *Rage of Mages* install. Binary: `builds/0010-main-menu/againrom.exe`, rebuilt
from `45c4a10`. Nothing was written anywhere inside the repository.

### AC-6 / SC-13 — headless, and it passes

    PS> .uilds0-main-menugainrom.exe -assets "<install>" -check
    againrom: 38 map rows, 8 of 8 buttons have a mask region
    exit=0

    real  0m0.092s

- **38 map rows**, and the split was confirmed **independently of the code under test**: `ls` counts
  **10** loose `.alm` files in the asset root, and `restool list scenario.res` counts **28** `.alm`
  entries. 10 + 28 = 38. **The count did not change in this pass, and never had.** Both defects that
  read as "maps are missing" were about order and presentation; the same 38 rows were there before.
- **8 of 8 buttons** carry a non-empty mask region.
- **Exit 0, and no window opened.**
- The same line and exit status with the root supplied through `AGAINROM_ASSETS` instead of `-assets`,
  and from `go run ./cmd/againrom -assets …` run at the `implementation` root.
- With no root configured at all: the message on stderr and **exit 2**.
- **R-5 measured, not assumed:** the whole startup — three archives opened, eighteen menu bitmaps
  decoded and validated, the tileset built, and 38 maps' metadata read — takes **0.092 s** wall clock
  on this run (0.155 s on the previous revision's; nothing in this pass touched that path, and the
  difference is run-to-run noise, not a measured improvement). The metadata-only read (DD11) is what
  keeps it from being 38 full map decodes.

### The list order against the real install (F2)

Read out of `game.NewFrontEnd` against the install, before and after F2, through a throwaway program
in a scratch directory outside the repository — no game byte and no output file went anywhere near
the tree.

**Before** (rows 0–24, i.e. exactly the picker's visible window): `10.alm`, `100.alm`, `101.alm`,
`110.alm`, `111.alm`, `120.alm`, `121.alm`, `130.alm`, `131.alm`, `140.alm`, `141.alm`, `150.alm`,
`151.alm`, `20.alm`, `30.alm`, `31.alm`, `40.alm`, `41.alm`, `50.alm`, `51.alm`, `60.alm`, `61.alm`,
`70.alm`, `71.alm`, `80.alm` — **every one of them recording an empty name**. The 10 named loose maps
sat at rows 28–37, off screen. That is the whole of defects 2 and 3, reproduced.

**After**: rows 0–9 are `Beast.ALM — "Beast  Land "`, `Cross.ALM — "Crossroads of Mystery"`,
`Forester.alm — "Forester"`, `Horror.alm — "Horror"`, `Islands.alm — "Deadly Islands"`,
`Kids.alm — "Kids Paradise"`, `Kids2.ALM — "Kids Paradise II"`, `LuMoir.alm — "LuMoir"`,
`Tomb.ALM — "Heroes' Tomb"`, `Waters.alm — "Waters"`; rows 10–37 are the campaign maps beginning
`10.alm`. The rows, their sources and their recorded names are byte-identical to before — only the
order moved.

Two things worth recording about the data itself: the **decoder was never at fault** (the names above
came out correctly on the pre-fix build too), and exactly **one** campaign map records a name —
`81.alm`, `"Panic"` — which is why DD19 groups by source kind rather than by "has a name".

### What the mask counts do and do not evidence

`8 of 8` says every button's hot index is present somewhere in the shipped mask. It says **nothing**
about *where* those regions are, and in particular nothing about whether the region carrying the first
index is the visually top-left one. That question is AC-11's, and it is not answered below.

## Limitations and criteria not claimed

Nothing that needs a human at a window is claimed as passed. That is unchanged by this pass, and the
list has grown.

- **AC-11 / SC-15 and AC-14 — PENDING, not run. These are the load-bearing ones, and they are the
  same question at opposite corners of the brooch.** No human has put a cursor on the real menu.
  1. **That the button which lights is the visually top-left one, and that the one which quits is the
     visually bottom-right one.** The research labels the hit mask's stored DIB row order as
     *inferred* — concluded from each button's placement rectangle bracketing its mask region, not
     read directly. The brooch layout is nearly symmetric about the horizontal midline, so a
     vertically flipped read maps 1↔4, 2↔3, 5↔8 and 6↔7 onto each other to within a few pixels and
     would very nearly satisfy that same bracketing check.
     **A correction to what the previous revision of this file claimed here.** It said that because
     NEW GAME is identified by its placement rectangle and never by a mask index, "a wrong row order
     changes only *which art lights*, never *which action a rectangle performs*". That over-claims.
     `Assets.ButtonAt` answers with a **mask-derived** number, and every downstream decision keys off
     that number: `Overlay` draws `HoverRects[n-1]` and the dispatch compares `n` to `NewGameButton`.
     So under a flipped read, hovering the screen area `(112,64)…(324,200)` would light the
     *bottom*-left rectangle **and** clicking it would open nothing — the picker would answer to the
     bottom-left gem instead. Likewise EXIT would answer to the top-right gem, not the bottom-right
     one. What DD8 actually buys is **falsifiability**, and that is worth having: bound to the table
     row rather than to an index literal, a flip shows up as a visible mismatch — the gem you hover
     lights a rectangle somewhere else — instead of silently and consistently relabelling the whole
     brooch, which is what an index literal would have done. `TestButtonAt`'s mirrored-mask case
     proves the code is **sensitive** to row order, so the manual check can falsify the inference
     rather than pass vacuously — but it has not been run. `plan.md` R-1 now carries the corrected
     statement.
     **One piece of weak evidence exists, and it is testimony, not an observation made here.** The
     owner's defect report says "in NEW GAME's map picker there are no map names" — so on the
     pre-fix build a brooch gem did open the picker, which means the gem they pressed answered with
     button 1. If that was the visually top-left gem, the inferred row order is right. Nobody
     recorded *which* gem was pressed, so this is suggestive and not evidence; it is exactly what
     AC-11 asks the owner to state explicitly on the re-run.
  2. **That hover and pressed art are not swapped.** That role assignment rests on `MENU-STATE-007`,
     which is **Medium** confidence. Every synthetic criterion passes identically under either
     assignment, because synthetic assets cannot tell hover art from pressed art.
  3. **That the EXIT binding is the right one at all.** `MENU-STATE-007` is Medium, and it is the
     claim that says the click dispatcher posts `WM_CLOSE` for button 8. Its *number* is decoded and
     quoted verbatim in the claim; what a human confirms is that the gem carrying that number is the
     one a player would call EXIT.
- **AC-7a / SC-14 — PENDING, not run *here*.** No human has opened the window from this context,
  clicked NEW GAME, picked a map and unwound with Esc. The owner did run the **pre-fix** build at
  `de7432a` and reached the picker, which is where four of the six reports came from; that is testimony
  about a build two commits' worth of behaviour ago, and it is recorded above as such rather than
  counted as evidence for this one. What is unevidenced is precisely the engine-facing dispatch and
  presentation layer (R-9): `App.Run`, the window itself, `WritePixels`, `DebugPrintAt`, and what the
  frame actually looks like. Everything decidable below that layer — the frame mapping, the flow, the picker, the
  composition, the latch, and the dispatch itself through `App.step` — is asserted headlessly above.
  Note that Ebitengine's draw calls do run without a graphics context, so `App.Draw` is exercised for
  not panicking on every screen; what cannot be done is reading pixels back out of an `*ebiten.Image`,
  which panics before the game starts.
- **Esc-closes-the-window in the standalone viewer has no automated pin, and never had one.**
  `inpututil.IsKeyJustPressed` offers no test seam, and none of the nine pre-existing `cmd/mapview`
  tests reaches `Viewer.Run()` — all nine run under `-check`. Its protection is that the Esc branch of
  `Viewer.Update` and the whole of `Viewer.Run` are **textually unchanged** by this story, which
  `git diff` confirms, plus AC-7a when it runs. It is not claimed as automated evidence.

**Esc at the front-end's menu was traced end to end for defect 1, and it is correct.** The trace, so
that "we looked" is checkable rather than asserted:

1. `readAppInput` sets `Escape` from `inpututil.IsKeyJustPressed(ebiten.KeyEscape)`.
2. `App.step` handles `in.Escape` **first**, before any per-screen dispatch, so no screen can swallow
   it.
3. `flow.escape()` returns `true` from `ScreenMenu` and `false` from the other two.
4. `App.Update` turns that `true` into `ebiten.Termination`.
5. `ebiten.RunGame` maps `Termination` to a **nil** return — `errors.Is(err, Termination)` in
   `run.go` of `ebiten v2.9.9`, the version `go.mod` pins — and `App.Run` guards it a second time.
6. `cmd/againrom`'s `run()` therefore returns **0**, which `main` passes to `os.Exit`.

Nothing on that path was changed. `pkg/ui/viewer.go`'s own Escape branch is **unreachable** from the
front-end: `App` calls `Viewer.step`, never `Viewer.Update`, and only one `RunGame` runs per process
(`ebiten.RunGame(a)` in `App.Run`, `ebiten.RunGame(v)` in `Viewer.Run`). The two paths cannot fight.
`TestAppDispatch` asserts the first four steps headlessly; steps 5 and 6 are read from the pinned
dependency's source and from `cmd/againrom`'s own tests, not observed at a window.

Smaller notes, none a defect:

- **A residual gap this story did not create and did not close.** `scripts/check-no-game-assets.sh`
  keys on an extension list that includes neither `bmp` nor `png`, and `.gitignore` does not ignore
  them. Nothing this story ships writes an image file — there is no export path, no `-out` flag and no
  on-disk cache (DD23) — so the protection here is that no code path produces such a file at all. But
  the guard would not catch a developer who extracted menu art by hand. Widening it would change
  0000's contract, so it is **reported rather than silently changed**. It is worth the owner's
  decision.
- **The two BMP readers in the tree accept slightly different subsets.** `pkg/render/terrain`'s
  rejects a negative DIB height; `pkg/render/menu`'s accepts it as top-down. The divergence is
  deliberate and recorded (DD4/DD7) so that a later extraction is a decision rather than a discovery.
- **Button 7's hover and pressed rectangles are identical** in the decoded tables (`324, 236, 236,
  152` for both), so on that button a rectangle-only assertion cannot catch a hover/pressed swap; the
  pixel-content comparison is what covers it there.
- **An honest limit on the opaque-copy assertion.** The overlays decode with alpha forced opaque, so
  `draw.Src` and `draw.Over` are pixel-indistinguishable. The test catches any averaging or a missing
  blit, but cannot by construction distinguish the two operators — which is why DD10 calls that a
  code-clarity decision rather than a behavioural one.
- **Scope decisions, as they now stand.** **Button 8 IS wired to quit** (F1) — the previous revision
  of this file recorded the opposite, on a reason that did not survive reading `MENU-STATE-007`. The
  other **six** buttons bind nothing, because that claim reports their message ids as undecoded and
  guessing one would violate golden rule 4. The flow still exposes no activation entry point other
  than NEW GAME; EXIT is dispatched in `App` because it changes no screen (DD31). The per-button
  disable bitfield is still not implemented, so every button is treated as enabled and nothing
  suppresses an overlay. `graphics.res` **is** required at startup although the menu never reads it,
  and `TestStartupFailures` asserts that explicitly.
- **Three things this pass changed that only a human can judge.** The wheel's **direction** (rolling
  away from the user moves towards the top of the list — the platform convention, matching the
  viewer's existing scroll-up-zooms-in, but unwatched); the wheel's **step** of three rows; and the
  default window at **1280×960**. All three are settled arbitrarily-but-defensibly and are exactly
  the kind of thing that reads wrong on first contact.

## Conclusion

Every automated criterion (SC-1…SC-12, SC-16…SC-18) passes. The gated ones keep the separate-context
authorship they were written with; the defect pass's three were written alongside their fixes under the
owner's calibration for this pass and carry mutation checks instead, which is recorded above rather
than glossed. The headless developer-run criterion (SC-13 / AC-6) passes against a lawful install: 38
map rows — independently confirmed as 10 loose plus 28 archived — 8 of 8 buttons with a mask region,
exit 0, no window, in 0.092 s. The standalone viewer's flag set, summary text and failure paths are
unchanged, pinned by nine pre-existing tests that neither this story nor this pass modified.

Of the six things the owner reported, **four are answered by a change with a test behind it**
(EXIT, the list order, the list's extent, the wheel), **one is a settled default that only looks right
or wrong at a window** (1280×960), and **one produced no commit at all** (the run note, under
untracked `builds/`).

**The story is complete except for AC-7a, AC-11 and AC-14**, all of which need a human at a window and
are declared pending above. AC-11 and AC-14 are the consequential ones and they are one question:
until they run, the correspondence between the visually top-left gem and the button the code lights,
and between the visually bottom-right gem and the button that quits, both rest on an **inferred** mask
row order; the hover-versus-pressed roles and the button-8 → `WM_CLOSE` binding both rest on a
**Medium** claim. None is marked passed, and no conclusion here rests on any of them.

**What the next person at the window should check, in order.** The brooch first: hover the top-left
gem and confirm the highlight lands on it and not on the bottom-left one, then click it and confirm the
picker opens; then press and release the bottom-right gem and confirm the program exits. Those two
settle the row order. Then the picker: the first ten rows should be the named loose maps, the header
should read `maps 1-25 of 38`, and the wheel should move the selection three rows per notch in the
direction that feels right. Then the window: 1280×960 at startup, resizable, the brooch art crisp
rather than blurred. Anything that fails there is data about the inference or about a default, not a
regression in what is asserted above.
