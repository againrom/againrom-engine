# Plan — placed-units diagnostic overlay

Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`. This file fixes the API contract, the design decisions,
and the success criteria against which the work is verified. It is derivable from the spec alone.

## Approach

Add a placed-units diagnostic overlay as new, pure, stdlib-only geometry in `pkg/render/terrain` plus a
second draw pass in each of the two terrain consumers, composed **after** the placed-objects overlay
(0008). The render tier gains `UnitMarkerColor`, `UnitMarkerRects` (a function of `(anchorCell, mapExtent,
cellPixels)` returning ≤ 2 half-open `image.Rectangle` arms, FR-2/FR-6) and `DrawUnitMarkers` (the PNG
rasterizer, FR-1/FR-3). Because the object overlay already ships the identical arm arithmetic at different
constants, the two are **not** written twice: the arm/clip core is extracted into one unexported function
that both the object and the unit entry points call at their own radius, thickness and colour, with the
object entry points keeping their exported signatures and observable behaviour unchanged (DD2).
`cmd/terraintool` and `cmd/mapview` each gain an independent opt-in `-units` flag that draws the unit pass
after the object pass and appends a `, units N` token after the `, objects N` token (FR-4, FR-5). `pkg/ui`
gains unit overlay state, a `SetUnits` setter, a `unitScreenRects` transform beside the object one, and an
ordered pass list that `Draw` consumes so the terrain → objects → units order is **observable rather than
implicit** (DD6). Nothing is drawn and no token is emitted unless the flag is set, so with the unit
overlay off the output and summary are byte-for-byte what the same invocation produced before this story —
with the object overlay off **or** on (P-3, AC-4). No package is added, so the DAG is unchanged; the
marker geometry imports no `formats` package (the two cmds do the `alm.Map.Units → cells` wiring).

**Terrain: greenfield, and DD2 does not change that.** The spec declares this story greenfield. DD2
restructures two shipped files, so the declaration was worth re-testing rather than assuming — and it
holds: the playbook's brownfield trigger is *existing behaviour changes, or new code lands in a subsystem
that cannot be confidently described*. Neither fires. `ObjectMarkerRects`, `DrawObjectMarkers` and
`MarkerColor` keep their exported signatures, their values and their observable results at every input;
the restructuring is the arithmetic identity of binding two package constants to two parameters; and the
preservation is not a claim but a **mechanically pinned** fact — 0008's `pkg/render/terrain/overlay_test.go`
and `pkg/ui/overlay_test.go` are the characterization pin, are **modified by no task in this story**, and
exercise the exact object rectangles, the clip truncation, the extremes and the allocation profile.
**Correction recorded:** `analysis.md` said in advance that choosing parameterisation "is brownfield for
those two files". That was over-cautious — it named the files touched rather than the behaviour changed.
The plan corrects it here rather than leaving two artifacts disagreeing, and the spec's declaration stands
unrevised. This is a judgement call on a process label, not on the code, and it is the kind of call the
owner may reverse; if reversed, the consequence is added discipline (characterization before change),
which the unmodified 0008 tests already supply.

## Facts verified during planning (baseline, frozen)

- `pkg/formats/alm` exposes `Map.Units []alm.Unit`; each `Unit` carries `X, Y uint32` (fixed-point `/256`)
  **and nothing else** — there is no other field the overlay could read even by mistake. The map size is
  `Map.Width`/`Map.Height int`.
- `alm`'s type-6 decode requires `len(payload) == 70 * Meta.Count6` exactly and errors otherwise, so
  `len(Map.Units)` **is** the map's own declared type-6 count whenever `alm.Open` succeeds — an identity
  the decoder enforces, not one an independent reading could confirm (see the verification note on AC-6).
  It is a plain fixed-stride table with no adaptive walk and no extension record, so unlike the type-4
  objects there is no fixture-layout constraint on where an off-map anchor may sit: any unit record may
  carry any `X`/`Y`, and `#type6 = 0` is representable (an empty type-6 payload with `Meta.Count6 == 0`
  decodes to an empty slice, and `-units` on such a map must still report `units 0`).
- `pkg/render/terrain` is stdlib-only, registered in `internal/archtest` + `docs/ARCHITECTURE.md`; adding
  to it needs no DAG edit. `CellSize = 32`.
- The 0008 overlay as landed in `pkg/render/terrain/overlay.go`: `MarkerColor` (the opaque object yellow),
  package constants `markerArmRadius = 6` / `markerArmThickness = 3`, `AnchorCell(x, y uint32) (col, row
  int)` = `(int(x>>8), int(y>>8))`, `scaleDim(n, cellpx int) int` = `max(1, (n*cellpx + 16) / 32)`,
  `ObjectMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle` (invalid-scale and off-map both
  return `nil` **before** any geometry is built; each arm is `Intersect`ed with the map pixel rect and
  dropped when empty), and `DrawObjectMarkers(img, cells, cols, rows, cellpx)` (intersects each arm with
  `img.Bounds()`, `SetRGBA`s the survivors, no-ops on a nil image).
- `DrawObjectMarkers` carries an **unenforced documented precondition**, not a guard:
  `img.Bounds().Min` must be the origin, because the arms are in map-pixel space. Nothing checks it; an
  origin-shifted image would compare two coordinate frames and `SetRGBA`'s own bounds check would swallow
  the writes silently.
