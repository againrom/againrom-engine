# Tasks — placed-units diagnostic overlay

**Reading key.** `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DDx` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0009-units-overlay/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; produces evidence,
not an implementation commit).

This story adds **no package**. `pkg/render/terrain`, `pkg/ui`, `cmd/terraintool` and `cmd/mapview` are
already in the DAG allow-map and `docs/ARCHITECTURE.md`, so no `internal/archtest` or `ARCHITECTURE.md`
edit belongs to any task — SC-8's first half is the existing fail-closed check *staying* green. The
render-tier geometry stays stdlib-only and imports no `formats` package (DD3); `pkg/ui` adds no new
dependency. The two cmds do the `alm.Map.Units → cells` wiring (DD3, DD8).

**The 0008 characterization pin.** `pkg/render/terrain/overlay_test.go` and `pkg/ui/overlay_test.go` are
the pin for the DD2 refactor and are **modified by no task in this story**. SC-8's second half is them
passing unchanged. A task that finds it needs to edit one of them has found a behaviour change and must
stop rather than adjust the pin.

**Separate-context tests.** `pkg/render/terrain/unit_overlay_test.go` (T1) and `pkg/ui/unit_overlay_test.go`
(T3) are authored by a context that has read `spec.md` and the relevant `plan.md` sections — the API
signatures and the FR-6/DD4/DD5/DD6 contract — but **not** the implementation, so their oracles are
independent transcriptions of the spec formula rather than restatements of the code. Because those files
share a package with the pin, they may reuse `specRound`, `specMapRect`, `sameRects`, `noPanicOverlay`,
`sinkRects`, `markerBG`, `assertScreenRects`, `rectInView` and `overlayViewer` verbatim, and MUST
re-derive under unit-specific names the helpers hardwired to the object constants: `specArmsRaw` →
`specUnitArmsRaw`, `specCross` → `specUnitCross`, `drawAndCheck` → `drawUnitsAndCheck`, `nativeArms` →
`nativeUnitArms`, `wantScreenRects` → `wantUnitScreenRects` (DD2's closing note).

## T1 — the shared marker core and the unit geometry  *(implementation)*

The whole render-tier half. Extract the arm/clip arithmetic 0008 ships into one unexported core, re-express
the object entry points through it unchanged, and add the unit entry points as its `(4, 1)` siblings. The
compositors are not touched — the overlay draws over their output (DD1).

- Files: `pkg/render/terrain/overlay.go` (MODIFY) — add unexported
  `markerRects(col, row, cols, rows, cellpx, radius, thickness int) []image.Rectangle` and
  `drawMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx, radius, thickness int,
  c color.RGBA)`, preserving the landed guard order (invalid scale, then off-map, then geometry), the
  horizontal-then-vertical arm order, the map-rect `Intersect` + empty drop, the fixed capacity-2 slice and
  the `len(out) == 0 → nil` collapse (DD2, DD4); re-express `ObjectMarkerRects`/`DrawObjectMarkers` as
  calls at `(markerArmRadius, markerArmThickness)` / `MarkerColor` with signatures, values and observable
  behaviour unchanged; add `var UnitMarkerColor = color.RGBA{0x00, 0xE5, 0xFF, 0xff}`,
  `unitArmRadius = 4`, `unitArmThickness = 1`, `UnitMarkerRects(col, row, cols, rows, cellpx int)
  []image.Rectangle` and `DrawUnitMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx int)`
  (DD4, DD5). Widen `AnchorCell`'s doc comment to name units as well as objects (DD3), and correct the
  `Intersect` comment: `Empty()` is used because it is the semantic test, **not** because `Intersect` can
  return an empty-but-non-zero rectangle — it normalises every empty result to the zero rectangle
  (`plan.md` baseline).
  `pkg/render/terrain/unit_overlay_test.go` (ADD, separate-context) — synthetic cell lists only; no game
  bytes.
- Covers: FR-1 (raster half), FR-2, FR-3, FR-4 (the PNG stacking half), FR-6; AC-1, AC-2, AC-3; P-1, P-2;
  SC-1, SC-2, SC-3, SC-4, SC-8; DD1, DD2, DD3, DD4, DD5; **R-3** (the pin must stay green **unmodified**
  through the refactor — that is the whole guard against a silent regression in shipped object geometry),
  **R-4** (the pixel half: the unit cross is a strict subset of the object cross, so SC-4 asserts both
  draw orders, not just the correct one), **R-6** (the even-thickness cases at `cellpx` 48/64/96 are where
  FR-6's centring term is observable at all).
- Trade-off accepted here: this is the story's largest slice and the only render-tier-only one (not
  demoable by itself). It stays whole because the core and the two entry points are only correct together,
  and both T2 and T3 consume them. It lands **before** any unit-specific wiring so a bisect separates "the
  refactor broke the object overlay" from "the units broke something" (R-3).
- **Done when:** `TestUnitAnchorCellAndOffMap`, `TestUnitMarkerRectsGeometry`,
  `TestUnitMarkerRectsExtremes` and `TestDrawUnitMarkers` pass — including at least one `cellpx ≥ 48` case
  where `⌊t/2⌋ = 1` (SC-2), the `cellpx ≤ 2` tiny-map case where the map-rect clip truncates and the
  `cellpx = 32` corner case where it does not, the undersized-`img` case (SC-4), and both stacking orders —
  **every test in the unmodified `pkg/render/terrain/overlay_test.go` still passes**, and the standing gate
  is clean.

## T2 — the `terraintool` opt-in overlay and count  *(implementation)*

Wire the PNG path: an opt-in flag that draws the unit markers after the object pass and reports the decoded
unit count, leaving the disabled path byte-for-byte the baseline in both object states. Depends on T1.

- Files: `cmd/terraintool/main.go` (MODIFY) — add `-units` (default off) with the help text
  `"overlay a diagnostic marker on each placed unit's anchor cell and report the unit count"`; add
  `[-units]` to the `usage` constant and the package doc comment; add `unitAnchorCells(m *alm.Map)
  []image.Point` beside the existing object helper (DD3, twin rather than generalisation); when the flag is
  set, call `terrain.DrawUnitMarkers` on the composited image **after** the object block with
  `m.Width`/`m.Height` and `terrain.CellSize*scale`, and append the literal `, units N`
  (`N = len(m.Units)`) after the objects token (DD7). `cmd/terraintool/main_test.go` (MODIFY) — parameterise
  the synthetic `.alm` fixture with unit anchor cells (in-map, one off-map, and one coincident with an
  object), and add a zero-unit fixture.
- Covers: FR-1, FR-4 (independent opt-in + PNG draw order), FR-5; AC-4, AC-5; P-3; SC-5; DD5, DD7;
  **R-5** (the token is emitted on the flag alone, so a unit-free map prints `units 0`), **R-6** (the
  `-scale 2` render is the reachable even-thickness case).
- **Done when:** `TestRenderUnitsOverlay` passes — every **in-map** unit anchor is cyan at `-scale 1` and
  `-scale 2`, the off-map unit is marked nowhere, cyan sits over yellow at the coincident cell, the summary
  carries `units N` after `objects N` and is identical under either CLI flag order, a zero-unit map prints
  `units 0`, and the flag-off PNG is byte-identical to the independently constructed oracle in **both**
  object states (`CompositeLit` alone, and `CompositeLit` + `DrawObjectMarkers`) with the pre-0009 summary
  shape — every pre-existing `cmd/terraintool` test still passes unchanged, and the standing gate is clean.

## T3 — the interactive viewer's unit overlay and its ordered pass list  *(implementation)*

The viewer half: unit overlay state and toggle, the shared camera transform, and — the part that exists for
FR-4 rather than for the units themselves — replacing `Draw`'s inline object loop with an ordered pass list
so terrain → objects → units is data a test can read instead of statement order nobody observes. Depends on
T1.

- Files: `pkg/ui/overlay.go` (MODIFY) — extract
  `(v *Viewer) overlayScreenRects(show bool, cells []image.Point, rects func(col, row, cols, rows, cellpx int) []image.Rectangle) []screenRect`
  (a **method**: it reads `v.cam` and `v.grid`), holding the nil-when-off rule, the per-arm
  `cam.WorldToScreen(Min)` + `Zoom`-scaled size, and the fully-outside-view cull; re-express
  `objectScreenRects()` through it; add `SetUnits(show bool, cells []image.Point)`, `unitScreenRects()`,
  the `overlayPass{Color color.RGBA; Rects []screenRect}` type and `overlayPasses() []overlayPass`
  returning objects then units, omitting a pass with no rects (DD6, DD8).
  `pkg/ui/viewer.go` (MODIFY) — `unitCells []image.Point` / `showUnits bool` state; `Draw` iterates
  `overlayPasses()` and fills each rect with `vector.DrawFilledRect(screen, …, pass.Color, false)`, float
  coordinates passed straight through (no independent snapping, FR-6).
  `pkg/ui/unit_overlay_test.go` (ADD, separate-context) — synthetic grid + positioned camera; the arm is
  transcribed from FR-6 and the placement derived from the camera contract, never a second call to the
  function under test.
- Covers: FR-1, FR-3 (the view-rect stage, realized by the framebuffer clip per DD6), FR-4 (the viewer draw
  order), FR-6 (viewer path); AC-4 (the automatable viewer witness); SC-6, SC-8, SC-10; DD6, DD8; **R-1**
  (the viewer is deliberately not pixel-identical to the PNG), **R-4** (SC-10 is what keeps a reversed pass
  order from silently erasing every coincident unit marker).
- **Done when:** `TestUnitScreenRects` passes at more than one pan/zoom position, culls a rect fully
  outside the view, pins all four toggle combinations, and returns nil with the unit overlay off or with no
  cells; `TestOverlayPassOrder` passes — two passes with `MarkerColor` first and `UnitMarkerColor` second
  when both are enabled, one pass when one is, none when neither; every pre-existing `pkg/ui` test still
  passes **unchanged**; and the standing gate is clean.

## T4 — the `mapview` opt-in overlay and count  *(implementation)*

Wire the interactive path: the flag reaches `load()`, which does the `alm.Map.Units → []image.Point`
conversion, hands the cells to the viewer, and reports the count on the headless summary between the object
count and the cadence clause. Depends on T3.

- Files: `cmd/mapview/main.go` (MODIFY) — add `-units` (default off) with the same help text as T2, add
  `[-units]` to the `usage` constant and the package doc comment, and pass it into `load()` as a fifth
  parameter after `showObjects`; when set, `load()` builds the cells via `terrain.AnchorCell`, calls
  `viewer.SetUnits(true, cells)` and appends `, units N` (`N = len(m.Units)`) immediately after the objects
  token, so `-check` prints it before the cadence `run()` adds (DD7, DD8).
  `cmd/mapview/main_test.go` (MODIFY) — parameterise the synthetic `.alm` fixture with unit anchor cells
  and add a zero-unit fixture.
- Covers: FR-1, FR-4, FR-5; AC-4, AC-5; SC-7; DD7, DD8; **R-5** (`units 0` on a unit-free map).
- **Done when:** `TestUnitsFlag` passes — `units N` appears with `N = len(m.Units)` in the DD7 position
  (after `objects N`, before `, water speed …`), identically under `-units -objects` and
  `-objects -units`; each flag alone emits only its own token; a zero-unit fixture prints `units 0`; and
  the no-flag summary is character-for-character the pre-0009 shape — every pre-existing `cmd/mapview`
  test still passes unchanged, and the standing gate is clean.

## T5 — overlay evidence against a lawful install  *(developer-run verification)*

Load a real GOG map with placed units in both viewers with both overlays on and record **evidence only** —
no game bytes, no rendered PNG, no map file.

- Produces: evidence in `verification.md` — the named map, its size and its reported `units N`; that
  `terraintool -objects -units` places cyan markers on cells that plausibly hold units, at `-scale 1` and
  `-scale 2`; and that `mapview -objects -units` shows the same markers and holds them on their terrain
  cells through pan and zoom.
- Covers: AC-6; SC-9, SC-11; R-1, R-4 (the observer-facing half), R-5.
- **Three parts with different status, per DD9, and they are not interchangeable:**
  *(a)* the **placement + count** half (SC-9) is headless and **required** — it has no "or a limitation"
  branch. Its count evidence must be recorded for what it is: `alm.Open` enforces
  `len(type6 payload) == 70 × Count6`, so `len(Units)`, `Count6` and `payloadSize/70` are the same number
  by construction and their agreement evidences a **successful decode, not a correct count**. The only
  external oracle is the Map Editor's or the game's own count for the named map; if it is not obtained,
  `verification.md` says so.
  *(b)* the **stacking** half is recorded as **not exercisable on real data** — no map in the shipped
  corpus places a unit and an object on the same cell — and is carried by SC-4 and SC-10 instead. It is
  not marked passed on the strength of a real render.
  *(c)* the **live pan/zoom** half (SC-11) is the story's one deferrable criterion. If no human runs it,
  `verification.md` records an explicit **pending limitation** naming what was not observed, AC-6 is not
  marked passed, and no conclusion rests on it.
- **Done when:** part (a) has been run and `verification.md` records what was actually observed; parts (b)
  and (c) are recorded with their status stated plainly. `bash scripts/check-no-game-assets.sh` stays
  clean and no rendered image, map file or game byte is committed.

## Traceability

Each row's plan criterion is one the plan itself attributes to that requirement.

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (opt-in marker draw in both viewers) | SC-4, SC-5, SC-6, SC-7, SC-9 | T1, T2, T3, T4, T5 |
| FR-2 (pure bounded geometry, no panic) | SC-1, SC-3, SC-4 | T1 |
| FR-2 / constraint (render tier stdlib-only, DAG) | SC-8 | T1, T3 |
| FR-3 (off-map first, then map-rect, then output-rect clip) | SC-1, SC-2, SC-3, SC-4 | T1 |
| FR-3 (view-rect stage in the viewer, via the framebuffer clip) | SC-6 | T3 |
| FR-4 (independent opt-in; units drawn over objects) | SC-4, SC-5, SC-6, SC-7, SC-10 | T1, T2, T3, T4 |
| FR-5 (report `len(Units)` after the object count) | SC-5, SC-7 | T2, T4 |
| FR-6 (the pinned pixel geometry) | SC-2, SC-4, SC-6, SC-9 | T1, T3, T5 |
| AC-1 | SC-1 | T1 |
| AC-2 | SC-2, SC-4 | T1 |
| AC-3 | SC-3, SC-4 | T1 |
| AC-4 | SC-5, SC-6, SC-7 | T2, T3, T4 |
| AC-5 | SC-5, SC-7 | T2, T4 |
| AC-6 (placement + count) | SC-9 | T5 |
| AC-6 (stacking) | SC-4, SC-10 — not exercisable on real data | T1, T3 |
| AC-6 (live pan/zoom) | SC-11 | T5 |
| P-1 (≤ 2 rects, never indexes out of bounds) | SC-1, SC-3, SC-4 | T1 |
| P-2 (no panic, no magnitude-proportional allocation) | SC-3 | T1 |
| P-3 (disabled ⇒ byte-identical, objects off and on) | SC-5, SC-7 | T2, T4 |
| R-1 (viewer not pixel-identical to the PNG — intended) | SC-6, SC-11 | T3, T5 |
| R-3 (the DD2 refactor must not move an object pixel) | SC-8 | T1 |
| R-4 (a reversed pass order erases, not degrades) | SC-4, SC-10 | T1, T3 |
| R-5 (`units 0` is representable and must report) | SC-5, SC-7, SC-9 | T2, T4, T5 |
| R-6 (even arm thickness at production scales) | SC-2, SC-5 | T1, T2 |
