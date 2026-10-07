# Spec — the step a unit took, drawn between the two cells it joins

## Problem and current behaviour

The open map screen draws each entity on the whole map cell the simulation put it on, and
decides whether it is walking — and which of eight directions it faces — from the **order
it holds**: an entity counts as moving when it has a target that is not the cell it stands
on. A mover's direction is remembered per entity; one that is not moving is drawn with the
direction it last moved in, or with the never-moved direction if it never has.

The simulation sets a target and clears it on arrival inside **one** advance, so an order
to an **adjacent** cell is in force at no tick boundary at all: the unit walks the cell, the
screen sees no target, draws it idle and leaves it facing where it faced before. The **last
step of every longer walk** is the same picture for the same reason. A unit blocked for a
tick, by contrast, still holds its target and is drawn walking while standing still.

An entity advances **one whole cell per tick**, all-or-nothing, and a map opens at 16 ticks
a second, so between two advances every entity is drawn in the same place: a walking unit
crosses the map in sixteen jumps a second.

The advance is paced by a period and an accumulator carrying the sub-tick remainder; a stop
is read before the elapsed span reaches it, and a rate change rewrites the period keeping
the remainder. Nothing on the draw path reads any of it.

## Functional requirements

- **FR-1** Whether a drawn entity is walking, and which direction it faces, MUST be derived
  from the **step it took into the cell it now stands on** — the change in its cell since
  the previous advance — and MUST NOT depend on whether it still holds an order. An order
  to an adjacent cell MUST therefore turn the unit and play its walking frames exactly as a
  longer order does, and so MUST the final step of a longer walk.
- **FR-2** An entity whose cell did not change on the most recent advance MUST be drawn
  idle, facing the direction of its most recent step. One that has taken no step yet —
  never moved, or appearing in a snapshot for the first time — MUST take the never-moved
  direction and no displacement, however long it has existed.
- **FR-3** Between advances an entity that moved on the most recent advance MUST be drawn
  **between the cell it left and the cell it entered**, displaced by how far the current
  tick has progressed: on the cell it left at that tick's start, on the cell it entered at
  its end, proportionally between. Its sprite, its selection mark and its health bar MUST
  carry the **same** displacement wherever each is drawn, so a unit and everything on it
  move as one body.
- **FR-4** In height-displaced mode the displacement MUST carry the terrain relief with it:
  a unit stepping between cells of different height MUST rise or fall across the tick
  rather than at its boundary, between exactly the two per-cell heights the glyphs on those
  two cells are lifted by.
- **FR-5** An entity that did not move on the most recent advance MUST be drawn with **no
  displacement**, on its own cell — standing, blocked, downed and dead alike.
- **FR-6** The displacement MUST be a function of the **same clock the advance is paced
  by**. While the world is stopped no entity's drawn position MUST change, however many
  frames are drawn; a change of rate MUST NOT place an entity beyond the cell it entered or
  behind the cell it left; and a viewer never told where it stands within a tick MUST draw
  every entity on its own cell.
- **FR-7** The simulation MUST NOT change: no new field, no change to the canonical byte
  form or its digest, and no position that is not a whole cell. Every value this story adds
  MUST be front-end state alone — never world state, never hashed, never serialized. For
  one command stream, `k` ticks MUST reach the same state and the same digest whatever was
  drawn between them.
- **FR-8** The **tick-0 picture MUST be unchanged**: a map opened and not yet advanced MUST
  draw every entity on the very cell its unit record placed it on, its sprite's ground
  point still the marker path's own anchor for that cell. A front-end that pushes no
  entities MUST be unchanged in every respect.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a driver over a hand-built world, one entity, an order to an **adjacent** cell | one advance is run and the pushed entities read | the entity reads as having moved, in the direction it moved, and its selected frame is that direction's walking frame |
| **AC-2** | unit | the same driver, an order four cells away | one advance per line until arrival, then one more | every step including the **arrival** step reads as a move in its own direction; the advance after arrival reads idle and keeps the arrival direction |
| **AC-3** | unit | an entity that has never moved; separately one held up for a tick by a neighbour standing in its way | advances are run | the first is idle in the never-moved direction throughout; the second is idle on the held-up tick and keeps its previous direction |
| **AC-4** | unit | a viewer holding one entity that stepped one cell east, at remainders 0, a quarter, a half and a whole period | the frame's entity geometry is read | sprite, mark and bar are each displaced by the same vector — one cell west scaled by the part of the tick still to run — reaching zero at a whole period |
| **AC-5** | unit | the same over a map with altitudes, the two cells differing in height | the geometry is read at those same four points | the vertical displacement runs from the left cell's own lift to the entered cell's lift across the tick, shared by sprite, mark and bar |
| **AC-6** | unit | entities standing, held up, downed and dead | frames are read at every remainder | none carries any displacement at any remainder |
| **AC-7** | unit | a stopped world with a unit mid-stride; a period shrunk below the remainder held; a viewer never given a phase | many frames are read | the stopped geometry is identical in every frame; the shrunk period draws the entity on the cell it entered and never past it; the untold viewer draws every entity on its own cell |
| **AC-8** | unit | one command stream over a hand-built world | `k` ticks are run with drawing driven at many remainders, and again with none | the pinned simulation fields, the byte form and the digest at every tick index are equal, and both wall checks are unmoved |
| **AC-9** | unit | a map opened and not advanced | the frame's entity geometry is read | every entity stands on its unit record's own cell and each sprite's ground point is the marker path's anchor for that cell |
| **AC-10** | manual | the game on a map with several units, against a lawful install | a unit is ordered one cell, then across the map; the world is paused mid-stride and resumed, and the rate raised and lowered | the one-cell order turns the unit and plays its walk; the long walk slides with no jump and rises and falls with the ground; the paused unit holds still; the rate changes how fast it slides, not how far it steps |

