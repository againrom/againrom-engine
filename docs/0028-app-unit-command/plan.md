# Plan — selecting a unit and ordering it from the running game

## Baseline

- `App.step`'s `ScreenMap` arm is the one advance call site; `flow` holds `tick` beside `viewer`, set
  by `choose`, dropped by `escape`. `appInput` reads the left button as press/release **edges**, the
  viewer's snapshot as a level; neither reads the right button.
- `Viewer.step` is `cmd/mapview`'s path too, and `dragIntent` is the one place a per-tick screen delta
  exists: anchor on the first held tick, difference on every later one.
- `camera.Camera` exposes `ScreenToWorld` and holds `Cols`/`Rows` from the terrain grid — the extent
  `mapload.FromALM` also records as the world's bounds, out of one decode.
- `overlayPasses` **is** the drawn order, and every glyph it places goes through `overlayScreenRects` —
  one camera transform, one relief lift, one per-rect cull — at native `terrain.CellSize` alone.
- `mapWorld.tick` hands `sched[world.Tick()]` straight to `sim.Step`, which sets targets in slice
  order, last write winning, then moves. `paceTo` runs 0–4 ticks a call, so most run none;
  `Schedule`'s entries are append-grown; `facing` is the precedent for an id-keyed map read only by
  lookup.

## Design decisions

### DD-1 — the id crosses as a plain integer, absence as a bool

`ui.MapEntity` gains `ID uint32`, filled by `entityDraws`; a selection is `struct{ id uint32; has
bool }`. `uint32` is the id's own width, so the conversion is lossless both ways where an `int`
would admit values no id can hold, and being a builtin it names no simulation type (FR-1, FR-3,
FR-7).

The widened gate lets `pkg/ui` say WHICH entity — hold a selection, hit-test one, address an order at
one. It still cannot name a `sim` type, learn anything past the cell and art it is handed, or
**construct** an id: absence is the bool rather than a sentinel, so every id it holds was minted
across the seam (FR-9).

Rejected: a render-tier id type aliasing the simulation's — `pkg/render` may see `pkg/sim`, so it
hands `pkg/ui` a simulation type under another name.

### DD-2 — the pick is a camera method, and outside is an answer

`camera.Camera.ScreenToCell(sx, sy float64) (col, row int, inside bool)` — `ScreenToWorld`, each axis
floored by `CellSize`, tested against `Cols`/`Rows`. It belongs here because the camera owns both the
inverse and the extent `VisibleTiles` clips against, so pick and draw answer to one (FR-7, C-2). The
floor is `math.Floor` and the range test runs in `float64`, so neither a truncation's off-by-one below
zero nor an implementation-defined conversion of a non-finite value is reachable. Outside is reported,
never moved inside: no clamp appears on the path (AC-1, AC-10).

No vertical origin is subtracted, per C-1 — the terms cancel under uniform relief, so a shift would be
wrong by the whole relief. Rejected with it, the nearest drawn unit inside a screen radius, needing a
height sampler and a radius in the pick.

### DD-3 — one accumulator, one pure function, four outcomes

`TapSlop = 4`. `Viewer` gains `dragMoved int`, raised inside `dragIntent` by `|sdx| + |sdy|` — the
delta that tick panned by — so "the view moved" and "this was a drag" are one number that cannot
disagree (FR-1). It is a path length: a press that wanders out and back panned that far and is a drag.
It is zeroed wherever a gesture begins — `dragIntent`'s anchor branch and the press edge — and neither
alone suffices: the anchor never runs for a press and release inside one frame, the press edge never
fires for a button already held when the map opened. `held`, set by that edge, makes a release with no
press nothing at all (AC-2). The right button is read as a press edge alone: no level, no anchor, so
no drag gesture for it is representable (FR-3).

`decide(cur selection, ents []MapEntity, cam *camera.Camera, g gesture) (selection, order, bool)` has
no receiver and calls `ScreenToCell` and nothing else (FR-8). `gesture` is one frame's edges and
cursor; the extent arrives inside `cam` (DD-2), so there is no second extent parameter.
`Viewer.command` is the impure shell: it calls this, stores the selection, hands the order back. A tap
resolves its cell, then hits the entity on it with the **lowest** `ID`: a minimum over the walk, not
the first match, so the tie stays stated over ids rather than becoming slice position (FR-7). A hit
selects, replacing; a miss and an outside cell both clear.