- `AnchorCell` is already generic: it takes two unsigned integers, not a record, and is the same bare
  `>>8` the spec requires for units. It needs no change and no unit-specific twin — only its doc comment
  names objects.
- `image.Rectangle` is a half-open integer rectangle. `Intersect` truncates each edge to the overlap and
  **normalises any empty result to the zero rectangle** (`image/geom.go`: `if r.Empty() { return
  Rectangle{} }`), so for an `Intersect` result `clipped.Empty()` and `clipped == image.Rectangle{}` are
  equivalent. `Empty()` is still the form to use — it is clearer and stays correct for a rectangle that
  did not come from `Intersect` — but **not** for the reason the landed comment in `overlay.go` and
  `docs/0008-structures-overlay/plan.md` DD4 give (both claim `Intersect` can return an
  empty-but-non-zero rectangle on an edge-only overlap; it cannot). That inherited inaccuracy is corrected
  in the code comment by T1 and recorded here; it changes no behaviour, because `Empty()` is what both the
  landed code and this plan call.
- `cmd/terraintool` composites (`Composite` when `-unshaded`, else `CompositeLit`), then conditionally
  calls `DrawObjectMarkers` over the finished image, then writes the PNG and prints one summary line
  ending `…, placeholder cells %d, <lightDesc><objectsDesc>`, where `objectsDesc` is the literal
  `, objects N` emitted only under `-objects`. `-scale N` makes the PNG `cellpx = CellSize*scale`, and
  `Composite`/`CompositeLit` reject `scale < 1`. The cell conversion lives in an object-specific helper
  `anchorCells(m *alm.Map) []image.Point`.
- **The only `cellpx` values a shipped invocation can produce are `32·scale` — 32, 64, 96, … — and
  `pkg/ui` always draws at `cellpx = 32`.** Any smaller `cellpx` is reachable only from a direct
  render-tier call in a test.
- `cmd/mapview` builds its summary in `load()` as `mapview: <title> WxH cells (N), tile slots L/S`, then
  appends `, objects N` there under `-objects`; `run()` afterwards appends `, ` + the water cadence
  clause. A units token must therefore be appended **inside `load()`, immediately after the objects
  token**, to sit between the object count and the cadence.
- Both cmds' `usage` constant **and** their package doc comment list the flag set, currently including
  `[-objects]`; both must gain `[-units]` or the shipped help text is wrong. `cmd/mapview`'s `load()` is
  `load(assetsFlag, graphicsFlag, mapPath string, showObjects bool)`; the units flag joins it as a fifth
  parameter **after** `showObjects`, mirroring the draw and token order.
- `color.RGBA` is a struct, so `MarkerColor` is a package `var`, not a `const`; `UnitMarkerColor` must be
  one too. Both are therefore mutable by an importer — an inherited property of the 0008 API that this
  story does not change and does not rely on.
- `pkg/ui` imports `pkg/render/terrain`, `pkg/render/camera` and `github.com/hajimehoshi/ebiten/v2`
  (+ `inpututil`, `vector`); it does **not** import `pkg/formats/alm` and the DAG forbids it to.
  `Viewer.Draw` runs the terrain tile loop, then a loop over `objectScreenRects()` filling each with
  `vector.DrawFilledRect(screen, …, terrain.MarkerColor, false)`. `objectScreenRects()` returns `nil` when
  the toggle is off or the cell list is empty; `screenRect` holds `float64` `X,Y,W,H` and the narrowing to
  `float32` happens only at the drawer call site. `camera.Camera` exposes `WorldToScreen(wx,wy)
  (sx,sy float64)`, `Zoom`, `ViewW`, `ViewH`.
- `vector.DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool)`
  paints over whatever is already in the framebuffer when `clr` is opaque; `image.RGBA.SetRGBA` is an
  unconditional store. So a pass drawn second lands on top of a pass drawn first on both paths.
- **The unit cross is a strict subset of the object cross at every scale.** With
  `r_u = max(1,⌊(4·c+16)/32⌋) ≤ r_o = max(1,⌊(6·c+16)/32⌋)` and `t_u ≤ t_o` (with the centred strips
  nested), a unit marker drawn at a cell that also holds an object covers only pixels the object marker
  already covers. Consequence: **if the two viewer passes were ordered units-then-objects, a coincident
  unit marker would be completely invisible**, not merely partly hidden — indistinguishable from "no unit
  here", which is the one thing this overlay exists to show. The draw order is therefore load-bearing and
  needs a witness that does not depend on someone looking at a window (DD6, SC-10).
- Both cmds resolve the asset root via `-assets`/`AGAINROM_ASSETS`, never hardcoded. `cmd/mapview` has a
  headless `-check` path that prints the summary and exits without a window.
- Both cmd test fixtures build a synthetic `.alm` at the EXP-0030 corrected framing whose type-6 payload
  is exactly one all-zero 70-byte record with `meta+0x24 = 1`; both must be parameterised to take a unit
  cell list, exactly as they already take an object cell list. Adding unit cells changes no existing
  assertion: no cyan is drawn without `-units`, so the existing object pixel counts are unaffected.
