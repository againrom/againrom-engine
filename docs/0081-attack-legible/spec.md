# Spec — a fight is legible

**Intensity: spec-anchored / static. Terrain: brownfield** for the entity record, its byte form, the
attack cycle and the drawing seam's facing — all four ship today — and **greenfield** for the attack
timeline and its selection.

A fight in this build reads as two statues losing health. The simulation resolves a blow and nothing
about it is visible: no unit turns, because there is no facing to turn, and no swing is drawn,
because the animation descriptor knows a move, an idle and a dying timeline and no attack one. This
story gives the simulation a facing, makes an attacker turn toward what it is hitting, and draws the
swing — those three, and what it is **not** is in Out-of-Scope with a reason per item.

## Functional requirements

- **FR-1 — a facing is SIMULATION STATE.** An entity carries one facing byte. It is canonical: the
  byte form writes it, it enters the digest, and two worlds differing in one facing are two worlds.
  **No value is refused**, by the constructor or by the decoder — every byte names a direction (FR-2),
  so there is nothing to fold or reject, and refusing one would make a state this package can build a
  state it cannot read back.

  **A unit that is not alive keeps its facing**: it is not residue of a state the unit has left, so
  the felled-unit clauses that drop a target, a crossing, a group term and an attack order take no
  clause for this one.

- **FR-2 — the byte's eight directions, and the three total maps.** The facing byte is in **32-unit
  steps over 256**, so eight directions, clockwise from north: 0 north, 1 north-east, 2 east,
  3 south-east, 4 south, 5 south-west, 6 west, 7 north-west, against a screen whose `+y` is south.
  - **byte to direction**, total over all 256: `dir = ((facing + 16) >> 5) & 7` — a byte this build
    never writes still names one of the eight.
  - **direction to byte** is `dir << 5`.
  - **a delta to a direction** is the pair of signs `(sign dx, sign dy)`. **A zero delta names no
    direction**, and every site that could produce one leaves the facing it found.

  **Every delta here points FROM the turning unit TO the cell it turns toward** — destination minus
  origin; the opposite subtraction is the same eight rotated by half a turn, which is why the order is
  fixed here. The zero byte is therefore **north** and not an absence.

- **FR-3 — a mover faces the cell it steps to**: the cell taken less the cell left, by FR-2, written
  where the step is taken and nowhere else — so a mover blocked, owing crossing (transit) ticks,
  giving up or arriving keeps the facing its last step left.

- **FR-4 — an attacker turns to face its victim ON THE ARM THAT STOPS ITS WALK.** The pursuit is
  asked at every turn of an alive attacker holding a victim the world still holds and has not killed;
  the arm that finds that victim **in reach** ends the walk, and it is that arm — and no other site —
  that faces the attacker at it: the victim's cell less its own, by FR-2.
  - **Out of reach** it is walking, and FR-3's facing stands: the pursuit points it one step at a
    time, and a bearing over a distance is not a thing this story invents.
  - **A victim on the attacker's own cell** is a zero delta, so the facing holds.
  - The reach is the one the blow and the pursuit's stop already read; there is no second comparison.

  **The strike turns nothing**, and that is decoded rather than economical: the swing start and the
  strike hold no facing test, no facing write and no call to any turn routine, and the turn-to-face
  producer set is six routines of which none is on that path. Turning an attacker as part of its
  swing would invent a coupling the original does not have.

- **FR-5 — a facing GATES NOTHING HERE, and both gates it should are named.** No blow, no step, no
  route search and no give-up reads the field.
  - **A blow is not refused for facing away.** In the original facing IS a precondition: the attack
    act-state is entered only while the direction from attacker to victim equals its current facing,
    re-tested every tick, so a victim that circles its attacker drops it back to the arm that turns
    and stands. That gate lives in the order machine's act-state entry, and this tree has no
    act-state machine for it to live in — disclosed here rather than built.
  - **A step is not withheld until a mover has turned**, and a turn costs no tick and destroys no
    route, where the engine's costs `ceil(arc / RotationSpeed)` ticks above one direction step. This
    build reproduces the free arm for every arc; what diverges is the duration.

