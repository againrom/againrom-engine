# Plan — one observed step, one phase, one displacement vector

## Approach

The driver keeps a per-entity memory beside the world it advances; it gains a second, the
cell each entity stood on before the last advance. The difference between that and the cell
it stands on now is **the step it took**, and it answers both halves of the contract: the
direction and the walk/idle classification are read off it (FR-1, FR-2), and it is the
segment the drawing interpolates along (FR-3). It crosses the seam as a cell delta. Beside
it the driver pushes, once per front-end call, **where that frame stands inside the current
tick**, from the remainder the pacing accumulator already carries (FR-6). The window tier
turns the two into **one world-pixel vector per entity per frame**, added to every glyph
that entity draws (FR-4, FR-5). Nothing is added to `pkg/sim` (FR-7), and a viewer told no
phase adds a zero vector, so it draws today's picture (FR-8).

## Baseline

- `pkg/sim.Step` applies a move-to in phase 1 and resolves movement in phase 2 of the same
  call, clearing target, stall count and route on arrival. Driven over a hand-built world,
  an adjacent-cell order leaves `HasTarget` false at the tick boundary, and a three-cell
  order reads true at its two intermediate boundaries and false at the arrival.
- The push classifies movement as `HasTarget && target-position != 0`, writes a mover's
  sign-octant into a per-entity facing map and reads that map for everyone else. It does no
  cell arithmetic — an entity's X and Y are already whole cells.
- The push runs after the step inside the tick and once from the constructor at scene 0, so
  it has one production call site; but the tree also calls the push and the snapshot build
  in succession, no advance between, to show that reading has no effect. The paced advance
  runs once per front-end call and decides how many ticks fire.
- The pace is a period plus an accumulator carrying the sub-tick remainder, and the ticker
  exposes its period and its counter but not that remainder. The stop is read after the
  baseline is written and **before** the elapsed span reaches the accumulator; a re-rate
  writes the period and keeps the remainder, so a shrunk period can leave a remainder larger
  than itself. A second instance of the same ticker drives the water.
- The seam value carries an id, a cell, art, a selected frame, a mirror bit, a life byte and
  a health pair, and every construction of it in the tree is a keyed literal.
- Four things are drawn per entity, all reaching world pixels from its **cell**: the sprite
  placement, the fallback square, the selection mark and the two health-bar arms. The last
  three share one per-cell transform — lift, camera, view cull — which the three diagnostic
  overlays reach through a cell-list helper, and which the bar instead calls directly from a
  walk of the entities.
- The displaced-mode lift is the mean of a cell's four corner altitudes and clamps its own
  column and row, so it is total at any value. Only the game front-end pushes entities.

## Files to touch

MODIFY `pkg/render/terrain/water.go` (the remainder accessor) and its `water_test.go` ·
MODIFY `pkg/game/world.go` (the step memory, the classification, the phase push) and its
`world_test.go` · ADD `pkg/game/facing_test.go`, `pkg/game/drawn_invariance_test.go` ·
MODIFY `pkg/ui/overlay.go` (the seam field, the vector, the four glyph paths) and
`pkg/ui/viewer.go` (the phase pair and its setter) · ADD `pkg/ui/shift_test.go`.

## Design decisions

### DD-1 — the tick writes the step memory before it steps; the push only reads it

The driver holds a map from entity id to the cell that entity stood on **before the most
recent advance**, and **the advance writes it** — every entity's cell recorded ahead of the
step, inside the one function that is a tick. The push only reads it, which keeps building
a snapshot a **pure, repeatable** read: asked twice with no advance between, it answers with
the same steps, directions and frames. Presence is part of the answer — an id the map has
not seen has taken **no** step, which is what makes the constructor's tick-0 push carry no
step rather than a delta from the map's corner, and an entity first appearing in a snapshot
start at rest. It is born with the map, dropped with it, never iterated, and read by
nothing canonical.

