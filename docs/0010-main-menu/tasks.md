# Tasks — main menu with map picker and NEW GAME

**Reading key.** `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DDx` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds, not interchangeable:
**implementation** (one coherent product change → exactly one implementation commit, trailer
`SDD-Task: 0010-main-menu/T<n>` — `F<n>` for the defect pass, because ids are never reused),
**developer-run verification** (agent authors, the developer or a human runs against a lawful install;
produces evidence, not an implementation commit), **release-gate** (produces a runnable artifact and an
index entry; nothing under it is committed).

**Two new packages enter a fail-closed DAG.** `internal/archtest` rejects any module package missing
from its allow-map, so the change that adds `pkg/render/frame` registers it and the change that adds
`pkg/render/menu` registers that; neither registration can be deferred. `docs/ARCHITECTURE.md` is
edited alongside, because the check is authoritative and the doc must not lag it. `internal/synth` needs
no registration — `archtest.Load` skips everything under `internal/`.

**The characterization pins land before the code they protect** (DD24), each as its own change so a
bisect separates "the pin was wrong" from "the change broke it": T6 before T7, T12 before T13.

**No existing test file is edited.** `cmd/mapview/main_test.go`, `pkg/ui/viewer_test.go`,
`pkg/ui/overlay_test.go`, `pkg/ui/unit_overlay_test.go` and `pkg/formats/alm/alm_test.go` must pass
byte-identical to their pre-story versions. Work that finds it needs one of them edited has changed
shipped behaviour and must stop rather than adjust the pin.

**Separate-context tests.** The acceptance tests for the decidable areas — `pkg/render/frame`,
`pkg/render/menu` (both halves), the picker and the flow in `pkg/ui`, and the map list in `pkg/game` —
are authored by contexts that have read `spec.md` and the relevant `plan.md` sections but **not** the
implementation, so their oracles are independent transcriptions of the spec rather than restatements of
the code. `internal/synth` builds **inputs only** (DD30), so sharing it across those contexts cannot
make a test agree with the implementation by construction.

## T1 — the 640×480 virtual frame  *(implementation)*

The letterbox layer, alone and first: nothing else in the story can place a cursor without it.

- Files: `pkg/render/frame/frame.go` (ADD) — `W`, `H`, `Placement`, `Fit`, `Valid`, `WindowToFrame`,
  `FrameToWindow`, `Scale`, `Origin`, in the exact `int64` rational arithmetic of DD5, with floats
  confined to `Scale`/`Origin` and used only to build a draw transform (DD26).
  `pkg/render/frame/frame_test.go` (ADD, separate-context).
  `internal/archtest/dag.go` (MODIFY) — register `"pkg/render/frame": {}`.
  `docs/ARCHITECTURE.md` (MODIFY) — one tier row, one DAG row.
- Covers: FR-7 (the frame half); AC-2; P-2; SC-2, SC-12; DD5, DD26; **R-7**.
- **Done when:** `TestFitAndMapping` passes with **exact** equalities and no tolerance — including
  700×500, 644×494 and 642×481, the sizes at which a float implementation provably loses a pixel row;
  a 1×1 window still contains the whole frame; a non-positive window is `Valid() == false` with every
  mapping refusing and no division executed; `WindowToFrame ∘ FrameToWindow` is the identity wherever
  `ok`, and `ok` is false exactly where no window pixel maps to that frame pixel — and the standing gate
  is clean.

## T2 — synthetic fixtures and the menu's BMP readers  *(implementation)*

The bytes side: the shared builders every later acceptance test needs, and the two readers that consume
them. Landing them together is deliberate — a builder with no reader has nothing to assert against, and
a reader with no builder has nothing to read.

- Files: `internal/synth/synth.go` (ADD) — `Archive`, `BMP8`, `BMP24`, `ALM`, `MenuArchive` with
  per-entry overrides (DD30); inputs only, no expected outputs.
  `pkg/render/menu/bmp.go` (ADD) — the unexported 24-bpp and 8-bpp readers (DD4, DD7): key on
  `bfOffBits`, require only that enough bytes are present, positive height = bottom-up normalised to
  top-down, negative height accepted as top-down, raw palette indices preserved for 8 bpp.
  `pkg/render/menu/bmp_test.go`, `internal/synth/synth_test.go` (ADD).
  `internal/archtest/dag.go` (MODIFY) — register `"pkg/render/menu": {}`.
  `docs/ARCHITECTURE.md` (MODIFY) — one tier row, one DAG row.
- Covers: FR-10 (the decode half); AC-12; SC-10, SC-12; DD4, DD7, DD30; **R-1** (the row-order
  convention is written here and nowhere else).
