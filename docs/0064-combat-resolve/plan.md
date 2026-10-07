# Plan — an attack that resolves

## Approach

`pkg/sim` gains a fourth command kind, a third loop in `Step`, twelve fields on `Entity` and one
draw helper on the world's generator. The order is a **state-setting** command like a move-to; the
damage is not carried by any command but by the new loop, which runs exactly once per advance. The
blow's arithmetic lives in one unexported method with one call site. Everything the cycle consists
of goes into the canonical byte form at the tail of the unit record, which moves the version.

Where each requirement is settled: FR-1 and FR-2 in DD-1, DD-1a and DD-9; FR-3 in DD-3; FR-4 in
DD-5 through DD-8a; FR-5 and FR-6 in DD-10 and DD-11; FR-7 and FR-8 in DD-12 through DD-14.

## Facts verified during planning

- `Step` runs commands in slice order, then the move loop over alive entities in ascending id, then
  one tick increment — its own doc calls those three **phases**, numbering the increment third —
  and the move loop's occupancy scratch is per-tick and discarded with it.
- `clearFelled` is the single site that drops a felled unit's residue, with two callers, and drops
  the order, the transit pair and the group term. `clearOrder` is the **world's** method: a stored
  route is a world field, so an entity method cannot end a walk.
- `counted` is `!Dead() && (Domain != DomainAir || !HasTarget)`, so a flyer holding no destination
  is in the air occupancy plane and one holding a destination is not.
- The world owns a SplitMix64 generator whose eight bytes of state are at header offset 9.
  **Nothing consumes it**: this is its first reader. `Hash` is FNV-1a over exactly `encode`'s bytes.
- The form is at version 8, `headerLen` 34, `entityLen` 44; every other version is refused and there
  is no migration path.
- The constructor normalises residue and the decoder refuses it, sharing `transitFault` where both
  refuse. The decoder assigns over the receiver only after every check has passed.
- `pkg/data`'s `UnitDef` carries the to-hit, defence, absorption, damage pair, always-hits mark,
  reach and the two cadence columns. `pkg/mapload/spawn.go`'s difficulty arm already reads and
  writes two of them; nothing else does.
- `pkg/mapload`'s `FromALM` fills `Class`, `HP`, `MaxHP`, `Domain` and `Speed` on a placement, so
  every world a loader builds carries a zero charge, a zero relax and a zero damage pair.
- Seven `pkg/sim` test files hold hand-written record widths, offsets, version bytes or pinned
  digests, and `pkg/mapload` holds three. **`pkg/game` holds none** — every digest comparison there
  is world against world inside one run.
- `replay_test.go`'s fixture is documented as putting every field the byte form carries in motion.

## Design decisions

- **DD-1 — the order sets state; the loop does the damage.** The command writes a victim and
  nothing else. Damage is applied only by the new loop, which runs once per `Step` by construction.
  *Rejected:* a damage-carrying command, the shape the debug arm has — a doubled command slice
  would then deal double damage, the hole `0019` disclosed.
- **DD-1a — an order naming the victim already held leaves the cycle alone.** *Rejected:* an
  unconditional restart, under which a caller re-issuing its order every tick — the obvious
  front-end shape — resets the charge every tick and never lands a blow.
- **DD-2 — the cycle runs in a third loop, after the move loop, before the tick increment, in
  ascending id.** So a blow reads **post-move** positions and a victim that stepped away this tick
  is out of reach. *Rejected:* inside the move loop, where a kill would free the victim's cell
  mid-walk and give the loop's five-outcome contract a sixth cause; and before the move loop,
  which would resolve a blow against positions the same tick is about to invalidate.
- **DD-2a — an attacking flyer is counted.** Clearing the destination puts it in the air plane for
  that tick's move loop, which is the existing rule — a flyer at rest is one its peers must route
  around — reached by a second route rather than a new one. *Rejected:* exempting an attacker,
  which would let two flyers rest on one cell.
