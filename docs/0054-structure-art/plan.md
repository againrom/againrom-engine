# Plan — a cell for an anchor, a strip for a sprite

## Shape

Three tiers do the three halves they can each do. `pkg/render/terrain` is standard-library-only and
is where both renderers already look, so the bundle types, the grid arithmetic, the strip and the
placement builder live there. Opening an archive, parsing a registry and decoding a sheet need the
container, format and data tiers, so the loader lives in `pkg/game`. Drawing lives where drawing
already lives. No package is added and no allow-map edge changes, so `internal/archtest` and
`docs/ARCHITECTURE.md` are untouched.

**Ordering.** T1 builds the pure geometry and draws nothing; T2 puts buildings in the window. That
ordering is forced rather than chosen: the render tier cannot open an archive, so its types must
exist before a loader can fill them, and both renderers place from one builder (P-7), so the builder
is the thing every wiring needs. Nothing before T2 produces a converter, a dump or a census — the
census is last, at T7, and the first task that draws anything draws it on screen.

## DD-1 — the bundle types and the arithmetic go in `pkg/render/terrain`, the loader in `pkg/game`

The split 0017 established for object art, for the same reason and with the same boundary: the
types sit where both renderers reach, and the decoding sits at the tier that may import everything.
New files `pkg/render/terrain/structures.go` and `pkg/game/structures.go`; nothing else gains a
package. Serves FR-1, FR-3.

## DD-2 — one entry per image row, and the entry has no anchor field

`StructurePlacement` carries `Cell`, `TopLeft`, `GridIndex`, `Class` and `Frame` — and no `Anchor`.
That absence is the story's central finding made structural: there is no expression in this package
into which a canvas or an anchor pixel could enter, so P-2 is enforced by the type rather than by a
test that has to keep noticing.

The unit of the list is the **blit**, not the cell: a rectangle cell of `n` image rows contributes
`n` entries. That is what lets the per-counter pass patch entries in place exactly as `AnimateStatics`
does, with no re-anchoring at all — the anchor being a cell, `TopLeft` cannot move when the frame
changes, which is strictly simpler than the object side and is why FR-5 can forbid moving it.

`Frame` is `*StaticFrame`, the object layer's own frame type. The window's texture cache keys on that
pointer, so structure art uploads through the existing cache with no change to `pkg/ui`'s cache, and
both blits and the sprite shading ladder serve unchanged. Serves FR-3, FR-5.

## DD-3 — the bundle is keyed by the placement key's low byte

`StructureSet{Classes [256]*StructureClass}`, mirroring `StaticSet`. `pkg/game` walks 1..255 through
`pkg/data`'s `ByID` and fills what resolves, so the registry's identity convention stays in the
package that decoded it and the per-placement lookup downstream is an array read that cannot go out
of bounds. A key above 255 is masked to its low byte at the one conversion site, which is also where
a record's `(X, Y)` becomes its anchor cell. Serves FR-1.

## DD-4 — every per-class derivation is computed once, at load

The live-cell count, the per-grid-index rank and the three block bases are functions of the class
alone, so the bundle carries them precomputed: `Rank []int` of length `TileWidth*FullHeight`, holding
the live rank of each grid index and `-1` for a cell the mask retires. A dead cell and a live cell
are then one array read apart at draw time, and the mask string is never re-scanned per frame.

The mask's length is validated here and nowhere else. It fails the whole animation gate rather than
being truncated or padded, because a mask of the wrong length has no correct interpretation: it
mis-ranks every live cell after the first divergence and draws wrong frames while failing nowhere.
Serves FR-2.

## DD-5 — the timeline expansion is `pkg/data`'s rule and stays there

`(*StructureClass).Timeline()` joins `(*ObjectClass).Timeline()` in `pkg/data/anim.go`, reusing the
existing `expandTrack`. It is a pure derivation over two fields the type already carries; it adds no
key row, touches no default table, and is not the defaults repair.

**Named dependency.** `pkg/data`'s `structureDefaults` is nil under a comment saying the registry's
unset-scalar defaults are not decoded. At this pin they *are* decoded, so that comment is stale and
an omitted key currently resolves to the Go zero rather than to the registry's own default. Repairing
it is a separate story off 0016. This story does not wait for it and does not work around it: FR-9
requires every scalar read here to decide the same thing at both values, which holds because the
extents and `Phases` are refused or fail their gate at either, and `Indestructible`, `Flat` and
`VariableSize` have the same value at either. The executor asserts that rather than assuming it.
Serves FR-2, FR-9.

## DD-6 — the lift takes corner heights, not a per-cell height

`StructurePlacements` takes `corner func(c, r int) int` — `Projection.Altitude` — and not the
`lift func(col, row) int` the object builder takes. A bilinear sample at the rectangle's centre falls
on a half-integer whenever an extent is odd, and `AnchorHeight` can only answer for a whole cell, so
the per-cell helper cannot express it. Nil is the flat geometry, as it is on the object side.

