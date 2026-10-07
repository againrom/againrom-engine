# Tasks — geometry, then buildings on the screen, then everything else

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. They land in ascending order, each
depending only on those before it. Each entry names the mutants SC-5 charges to the lines it writes,
applies them where it wrote them and reverts them there.

## T1 — the bundle, the grid and the placement builder

**files** ADD `pkg/render/terrain/structures.go`, `pkg/render/terrain/structures_test.go`,
`internal/synth/structsheet.go`; MODIFY `pkg/render/terrain/doc.go`,
`internal/synth/synth_test.go`

FR-3, FR-4, FR-10 — DD-1, DD-2, DD-7, DD-11.

**fences** the package gains no import inside the module and none outside it; no archive, registry or
sheet type appears and no address string is spelled here. The placement type has **no anchor field**,
and no expression in the file reads a class width, height or centre — if one could, the story's
premise is gone. Nothing in `statics.go`, `units.go`, `overlay.go` or `project.go` changes, and the
marker geometry is neither called nor consulted. The lift is a parameter that this task always
receives as nil; the height sample itself is T3's.

**done when** AC-3 holds cell for cell and frame for frame against a hand-computed strip, plus P-1
and P-2 as generated properties over classes mixing overhang, no overhang, and rectangles crossing
the map edge. Mutants: `rowTop` computed with `TileHeight` and `FullHeight` transposed; the
`ROW0 == 0` arm dropped so the back row draws no overhang; the strip's vertical step taken as
`+CellSize` instead of `-CellSize`; the grid index built column-major; the sort made unstable; the
column order made ascending.

## T2 — the loader, and the first buildings in a window

**files** ADD `pkg/game/structures.go`, `pkg/game/structures_test.go`; MODIFY `pkg/game/mapload.go`,
`pkg/ui/viewer.go`, `pkg/ui/statics.go`, `cmd/mapview/main.go`

FR-1, FR-10 — DD-1, DD-3.

**fences** `pkg/data` is not touched at all in this entry. The object loader, its sheet cache and the
window's texture cache are reused, never copied and never refactored; if the cache needs a second key
shape the design is wrong, not the cache. The bundle rides the load path as a parameter beside the
object layer's, with a zero value that is exactly the pre-story path — no setter is added to the
viewer. No flag turns this on by default in the game front-end. Nothing here computes a height.

**done when** AC-1 holds with each exclusion counted apart and only an unparseable registry
returning an error, and AC-9 holds for all three no-bundle shapes; and a shipped map opened in
`mapview -structures` shows its buildings standing on their own cells, each with its diagnostic cross
at its top-left corner. Mutants: the key masked with `& 0x7f`; the exclusion for a non-square frame
dropped; a zero extent admitted; an absent sheet promoted to an error; `Classes[0]` filled.

## T3 — the height the whole building stands at

**files** MODIFY `pkg/render/terrain/structures.go`, `pkg/render/terrain/structures_test.go`,
`pkg/game/mapload.go`, `pkg/ui/viewer.go`

FR-3 — DD-6.

**fences** the sample is taken once per structure and stamped; no per-entry evaluation and no
memoisation that could make two entries of one building disagree. The parameter is the corner-height
accessor, not the per-cell one, and the four samples go through one expression with no branch on
parity. `destX` gains no term. `AnchorHeight` is not called, and the object builder's own lift path
is not modified.

**done when** AC-5 holds at all three parity combinations, the odd/odd case agreeing with the centre
cell's four-corner mean by computation rather than by a special case, and P-3 holds as a property
over random height grids. Mutants: the centre taken as the anchor cell; the halving done with a
right shift; the lift added instead of subtracted; `originY` dropped; the sample taken per entry.

## T4 — animation: the timeline, the mask and the ranks

**files** MODIFY `pkg/data/anim.go`, `pkg/data/anim_test.go`, `pkg/render/terrain/structures.go`,
`pkg/render/terrain/structures_test.go`, `pkg/game/structures.go`, `pkg/ui/viewer.go`

FR-2, FR-5, FR-9 — DD-4, DD-5.