*Rejected:* writing it from the push, which reads more naturally and is wrong — the push
would consume its own input, so a second call with no tick between would report a mover as
stationary, and the tree already calls the push and the build in succession precisely to
show that reading a snapshot has no side effect. *Rejected:* deriving the delta in the
window tier from the previous snapshot, matched by id — a second copy of the world's own
order, living where a world may not be named.

### DD-2 — the step crosses the seam as a delta, not as the previous cell

The seam value gains one field: the cell delta the entity moved by, zero for one that did
not move. The zero value is therefore right for a caller that names nothing — the life
byte's rule — and every keyed literal in the tree keeps its meaning. The previous **cell**
was rejected on exactly that: its zero value is a real cell, so every hand-built seam value
would read as a unit just walked in from the corner.

### DD-3 — the facing memory is written only where a step exists

The classification becomes "the delta is nonzero"; a mover's sign-octant is written to the
facing memory and everyone else reads it, so the write stays idempotent under a repeated
build. The octant table and its unreachable centre are unchanged. Two consequences are
taken deliberately: a unit held up for a tick reads idle and keeps its direction (FR-2),
and a unit that is not alive can never write a facing at all, because the resolution loop
advances only the living — so a fallen unit keeps its last step's direction with no rule of
its own.

### DD-4 — a read-only remainder, read from the driver's pacing clock and no other

The clock type gains one accessor beside its period and its counter, reporting the
microseconds accumulated toward the next tick. **The one read is the driver's own pacing
clock, never the viewer's water instance**: those are two instances of the same type,
re-rated through two separate call sites and each carrying its own remainder, so the phase
has to come from the clock that decides when a tick fires (C-3). *Rejected:* an accumulator
of the drawing's own, the second clock C-3 refuses. *Rejected:* exporting the field, which
would make a remainder settable and so let a clock disagree with itself.

### DD-5 — the phase crosses as two integers, pushed by the paced advance

The viewer takes an elapsed and a period as plain ints and stores them; the paced advance
pushes the clock's own two values after its tick loop. Two ints and not a fraction: the
displacement lands on whole render pixels, and this seam's values are scalars by rule. The
stopped path returns **before** the elapsed span reaches the accumulator, so the phase last
pushed is still the one in force and a pause needs no statement here (FR-6).

*Rejected:* pushing a ready-made per-entity displacement from the driver — pixel geometry in
the tier that owns a world, and a rebuild of the whole snapshot every frame where the
snapshot is a per-tick value.

### DD-6 — one displacement vector per entity per frame

One viewer-side function takes a seam value and returns this frame's world-pixel
translation: the entity's step in cells, negated, times the cell size, times the part of the
tick still to run over the whole period. The sprite adds it to its placement's top-left; the
square, the mark and the two bar arms add it to their world-pixel arm **before** the
transform they share. That transform — lift, camera, cull — keeps its single copy, so the
cull runs on the rectangle actually drawn and the four parts of one unit cannot separate
(FR-3, P-3).

*Rejected:* displacing in screen pixels after the camera, where the zoom would multiply an
already-rounded value and the cull would run on the undisplaced rectangle.

### DD-7 — the relief is folded into the same vector

The shared transform lifts a glyph by the **entered** cell's own height, so the vector's
vertical term also carries the difference between the left cell's lift and the entered
cell's, scaled by the same part of the tick: the drawn height then runs between exactly
those two values across the tick (FR-4). The left cell is `cell - step`, and the lift lookup
clamps its own indices, so it is safe at any value. Flat mode adds nothing — the term is
reached only where the transform lifts at all.

### DD-8 — the clamps live with the arithmetic

A zero step returns the zero vector before anything is computed (FR-5). A non-positive
period returns it too — that is a viewer never told where it stands, and it is what makes
the standalone front-end and every existing test draw what they drew before this story
(FR-6, FR-8). A remainder outside its period is brought inside it, the state a re-rate
leaves behind when the new period is shorter than the remainder already accumulated;
unclamped it would throw an entity past the cell it entered.

