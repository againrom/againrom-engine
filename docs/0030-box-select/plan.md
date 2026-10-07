# Plan — a set where a single id was, and a rectangle over the tap's own cells

## Baseline

`Viewer.step` is `cmd/mapview`'s path as well as the map screen's. Inside it, `dragIntent`'s anchor
branch — the first tick with `!v.dragging` — is the **only** place that knows a gesture is beginning,
and it already zeroes `dragMoved` there; its difference branch holds the one subtraction, feeding the
camera and the accumulator from a single delta. `dragX/dragY` are overwritten every tick, so no press
point survives. `panIntent` returns before its edge-scroll term whenever `in.PrimaryDown`. `Input`
carries the pan keys, the cursor, the wheel and `PrimaryDown` and **no modifier of any kind**;
`appInput` adds the primary edges and `SecondaryPressed`, a press edge with no level.

`decide(cur, ents, cam, g)` is pure, calls `Camera.ScreenToCell` and nothing else, and hits a unit by
**cell equality**, minimum `ID` on a tie. There is **no sprite-rectangle pick anywhere in the tree**.
`ScreenToCell` floors each axis by `CellSize` in `float64` against `Cols`/`Rows` and reports outside
rather than clamping. `command` is unexported — what keeps `cmd/mapview` selection-less — and
`App.step`'s `ScreenMap` arm is its one caller, running it after `v.step`. `flow` holds `tick` and
`order` beside `viewer`, set by `choose`, dropped by `escape`.

`overlayScreenRects(show, cells, glyph)` is the one world-to-screen transform behind every glyph: the
displaced lift `-AnchorHeight(cell) - MinV`, `cam.Zoom`, the cull. `overlayPasses()` **is** the
draw order and omits an empty pass; `selectionScreenRects` rides it over `selectionCell`'s single cell,
appended last. Every glyph builder in `pkg/render/terrain` takes `(col, row, cols, rows, cellpx)`.
`mapWorld.enqueue` appends one `sim.Command` per call; `tick()` empties `pending` between assembly and
`sim.Step`.

`pkg/sim.Step(w, cmds []Command)` already takes a whole slice: targets in slice order, then each unit
holding one resolved in ascending id through `searchRoute`, which dispatches on the world's own mode
byte. A search finding no route raises `Stall` and touches nothing else; `Stall >= 16` clears the target
with no residue; and a unit never enters a cell another holds, so a second unit ordered onto an occupied
cell finds no route at all, in either mode.
`TestNoExportedMethodCarriesASelectionOrAnOrder` is the shipped export census over `pkg/ui`.

## Design decisions

### DD-1 — the selection is a slice of ids, ascending

`[]uint32`, replaced whole by a tap or a release and by nothing else. The snapshot arrives in ascending
id, so a walk keeping the covered units is sorted **by construction** and nothing sorts afterwards;
FR-4's emission order is the slice's own rather than a step taken to obtain it. A tap keeps the
minimum-id rule by yielding a slice of one.

Rejected: **a `map[uint32]bool`** — Go randomises range order per process, so the order commands leave
in would be per-process, which is exactly why the shipped commanded-set is read by key and never ranged.
Rejected: **sorting the slice after the walk** — a step that can be forgotten where the walk cannot
(FR-2, FR-3, FR-4).

### DD-2 — command mode is an unexported flag the front-end sets

One bool on `Viewer`, written where `flow.choose` already stores `tick` and `order` and cleared where
`escape` drops them. `cmd/mapview` is another package, so it can no more set this than call `command`:
"the standalone viewer keeps its plain-drag pan and draws no rectangle" stays **Go's export rule** — the
mechanism that already keeps it selection-less — rather than a condition someone must keep true.

Rejected: **gating on `len(v.entities) > 0`** — a world holding no unit would silently lose its pan, and
the value moves every tick, so a gesture could change class inside one press. Rejected: **a parameter on
`step`**, whose signature is `Viewer.Update`'s path too, so every caller would name a gesture set
(FR-1, FR-6).

### DD-3 — the modifier joins `Input`; the latch is set at the anchor