- **Test-tier helper collision.** `pkg/render/terrain/overlay_test.go` (package `terrain_test`) declares
  `specRound`, `specArmsRaw`, `specMapRect`, `specCross`, `sameRects`, `noPanicOverlay`, `sinkRects`,
  `markerBG`, `drawAndCheck` at package scope; `pkg/ui/overlay_test.go` (package `ui`) declares
  `nativeArms`, `wantScreenRects`, `assertScreenRects`, `rectInView`, `overlayViewer`. New test files live
  in the same packages and may not redeclare any of them. Reusable verbatim: `specRound`, `specMapRect`,
  `sameRects`, `noPanicOverlay`, `sinkRects`, `markerBG`, `assertScreenRects`, `rectInView`,
  `overlayViewer`. Hardwired to the object constants and therefore needing unit-named twins:
  `specArmsRaw` → `specUnitArmsRaw`, `specCross` → `specUnitCross`, `drawAndCheck` → `drawUnitsAndCheck`,
  `nativeArms` → `nativeUnitArms`, `wantScreenRects` → `wantUnitScreenRects`.

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `pkg/render/terrain/overlay.go` | MODIFY | Extract the unexported arm/clip core `markerRects(col,row,cols,rows,cellpx,radius,thickness int)` and rasterizer `drawMarkers(img, cells, cols, rows, cellpx, radius, thickness int, c color.RGBA)`; re-express `ObjectMarkerRects`/`DrawObjectMarkers` as calls into them at `(6,3)`/`MarkerColor`, signatures and behaviour unchanged; add `UnitMarkerColor`, `unitArmRadius = 4`, `unitArmThickness = 1`, `UnitMarkerRects`, `DrawUnitMarkers` (DD2, DD4, DD5). Widen `AnchorCell`'s doc comment to name units as well as objects (DD3). Correct the `Intersect` comment (baseline). Stdlib only. |
| `pkg/render/terrain/unit_overlay_test.go` | ADD | Synthetic unit geometry + rasterizer tests (AC-1, AC-2, AC-3, P-1, P-2). Separate file so 0008's `overlay_test.go` stays untouched as the characterization pin. |
| `pkg/ui/overlay.go` | MODIFY | Extract the unexported method `(v *Viewer) overlayScreenRects(show bool, cells []image.Point, rects func(col,row,cols,rows,cellpx int) []image.Rectangle) []screenRect`; re-express `objectScreenRects` as a call into it; add `SetUnits`, `unitScreenRects`, and `overlayPasses() []overlayPass` — the ordered pass list `Draw` consumes (DD6). |
| `pkg/ui/unit_overlay_test.go` | ADD | Synthetic transform/cull tests for the unit path, the four toggle combinations, and the pass-order witness (FR-4, FR-6 viewer path). |
| `pkg/ui/viewer.go` | MODIFY | `unitCells []image.Point` / `showUnits bool` state; `Draw` iterates `overlayPasses()` instead of the inline object loop, so the draw order is data the tests can read (DD6). |
| `cmd/terraintool/main.go` | MODIFY | Add `-units` to the flag set, the `usage` constant and the package doc comment; after the object block, if set, build cells from `m.Units` via `unitAnchorCells` and call `DrawUnitMarkers`; append `, units N` after the objects token (DD7). Disabled path unchanged (P-3). |
| `cmd/terraintool/main_test.go` | MODIFY | Parameterise the fixture with unit anchor cells; assert the AC-1/AC-4/AC-5 behaviour at `-scale 1` and `-scale 2`, the zero-unit map, both CLI flag orders, and the byte-identity oracles (SC-5). |
| `cmd/mapview/main.go` | MODIFY | Add `-units` to the flag set, `usage` and the package doc comment; pass into `load()`, which builds the cells, calls `viewer.SetUnits(true, cells)` and appends `, units N` right after the objects token (DD7, DD8). Disabled path unchanged. |
| `cmd/mapview/main_test.go` | MODIFY | Parameterise the fixture with unit anchor cells; assert the token, its position, both flag orders, the zero-unit map, and the pre-0009 shape without the flag (SC-7). |

No `internal/archtest` or `docs/ARCHITECTURE.md` edit (no new package). No `pkg/formats` edit. The
compositors (`composite.go`, `lit.go`) are not touched — the overlay draws over their output.
**0008's `pkg/render/terrain/overlay_test.go` and `pkg/ui/overlay_test.go` are not touched by any task.**

## Design decisions

- **DD1 — Both overlay passes draw over finished output; the compositors are untouched, and so is the
  object pass.** The PNG unit pass runs after `DrawObjectMarkers`, and the viewer unit pass after the
  object pass, so the units overlay composes with unshaded terrain, shaded terrain, water and the object
  overlay without modifying any of them. With the unit overlay off, no unit overlay code runs, giving the
  byte-identical baseline P-3 demands — including the objects-enabled baseline, because the object pass is
  reached identically either way.
  *Rejected:* a single combined "overlays" pass taking both cell lists — it would couple two independent
  toggles into one call site and make "objects on, units off" a code path rather than an absence of one,
  which is exactly what AC-4 tests.

