# Spec — selecting a unit and ordering it from the running game

## Problem and current behaviour

The game front-end shows one screen at a time. Its map screen makes, once per tick of that screen,
exactly one advance of whatever the loader put under the open map, then drives the camera from one
input snapshot. The advance is **paced**, not one world tick per frame: it turns elapsed wall-clock
time into whole logic ticks at sixteen a second and runs between zero and four per call. Each tick
applies the map's placeholder script entry for the world's own tick, steps the world once, and hands
the window a fresh snapshot of plain map cells and resolved art carrying **no unit identity at all**.
That script is not one opening order: every unit is issued a fresh target every twenty-four ticks,
staggered per unit, for some five hundred and seventy ticks after the map opens.

A step is the only write path into a world, and it applies every command in slice order before moving
anything, so a later command for a unit overwrites an earlier one. The map extent is recorded state
and not a movement constraint: nothing clamps a target, so a unit ordered past the edge walks off the
map for good.

The camera pans, zooms and edge-scrolls and **already exposes its own inverse**: a screen point maps
to a world point that round-trips with the forward transform. What nothing does is turn that world
point into a map cell, or say which unit stands there. So there is **no interactive control** — the
player cannot select a unit and cannot order one anywhere. The left button drags the view; the right
button is not read at all.

## Functional requirements

- **FR-1** A left press and its release is a **tap** when the total cursor movement while the button
  was held is under four screen pixels, and a **drag** otherwise. A tap MUST select the unit on the
  cell it resolves to, replacing any previous selection, and MUST clear the selection when that cell
  holds no unit or lies outside the map. A drag MUST pan exactly as today and MUST NOT touch the
  selection. At most one unit is ever selected.
- **FR-2** A unit selected **and present in the most recent snapshot** MUST be highlighted on its own
  cell, carrying the vertical relief offset the cell markers already carry in displaced mode, and the
  highlight MUST leave that unit's art and any marker on that cell visible. Nothing selected, or a
  selected unit absent from the snapshot, MUST draw no highlight.
- **FR-3** A right press with a unit selected, resolving inside the map extent, MUST issue exactly one
  move order naming that unit and that cell, whatever it is walking toward now. With no selection, or
  resolving outside the extent, it MUST issue none. The right button MUST have no drag gesture.
- **FR-4** An order MUST reach the world only through an advance, MUST apply **after** that advance's
  scripted commands, and MUST apply at **exactly one** advance and no later one. Orders pending at one
  advance MUST apply in issue order, so for one unit the last issued wins. How many an advance applied
  MUST be reportable without a window.
- **FR-5** An advance with no pending order MUST advance the world on exactly the commands it would
  have before this story.
- **FR-6** From the first order naming a unit, that unit MUST take no further command from the map's
  placeholder script; every other unit MUST keep taking its own.
- **FR-7** A cursor position MUST resolve to a map cell through the camera's own inverse, the world
  point it names divided down by the cell size. A cell outside the extent MUST be reported as outside,
  never moved inside. A unit is hit exactly when it stands on the resolved cell; where two stand on one
  cell the **lower id** is hit.
- **FR-8** The next selection and the order, if any, MUST come from one pure function of the selection,
  the snapshot, the camera, the extent, the cursor and one frame's button edges: no input or output, no
  drawing, no change to any world.
- **FR-9** The simulation package MUST NOT change: no new world field, no change to the canonical byte
  form or its digest. Two worlds equal at the start and advanced by equal combined command streams MUST
  hold equal state at every tick. The selection, the resolved cell, the highlight and the pending
  orders MUST be front-end state alone — never world state, never hashed, never serialized.
- **FR-10** The map screen MUST make exactly one advance request per tick of that screen, order pending
  or not. The standalone map viewer MUST gain no selection, order or highlight, and the game's headless
  check output MUST be unchanged.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a camera at several positions and zooms | a cursor is resolved | it names the cell holding `camXY + screen/zoom`, a world point taken to screen and back is itself, and a cell beyond the extent comes back marked outside rather than clamped |
| **AC-2** | unit | press/move/release sequences, one pressing and releasing in a single frame, one moving exactly four pixels | each is run | under four pixels is a tap at the release position, four or more is none, a release with no press is nothing |
| **AC-3** | unit | units on known cells and a current selection | a tap resolves on a unit's cell, an empty cell, another unit's cell | it selects, clears, replaces |
| **AC-4** | unit | a snapshot, a camera, and a selection or none | a right press resolves inside the extent with a selection, without one, and outside the extent | one order naming the selected unit and that cell; then none; then none |
| **AC-5** | unit | the map screen over an **empty** script, driven by advance requests alone and never drawn | one unit tapped, a far in-extent cell right-clicked, advanced to arrival | it walks one cell per axis per tick, arrives, and the digest at every tick equals a headless run over that single command at that tick |
| **AC-6** | unit | a map whose script keeps issuing targets, driven past two of them | one unit selected and ordered elsewhere | it reaches the ordered cell and takes **no** scripted target again, while a second unit keeps reaching its own |
| **AC-7** | unit | K ticks with no button input; then two runs of one identical click script | all advanced | the input-free digests equal a headless run over the script alone at every tick, and the two scripted runs agree at every tick |
| **AC-8** | unit | a selected unit, right-clicked on two different in-extent cells between two advances | both advances run | the first reports **two** applied, in issue order, and the unit heads for the **second** cell; the next reports **zero** |
| **AC-9** | unit | a present selected id, an absent one, and none | the highlight is computed, flat and displaced | the present id yields its cell's footprint, offset by that cell's relief when displaced; the others yield none; the unit's art and a marker on that cell both still draw |
| **AC-10** | unit | a camera and snapshot resolving to a cell outside the extent | a right press with a selection, and a tap, resolve there | **no** order, so nothing is walked off the map; the tap clears the selection |
| **AC-11** | unit | the surfaces this story leaves alone | the simulation's pinned field sets, byte form and digest, both wall checks, the standalone viewer's render, and the headless check line | each is what it was, the viewer rendering with no selection at all |
| **AC-12** | manual | the game on a map with relief | a unit left-clicked, a ground cell right-clicked, empty ground left-clicked | the unit is highlighted with its art visible and walks to the clicked cell, staying out of its scripted lap; the empty click deselects; pan, zoom and displaced terrain still behave |

