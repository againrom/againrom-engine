# Spec — selecting a group and ordering it from the running game

## Problem and current behaviour

The game's map screen reads two mouse buttons and **no modifier key at all**. A **left tap** — a press
and its release, the cursor having travelled under four screen pixels while the button was held —
selects the unit standing on the cell the cursor resolves to, replacing whatever was selected, and
clears the selection on an empty or off-map cell. A **left drag** pans the camera. A **right press**
orders the one selected unit to the resolved in-map cell. **At most one unit is ever selected**, so
more than one cannot be moved at a time — and the drag that would draw a selection box is already
spent on the camera.

A unit is hit exactly when it stands on the resolved cell — the camera's own inverse, floored by the
cell size — the lower id winning where two share one. The selected unit is marked by a hollow rim on
its cell, lifted by the relief in displaced mode; one absent from the latest snapshot is marked not at
all.

An order leaves the front-end one entity and one cell per call, at most once per tick of the map
screen, and reaches the world only through that screen's single advance — which runs once per tick
whether an order was issued or not, and already applies a whole batch of commands in slice order. The camera step that screen drives is the **same** step the
standalone terrain viewer drives, and that viewer has no world under it and no route to a selection.
Edge-scroll is suppressed while the left button is held; keyboard pan and wheel zoom stay live.

Under the map, a unit holding a target is routed afresh each tick and advances one cell, never entering
a cell out of bounds, blocked, or held by another. A tick that finds no route leaves it where it stands
and raises its stall count; sixteen consecutive stalls clear the target. So a cell holds one unit, and
a second ordered onto an occupied one stops where the refusal found it.

## Functional requirements

- **FR-1** With a world under the map screen, a left press begun **with either Shift key held** MUST
  pan by dragging exactly as the unmodified left drag panned before and MUST change no selection; one
  begun **without** MUST pan by zero. The standalone terrain viewer MUST be unchanged: its plain left
  drag MUST still pan and no modifier MUST alter it.
- **FR-2** With a world under the map screen, a left press begun without Shift and released past the
  slop threshold MUST replace the selection with **every unit standing on a cell the SCREEN rectangle
  between the press and release points covers**, resolved against the camera as it stands at the
  release — a cell is covered when the rectangle meets any part of it. The rectangle MUST be
  orientation-independent; one of zero width or height, including one whose two points coincide, MUST
  still cover the cells its line or point meets; and a release covering no unit MUST clear the
  selection.
- **FR-3** A left tap MUST select the unit hit on the cell it resolves to, replacing the whole
  selection, and MUST clear it when that cell holds no unit or lies outside the map — the pre-existing
  single-unit outcome, as a selection of one or of none.
- **FR-4** A right button **going down** and resolving inside the map extent MUST issue one move order
  per selected unit **present in the most recent snapshot**, each naming that same cell, **in ascending
  entity id**. It MUST issue none when nothing is selected, when no selected unit is present in that
  snapshot, when the cell lies outside the extent, or **while a left press is still in progress**. The
  right button MUST have no drag gesture, and no level of it MUST be read.
- **FR-5** Every selected unit present in the most recent snapshot MUST be marked on its own cell,
  carrying the relief offset a single selected unit's mark carries, and MUST leave that unit's art and
  any marker on that cell visible. One absent from that snapshot, and an empty selection, MUST be
  marked not at all. An absent id MUST be **skipped, not dropped**: only a tap or a release replaces
  the selection.
- **FR-6** Once a left press begun without Shift has passed the slop threshold, and while it is still
  in progress with a world under the map screen, the screen rectangle between the press point and the
  current cursor MUST be drawn as an outline leaving its own interior visible, and MUST disappear on
  release. A press under the slop, one begun with Shift, and the standalone terrain viewer MUST draw
  none at all.
- **FR-7** Which gesture a left press is MUST be fixed as the button goes down and MUST hold for the
  whole press: a modifier pressed or released mid-drag MUST NOT change it.
- **FR-8** The simulation package MUST NOT change: no new field, no change to the canonical byte form or
  its digest. The selection, the rectangle and all press bookkeeping MUST be front-end state alone —
  never world state, never hashed, never serialized. The map screen MUST still make exactly one advance
  per tick whether it issued zero, one or many orders, and one frame's orders MUST all reach **one**
  advance, in issue order.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a world under the map screen, and separately a viewer with none | a left drag runs with Shift held and without | with a world, Shift pans by the delta the unmodified drag produced before and selects nothing while plain pans zero; without one, plain pans by that delta and Shift changes nothing |
