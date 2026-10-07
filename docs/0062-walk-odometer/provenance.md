# Provenance — the walk odometer

Research pin: `a93d19a8` (`research/`).

## Backing

| Spec anchor | Claim | Confidence, as the row grades it |
|---|---|---|
| FR-1 — the walk timeline is advanced by distance, and it is the only animation that is | `ANIM-WALK-013` | **High** for the arm, transcribed operand by operand, with the four `+1`-per-tick arms quoted beside it; **Medium** that nothing else writes the field (a displacement sweep cannot see a structure copy) |
| FR-1, P-3 — it is presentation state, stepped by the simulation's own tick and stopped when that tick stops | `ANIM-CLOCK-001` | **High** for the field census and the single-caller chain; **Medium** for the same copy blind spot |
| FR-2 — the draw takes the accumulated distance divided by sixteen, then modulo the cycle | `ANIM-PHASE-003`, `ANIM-WALK-013` | **High** — the shift and the modulus are named instructions in the driver and in the body draw |
| FR-2 — travel is measured on a 1/256-of-a-cell grid per axis, so one cell is 256 of them on each axis it moves along | `MOVE-STEP-010`; the direction tables shifted left 8, through `ANIM-WALK-014` | **High**. `MOVE-STEP-010` carries a retracted clause about which cell a boundary hook is passed; that clause is not cited here and the sub-cell grid re-reads unchanged |
| FR-3 — a straight cell advances the timeline by exactly sixteen steps, for every class at every speed | `ANIM-WALK-014` | **High** — an exhaustive re-execution over all 262 shipped (class, definition) pairs, `16:262`, closed additionally by an arithmetic identity that a rounding error would break |
| FR-4 — a diagonal costs the euclidean length of each tick's own displacement, truncated once per tick | `ANIM-WALK-013` (the `FSQRT`/`FIADD`/`ftol` sequence), `ANIM-WALK-014` | **High** for the sequence; **Medium** for the `20:6 21:126 22:130` histogram it produces |
| FR-5 — the odometer is not reset when a crossing starts, and consecutive cells continue one count | `ANIM-WALK-013` | **High** — an address-by-address census of every writer of the field inside the message dispatcher, none of them in the move arm |
| FR-5 — standing still resets it, for a class with no idle cycle | `ANIM-WALK-013`, `ANIM-IDLE-009` | **High** — the two stores are named, and the idle fork is read off the same class scalar |
| FR-6 — the cycle length is the class's own expanded Move track, and the shipped data does not align it everywhere | `ANIM-WALK-015` | **High** for the modulus, the array and the loader's own key binding; **Medium** for the `4:6 8:4 10:4 12:8 14:20 16:208 20:8 24:4` histogram and the 44 unaligned pairs |
| FR-7 — every other animation stays on the tick | `ANIM-PHASE-003`, `ANIM-AMBIENT-016` | **High** — four arms quoted incrementing by one, and the ambient counter enumerated as a third counter with a disjoint owner set |
| FR-7 — no speed setting changes the ratio between them | `ANIM-PACE-017` | **High** — the ladder, the truncating divide and the deadline loop read end to end; ticks per cell, steps per cell and steps per tick are each unmoved by the index |

## Ours by choice

| Choice | What would overturn it |
|---|---|
| **The per-tick share, in closed form.** The engine divides a remaining sub-cell delta by the ticks still owed, once per tick, off a sub-cell position this tree does not carry. We reproduce the same sequence of shares from the crossing's tick count alone: the whole cell split into `q` on the early ticks and `q+1` on the last `256 mod n`. It is the same integer sequence, not an approximation of it. | A crossing whose remaining delta is not a whole cell — a mover interrupted mid-cell, or one whose sub-cell position is not centred when the crossing begins. Either makes the split a function of a position we would then have to carry. |
| **The odometer's home.** A memory of the map screen's own, beside the step, facing and death memories it already keeps, and not a field of the simulation. `ANIM-CLOCK-001` puts the engine's on the client drawable, so this is where the game keeps it too; what is ours is the stricter rule that it stays out of the hashed state and out of the byte form even where a later feature would find it convenient there. | A rule that must be decided identically on two machines from the odometer — a replay that draws, or a game rule that reads a walk frame. |
| **Two clocks where the engine has one field.** The engine's walk odometer and its idle counter are the same word, so a class with an idle cycle resumes its walk from wherever idling left the count. Ours are separate: the walk gets the odometer, the idle cycle keeps the tick clock the animation story gave it. | A class with an idle cycle whose walk visibly restarts in the wrong place — which needs the two to be observed together, and needs the idle clock to become per-entity first. |
| **The reset gate.** The engine forks on the class's idle-phase scalar; we fork on the gate that already decides whether an idle cycle draws at all — the same scalar, and the non-empty track beside it. | A class whose idle scalar is positive but whose expanded idle track is empty. The gate would then say "no idle cycle" where the engine's says "idle", and the odometer would reset where the engine's does not. No shipped class is in that state. |
| **Where the two decoded numbers sit.** The sixteen and the two hundred and fifty-six are kept beside the timeline they belong to rather than beside the movement law, because the only thing either number is for is choosing a walk frame. | A second consumer of the sub-cell grid outside the animation — a drawn sub-cell position, say — which would make one of the two numbers a movement fact with two homes. |

## Open

- **The second walk-advance arm.** `ANIM-ARM-018` transcribes a whole second arm that advances the
  phase from two per-axis odometers instead, entered on a negative class id. The row grades its
  reachability **Unknown** and argues rather than shows it unreachable. Nothing here implements it
  and nothing here assigns a meaning to a negative class id.
- **The two per-axis odometers.** The main arm maintains them and never reads them
  (`ANIM-WALK-013`); they exist for the arm above. We keep neither.
- **The diagonal's total in timeline steps.** `ANIM-WALK-014` measures 20, 21 or 22 over the shipped
  pairs. That number is a function of how many ticks the crossing takes, and our tick count is our
  own rate law's. We reproduce the *derivation* — the euclidean length, truncated once per tick —
  and assert no single total. Closed by a comparison against research's own per-pair figures once
  this tree carries a cost plane and can produce the same crossings.
- **Why the walk is distance-clocked at all.** Recorded as open in the research ledger itself; no
  instruction states an intent and none is expected to.

## Removed

| Dropped | Why |
|---|---|
| A guarantee that a walk cycle begins and ends on a cell boundary | `ANIM-WALK-015` measures 44 of 262 shipped pairs where the cycle length does not divide sixteen. The alignment is an authoring convention the registry mostly keeps, and nothing in the engine requires it, so a contract asserting it would be asserting a property of the data as a property of the code. FR-6 states the absence instead. |
| A per-speed correction that would keep the walk in step with the world's ambient animation | `ANIM-PACE-017` establishes that no speed setting moves the ratio, and `ANIM-AMBIENT-016` that the two coincide at exactly one movement rate and nowhere else. The mismatch is the game's, and smoothing it would be inventing a rule to hide a decoded one. |
