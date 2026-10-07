# Tasks — height-projected placements

Legend: **files** the task may change · **done when** the observable it must leave behind.

## T1 — `AnchorHeight`

**files** `pkg/render/terrain/project.go`, a new `pkg/render/terrain/anchor_height_test.go`

Add `Projection.AnchorHeight(col, row int) int` per the height lookup contract: clamp `col`/`row`
into the cell range with `Altitude`'s own min-then-max order before forming `col+1`/`row+1`, read the
four corners through `Altitude`, sum and `/4` (DD-1).

**done when** SC-1, SC-2 and SC-10 pass: the clamp-order break (adding 1 before clamping, at
`math.MaxInt`) fails a test, the borrowed slice is unchanged after many calls, and every assertion
runs with no window open.

## T2 — the terrain-tier per-marker lift

**files** `pkg/render/terrain/overlay.go`, a new `pkg/render/terrain/marker_lift_test.go`

Add `DrawObjectMarkersAtHeights`/`DrawUnitMarkersAtHeights`, each taking a `liftY func(col, row int)
int` alongside its existing `*At` parameters. Thread it through `markerRects` (a new `liftY int`
parameter added only to `cy`, never to `mapRect`) and `drawMarkers` (a new `liftAt` callback,
nil-safe, read once per cell) (DD-2). The four existing exported functions keep their signatures and
call the shared helpers with a zero lift.

**done when** SC-7 (the stage-1/stage-2 split) and SC-11 (culling after the offset) pass, and SC-4
still holds — `overlay_test.go` is **not** edited, and it still passes: `ObjectMarkerRects`/
`UnitMarkerRects`/`DrawObjectMarkersAt`/`DrawUnitMarkersAt` are byte-identical at a zero lift.

## T3 — `cmd/terraintool` wiring

**files** `cmd/terraintool/main.go`, a new `cmd/terraintool/lift_test.go`,
`cmd/terraintool/main_test.go` (one test, named below). Needs T1 and T2 landed — it calls both.

When the render is not `-flat` and either `-objects` or `-units` is set, build
`proj := terrain.Project(m.Altitudes, m.Width, m.Height)` (DD-4) and call
`DrawObjectMarkersAtHeights`/`DrawUnitMarkersAtHeights` with `liftY = func(col, row int) int { return
-proj.AnchorHeight(col, row) * *scale }` in place of the plain `*At` calls; `-flat` keeps calling the
existing `*At` functions untouched, so `proj` is never built on that path.

`TestRenderMarkersAtProjectedOrigin` in `main_test.go` asserts the superseded rule — 0012 FR-11's
un-displaced lattice — on a sloped fixture whose object anchor `(0,0)` now lifts by 10 output px per
scale. Update its expected coordinates to carry `AnchorHeight`; it is the second and last existing
test file this story may touch (SC-4).

**done when** SC-7, SC-8 and SC-9 pass: the AC-7 top-edge case clips at the unlifted extent, its
mirror (a bottom-edge anchor, negative lift) is run too, `-flat` output is byte-identical at several
scales, and the summary line gains no token; draw order (objects then units) is unchanged (FR-5,
DD-6).

## T4 — `Mode()` supersedes the overlay rule

**files** `pkg/ui/viewer.go`, `pkg/ui/mode_test.go`, `pkg/ui/light_test.go` (three tests, below)

Drop `Mode()`'s `!v.showObjects && !v.showUnits` clause (DD-5) and add the deliberate one: `Mode()`
becomes `v.proj != nil && !v.flat`, with a `flat` field and `SetFlat(bool)` (DD-7). Rewrite
`TestModeFollowsOverlaysEnabledAfterConstruction` in place to assert the supersession — toggling
either or both overlays, in any order, over a valid altitude grid leaves `Mode()` Displaced and
`WorldH()` unchanged — and add SC-13's flat-and-still-lit case.

`light_test.go` reaches flat mode in three places through the deleted rule: two use
`SetObjects(true, nil) // force flat mode` as a fixture device over a lit grid, one encodes the rule
in an expectation table. Move them to `SetFlat(true)`; change no expectation about geometry or
lighting. It is the third and last existing test file this story may touch — `mode_test.go` is yours,
`cmd/terraintool/main_test.go` was T3's.

**done when** SC-3, SC-4 and SC-13 pass: all four overlay combinations hold Displaced, `SetFlat(true)`
holds Flat **and** `Lit()`, and no fourth existing test file is edited.

## T5 — the marker's own height, in `pkg/ui`'s world space

**files** `pkg/ui/overlay.go`, a new `pkg/ui/marker_height_test.go`. Needs T1, and **must land after
T4**: `v.proj != nil` can be true while a pre-T4 `Mode()` still forces flat, and that interim state
draws lifted markers over flat terrain — visibly wrong, and invisible to either task's own tests.