- **Done when:** `TestDecodeBMP` passes — a 24-bpp stream decodes to the declared dimensions with the
  expected pixels; an 8-bpp stream preserves **raw indices** rather than resolving them through the
  palette; a positive-height stream is normalised bottom-up→top-down and a negative-height one is taken
  as already top-down, with the two producing mirrored rows from the same pixel bytes; **a stream with
  trailing bytes beyond the computed minimum decodes** (every shipped menu bitmap has two) while a
  truncated one is rejected; a bad magic, a wrong DIB header size, an unsupported bpp and a pixel offset
  outside the stream are each rejected with a nil image; `internal/synth`'s builders round-trip through
  the readers and through `res.Open`/`alm.Open` — and the standing gate is clean.

## T3 — the menu asset contract  *(implementation)*

What the eighteen entries are, how a mask index becomes a button, which rectangle each overlay occupies,
which button NEW GAME is, and what makes an asset set invalid. Depends on T2.

- Files: `pkg/render/menu/menu.go` (ADD) — `FrameW`/`FrameH`, `ButtonCount = 8`, `EntryPrefix =
  "graphics/mainmenu/"` and the eighteen entry names (DD3), `HoverRects`/`PressedRects` as
  `[8]image.Rectangle` built from the decoded `{x,y,w,h}` tables, `ColumnSplit = 320`, `NewGameButton`
  computed by the DD8 rule, `EntrySource`, `Assets`, `Load` (DD28's fixed order, first failure, nil on
  error), `ButtonAt` (DD6's explicit 256-entry table), `MaskRegions` (DD29's per-button pixel counts).
  `pkg/render/menu/menu_test.go` (ADD, separate-context).
- Covers: FR-8 (the selection half), FR-9a (the identification half), FR-10; AC-9, AC-12; P-3, P-4;
  SC-7, SC-9, SC-10; DD1, DD2, DD3, DD6, DD8, DD28, DD29; **R-1** (the geometry anchoring and the
  row-order sensitivity witness).
- **Done when:** `TestButtonAt` passes — the eight hot indices select 1…8; index 0, the edge ramp
  `0x10`…`0x1e` and assorted strays select none; out-of-frame selects none; an **all-black palette**
  changes nothing (the palette is never consulted); and a **vertically mirrored** mask changes which
  button a fixed point selects. `TestNewGameButton` passes — the computed rect is `(112,64)…(324,200)`,
  it is not the bottom-left entry, a smallest-`y`-over-the-whole-table rule computed in the test picks a
  **different** button, and permuting the index→button table leaves `NewGameButton` unchanged.
  `TestLoadRejects` passes — a missing entry, a base or mask not 640×480, and an overlay disagreeing
  with its own table row each give a non-nil error and a **nil** asset set naming the entry, and a set
  with two defects reports the **first** in DD28's order. `TestMaskRegions` passes — a button with no
  pixel of its index counts zero and the rest count their exact pixel totals — and the standing gate is
  clean.

## T4 — selection state and opaque composition  *(implementation)*

The behavioural half of the menu: the latch, and the frame it produces. Depends on T3.

- Files: `pkg/render/menu/state.go` (ADD) — `State{Selected int; Pressed bool}`, `Selection` with
  `Move`/`Press`/`Release`/`State` per DD9, `Assets.Overlay(State)` returning **at most one** image with
  its rectangle, and `Assets.Compose(State) *image.RGBA` = base copy then one `draw.Src` blit (DD10).
  `pkg/render/menu/state_test.go` (ADD, separate-context).
- Covers: FR-8 (the overlay half), FR-9a (the activation half); AC-10; P-5; SC-8; DD9, DD10; **R-2**
  (the hover/pressed roles are selected here and nowhere else, so the Medium claim has exactly one
  consumer), **R-3** (the opaque copy is the whole transparency policy).
- **Done when:** `TestSelectionAndCompose` passes — hover composites the hover bitmap 1:1 at the hover
  rect; press composites the pressed bitmap at the **pressed** rect; moved off while held the frame
  equals the base and no other button lights; releasing off the latched button activates nothing;
  releasing on it activates it and reports **which** button, so a caller can tell NEW GAME from the
  other seven, EXIT among them; a press on no button latches nothing while hover keeps tracking;
  `Overlay` never yields two images; with nothing selected `Compose` is pixel-identical to the base —
  and the standing gate is clean.

## T5 — a metadata-only map read  *(implementation)*

The additive `alm` entry point that makes FR-2a's and FR-3's two failure states distinguishable.

