# Plan — the player can order an attack

## Decisions

- **DD-1 — a THIRD seam, not a widening of the move seam.** `MapAttack func(entity, victim uint32)`
  goes beside `MapOrder` and `MapAffect`, appended last to the loader's tuple. This is 0058's
  decision made again on the same grounds, re-tested rather than inherited: widened, `MapOrder`
  becomes `(entity, x, y, victim, attack)` and **every existing call site has to say it is not an
  attack**, while every move carries a victim it never means. The case differs from 0058's in one
  way and argues the same direction — an attack names an ENTITY where a move names a CELL, so a
  widened seam carries two mutually dead payloads rather than one dead flag. Rejected also: reusing
  `MapAffect`, which applies a blow directly and is not an order.

- **DD-2 — the consuming press is the SECONDARY button.** The decoded shape is *arm a mode, then
  click*, and that click is the original's primary because the primary is that engine's act button.
  Here the primary selects and the secondary orders (0028 FR-3); moving *which button acts* would
  rewrite the tap, the box and the slop accumulator for a story about orders. The secondary press
  reproduces what matters — one press, two possible orders, chosen by what is under the cursor —
  and leaves the select path untouched.

- **DD-3 — the arm lives on the `Viewer`, beside the selection.** Not on the `flow`: the selection
  it is gated against is the viewer's, and two owners would be two lifetimes to keep level. It also
  discharges FR-1's last clause for free — the viewer is dropped when the map screen is left, so
  the arm goes with it and there is nothing to forget.

- **DD-4 — `decide` returns ONE decision list, and the internal order type carries its kind.** The
  seam stays two functions (DD-1); this is one level below it. One list keeps *ascending id* a
  property of one walk rather than of two kept in step, and the kinds are exclusive per frame, so a
  second slice would be empty on every frame the first was not. Rejected: a second pure function for
  the armed arm, splitting the press's outcomes across two bodies and putting P-2's totality out of
  any test's reach.

- **DD-5 — the arming gate is ONE predicate, and it compares an OWNER.** It calls `presentSelected`,
  the same filter the orders, the marks and the blow keys read, so *"a selection that can be ordered
  to attack"* and *"a selection an order is emitted for"* cannot come apart; then it compares that
  set's primary owner against the local participant, which is `AI-PANEL-061`'s own bit. **Rejected
  outright: a unit-class test** — the clause that would have produced one is struck through and
  `AI-PANEL-060` reads the routine as never touching the selection, so a control greyed out by class
  is a rule the game does not have, which is worse than a missing rule because nothing makes it
  visible.
  The local participant crosses as **one `uint32` beside the selection, not as a bool per entity**,
  and the reason is where the comparison lives: the decoded routine makes it in the view, so moving
  it behind the seam would put the rule on the far side and hand this tier an answer it could not
  check. Two builtins name no simulation type, which is what makes the faithful placement legal
  here. Zero is *none established*, sharing the zero 0071 already gives a unit owned by nobody, so
  the absence costs no second flag and shadows no 1-based slot.

- **DD-6 — the key is `F`, on the letter register.** An action takes a letter, a diagnostic a
  function key (0058 DD-11). `A` is the camera's pan-left; `K` and `L` are the blow keys. Nothing
  decoded says what the original bound and this asserts nothing.

- **DD-7 — the approach RE-AIMS every turn, inside the move loop.** One helper per alive attacker
  holding a victim, **after** the crossing check and **before** the target is read: after, so a
  mover that arrived mid-stride finishes the crossing it began; before, so the destination the rest
  of the loop reads is this tick's. Re-aiming rather than aiming once at the ordered cell is what
  makes a moving victim followed; a one-shot destination walks to where the victim *was*, which the
  original does not do and which reads as a bug rather than a simplification. The **cost is
  disclosed**: the staleness test has no period, so a moving victim buys one far search per victim
  step. A stationary one buys none — the destination does not change and the stored route still ends
  on it.

- **DD-8 — the stop distance is `inReach`, called.** The blow is already refused outside a Chebyshev
  cell by one function; the approach stops at that same call rather than at a second comparison, so
  *"walked close enough"* and *"close enough to strike"* are one predicate and a later story moving
  reach onto the entity moves both at once. `AI-CMD-054` puts the same number in `ord+0x14` and in
  `actor+0x12c`, which is the shape this reproduces.

- **DD-9 — the approach ENDS an order rather than merely stopping it.** Reaching the victim and
  losing it both call the loop's own `restAt`, the call an arrival makes, so an attacker's residue
  is dropped by the rule that drops everybody else's — and a flyer's occupancy is right on the tick
  it rests, which is why `restAt` exists.