- **DD-3 — three phase values of our own, `Ready` the zero value, and a charge below one counts as
  one.** A ready attacker loads and starts charging in the same turn, so `Ready` costs no tick and
  the published period is reproduced exactly — which it is not at a charge of zero, where the first
  blow would land a tick early and every later period be one long. *Rejected:* the engine's own
  0/5/7, which would imply a correspondence the rest of the record does not have; a `Ready` that
  costs a tick; and chaining free transitions, which at a zero cadence does not terminate.
- **DD-4 — the countdown is `int32`, refused below zero and above what its phase could have
  loaded.** That upper bound is `transitFault`'s own argument: the count is loaded from the charge
  or from the relax plus the jitter and only falls. *Rejected:* a `uint8` matching the engine's
  signed byte, which needs a clamp on charge and relax to stay representable.
- **DD-5 — the seven numbers are `int32`, range checked nowhere**, as `Speed` and the health pair
  are, and **every sum and comparison a blow makes is taken in `int64`**, as this package already
  takes its cell counts, centroids and record spans. Health saturates at the least `int32` rather
  than wrapping. *Rejected:* the engine's own narrower widths, which would make a definition
  `pkg/data` produces unrepresentable; and `int32` arithmetic, which turns a to-hit near the top of
  the range into a certain miss.
- **DD-6 — reach is a package constant of one cell, not a field.** *Rejected:* a per-unit field —
  `pkg/data` carries a reach column, but only equipment moves it off 1 and equipment is out of
  scope, so the field would be four bytes of hashed state with one reachable value.
- **DD-7 — one draw helper, `uniform(n) → [0, n]`.** It advances the generator even where the
  answer is fixed and **returns 0 for any `n` at or below zero**, so a negative damage spread
  cannot produce a huge roll. It is a widening multiply of the drawn word by `n+1`, high half.
  *Rejected:* a modulo, whose bias is larger; and rejection sampling, whose draw count depends on
  the values drawn.
- **DD-8 — the resolution is one unexported method with one call site**, and it makes both its
  draws or neither: the two refusals that cost nothing to test — out of reach, and a victim with no
  health system — stand together at its head. *Rejected:* splitting the roll from the application;
  and testing the health system after the draws, which buys nothing.
- **DD-8a — the relax draw is taken by the cycle on every strike tick, after the resolution
  returns.** A strike tick costs three draws where the blow was attempted and one where it was
  refused. *Rejected:* drawing it only when the blow landed, which would give a refused cycle a
  different period from a resolved one.
- **DD-9 — a fresh attack order ends the walk through `clearOrder` and a fresh walk order ends the
  attack**, at all three sites an order is written. *Rejected:* clearing the entity's target
  without the world's route, which leaves a route on a unit with no target — a world `Step` writes
  and the decoder refuses.
- **DD-10 — an attack order leaves the transit pair and the group rate term alone.** A mover
  ordered to attack mid-crossing pays its crossing in the move loop and runs its cycle in the third,
  because a crossing is a fact about where a body is and the cycle is not. *Rejected:* clearing the
  pair, which is what being felled does and what nothing else does; and zeroing the group term,
  which the two move arms do because the order they carry is a movement order.
- **DD-11 — an order pointing at a corpse is legal in the byte form.** A victim killed by a higher
  id leaves a lower id's order on it until that attacker's next turn, so it is a state a tick
  leaves. *Rejected:* refusing it, which would refuse a world `Step` produces.
- **DD-12 — the version rises by one, and the twelve fields go to the record's tail** in one block, in
  the order the cycle and then the resolution read them. *Rejected:* interleaving them with related
  fields, which moves every offset after the insertion and buys nothing.
- **DD-13 — the constructor normalises a self-target, an unheld victim and cycle residue; the
  decoder refuses all three.** *Rejected:* refusing in the constructor too — a caller handed a stale
  victim can do nothing with an error.
- **DD-14 — the debug kill and damage commands are kept unchanged**, and the accumulation the
  damage arm carries is witnessed by a test rather than left as prose. *Rejected:* a set-health
  form — it closes the accumulation but changes a shipped debug seam and its front-end caller, on a
  story whose deliverable is elsewhere. The witness makes that later change fail loudly.