- **DD2 — One shared arm/clip core; two named entry points; 0008's exported surface unchanged.** The arm
  arithmetic FR-6 specifies for units is the arithmetic 0008 already ships at different constants: same
  centre, same `round(n·cellpx/32)` scaling, same `[c-r, c+r+1)` arm, same `[c-⌊t/2⌋, c-⌊t/2⌋+t)` strip,
  same off-map-first rule, same map-rect clip. It is therefore written **once**, as unexported
  `markerRects(col, row, cols, rows, cellpx, radius, thickness int) []image.Rectangle` and
  `drawMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx, radius, thickness int,
  c color.RGBA)`. `ObjectMarkerRects`, `DrawObjectMarkers` and `MarkerColor` keep their exact exported
  signatures, values and observable behaviour and become thin calls at `(6, 3)`; `UnitMarkerRects`,
  `DrawUnitMarkers` and `UnitMarkerColor` are their `(4, 1)` siblings. The radius and thickness stay
  **package-internal constants**, never caller-supplied, so no caller can produce a glyph FR-6 does not
  define.
  *Rejected:* duplicating the arm arithmetic into unit-named production copies — the half-open extents and
  the round-to-nearest scaling are precisely where a copy-and-tweak drift bug hides, and the centring term
  must reproduce identically for units rather than by coincidence.
  *Rejected:* exporting a generic `MarkerRects(…, radius, thickness)` so `pkg/ui` could parameterise it —
  it would move the FR-6 constants outside the package that owns them and let a caller draw a glyph the
  spec does not define.
  *Rejected:* renaming `MarkerColor` to `ObjectMarkerColor` for symmetry — it is exported and referenced
  from `cmd/terraintool`'s tests, so the rename is a shipped-API break bought for nothing.
  **The test tier deliberately does the opposite, and that is not an inconsistency.** 0008's test helpers
  are hardwired to the object constants and may not be edited (the pin), so the unit tests re-derive the
  FR-6 arithmetic under unit-named helpers. In the production tier a second copy is a defect source; in
  the test tier a second **independent transcription of the spec** is the whole point of a separate-context
  oracle — a shared helper between the two tiers would make the test agree with the code by construction.

