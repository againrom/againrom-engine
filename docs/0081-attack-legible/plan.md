# Plan — a fight is legible

Derived from `spec.md`. Every decision below is settled here; tasks decompose it and invent nothing.

## Shape

Four tiers, in dependency order: the simulation gains the field and its two writers; the data tier
finishes a derivation it already starts; the render tier gains a third pure selection; the drawing
seam stops holding its own answer and dispatches over the three. A developer tool gains one printed
value so the whole of it is measurable headlessly.

## Design decisions

- **DD-1 — the facing is a `uint8` on `Entity`, in the decoded field's own units** (FR-1, FR-2). Not
  a direction index in `[0,8)` and not a named type. The engine's field is a byte in 32-unit steps and
  the arc arithmetic a turn rate needs is over that byte, so an index is the one representation the
  story that gives the turn a cost cannot proceed from without a second form change. It goes on
  `Entity` because every field there is an integer or a bool and the type is copied by assignment.

- **DD-2 — no refusal and no not-alive clause** (FR-1). Every one of the 256 values names a direction
  under DD-3's total map, so there is nothing to fold onto and nothing to reject. The four felled-unit
  clauses beside it refuse residue of a state the unit has left, and a facing is not one.

- **DD-3 — three total conversions, in one file, and each is one expression** (FR-2).
  `FacingDir(facing)` is `((int(facing) + 16) >> 5) & 7`, the decoded rounding, computed in `int` so
  the `+16` is visible rather than resting on the wrap and the mask agreeing; it is the ONE name this
  story exports, because a direction index is all the seam and the tool ever need.
  `facingOfDir(dir)` is `dir << 5`, and `facingToward(dx, dy)` a lookup into a `[3][3]` sign table
  holding the eight directions and a ninth entry reporting "none"; both stay unexported. The delta
  table is written out as `MOVE-DIR-034` states it rather than transcribed from the render tier's
  octants, and a test asserts the two against each other — which is what makes DD-9's rotation a
  measurement instead of a constant.

- **DD-4 — the mover's write is at the step and is one statement** (FR-3). In `Step`'s move loop
  immediately after `e.X, e.Y` are assigned, reading the same `from` cell the plane update reads, and
  above the rate call so the terrain-cost arm and this one do not touch one line. Nothing else writes
  a facing on a movement path — not the give-up, not the arrival, not the crossing — so "a facing is
  the last step's" holds by there being one assignment. It lands on the crossing'''s FIRST tick, when
  the cell is committed, so a rated mover points along the stride it is drawn sliding down.

- **DD-5 — the attacker's turn is on `approach`'s STOP ARM, not in `advanceAttack`** (FR-4). The
  first draft put it at the top of the attack turn, and `AI-FACE-066`/`AI-FACE-067` refute that
  placement: the swing start and the strike hold no facing test, no facing write and no call to any
  turn routine, the producer set is six routines and none is on that path, and the claim says in
  terms that a consumer turning the attacker as part of the swing has invented a coupling. The turn
  belongs to the approach, whose in-reach arm already ends the walk — the original's own
  stand-and-face — so it goes there, guarded by the SAME `inReach` that arm already calls.

  Two consequences fall out, both improvements: the turn no longer reaches a mover mid-crossing —
  `approach` sits behind the move loop's transit skip — so FR-3 has no exception, and the attack phase
  is left holding no facing code at all.

- **DD-6 — the byte form takes the field at the record's tail, `+91`, and the version is 14**
  (FR-6). `entityLen` 91 → 92. Version 13 widened the PLANES, not the record, so every offset inside a
  record runs unbroken from version 12 to here and a version-13 buffer differs from this one by
  exactly one byte per record — which makes refusing it the sharpest refusal in the sweep.

  **The tests spell 91 out by hand rather than reading `entityLen`**, deliberately and since the form
  existed, so every one fails here and is re-stated in the same commit. That is the mechanism working:
  a test computing the width off the production constant would have passed silently.