`Input` gains `Shift bool`, read in `readInput` from either Shift key. `Viewer` gains the latched mode
and `pressX/pressY`, all three written **only** in `dragIntent`'s anchor branch, beside the zeroing of
`dragMoved` already there. One branch decides "a gesture began", so the latch, the press point and the
accumulator cannot disagree about when — FR-7 as a shape rather than a rule. The difference branch keeps
its single subtraction and keeps raising the accumulator in either mode; in marquee mode it returns a
zero pan, so the slop measures the same path length for both gestures.

Rejected: **latching in `command` from the primary press edge** — it runs after `v.step`, so the latch
would arrive a call late and sit a method away from the accumulator it must agree with. Rejected:
**reading the modifier live**, which is C-3 (FR-1, FR-7).

### DD-4 — the camera answers a clipped, normalised cell range

`Camera.ScreenToCellRange(x0, y0, x1, y1) (TileRange, bool)` beside `ScreenToCell`: each corner through
`ScreenToWorld`, `math.Floor` by `CellSize`, the two swapped into min/max, the max end raised by one for
a half-open range, then clipped to `Cols`/`Rows` — `false` when the rectangle misses the map. It belongs
on the camera because the camera owns both the inverse and the extent, so pick and draw answer to one. The `+1` is what makes a zero-width or zero-height rectangle one cell wide on
that axis instead of empty, and normalising here makes orientation-independence one expression's
property (FR-2).

Rejected: **transforming each unit's cell forward and testing the rectangle** — a per-unit transform
where one inverse serves the whole snapshot, and it would have to decide what a cell's screen rectangle
is, which is the drawn-rect geometry C-1 rules out. Rejected: **anchoring the rectangle in world
coordinates**, which tracks terrain under a keyboard pan and leaves the cursor drawing it.

### DD-5 — one pure `decide`, widened to a set

`gesture` gains `boxed bool` beside `tap` and `right`; `decide` returns `([]uint32, []order, bool)` and
still calls the camera and nothing else. Keeping **one** function keeps P-3's totality observable: a
second pure function for the release would split the four outcomes across two bodies and no test could
see they are exhaustive. The body stays a branch chain — box, else tap, else order, else nothing — so a
release and a right press in one frame yield the release alone. The right arm returns early while
`v.held`, FR-4's in-progress clause read off state the shell already keeps (FR-2, FR-3, FR-4).

### DD-6 — the mark takes a cell list; absence is filtered, never pruned

`selectionCell` becomes `selectionCells`, one merge walk of the snapshot keeping the ids the selection
holds, in ascending id; `selectionScreenRects` hands that list to `overlayScreenRects` unchanged, so the
lift, the transform, the cull and the glyph are the shipped ones and no second copy exists. The same
walk feeds FR-4's emission, so "present in the snapshot" is one predicate, not two kept in step.

The set is **not pruned**: pruning is a per-tick write to selection state driven by the snapshot, making
what is selected a function of which ticks ran between two clicks. Disclosed cost — an id that left and
returned would be marked and ordered again; no world can resurrect an id today, and FR-5 fixes
skip-not-drop so this is stated rather than inherited (FR-4, FR-5).

### DD-7 — the outline is the one screen-space pass

Every other pass is built from cells and goes through the world-to-screen transform; this one is already
in screen pixels, so it bypasses that transform and is built from `pressX/pressY` and the current cursor
alone. It is four strips — a hollow rim, for the selection mark's own reason: filled, it would hide the
very units the drag is choosing. Appended **after** the selection pass, it covers nothing, being a rim on
the drag's own boundary.

Its thickness and colour join `EdgeMargin` and `PanSpeed` as the viewer's own UX constants rather than
the render tier's glyph palette: every builder there takes a cell and a scale, which a screen rectangle
has neither of. Rejected: **a cell-space glyph in the render tier**, which snaps the feedback to the
cell lattice while the cursor moves by pixels (FR-6).

### DD-8 — the seam is unchanged, called once per unit

`MapOrder` keeps its three scalars. `App.step`'s map arm walks the orders `decide` produced and calls
`issue` for each in ascending-id order, so k orders are k appends into the shipped queue and are drained
by the one advance that truncates it — FR-8's "one frame's orders reach one advance" is that truncation's
position, not a new rule. `pkg/game` and `pkg/sim` are therefore untouched.

Rejected: **widening `MapOrder` to take a collection** — the window tier would build and own a collection
for the far side, and the far side would gain a second append path beside the one statement that both
queues an order and marks its entity commanded (FR-4, FR-8).

