# Spec — dim the non-playable map border

## Problem and current behaviour

Every map has a playable region smaller than its tile grid: a fixed margin of **8 cells** surrounds it
on all four sides, stamped by the map→simulation ingest and carried by no ALM record. The interactive
viewer draws that margin at full brightness, identical to the play area, so the screen gives the player
no way to see where the map they can use actually stops. This story dims it — **darker, and never
black**.

The margin is already impassable: the derivation that builds the simulation's block plane counts it
among the arms that block a ground mover. Nothing about enterability moves here. What moves is one
number per tile corner, in the interactive viewer only.

## The margin

For a `Width x Height` map, a **derived block plane** carries one byte per cell, row-major from (0,0).
Two bits of that byte are defined: **bit 0** blocks a ground mover, **bit 1** blocks an air mover. The
margin is bit 1's only writer, so within a map's own plane:

```text
cell (col,row) is in the margin   iff   block[row*Width + col] & 0x02 != 0
```

That equivalence is the whole of how the drawing tier learns the margin. **The margin's width is not
named anywhere in the drawing tier, or in any tier below the one that derives the plane.** It is one
constant in one place in the program, and a correction to it must reach the picture with no edit
elsewhere.

A map no wider than twice the margin depth on either axis has **no playable region at all** — the two
sides' rings meet — and every one of its cells is in the margin. This falls out of the arithmetic that
builds the plane and is not special-cased.

## The brightness this composes with

The viewer's terrain brightness is a per-corner multiplier. A lit cell's four corners take
`m(level) = (96 - level)/32` of the cell's four corner relief levels — **3.0 at level 0, 1.0 at level
64, 1/32 at level 95** — and the quad's interior is interpolated between them. An unlit viewer uses the
neutral multiplier `1` at every corner. A cell whose tile word resolves to an absent tileset slot draws
a solid **placeholder fill** and takes the neutral `1` as well: it is a diagnostic, not terrain.

Because that multiplier exceeds 1 over most of its range, **no ordering between cells is available**: a
dimmed bright cell can be brighter than an undimmed dark one, on the shipped level range as well as in
principle. This story therefore claims a per-cell property only — a margin cell is strictly darker than
the same cell undimmed — and claims nothing about how it compares to any other cell.

## Functional requirements

- **FR-1 (the margin predicate, at the tier that draws)** — A map's cell layers gain the derived block
  plane as an optional layer beside the tile, altitude and object grids, and a **pure, total** predicate
  answers whether a given cell is in the margin, by the bit-1 test above and by **no depth constant of
  its own**. It is total in the permissive direction: an absent plane, a plane shorter than the cell
  asked for, a non-positive dimension, or a cell outside the grid all answer **false**, so a caller that
  supplies no plane sees exactly the picture it saw before the layer existed.

- **FR-2 (the plane reaches the viewer)** — The single load path that turns map bytes into a running
  viewer — the one both the game and the standalone developer viewer come through — fills that layer
  from **the same derivation the simulation's world builder calls**, over the map it has just decoded.
  It is filled unconditionally, as the object grid is: which layers a viewer sees depends on what the
  map contains, not on what the caller asked for.

- **FR-3 (the dim, and the one answer it does not touch)** — The viewer's single per-tile corner-scale
  source returns, for a cell in the margin, each of its four corner multipliers scaled by a named
  constant; for every other cell it returns exactly what it returns today. It applies to **both**
  terrain answers — the lit one and the neutral unlit one — so the margin is marked whether or not the
  light is on, since the dim reports where the player may go and is not lighting. It does **not** apply
  to the **placeholder** answer, which stays at full brightness as the diagnostic it already is.

