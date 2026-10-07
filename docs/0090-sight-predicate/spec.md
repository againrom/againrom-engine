# Spec — a group only engages what it can actually see

**Intensity:** spec-anchored / static. **Terrain:** brownfield in `pkg/sim` — the group engagement
decision ships and its candidate population changes — and greenfield for the sight window and the
march.

## Why

A group in this build acquires everything hostile inside a plain Chebyshev disk. It notices a unit
through a mountain, and it fails to notice one standing below it that the original would have seen
from the high ground. Both halves matter: the region the game admits is not a disk, it is a budget
spent walking outward, and the largest term in that budget is the observer's own altitude.

What this contract fixes is the region, and the one consumer of it: which entities a group's
decision may consider at all. Everything the decision then does with that population — the
filtering, the clip, the scoring, the order it writes — is 0086's and is unchanged.

## Requirements

### The sight window

**FR-1** A build carries a **sight window**: a square of cell offsets `(dx, dy)` with `|dx| <= 20`
and `|dy| <= 20`. It is derived from geometry and one **shift** parameter of value 7, and from
nothing any map, world or entity carries. It is **not canonical state**: no byte form carries it, no
digest covers it, and no world can differ from another in it.

**FR-2** Every window offset but the origin has a **predecessor** one cell away, and the
predecessor is **strictly closer to the origin in Chebyshev distance**. It is `(-sgn dx, 0)` where
`|dy| < |dx| >> 1`, `(0, -sgn dy)` where `|dy| > 2|dx|`, and `(-sgn dx, -sgn dy)` between them —
except at the two offsets `(1, 0)` and `(-1, 0)`, whose predecessor is the origin.

**FR-3** Every window offset but the origin has a **step cost**: the integer
`trunc( 128 * sqrt(dx*dx + dy*dy) / max(|dx|, |dy|) )`. It is 128 on the axes, 181 on the diagonals,
and never outside 128..181. The origin has no step cost and none is ever read.

### The march

**FR-4** A unit's **sight budget** is `64 + (range << 7)`, where `range` is its sight range in whole
cells.

**FR-5** A march from an observer keeps one **running value** per window offset. The origin's is the
budget. Every other visited offset's is
`value(predecessor) - stepCost(offset) - height(cell) + height(observer cell)`,
where `height` is the world's height plane byte read as a **signed** byte, and zero for a cell the
plane does not describe.

**FR-6** A cell is **visible** exactly when its running value is greater than zero. The value is
kept whether or not it is, and a cell whose predecessor is not visible is still evaluated from that
predecessor's kept value: **a blocking cell is not pruned and casts no shadow of its own.**

**FR-7** The march visits offsets in **Chebyshev rings** of radius 1 through 19, and stops after the
first ring in which no cell was visible. Ring 20 — the window's own outermost ring — is never
visited.

**FR-8** The observer's own cell is visible, whatever the terrain under it and whatever the budget.

**FR-9** A cell is **not visited** — no value is kept for it and it is not visible — when it lies
outside the world's bounds, or when the world's grid closes it to an air mover. An unvisited cell's
value is zero for any later cell that names it as a predecessor.

### What a group sees

**FR-10** A group's sight is the **union** of one march per living member, each from that member's
own cell with that member's own sight range. It is rebuilt from nothing on every decision and no
part of it is remembered.

**FR-11** An entity is a **candidate** for a group's decision exactly when its own cell is in that
union. This replaces the Chebyshev-disk rule the build carried before, and it is the only input to
the decision that changes.

## Acceptance criteria

- **AC-1** Every one of the window's 1 680 non-origin offsets has a predecessor strictly closer to
  the origin, and each of `(1,0)` and `(-1,0)` names the origin itself; without FR-2's two named
  exceptions exactly two offsets would fail that.
- **AC-2** The step cost is 128 at every axis offset, 181 at every diagonal offset, and within
  128..181 everywhere else.
- **AC-3** On a world with no height plane, the number of cells a single march makes visible is
  9, 21, 45, 69, 105 and 145 for sight ranges 1 to 6, and 1 253 at range 19.
- **AC-4** On the same world, a march of range `r` reaches exactly `r` cells along each axis and
  `(64 + (r << 7)) / 181` cells along each diagonal.
- **AC-5** A ridge of raised ground between two hostile units stops the acquisition that the same
  pair makes on flat ground at the same distance.
- **AC-6** An observer standing high acquires a candidate further away than the flat-ground reach,
  and one standing low does not acquire a candidate the flat ground would have given it.
- **AC-7** With every altitude byte at 127 or below, no cell is visible whose predecessor was
  evaluated and found not visible. With one authored byte at 128 or above, such a cell is visible —
  and that is the whole of the difference between this build and one that prunes at a blocker.
- **AC-8** A cell the grid closes to an air mover is never visible, does not carry a value to a
  later cell, and does not keep a ring alive; an observer standing on such a cell still sees it.
- **AC-9** No cell outside the world's bounds is ever visible, and on a world 256 cells wide an
  observer near the left edge lights no cell on the right edge.