- **DD-7 — the data tier's `Anim()` gains three fields and no arithmetic** (FR-7). `AttackSlot` is
  `phaseCount(c.AttackPhases)` — the same clamped scalar the same expression already multiplies into
  `DyingBase` and `TailBase`; `AttackTrack` is `expandTrack(c.AttackAnimTime, c.AttackAnimFrame)`;
  `AttackOK` is `at > 0 && len(AttackTrack) > 0`. The gate therefore tests the CLAMPED scalar, which
  is what the two shipped gates test and not the resolved one — the clamp maps every non-positive
  value to 0, which fails `> 0` exactly as the value it replaced did, so the two are one predicate
  written over fewer names. The render tier's mirror takes the same three names, field for field.

- **DD-8a — the run is the ART'S length, indexed with no modulus, and it ENDS** (FR-8). The first
  draft reduced the clock modulo the track's period, authored because the arm's arithmetic was
  transcribed nowhere. `ANIM-RUN-004`, `ANIM-PHASE-003` and `ANIM-STATE-023` transcribe it: one per
  tick from zero, no modulo, exactly `len(AttackAnimTime expansion)` ticks, then forced back to state
  0. So the modulus goes and a clock at or past the length is REFUSED — the caller's fall-through is
  what that forcing looks like from the drawing side. The seam working as intended: a value changed,
  the structure did not.

- **DD-8 — a separate `SelectAttackFrame`, not a fourth arm of `SelectUnitFrame`** (FR-8). Two
  reasons, the second load-bearing. The arm would need a flag and a clock, giving `SelectUnitFrame`
  eight positional parameters of which two are adjacent bools. And a refusal must reach the caller's
  fallback: `SelectUnitFrame` answers a bad index as frame 0 unmirrored, the right total answer
  *there* and the wrong one here — exactly the argument `SelectDeathFrame`'s third return already
  makes, so this is the shipped pattern applied again. The file's header rule becomes "one selection
  per DRAWN state, and the caller chooses" — three functions, one dispatcher.

- **DD-9 — the seam translates, and the translation is `(dir + 4) & 7`** (FR-10). The two orderings
  differ by a rotation of four: `MOVE-DIR-034` numbers clockwise from north, the sheet's octants from
  south. The constant is not chosen — `AC-9` asserts it against `signOctant`, the derivation the tree
  already draws movers with, over all eight deltas, and that assertion is written **before** the
  memory it replaces is deleted so the two answers are compared while both exist.

- **DD-10 — the octant memory and its sign derivation are deleted** (FR-10, P-3). `mw.facing`,
  `signOctant`, `signOctants` and the now-callerless `signIndex` go. Keeping them beside a canonical
  facing would be two sources for one fact, free to disagree after any write that moves one and not
  the other — the failure the world's own field-set pin exists to refuse, one tier up. The step delta
  itself STAYS: `moving` and the odometer read it, and neither is a direction.

  **Three test files build `mapWorld` literals naming the deleted field, and one pins the old
  default** — a never-moved entity asserted to draw octant 0, south. That assertion is the shipped
  statement of the behaviour R-2 changes, so it is REWRITTEN to north rather than dropped.

- **DD-11 — the swing clock is a seam map keyed by id, never ranged, counting ONE RUN** (FR-11). A
  `map[sim.EntityID]int` beside a `map[sim.EntityID]sim.AttackPhase`, walked over the world's own
  entity slice so the write order is ascending id and never a Go map's range order, and assigned
  rather than deleted, as the death clock and the odometer already are. Advanced AFTER the step, from
  post-step state, because that is the state the push then selects on.

  **It restarts at the swing START and not at "began drawing"**, and the phase memory is what that
  costs: the run begins the tick the attacker leaves the ready phase and loads its countdown, the
  instant the engine sends the message that enters the drawn attack state. The first draft's
  drawn-tick count would run unbroken across a whole fight and draw one swing ever, because DD-8a's
  run ends.

- **DD-12 — "swinging" is holding a victim and not `moving`** (FR-9). The seam already derives
  `moving` and reads the entity's attack field, so the conjunction adds no state and no lookup.
  **`moving` is not "moved this tick"**: the step memory holds its cell across a crossing, so the flag
  means *is mid-stride, or moved* — stated because a fixture written from the looser wording would be
  the wrong fixture. It does NOT ask whether the victim is in reach: that is a simulation law behind
  an unexported predicate, and a copy in the drawing tier is what the crossing and the rate
  composition are already refused for. **The cost is named**: an attacker standing still out of reach
  draws a swing.