**Error cases: None applicable.** Every input is total — a remainder outside its period is
brought inside it, an absent or non-positive period draws no displacement at all, and a
direction is chosen only from a step that exists — so there is nothing here to refuse.

## Derived properties

- **P-1 (invariant)** — Drawing is a function of the pushed snapshot and of where the frame
  stands within a tick, alone: no world field, byte form or digest differs on account of
  any value this story adds.
- **P-2 (negative-invariant)** — For any number of frames drawn with no advance between
  them, and for any span the world spends stopped, every entity's drawn position, frame and
  direction is identical — the snapshot they are read from included, so asking for it twice
  with no advance between changes nothing and answers the same.
- **P-3 (invariant)** — One displacement per entity per frame: an entity's sprite, its
  selection mark and its health bar are displaced by the same vector, always.
- **P-4 (completeness)** — Every entity's drawn position lies on the closed segment between
  the cell it left and the cell it entered; for an entity that did not move that segment is
  its own cell. There is no third case and none is undefined.
- **P-5 (invariant)** — The endpoints agree with the undisplaced picture: at a remainder of
  zero an entity is drawn exactly where it would be drawn standing on the cell it left, and
  at a whole period exactly where it is drawn standing on the cell it entered.

## I/O examples

```
step (dx,dy) in cells, cell 32 px, remainder r of period p, left = p - clamp(r, 0, p)
  offset px    = (-dx * 32 * left / p,  -dy * 32 * left / p)
  east step, p = 62000:  r=0 -> (-32,0)   r=31000 -> (-16,0)   r>=p -> (0,0)
  with altitudes, lift of the cell left L0 and of the cell entered L1:
               offset y -= (L0 - L1) * left / p
  p <= 0, or step (0,0)                    -> offset (0,0)
  both axes take that one formula; all of it whole numbers, truncating; 32 is the
  render's own cell, so the camera's zoom multiplies the result and never the inputs
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The in-between position is a **drawing**, never simulation state. | **(A) a fractional coordinate in the simulation** — the byte form widens, the digest's shape moves, and collision, occupancy and search each acquire a sub-cell question: a large correctness surface bought for a visual. **(B, chosen) a render displacement** — no new canonical field, nothing hashed, the world advanced by the integer steps it already was. |
| **C-2** | The picture **follows** the simulation by up to one tick. | **(A) extrapolate toward the target** — no lag, but the next step is a guess the search may not take, and a wrong guess snaps back visibly. **(B, chosen) interpolate between two cells that have both already happened**, at the cost of one tick's delay. |
| **C-3** | Where a frame stands within a tick comes from **the clock the advance is paced by**. | **(A) a wall clock on the draw path** — a stopped world would slide. **(B) an accumulator of the drawing's own** — two clocks to keep in step, disagreeing exactly under a pause or a rate change. **(C, chosen) the remainder the pacing accumulator already carries**, the quantity the advance is itself a function of. |
| **C-4** | Direction and walk/idle come from the **step taken**, not the order held. | **(A) the order** — false at every tick boundary of a one-cell walk and of every arrival, which is the defect. **(B) the order, widened to survive arrival** — a simulation change to serve a drawing, and still wrong for a unit held up. **(C, chosen) the observed step**, the very quantity the displacement is measured along; a direction is still remembered across the ticks that have no step, exactly as it is today. |

## Out of scope

- **Sub-cell position in the simulation** — a fractional coordinate, a movement remainder,
  or any widening of the canonical byte form or its digest. That is the second of the two
  possible fixes and is deliberately not taken (C-1).
- **Per-unit movement speed**: a unit still steps one cell per tick, and nothing here makes
  one unit faster than another.
- **Turning animation** — a unit still snaps to its new direction — **smoothing the frame
  selection**, which still changes only at a tick boundary, and **any easing** of the
  displacement, which is proportional and nothing else.
- **A corpse's own facing rule**, and any death animation.
- **Interpolating anything that is not an entity** — water, static objects, the camera and
  the diagnostic overlays are untouched — and **any on-screen indication** of the
  displacement or control to turn it off.

**Disclosed limitations**, accepted and owned: the entity layer is drawn up to one tick
behind the world (C-2), so at 16 a second an order looks about 62 ms late; a unit
alternating between stepping and being held up now alternates between walking and idle
frames; the displacement is in whole render pixels, so a zoomed-in step advances in the
render's pixel steps and not the display's; and a frame in which several ticks fired at once
interpolates the last of them alone, so a catch-up burst still jumps.

## Verification mapping

AC-1 … AC-9 are unit tests over hand-built worlds, viewers and remainders passed in — no
window, no install, no clock read — so all are CI-automatable; AC-10 needs a lawful install
and a window. P-1 and P-2 are witnessed at every frame the AC-7 and AC-8 schedules drive,
P-3 and P-5 at the points AC-4 and AC-5 read; P-4 is **sampled**.

## Gate check

FR-1 → AC-1, AC-2, C-4 · FR-2 → AC-3 · FR-3 → AC-4, P-3, P-4, C-2 ·
FR-4 → AC-5 · FR-5 → AC-6, P-4 · FR-6 → AC-7, P-2, C-3 · FR-7 → AC-8, P-1, C-1 ·
FR-8 → AC-9, P-5.