- **DD3 — `AnchorCell` is reused unchanged; the `/256 → cell` shift stays in exactly one place.** The
  spec's `(X>>8, Y>>8)` for units is the same conversion `AnchorCell` already performs for objects, on the
  same `uint32` inputs. It gets a doc comment naming both callers and no code change.
  **One tier up, the answer is the opposite and is decided here:** `cmd/terraintool`'s `anchorCells(m
  *alm.Map)` walks `m.Objects`, and `alm.Object` and `alm.Unit` are distinct types, so units get a
  **twin** `unitAnchorCells(m *alm.Map)` rather than a generalisation. Two four-line loops over two
  different struct types are cheaper and clearer than a type parameter plus an accessor constraint, and
  the shared arithmetic they both call is `AnchorCell` itself, which is where DD3's single-home rule
  actually bites.
  *Rejected:* a `UnitAnchorCell` twin in the render tier — it would duplicate a one-line shift;
  *rejected:* passing `alm.Unit`/`alm.Map` into the render tier — a DAG violation, and forbidden by the
  spec's constraints.

- **DD4 — The unit cross is `radius = 4`, `thickness = 1` at native scale, scaled by the shared
  `scaleDim`.** For anchor cell `(col,row)` at `cellpx`: centre `c = (col*cellpx + ⌊cellpx/2⌋, row*cellpx +
  ⌊cellpx/2⌋)`, `r = scaleDim(4, cellpx) = max(1, ⌊(4·cellpx+16)/32⌋)`, `t = scaleDim(1, cellpx) =
  max(1, ⌊(cellpx+16)/32⌋)`. Horizontal arm `[cx-r, cx+r+1) × [cy-⌊t/2⌋, cy-⌊t/2⌋+t)`, vertical arm
  `[cx-⌊t/2⌋, cx-⌊t/2⌋+t) × [cy-r, cy+r+1)`. At `cellpx = 32` this is `r=4, t=1`, i.e. exactly
  `[cx-4,cx+5) × [cy,cy+1)` and `[cx,cx+1) × [cy-4,cy+5)` around `(32·ax+16, 32·ay+16)` — the FR-6 native
  instance. `cellpx < 1` yields `nil`; an off-map anchor yields `nil` **before any geometry is built**,
  which is what makes FR-3's extreme-magnitude clause hold without any wrap check: whatever `cols`/`rows`
  are, an anchor with `col >= cols` is rejected at the guard before any multiplication, so the property is
  **structural** rather than a consequence of a bound on the map size. It is *not* justified by
  "`AnchorCell`'s widest output `0xFFFFFF` always exceeds the map extent" — `Map.Width`/`Height` come from
  untrusted `u32` reads, bounded only indirectly by the grid payload-length checks, so a map declaring a
  huge extent is representable; SC-3's **in-map** extreme case is what covers that residue.
  All arithmetic is integer, into a fixed-capacity-2 slice, so nothing allocates with
  coordinate magnitude (P-2), and Go wraps rather than panics on overflow regardless, so AC-3's "no panic"
  holds unconditionally.
  **Where the centring term is observable — the reason SC-2's scale list is what it is.** `t ≥ 2` requires
  `cellpx ≥ 48`, so at the native 32 and at every smaller scale `⌊t/2⌋ = 0` and FR-6's centred strip
  degenerates to `[c, c+t)`. An implementation that omitted the `-⌊t/2⌋` term entirely would be
  indistinguishable from a correct one at `cellpx ≤ 47`. SC-2 therefore **must** include a `cellpx ≥ 48`
  case, and it pins 48 (the boundary), 64 (`-scale 2`, the first non-native scale a shipped invocation can
  actually emit) and 96 (`t = 3`, odd again, proving the term is not simply "always 1 at large scales").
  **Where the map-rect clip is observable.** An arm fits strictly inside its own cell when
  `r ≤ ⌈cellpx/2⌉ − 1`; with `r = max(1, ⌊(cellpx+4)/8⌋)` that holds from `cellpx = 3` upward, and fails
  only at `cellpx ∈ {1, 2}`. So for an interior anchor the map-rect intersection truncates nothing at any
  realistic scale; it is observable only at `cellpx ≤ 2`, or at an edge/corner anchor **at those same
  small scales**. SC-2 states the clip cases as one case, not two, for that reason.
  *Rejected:* a symmetric `[c-r, c+r)` arm — it would give an 8-pixel arm at native scale instead of the
  9 FR-6 pins, and fail AC-2.

- **DD5 — `DrawUnitMarkers` fills each map-clipped arm intersected with the image bounds in
  `UnitMarkerColor`, opaque, after the object pass.** Same two-stage clip as the object rasterizer (FR-3
  stage 1 in `UnitMarkerRects`, stage 2 against `img.Bounds()`), same empty skip, same `SetRGBA` store.
  `UnitMarkerColor = color.RGBA{0x00, 0xE5, 0xFF, 0xff}` (FR-6). It inherits the shared core's **unenforced
  documented precondition** that `img.Bounds().Min` is the origin; every production caller satisfies it
  (`Composite`/`CompositeLit` allocate `image.Rect(0,0,w,h)`), and SC-4's undersized case keeps the origin
  and shrinks only `Max` so the criterion cannot pass vacuously. A nil image and an empty cell list are
  no-ops, which is FR-2's "empty target rectangle yields no geometry, never a panic" on the draw side.
  Because the store is unconditional and opaque, a unit arm overlapping an object arm **replaces** those
  pixels — the FR-4 stacking requirement — and two coincident units write identical pixels twice, so the
  visible result is one marker (FR-3).
  *Rejected:* alpha-blending — FR-6 says opaque, and a blend would make a stacked unit marker's colour
  depend on whether an object happened to sit under it, defeating the diagnostic.

- **DD6 — The viewer's overlay draw order is data, not statement order.** `(v *Viewer) overlayScreenRects(
  show bool, cells []image.Point, rects func(col,row,cols,rows,cellpx int) []image.Rectangle) []screenRect`
  — a **method**, since it reads `v.cam` and `v.grid` — holds the whole transform: nil when `!show` or no
  cells; otherwise per cell, per native (`cellpx = terrain.CellSize`) map-clipped arm, top-left
  `cam.WorldToScreen(minX, minY)` and size `(dx, dy) × cam.Zoom`, dropping rects fully outside
  `[0,ViewW) × [0,ViewH)`. `objectScreenRects()` and `unitScreenRects()` are one-line calls passing
  `terrain.ObjectMarkerRects` / `terrain.UnitMarkerRects`; passing the generator as a function value is
  what lets the transform be shared without exporting the FR-6 constants (DD2).
  `overlayPasses() []overlayPass` then returns the passes **in draw order** — objects first, units second,
  each `overlayPass{Color color.RGBA; Rects []screenRect}`, omitting a pass with no rects — and `Draw`
  simply iterates it, filling each rect with `vector.DrawFilledRect(screen, …, pass.Color, false)`.
  **This exists to make FR-4's order testable.** The baseline records that the unit cross is a strict
  subset of the object cross, so a reversed order does not degrade a marker, it erases it; leaving that
  property to the order of two loops in `Draw` would leave the story's only guard against total marker
  loss to a manual observation the story is not able to guarantee (DD9). With `overlayPasses`, SC-10
  asserts the order directly, with no window.
  `screenRect` keeps its `float64` fields and the narrowing to `float32` stays at the drawer call site, so
  SC-6's oracle can be the camera contract itself with no rounding step between them. Reading `v.cam.Zoom`
  (the exported field the tile loop reads) rather than the private clamped zoom is deliberate and
  unchanged from 0008: the two transforms then agree in every camera state, including an unclamped one a
  caller created by assigning `Zoom` directly. FR-3's output-rect stage on this path is the framebuffer
  clip, exactly as FR-3 prescribes — clipping the screen rect to integer view bounds would need the
  independent pixel snapping FR-6 forbids.
  *Rejected:* keeping two inline loops in `Draw` — it is one line shorter and leaves FR-4's viewer half
  with no automated witness at all; *rejected:* one loop over a merged rect list — it would lose the order
  FR-4 fixes and make the two toggles interdependent; *rejected:* pre-rasterizing markers into the
  `(slot,sub)` tile cache — the cache key carries no position; *rejected:* running `DrawUnitMarkers` on a
  CPU image and uploading it — it bypasses the camera transform FR-6 requires and would not track pan/zoom.

- **DD7 — Wiring is an opt-in `-units` flag in each cmd, off by default, its token after the objects
  token.** `terraintool -units`: after the object block, build `cells := unitAnchorCells(m)`, call
  `DrawUnitMarkers(render.Image, cells, m.Width, m.Height, CellSize*scale)`, and append the literal
  `, units N` (`N = len(m.Units)`) after `objectsDesc`. `mapview -units`: the flag is passed into
  `load()`, which builds the same cells, calls `viewer.SetUnits(true, cells)` and appends `, units N`
  immediately after the objects token — inside `load()`, so it lands before the `, water speed …` clause
  `run()` adds and after the object count. Each token is emitted **solely** on its own flag, never on the
  count: a map with `len(m.Units) == 0` under `-units` still prints `units 0`, because FR-5 asks for the
  decoded count to be reported, not for markers to exist. The composed order is fixed by the code, not by
  the CLI, so `-units -objects` and `-objects -units` produce the identical summary. All four on/off
  combinations are reachable, and the units-off summary is character-for-character its pre-0009 shape
  (P-3, AC-4). `N` is the decoded total `len(m.Units)` — off-map anchors are counted even though they are
  marked nowhere, because FR-5 asks for the decoded count, not the drawn one. Both cmds build
  `[]image.Point` with stdlib `image` only, and both add `-units` to their `usage` string and package doc
  comment so the shipped help text matches the flag set. The flag's help text is fixed here, mirroring the
  object flag's: `"overlay a diagnostic marker on each placed unit's anchor cell and report the unit
  count"`.
  *Rejected:* one `-overlays` flag taking a list — it makes independence a parsing question rather than a
  structural one; *rejected:* on-by-default — the spec requires off-by-default with a byte-identical
  baseline; *rejected:* suppressing the token when the count is zero — it would violate FR-5 and make
  "unit-free map" and "overlay not requested" indistinguishable in the output.