- Files: `pkg/formats/alm/alm.go` (MODIFY) — extract the ten-record walk and the type-0 payload decode
  into unexported helpers used by **both** `Open` and the new `OpenInfo`; add `Info` and `OpenInfo`
  (DD11). `Open`'s signature, behaviour and error strings unchanged.
  `pkg/formats/alm/info_test.go` (ADD).
- Covers: FR-2a (the recorded-name read), FR-3 (the "passed metadata, failed full decode" state); AC-1a,
  AC-3a; DD11; **R-5** (the listing pass decodes only type-0).
- Brownfield: `pkg/formats/alm/alm_test.go` is the characterization pin and is **modified by no task**.
- **Done when:** `TestOpenInfo` passes — `OpenInfo` agrees with `Open` on width, height, name and
  description for every fixture where both succeed; on a **well-framed** map whose type-1 record
  declares a `payloadSize` inconsistent with `2·W·H`, `OpenInfo` **succeeds** and `Open` **fails**; a
  physically truncated file fails both; a bad magic, a wrong `recordCount` and the rejected
  `formatVersion` fail both identically — **every test in the unmodified `alm_test.go` still passes**,
  and the standing gate is clean.

## T6 — pin the shipped `Viewer.Update` before touching it  *(implementation)*

A characterization pin, landing **before** T7 restructures the method. It exists because nothing in the
repo pins `Update` today, although it is callable headlessly (DD24).

- Files: `pkg/ui/update_pin_test.go` (ADD).
- Covers: FR-4a; AC-5a; SC-5; DD24; **R-4**.
- **Done when:** `TestUpdatePinnedBehaviour` passes **against the unchanged `viewer.go`** — with the
  camera placed away from its clamp bounds, each `Update()` returns `nil` and moves the camera by
  exactly the shipped edge-scroll amount on both axes (the headless cursor reads `(0,0)`, which is
  inside the view), and the first call establishes the animation baseline without firing a burst of
  ticks — and the standing gate is clean.

## T7 — one camera step behind an input snapshot  *(implementation)*

The seam FR-4a needs: both entry points advance the camera through the same method. Depends on T6.

- Files: `pkg/ui/viewer.go` (MODIFY) — add `Input` (raw engine reads only: pan keys, cursor position,
  wheel; **no** "cursor inside" flag, which is a function of the camera and stays inside `step`),
  `readInput()`, and `(*Viewer).step(in Input, now time.Time)` holding the animation advance, the pan
  intent and the wheel zoom; `Update` becomes its existing Esc branch — **textually unchanged** — plus
  `v.step(readInput(), time.Now())`. `Run` is not touched.
  `pkg/ui/input_test.go` (ADD).
- Covers: FR-4a; AC-5a; SC-6 (the behavioural half); DD12, DD24; **R-4**.
- Brownfield: `pkg/ui/viewer_test.go`, `overlay_test.go`, `unit_overlay_test.go` and T6's pin are the
  pins and are **modified by no task**.
- **Done when:** `TestViewerStep` passes — each pan key moves the camera by `PanSpeed` in world pixels;
  edge-scroll fires only while the cursor is inside the view, with the correct axis and sign at all four
  edges; a wheel notch zooms about the cursor within the zoom limits; the camera clamps at the world
  edges under any sequence; the water counter advances from the **injected** timestamp and the first
  call only establishes the baseline — **T6's pin passes unmodified**, every pre-existing `pkg/ui` test
  passes unmodified, `git diff` shows the Esc branch and `Run` unchanged, and the standing gate is
  clean.

## T8 — the picker model  *(implementation)*

Rows, selection, the scroll window and the frame hit-test — everything a human needs to operate a
38-row list, decided without an engine.

- Files: `pkg/ui/picker.go` (ADD) — `PickerRow`, `Picker`, the DD13 layout constants, `Move`, `Select`,
  `RowAt` (y-only), `Choose`, `SetUnusable`, `Visible`, `Selection`, and the `"> "` / `"  "` row prefix
  the draw path uses.
  `pkg/ui/picker_test.go` (ADD, separate-context).
- Covers: FR-3 (the picker half); AC-4; P-1; SC-4; DD13; **R-6**.
- **Done when:** `TestPicker` passes at **38** rows, where the list is longer than the window — `Move`
  clamps at both ends; the scroll window follows the selection minimally and never shows a row that does
  not exist; `Select` sets an absolute index, scrolls it into view and refuses an out-of-range one;
  `RowAt` agrees with `Visible()` at the first row, the last visible row, one line above the list, one
  line past it and past the last row; the click path `RowAt → Select → Choose` reaches exactly the row
  under the cursor; `Choose` refuses an unchoosable row; `SetUnusable` demotes a choosable one; and the
  selected row is the only one carrying the marker prefix — and the standing gate is clean.

## T9 — the screen flow  *(implementation)*