In `overlayScreenRects`, when `v.Mode() == ModeDisplaced` — the mode, **not** `v.proj != nil`, which
DD-7 separated from it — shift each arm `rects` returns by `arm.Add(image.Pt(0, dy))`,
`dy = -v.proj.AnchorHeight(cell.X, cell.Y) - v.proj.MinV`, before `WorldToScreen` (DD-3); flat mode
takes the unshifted arm exactly as before, which is why `overlay_test.go`/`unit_overlay_test.go` —
built over grids with no altitude layer — need no edit at all.

**done when** SC-5, SC-6 and SC-11 pass: an object and a unit anchored in the same sloped cell carry
the identical offset, draw order survives a newly-created overlap, and the view cull runs after the
offset in both directions. One assertion also reads `v.Mode()` where the lift is applied, so landing
this before T4 fails loudly instead of shipping lifted markers over flat ground.

## T6 — `mapview -flat`

**files** `cmd/mapview/main.go`, a new `cmd/mapview/flat_test.go`. Needs T4 landed — it calls
`SetFlat`.

Add `-flat` and wire it to `SetFlat` (FR-8, DD-7). Its help line reads as the diagnostic it is,
alongside `-unshaded`. The summary text gains nothing: no token, no reordering, no dependence on the
mode (FR-3). `cmd/againrom` gains no flag.

**done when** SC-13's command half passes: `-flat` over a valid altitude grid renders flat with the
map still lit, `-check` output is byte-identical with and without it, and `cmd/againrom` has no such
flag. `cmd/mapview/main_test.go` is **not** edited — SC-4's budget is spent.

## T7 — `-flat` in the tool's own documented flag surface

**files** `cmd/mapview/main.go` (the `usage` const and the package doc's `Usage:` bracket line only),
`cmd/mapview/flagset_test.go`. Needs T6 landed.

T6 added the flag and left `usage` alone, because `flagset_test.go` pins it byte-for-byte and a fourth
existing-test-file edit had not been declared. It is now (SC-4). Not cosmetic: `fs.SetOutput(io.Discard)`
means `-h` prints nothing, so that string is the only place a user reads the flag set — a flag absent
from it is undiscoverable, not merely inconsistent.

Add `[-flat]` to `usage` after `[-unshaded]`, the identical token to `shippedUsage`, and `{"flat", ""}`
to `shippedFlags`. Change nothing else in either file: no test renamed, no assertion added or removed,
no flag reordered.

The pin's own hole — a hardcoded list sees a removal but never an addition — is **not** yours to close:
walking the parser instead needs `run()` to expose its `FlagSet`, which belongs to its own story.

**done when** SC-4 holds at four files, `go test ./cmd/mapview/` is green, and two breaks are run and
reported: dropping `[-flat]` from `usage` alone must turn `TestFlagSet`'s usage-line pin red, and
dropping the `shippedFlags` entry alone must leave everything green — that second result is the hole,
and measuring it is the point.

## T8 — the two properties nothing measures yet

**files** `pkg/render/terrain/anchor_height_test.go`, `pkg/render/terrain/marker_lift_test.go`,
`pkg/ui/marker_height_test.go` — all three created by this story, so SC-4's four *existing* files is
unaffected; don't stop on it. Nothing predating `449538b` may change.

Two spec properties are asserted nowhere: the recipe implies both, and implication is what this
story spent five tasks refusing to take for a measurement.

**P-2** (bound) `AnchorHeight` lies between the min and max of its cell's four corner altitudes. Over
the fixtures and probes the recipe tests sweep, read the corners independently through `Altitude` and
assert `min ≤ AnchorHeight ≤ max`. A **negative** fixture is required: truncation toward zero, not
floor, is where a `/4` bound breaks.

**P-6** (drawing half) Construction and the lookup each have a test; **drawing** none. Byte-compare
the borrowed slice across a displaced draw — `Draw{Object,Unit}MarkersAtHeights` and `pkg/ui`'s
`overlayScreenRects`.

**done when** `go test ./...` is green, SC-4 holds at four, and three breaks are run and
reported: returning the unnormalised corner **sum** must fail P-2's test; returning `max+1` must fail
it too (a sum passes trivially on an all-zero fixture, so this proves the fixtures discriminate); and
a draw path writing one byte into the slice must fail P-6's.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-7 | DD-1 |
| T2 | FR-4, FR-6, FR-7 | DD-2 |
| T3 | FR-4, FR-5, FR-6 | DD-2, DD-4, DD-6 |
| T4 | FR-2, FR-8 | DD-5, DD-7 |
| T5 | FR-3, FR-5, FR-6 | DD-3, DD-6 |
| T6 | FR-8 | DD-7 |
| T7 | FR-8 | DD-7 |
| T8 | FR-1, FR-7 | DD-1, DD-2, DD-3 |