- **DD8 — The viewer receives already-shifted `[]image.Point` cells from `cmd/mapview`, not `alm` types.**
  `pkg/ui` must not import `pkg/formats/alm` (the DAG forbids it). `cmd/mapview` does the
  `alm.Map.Units → []image.Point` wiring once at load and hands the slice to the viewer via `SetUnits`,
  exactly as it already does for objects.
  *Rejected:* importing `alm` into `pkg/ui` — a DAG violation and unnecessary.

- **DD9 — AC-6 splits into three parts with different evidentiary status, and only one of them may be
  deferred.** The spec's AC-6 bundles four claims: markers on the right cells, the count matching, unit
  markers above coincident object markers, and alignment holding through pan/zoom. They are not equally
  obtainable, and the plan says so up front rather than discovering it at verification:
  *(a) the placement and count half* is scriptable headless against a lawful install and **is run** (SC-9);
  *(b) the draw-order half* has **no observable instance on any shipped map** — no map in the corpus places
  a unit and an object on the same cell — so it cannot be verified by looking at a real render at all; it
  is carried by the synthetic SC-4 (PNG) and SC-10 (viewer), and AC-6's stacking clause is recorded as
  *not exercisable on real data*, not as passed;
  *(c) the live pan/zoom half* needs a human at a window. It is recorded as an explicitly declared
  **pending limitation** if no such run happens (SC-11), and the conclusion must not rest on it. Because
  (b) is closed synthetically and (c) is the only deferrable part, deferring it no longer leaves the
  viewer overlay with zero automated verification — which is what made the deferral cheap-looking and
  dangerous.
  *Rejected:* faking a window in a unit test — it would assert nothing about real alignment and
  misrepresent manual evidence as automated; *rejected:* treating AC-6 as one pass/fail unit — it would
  let the obtainable evidence be blocked by the unobtainable, or the unobtainable be waved through with
  the obtainable.

## Success criteria

Each maps to a named test (unit) or a developer-run procedure (manual).

1. **SC-1 (FR-1, FR-2, FR-3, AC-1, P-1)** — over a synthetic fractional `/256` cell list, `AnchorCell`
   shifts by 8 and each in-map unit yields a cross at `(X>>8, Y>>8)`; the fractional low byte changes no
   output; two units on one cell yield **identical** geometry; each off-map anchor yields `nil`; and no
   call ever returns more than 2 rectangles. *Test:* `TestUnitAnchorCellAndOffMap`.
2. **SC-2 (FR-6, FR-3, AC-2)** — `UnitMarkerRects` returns exactly the FR-6 rectangles at
   `cellpx ∈ {32, 15, 16, 17, 48, 64, 96}` — the native scale, three downscales, and **three scales at
   which the centring term is non-degenerate** (`48` and `64` give `t = 2, ⌊t/2⌋ = 1`; `96` gives
   `t = 3`), so an implementation omitting FR-6's `-⌊t/2⌋` term fails. `64` is `-scale 2`, the first
   non-native `cellpx` a shipped invocation can emit. Plus the clip case: on a 1×1 map at `cellpx ∈ {1, 2}`,
   where the arms overrun the map on every side, each arm is truncated by `min`/`max` only, with no
   synthetic border — and a corner anchor at `cellpx = 32` correspondingly truncates **nothing** (the
   cross fits inside its cell from `cellpx = 3` up), which the test asserts rather than assumes.
   *Test:* `TestUnitMarkerRectsGeometry`.