The pure menu ↔ picker ↔ map state machine, including the load that fails. Depends on T8.

- Files: `pkg/ui/flow.go` (ADD) — `Screen`, `MapLoader`, `flow` with the **menu as its initial screen**,
  `activateNewGame`, `choose`, `escape`, and the message a failed load sets (DD14).
  `pkg/ui/flow_test.go` (ADD, separate-context).
- Covers: FR-3, FR-7 (the Esc semantics and the root screen), FR-9a (menu→picker; the six unbound
  buttons do nothing); AC-3a; P-1; SC-3; DD14.
- **Done when:** `TestFlow` passes — a fresh flow **starts on the menu**; NEW GAME moves menu→picker; a
  loadable choice moves picker→map; a **failing** loader leaves the flow in the picker with a non-empty
  message and that row unusable, without exiting; Esc moves map→picker and picker→menu; Esc at the menu
  reports exit and Esc nowhere else does; the same event sequence from the same start state is
  reproducible; and the flow exposes **no activation for any button other than NEW GAME**, so a release
  on one of the unbound buttons leaves screen, rows and message unchanged — and the standing gate is
  clean.

## T10 — the front-end shell  *(implementation)*

The Ebitengine `Game` that owns the 640×480 offscreen frame, draws each screen into it, scales it to the
window, and dispatches one input snapshot. Depends on T1, T3, T4, T7, T9.

- Files: `pkg/ui/app.go` (ADD) — `App`, `NewApp`, `appInput`, `(*App).step(in appInput, now time.Time)
  (exit bool)`, `Update` (gatherer only), `Draw`, `Layout`, `Run` (DD15, DD16, DD27). The menu frame is
  recomposed only when the `menu.State` changes; the picker screen draws header, marked rows clipped to
  `pickerCols`, and the flow's message at `pickerMessageY`; the map screen bypasses the frame and
  `Layout` forwards the window size to the viewer; the window opens at 640×480, resizable.
  `pkg/ui/app_test.go` (ADD).
- Covers: FR-3, FR-7, FR-8 (the dispatch half), FR-9a; AC-3a, AC-5a; SC-3, SC-6 (the structural half);
  DD15, DD16, DD27; **R-9**.
- **Done when:** `TestAppDispatch` passes — a cursor move over a hot mask pixel selects that button, a
  press latches it, a release on it activates NEW GAME and moves to the picker, a release off it does
  not, and a release on any other button changes nothing; a window position in the **letterbox** selects
  nothing; on the menu Up/Down/Enter are ignored and only Esc acts; in the picker Up/Down move, Enter
  chooses, a primary release chooses the row under the cursor, and Esc returns to the menu.
  `TestMapScreenRoutesThroughStep` passes — an `App` on the map screen and a standalone `Viewer` given
  the same `Input` and the same `Layout` call end in **identical** camera state.
  `TestAppDrawDoesNotPanic` calls `Draw` on each screen headlessly (pixels are never read back — they
  cannot be) — and the standing gate is clean.

## T11 — archives, tileset and the map inventory  *(implementation)*

The `pkg/game` half that reads an install: the three required archives, and the one interleaved map
list. Depends on T5.

- Files: `pkg/game/archives.go` (ADD) — `Archives`, `OpenArchives(root)` (FR-1 fail-early, all three
  including `graphics.res`, fixed order, first failure), `OpenTileset(path)` with the
  `open <path>: <err>` wrapping (DD17, DD20, DD25).
  `pkg/game/maplist.go` (ADD) — `MapEntry`, `NamedReader`, `BuildMapList`, `DirMaps` (non-recursive),
  `ArchiveMaps` (flat) (DD18, DD19).
  `pkg/game/archives_test.go`, `pkg/game/maplist_test.go` (ADD; the list test separate-context).
- Covers: FR-1, FR-2a, FR-10 (the archive half); AC-1a, AC-12; SC-1, SC-10; DD17, DD18, DD19, DD20, DD25.
- **Done when:** `TestBuildMapList` passes — one interleaved list in the DD19 order; every input
  contributes exactly one row; equal source texts from the two sources give two rows, loose first; an
  empty recorded name is listed and choosable; undecodable bytes are listed, distinguished in `Text()`
  and not choosable. `TestDirMapsScan` passes — `.ALM` in any case kept, a `.LM` near-miss dropped, a
  `.alm` in a **subdirectory** not scanned. `TestOpenArchives` passes — each of the three missing
  archives is reported by path with a non-nil error, and a set with two missing reports the first in
  DD17's order — and the standing gate is clean.

## T12 — pin `cmd/mapview`'s flag surface before touching it  *(implementation)*

