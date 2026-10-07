# Spec — the walk odometer

Intensity: **spec-first / static**. Terrain: **brownfield** for the live frame selection and the
map screen's per-entity memories, whose behaviour changes; **greenfield** for the odometer
arithmetic itself.

## Problem and current behaviour

A living unit's walk frame is chosen from a counter that advances **once per map-screen tick**. It
was right while every unit crossed a cell in one tick. It is not right now: a unit crosses a cell in
a number of ticks fixed by its own speed, so a slow unit and a fast one both play the same number
of walk frames per second while covering very different ground. The legs and the stride have no
relation at all — the slower the unit, the more it treadmills.

The game does not do this, and the difference is not a tuning constant. Its walk timeline is the
**one animation it does not advance on the tick**. It advances on **distance travelled**: the
drawable carries a running count of how far the unit has walked, and the frame is a function of that
count. Every other timeline in the game — idle, attack, death, the world's ambient scenery — is
advanced by one per tick, and this story moves none of them.

The consequence a reader should hold on to is that the frame count per cell is a property of the
**grid**, not of the unit: a slow unit holds each frame longer and never skips one, and no speed
term appears anywhere in the choice of frame.

## Functional requirements

- **FR-1 — the walk timeline is advanced by distance travelled.** A living unit that is walking
  takes its position **in the walk cycle** from how far it has walked. No tick count, no clock, no
  speed and no cadence setting enters that; which frame that position names is still the class's
  own art at the unit's own facing, exactly as before. The quantity is a running count of travel,
  kept per unit, and it is **presentation state**: it is not simulation state, it is never hashed,
  it never enters a canonical byte form, and it is born and dropped with the map screen.

- **FR-2 — one timeline step is a sixteenth of a cell.** Travel is counted on a sub-cell grid of
  **256 units to a cell along each axis**. The timeline step a unit is on is the accumulated count
  divided by **16**, rounded toward negative infinity, and the frame drawn is that step reduced
  modulo the class's own cycle length. Both numbers are decoded and neither is tuned.

- **FR-3 — a straight cell crossing advances the timeline by exactly sixteen steps.** For every
  class and at every speed, without exception and with no residue carried into the next cell. The
  crossing's ticks **divide the cell between them**, and how they divide it is part of this
  contract: each tick's share of an axis is a whole number of sub-cell units, the shares sum to
  exactly one cell, no two of them differ by more than one, and the larger ones fall on the
  crossing's **last** ticks. A crossing that takes more ticks therefore divides the same sixteen
  steps among more of them; it never produces more steps or fewer, and the sixteen are exact, with
  no part of a seventeenth left owing.

- **FR-4 — a diagonal crossing costs its own euclidean length.** What is added to the count on one
  tick is the euclidean length of that tick's own two-axis share, and the count holds **whole units
  only**: the length is taken to its whole part as it is added, so a fraction a tick does not fill
  is lost rather than carried. A straight tick's share is already a whole number and nothing is lost
  there, which is what lets FR-3 close exactly. A diagonal tick's is not, so a diagonal cell costs
  more than a straight one and less than its exact euclidean length — by an amount that depends on
  how many ticks the crossing took, and that at the very slowest crossings closes to nothing.

- **FR-5 — the count runs on across cells, and only a pause resets it.** Beginning a crossing does
  not reset it, so consecutive cells of one walk continue a single count and a cycle that does not
  finish on a cell boundary carries its remainder into the next cell. What resets it to zero is
  **standing still**: one tick on which a unit neither begins nor continues a cell crossing resets
  the count, and only for a class that has **no idle cycle**. A class that has one keeps its count
  across the pause. A tick inside a crossing is never a stationary tick, however small that tick's
  own share of the cell is, and the tick a unit **arrives** on is inside the crossing it completes —
  the earliest tick that can reset the count is the one after it. "Standing still" is a fact about
  the ground covered and not about the order held: a unit blocked by another for one tick, or one
  whose route search failed that tick, has stood still and is treated as such, which is what the
  game does too.

