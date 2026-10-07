# Plan — a running world under the map screen

## Baseline

`ui.MapLoader` is `func(index int) (*Viewer, error)`; `flow` holds the viewer it returned, `choose`
sets it and `escape` clears it on the map screen. `App.step` handles Esc before its per-screen
switch and returns, so the leaving tick reaches no screen arm; the map arm calls `Viewer.step`,
which the standalone viewer's `Update` calls too. `NewApp`'s loader is `FrontEnd.loadMap`, which
goes through `LoadMapViewer` — `cmd/mapview`'s entry too — for a `MapView{Viewer, Map, Title}`.
`MarkerCells` converts unit records to cells through `terrain.AnchorCell`.

`overlayScreenRects(show, cells, glyph)` is the one marker transform: rects built at
`terrain.CellSize`, shifted in displaced mode alone by `dy = -proj.AnchorHeight(cell) - proj.MinV`,
each rect's `Min` through `cam.WorldToScreen` with its size scaled by `cam.Zoom`, and a rect wholly
outside the view dropped. `overlayPasses` returns the object, unit and static passes in that order,
omitting an empty one, and `Draw` walks that slice after `drawStatics`. `markerRects` rejects
`cellpx < 1` and an off-map cell before building geometry, centres on `MarkerAnchor` and clips each
arm to the map rect; `scaleDim(n, cellpx)` is `max(1, round(n*cellpx/32))`.