- **FR-6 — the byte form takes the field and its version becomes 14.** The facing is the entity
  record's own new tail, so no offset before it moves; the record widens by one byte, from the 91 it
  stands at to 92. **13 is the version this tree already writes** — it is another story's, landed,
  and it widened the planes rather than the record — so this one is 14. A version-13 form is refused
  as every earlier version is, and it is the SHARP case: its header, its planes and every offset
  inside its first record are identical to version 14's, so nothing but the version byte separates a
  correct decode from one that reads every record after the first a byte early.

- **FR-7 — the attack timeline reaches the animation descriptor.** It gains one direction slot length,
  one expanded track and one gate for the attack block, derived exactly as the move and idle blocks'
  are from the class's own already-loaded attack keys: the slot the phase count clamped at zero, the
  track the same run-length expansion, the gate the conjunction of the two. `AttackBase` is unchanged.
  The two tiers' descriptors stay mirrors **field for field**.

- **FR-8 — a swing is DRAWN, by a selection of its own.** A third pure selection stands beside the
  living and the not-living ones: `frame = AttackBase + slot*AttackSlot + AttackTrack[step]`, `slot`
  and the mirror by the direction rule every animated block already shares, `step` the swing clock
  reduced modulo the track's own period. It reports **whether there is a frame at all**, like the
  death selection and unlike the live one: false for a class with no attack block, no track or a
  failed gate, and false for an index outside the sheet's own frame count. No input panics, no
  division by zero occurs, and equal inputs give equal answers.

  **The run is sized from the ART and plays ONCE**: the timeline is indexed one entry per tick from
  zero with **no modulus**, and the run lasts exactly the expanded track's own length. A clock at or
  past it is a run that has ENDED, reported as a refusal so the caller falls through to the standing
  drawing. Looping is the move arm's behaviour and no other's.

  **The swing is NOT synchronised with the blow, and that is fidelity.** The swing frame, the swing
  sound and the damage are scheduled from three different numbers, and binding any two reproduces the
  original on at most 2 of the 157 shipped pairs that answer.

- **FR-9 — the drawing seam chooses between the three, in one place.** For each entity, in order: not
  alive takes the death selection; otherwise one **holding a victim that did not move this tick**
  takes the attack selection; otherwise the live one. Each refusal falls through, so a class with no
  attack block draws what it draws today. The order is the contract: a body never swings.

- **FR-10 — the drawn direction IS the simulation's facing.** The seam translates FR-2's direction
  index onto the **sheet's octant ordering** — S=0, SW=1, W=2, NW=3, N=4, NE=5, E=6, SE=7, south-first
  where FR-2's is north-first — by whatever carries each of the eight onto the octant naming the same
  compass direction, and reads nothing else: the step-derived octant memory is **gone**, and with it
  the only other answer in the tree. Both orderings name the same eight, so a unit drawn walking or
  dead faces where it faced before. **One thing changes**: a unit that has never turned faced south
  and now faces north (FR-2), neither being decoded.

- **FR-11 — the swing clock is the SEAM'S OWN, and it counts one RUN.** It returns to zero at the
  tick an attacker's cycle enters its charge — the swing start, which is the instant the engine sends
  the message that enters the drawn attack state — and rises by one per map-screen tick after it. An
  entity holding no victim holds a zero. So each pass of the attack cycle draws exactly one pass of
  the art, and FR-8's refusal past the track's length is what stands the unit up again between them.
  Nothing canonical reads the clock, and a run that draws swings and a run that does not are the same
  run, tick for tick and digest for digest.

- **FR-12 — the mission tool reports an attacker's facing**, so a turn is measurable without a
  screen.

