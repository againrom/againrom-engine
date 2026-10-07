# Tasks — 0060 debug readout

Kinds: **impl** (changes the program). Each is one commit trailered `SDD-Task: 0060-debug-readout/T<n>`.

## T1 — the period reads back as a rate, and the box learns to take rows

Files: `pkg/render/terrain/water.go`, `pkg/ui/panel.go`.

Add the inverse of the rate-to-period map beside its forward twin, total and clamped on the same
bounds (DD-5, FR-4). Prove the round trip exhaustively over the whole range and past both ends
(SC-3).

Then split the panel's composition so the shared half takes **already-resolved label/value pairs**
and the subject-to-text half stays where it is (DD-6). No pixel of the unit panel moves: the same
rows, through the same measurement, fit and paint.

Scope fence: no new field, no new layout, no readout yet. Nothing outside these two files.

Done when: the inverse exists with its exhaustive round-trip test; the panel's own suite passes
unchanged; the shared composition is reachable with a hand-built row list.

## T2 — the readout, drawn from what it is given

Files: `pkg/ui/panel.go` (or a new file beside it), `pkg/ui/overlay.go`, `pkg/ui/viewer.go`.

The pushed value type and its setter and reader; the readout's own field constants in the shared
field space and its resolver; its authored layout in the free corner; the widening of the drawn
entity by the one speed scalar; the visibility flag stored so that shown is the zero value; the
compose-and-place path, keyed and cached; and the draw call after the unit panel (FR-1, FR-3, FR-6,
FR-7, FR-9, FR-10, FR-11, FR-12; DD-1, DD-6, DD-7, DD-9, DD-11, DD-12, DD-13, DD-14).

The cursor's cell is resolved through the ground pick at the draw, from a cursor the camera step
stores; the entity count and the frame rate come from what the viewer already holds and from the
engine.

Scope fence: nothing in `pkg/game`, nothing in `pkg/sim`, no key binding, and no edit to the unit
panel's layout, fields, corner or rows.

Done when: SC-1, SC-8 and SC-9 hold; a viewer with no font draws nothing and fails at nothing; a
hidden readout composes nothing.

## T3 — the world pushes what the world knows

Files: `pkg/game/world.go`.

Fill the drawn entity's speed from the simulation's own (DD-10, FR-5). Push the readout from the
paced advance on **every** exit including the stopped one, and from the tick-0 push, reading the
period from the ticker the advance divides by, the stop from the flag the advance tests, and the
tick and digest from the world (FR-2; DD-2, DD-3, DD-4, DD-8). Compute the digest only for a viewer
reporting the readout shown.

Scope fence: nothing in `pkg/sim`; no new writer of the rate; no second clock; the front-end's own
ladder is not read here and not moved.

Done when: SC-2, SC-4 and SC-5 hold — including the assertion that a clock re-rated past either end
reports the clamped rate and **fails if it reports the requested one**.

## T4 — the key

Files: `pkg/ui/app.go`.

Bind the readout's toggle to the diagnostic register's free F-key, as a press edge, read on the map
arm alone beside the lattice's own (FR-8).

Scope fence: one key; no other binding moves; no other arm gains a read.

Done when: SC-6 holds — the press edge toggles it, every other screen's arm cannot reach it, and the
default is shown.

## T5 — the group rate, on a line of its own

Files: `pkg/ui/overlay.go`, `pkg/ui/readout.go`, `pkg/game/world.go`.

Widen the drawn entity by the second rate scalar, state it on its own row beside the speed, and fill
it from the simulation's own (FR-13, DD-15).

Scope fence: nothing in `pkg/sim`; no composition of the two numbers anywhere on this side; no other
row moves and no other field changes.

Done when: SC-10 holds over a fixture whose two numbers differ.

## Traceability

| Task | FR | DD | SC |
|---|---|---|---|
| T1 | FR-4 | DD-5, DD-6 | SC-3 |
| T2 | FR-1, FR-3, FR-6, FR-7, FR-9, FR-10, FR-11, FR-12 | DD-1, DD-7, DD-9, DD-11, DD-12, DD-13, DD-14 | SC-1, SC-8, SC-9 |
| T3 | FR-2, FR-5 | DD-2, DD-3, DD-4, DD-8, DD-10 | SC-2, SC-4, SC-5 |
| T4 | FR-8 | — | SC-6 |
| T5 | FR-13 | DD-15 | SC-10 |

SC-7 is the cost measurement and belongs to the evidence stage, not to a task.