## Files to touch

| Path | Intent |
|---|---|
| `pkg/sim/combat.go` | ADD — the phase type, the cycle, the resolution, the reach test |
| `pkg/sim/world.go` | MODIFY — the twelve `Entity` fields, `clearAttack`, the constructor's rules |
| `pkg/sim/step.go` | MODIFY — the attack kind and its arm, the move arm's clearing, the third loop, `clearFelled`, and the doc clauses the loop falsifies: that nothing here draws, that an entity holding a target has five outcomes, and `restAt`'s note on when the scratch exists |
| `pkg/sim/group.go` | MODIFY — the group arm clears the attack order |
| `pkg/sim/rng.go`, `pkg/sim/doc.go` | MODIFY — `uniform`; the determinism wording |
| `pkg/sim/binary.go` | MODIFY — the version, `entityLen`, the encoder, the decoder and its refusals |
| `pkg/sim/combat_test.go` | ADD |
| `pkg/sim/replay_test.go` | MODIFY — its fixture must put the new fields in motion or its own claim is false |
| `pkg/sim/{binary,budget,nostate,domain,relaxation,routeform,gridform}_test.go` | MODIFY — re-pin |
| `pkg/mapload/{fromalm,gridform}_test.go` | MODIFY — three pinned digests |

## Risks

- **R-1 — a version bump invalidates every pinned digest and hand-written width in the tree.**
  *Mitigation:* each is re-derived from the world it names, keeping the superseded value in the note
  beside it, as those files already do.
- **R-1a — `budget_test.go` holds a test whose NAME asserts the form version did not move in its
  own revision.** Bumping its literal leaves it asserting the opposite of what happened.
  *Mitigation:* re-anchor it to the version this story inherits and name the story that moved it.
- **R-2 — an attacker can outlive its victim.** *Mitigation:* FR-6, checked before the cycle runs.
- **R-3 — a draw count that depended on something outside the world would make a replay diverge for
  a reason no recorded state explains.** *Mitigation:* DD-7 and DD-8a fix the count as a function of
  the world's own state; a criterion measures the generator rather than the outcome. Within one
  advance a kill by a lower id does change how many draws a higher id makes, and that is inside the
  contract because it is a function of state the byte form carries.
- **R-4 — every world a loader builds carries a zero cadence and a zero damage pair**, so an
  ordered attack there resolves nothing and repeats every few ticks. *Mitigation:* DD-3 makes that
  terminate and stay inside FR-3's period; the state is exercised rather than assumed.
- **R-5 — a sibling lane may need the next form version.** *Mitigation:* one constant, one reader,
  and the number is whatever is free at the merge rather than one chosen in this lane.

## Success criteria

| # | Condition | How |
|---|---|---|
| SC-1 | A blow lands on the charge-th advance, and consecutive blows are `charge + relax + [0,3]` apart | automated, an advance-by-advance trace |
| SC-2 | Two worlds from one seed alike but for one entity's speed agree tick for tick in every other field | automated |
| SC-3 | An ordered attack kills its victim; it passes through downed, and both orders end | automated |
| SC-4 | Every refused shape of FR-8 is refused and the receiving world is untouched | automated, one case per refusal |
| SC-5 | A world round-trips mid-cycle and both copies advance identically for many ticks, digest for digest | automated |
| SC-6 | Duplicating every attack order in a frame changes no field of the result | automated, and the same test states what the debug damage command does |
| SC-7 | The record width, the version byte, the offsets and the field-set pin name the new form, and each of the twelve fields moves the digest | automated |
| SC-8 | A strike advance costs exactly three draws in reach and exactly one out of it, counted off the generator's own state | automated |
| SC-9 | The whole gate is green with no game install present | `go build ./... && go vet ./... && gofmt -l . && go test -count=1 -trimpath ./...`, plus the three repo scripts |
| SC-10 | An attacker whose victim a lower id killed this tick ends its order that tick and draws nothing | automated |
| SC-11 | Re-issuing the same order every tick still lands blows on the period; re-issuing a different victim every tick never lands one | automated |