**fences** the only change to `pkg/data` is one method beside the object one, reusing the existing
expansion — `keys.go`, `classes.go`, `load.go` and `structureDefaults` are not touched, and this
entry does not repair the registry's unset-scalar defaults. A mask of the wrong length is refused
whole: nothing truncates, pads, or indexes it. Ranks are computed at load, never per frame, and the
mask string is not re-scanned at draw time. The per-counter pass moves no `TopLeft` and allocates
nothing after its first call. No per-cell or per-placement stagger is introduced.

**done when** AC-2 holds for all four precondition shapes, AC-4's live and base arms hold across two
whole cycles, P-4 holds over sheets deliberately short of their own blocks, and P-8 holds for every
scalar this contract reads at both candidate values of an omitted key. Mutants: the mask length
compared against `TileWidth*TileHeight`; the rank counting live cells inclusively; `phase-1` written
as `phase`; the zero-phase arm dropped; the gate accepting `Phases >= 1`.

## T5 — the two passes, and the merge with the object plane

**files** MODIFY `pkg/render/terrain/structures.go`, `pkg/render/terrain/structures_test.go`,
`pkg/ui/statics.go`, `pkg/ui/viewer.go`

FR-6 — DD-8.

**fences** the merge is one helper in the render tier and both call sites call it; neither renderer
carries its own ordering opinion, and no second sort exists. The object layer's own list is neither
rebuilt nor reordered — it is consumed as the builder made it. The entity passes are not moved and
the pass table's existing order is otherwise unchanged. No structure is drawn twice, and a class with
`Flat` set is drawn in the early pass only.

**done when** AC-7 holds for the flat pass, the merged order, the descending columns and the
record-order ties, and P-5 holds as byte identity of the drawn frame for a map with no bundle.
Mutants: the flat test inverted; the flat pass drawn after the object layer; the merge comparing
columns before rows; the merge emitting the object side first at an equal row; a structure enqueued
in both passes.

## T6 — the ruin diagnostic

**files** MODIFY `pkg/render/terrain/structures.go`, `pkg/render/terrain/structures_test.go`,
`pkg/ui/viewer.go`, `cmd/mapview/main.go`

FR-7 — DD-9.

**fences** destruction is a parameter of the draw and never a field of a placement, class or bundle.
The flag exists on the developer tools alone and cannot be reached from the game front-end. The
anchor cross keeps deriving its cell from the record alone and is not routed through this story's
geometry, and its own code is untouched.

**done when** AC-4's ruin arm holds — the block addressed from the end of the sheet, a class with
`Indestructible` set returning to base, and a sheet with no ruin block redrawing its intact art — and
the switch changes the picture on a shipped map without changing any count. Mutants: the ruin base
taken as `TileWidth*FullHeight*2`; `Indestructible` ignored; the ruin arm ordered after the animation
arm; the diagnostic reachable from the front-end.

## T7 — the raster harness and the census

**files** MODIFY `cmd/terraintool/main.go`, `cmd/mapview/main.go`,
`pkg/render/terrain/structures.go`, `pkg/render/terrain/structures_test.go`

FR-8, FR-10 — DD-10.

**fences** the counts are filled by the builder beside the list they count; no tool walks a list to
recount it, and the two levels are separate values that cannot be summed into one another. The
harness places from the same builder output the window does — no second placement path, and no
per-tool adjustment of it. The three corpus observations are printed, never folded into a contract
clause or a default.

**done when** AC-9's census clause and AC-10 hold, P-6 holds as a property over generated maps, and
P-7 holds by the two front-ends being shown to place from one list for one map. Mutants: the
overhang counted as a separate cell rather than a frame; out-of-extent cells counted as drawn; the
per-placement skips overlapping; the harness rebuilding the list with its own geometry.

## Traceability

| Task | FR | DD | Criteria |
|---|---|---|---|
| T1 | FR-3, FR-4, FR-10 | DD-1, DD-2, DD-7, DD-11 | AC-3, P-1, P-2 |
| T2 | FR-1, FR-10 | DD-1, DD-3 | AC-1, AC-9 |
| T3 | FR-3 | DD-6 | AC-5, P-3 |
| T4 | FR-2, FR-5, FR-9 | DD-4, DD-5 | AC-2, AC-4, P-4, P-8 |
| T5 | FR-6 | DD-8 | AC-7, P-5 |
| T6 | FR-7 | DD-9 | AC-4 |
| T7 | FR-8, FR-10 | DD-10 | AC-9, AC-10, P-6, P-7 |
