# Analysis — the step a unit takes, and the frames between two cells

## Intensity & terrain

**spec-first / static**, the profile's default for rendering work. **Brownfield** in two
areas — the driver's entity push, which already classifies movement, and the window tier's
four entity glyph paths, which already place a unit on its cell — and greenfield for the
phase seam added beside them. No watcher tool exists, so this is static plus discipline.

## The owner's report, split in two

Two observations arrived together and are not one defect. *A unit ordered one cell away
plays no animation and does not turn.* And *units teleport from cell to cell with no frames
in between.* The first is a bug in what the renderer asks; the second is what one cell per
tick looks like and is a missing feature, not a fault.

## What the renderer asks today, and why it gets the wrong answer

The entity push classifies movement off the entity's **order**: it takes the delta from the
entity's position to its target and calls the entity moving when it holds a target and that
delta is nonzero.

`pkg/sim`'s `Step` sets a target in phase 1 and clears it in phase 2 of the **same call**
whenever the entity reaches it: the move-to arm writes the target, the resolution loop walks
one cell, and the arrival branch clears target, stall count and route together. So the push,
which runs after the step, sees a target that has already gone.

Driven directly against a hand-built 8x8 world, entity at (2,2):

```
order (3,2) applied and walked in one Step  -> X=3 Y=2 HasTarget=false
order (6,2), then one Step per line:
  tick 0  X=4 Y=2 HasTarget=true
  tick 1  X=5 Y=2 HasTarget=true
  tick 2  X=6 Y=2 HasTarget=false     <- the arrival tick
```

So the defect is wider than the report: a **one-cell order is invisible at every tick
boundary**, and so is the **last step of every longer walk**. In both cases the push reads
the entity as not moving, selects the class's idle frame, and takes the facing out of its
memory — whichever direction the unit last walked, or the never-moved facing if it never
has.

The delta the renderer needs is observable without touching the simulation: it is the
change in the entity's own cell between one push and the next. That quantity is nonzero
exactly when the unit moved, and it is the same quantity the interpolation needs, so both
halves of this story read one number.

## Why the jerk is not a defect

An entity advances **one whole cell per tick**, all-or-nothing, since the walking skeleton:
there is no partial move, no sliding along a free axis, and the near search returns cells.
The map opens at 16 ticks a second, so a walking unit changes cell 16 times a second and
the picture between those changes is identical. That is the granularity the simulation is
built on, and the jerk is its faithful presentation.

Two ways to remove it exist and they are not variations of one idea. Drawing a unit
part-way between two cells is a **render** decision: no new simulation field, nothing in
the byte form, nothing in the digest, and the world is advanced by exactly the integer
steps it was before. A fractional coordinate in `pkg/sim` is a **canonical state** change:
the encoding widens, the digest's shape moves, every consumer of a cell learns about
fractions, and collision, occupancy and search all acquire a sub-cell question. This story
takes the first, and the spec says so out loud so a later reader does not read the choice
as an oversight.

## The clock the phase has to come from

The open map screen is paced by a period and an accumulator on the driver: elapsed
microseconds go in, whole ticks come out, and the sub-tick remainder is carried rather than
dropped. The pause is a switch read **before** the elapsed span reaches that accumulator,
so a stopped world consumes its elapsed time and accumulates nothing; the rate control
rewrites the period and keeps the remainder.

That remainder is exactly "how far through the current tick this frame is", and it is
already the one quantity the world's own advance is a function of. Anything else — a wall
clock read on the draw path, or a second accumulator kept beside this one — is a second
clock, and under a pause the two would disagree in the visible direction: the world frozen
and the units sliding.

The remainder is held unexported and the ticker exposes its period and its counter only, so
reading it needs one accessor beside those two. Nothing else about the cadence changes.

## Implicit assumptions checked against the source, not assumed

- **A downed or dead unit cannot drift.** The resolution loop advances only entities that
  are alive, so a unit that is not alive keeps its cell and its observed delta is zero.
  Nothing has to special-case it.
- **A unit felled mid-walk does not drift either.** A blow is applied in phase 1 and clears
  the order it fells a unit out of, and the loop then skips it, so it does not move on the
  tick it dies.
- **The height lookup is total.** The displaced mode's per-cell lift clamps its column and
  row into the grid before forming the corner indices, so asking it for the cell a unit
  came from is safe at any value.
- **The frame selector already freezes when the world does.** The scene clock the frame
  index is selected at advances inside a tick and nowhere else.
- **No front-end but the game pushes entities.** The standalone viewer builds a viewer that
  is never given any, so it produces no entity pass at all.
- **Every construction of the seam value in the tree is a keyed literal**, so a field added
  to it does not silently reinterpret an existing one.

## What we looked at

`pkg/sim/step.go` (the command arm, the resolution loop, the arrival and give-up clearing),
`pkg/game/world.go` (the cadence, the pause, the push and the classification),
`pkg/ui/overlay.go` (the seam value and the four entity glyph paths), `pkg/ui/viewer.go`,
`pkg/ui/app.go` (the one advance call on the map arm), `pkg/render/terrain/water.go` (the
ticker), `pkg/render/terrain/project.go` (the height lookup) and
`pkg/render/terrain/units.go` (the sprite placement). This story asserts no fact it did not
decide itself: every statement in the contract is a choice about our own renderer, checked
against our own code.