A tap and a right press in one frame yield the **tap alone**, so the body is a two-level branch — tap,
else order, else nothing — total by construction (P-5). Rejected: honouring both against the freshly
selected unit — an order's subject would then depend on a 16 ms coincidence, and a fifth outcome joins
the four counted.

### DD-4 — the machinery lives on the Viewer, unexported

Pick and highlight both need camera, extent and snapshot, so the selection lives on the `Viewer` — and
`command` is **unexported**. `cmd/mapview` is another package and cannot call it, so the standalone
viewer gaining no selection, order or highlight is Go's export rule rather than a convention, and its
render stays byte-identical (FR-10, AC-11).

The map arm calls it **after** `v.step`, which the accumulator forces: this tick's delta must be in
`dragMoved` before the release is judged. So the arm's call order becomes observable, and an order
issued on a frame reaches the *next* advance, one frame plus at most one tick period later — accepted
against judging a release before this frame's movement is in. The advance call stays first,
unconditional, single (FR-10).

### DD-5 — the order leaves through a seam beside the tick

`MapOrder func(entity uint32, x, y int)` joins `MapTick` as `MapLoader`'s fourth result and as a
`flow` field `choose` sets and `escape` drops, so it lives and dies with the viewer as the tick does.
Three scalars are the whole payload, so no new value type crosses, and `MapTick` is untouched:
parameterless, one unconditional call per map-screen tick (FR-4, FR-10).

Rejected: widening `MapTick` to take the frame's orders and return how many it applied. The queue then
sits in `pkg/ui` while whether a logic tick fired is decided in `pkg/game`, so `pkg/ui` must retain
whatever the count says went unconsumed — and an advance firing no tick drops the order, turning
exactly-one into none.

### DD-6 — the queue, the drain and the count sit beside the world

`mapWorld` gains `pending []sim.Command`. `enqueue`, the `MapOrder` the loader hands over, converts
and appends and does nothing else; `sim.Step` is still reached from `tick` alone, so an order reaching
the world only through an advance is one call site's property (P-1).

`tick` becomes `tick() int`: assemble the slice, truncate `pending` to zero length, step, return how
many were applied. The truncation stands between assembly and step, so no path applies a batch twice
(P-6). `paceTo` returns `(ticks, applied)` and `paced` discards both, staying `ui.MapTick`, so the
report rides the spelling that already takes its instant as an argument, needing no window (FR-4). A
call firing no tick applies none and keeps the queue; the drain is at the first tick of the advance
firing one, so the count is the queue's length there and zero elsewhere (AC-8).

### DD-7 — the exclusion at issue; the untouched path stays untouched

`commanded map[sim.EntityID]bool`, written by the same statement that enqueues, read by lookup and
never ranged — `facing`'s precedent, so no map order reaches a world (FR-6, FR-9). Nothing clears it
while the map is open — C-4's "for good" — and it dies with the `mapWorld`.

`commands() []sim.Command` assembles one tick's stream: the schedule's entry for the world's own tick,
filtered into a **fresh** slice when `commanded` is non-empty, `pending` appended after into a fresh
slice when it is non-empty. With neither, the schedule's own slice is returned unchanged rather than
copied, so no-invented-command is structural and witnessable by identity and no `append` writes into a
schedule entry's spare capacity (FR-5, P-3).

Splicing pending after the script makes the player win a shared tick under last-write, and its witness
is the assembled slice, not a digest: `commanded` removes the only scripted command an order could
lose to, so no world state distinguishes the two splice orders (SC-9).

### DD-8 — the highlight is the footprint's rim, and it goes last

`terrain.SelectionMarkerRects` joins the glyph family, taking its signature and its two shared rules —
the off-map/invalid-scale rejection and the per-rect clip — and returns the footprint's border as four
disjoint strips `scaleDim(2, cellpx)` thick, top and bottom full width, left and right between. Hollow
satisfies FR-2's visibility clause where a filled footprint erases the art it marks.

`Viewer.selectionScreenRects` rides `overlayScreenRects` over that entity's cell alone, shown only
when the selection resolves in the held snapshot, so it takes the shipped glyphs' own lift, transform
and cull, and an absent id yields no cell and so no pass (FR-2, AC-9).