- **AC-10** A candidate visible to one member of a group is a candidate for every member of it, and
  a candidate visible to no member is a candidate for none.
- **AC-11** A world's byte form, its decoded field set and its digest are what they were: a world
  round-trips unchanged, two worlds built alike hash alike, and the form version has not moved.
- **AC-12** On flat ground a candidate at the sight range along an axis is still acquired and one a
  cell past it is still not.

## Properties

- **P-1** *Determinism.* The march reads no clock, iterates no map, draws from no generator and
  computes with no floating-point value. Two worlds with equal byte forms advanced against equal
  commands stay equal, and a recorded drive replays byte-identical.
- **P-2** *Idempotence.* Two marches over a world nothing has changed produce the same set of
  visible cells, and taking two decisions in succession leaves the same assignment.
- **P-3** *Star-shapedness under the shipped altitude range.* Over a swept space of terrains whose
  every byte is at most 127, no visible cell has a predecessor that was evaluated and blocked. This
  is the measured form of the seam AC-7 names, and it is the test that fails the day a map authors a
  byte of 128 or more.
- **P-4** *Narrowing on flat ground.* On a world naming no height plane, every visible cell lies
  within the Chebyshev disk of the sight range, so the population this build considers is a subset
  of the one it considered before.

## Divergence from the published law

- **D-1** *The sight range is one constant for every unit.* The law reads it off the actor, where a
  database column puts it and a hero's own recomputation overwrites it. A per-entity range is hashed
  simulation state and needs a byte-form version; this story does not move one. Every unit here
  scans 5 cells, which is that column's own default, and the seam is the single constant.
- **D-2** *The playable rectangle is read off the grid rather than computed from the bounds.* The
  law tests each ring cell against a rectangle inset 8 cells from every edge, written at map load.
  FR-9 tests the grid's air bit instead. On every map this build loads the two are the same set of
  cells, because that bit is set for a cell within 8 of an edge and for nothing else. They differ on
  a world assembled by hand: one naming no grid has no inset at all, and one setting that bit
  somewhere else carves sight there. The second is out of reach of any loader.
- **D-3** *The law's boundary test is defective and the defect is not reproduced.* It compares the
  rectangle as bytes while indexing the planes in 32 bits, so on a 256-wide map a true column of -9
  wraps to 247, passes the test and lights a cell on the far side of the map. It needs a ring of
  radius 17 or more, so it cannot occur until sight runs far past its range down a slope, and 11 of
  the 72 shipped maps are wide enough. FR-9 bounds-checks exactly. This build therefore lights
  strictly fewer cells than the law on those maps, and lights none the law would not.
- **D-4** *The seam that FR-6 and AC-7 exist for.* Because a blocker is not pruned, the region is
  star-shaped only while `height(observer) - height(cell)` cannot reach the cheapest step cost of
  128. Every shipped altitude byte is 0..127, so the largest rise available is 127 — one short — and
  an implementation that pruned at the first blocker would be indistinguishable here and would
  diverge on the first map authoring a byte of 128 or more. That limit is a fact about the shipped
  corpus, not about the code; lifting it changes no shipped file, only what a map may say. The
  faithful walk is built, so the limit is a measurement (P-3) rather than an assumption.
- **D-5** *The shift is a code constant.* The law takes it from a registry key, which ships as 7 in
  both roots against a compiled default of the same 7. This tree reads no registry, so it is written
  as the constant FR-1 names. Moving it moves this predicate's precision and nothing else in the
  build — the drawn fog is a second implementation with its own copy.

- **D-6** *The clip 0086 shipped as inert is inert only on flat ground now.* That story recomputes a
  guarding group's notice radius fresh instead of freezing it, and argued from the Chebyshev
  triangle inequality that the clip could then never drop a candidate — an argument whose premise
  was that the visible set is the disk of the sight range. It is not, once altitude is in: from high
  ground a candidate can be visible far past that range, and the clip can drop it. The property that
  story pins is kept and is still measured, on a world naming no height plane, where it still holds.

## Out of scope

The player's fog of war, which is a second implementation of this algorithm on a different object
with a different clock, a different store and different consumers. A per-entity sight range and the
byte form that would carry it. The remembered attacker's cell that the law forces into the stamp for
twenty decisions. The invisibility test the law applies to a candidate. The squared-distance grid
built beside these two, which is a disc test and not this predicate. Everything 0086 placed out of
scope stays there.

## Traceability

| FR | AC | Property |
|---|---|---|
| FR-1 | AC-11 | P-1 |
| FR-2 | AC-1 | — |
| FR-3 | AC-2 | — |
| FR-4 | AC-3, AC-4 | — |
| FR-5 | AC-5, AC-6 | — |
| FR-6 | AC-7 | P-3 |
| FR-7 | AC-3, AC-8 | P-4 |
| FR-8 | AC-8 | — |
| FR-9 | AC-8, AC-9 | — |
| FR-10 | AC-10 | P-2 |
| FR-11 | AC-10, AC-12 | P-2, P-4 |
