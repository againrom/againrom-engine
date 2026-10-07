# Spec — the step cost, relative to the mover

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/sim` (the advance's rate site
is re-expressed, and its behaviour must not move) and in `pkg/ui` (an existing box grows two rows);
**greenfield** for the query and the seam that carries it.

## Why

The debug readout states a unit's speed inputs and the length of the crossing it is already on. It
cannot answer what a step onto a **named** cell would cost that unit. The raw byte the cost plane
holds is not that answer: it is the same for every unit standing anywhere, and the movement law
weighs it against a second cell, a height difference, a domain and an effective speed before any
number falls out. What is wanted is the law's answer **for that mover**.

## Requirements

**FR-1 — the rate.** For the selected unit and the cell under the cursor, the readout states the
movement law's rate for one transit **from the cell that unit stands on to the cell under the
cursor**, as the bare integer the law produces.

**FR-2 — the transit.** Beside it, the readout states how many ticks that transit takes at that
rate, in the same form the readout already states a tick span.

**FR-3 — the movement code's own number.** Both values are produced by the same computation the
advance uses to rate a mover's step, reached through a query, and are not recomputed anywhere else.
There is exactly one place in the tree that composes the law's inputs from a world.

**FR-4 — which mover, which cell.** The mover is the one the readout's other unit rows describe and
the unit panel describes — the same filter, called, not copied. The cell is the ground pick's own
answer, the same one the `CURSOR` row states.

**FR-5 — absence, four causes.** No value is stated when there is no unit to state it about, when
the cursor resolves to no cell on the map, when the cell under the cursor is the cell the unit
already stands on, or when the mover carries no effective speed. The first omits both rows together
with the readout's other unit rows; the other three state the readout's absence marker.

**FR-6 — a pair that is no single step is marked.** When the cell under the cursor is more than one
cell away from the mover's own, the stated values carry a marker word distinguishing them from a
step the game could take. The values themselves are unchanged by it.

**FR-7 — the divergence is disclosed.** Two terms of the published per-step law are not computed by
this tree, and neither is folded into the stated figures:

- the movement multiplier is applied at the shipped default and **its map parameter is not read** —
  a map that customises it is rated as though it had not;
- the original's cost accessor is **not a pure read** and this tree's is — it never rewrites a
  cell's stored cost, so a map whose cells the original would rewrite is rated from the unmodified
  plane.

Both are stated in the source at the site that computes the figure, and in the story's evidence.

**FR-8 — the simulation's state does not move.** No field is added to, removed from or retyped in
any simulation state type; the canonical byte form and its version are unchanged; the digest over a
given world is unchanged. Whatever this story adds to that package is a **read**.

**FR-9 — the surfaces stay separate.** The values are added to the debug readout alone. The unit
panel gains no row, and the readout keeps its default: it is shown unless hidden, and its hide key
still hides it.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a rated ground mover on a cost and height plane, cursor one cell away | the readout is composed | the rate row states exactly the rate the advance computes for that same transit, and the transit row states that rate's tick span |
| **AC-2** | the same, with the destination cell's cost byte changed | the readout is composed | the stated rate changes accordingly — the value is a function of the terrain under the cursor, not of the mover alone |
| **AC-3** | two movers of different effective speed, cursor on one cell | the readout is composed for each | the two state different values — the figure is relative to the mover |
| **AC-4** | a mover whose group term is nonzero | the readout is composed | the stated values are those the group term produces, not those the unit's own speed would |
| **AC-5** | a mover on a downhill step, and the same mover on the uphill reverse | the readout is composed for each | the two differ, and the downhill one is the faster — source and destination are not interchangeable |
| **AC-6** | a diagonal neighbour and an orthogonal neighbour of equal cost and height | the readout is composed for each | the diagonal transit is the longer |
| **AC-7** | no unit selected | the readout is composed | neither row is present, and the readout's other unit rows are absent with them |
| **AC-8** | a unit selected and the cursor off the map, or on the unit's own cell | the readout is composed | both rows are present and state the absence marker |
| **AC-9** | a unit whose effective speed is zero, cursor on a neighbouring cell | the readout is composed | both rows state the absence marker rather than the value a total law would yield for it |
| **AC-10** | a unit, and a cursor cell more than one cell away | the readout is composed | the values are stated with the marker word |
| **AC-11** | a viewer that was never handed the query | the readout is composed | both rows state the absence marker and the readout draws otherwise unchanged |
| **AC-12** | a world before and after this story | it is encoded and hashed | the byte form's version literal, the encoded bytes and the digest are identical |
| **AC-13** | any composed readout | its rebuild key is compared | the two new values are part of what the key compares |
| **AC-14** | the shipped readout layout | it is read | the two rows are present in it, and no existing row's field number, label or order has moved |

## Properties

**P-1 — negative invariant: no number the movement code would not use.** For every input on which
the advance declines to rate a mover, the readout states no number for that mover. The law is total
where the advance is not, and that difference is never displayed.

**P-2 — invariant: one composition site.** The rate the readout states and the rate a mover moves at
are produced by one function over one world. Changing the law's inputs in that function changes both
or neither.

**P-3 — invariant: the drawing tier names no simulation.** Everything crossing into the drawing tier
for this is a builtin, and that tier's permitted imports do not grow.

**P-4 — invariant: what is drawn is what is keyed.** Every value this story puts on screen is part
of the comparison that decides whether the box is recomposed.

## Divergence from the published law

Stated here because a figure presented without it is a partial law wearing a whole one's clothes.

Beyond FR-7's two terms, the **destination of a step is an input** to this contract rather than
something derived. The published law derives it from a facing byte through a direction table; this
tree derives a mover's step from its route search. The readout is asked about a cell the reader
points at, so it takes the pair and reproduces neither derivation.

The stated figure for a pair more than one cell apart is the law evaluated on that ordered pair. It
is not a path cost, and no path is searched; FR-6's marker exists so it cannot be read as one.

## Out of scope

The unit panel; the attack pointer; the dialogue window; facing and animation; damage numerals; a
reader for the movement multiplier's registry key; the original's conditional rewrite of a cell's
stored cost; any change to how a mover is routed, rated or advanced; any change to the cost or
height planes or to how they are loaded.

## Traceability

| Requirement | Criteria |
|---|---|
| FR-1 | AC-1, AC-2, AC-3, AC-5, AC-6, AC-14 |
| FR-2 | AC-1, AC-6, AC-14 |
| FR-3 | AC-1, AC-4, AC-5 |
| FR-4 | AC-3, AC-4, AC-7 |
| FR-5 | AC-7, AC-8, AC-9, AC-11 |
| FR-6 | AC-10 |
| FR-7 | AC-1 |
| FR-8 | AC-12 |
| FR-9 | AC-13, AC-14 |
| P-1 | AC-8, AC-9, AC-11 |
| P-2 | AC-1, AC-4, AC-5 |
| P-3 | AC-14 |
| P-4 | AC-13 |