3. **SC-3 (FR-2, FR-3, AC-3, P-1, P-2)** — `cellpx < 1` yields `nil`; an anchor at the widest value
   `AnchorCell` can produce from a `uint32` maximum (`0xFFFFFF`) against a small map yields `nil` with no
   panic, zero allocations and no wrap into a visible marker; an in-map extreme-magnitude anchor yields
   ≤ 2 arms with no panic and no magnitude-proportional allocation. *Test:* `TestUnitMarkerRectsExtremes`.
4. **SC-4 (FR-1, FR-3, FR-4, FR-6, AC-2, AC-3, P-1)** — `DrawUnitMarkers` fills exactly the clipped arm
   pixels of each in-map cell in `UnitMarkerColor` and leaves every other pixel untouched; off-map cells
   draw nothing; a nil image and an empty target rectangle are no-ops rather than panics; with an `img`
   smaller than the map extent but still origin-anchored the arms truncate to the image bounds with no
   synthetic border and no out-of-bounds write; and — the **FR-4 stacking witness, which no real map can
   provide** — drawing `DrawObjectMarkers` then `DrawUnitMarkers` at the same cell leaves
   `UnitMarkerColor` on every pixel of the unit cross and `MarkerColor` on the object pixels the unit
   cross does not cover, while the reverse order leaves no `UnitMarkerColor` pixel at all.
   *Test:* `TestDrawUnitMarkers`.
5. **SC-5 (FR-1, FR-4, FR-5, AC-4, AC-5, P-3)** — `terraintool -units` marks every **in-map** unit anchor
   cell cyan (an off-map anchor nowhere) and appends `units N` with `N = len(m.Units)`, at `-scale 1`
   **and** at `-scale 2` (`cellpx = 64`, where the marker thickness is even); `-objects -units` reports
   the object count **then** the unit count and puts cyan over yellow at a cell holding both; the summary
   is identical under `-units -objects` and `-objects -units`; a fixture with **zero** units still prints
   `units 0` under `-units`; and **without** `-units` the PNG is byte-identical to an independently
   constructed oracle in both object states — `CompositeLit(...)` alone, and `CompositeLit(...)` +
   `DrawObjectMarkers(...)` — rather than to a second no-flag run.
   *Independence, stated honestly:* that oracle is independent of the **unit** code, which is what AC-4
   asks. It is **not** independent of the DD2 refactor, since the objects-on arm calls the same
   `DrawObjectMarkers` the subject does; a DD2 regression is caught by SC-8, not here.
   *Test:* `TestRenderUnitsOverlay` (+ every existing `cmd/terraintool` test stays green unchanged).
6. **SC-6 (FR-1, FR-3, FR-4, FR-6, AC-4 viewer)** — at more than one pan/zoom position `unitScreenRects`
   equals, for each in-map unit, the native arm transformed by the camera. **Both halves of the oracle are
   independent:** the arm is transcribed from FR-6 in the test (`nativeUnitArms`, never read from
   `terrain.UnitMarkerRects`), and the placement is `cam.WorldToScreen(corner)` + a `cam.Zoom`-scaled size
   in `float64` derived from the camera contract, never a second call to the function under test. Rects
   fully outside the view are culled. All **four** toggle combinations are pinned — (off,off), (objects
   on, units off), (units on, objects off), (both on) — and with the unit overlay off, or with no unit
   cells, `unitScreenRects` returns `nil` (the AC-4 viewer witness). *Test:* `TestUnitScreenRects`.
7. **SC-7 (FR-1, FR-4, FR-5, AC-4, AC-5)** — `mapview -units` reports `units N` (`N = len(m.Units)`,
   off-map anchors included) on the `-check` summary in the DD7 position; with both flags the objects
   token precedes the units token and both precede the cadence clause, identically under
   `-units -objects` and `-objects -units`; each flag alone emits only its own token; a zero-unit fixture
   still prints `units 0`; and with neither flag the summary is character-for-character its pre-0009
   shape. *Test:* `TestUnitsFlag` (+ the existing `TestObjectsFlag` stays green unchanged).
8. **SC-8 (FR-2, DAG, and the DD2 characterization pin)** — `pkg/render/terrain` still imports only
   stdlib and `pkg/ui` adds no new dependency, so the fail-closed `internal/archtest` check stays green;
   **and** every 0008 overlay test passes **unmodified** after the DD2 refactor. That second half is the
   only thing standing between the refactor and a silent regression in shipped object geometry, and it is
   also what SC-5's objects-on oracle rests on. *Test:* the existing `internal/archtest` live-tree check
   + the unmodified `pkg/render/terrain/overlay_test.go` and `pkg/ui/overlay_test.go`.
9. **SC-9 (FR-1, FR-5, FR-6, AC-6 placement + count)** — a developer run renders a **named** real GOG map
   with placed units through `terraintool -objects -units` at `-scale 1` and inspects the result: unit
   markers sit on the units' terrain cells, and the reported `units N` is recorded. *Method:*
   developer-run, headless, **required** — this half has no "or a limitation" branch. *Evidentiary limit,
   stated now rather than discovered later:* the reported count cannot be corroborated by anything that
   goes through our own reader. `alm.Open` rejects any file whose type-6 payload is not exactly
   `70 × Count6`, so every derivation available to us — `len(Map.Units)`, `Count6`, `payloadSize / 70`,
   and everything `almtool` prints — is the same number by construction, and their agreement proves only
   that the file decoded. The count half is therefore recorded as **evidence of a successful decode, not
   of a correct count**; the only external oracle is the Map Editor's or the game's own unit count for a
   named map, and if that is not obtained, SC-9 says so. The **placement** half carries the diagnostic
   value and is what this criterion actually tests.
