# Plan — one timeline, one counter, and a gate the map holds

## Baseline

`LoadStatics` resolves each object class to exactly one converted frame, `sheetCache.frame(path,
Index)`, and whether that is nil IS the drawability rule: the census counts a byte naming no class
and a class with no art apart. `sheetCache.frames(path)` already sits beside it — the unit loader's
path — converting a whole sheet once per path, memoising failures like successes, and returning nil
for a sheet that is absent, undecodable, palette-less or empty. Its per-frame conversion is
`frame()` itself, so the only exclusion `frames()` can hit is the whole-sheet one.

`StaticPlacements` walks the object grid once per geometry and, per placed cell, calls
`StaticAnchor(col, row, canvasW, canvasH, centerX, centerY, frameW, frameH, lift, originY)`, storing
`Cell`, `TopLeft`, `Anchor` and the frame pointer. `Ground()` is `TopLeft + Anchor` and `Rect()` is
`TopLeft` plus the frame's size. Only `anchorX`/`anchorY` read the frame's size, so **one cell's
ground point is the same world point for every frame of its class** and the lift and origin enter it
once, at the build.

Both geometry lists are built in `NewViewerWithStatics` and never rebuilt; `staticPlacements()`
selects one per frame and `drawStatics` draws it unchanged, gated on `showStaticArt`. The raster
tool builds its own list once, refuses any scale but 1, and blits `BlitStatic` or `BlitStaticLit`
per placement at the row `SpriteRow(light)` gives. The window's texture cache is keyed
`spriteTextureKey{frame *StaticFrame, row int}` and is nil until the first draw.

`Ticker` holds a period, an accumulator and a `uint32` count. `advanceAnimation` takes the baseline
timestamp and then returns before `AdvanceMicros` when `v.animate` is false, so the switch holds the
counter rather than accumulating a paused span. `resolveCell` calls `ResolveAnimated(word, col, row,
v.anim.Count())` when animation is on and `Resolve(word)` otherwise. `SetRate` re-rates the ticker
and returns the rate it adopted; the flow spends that return in one statement,
`f.cadence(f.viewer.SetRate(f.rate), f.stopped)`, and no method on the viewer takes or reports a
stop. `mapWorld.scene` is a different clock, `+1` per world tick, read only by the unit selector.

`pkg/data.expandTrack(times, frames)` is already the run-length rule — frame `i` appended `times[i]`
times, a non-positive time appending nothing, the walk ending when either slice ends.
`ObjectClass` carries `AnimationTime` and `AnimationFrame`
resolved through the inheritance chain and validated to one length, with no consumer.

`terrain.Grid` carries `Tiles []uint16` beside `Overlay`, and `splitTileWord` masks the strip fields
and never reads bits 15..14. `pkg/render/terrain` imports the standard library alone; `pkg/sim` is
imported by no file this story touches.

## Design decisions

### DD-1 — the timeline is derived in the data tier and carried across as plain ints

`ObjectClass` gains a derivation beside `UnitClass.Anim()` returning the expanded timeline, reusing
`expandTrack` unchanged; the loader copies it onto `StaticClass` as a `[]int`, the `StaticPixel`
precedent. The render tier therefore re-derives no registry rule and never learns that a timeline
came from two arrays.

Rejected: **expanding in the render tier** — it may not import `pkg/data`, so the rule would exist
twice, in the tier that owns registries and in the tier that draws. Rejected: **expanding at draw
time** — the walk is per class, and a class placed at two hundred cells would rebuild it two hundred
times per rendered frame to answer one index.

### DD-2 — a drawable class carries its whole sheet, and `Frame` stays the drawability answer

`StaticClass` gains `Frames []*StaticFrame` — the loader's per-path memo, shared pointer for pointer
between classes naming one sheet — plus `Index` and the timeline, and keeps `Frame` as the plain
arm. The loader's own arithmetic then gives `Frame != nil` exactly when `Frames != nil` and `Index`
is inside it, because `frames()` fails only on the whole-sheet exclusions and `frame()` adds only
the range test. So the four exclusions, the two artless answers and the census that counts them
apart are untouched, and every selection FR-2 can make has somewhere to land.

