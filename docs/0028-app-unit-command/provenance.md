# Provenance — selecting a unit and ordering it from the running game

Every normative sentence in the contract is the project's own engineering over the project's own code.
Two things the baseline stated as established are demoted below, and two are recorded as open rather
than defaulted.

## Backing

Empty, and that is a fact about the story rather than a gap in the ledger: no claim is consumed, so
the story moves no research pin, and nothing in the contract asserts a property of the original game,
of a game file or of any decoded structure. The decoded values it stands on — the terrain projection
and its vertical origin, the logic-tick cadence, the per-unit cell conversion — arrive through values
earlier stories established and are not re-asserted here.

## Ours by choice

| Contract anchor | What the evidence says |
|---|---|
| **Left tap selects, right press orders, left drag pans** (FR-1, FR-3, C-3) | Nothing establishes it. The scheme is the real-time-strategy genre's convention and is adopted **by choice**; whether the original bound its buttons this way has not been determined, and no third-party remake, spec or source was read to arrive at it |
| **A four-pixel tap slop, measured as movement accumulated while the button was held** (FR-1) | No source. Accumulating the same per-tick deltas the shipped drag already pans from makes "the view panned" and "this was a drag" one question rather than two that can disagree |
| **The highlight sits on the cell, carries the relief offset, and leaves the unit's art and any marker visible** (FR-2) | Ours, reusing the cell-marker convention the diagnostic glyphs already carry — though the visibility clause is forced by the frame's own pass order rather than preferred |
| **Orders apply after the tick's scripted commands, how many applied is reportable, and an advance with none behaves exactly as before** (FR-4, FR-5, P-3) | Ours. Last-write inside one step is the shipped rule, so putting the player's orders last is what makes the player win, and this is the only place that choice can be made. The count is there to be a witness rather than a feature — see *Open* for what it cannot witness — and the unchanged-advance clause is the regression fence that makes "no phantom command" checkable rather than assumed |
| **A commanded unit leaves the placeholder script for good** (FR-6, C-4) | Nothing bears on it. The script is our own placeholder, so what it yields to and when is entirely ours to define |
| **The pick is the flat ground cell in both terrain modes, and the lower id wins a shared cell** (FR-7, C-1) | Ours, and nothing describes the original's hit test. Correcting for the projection's vertical origin was measured against the drawn geometry and rejected, not merely skipped: a cell draws at `row*32 - height - origin`, and the origin is the negated greatest height of the top row, so under uniform relief the two cancel and the uncorrected pick is exact where a corrected one is wrong by the whole relief. The tie needs an answer that is not "cannot happen": occupancy makes a shared cell unreachable in a well-formed world, but advancement advances a malformed one rather than repairing it |
| **Ordering is gated on the map extent, in the front-end** (C-2) | Ours, and the gate has to live somewhere, because our own step applies no bounds clamp |
| **One pure function decides the selection and the order, and the standalone viewer gains none of it** (FR-8, FR-10, P-5) | Ours twice over, and no source speaks to either: both are questions about how our own front-ends are built |
| **Selection, pick, highlight and pending orders are front-end state alone** (FR-9, P-1, P-2, P-4) | The determinism wall applied to this story rather than anything new, so nothing outside our own rules bears |

## Open — assigned no meaning by the contract

- **What the original's own mouse scheme is.** Undetermined. The convention is adopted, not reproduced,
  and if a later finding contradicts it the constraint that changes is C-3; nothing else depends on the
  binding.
- **Where the game's cursor hotspot sits.** The cursor art is decoded and available; the hotspot is
  **not** a decoded fact. The story that draws it chooses one — the glyph's top-left corner and its
  drawn tip being the two candidates — and must record that as a choice. Nothing here establishes it and
  nothing here depends on it: the pick reads the pointer position the engine reports, which no drawn
  glyph moves.
- **The world clock's honest rate range, and what a stop is.** Measured against the shipped cadence,
  not guessed. The period is `1000 / rate` in whole milliseconds, so the mapping is faithful only to
  about 100 ticks a second: a rate of 256 truncates to a 3 ms period, about 333 ticks a second, and a
  contract claiming 256 would claim something the cadence cannot deliver. Separately, an out-of-range
  rate index is pulled back into the shipped nine-row table whose slowest row is 8 ticks a second, so
  **a rate of zero is the slowest rate and not a stop** — a stop has to be a switch beside the rate.
  Two questions are left open rather than defaulted: whether the control should be stated as a rate at
  all, and whether the world's rate and the water layer's rate move together. They are two separate
  tickers today, started from the same row.
- **Whether replay idempotence holds within one advance's command list.** Undecidable while move-to is
  the only kind of command: last-write makes a doubled application indistinguishable from a single one,
  so no test over world state or over digests separates them. The contract witnesses the boundary — one
  order, one advance, never a later one — and says plainly that the interior is argued (P-6). The
  witness arrives with the first command that accumulates.

## Removed from the baseline and why

- **"The original's scheme."** The baseline's own provenance paragraph called left-select / right-move
  the original's scheme while also calling it a genre convention. The first half is established by
  nothing available here and is dropped; the second survives, as a choice.
- **Exposing the camera's screen-to-world inverse, as a deliverable.** It is already exposed and already
  called by the displaced draw path. What was missing is the cell step and the extent answer.
- **"The order overrides the demo target" as the whole of the override.** True for one step and false
  after it: the placeholder script issues a fresh target for every unit every twenty-four ticks, so the
  clause describes a feature that stops working within two seconds. Replaced by FR-6.
- **"The ring demo, issued at tick 0."** A fact about a previous build; this script walks a square in
  laps and issues orders for hundreds of ticks.
- **A knob setting one world tick per frame, and a criterion driven through the draw path.** Neither
  exists; the advance is paced from a clock, so the criteria are written over advance requests.
- **A criterion asserting the standing build, test, format and asset-guard gate**, which is the
  repository's obligation and not a property of this change.
