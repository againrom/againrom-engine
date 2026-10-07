# Spec — a placed structure is drawn

## Problem and current behaviour

A map's placed structures reach the screen as a yellow cross and nothing else: `pkg/data` decodes all
66 structure classes, one developer tool is the only caller, and no sheet is opened, no frame
decoded, no pixel drawn. The object and unit art already in the tree is placed from a class **canvas**
and the frame's own **anchor pixel**, and a structure class declares neither.

Structure **footprints on the block plane** are a separate delivery. Nothing here reads a definition
table, resolves a footprint or touches a passability byte.

## The placement

A **placement** is one type-4 record: a fixed-point anchor `(X, Y)` in 1/256 of a cell, and a key.
Its **class** is the key's **low byte**, an identity in the structure registry whose domain is
`1..66`; a key naming no loaded class draws nothing. Its **anchor cell** is `(X >> 8, Y >> 8)` — no
origin subtracted, no inset, no rounding — and that cell is the art rectangle's **top-left**: minimum
column, minimum row.

**There is no anchor pixel and no canvas.** A structure is positioned by that cell alone.

## The art rectangle

The class declares `TileWidth` and `TileHeight`, and the art covers exactly that rectangle of cells
from the anchor cell, running right and down — whatever any other table says about which of those
cells a building occupies.

A placement may carry an eight-byte **extension** whose two extents reach a **different** rectangle:
the footprint on the block plane, which this contract does not read. Its discriminator is the key
`0x21`, whose class is one of the two FR-1 excludes for `VariableSize`, so no placement drawn here
carries one.

## The sheet and its grid

`File` is a **path**, backslash-separated and extensionless; the sheet is that path under the
graphics container's `structures` directory with `.256` appended, addressed lower-cased whole. This
registry has no file table and no inheritance: a class carries its own path and resolves alone.

The sheet is a **row-major grid of tile-sized frames, `TileWidth` columns wide and `FullHeight` rows
tall**. Image cell `(k, c)`, `k` in `[0, FullHeight)` and `c` in `[0, TileWidth)`, has **grid index
`k*TileWidth + c`**. Every frame is exactly one map cell, `CellSize` square.

## The strip

A structure is drawn **once per rectangle cell, and each of those draws is a vertical strip of image
rows** — not one image, and not one image per cell. For the rectangle cell whose local position is
`(COL0, ROW0)` = `(col - anchorCol, row - anchorRow)`:

```text
rowTop = ROW0 - TileHeight + FullHeight
limit  = rowTop        when ROW0 != 0
limit  = 0             when ROW0 == 0          the back row also draws the overhang

for k = rowTop down to limit:
    grid index = k*TileWidth + COL0
    destX      = col*CellSize
    destY      = row*CellSize - lift - originY - (rowTop - k)*CellSize
```

`destX`/`destY` are the frame's **top-left** in world pixels; `originY` is the render's own vertical
origin, 0 in the flat geometry. The grid's bottom `TileHeight` rows map one-to-one onto the
rectangle's rows; its top `FullHeight - TileHeight` rows — the **overhang** — stack above the back
row. Where `FullHeight == TileHeight` the strip is one frame per cell.

## The three blocks

`frames` is the sheet's own frame count, `TW`/`FH` the grid extents, `L` the count of `AnimMask`
bytes that are not `-`, and `rank(i)` how many such bytes precede grid index `i`.

| Block | Range | Frame for grid index `i` |
|---|---|---|
| base | `[0, TW*FH)` | `i` |
| animation | `[TW*FH, TW*FH + P*L)` | `TW*FH + (phase-1)*L + rank(i)` |
| ruin | `[frames - TW*FH, frames)` | `frames - TW*FH + i` |

The **ruin block is addressed from the end of the file**; nothing stores its base. A `phase` of 0, a
grid cell whose mask byte is `-`, and a class that does not animate all take the **base** frame. The
ruin block is taken only when the structure is destroyed **and** `Indestructible` is 0.

## Animation

A class animates when **all** hold: `Phases > 1`; the run-length expansion of `AnimTime`/`AnimFrame`
— `AnimFrame[i]` appended `AnimTime[i]` times, an entry of non-positive time appending nothing but
still consuming its round — is non-empty; and `AnimMask` is non-empty with length **exactly
`TileWidth * FullHeight`**. The mask pictures the **sheet grid**, not the rectangle, and is indexed
by the grid index.

`phase` is `timeline[counter mod len(timeline)]`. Every animating structure on a map advances
together: there is no per-cell and no per-placement stagger.

## The lift

The lift is **one value for the whole structure**, not one per cell: a bilinear sample of the
terrain's corner heights at the **rectangle's centre**. With `x2 = 2*anchorCol + TileWidth` and
`y2 = 2*anchorRow + TileHeight` it is the mean of the corner heights at `(x2/2, y2/2)`,
`((x2+1)/2, y2/2)`, `(x2/2, (y2+1)/2)` and `((x2+1)/2, (y2+1)/2)`, every division truncating toward
zero. Where both extents are odd those four samples are one cell's four corners.