- **FR-6 — the cycle length is the class's own data, and no alignment is assumed.** How many
  timeline steps make one walk cycle is a per-class number carried by the shipped registry, and
  nothing anywhere requires it to divide sixteen. The great majority of classes are authored so
  that one cycle falls in one cell; a minority are not, and for those the frame at a cell boundary
  legitimately depends on how far the unit has walked since it last stopped. Nothing rounds, snaps
  or realigns the count at a cell boundary, at a cycle boundary or on a change of direction.

- **FR-7 — nothing else moves.** The idle cycle, the standing frame, the fall of a body, the
  world's ambient animation, a unit's facing and the drawn position of a unit between two cells all
  keep exactly the behaviour they have. Every one of them but the drawn position is advanced by the
  tick and stays so, and no game-speed setting — the tick length the two cadence keys move —
  changes the ratio between any of them and the walk.

## Acceptance criteria

- **AC-1** — two units of different speeds crossing the same cell each advance the timeline by the
  same sixteen steps and show the same frames over the same ground; the slower shows more of those
  steps, because it is drawn on more ticks, and the faster skips some.
- **AC-2** — for every crossing length the speed law can produce, a straight cell advances the
  timeline by exactly sixteen steps, and the per-tick shares of one cell sum to exactly one cell.
- **AC-3** — the count after one diagonal cell equals the sum of the per-tick euclidean lengths, each
  taken to its whole part as it is added; it is never less than a straight cell's sixteen steps'
  worth and never more than the exact euclidean length of a cell diagonal, and at the slowest
  crossings the truncation brings it down to the straight cell's own figure.
- **AC-4** — a unit walking several cells without stopping shows a count that is the running total
  over all of them, and a class whose cycle length does not divide sixteen shows, in the order the
  running total predicts, frames at its cell boundaries of which **no two consecutive ones are the
  same**.
- **AC-5** — a class whose cycle length divides sixteen shows the same frame at every cell boundary
  of a straight walk.
- **AC-6** — a unit that stops for one tick restarts its walk from the cycle's first step if its
  class has no idle cycle, and resumes from the count it stopped with if it has one.
- **AC-7** — the frame selection is total: any count, negative included, and any cycle length
  including none at all, answer a frame that the sheet can hold, without a panic and without a
  division by zero.
- **AC-8** — the same world advanced by the same commands reaches the same canonical byte form and
  the same digest at every tick index as it did before this story, and a snapshot built twice with
  no tick between selects the same frames both times.
- **AC-9** — a unit standing still, one that is falling, a body on the ground and the world's ambient
  animation each draw exactly what they drew before this story, at every cadence setting.

## Properties

- **P-1 — the odometer is a function of the walk, not of the frame rate.** Two runs of the same
  world at different cadence settings put a unit on the same timeline step at the same tick index.
- **P-2 — the count is monotone and bounded per tick.** It never decreases while a unit is walking —
  only a stationary tick can lower it, and only to zero — and one tick can add no more than the
  euclidean length of one cell diagonal.
- **P-3 — nothing under the determinism wall moves.** No simulation file changes, and the odometer
  reaches no simulation field, no byte form and no digest.
- **P-4 — building a snapshot does not consume its own input.** The count is advanced by the tick
  that ran, never by the act of drawing; asking for the same tick's picture twice answers the same
  picture.

## Out of scope

- **The idle cycle's own clock.** In the game the walk count and the idle count are one field, so a
  class with an idle cycle resumes its walk where idling left the count. Here they are two, and the
  idle cycle keeps the free-running clock it already has. FR-5's reset is the only place the two
  meet.
- **The second walk-advance arm.** The game carries an alternative arm that drives the phase from
  two per-axis counts and adds a sub-cell drawing offset. Whether anything ever reaches it is
  unestablished, and nothing here implements it or the two counts it reads.
- **Anything that would make the odometer exact against the game's own recorded totals for a
  diagonal.** That total depends on how many ticks a crossing takes, which is the speed law's, not
  this story's.
- **The drawn position of a unit between two cells**, which interpolates smoothly over a crossing
  and keeps doing so. It and the odometer are two readings of one movement and are not unified.