## Risks

- **R-1** Panning by drag is the gesture a player of this build has learned, and it moves behind a key
  with nothing on screen to say so.
- **R-2** A group ordered to one cell arrives one unit deep, the surplus stopping where it stands rather
  than closing in. It will read as a bug in a build the owner runs, and it is not this story's to fix:
  the behaviour is the world's own rule and the determinism wall forbids touching it here.
- **R-3** A route is searched per moving unit per tick, so a box over a large group multiplies that by
  the group's size until they arrive or give up.

## Success criteria

- **SC-1** AC-1 and AC-8 hold in full, both drags driven through the shipped `dragIntent` rather than a
  delta of the test's own, the no-world half asserted on a viewer built as `cmd/mapview` builds one, and
  the accumulator checked to rise equally in both modes — so the slop is not silently a pan-only measure
  (FR-1, FR-7).
- **SC-2** The cell range is pinned against `ScreenToCell` itself: at several positions and zooms a
  one-pixel rectangle yields exactly the cell that method names. Zero-width, zero-height and
  corner-swapped rectangles, one off each of the four edges, and one straddling an edge are each their
  own case (FR-2).
- **SC-3** AC-2 and AC-3 hold in full over a **non-identity** camera, panned off the origin and zoomed
  off 1, an identity fixture being unable to witness the transform at all; replace-and-clear is asserted
  against a prior selection of more than one (FR-2, FR-3).
- **SC-4** AC-4 holds in full, the ascending order asserted on the emitted slice with the snapshot given
  in both id orders so the order is the set's, not the snapshot's; P-3 sampled over the edges by latched
  mode by hit/miss/outside by empty/one/many; P-5 witnessed on a press that never releases (FR-4).
- **SC-5** AC-5 holds in full on a hand-built world: targets compared field by field at the advance that
  applied them, the shared-cell check run after **every** tick rather than at the end, and the surplus
  units' resting places recorded as an observation. P-1 sampled by issuing with no advance, digest
  unmoved (FR-4, FR-8).
- **SC-6** AC-6 holds in full, flat and displaced, each mark carrying the lift a marker on its own cell
  carries and the pass still holding the units' sprites; the empty and all-absent cases assert **no pass
  is appended** (FR-5).
- **SC-7** AC-7 holds in full, the four strips asserted to leave the interior uncovered at both drag
  orientations, and the no-world viewer asserted to build none (FR-6).
- **SC-8** AC-9 and AC-10 hold in full: digests at every tick against a headless run, two runs of one
  script compared tick by tick, the pinned fields, byte form, digest and both wall checks unmoved, the
  standalone render and headless line byte-identical, the advance count read at zero, one and many
  orders. P-2 sampled by driving a selection, a rectangle and a queue against a run without (FR-8).
- **SC-9** Six mutants, each applied to production code, run over the whole tree with its failing tests
  named, and reverted: the latch read live instead of at the anchor; the range's half-open `+1` dropped;
  its normalisation removed; the emitted slice reversed; the presence filter deleted from the walk that
  feeds both the marks and the orders; the command-mode gate deleted, taking the standalone viewer's pan
  away. The pass-order swap is **not** on the list — it moves no pixel a test can read, and a criterion
  invented for it would be a mutant manufactured to fill a slot (FR-1, FR-2, FR-4, FR-5, FR-7).
- **SC-10** AC-11 is run against a lawful install on a map with relief (FR-1, FR-2, FR-3, FR-4, FR-5,
  FR-6).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1, C-2 | DD-2, DD-3 | SC-1, SC-9, SC-10 |
| FR-2, C-1 | DD-1, DD-4, DD-5 | SC-2, SC-3, SC-9, SC-10 |
| FR-3 | DD-1, DD-5 | SC-3, SC-10 |
| FR-4, C-3 | DD-1, DD-5, DD-6, DD-8 | SC-4, SC-5, SC-9, SC-10 |
| FR-5 | DD-6 | SC-6, SC-9, SC-10 |
| FR-6 | DD-2, DD-7 | SC-7, SC-10 |
| FR-7 | DD-3 | SC-1, SC-9 |
| FR-8 | DD-8 | SC-5, SC-8 |