Rejected: **dropping `Frame` and reading `Frames[Index]` at each site** — it moves the decision the
census depends on into every caller. Rejected: **loading whole sheets only for classes carrying a
timeline** — the memo is per path and classes share paths, so it saves nothing and adds a condition
under which a class's frames are sometimes present.

### DD-3 — one pure selector, and it does not compute its own gate

The three arms of FR-2, FR-3's euclidean step and the range guard live in a single function taking
the period, the timeline, `Index`, the sheet's own frame count, the cell, the counter, the animation
switch and **the gate as a boolean**. It mirrors `SelectUnitFrame` down to the guard being the last
act, so an out-of-range value is answered once rather than at each arm.

Rejected: **a selector that reads the tile grid** — the gate is per cell and settled at the build,
the frame is per counter, and folding them makes the per-counter pass carry the whole grid.
Rejected: **each front-end choosing arms for itself** — the arm order is the contract, and a second
copy is how a raster and a window come to draw different frames while both look right.

### DD-4 — the gate is decided once per build; only open cells are re-selected per counter

`StaticPlacements` takes the gate mode and returns a third value beside the list and the census: the
**animated subset**, one entry per placement whose cycle is open, carrying that placement's index in
the list and the class it drew from. `StaticCounts` gains `Animated`, its length. The per-counter
pass walks the subset alone.

This is what makes P-5 structural: where no cell opens a gate the subset is empty, the per-frame
work is zero, and the pre-story picture is what the built list already holds rather than something a
selector has to keep reproducing.

Rejected: **rebuilding the whole list per counter** — a `W*H` walk and a fresh placement slice on
every rendered frame, which is the shape 0029 measured the cost of. Rejected: **a per-placement
`animated` flag walked every frame** — the walk is then the placement count instead of the
open-cycle count, so a map with thousands of trees pays thousands of comparisons to answer "no".

### DD-5 — an animated placement is re-anchored from its own ground point

The per-counter pass computes the new anchor pixel from the class canvas and the **drawn** frame's
size and sets `TopLeft = Ground() - Anchor`. It needs no lift, no origin and no projection, so it
takes none and the two geometries stay exactly one build apart.

Rejected: **calling `StaticAnchor` again** — its signature needs `lift` and `originY`, so a
per-frame pass would carry a projection and could disagree with the build about a cell's height.
Rejected: **leaving `TopLeft` where it is** — frames of one sheet need not share a size, so a taller
cycle frame would stand off its own ground point and P-4 would be false by construction.

### DD-6 — the pass writes into a caller-owned buffer, and returns the built list when nothing animates

The window keeps one scratch slice, reused every frame; the raster tool takes one copy. With an
empty subset the pass returns the built slice itself, so an ordinary map allocates nothing and draws
the very list the builder produced.

Rejected: **mutating the built list in place** — the window would stop placing from what the builder
returned, which is the whole content of the two renderers' agreement. Rejected: **a fresh slice per
rendered frame** — hundreds of
kilobytes of garbage a second to move a handful of entries.

### DD-7 — the counter is read where water reads it, and the switch at the draw

The window hands the pass `v.anim.Count()` and `v.animate`. Because `advanceAnimation` already
returns early on the switch, "off" is *both* frame 0 and a held counter with no second statement to
keep true, and switching back on resumes from the held value by doing nothing. Nothing is gated at
the build, so turning the switch changes what is painted and never what was placed — 0017's own rule.

Rejected: **a counter or a rate of the object layer's own** — a third cadence to re-rate, stop and
switch in step with two others, and 0041's out-of-scope named this consumer by name. Rejected:
**`mapWorld.scene`** — the stop halts it, so objects would freeze while the water beside them moved,
and the standalone viewer has no scene clock at all.

### DD-8 — the diagnostic is a two-valued gate mode in the builder's signature

`AnimGateTiles` and `AnimGateAll`, passed positionally, echoing `withHeights`/`withoutHeights` in
the compositor. It is read at the one place the gate is computed, so no other tier learns of it.

