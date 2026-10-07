# Spec — the cadence ladder

The two cadence keys must move the game's own speed setting, not a model of our own laid over the
top of it. Today they move ours: a map opens at one cadence and the first press replaces it with a
different one that names the same number of ticks a second, and the cadence the map opened at
cannot be reached again by any sequence of presses.

## Context

Two independent starting values exist in two places and disagree from the first frame. The world's
clock opens at the **game's own** period for its default speed setting: that setting is 16 ticks a
second, the game computes a tick as `1000/16` milliseconds by integer division, and the result is
**62 ms — 62 000 µs**, not 62.5. The front-end's key ladder opens at a **rate** of 16 in a model of
ours that is carried to the microsecond precisely so it does *not* truncate, and its period is
therefore **62 500 µs**. The first press writes the clock from the ladder, so our value silently
replaces the decoded one and 62 000 becomes unreachable.

**62 000 is not a defect and does not move.** The truncation is the game's own arithmetic and the
992 ms water cycle follows from it; that cycle is a load-bearing figure elsewhere. The fix is to
the ladder.

## Functional requirements

- **FR-1 — the ladder passes through the game's settings.** There is one ordered ladder of
  cadences. Its middle is the game's nine speed settings, one rung each, at their own periods; `+`
  moves one rung faster and `−` one rung slower. Outside those nine the ladder continues into a
  disclosed extension, reached only past the ends of the shipped set, whose two ends are the same
  outermost cadences the front-end could reach before this story. The cadence a map opens at is a
  rung, and every rung is reachable from it in both directions.

- **FR-2 — the opening cadence does not move.** A freshly opened map runs at the game's own period
  for its default setting, and no code path may recompute that period from a rate. A cadence key
  that changes only the stop writes back the period already in force.

- **FR-3 — the round trip is exact.** From any rung, n presses of `+` followed by n presses of `−`
  return to that rung, for every n whose run does not reach an end of the ladder. At an end the
  ladder **saturates**: a press that would leave it changes nothing, is not remembered, and writes
  nothing across the cadence seam.

- **FR-4 — the screen says which side of the shipped set it is on.** The readout states, for the
  cadence the world's clock is holding, either which of the game's nine settings it is or that it
  is our extension and how far past which end. A cadence that is on no rung at all states neither
  and says so. The match against the nine is exact; the nearest is never named.

- **FR-5 — one arithmetic.** The meaning of a cadence key is decided in exactly one place. The
  value that crosses to the two consumers of a cadence — the world's clock and the ambient
  animation counter — is a **tick length in microseconds**, computed once on the statement that
  changes it and adopted verbatim on both sides. Neither consumer computes a period of its own.

- **FR-6 — the readout still reads the clock.** Everything the readout states about the cadence,
  including FR-4's line, is derived at the moment it is stated from the period the world's clock is
  holding, and never from the key ladder or from the animation counter.

## Acceptance criteria

- **AC-1** — a freshly opened map's cadence is a rung of the ladder, and it is the game's default
  speed setting at that setting's own period.
- **AC-2** — the ladder's middle nine rungs are the game's nine settings at the periods the game's
  own arithmetic gives; the rungs outside them strictly quicken and strictly slow away from those
  ends, and the ladder's two extreme rungs are the same two cadences the front-end could reach
  before this story.
- **AC-3** — for every start rung and every press count, `+` n times then `−` n times lands on the
  rung the saturating ladder predicts, and on the start rung itself whenever the run stays on the
  ladder.
- **AC-4** — a press moves the world's clock and the animation counter to the same period, and over
  a driven second both fire the same number of ticks.
- **AC-5** — a press past either end makes no cadence call at all.
- **AC-6** — the opening period is the game's truncated one and not the untruncated period our own
  rate model gives for the same number of ticks a second.
- **AC-7** — the readout states which of the nine on every shipped rung, which side and how far on
  every extension rung, and the absence marker for a period on no rung.

## Properties

- **P-1 — total.** Every ladder function answers for every input: a rung outside the range names
  the nearest one inside it, and a period no rung produced still resolves rather than refusing.
- **P-2 — no wrap, no drift.** The ladder saturates at both ends. A press never wraps around, and a
  sequence of presses never leaves the ladder standing anywhere but on the rung its net movement
  names.
- **P-3 — nothing under the determinism wall moves.** No file under `pkg/sim` is touched, and the
  same command stream advanced at every rung and through a stop-and-resume schedule reaches the
  same canonical byte form and the same digest at every tick index.

## Out of scope

- Persisting a chosen setting between sessions.
- Any change to what a unit's own speed means. A unit's ticks per cell is fixed by its speed; the
  cadence decides how long a tick lasts. They are different quantities and stay so.
- The stop, and what does and does not keep running while it is set.