**Error cases: None applicable.** A click on empty ground, outside the map, or with no selection is a
**normally resolved outcome** — deselect, or nothing — and not a failure; an order naming a unit the
world no longer holds is already a step no-op.

## Derived properties

- **P-1 (invariant)** — An order reaches the world only through an advance; nothing else changes world
  state.
- **P-2 (invariant)** — No world field, byte form or digest differs on account of the selection, the
  resolved cell, the highlight or a pending order.
- **P-3 (negative-invariant)** — An advance with no pending order advances on exactly the script's own
  commands for that tick, inventing none.
- **P-4 (invariant)** — Two worlds equal at the start and advanced by equal combined streams, script
  and orders together, are equal at every tick.
- **P-5 (completeness)** — Every combination of button edge, resolved cell and selection lands in
  exactly one of four outcomes — select, clear, order, nothing. There is no fifth and none is undefined.
- **P-6 (idempotence)** — One order is applied by exactly one advance and by no later one, so replaying
  a session's advances from the same initial world reproduces its digests. **Within** one advance's
  command list this is argued, not witnessed: while move-to is the only kind of command, last-write
  leaves a doubled application indistinguishable from a single one.

## I/O examples

A command names a unit and a cell; no new kind of order is introduced. Resolution is the camera's own
inverse and a floor division, in both terrain modes alike.

```
cell = floor((camXY + screen/zoom) / 32)                   order only if 0 <= cell < extent

advance i    script[t]   ++ [order(u,c), order(u,c')]  -> u toward c', 2 applied
advance i+1  script[t+1] with commands naming u cut    -> u walks on,  0 applied
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | A cursor resolves to the **flat ground cell** under it — the camera's inverse, then floored by the cell size — in displaced mode as well as flat, so no height is inverted. | **(A) the nearest drawn unit within a screen radius** handles relief, at the cost of the height sampler and a radius in the pick. **(B, chosen) the flat cell** — pure over cursor, camera and snapshot. Shifting by the projection's vertical origin was rejected: that origin already cancels a typical cell's own height, so the shift makes a uniformly raised map worse. |
| **C-2** | **Ordering is gated on the extent**, and the gate is here rather than in the step. | Without it a click past the edge loses a unit; clamping in the step instead makes the extent a movement constraint for every caller. |
| **C-3** | **Left tap selects, right click orders, left drag pans.** | One button for both roles leaves an empty tap ambiguous between deselect and order-here; a modifier plus a click needs keyboard state and is harder to find. |
| **C-4** | **A commanded unit leaves the script for good**, not for one tick. | Last-write for a single tick is smaller and wrong: a fresh scripted target arrives every twenty-four ticks, so the order would be overwritten inside two seconds and the feature would read as broken. |

## Out of scope

- **Multi-select** — a drag box, control groups, formations, unit grouping.
- **Other orders** — attack-move, patrol, stop, hold, guard, right-click-an-enemy (there is no
  combat), queued waypoints — and **order feedback**: a move cursor, a click ripple, any sound.
- **The game's own mouse cursor.** Its art and where its hotspot sits are a drawing question; the pick
  reads the pointer position the engine reports, which no drawn glyph moves.
- **Stopping the world, and choosing how fast it runs.** Neither a rate of zero nor a 1–256 rate range
  can be spelled with the cadence as it stands: a stop has to be a switch beside the rate rather than a
  value of it, and the rate table the water layer paces from would move with any new range. Both belong
  to a story about the world's clock.
- **Pathfinding.** A unit ordered into a blocked path waits where it stands; nothing routes around, and
  no terrain or object blocks it.
- **Saving or replaying a session's orders**, and more than one world at a time.
- Any change to the simulation, the script's shape, decoding, the projection, terrain, water, lighting,
  the object layer, the markers, unit animation, how a map is opened, or the standalone viewer.

**Disclosed limitations**, accepted and owned: in displaced mode the resolved cell is exact on flat and
on uniform relief, and off by at most the map's altitude spread in rows otherwise (C-1); and orders
still pending when the map screen is left are dropped with it.

## Verification mapping

AC-1 … AC-11 are unit tests over synthetic worlds, snapshots and cameras built in test code — no
window, no graphics context, no game install — so all are CI-automatable. AC-12 is a developer run
against a lawful install and needs a window. P-1 … P-5 are universals and are **sampled, not proved**;
P-6 is witnessed at the advance boundary and argued within one advance.

## Gate check

FR-1 → AC-2, AC-3, AC-10, P-5, C-3 · FR-2 → AC-9 · FR-3 → AC-4, AC-10, P-5, C-2 · FR-4 → AC-5, AC-8, P-1,
P-6 · FR-5 → AC-7, P-3 · FR-6 → AC-6, C-4 · FR-7 → AC-1, AC-3, AC-10, C-1 · FR-8 → AC-3, AC-4, AC-9,
P-5 · FR-9 → AC-7, AC-11, P-2, P-4 · FR-10 → AC-11, AC-12.