`mapload.FromALM` builds one entity per placed unit, id the unit's slice index and cell `(u.X>>8,
u.Y>>8)`. `sim.Step(w, cmds)` is the only advance; `sim.Run(w, schedule, ticks)` takes `schedule[i]`
at its i-th step and no commands past the end. `*World`'s method set and the sweep over its readers
are pinned in `pkg/sim/world_test.go`. `internal/archtest`'s allow map gives `pkg/ui` `{pkg/render,
pkg/render/}` and `pkg/game` `{pkg/}`.

**Frames and units.** A cell is an integer column and row on the map's row-major lattice. Glyph
geometry is in map pixels at `cellpx` per cell, `CellSize` being 32; a placed rect is in screen
pixels, where the zoom reaches size and origin alike. A tick is an index, not a duration.

## Design decisions

### DD-1 — the advance is a `func()` the UI tier holds and cannot look into

`ui` gains `type MapTick func()`, and `MapLoader` becomes `func(index int) (*Viewer, MapTick,
error)`. `flow` keeps that function beside `viewer`: `choose` stores both or neither, `escape` drops
both on the map screen, and `App.step`'s map arm calls it — nil-checked — before `v.step`. A
parameterless function names no type, so the window tier still cannot spell one, and the world it
closes over is dropped in the same statement as the viewer. Which of the two runs first is not
observable — neither reads the other's state — so nothing asserts one.

Rejected: hanging the hook on `*Viewer`, which `flow` already holds — `Viewer.step` is the
standalone viewer's path too, so a world would advance in `cmd/mapview` against FR-3 and one advance
would have two entry points. Rejected: widening the allow map so `pkg/ui` could hold a world, at the
cost `analysis.md` records. That map is not edited; instead `dag_test.go` pins `pkg/ui`'s entry in
it, so widening it later fails a test rather than passing review.

### DD-2 — `pkg/game`'s `mapWorld`, and the index it steps by

`pkg/game` adds an unexported `mapWorld{world *sim.World; sched [][]sim.Command; view *ui.Viewer}`.
`newMapWorld(w, sched, v)` pushes the tick-0 cells; `openMapWorld(m, v)` supplies
`mapload.FromALM(m)` and `mapload.Schedule(m)`; `tick()` reads `t := world.Tick()`, takes `sched[t]`
when `t < len(sched)` and nothing otherwise, calls `sim.Step`, then pushes the new cells.
`entityCells` is `int(e.X), int(e.Y)` per entity in the order `Entities()` returns — one point each,
no arithmetic of its own.

**The index is the world's own tick**, never a counter the driver keeps: one counting for itself
would agree with `sim.Run` on a world it alone advanced, and so disagree with nothing. Two
constructors exist so a test can drive a schedule it wrote by hand; production enters through
`openMapWorld` alone.

`FrontEnd.loadMap` builds it from the `MapView.Map` its own call already decoded, and returns
`mv.Viewer, mw.tick, nil`. `LoadMapViewer` is untouched: it serves `cmd/mapview` too (FR-3), so
building the world outside it is what leaves the standalone viewer no world to own. Rejected: a
world built inside `LoadMapViewer` and ignored by the tool — an ignored world is still an owned one.

### DD-3 — the schedule: staggered laps of a square, sized to the map

`mapload.Schedule(m *alm.Map) [][]sim.Command`, reading `m.Units`, `m.Width` and `m.Height` alone.
Entity i is given `legs` targets, the j-th at tick `i*stagger + j*legCells`, offset from its own
start cell by the j mod 4 entry of `(dx,0), (dx,dy), (0,dy), (0,0)` — a closed square walked over
and over. `dx` is `+min(legCells, width-1-x)` when the high side has more room and `-min(legCells,
x)` otherwise, `dy` the same rule on rows: every target lands inside the map and no leg is shortened
while room remains. `legCells` is 24, `legs` 24, `stagger` 7. The slice ends one past its last
command, so the closing leg is walked out under FR-2's "no commands past the end" rather than padded
with empty ticks; a map with no units yields none at all.

The numbers answer a schedule that reads as a blink: a leg is 24 ticks, six laps 576, each further
entity starts 7 later, so a map's motion runs about `570 + 7n` ticks against the 256 one crossing of
the widest map takes. 7 being coprime with 24, entities within a run of 24 never turn together and
the field never reads as one rigid formation. Entities are named by the ids `FromALM` assigns, which
is why this lives beside it: one package holds the id convention and the one `u.X>>8` both functions
call.

Rejected: one long crossing per entity — it exhausts in the time it takes to cross and then rests,
the shape this avoids; a drawn or clock-derived walk, against FR-4; and `pkg/game` as its home,
which would put the id convention on two sides of a tier boundary.

### DD-4 — the glyph: an even-sided square on the family's own anchor and clip

`terrain.EntityMarkerRects(col, row, cols, rows, cellpx)` has the signature `overlayScreenRects`
takes and returns at most one rect: `s := scaleDim(10, cellpx)` and the square `[cx-s/2, cx-s/2+s)`
on both axes about `MarkerAnchor`'s point — the centred-strip rule the cross's thickness uses,
applied twice rather than once. The invalid-scale and off-map rejections and the map-rect clip move
out of `markerRects` into helpers both builders call, so the family keeps one rule for each.
`EntityMarkerColor` is magenta, apart from the object yellow, the unit cyan and the static red.

`s <= cellpx` at every scale — `scaleDim` is linear and 10 below 32 — so the square never leaves its
own cell and the clip cannot trim an in-map one; that inertness is asserted, not assumed. Rejected:
a `DrawEntityMarkers` twin on the raster path, which the contract puts out of scope.

### DD-5 — the entity pass is the first pass, and the slice is where the order is read

`Viewer` gains `entityCells`, `SetEntityCells(cells)` and `EntityMarkers() (cells int)`;
`entityScreenRects` calls `overlayScreenRects` with the new glyph, so the displaced lift, the camera
transform and the view cull are the family's, not a second copy. `overlayPasses` prepends the entity
pass, and `Draw` is unchanged — terrain, static art, then the slice — so *under the art* is
inherited from a call order this story does not touch; only the order inside the slice is new.

There is no show flag: having been given cells is the switch, so a viewer given none returns the
slice that shipped. `SetEntityCells` calls no `syncWorld` — the world's extent cannot depend on
which cells hold entities, and this setter runs every tick where the two overlay setters run once.

### DD-6 — the tick-0 agreement is measured by containment, and the cells stay two derivations

Nothing in `pkg/game` shifts a unit anchor: an entity's cell arrives already shifted through
`mapload`, the marker's through `terrain.AnchorCell`. The two `>>8`s stay one per package, in
packages that do not import each other, and the expected cells are literals beside the fixture, so
neither is checked against the other alone.

On screen the two glyphs are **not** compared for equal centres. The square's side is even and the
cross's arms are odd, so their centres differ by exactly half a map pixel — half a screen pixel at
zoom 1 — and a test demanding equality could only pass by copying that half back out of the code. It
asserts containment instead: the unit cross's arm centre falls inside the entity square, in screen
pixels. Its half-extent is 5 map pixels against the 32 a one-cell disagreement moves, so it
discriminates at a sixth of a cell and asserts no arithmetic either side performs.

### DD-7 — fixtures, and the two places AC-5 is witnessed

Every fixture is built in test code: `alm.Map` literals with hand-written unit anchors (no
`alm.Open`; `internal/synth` gains nothing), a `*ui.Viewer` over `&terrain.Tileset{}` and a grid
literal, hand-written schedules. A test written over 32-pixel cells refuses to run when `CellSize`
is not 32, as the marker tests do.

AC-5 is witnessed **twice, and neither half is redundant**. `pkg/game` runs the production driver k
times and compares its digest with `sim.Run` over a fresh world of the same map — but never goes
through `App`, so a second advance in the dispatch is invisible there. `pkg/ui` drives a counting
`MapTick` through the screens — but never sees a world, so an index read off the wrong quantity is
invisible there. Together: once per map-screen tick, nowhere else, and to the state a headless run
reaches.

## Success criteria

- **SC-1** A counting `MapTick` reports exactly one call per map-screen tick — under neutral input
and under pan, drag and wheel — and none on a menu, picker or Esc tick, the camera and the water
counter still moving on the tick the count rises. After Esc the flow holds neither viewer nor tick;
a loader returning an error leaves both nil, the picker showing (AC-2, AC-3, AC-9, P-2, P-4).
- **SC-2** A world opened over a synthetic map stands at tick 0, one entity per placed unit, its
bounds the map's extent; opening, ticking, leaving by Esc and opening again calls the loader a
second time, leaves the flow holding the second tick, and yields a world whose cells equal a copy of
the first's taken before it was ticked (AC-1, AC-4).
- **SC-3** Over a hand-written schedule the entry for tick t is applied at t and at no other tick,
and past the schedule's end no command is applied at all — an entity still short of its last target
walking on to it, one already there standing (AC-2).
- **SC-4** k drives of the production path, its viewer taking the cells each tick, digest-equal a
headless run of that map's own schedule from a fresh world: for a k by which an entity has moved,
and for a k past the schedule's end (AC-5, P-1).
- **SC-5** Two calls of `Schedule` over one map are deeply equal; no target it names lies outside
the map's extent, a unit against each edge included; successive entities' first commands differ by
the stagger; the last index it fills carries a command (FR-4).
- **SC-6** The square is 10 output pixels on both axes at native cell size, centred on the cell
centre, one rect per in-map cell and none for an off-map cell or `cellpx < 1`; at every scale
checked it is no wider than its own cell, the clip leaving a corner cell untouched;
`EntityMarkerColor` equals none of the three diagnostic colours (AC-6).
- **SC-7** With entity cells set the entity pass is first and the three diagnostics keep their order
behind it; with none set the slice is the one that shipped; an entity rect carries its cell's lift
displaced and none flat, one placed off-view is dropped, and the marker count before the cull equals
the world's entity count (AC-6, AC-7, AC-10, P-5, P-6).
- **SC-8** Over units at known anchors, one with a low byte other than `0x80`, the entity cells and
the unit marker cells both equal the literals written beside the fixture; in a viewer holding both,
the unit cross's arm centre falls inside the entity square, in flat and displaced mode and at two
camera positions (AC-8, P-3).
- **SC-9** `pkg/sim` gains no file, its method-set pin and sweep unedited; the allow map is unedited
and `pkg/ui`'s entry in it pinned; the live-tree DAG check still shows `pkg/ui` importing no sim
package; a viewer from `LoadMapViewer` reports no entity cells; the suite is green with no game
install, no GPU and no window (AC-10, FR-9, FR-11).
- **SC-10** (manual) In the built binary over a lawful install, every entity square stands under the
unit cross it was built from at tick 0 and the entities walk as the schedule runs — the first real
map ever through `FromALM` (AC-11).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-2 | SC-2 |
| FR-2 | DD-1, DD-2 | SC-1, SC-3 |
| FR-3 | DD-1, DD-2 | SC-1, SC-9 |
| FR-4 | DD-3 | SC-5 |
| FR-5 | DD-4, DD-5 | SC-7, SC-10 |
| FR-6 | DD-4 | SC-6, SC-7 |
| FR-7 | DD-5 | SC-7 |
| FR-8 | DD-6 | SC-8 |
| FR-9 | DD-1, DD-5 | SC-9 |
| FR-10 | DD-2 | SC-4 |
| FR-11 | DD-7 | SC-9 |
