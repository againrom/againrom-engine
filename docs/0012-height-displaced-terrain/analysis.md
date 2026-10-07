# Analysis — height-displaced terrain geometry

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** |
| Terrain — the whole-map compositor and the tool that drives it | **brownfield** |
| Terrain — the projection, the edge walk and the span resample | **greenfield** |

The profile's default for rendering work is `spec-first / static`; this is a recorded step-up. The
change redefines the geometry five shipped stories already produce output through (0004, 0006, 0007,
0008, 0009) and a planned one extends (0015), and it is not reversible by a flag — the flat raster
survives only as a diagnostic. `static` because no watcher tool exists in this repo.

Both terrains coexist deliberately: the existing draw loops have behaviour to preserve and are
pinned before they change; the projection and raster arithmetic is new code and is driven red-first.

## What we did not know

The item was imported from a baseline written when the project had **no** game evidence for terrain
geometry. Three of its foundations do not survive contact with the pinned research, and each fails
in a different way:

1. **The displacement direction was third-party.** The baseline took "elevated terrain moves up"
   from two consulted reimplementations of a related title and said so, flagging it unresolved. Our
   golden rule 4 forbids the source outright, so the question was never *how confident is that* —
   it was *do we have it from the game*. We now do (`TERR-GEOM-031`), independently, and the
   earlier basis is not carried forward in any form.
2. **The raster contract was our own invention.** `edge(left,right,sx) = left + roundNearestAway((right-left)*sx/32)`
   and `sourceY = floor((py-top)*32/span)` were deterministic engineering, explicitly disclaimed
   against the original. The engine's actual rule is now decoded — a startup-built step table, two
   differently-oriented edge walks, and a 16.16 DDA — so keeping the invention would have been
   deliberate infidelity with a live alternative in hand.
3. **The framing was inverted.** "Adds an explicit displaced mode", flat by default, treats
   displacement as an enhancement. On the shipped corpus the overwhelming majority of cells take
   the sloped path (`TERR-GEOM-032`), so flat-by-default is not a conservative baseline — it is a
   renderer the engine does not contain.

Two things we still do not know, and are not resolving here: what the engine does when the height
delta across a cell edge exceeds the step table's last row, and why the drawn edge and the engine's
own hit-test edge disagree by up to three rows. Both stay open, and neither blocks the contract.

## What we looked at

- The pinned `research/` submodule, frozen for this story: `TERR-GEOM-031...036` for the geometry,
  `TERR-EDGE-024...026` for the far edge, `TERR-LIGHT-011` (amended) for how shading rides on a
  stretched span, `TERR-LIGHT-028` for height signedness.
- `claims/retracted.md` **before** leaning on any of them. It is the reason two of those rows are
  read narrowly: `TERR-LIGHT-011` carried High while describing only half of what the blitters do,
  for twelve experiments, and `TERR-LIGHT-016`'s height figures are withdrawn — so *how much*
  relief the shipped maps carry is not something this story may assert from that row.
- The shipped code (below), and the imported baseline.

## Three-phase context build

**Architecture.** The render tier takes decoded primitives only — a tile-word grid, a height grid,
32x32 paletted sub-cells, integers — and never sees an archive or a map file. One tool wires the
formats tier to it and writes a PNG; the interactive viewer resolves and draws cells on its own path
and does not call the compositor at all, which is why that path can be left alone here.

**Module.** The terrain package holds seven concerns: BMP decode, strip slicing, tile-word mapping,
water phase, relief levels, shading, and two whole-map draw loops — an unshaded one and a shaded one
that differ only in the per-pixel colour step. Marker overlays run *after* a finished image and take
`(cell, cols, rows, cellPixels)`.

**Detail.** Both draw loops allocate `W*32*scale x H*32*scale`, iterate cells row-major, resolve the
word, place a 32x32 sub-cell at `(col*32*scale, row*32*scale)` and replicate each source pixel
`scale x scale`. Heights reach only the level grid, which already reads them as **signed** bytes —
the one thing the geometry needed that the code already had right. The shaded loop interpolates the
four corner levels bilinearly over the 32x32 *source* cell and clamps the far-edge `+1` vertex into
the grid.

## Implicit assumptions the change breaks

Hunted deliberately, because these are where a regression would hide:

- **Output height is `H*32*scale`.** True today by construction, assumed by the tool's summary line
  and by tests that compute expected bounds arithmetically. It stops being true.
- **Every output pixel is covered.** Nothing today produces an uncovered pixel, so no test or caller
  has ever seen one and the image is opaque everywhere. Displacement produces uncovered pixels above
  and below the terrain and, rarely, inside it.
- **A cell's pixels are a function of that cell alone.** Cells now overlap, so paint order becomes
  observable rather than incidental.
- **A marker's position follows from `(col, row)` and the cell size.** It still does — but it no
  longer follows to *terrain*: the cell lattice and the drawn quads part company.
- **Recorded evidence stays valid.** The MD5s in 0004's, 0006's, 0008's and 0009's verification
  evidence were taken on the flat raster and stop matching the moment the default changes. That is
  evidence to re-run, not a regression to chase.