- **DD-13 — the mission tool prints the direction, not the byte** (FR-12). One compass name on the
  existing attack line, through `sim.FacingDir` — the ONE conversion this story exports; the other two
  stay unexported, the seam and the tool both needing a direction index and neither a facing.

- **DD-14 — the descriptor mirror's fixture gains the two attack keys** (FR-7). That test builds a
  class whose every derived field is pairwise DISTINCT, so a copy from the wrong source cannot pass by
  coincidence. It set an attack phase count and no attack arrays, so the new track would mirror nil
  onto nil and the new gate false onto false — the property lost for exactly the fields this story
  adds.

## Alternatives rejected

- **A facing gate on the blow.** It is decoded — a precondition — and it still is not built here: the
  gate lives in the order machine's act-state entry and this tree has no act-state machine. Building
  it against no act-state would be inventing the site as well as the rule.
- **Reproducing `MOVE-TURN-031`'s turn cost.** The route destruction and the step gate live in the
  route search and the rate law; a second author there, for a drawing story, is the contention this
  pipeline is arranged to avoid.
- **A bearing over a distance for an attacker's facing.** The engine derives a desired facing from
  the NEXT CELL, always adjacent; a sign rule over a distant victim answers south-east for one five
  cells east — worse than the walk's own facing.
- **Keeping `mw.facing` as a cache.** Rejected under DD-10.
- **A `Facing` type with eight named constants.** It would put a `defined()` refusal on a byte every
  value of which is legal, and the eight names belong to the direction index, a derivation.

## Risks

- **R-1 — the rotation constant is wrong and every unit draws sideways.** The whole visible
  deliverable is the direction. Countered by DD-9's assertion over all eight deltas against the
  shipped derivation, landed before the memory it replaces is removed.
- **R-2 — the never-turned default flips every resting unit from south to north.** A visible change
  the owner sees on the first frame. Disclosed in FR-10 rather than hidden behind an invented south
  default, and bounded: only entities that have never stepped and never attacked.
- **R-3 — the version number is wrong because master moved under the branch.** It did: 13 landed
  mid-story and DD-6's first draft called it concurrent and unused. Countered by rebasing before any
  commit and re-deriving DD-6, FR-6 and AC-6 from the tree that exists — the version narrative in
  `binary.go` is the one document a later reader trusts about the form's history.
- **R-4 — a class whose attack block the sheet does not hold draws a garbage frame.** Countered by
  DD-8's third return, the selection's bounds guard against the sheet's real frame count, and the
  dispatcher's fall-through.
- **R-5 — the swing clock changes a digest.** It cannot: it lives on the seam. Asserted, not argued
  (AC-11).

## Success criteria

- **SC-1** (FR-1, FR-2, FR-6; AC-1, AC-2, AC-6) The field is canonical, the three conversions total,
  the form version 14 at 92 bytes, and 13 refused.
- **SC-2** (FR-3; AC-3) A mover's facing is its last step's, in all eight directions, and nothing else
  on a movement path writes one.
- **SC-3** (FR-4, FR-5; AC-4, AC-5) An attacker in reach faces its victim; a facing changes no blow,
  no step and no route.
- **SC-4** (FR-7; AC-7) The descriptor carries the attack block's slot, track and gate; the two tiers
  stay mirrors.
- **SC-5** (FR-8; AC-8) The attack selection is pure, guarded, total, and reports refusal.
- **SC-6** (FR-9, FR-10, FR-11; AC-9, AC-10, AC-11) The seam dispatches over three selections, draws
  the simulation's own direction, holds one facing answer, and adds nothing the digest can see.
- **SC-7** (FR-12; AC-12) The mission tool states an attacker's facing, measured on both lawful roots.
- **SC-8** (FR-13; P-1, P-2, P-5) Nothing else moves and the whole local gate is clean.