The second characterization pin (DD24), landing **before** T13 rewires the command.

- Files: `cmd/mapview/flagset_test.go` (ADD).
- Covers: FR-4a; AC-5a; SC-5; DD24; **R-4**.
- **Done when:** `TestFlagSet` passes against the unchanged `main.go` — the flag set is exactly
  `assets, graphics, map, check, noanimation, speed, objects, units` and the `usage` string is exactly
  its shipped text — and the standing gate is clean.

## T13 — the shared map-load path, and `cmd/mapview` re-expressed through it  *(implementation)*

The extraction FR-4a demands and its first consumer, in one change because they are only correct
together: the command must not build against a half-moved path. Depends on T11, T12.

- Files: `pkg/game/mapload.go` (ADD) — `MapView{Viewer *ui.Viewer; Map *alm.Map; Title string}` and
  `LoadMapViewer(tiles, data, fallbackTitle)`, the **only** place map bytes become a running viewer;
  errors returned unwrapped so each caller labels them (DD20, DD25).
  `pkg/game/mapload_test.go` (ADD).
  `cmd/mapview/main.go` (MODIFY) — `load()` calls `game.OpenTileset` and `game.LoadMapViewer`; the
  summary literal, both error wrappings, the flag set, `usage`, the package doc comment and the Esc
  behaviour are untouched.
- Covers: FR-4a; AC-5a; SC-5, SC-6; DD20, DD25; **R-4**.
- Brownfield: `cmd/mapview/main_test.go` and T12's pin are the pins and are **modified by no task**.
- **Done when:** all **nine** pre-existing `cmd/mapview` tests and T12's pin pass **unmodified**;
  `git diff` of `cmd/mapview/main_test.go` against the pre-story baseline is empty; `TestLoadMapViewer`
  passes — the recorded map name titles the viewer, an empty name falls back to the supplied text, and a
  malformed map surfaces an error before any viewer exists; `open <path>` and `decode <path>` are still
  produced character-for-character — and the standing gate is clean.

## T14 — the front-end assembly and the game entry point  *(implementation)*

The last wiring: `pkg/game` composes archives + menu assets + tileset + map list into an app, and
`cmd/againrom` becomes the game's entry point with its headless mode, its asset-root contract and its
error contract. Depends on T10, T11, T13.

- Files: `pkg/game/frontend.go` (ADD) — `FrontEnd`, `NewFrontEnd(root)` (archives, menu assets, tileset
  built **once**, map list), the `MapEntry → ui.PickerRow` conversion, `CheckLine()` rendering
  `againrom: <N> map rows, <K> of 8 buttons have a mask region`, and `App()` building the `ui.App` with
  the loader closure that reads a chosen row's bytes and calls `LoadMapViewer` (DD17, DD21, DD29).
  `pkg/game/frontend_test.go` (ADD).
  `cmd/againrom/main.go` (MODIFY) — `-assets` **and** the `AGAINROM_ASSETS` fallback through
  `game.ResolveAssetRoot`, `-check`, `run(args, getenv, stdout, stderr) int`, FR-10 failures to stderr
  with a non-zero exit in **both** modes (DD22).
  `cmd/againrom/main_test.go` (ADD).
- Covers: FR-1, FR-2a, FR-3, FR-5, FR-7, FR-9a, FR-10; AC-3a, AC-6 (synthetic half), AC-12; P-4; SC-11;
  DD17, DD21, DD22, DD23, DD29.
- **Done when:** `TestCheck` passes over a synthetic install on a temp dir — the DD21 line with both
  counts correct, exit 0, no window; the same run works with the root from `-assets` **and** from
  `AGAINROM_ASSETS`, and the flag wins when both are set; a mask with one empty button region prints
  `7 of 8` and still exits 0; no asset root configured is reported on stderr and exits non-zero; every
  FR-10 failure prints to **stderr** and exits non-zero in both the windowed and the `-check` mode —
  and the standing gate is clean.

## T15 — evidence against a lawful install  *(developer-run verification)*

Run the front-end against a real install and record **evidence only** — no game bytes, no extracted art,
no screenshot.