- **FR-4 (the dim's bound, and that it composes)** — The constant is strictly between 0 and 1: at 1 the
  story does nothing, and at 0 the margin is black, which is the one outcome ruled out. It **multiplies**
  the corner multipliers rather than replacing them, so a lit margin cell keeps its own relief one shade
  darker instead of flattening to a single value. Because every corner multiplier is positive and the
  constant is positive, a dimmed corner is strictly smaller than its undimmed self and strictly greater
  than zero, whatever the light did first.

  *Folded from hotfix `7e4245c` — see `docs/hotfix/ARCHIVE.md#7e4245c`.* The constant is **0** and
  the map's margin is black (owner as author, 2026-08-01). The strictly-between bound, the
  multiply-so-relief-survives decision plan `DD-4` states, and `AC-3`'s reading of both are void
  rather than retuned.

- **FR-5 (how many cells are dimmed)** — The viewer reports the **count** of cells in the margin, as a
  read-only accessor beside the ones that report its overlays. It answers 0 for a viewer whose grid
  carries no plane. It is counted by asking the FR-1 predicate cell by cell rather than by reading the
  plane a second way, so it reports what the draw path will do rather than a second opinion about it.

- **FR-6 (what does not move)** — No simulation state, digest, byte form, decoded format or map record
  changes. No file under the determinism wall is opened. The PNG export path is not dimmed and is not
  touched. No front-end's summary line gains a token, and no command-line flag is added. The margin's
  width gains no second constant at any tier.

## The boundary is a hard step, on purpose

A margin cell and its playable neighbour share an edge, and because the dim is decided **per tile** that
edge is a hard brightness step: the two quads carry different values at the same geometric position. The
renderer interpolates everything else it draws, so the step is conspicuous — and it is the deliverable
rather than a rough edge. The picture exists to show *where* the playable region stops, and a ramp
blurs the one line it is drawing. A gradient is a non-goal, not an omission.

## Acceptance criteria

Synthetic throughout: every block plane and every map below is built in test code from byte literals or
a synthetic stream. No game install is read and no window is opened.

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | a plane with bit 1 set on a ring and bit 0 set on an interior cell as well | the predicate over every cell | exactly the bit-1 cells answer true; the bit-0-only interior cell answers **false**; a plane whose ring meets itself answers true everywhere |
| AC-2 | an absent plane, an empty one, one shorter than the extent, a cell outside the grid, a non-positive dimension, the zero grid | the predicate | **false** in every case, no panic and no out-of-range index |
| AC-3 | a lit map with relief, some cells in the margin | the corner-scale source on a margin cell | each of the four corners equals its own undimmed value times the constant; each is strictly smaller than that value and strictly greater than 0; a cell whose corners disagreed still has four disagreeing corners |
| AC-4 | the same viewer | the corner-scale source on a cell outside the margin | exactly the value it answered before this story |
| AC-5 | a viewer that is not lit — no usable altitude layer, and separately the unshaded diagnostic on | the corner-scale source | a margin cell answers the constant on all four corners; a cell outside answers 1 on all four |
| AC-6 | a lit map whose margin holds a cell resolving to an absent tileset slot | the corner-scale source on it, and on its margin neighbour | the placeholder answers 1 on all four corners; the neighbour is dimmed, so the dim is running |
| AC-7 | a grid carrying **no** plane, and separately one whose plane marks nothing | the corner-scale source over every cell | the pre-story value on every cell, in both cases |
| AC-8 | a synthetic map loaded through the one load path, at three shapes including an oblong and its transpose | the viewer's margin count | the closed form `W*H - (W-2d)(H-2d)` for the derived depth `d`, and equal to the simulation plane's own margin count over the same decoded map |
| AC-9 | a synthetic map no wider than `2d` on either axis, loaded the same way | the viewer's margin count | `W*H` — every cell is in the margin |

## Properties

- **P-1** — The predicate and the corner-scale source stay pure: no graphics context, no window, no
  clock, no file. The dim reads the cell's grid position and the plane, and nothing else — not the
  render mode, the zoom, the camera or the animation phase.
- **P-2** — No simulation, determinism, digest or decoded-format surface is touched by this story.
- **P-3** — The margin's width is named in **exactly one place** in the program, at the tier that
  derives the plane. No tier below it restates the number.

## Out of scope

- **A gradient at the boundary** — a hard per-tile step, as above.
- **Anything but terrain** — objects, structures, unit sprites and diagnostic overlays do not route
  through the corner-scale source and stay at full brightness even standing inside the margin.
- **The PNG export** — it composites through the render tier directly and sets no plane, so it is
  undimmed by construction. Seeing the full grid is what that tool is for.
- **A summary token or a flag** — the front-ends' summary composition is pinned character-for-character
  by three earlier stories' criteria, and this story does not open that.
- **Making the margin more or less enterable**, and any simulation, passability, culling or codec change.
