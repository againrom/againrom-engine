# Analysis — a structure is not a sprite standing on a cell

| Axis | Declared |
|---|---|
| Intensity | **spec-anchored / static** — the sheet-grid model is a decode contract that outlives this delivery: the shadow, the `b` overlay, the ruin arm's live input and the two bridge subclasses are all deferred into it by name. No watcher exists, so it is discipline |
| Terrain | **greenfield** for the new bundle, loader and builder; **brownfield** for the windowed viewer's draw sequence and the raster harness's, where a map loaded with no structure bundle must produce the picture it produces today |

## What we did not know

**Whether the placement code already in the tree could serve.** It cannot, and the reason sits at
the anchor rather than in a detail. Object and unit art is positioned by a class **canvas** and the
frame's own **anchor pixel** — `StaticAnchor`'s ten positional integers, four of which a structure
class simply does not declare. A structure's anchor is a **cell**, its destination is that cell's
top-left screen pixel, and one frame of its sheet **is** one map tile. A plan that reused the object
draw path would have had to invent a canvas and a centre for a class that has neither, and would
then have been calibrating an invention against shipped art.

The finding survives into the type system rather than into a comment: this story's placement value
has no anchor field, and no expression in it is one a canvas could enter.

**Where the art's rectangle comes from.** Two rectangles exist for one building and they are
independently sourced. The **art** rectangle is the registry's `TileWidth x TileHeight`. The
**block** rectangle is the definition table's `sizeX`/`sizeY`, which `0051-structure-footprints`
consumes. Nothing at this pin asserts that the two agree: `DAT-BLD-005` establishes that the tables
are one roster re-keyed and that their *names* agree 66 for 66, and says nothing about their extents.

That decides the story's dependency shape. The draw reads the **registry alone** — it opens no
definition table and resolves no footprint, so it is independent of `0051` except for the anchor-cell
rule and the low-byte class key, both quoted into `spec.md` as S-2 requires. Whether the two
rectangles agree over the shipped corpus is a question for the census, not a premise of the contract.

## What we looked at

**The class record.** `pkg/data.StructureClass` already carries every key the draw needs — `ID`,
`File`, `TileWidth`, `TileHeight`, `FullHeight`, `Phases`, `AnimMask`, `AnimTime`, `AnimFrame`,
`Indestructible`, `Flat`, `VariableSize` — and `LoadStructureClasses` already resolves the sprite
path. Two gaps: nothing expands the animation pair into a timeline for a structure, where the object
side has `Timeline()`; and nothing validates the mask's length.

**The sheet.** `pkg/formats/spr256` decodes to a flat `Frames` list and exposes no offset, hotspot
or anchor on a frame — only `Width`, `Height`, `Pixels`. For a structure that absence is exactly
right rather than a limitation: the grid index does all the addressing.

**What carries over unchanged.** `terrain.StaticFrame` and its palette, both blits, the sprite
shading ladder, and the window's frame-keyed texture cache — which keys on `*terrain.StaticFrame`,
so structure art uploads through it with no change to it at all. `Projection.Altitude` gives the
corner heights the lift needs; `Projection.AnchorHeight` averages four of them, which turns out to
be the degenerate case rather than the general one.

**What the tree cannot supply to the decoded frame selector.** Two of its inputs do not exist here.
The **health** that chooses the ruin block is filled from a create message and a definition-table
row, and nothing in this tree gives a structure health at all; the **fog state** that gates both the
phase advance and the whole main draw pass has no model here either. Neither is a gap in the decode
— both are decoded — so the contract states what it does in their absence rather than leaving it to
an implementation.

## Two things the corpus alone would not have told us

The **animation mask's length** is `TileWidth x FullHeight`, and the rival — the footprint's own
`TileWidth x TileHeight` — is not merely unproven but measurably wrong on six of the fourteen classes
that spell a mask. A consumer taking the footprint reading gets a mask too short, ranks the live
cells wrongly from the first divergence onward, and draws the wrong animation frame while failing
nowhere. That is why the length is a validated precondition in the contract rather than an assumption
inside a loader.

The **ruin block is addressed from the end of the file**. Nothing stores its base, so the block moves
when the file's frame count moves. Appending a frame to a shipped sheet is therefore a silent
corruption of that sheet's ruin art — and appending a frame is the first customisation anybody would
try on this format.

## The two classes this story cannot draw

`VariableSize` is not a data flag. The two wooden bridges are separate C++ subclasses whose frame
selector is a nine-patch over their own rectangle, with 9 and 14 frames against a `1 x 1` grid, so
the grid identity does not hold for them and the grid model would draw nonsense. The **index mapping
of that nine-patch is not published** — corner, edge and interior are named; which frame is which is
not — so implementing it here would be guesswork against shipped art. They are excluded by flag and
counted apart, which is a visible absence rather than a wrong picture: one of the two is placed 8
times over the shipped maps and the other is placed on no shipped map at all.