- Produces: evidence in `verification.md`.
- Covers: AC-6, AC-7a, AC-11, AC-14; SC-13, SC-14, SC-15; R-1, R-2, R-3, R-5.
- **Three parts with different status, and they are not interchangeable:**
  *(a)* **SC-13 is fully headless and required.** `againrom -assets <root> -check` must print 38 map
  rows and 8 of 8 buttons with a mask region and exit 0; the observed line verbatim and the observed
  wall time (R-5) are the evidence. It has no "or a limitation" branch: the install is available and the
  run needs no window.
  *(b)* **SC-14 (AC-7a) needs a human at a window.** If it is not run it is recorded as an explicit
  pending limitation naming what was not observed — the engine-facing dispatch and presentation layer
  (R-9), and Esc-closes-the-window, which has no automated pin either (SC-5).
  *(c)* **SC-15 (AC-11) is the load-bearing one and likewise needs a human.** Its pending declaration
  must state precisely what stays unproven: that the **visually top-left** brooch button is the one that
  lights, and that hover and pressed are not swapped. It is never marked passed on the strength of a
  synthetic test, because no synthetic test can distinguish either case.
  *(d)* **AC-14 needs the same human and settles the same question from the other end.** Pressing and
  releasing the **visually bottom-right** brooch gem must end the program. Under a flipped mask row
  order it would be the top-right one instead (R-1), so AC-14 and AC-11 falsify or confirm the same
  inference at opposite corners of the brooch — either one failing is the same finding.
- **Done when:** part (a) has been run and the evidence records what was actually observed; parts (b)
  and (c) are recorded with their status stated plainly; `bash scripts/check-no-game-assets.sh` (tree
  and `--history`) stays clean and no game byte, bitmap or screenshot is committed.

## T16 — the runnable build  *(release-gate)*

- Produces: `builds/0010-main-menu/` holding the built `againrom` binary and a `README.md` giving the
  exact invocation against a lawful install, sourcing the asset root from `-assets`/`AGAINROM_ASSETS`
  and never a hardcoded path; plus a ticked row in `builds/README.md`.
- Covers: the project's build convention; FR-1's configuration rule.
- **Done when:** the binary is built from the story's head commit, the README states both invocations
  (windowed and `-check`) and both asset-root sources, `builds/README.md`'s `0010-main-menu` row is
  ticked, and **nothing under `builds/` is committed** — it is gitignored (golden rule 1).
- **Defect 4, folded in here.** The run note must open with a **copy-paste block that runs from the
  `implementation` root with the developer's own install path already filled in** — both the
  `go run ./cmd/againrom …` form and the built-binary form — because a note whose every command has to
  be edited before it runs is a note nobody runs. Nothing under `builds/` is ever committed
  (golden rule 1), so a real path here reaches no shipped source and does not touch golden rule 3,
  which is about what a *binary or its source* has compiled in. The placeholder-path section and the
  asset-root explanation stay below it for anyone whose install is elsewhere.

## The defect pass — F1…F5  *(implementation)*

Five slices from the owner's report on the landed build. They are **F**-numbered because `T1`…`T16` are
spent and an id is never reused; each is still one implementation task → exactly one commit, trailer
`SDD-Task: 0010-main-menu/F<n>`, and each carries the same standing gate. Defect 4 — the run note —
produces **no** commit at all: `builds/` is gitignored, so it lands under F5's rebuild and nowhere in
the history.

The order is deliberate: F1 is independent, F2 and F3 are the two halves of one root cause and F2 is
the one that must be right before F3's numbers mean anything, and F4/F5 touch the same dispatch and
window that F1 and F3 leave settled.

### F1 — the brooch's EXIT button quits  *(implementation)*

Defect 1. Button 8 is the one per-button command `MENU-STATE-007` decodes, so leaving it dead was a
scope decision, not a data gap, and the owner reversed it.

- Files: `pkg/render/menu/menu.go` (MODIFY) — `ExitButton` (DD31). `pkg/ui/app.go` (MODIFY) —
  `stepMenu` returns `exit` and `step` propagates it. `pkg/render/menu/menu_test.go`,
  `pkg/ui/app_test.go`, `pkg/ui/flow_test.go` (MODIFY).
- Covers: FR-11, FR-9a (six, not seven); AC-14's automatable half; SC-17; DD31; **R-1**.
- **Done when:** `TestExitButton` passes — `ExitButton` is in `[1, ButtonCount]`, is not
  `NewGameButton`, and its hover rect is entry 8, `(324,276)…(532,412)`, the bottom entry of the right
  column. `TestAppDispatch` passes — press and release both on EXIT reports exit; press on it released
  away from it does not and changes nothing; each of the six buttons that is neither NEW GAME nor EXIT
  is pressed and released with no exit and no screen change — and the standing gate is clean.

### F2 — the map list puts loose maps first  *(implementation)*

Defects 2 and 3, first half. The gated order put 28 nameless campaign maps ahead of 10 named loose
ones and filled the visible window with bare numbers.

- Files: `pkg/game/maplist.go` (MODIFY) — the sort key becomes `(kindRank, fold(Source), Source)`
  (DD19). `pkg/game/maplist_test.go` (MODIFY) — the hand-derived expectation re-derived under the new
  rule.