| **AC-2** | unit | units on known cells, a prior selection, and a camera **panned off the origin and zoomed off 1** | a drag is released over `k` units, over empty ground, corner-swapped on either axis, and at zero width and height | exactly those `k`, replacing; cleared; each swapped set equals its un-swapped one; each degenerate one catches the units under its line |
| **AC-3** | unit | units on known cells, a current selection, and the same non-identity camera | a tap resolves on a unit's cell, an empty cell, an off-map point | selects that one, clears, clears |
| **AC-4** | unit | a selection of `k` present units | the right button goes down inside the extent; then outside it; then inside while a left button is held; then inside with the selection emptied; then inside with every member gone from the snapshot | `k` orders, one per present member, all naming that cell, in ascending id — then none, in each of the four remaining cases |
| **AC-5** | unit | a synthetic world of known layout, several units selected | the group is ordered to one reachable unoccupied cell and advanced to quiescence | every ordered unit the world still holds carries that cell as its target at the advance that applied the orders; no two share a cell at any tick; the first to get there stands on it. **No arrival is claimed for the rest** — where the world leaves them is recorded, not required |
| **AC-6** | unit | `k` present selected units, an absent selected id, and none | each is marked, flat and displaced | `k` marks on their own cells, each offset by that cell's relief when displaced, the art still drawn; the absent id and the empty selection yield no mark and no pass |
| **AC-7** | unit | a plain drag, a plain press under the slop, a Shift drag, and a viewer with no world | each runs several frames, then releases | the outline runs from the frame the first passes the slop and is gone after release; absent in all three others |
| **AC-8** | unit | a press begun without Shift, and one begun with it | Shift is pressed mid-drag in the first, released mid-drag in the second | the first still resolves as a selection and pans zero; the second still pans and touches no selection |
| **AC-9** | unit | `K` ticks with no button input; two runs of one input script; a left button held many frames and never released | all are advanced | the input-free digests equal a headless run over the schedule alone at every tick; the two scripted runs agree at every tick; the held button issues **zero** orders |
| **AC-10** | unit | the surfaces this story leaves alone | the simulation's pinned fields, byte form and digest, both wall checks, the standalone viewer's render and headless line, and the advance count per map-screen tick at zero, one and many orders | each is what it was, the count one every time |
| **AC-11** | manual | the game on a map with relief and several units | a box is dragged round two, ground right-clicked, empty ground clicked, Shift held and dragged | both are marked and walk off toward the cell; the empty click clears; the Shift drag pans and marks nothing; pan, zoom and displaced terrain still behave |

**Error cases: None applicable.** A drag over empty ground, a click past the map edge and a right
press with nothing selected are **normally resolved outcomes** — clear, or nothing — not failures.

## Derived properties

- **P-1 (invariant)** — An order reaches the world only through an advance; nothing done to a selection
  or a rectangle changes world state on its own.
- **P-2 (invariant)** — No world field, byte form or digest differs on account of a selection, a
  rectangle, a mark or a pending order.
- **P-3 (completeness)** — Every combination of button edges, modifier at the press, resolved cell and
  current selection lands in exactly one of four outcomes — replace the selection, clear it, order it,
  nothing. There is no fifth and none is undefined.
- **P-4 (invariant)** — The orders one right press issues are a function of the selection and the
  resolved cell alone, in ascending id, so two runs of one input script produce one command stream and
  one digest per tick.
- **P-5 (negative-invariant)** — For any left press not yet released, and any release covering no unit,
  **no order is issued**: a rectangle being dragged never fabricates one.

## I/O examples

An order still names one entity and one cell — no new kind, no new payload.

```
covered = floor((camXY + screen_min/zoom)/32) .. floor((camXY + screen_max/zoom)/32), clipped to extent

press(shift=0) .. move .. release  -> selection := {ids on covered cells}, camera unmoved
press(shift=1) .. move .. release  -> camera pans, selection unmoved, nothing drawn
right press, selection {7,3,9}     -> order(3,c) order(7,c) order(9,c)   one advance, 3 applied
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | A unit is caught by the rectangle exactly when it **stands on a covered cell** — the tap's own resolution, generalised from one cell to a band. | **(A) the drawn rectangle** catches a unit anywhere on its visible body, a sprite's crown included; it costs a second geometry that can disagree with the tap over the very same pixels, so a click and a one-cell box would answer differently. **(B, chosen) the cell** — one hit rule for both, and the relief error below stays the tap's own. |
| **C-2** | **Shift takes panning**; the unmodified drag becomes the selection box. | The modifier on the box instead leaves the more frequent gesture behind a key; a middle button needs one not every mouse has; a latching mode is invisible state with nothing on screen to show it. |
| **C-3** | Every ordered unit goes to the **same** cell. | A distinct cell each is a formation — a rule about where a group stands that the world has no notion of; inventing one puts a placement policy in the front-end, where the world decides it. |

## Out of scope

- **Adding to a selection** — a modifier with a click or a box, double-click for all of a type, saved
  control groups — and **formations**: every unit shares the target this slice.
- **Any change to the simulation**, expressly including an adjacent-cell fallback for a unit whose
  target is occupied — that rule is the world's and stays as it is.
- **Other orders** — attack-move, patrol, stop, hold, queued waypoints, right-drag — and order feedback:
  a move cursor, a click ripple, any sound.
- **A selection cap**; selecting anything that is not a unit; a box in the standalone viewer.

**Disclosed limitations**, accepted and owned: the relief error the cell resolution carries applies to
**every edge** of the rectangle, so near a cliff a box round one unit can catch its neighbour, and a
box across a sprite's crown that misses the cell under it catches nothing (C-1); a group ordered to one
cell **arrives one unit deep**, the rest stopping where they stand once the first is there and giving
up after sixteen refusals (C-3); and pan and zoom stay live during a drag, so the cells caught are
those under the rectangle at the release.

## Verification mapping

AC-1 … AC-10 are unit tests over synthetic worlds, snapshots and cameras built in test code — no
window, no game install — so all are CI-automatable. AC-11 needs a lawful install and a window. P-1 … P-4 are **sampled, not proved**; P-5 is witnessed at the release and at every frame of
a held press.

## Gate check

FR-1 → AC-1, AC-8, C-2 · FR-2 → AC-2, C-1 · FR-3 → AC-3, P-3 · FR-4 → AC-4, AC-5, P-3, P-4, P-5,
C-3 · FR-5 → AC-6 · FR-6 → AC-7 · FR-7 → AC-8 · FR-8 → AC-5, AC-9, AC-10, P-1, P-2.