10. **SC-10 (FR-4 viewer draw order)** — with both overlays enabled and cells for each,
    `overlayPasses()` returns exactly two passes, `[0]` carrying `terrain.MarkerColor` and `[1]` carrying
    `terrain.UnitMarkerColor`, in that order; with only one overlay enabled it returns that one pass; with
    neither it returns none. Since `Draw` does nothing but iterate that slice, this pins the terrain →
    objects → units order without a window. *Test:* `TestOverlayPassOrder`.
11. **SC-11 (FR-6, AC-6 live half)** — a developer run opens a real GOG map in `mapview -objects -units`
    and confirms unit markers hold their terrain cells through pan and zoom. *Method:* developer-run at a
    window. If it is not performed, `verification.md` records it as an explicit **pending limitation**
    naming what was not observed, AC-6 is **not** marked passed, and no conclusion rests on it. This is
    the story's one deferrable criterion; SC-10 and SC-6 hold the viewer's order and transform
    automatically in the meantime.

## Risks (product)

- **R-1 — the viewer overlay is not pixel-identical to the PNG overlay.** FR-6 routes the viewer's markers
  through the float camera transform (as terrain is) while the PNG uses integer `cellpx` math, so at a
  given zoom a marker can differ from the PNG by up to a pixel at a boundary. This is intended: a marker
  must track its terrain cell, not a fixed pixel grid. *Mitigation:* SC-6 asserts the viewer transform
  matches the terrain-tile transform (the property that matters); AC-6 checks alignment with terrain
  cells, never PNG-equality. Inherited unchanged from the object overlay.
- **R-3 — the DD2 refactor could silently change the shipped object overlay.** Restructuring shipped,
  tested geometry risks an off-by-one no unit-overlay test would catch. *Mitigation:* 0008's test files
  are the characterization pin and are **not modified** by any task (SC-8); they exercise the exact FR-6
  object rectangles at odd/even `cellpx`, the `cellpx ≤ 2` truncation, the extremes, the raster and the
  allocation profile, so a one-pixel drift in the shared core fails them. The refactor also lands in its
  own commit ahead of any unit-specific wiring, so a bisect separates "the refactor broke it" from "the
  units broke it".
- **R-4 — a stacked unit marker does not merely sit on top of the object marker, it sits *inside* it.**
  The unit cross is a strict subset of the object cross at every scale, so at a cell holding both, a
  correct render is a cyan cross nested in a yellow one — and an **incorrect** render (the passes ordered
  units-then-objects) shows no cyan at all rather than something visibly wrong. The failure is silent and
  looks exactly like "no unit here". *Mitigation:* SC-10 pins the pass order automatically and SC-4 pins
  the pixel outcome in both orders, so the property does not depend on anyone looking at a window; AC-6's
  observer is additionally told to expect a nested pair rather than a replaced one.
- **R-5 — a unit count of zero is representable and must still report.** Unlike the type-4 objects (where
  `#type4 = 0` cannot be represented at all), an empty type-6 payload decodes to an empty slice, so
  `-units` on a unit-free map draws nothing and must print `units 0`. The tempting implementation —
  guarding the token on the count as well as the flag — violates FR-5 and makes "unit-free map"
  indistinguishable from "overlay not requested". *Mitigation:* SC-5 and SC-7 each carry a zero-unit
  fixture; DD7 states the token is emitted solely on the flag. For manual runs, the developer selects a
  map whose summary reports a non-zero `units N` before drawing any conclusion from what is on screen.

- **R-6 — the unit marker's arm thickness turns even at the scales users actually render at.** For
  objects, `t` first becomes even at `cellpx = 17`, a scale `terraintool` cannot emit (it renders at
  `32·scale`), so 0008's equivalent bias was a test-tier curiosity. For units the native thickness is 1,
  which moves the boundary to `cellpx = 48`: `-scale 2` (`cellpx = 64`) gives `t = 2`, and FR-6's centred
  strip `[c-⌊t/2⌋, c-⌊t/2⌋+t)` then places the arm one pixel toward the low side of the centre pixel. So
  the **only** production scales at which the centring term does anything are the non-native ones, and a
  marker at `-scale 2` is deliberately not centre-symmetric. This is the exact, intended FR-6 contract,
  not a defect — but it is asymmetric with the sibling story in a way that makes "it worked for objects"
  a misleading guide. *Mitigation:* SC-2 pins the exact rectangles at `cellpx` 48, 64 and 96, so the term
  is observed rather than assumed; SC-5 renders `terraintool -units` at `-scale 2` so the reachable case
  is exercised end to end; DD4 records where the boundary sits and why the scale list is what it is.

*(R-2 was a statement about how much AC-6's count comparison actually proves. That is a property of the
verification procedure, not of the product, so it is not carried here as a product risk; it is stated
where it applies, in SC-9.)*