- Covers: FR-2a (the ordering half); AC-1a; SC-1; DD19; **R-6**.
- **Done when:** `TestBuildMapList` passes over the unchanged 12-row fixture — every loose row precedes
  every archive row, each group is ordered case-insensitively by source text with the case-sensitive
  tiebreak inside it, the two `dup.alm` rows are still two rows with the loose one first, and every
  other clause of SC-1 is unchanged — and the standing gate is clean.

### F3 — the picker states how long the list is  *(implementation)*

Defects 2 and 3, second half. A 38-row list drawn 25 rows at a time looked exactly like a 25-row list.

- Files: `pkg/ui/picker.go` (MODIFY) — `HeaderText()` derived from `Visible()` (DD13).
  `pkg/ui/app.go` (MODIFY) — `drawPicker` draws it. `pkg/ui/picker_test.go` (MODIFY).
- Covers: FR-2a (the extent half); AC-13; SC-16; DD13; **R-6**.
- **Done when:** `TestPickerHeaderText` passes — at 38 rows the header's total and **1-based** range
  agree with `Visible()` at the top of the list, mid-scroll and at the bottom, read out of the text
  rather than compared to a literal; a list shorter than the window reports its whole self; an empty
  list reports no range — and the standing gate is clean.

### F4 — the wheel scrolls the picker  *(implementation)*

The owner reached for the wheel and it did nothing.

- Files: `pkg/ui/app.go` (MODIFY) — `appInput.WheelY`, read in `readAppInput`, dispatched in
  `stepPicker` (DD32). `pkg/ui/picker.go` (MODIFY) — `pickerWheelRows`.
  `pkg/ui/app_test.go` (MODIFY).
- Covers: FR-12; AC-15; SC-18; DD32.
- **Done when:** `TestAppDispatch`'s wheel arm passes — on the picker a positive `WheelY` moves the
  selection towards row 0 by `pickerWheelRows` and a negative one towards the last row, `Visible()`
  follows, rolling past either end clamps and chooses nothing, and the same input on the menu and on the
  map screen leaves the picker untouched — and the standing gate is clean.

### F5 — the window opens at 2×  *(implementation)*

640×480 is a quarter of a 1080p desktop; the owner reported it as too small to use.

- Files: `pkg/ui/app.go` (MODIFY) — `MenuWindowW/H` = `frame.W*2`, `frame.H*2` (DD16).
- Covers: FR-7 (unchanged contract, new default); DD16.
- **Done when:** the startup size is an exact integer multiple of the frame, so `frame.Fit` yields the
  rational `2/1` and nothing is resampled; the letterbox arithmetic, `Layout`, and the map screen's
  window-sized camera are untouched; every existing `pkg/ui` test passes unmodified — and the standing
  gate is clean.

## T17 — the placement markers on the game's map screen  *(implementation)*

The FR-13 wiring in one change, because its halves are only correct together: the derivation and the
selection parameter in the shared load path, the front-end's default, the game's flag, and the
developer tool re-expressed through the widened path.

- Files: `pkg/game/mapload.go` (MODIFY) — `Markers`, `MarkerCells`, `LoadMapViewer` taking the
  selection and wiring both overlays. `pkg/game/frontend.go` (MODIFY) — the field, its default, its use
  on every load. `cmd/againrom/main.go` (MODIFY) — the flag, the usage line, the value reaching the
  front-end. `cmd/mapview/main.go` (MODIFY) — `load()` passes its two flags through; **its own
  signature and its summary building stay as they are**, so its pin needs no edit.
  `pkg/ui/viewer.go` (MODIFY) — read-only overlay accessors, no behaviour change.
  `pkg/game/mapload_test.go`, `cmd/againrom/main_test.go` (MODIFY); `pkg/game/frontend_test.go` (ADD).
- Covers: FR-13; AC-16; P-6; SC-19; DD33, DD20 (as revised).
- Brownfield: `cmd/mapview`'s two test files are the pins and are **modified by no task**.
- **Done when:** `TestMarkerCells`, `TestLoadMapViewerMarkers`, `TestFrontEndMarkers` and
  `TestMarkerFlag` pass; a map loaded without a selection leaves both overlays off; every pre-existing
  `cmd/mapview` and `pkg/ui` test passes **unmodified**, `git diff` of `cmd/mapview/main_test.go` is
  empty and that tool's summary text, flag set and `usage` are character-for-character unchanged — and
  the standing gate is clean.

## Traceability