It is appended **last**: over the sprite it must be to be seen, and it covers nothing, since at native
scale every other glyph lies within `scaleDim(6)` of the cell centre while the rim hugs the
footprint's edge sixteen pixels out. Rejected: a filled cell, and a pass under the sprites.

## Risks

- **R-1** The slop is a path length, so a slow hand wandering four pixels while clicking pans and
  changes no selection. Accepted: the view moved by exactly that much, so player and code named the
  same gesture, and four pixels is an eighth of a cell at native zoom.
- **R-2** C-1's disclosed error reaches shipped behaviour: near a cliff on a map with a large altitude
  spread, a click resolves a cell up to that spread in rows away, so the wrong unit is selected or the
  wrong ground ordered. Accepted; the pick is exact on flat and on uniform relief.

## Success criteria

- **SC-1** AC-1 holds in full over several positions and zooms, the round trip asserted both ways, a
  cell beyond each of the four edges reported outside and its indices unread (FR-7, C-1).
- **SC-2** AC-2 holds in full, its three named sequences driven through the shipped `dragIntent`
  rather than a second delta of the test's own (FR-1).
- **SC-3** AC-3, AC-4 and AC-10 hold in full; P-5 sampled over the cross product of the two edges by
  hit/miss/outside by selection/none, each landing in exactly one outcome; the shared-cell tie
  asserted over ids given in both slice orders (FR-1, FR-3, FR-7, FR-8, C-2, C-3).
- **SC-4** AC-9 holds in full, flat and displaced, the displaced rim carrying the lift a marker on
  that cell carries, the pass list still holding the entity's sprite and a coincident cross (FR-2).
- **SC-5** AC-5 holds in full — the digest at every tick against a headless run over the one command
  at the tick it drained — and P-1 sampled by enqueueing with no advance, digest unmoved (FR-4, FR-8).
- **SC-6** AC-8 holds in full — two orders between two advances, the first reporting two and the
  second zero, the unit heading for the second cell — plus a third advance after one further order
  reporting one, so the count is no constant (FR-4, P-6).
- **SC-7** AC-6 holds in full over a hand-written schedule still issuing targets past two of the
  commanded unit's turns, a second unit reaching its own throughout (FR-6, C-4).
- **SC-8** AC-7 holds in full; P-3 witnessed by identity — nothing enqueued, nothing commanded, the
  assembled slice IS the schedule's entry — and P-4 sampled over two worlds advanced by one combined
  stream, digests compared at every tick (FR-5, FR-9).
- **SC-9** Five mutants, each measured and reverted with its failing tests named: `ScreenToCell`'s
  outside test deleted; the slop comparison widened to `<=`; the exclusion write removed; the queue
  left untruncated; pending spliced before the script. The last two are killed narrowly and nowhere
  else — the untruncated queue by the count alone, a re-applied identical order being invisible in
  world state under last-write; the reversed splice by the assembled-slice assertion alone (DD-7).
  Neither moves a digest, and neither is claimed to (FR-4, FR-5, FR-6, FR-7).
- **SC-10** AC-11 holds in full: pinned field sets, byte form and digest unchanged, both wall checks
  green over packages that gained no import, the standalone render and the headless check line
  byte-identical; P-2 sampled by driving a selection, a pick and a pending order and comparing digests
  with a run without (FR-9, FR-10).
- **SC-11** AC-12 is run against a lawful install on a map with relief (FR-1, FR-2, FR-3, FR-10).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1, C-3 | DD-1, DD-3 | SC-2, SC-3, SC-9, SC-11 |
| FR-2 | DD-1, DD-8 | SC-4, SC-11 |
| FR-3, C-2 | DD-1, DD-3 | SC-3, SC-11 |
| FR-4 | DD-5, DD-6 | SC-5, SC-6, SC-9 |
| FR-5, P-3 | DD-7 | SC-8, SC-9 |
| FR-6, C-4 | DD-7 | SC-7, SC-9 |
| FR-7, C-1 | DD-1, DD-2, DD-3 | SC-1, SC-3, SC-9 |
| FR-8 | DD-2, DD-3 | SC-3, SC-5 |
| FR-9 | DD-1, DD-6, DD-7 | SC-8, SC-10 |
| FR-10 | DD-4, DD-5 | SC-10, SC-11 |
