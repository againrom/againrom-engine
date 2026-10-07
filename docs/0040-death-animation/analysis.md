# Analysis — what a unit does when it stops being alive

## Intensity & terrain

**spec-first / static**, the profile's tier for rendering work. **Brownfield** in two areas —
the driver's entity push, which already resolves a class and selects a frame, and the unit
layer's selector, which already owns the sheet's block arithmetic — and greenfield for the
corpse link and the death clock added beside them. No watcher tool exists, so this is static
plus discipline.

## What the tree draws today

A unit at zero health or below is drawn exactly as a live one. The push classifies it moving
or not from the step it took, hands the octant to the one selector, and gets back the move
cycle, the idle cycle or the standing frame. A corpse therefore stands upright and, if its
class carries an idle cycle, keeps playing it forever.

Nothing else about the picture says it is not alive: the seam carries a life byte, and its
three consumers are the health bar (a dead unit draws none, a downed one an empty track), the
selection filter and the order filter. **There is no life-state tint in this tree** — none
was ever built here — so a downed or dead unit is visually identical to a live one at rest.

The sheet's **dying block is addressed by nothing**. The descriptor already computes its
base, because the base of the block after it is a function of the dying block's length, but
no selector reads it.

## What the simulation does with a dead unit

Read rather than assumed. A kill sets health to `-1`, "the largest value that is dead";
damage subtracts an amount but **refuses an entity that is already dead**; there is no heal
and no third writer. Health is signed and never clamped, so an overkill blow lands where it
lands and stays there. A dead entity is not advanced, holds no order, contributes no
occupancy, and is **never removed from the world**: its id is not freed and its record is
still serialized. Nothing anywhere decrements a health over time.

Two consequences that decided this story's shape:

1. A corpse **already persists** in canonical state, for free, for as long as the map is
   open. Nothing had to be added for a body to stay on the field.
2. A health below zero **never moves again**. Whatever the killing blow produced is the value
   forever.

## What the pin says, and where it lands

`ANIM-DEATH-007` (High) with `TERR-SPR-047`, `REG-UNITS-050`, `HERO-DEATH-026` and
`MOVE-ID-016` describe a death animation in two halves.

The **fall** is a run of `2 x DyingPhases` ticks started by the transition into the first
decay stage, and the drawn frame inside it is the dying block's frame at `clock/2` — one
dying frame held for exactly **two ticks**. When the run's counter reaches zero the draw's own
corpse fork takes over on the decay stage alone: stage 1 freezes on dying frame
`DyingPhases - 1`, stages 2/3/4 draw bone frames 0/1/2, and at stage 5 the actor is gone.

The frames of both come out of **another class's sheet**: the dying class-id names the class
whose sheet carries the corpse, and the corpse arm reads that class's own layout switch and
phase scalars in place of the unit's own. The arm subscripts the class table **once**.

The decay stage itself is **not the client's**. It is one byte shipped by the server, set to 1
on the tick health reaches zero or below and advanced by a decay machine — one health per two
ticks — as health passes -10, -20, -40 and -600. The thresholds never reach the drawing side
as numbers; it sees only the stage.

## The lane question, answered before the contract was written

The stage is server state in the original, and this tree has no client/server split, so the
question is whether the stage can be **derived** from health, which is already hashed.

It cannot, and the reason is not our tree's — it is the original's. The death arm sets stage 1
**unconditionally** on the first tick health is at or below zero, whatever health actually
reads; every later stage is produced by the decay walk. So the stage is not a function of
health even where all the state exists. Deriving it from health here would draw bones the
instant an overkill blow landed, which the engine never does.

And our world has no decay walk: a killed unit sits at `-1`, a damaged one at wherever the
blow left it, and neither ever moves again. **The only stage this world can be in is stage 1**
— the fall, then the last dying frame held.

So the whole of what can honestly be drawn today is a pure function of the class's decoded
descriptor, the facing the seam already remembers, and how many ticks have passed since the
unit was first observed not alive — the last of which is render-side, born with the map screen
and dropped with it. No simulation field, no widening of the byte form, no corpse that has to
persist independently of health. **A drawing.**

What the stages past 1 would need is exactly what makes them somebody else's: a dying
countdown per unit and a health that walks downward on a schedule, both inside `Step`, both
hashed. That is recorded and not built.

## Two older baselines of our own, and what survived

An older pair of this project's own clean-room baselines covered this ground. Checking their
open research items against the pin before importing anything is what stopped the import: both
asked for exactly what `ANIM-DEATH-007` and `REG-UNITS-050` now answer, and the answers are
not what the baselines guessed.

- Their block arithmetic — bases summed from separate 16-way and 8-way direction counts, a
  bone base of its own, a `BonePhases >= 3` cutoff deciding whether a class has bones at all
  — is a **different model in kind** from the one `pkg/data` already carries, where the layout
  pair is resolved from the class's own switch and the bone and idle blocks **share** one
  base. Importing it would have overwritten decoded code with a hypothesis, and it would have
  compiled and passed.
- Their cadence was a pair of tuning constants, and the second baseline existed almost
  entirely to **retune** them after a playtest found the fall far too slow. The cadence is now
  a decoded number. There is nothing left to tune, and the whole premise of that baseline goes
  with it.
- Their corpse resolution walked a **chain** with a cycle guard. The corpse arm subscripts
  once. A chain walk is logic no evidence asks for.
- Their tint-gating fix repairs a darkening this tree does not have.
- Their dying-class-supplies-the-corpse reading was flagged as unconfirmed. It is now High.

What survived is the shape of the question and one thing worth keeping: **never vanish**. A
death path that yields no frame must fall back to the drawing that existed before it, not to
nothing.

## What was deliberately not looked at

The corpse's shadow, its overlay sheets, the attack block, and any sound. The four classes
that leave no corpse at all are keyed on a column of a database this tree does not decode, so
that exclusion cannot be implemented here and is recorded rather than guessed at.
