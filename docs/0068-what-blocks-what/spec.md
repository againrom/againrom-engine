# 0068 — what blocks what, and what stands in front of what

Intensity: rigour **medium**, ambition **low**. Terrain: brownfield — two defects the owner saw on
screen, both in code that already exists and is already correct where it is called from.

## The problem, in the owner's words

Buildings do not collide. The bridge is impassable. Units draw over buildings and trees.

## What is actually wrong

**One cause for the first two.** The block plane a world routes on is the five terrain arms with the
map's placed structures applied on top, and the second stage is driven by the **buildings collection
of the definition table**. The front-end's table load reads the units and humans collections and
**leaves the buildings collection unset**, so on every world the game builds — the map screen's and
a mission's alike — the structure stage resolves nothing and the plane is the ingest plane alone.
Placed structures therefore close no cell, and the subtractive arm that OPENS a bridge deck never
runs, so the deck keeps the blocked terrain the ingest gave it.

**The third is a missing depth rule.** The map's two cell-anchored art planes — structures and
static objects — are merged into one back-to-front order by rectangle row and painted as the frame's
content band. Units are painted **after that whole band**, so a unit is in front of every building
and every tree on the map whatever row it stands on.

## Scope

In: the buildings collection reaching the front-end's table; units joining the cell-anchored draw
plane's row order.

Out: **the decoded per-class draw-layer scheme** — the two drawable classes, their layers 2/3/4, the
corpse arm and the unconditional layer of the flying class. A single row-ordered plane is one rule
for every drawable; the layer scheme is a second, coarser key **above** it, and this story implements
the first and not the second. A flier standing behind a building is therefore drawn behind it here,
which the original would not do. Named, not hidden.

Also out: the viewer's own `terrain.Grid.Block`. It is derived with no table, and this story does not
change that — see FR-6, which states why that is not a second instance of the same defect.

## Functional requirements

**FR-1 — the table carries buildings.** The definition table the front-end loads carries the
archive's **buildings** collection beside the two it already carries. Its absence is not a new
failure mode: the file either parses, in which case every collection exists, or it does not, in which
case the load already fails.

**FR-2 — a placed structure closes what it says it closes.** In a world built from a table carrying
that collection, every cell a placement's footprint attaches and whose blocking bit is SET is closed
to a ground mover, overriding whatever the terrain arms said.

**FR-3 — and opens what it says it opens.** Every cell a placement's footprint attaches and whose
blocking bit is CLEAR is OPENED to a ground mover, overriding whatever the terrain arms said. This
is the requirement the bridge rides on: a deck is blocked terrain in the ingest plane and only this
arm clears it.

**FR-4 — the crossing, and not merely the count.** A map whose ingest plane leaves two ground regions
disconnected, and which places a structure whose attach set spans the gap with its blocking bits
clear, yields a world in which the two regions are **connected**. Counting opened cells is not
crossing, and only the connectivity statement can fail.

**FR-5 — units are in the plane.** A unit's sprite is drawn in the same back-to-front order as the
structure and object planes, by the row of the cell it stands on: **after** every drawable standing
on an earlier row and **before** every drawable standing on a later row. At an equal row the unit is
drawn last, in front. Ground decoration — a structure class the bundle marks flat — stays under
every unit whatever row it stands on.

That order binds units **against each other** as well, and this is a second visible change rather
than a side effect: units were drawn in world-entity order, which is ascending id and says nothing
about rows, so of two overlapping units the one in front was whichever the world happened to hold
first. Two units on one row keep their id order.

**FR-6 — one thing that does NOT change, stated so it is not read as an oversight.** The viewer's
grid holds a block plane derived with no table. Its **one** reader tests bit 1, the air bit, and the
structure stage writes bit 0 alone — so handing that derivation a table would move no pixel and
answer no question. It is left as it is, and this clause is why.

**FR-7 — nothing else moves.** No entity gains a field, no serialized form moves, and `pkg/sim` is
not touched. A world built from a table whose buildings collection holds no rows is byte for byte
the world that was built before this story.

## Acceptance criteria

**AC-1** The loaded table's buildings collection reports the entry count the archive's own rows
describe, and an entry's parameters are the ones written.

**AC-2** A map placing a structure whose blocking bits are set, on ground the arms leave open, yields
a world whose plane closes exactly the attached cells the blocking set names.

**AC-3** A map placing a structure whose blocking bits are clear, on cells the arms close, yields a
world whose plane opens exactly those cells.

**AC-4** Two ground regions the arms leave disconnected are connected in the world built with the
table, and are **not** connected in the world built without it. Both halves are asserted; the second
is what makes the first mean something.

**AC-5** The front-end's map screen and a started mission both get FR-2/FR-3's plane, and neither is
asserted through a private path: each is measured on the world the front-end actually hands out.

**AC-6** A unit standing on a row **behind** a drawable is submitted to the draw target before it,
and a unit standing on a row **in front of** it is submitted after it.

**AC-7** A unit is submitted after every flat structure, whatever the rows.

**AC-8** With no entities held, the sequence of frames submitted for one frame of the art plane is
exactly the sequence submitted before this story.

**AC-9** With no structure and no object bundle, the frames submitted for the entities are exactly
the ones submitted before this story — same rectangles, same cull, same mirror bits — reordered by
row and by nothing else. An entity that resolves no art still draws its square and joins no band.

## Properties

**P-1** `pkg/sim` is unchanged and no serialized byte form moves; no digest pinned in this tree
changes value.

**P-2** A table whose buildings collection resolves nothing gives the plane the arms alone give,
byte for byte — the pre-story plane is reused rather than reproduced.

**P-3** The draw order is computed from the three lists' cells alone. Nothing re-anchors a placement
and nothing sorts a list a builder produced.

## Out of scope

The per-class layer scheme (stated above). Air movement. The blocked-cell diagnostic tint, which
already derives its own plane with its own table. Any change to how a footprint is resolved — that
pass exists, is correct, and is not edited here.