It is **subtracted**, so higher ground raises a structure up the screen, and the displacement is
vertical only: `destX` carries no term from it.

## Functional requirements

- **FR-1** A loader MUST turn the graphics container's structure registry into a bundle keyed by the
  **placement key's low byte**, so nothing downstream learns the registry's identity convention. A
  class whose sheet is absent, undecodable or palette-less, whose sheet holds a frame that is not
  `CellSize` square, whose extents are not all positive, or whose `VariableSize` is non-zero MUST
  resolve to a class that draws nothing, counted apart — never to a missing class, never to a failed
  load. Only an unreadable or unparseable registry is an error.
- **FR-2** A class meeting the animation preconditions MUST carry its expanded timeline, live-cell
  count and per-index rank; one failing any of them MUST carry an **empty** timeline. A mask of the
  wrong length MUST fail that gate rather than be truncated, padded or indexed.
- **FR-3** A pure builder MUST yield one entry per **image row of each rectangle cell**, per *The
  strip*, carrying its world-pixel top-left, grid index, rectangle cell and class, and **no anchor
  pixel**. A cell outside the map extent MUST contribute no entry.
- **FR-4** The list MUST be ordered by rectangle cell — rows ascending, columns **descending**, ties
  in map record order. Order within one cell's strip is free.
- **FR-5** Selection MUST follow *The three blocks*, and an index outside the sheet MUST fall back to
  that grid index's base frame. A per-counter pass MUST re-select a frame **without moving its
  top-left** and MUST walk only entries that can change.
- **FR-6** The plane MUST draw in **two passes**: `Flat` non-zero **before** the static-object layer,
  `Flat` zero **merged with** it in rectangle-row order. The windowed viewer and the raster harness
  MUST both draw the one builder's output unchanged.
- **FR-7** Two diagnostics MUST exist, each ours: a **ruin** draw of every drawable structure,
  respecting `Indestructible`; and the existing anchor cross, which MUST keep deriving its cell from
  the record alone and MUST NOT be routed through this story's geometry.
- **FR-8** A census MUST report two levels with disjoint counters. **Per placement:** drawn, no
  class, undrawable, variable-size. **Per cell:** strips, frames, of those the overhang frames, and
  cells dropped outside the extent. It MUST be reachable from the viewer's headless check and from
  the raster harness.
- **FR-9** Every scalar key this contract reads MUST decide the same thing at the value an omitted
  key resolves to today and at the registry's own decoded default, so nothing here moves when that
  default is corrected elsewhere.
- **FR-10** All of the above MUST run headlessly against synthetic registries, sheets and maps, with
  no install and no window. With **no** bundle, a loaded map's picture, placement lists and counts
  MUST be byte-identical to what that map produced before this story.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | seven classes: whole; absent sheet; palette-less sheet; a 32x31 frame; a zero extent; `VariableSize` set; `ID` above 255 — each scalar key also omitted, at both the value the tree resolves today and the registry's own default | loaded | the first draws; the next five resolve, draw nothing, counted apart; the last is reachable by no key; only an unparseable registry errors; and the two default values give identical bundles |
| AC-2 | unit | `Phases` 1 with a mask; `Phases` 3 with a timeline and a mask of length `TW*TH != TW*FH`; `Phases` 3, a timeline, no mask; all three present | loaded | only the last carries a non-empty timeline; the wrong-length mask is refused whole, neither truncated nor padded |
| AC-3 | unit | a `3 x 2` class of `FullHeight` 5; one whose `FullHeight` equals its `TileHeight`; one rectangle overrunning the last column and row; one anchored wholly outside | built | the first two match *The strip* cell for cell and frame for frame, the second yielding one frame per cell; out-of-extent cells are dropped and counted with no cell of the next row touched; the outside placement yields nothing |
| AC-4 | unit | a sheet of `TW*FH*2 + P*L` frames, a mask mixing `-` and live bytes, over two whole cycles; then sheets short of the animation and of the ruin block | selected | every arm of *The three blocks* is taken exactly when its stated condition holds, at every counter; `Indestructible` returns a destroyed selection to base; every short-sheet selection lands inside the sheet at that index's base frame |
| AC-5 | unit | distinct corner heights; classes `3 x 3`, `4 x 3`, `4 x 4` at one anchor | the lift is taken | each equals *The lift*, the odd/odd case being the centre cell's four-corner mean; every entry of one structure carries the **same** lift; each `destX` is the flat geometry's |
| AC-7 | unit | two structures on adjacent rows whose art overlaps, one `Flat` and one not, beside a static-object layer | built and drawn | the flat one precedes every static object; the other falls in rectangle-row order among them; within a row columns descend; equal cells keep record order |
| AC-8 | unit | each scalar this contract reads, omitted, at both the value the tree resolves today and the registry's own default | loaded and built | the two give identical bundles, lists and counts |
| AC-9 | unit | one map with no bundle, one with a nil bundle, one with a bundle no placement resolves in | loaded and drawn | all three give the pre-story picture, lists and counts byte for byte, and the census reports both levels with no install |
| AC-10 | corpus | every shipped map and the graphics container, from a lawful install | censused | every map censuses without error, both levels are recorded, and three observations are reported and not reconciled: which class ids carry `VariableSize`, how many placements carry an extension, and on how many classes the registry rectangle differs from the definition table's |