- **DD-10 — no `formatVersion` bump, and the check is stated rather than assumed.** The approach
  writes `TargetX`, `TargetY` and `HasTarget`, which every mover carries and the byte form already
  encodes; no field is added and no width moves, and both the constructor and the decoder already
  admit an entity holding a destination and a victim at once — neither has a clause forbidding the
  pair. **No bump is taken. The number allocated to this story was corrected mid-story from 13 to
  14 — 13 went to 0076, which reaches hashed state by construction — and 14 is not taken either.**
  The check is a test, not a claim: `TestAnApproachRoundTrips` asserts the encoded version byte is
  **12** by number, so a later change that did add a field would fail here rather than pass quietly.

- **DD-11 — the far side MARKS the attacker commanded, where the blow seam does not.** The commanded
  set stops the placeholder script overwriting a player's order; an attack IS an order, so a unit
  that took one leaves the script exactly as a move order's does. A blow says nothing about what a
  unit should do and does not mark. It is the one line where the two far sides differ.

- **DD-12 — the armed state is stated on the readout, not drawn.** `AI-CURSOR-052` binds the attack
  cursor to its own art at High, so a cursor swap is a drawing-tier story with real assets; a
  readout row is a diagnostic, costs one field and one row, and asserts nothing. Without it a
  one-shot armed mode is invisible.

- **DD-13 — what is deliberately NOT built, each because the game does not have it.** No retaliation
  (`AI-RETAL-056`). No test of any kind at the PRESS — the ownership gate is at the arming key and
  nowhere else, because the click-time test is a runtime-class test (`AI-CLICK-050`). No second
  behaviour tree by owner (`AI-ARBITER-057`). No acquisition, guard or aggressive stance: those
  write the same two per-actor states from the AI's side (`AI-STRIKE-055`), the other half of the
  pair this story opens.

- **DD-14 — the 253 fan-out ceiling is NOT imposed, and that is a decision.** The ceiling sits on
  the command record's **shared** id append, which the move builder uses as much as the attack
  builder; imposing it here would put half a rule in one story and leave the move press — whose
  contract orders *every* present member (0030 FR-4) — on the other half. And the overflow is not a
  truncation one could reproduce: the walk is a hash order, so what survives is unpredictable where
  ours would be the lowest ids. Diverging deterministically and saying so is the honest form; an
  id-ordered cap would look like the rule and be a different one.

## Success criteria

- **SC-1** The arming key raises the flag over a selection with a present unit and leaves it down
  otherwise; a second press lowers it; a selection replaced while it is up leaves it up; and with a
  local participant established it answers on the primary present id's owner and on nothing about
  its class. *(FR-1; AC-1, AC-1a, AC-15)*
- **SC-2** An armed press over a drawn unit yields attacks and no moves; over empty ground it yields
  exactly the unarmed press's moves. *(FR-2, FR-3; AC-2, AC-3, AC-6)*
- **SC-3** The flag is down after every secondary press made while it was up, including one that
  yielded nothing, and an unarmed press never yields an attack. *(FR-2; AC-4, AC-5)*
- **SC-4** The seam carries two ids, is nil-safe, and its far side appends one command, marks the
  attacker and advances nothing. *(FR-4, FR-5; AC-7)*
- **SC-5** An attacker walks to a distant victim and lands a blow; an adjacent one never moves; a
  moving victim is followed; a lost victim ends the walk; a move order ends a fight whole. *(FR-6;
  AC-8, AC-9, AC-10)*
- **SC-6** `formatVersion` is 12 and a world holding an attacker mid-approach round-trips. *(FR-6,
  P-3; AC-11)*
- **SC-7** The readout row states the flag, and leaving the map screen clears it. *(FR-7, FR-1;
  AC-12, AC-13)*
- **SC-8** The gate is clean and `pkg/ui` still names no simulation type. *(FR-8, P-1, P-2, P-4)*
- **SC-9** A press over more than 253 present units orders every one of them. *(FR-9; AC-14)*

## Risks

- **R-1 — the approach supersedes a shipped clause and its tests.** 0064 FR-2's *"never both"* is
  asserted by name in `pkg/sim/combat_test.go`. The tests move in the same commit as the behaviour
  and the superseded half is re-asserted in its new form, so no assertion is deleted without one
  standing where it stood.
- **R-2 — an attacker that cannot reach its victim re-searches every tick.** One whole-map sweep per
  pursuing unit per tick, reachable only where the far search settles. Disclosed in DD-7 rather than
  guarded; the guard would be a period, and a period is state.
- **R-3 — the armed press changes an existing branch of `decide`.** The new arm goes into that
  branch chain rather than beside it, so a frame producing both an attack and a move would fail a
  totality test that already exists.