Each row's plan criterion is one the plan itself attributes to that requirement.

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (window; root from flag **or** env; three required archives incl. `graphics.res`) | SC-10, SC-11, SC-13, SC-14 | T11, T14, T15 |
| FR-2a (one list, loose rows before archive rows, source + recorded name, unreadable rows, non-recursive scan, the extent stated on screen) | SC-1, SC-16 | T5, T11, F2, F3 |
| FR-3 (click + Up/Down/Enter, Esc unwinds, a failing full load is reported) | SC-3, SC-4 | T8, T9, T10 |
| FR-4a (one viewer behind two entry points; the standalone viewer unchanged) | SC-5, SC-6 | T6, T7, T10, T12, T13 |
| FR-5 (`-check` prints both counts, exits 0, no window; FR-10 overrides) | SC-11, SC-13 | T14, T15 |
| FR-7 (menu is the root screen; the 640×480 frame scaled, centred, letterboxed; Esc semantics) | SC-2, SC-3 | T1, T9, T10, F5 |
| FR-8 (at most one button selected; one opaque overlay 1:1; the latch) | SC-7, SC-8 | T3, T4, T10 |
| FR-9a (NEW GAME by placement rectangle, never by mask index; press+release on it; the six unbound buttons do nothing) | SC-9, SC-8, SC-3 | T3, T4, T9, T10, F1 |
| FR-11 (EXIT quits, by the decoded button number, same activation rule) | SC-17 | F1 |
| FR-12 (the wheel scrolls the picker) | SC-18 | F4 |
| FR-13 (both placement markers on the game's map screen; one flag, default on; the derivation independent of art placement) | SC-19 | T17 |
| FR-10 (incomplete or inconsistent assets reported before any window, non-zero exit) | SC-10, SC-11 | T2, T3, T11, T14 |
| AC-1a | SC-1 | T11, F2 |
| AC-2 | SC-2 | T1 |
| AC-3a | SC-3 | T9, T10, T14 |
| AC-4 | SC-4 | T8 |
| AC-5a | SC-5, SC-6 | T6, T7, T10, T12, T13 |
| AC-6 | SC-11 (synthetic), SC-13 (install) | T14, T15 |
| AC-7a | SC-14 — developer-run | T15 |
| AC-9 | SC-7 | T3 |
| AC-10 | SC-8 | T4 |
| AC-11 | SC-9 (the rule), SC-15 (the game) | T3, T15 |
| AC-12 | SC-10, SC-11 | T2, T3, T11, T14 |
| AC-13 | SC-16 | F3 |
| AC-14 | SC-17 (the rule), T15's live run (the game) | F1, T15 |
| AC-15 | SC-18 | F4 |
| AC-16 | SC-19 | T17 |
| P-1 (picker and transitions are pure and deterministic) | SC-3, SC-4 | T8, T9 |
| P-2 (the whole frame maps inside any window; invertible on its image) | SC-2 | T1 |
| P-3 (no non-hot index and no out-of-frame position selects anything) | SC-7 | T3 |
| P-4 (no partly-defined asset set is ever used) | SC-10 | T3, T11, T14 |
| P-5 (at most one overlay, at its own rectangle) | SC-8 | T4 |
| P-6 (a marker's cell is a function of its own placement record alone) | SC-19 | T17 |
| Constraint (mask read as raw indices; the palette carries no meaning) | SC-7 | T3 |
| Constraint (no invented transparency rule) | SC-8, SC-15 | T4, T15 |
| Constraint (asset root from flag/env; no game data in the repo; synthetic fixtures) | SC-11, SC-13 | T2, T14, T15, T16 |
| Constraint (the DAG; the render tier takes bytes, not formats types) | SC-12 | T1, T2 |
| Constraint (the picker's text is placeholder debug text; no font asset read) | SC-4 | T8, T10 |
| Out of scope (no command bound for the six buttons whose message ids are undecoded) | SC-3, SC-17 | T9, F1 |
| Out of scope (no disable bitfield; every button enabled) | SC-8 | T4 |
| Out of scope (no keyboard navigation of the menu buttons) | SC-3 | T10 |
| R-1 (inferred row order; it reaches EXIT too) | SC-7, SC-9, SC-15, SC-17 | T2, T3, T15, F1 |
| R-2 (Medium hover/pressed roles) | SC-15 | T4, T15 |
| R-4 (the standalone viewer must not drift) | SC-5, SC-6 | T6, T7, T12, T13 |
| R-5 (38 maps read at startup) | SC-13 | T5, T15 |
| R-6 (more rows than the frame shows — and no sign on screen that there are) | SC-4, SC-1, SC-16, SC-18 | T8, F2, F3, F4 |
| R-7 (the letterbox must not lose a pixel row) | SC-2 | T1 |
| R-8 (no bitmap is ever written) | SC-13 | T14, T15 |
| R-9 (the shell is not observable in tests) | SC-14 | T10, T15 |