### DD-9 — the two entity glyph families get a walk of their own; the three overlays keep theirs

The square and the selection mark are built today from bare cell lists handed to the shared
cell-list helper, which the object, unit and static-object overlays also use. Both move to
**per-entity walks on the health bar's existing model** — that one already builds its arms
and calls the shared per-cell transform directly — so each glyph holds its entity, and so
its vector, without a cell list in between; the entity split therefore stops handing back a
bare cell list for the squares. The three diagnostic overlays keep the cell-list helper and
its signature unchanged: they have no entity, and converting their lists per frame would
allocate over thousands of cells for a vector that is always zero.

*Rejected:* widening the shared helper and the per-cell transform to carry a vector for all
five families — three of them would pass a zero they can never use, and the two that need
one would reach it through a signature shaped for the three that do not.

## Risks

- **R-1** A unit whose bar or mark trails its sprite reads as a rendering fault, not a
  smoothing. *Mitigation:* one function computes the vector and every glyph adds that same
  value; SC-3 compares the three families per entity at four phases.
- **R-2** The entity layer follows the world by up to one tick, so an order can feel late
  (C-2). *Mitigation:* disclosed in the contract; SC-9 reads it in a real run at the shipped
  rate, where the delay is about 62 ms.
- **R-3** A unit repeatedly held up now alternates walking and idle frames. *Mitigation:*
  disclosed; the frames report what the unit did rather than what it intended, which is this
  layer's rule everywhere else.
- **R-4** A paused world whose units slide, or a resume that jumps, stops the pause reading
  as a pause. *Mitigation:* the phase is the pacing clock's own remainder, which a stop never
  advances (DD-5); SC-6 drives many stopped frames and a re-rate.

## Success criteria

1. **SC-1** *(automated)* AC-1 and AC-2 hold, over a driver advanced one tick at a time with
   the pushed entities read between them — FR-1.
2. **SC-2** *(automated)* AC-3 holds, the direction read off the push and not off the memory
   behind it — FR-2.
3. **SC-3** *(automated)* AC-4 holds, the three glyph families compared **per entity** at
   each phase rather than family by family — FR-3, P-3, R-1.
4. **SC-4** *(automated)* AC-5 holds over a grid whose two cells differ in height — FR-4.
5. **SC-5** *(automated)* AC-6 holds at every remainder, in all four resting states — FR-5,
   P-4.
6. **SC-6** *(automated)* AC-7 holds in each of its three arms, the stopped geometry
   compared value for value across many frames — FR-6, P-2, R-4.
7. **SC-7** *(automated)* AC-8 holds, the digests compared at every tick index, and
   `pkg/sim`'s pinned fields and both wall checks unmoved. A snapshot built twice with no
   advance between is equal to itself — FR-7, P-1.
8. **SC-8** *(automated)* AC-9 holds against the tree as it stands, and again after — FR-8,
   P-5.
9. **SC-9** *(manual)* AC-10, run against a lawful install — FR-1, FR-3, FR-4, FR-6, R-2,
   R-3.
10. **SC-10** *(automated, mutation)* Two mutants of the load-bearing arithmetic — the tick
    part taken as elapsed rather than as what remains, and the remainder clamp removed — are
    each applied to production code, the tree run with the failing tests named, and reverted
    with byte identity confirmed.

## Traceability

FR-1, FR-2 → DD-1, DD-2, DD-3 → SC-1, SC-2 · FR-3 → DD-2, DD-6, DD-9 → SC-3 ·
FR-4 → DD-7 → SC-4 · FR-5 → DD-8 → SC-5 · FR-6 → DD-4, DD-5, DD-8 → SC-6 ·
FR-7 → DD-1, DD-5 → SC-7 · FR-8 → DD-8 → SC-8.