It is evaluated **once per structure** and stamped into every entry of that structure, which is what
makes P-3 checkable: a per-entry evaluation would let two cells of one building disagree about their
own ground. All four samples go through one integer expression with truncating division, so the
odd/odd case reduces to the centre cell's four-corner mean by arithmetic rather than by a special
case. Serves FR-3.

## DD-7 — order is a stable sort, once, in the builder

Entries are produced per placement in map record order and then sorted **stably** by row ascending
and column descending. Stability is what makes "ties in record order" true without a tiebreaker
field, and doing it in the builder is what keeps the two renderers from each having an opinion.

Row order is load-bearing and column order is not: strip frames are one cell wide so two columns
never overlap, but a back row's overhang covers the rows in front of it, so a structure whose art
reaches up must be drawn after everything on the rows it reaches over. Column order is reproduced
anyway because it is free and it is what the original walks. Serves FR-4.

## DD-8 — the two passes, and the merge that is ours

The flat pass is decoded and is reproduced exactly: `Flat` non-zero draws before everything else on
the map. The main pass is decoded to run immediately before the unit plane at the same cell, and
this tree has no per-cell interleave with entities, so it is placed where the tree's object art
already sits — before the entity passes — and the consequence is disclosed in the contract.

Against the **static-object** layer the arrangement is **ours by choice**: nothing at this pin places
the type-3 object plane in the original's cell walk. Both lists are already row-sorted, so a
two-pointer merge by rectangle row is the arrangement that keeps back-to-front order between two
cell-anchored planes; any fixed order puts one plane permanently in front regardless of row. The
merge is a helper in the render tier that both the window and the raster harness call, so the choice
exists once. Serves FR-6.

## DD-9 — destruction is a draw-time mode, not a field

`AnimateStructures(dst, places, animated, counter, animate, ruined bool)` mirrors `AnimateStatics`.
`ruined` walks every entry, as `!animate` does on the object side, and honours `Indestructible` per
class. No placement, class or bundle carries a destruction state, so nothing can leak into a
simulation later mistaking a diagnostic for a fact. Exposed as a flag on the developer tools only,
never from the game front-end. Serves FR-5, FR-7.

## DD-10 — the census is the builder's, the tools only print it

`StructureCounts` is filled by the builder beside the list it counts, so a printed number is the
number that build produced rather than a second walk's opinion of it. The per-placement and per-cell
levels are separate structs so neither can be added to the other by accident. `cmd/mapview -check`
and `cmd/terraintool` print it; `AC-10`'s three observations are computed by the harness from the
same bundle. Serves FR-8.

## DD-11 — fixtures are synthetic and built in test code

`internal/synth` gains a structure-sheet builder emitting a `.256` of `n` frames at a stated size,
and a structure-registry text builder. No test reads an install. A sheet with a non-square frame and
a sheet short of its own blocks are ordinary fixtures, because the exclusions and the fallbacks are
contract rather than defence. Serves FR-10.

## Risks

- **R-A** The registry rectangle and the definition table's may disagree on shipped classes, which
  would put a building's art and its footprint on different cells. This story reads only the
  registry, so it cannot be wrong about its own rectangle; the disagreement is measured and reported
  by AC-10 rather than reconciled, and a reconciliation belongs to whichever story holds both.
- **R-B** A wooden bridge draws nothing, and one is placed 8 times over the shipped maps. The
  absence is visible and counted rather than silent, and R-1 says what would close it.
- **R-C** Merging two planes changes an existing draw order, so a map with no structures must be
  proved unchanged. P-5 and AC-9 exist for that and are the reason the zero bundle is a first-class
  case rather than a guard.

## Success criteria

- **SC-1** Every shipped map censuses through the harness with no error and both count levels
  recorded, and the per-placement counters sum to the map's placement count.
- **SC-2** A rendered map carrying buildings is produced from a lawful install and shipped as an
  owner-review artifact — the picture, not a count.
- **SC-3** With no bundle, every shipped map's rendered bytes, placement lists and counts equal the
  pre-story ones exactly.
- **SC-4** The window and the raster harness produce the same structure geometry for one map, checked
  as the list they both place from rather than as two pictures.
- **SC-5** The mutants each task names are applied where that task wrote them and are killed by that
  task's own tests.
- **SC-6** AC-10's three observations are recorded as figures with the corpus they came from, and
  none of them is reconciled into a contract clause.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-3 | SC-1 |
| FR-2 | DD-4, DD-5 | SC-5 |
| FR-3 | DD-1, DD-2, DD-6 | SC-4 |
| FR-4 | DD-7 | SC-4 |
| FR-5 | DD-2, DD-9 | SC-5 |
| FR-6 | DD-8 | SC-2, SC-3 |
| FR-7 | DD-9 | SC-2 |
| FR-8 | DD-10 | SC-1, SC-6 |
| FR-9 | DD-5 | SC-5 |
| FR-10 | DD-11 | SC-3 |