- **FR-13 — nothing else changes.** The route search, the movement rate law, the crossing, the stall
  count, the give-up, the script, the notice, the readout and the press path answer as they do today,
  and `pkg/ui` still names no simulation type.

## Out of scope, each with its reason

- **Damage numerals.** Decoded end to end while this story was in flight — construction, colour,
  lifetime, drift, merge rule and a toggle — and allocated to a story of their own against that
  decode. Nothing here draws one.
- **The enemy fighting back.** `AI-RETAL-056` refutes retaliation-as-an-order: being struck sets one
  flag whose single consumer turns the victim one direction step from where it already faced, and
  writes no target and no order. The reply comes through ordinary acquisition, which this tree has
  none of. A "hit back when hit" rule would be a divergence that looks like fidelity.
- **The turn's cost, its rate column and its route destruction, and the facing PRECONDITION on a
  blow** (FR-5).
- **A facing in the placed unit record.** The decoded record carries none.

## Acceptance criteria

- **AC-1** An entity naming no facing faces north; every one of the 256 values survives a round trip;
  two worlds differing only in one facing have different digests.
- **AC-2** `dir = ((facing + 16) >> 5) & 7` is asserted over all 256 bytes and lands in `[0,8)`; the
  eight directions round-trip `dir << 5`; each of the eight deltas maps to its direction and the zero
  delta to none.
- **AC-3** A mover faces the direction of each step as it takes it, in all eight directions; one
  blocked, one owing crossing ticks, one that gives up and one that arrives keep their last step's.
- **AC-4** An attacker ordered onto an adjacent victim faces it within one tick, from each of the
  eight relative positions; one ordered onto a distant victim faces the way it walks until it arrives
  and its victim afterwards; a victim on the attacker's own cell leaves the facing alone; and an order
  whose victim died this tick turns nothing.
- **AC-5** One schedule run from two worlds differing only in a facing gives the same health, cells
  and routes — a facing changes no blow, no step and no route.
- **AC-6** A felled unit keeps the facing it died with across the blow, the constructor and a round
  trip; version 13 is refused; the version byte is 14 and the record 92 bytes.
- **AC-7** The descriptor's attack slot, track and gate are what the class's keys imply, including the
  clamped-at-zero absent phase and the empty-track gate failure; the two tiers hold one field set.
- **AC-8** The attack selection returns the block's own frames for every tick of the run and refuses
  at and past its length; refuses for no block, no track, a failed gate and an out-of-range index; and
  is total over a negative or huge clock and a zero frame count.
- **AC-9** For each of the eight deltas the octant the translated facing yields is the octant FR-10's
  ordering gives that delta's compass direction — over all eight, not a sample.
- **AC-10** A swinging entity draws an attack frame; a corpse draws a death frame; a class with no
  attack block falls through to the live selection.
- **AC-11** An attacker beside its victim draws the attack block for exactly the run's length and its
  idle drawing afterwards; the clock is zero for an entity holding no victim; and a schedule run with
  and without a viewer gives one digest.
- **AC-12** Against both lawful roots the mission tool orders an attack, fells the victim, and reports
  the attacker facing it.

## Properties

- **P-1** `pkg/sim` reads no clock, names no float and imports nothing outside the standard library.
- **P-2** The three drawn selections are jointly total and ordered: every entity reaches exactly one
  drawing, and none panics or divides by zero at any input.
- **P-3** Exactly one AUTHORITY answers which way a unit faces. The direction index, the sheet octant
  and the mirror bit are derivations of it, computed per call and stored nowhere; what is forbidden is
  a second REMEMBERED facing, which is what the deleted octant memory was.
- **P-4** Nothing the drawing seam holds reaches the world: the swing clock, like the death clock and
  the odometer, is invisible to the byte form and the digest.
- **P-5** The full local gate is clean: build, vet, gofmt, the test suite, the three repo scripts.