Error cases — no class, an undrawable class, a variable-size class (AC-1), a wrong-length mask
(AC-2), a cell outside the extent (AC-3), an index past the sheet (AC-4) — are each a **skip**.
AC-6 and AC-8 were folded into AC-3 and AC-1; their ids are retired, not reused.

## Derived properties

- **P-1** (invariant) The builder is pure and total: any map, bundle and geometry yield a list,
  nothing is mutated, and the same inputs always yield the same list.
- **P-2** (negative-invariant) No entry carries an anchor pixel and no `destX` depends on anything
  but its column, at any class, sheet or geometry.
- **P-3** (negative-invariant) One structure's entries all carry one lift, and a map's flat and
  displaced lists differ by a vertical translation alone.
- **P-4** (negative-invariant) No selected index falls outside the sheet, at any counter, phase, mask,
  frame count or destruction state.
- **P-5** (negative-invariant) With no bundle, no resolvable class, or only undrawable classes, every
  byte of the picture and of the pre-existing placement lists is what it was before this story.
- **P-6** (completeness) Drawn plus the per-placement skips equal the placement count; strips plus
  out-of-extent drops equal the cells of every drawn placement's rectangle.
- **P-7** (invariant) The viewer and the raster harness draw one map from one list, so no second
  placement path exists that could disagree with the first.
- **P-8** (negative-invariant) For every key read, the two candidate values of an omitted key give the
  same bundle, list and census.

## I/O examples

```text
mapview -assets <root> -map <name> -structures -check
structures: 137 drawn, 0 no class, 0 undrawable, 2 variable-size
cells: 402 strips, 913 frames (511 overhang), 6 outside the map
```

## Constraints and alternatives

Selected, each *disclosed* as a named divergence, with its observable cost: **no shadow** — a
building casts none; **no `b` overlay pass** — a structure shipping an overlay layer loses it; **art
draws before every unit and entity whatever their rows** — a unit on a row behind a building draws
in front of it, which is the tree's existing arrangement for object art rather than a new divergence
class; **the phase is not gated on visibility** — every animating structure cycles always; **the two
`VariableSize` classes draw nothing** — two bridge classes absent, one of them placed on shipped
maps. Also selected: **destruction is a diagnostic, not a state**.

**Named seams (G2).** Three declined lifts. None would change a shipped byte; each changes what
shipped bytes **mean**, so each must be lifted by addition rather than by reinterpretation:

1. **The ruin block is addressed from the end of the sheet.** Appending a frame to a shipped sheet
   moves the ruin grid. An extended form must carry an explicit ruin base as **additional** data
   defaulting to `frames - TileWidth*FullHeight`.
2. **`AnimMask`'s length is exactly `TileWidth * FullHeight`.** A precondition, not a hint: any other
   length mis-ranks the live cells and draws wrong frames while failing nowhere. A longer mask must
   be additional.
3. **Every frame of a structure sheet is `CellSize` square.** A frame *is* a cell, which is what makes
   the anchor a cell; any other size is refused rather than mis-drawn.

## Out of scope

- The shadow pass, the `b` overlay pass, and per-owner colour — a structure's world sprite takes
  none, which is decoded rather than omitted.
- The structure as a **simulation entity**: health, destruction, selection, usability, light radius
  and pulse, the shop branch, the minimap blip.
- The **footprint on the block plane** and the definition table; the two `VariableSize` classes'
  nine-patch selector; the fog of war and its two gates.
- **Correcting the registry's unset-scalar defaults**, which are decoded and which this tree does not
  yet apply. FR-9 makes this contract indifferent to that correction; it does not make it.
- Any change to terrain, object or unit art, the marker overlays, registry parsing or sheet decoding.

## Research needed

- **R-1 — which frame of a wooden bridge's sheet is which patch of its nine-patch selector?** The
  selector, its two frame counts and its inputs are established; the mapping from rectangle position
  to frame index is not. *Until resolved:* the two classes draw nothing.

## Verification mapping

AC-1 to AC-5, AC-7, AC-9 and P-1 to P-8 are CI-automatable headlessly. AC-10 needs a lawful install;
counts recorded, no game bytes committed.

Gate coverage: FR-1 to AC-1, AC-9 · FR-2 to AC-2 · FR-3 to AC-3, P-1, P-2 · FR-4 to AC-7, P-6 ·
FR-5 to AC-4, P-4 · FR-6 to AC-7, P-7 · FR-7 to AC-4 · FR-8 to AC-9, AC-10, P-6 · FR-9 to AC-1,
P-8 · FR-10 to AC-9, P-5. The lift: AC-5, P-3.
