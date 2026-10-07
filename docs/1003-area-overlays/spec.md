# 1003 — spec (canonicalized to as-built)

The behavioural contract for how an area spell effect is drawn. Self-contained: claim provenance is
in `contract.md`.

## FR-1 — a burning cell shows only the full-fire frames

The retained per-cell overlay of `wall_of_fire` draws frame
`abs(counter/2 + cellX*cellY) mod 5 + 3` of its sheet. Only frames 3, 4, 5, 6 and 7 of the shipped
eleven-frame sheet are ever drawn: no birth frame and no fade frame. The 5 and the 3 are engine
constants and do not move when the registry is edited.

## FR-2 — the retained overlay is a cloud's, on four spells, with no per-cell age

A record in the **cloud** tick mode is drawn as one sprite per cell it currently holds. A record in
the **ring** (staged) or **blast** mode draws nothing through this path.

Four spell ids have overlay art: 3, 7, 8 and 19, drawing projectile records 15, 23, 25 and 47. The
other two cloud rows paint their cells, damage through them and draw nothing. A picture naming no
loaded sheet draws nothing.

The frame is a function of the world's tick counter and of the cell alone. It is **not** a function
of how long the record has stood: the same record at age 0 and at age 200 draws the same frame on
the same cell on the same tick. Spells other than 3 take `abs(counter/2 + cellX*cellY) mod Phases`,
with the sheet's own phase count and no bias.

## FR-3 — a staged effect puts one short-lived object on each cell it paints

Every stage of a staged effect reports the cells it accepted. Stage 0 reports at the landing; each
later stage reports on the tick it runs, three ticks apart. The terminal stage reports on the tick
its record is removed.

For each reported cell the client creates one object at that cell, drawing the spell's burst
picture `2*id + 9` for that picture's own life — 16 ticks, 18 for Acid Stream's picture 27. A spell
whose burst picture names no sheet creates none.

Because a stage runs every three ticks and an object lives sixteen, five or six stages are drawn at
once. An empty stage reports nothing.

The report is a return value of the advance and not state: no field of `World` carries it, the byte
form is unchanged, and a world advanced with no observer is the same world byte for byte.

## FR-4 — a spell sprite stands on its cell's centre

The world pixel a spell sprite's frame is blitted at is `cell*CellSize + CellSize/2` less the
registry's centring halves, before the relief lift and the camera. It was `cell*CellSize` less the
halves, which is the cell's top-left corner and put every spell sprite 16 px left and 16 px up of
its own cell. The same term applies to the heal shower and to the archer's shot mark.

## Refusals

A nil sheet, a frame index the sheet does not hold, and a frame of no area each draw nothing.
A record in a mode with no overlay art draws nothing. A sheet stating no positive phase count has
no frame. None of these is an error.