Rejected: **a package-level variable** — invisible at the call site and shared between two viewers
in one process. Rejected: **a bare bool** — five positional arguments of which one is an unnamed
`true`, at three call sites.

### DD-9 — the raster tool resolves its counter once, above the statics loop

`-tick` parses to a `uint32`, the list is built with the chosen gate, the per-counter pass runs once,
and the blit loop draws what it returns. So the PNG is a still of one counter, and `-scale 1`,
`-flat` and `-unshaded` keep the meanings they have.

Rejected: **rendering a sequence** — the tool writes one image, and a sequence is the caller's own
loop over `-tick`.

### DD-10 — the open-cycle count rides on the census, not on a report of its own

`StaticCounts.Animated` is filled by the builder and printed beside the placement count in both
summaries, so the number is the one the build actually produced.

Rejected: **a second reporting path** — two numbers derived from two walks, free to disagree about
one map.

## Risks

- **R-1** The gate is closed everywhere on shipped data, so the animated arm's only routine exercise
  is synthetic. A defect there can sit in a lawful install unseen until the reachability question is
  answered elsewhere, and no owner review will find it. SC-8's mutants are the whole mitigation.
- **R-2** Carrying every frame of a sheet raises the loader's decode and memory cost. The ceiling is
  the number of distinct sheets the registry names, not the class count, because the memo is per
  path — but it is paid at map load, where a run already waits.
- **R-3** Objects cycle while the world is stopped. A reviewer who pauses to inspect a frozen scene
  sees foliage moving, and that reads as a bug until the disclosure is read.
- **R-4** The water stagger and the object stagger are transposes of one another and live in one
  package. A copy between them agrees on the diagonal and nowhere else, which is exactly the kind of
  wrong a rendered picture does not show.

## Success criteria

- **SC-1** AC-1 and AC-6 hold in full, AC-1's four pairs each a case of its own with its expected
  timeline written out by hand rather than produced by the expansion under test (FR-1).
- **SC-2** AC-2 holds over all four cells and all 32 counters, its expectation computed from the
  contract inside the test, and the transposed stagger asserted to differ at every counter (FR-3).
- **SC-3** AC-3 holds on every gate case in both diagnostic states, the last-column and last-row
  cells each on a case of its own rather than folded into one (FR-4).
- **SC-4** AC-4 and AC-9 hold in full, each arm and each refusal on its own case, and the
  bundle-less build compared **whole** against the pre-story list rather than by count (FR-2, FR-9).
- **SC-5** AC-5 holds with its two frames differing in **both** dimensions, the ground point compared
  as a point and the two top-lefts as points, so a single-axis error cannot average out (FR-6).
- **SC-6** AC-7 holds, the raster pixels and the window's texture pixels compared byte for byte at
  one counter and at a second counter that selects a different frame (FR-7).
- **SC-7** AC-8 holds, the counter read through both consumers and compared as one value, and 0041's
  digest and byte-form checks re-run unmoved (FR-5, FR-9).
- **SC-8** Five mutants, each applied to production code, run over the whole tree with its failing
  tests named, and reverted: the stagger transposed to water's `(col+1)*row`; the counter shifted
  `>>2` as water's is; the gate reduced to the period test alone; the disabled arm drawing `Index`
  instead of frame 0; the re-anchor dropped so an animated placement keeps the built frame's
  top-left. Each is a design decision expressed as a defect — DD-3, DD-7, DD-4, DD-3, DD-5 — so a
  survivor is a criterion that does not discriminate its own decision, and is reported as one.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-2 | SC-1 |
| FR-2 | DD-2, DD-3 | SC-4, SC-8 |
| FR-3 | DD-3 | SC-2, SC-8 |
| FR-4 | DD-4, DD-8 | SC-3, SC-8 |
| FR-5 | DD-7 | SC-7, SC-8 |
| FR-6 | DD-5, DD-6 | SC-5, SC-8 |
| FR-7 | DD-3, DD-6 | SC-6 |
| FR-8 | DD-7, DD-9, DD-10 | SC-6, SC-7 |
| FR-9 | DD-6, DD-7 | SC-4, SC-7 |
