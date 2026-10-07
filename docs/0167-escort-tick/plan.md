# 0167 — escort tick: plan

## Shape

One new file, `pkg/sim/escort.go`, holding both arms and the three helpers they
share. `pkg/sim/actor.go` gains two cases in its switch and nothing else. No
other package changes, no field is added, and the byte form is untouched.

| Requirement | Where it lands |
|---|---|
| FR-1 | `actor.go`, `actorPass`'s switch |
| FR-2 | `escort.go`, `escortSubject` |
| FR-3 | `escort.go`, `escortStop` and `escortSubject` |
| FR-4 | `escort.go`, `escortClose` |
| FR-5, FR-6, FR-7 | `escort.go`, `coverEngage` |
| FR-8 | `escort.go`, `armDefend` |
| FR-9 | `escort.go`, `armFollow` and `acquireStanding` |
| FR-10 | `escort.go`, `escortStepAway` |
| FR-11 | `escort.go`, by what the arms do not write; witnessed by AC-9 and AC-12 |
| FR-12 | `escort.go`, integer arithmetic only; enforced by `internal/archtest`'s source scan |

## Design decisions

**DD-1 — the two arms are two functions and the shared work is three helpers.**
`AI-FOLLOW-112` states that the two arms are not one behaviour parameterised: the
out-of-range half is store for store the same, and the within-range halves are a
cover scan on one side and three instructions on the other. Two functions calling
one `escortClose` and one `escortStepAway` is that shape. A single function with a
state parameter would have to carry the difference as a branch on the state byte
in the middle of its body, which is the reading the claim rules out.

**DD-2 — the escorted unit is resolved once per arm, into an index.** Every later
step needs the unit's cell, its owner and its domain. `indexOfEntity` is the one
lookup this package has, and resolving once keeps a defender from scoring a block
built around one entity and stepping away from another.

**DD-3 — an escort whose target the world no longer holds does nothing.**
`AI-FOLLOWDEATH-119` establishes that the law dereferences `ord+0x10` with no
null test and no dead test, and that nothing clears it. Both are facts about a
pointer that stays valid while the object lives. This build removes an entity from
its slice at the end of the decay ladder, so the id can name nothing, and there is
no behaviour to transcribe for that case. The arm returns before it decides
anything. The consequence is stated in the claim's own terms: while the escorted
unit is merely DEAD, and still in the slice, both arms keep running against it —
the escort closes on the corpse and covers it — which is what the claim predicts
of the original and is left standing here rather than corrected.

**DD-4 — the step-away is computed in 8.8 fixed point, as integers, with no
sub-cell term.** `AI-FOLLOWGAP-114` gives the arithmetic in 8.8: the escorted
unit's cell packed with no sub-cell, the escort's own cell packed with its
sub-cell, a zero delta forced to 1, the dominant axis moved the stop distance,
the minor axis rounded onto the line, both clamped to `[8, dim-9]`. This build
has no sub-cell field. Both positions are therefore packed the same way, at
`cell*256`, and the delta is cell-granular.

The grid is kept rather than the whole computation being done in cells, because
the minor axis takes a proportional part of the major move and that part is a
sub-cell quantity. The consequence of dropping the sub-cell term is stated rather
than hidden: it is what makes the helper's own zero-forcing reachable. The claim
records that branch's purpose as keeping the answer off the escorted unit's own
cell, which is a purpose only a reachable zero has, so the two read together.
Under the alternative transcription — the escort's position carrying the
cell-centre 0x80 the law's own centre test reads — no delta could ever be 0, that
branch would be dead, and an escort standing due west of its subject would step
away north-west. Neither reading is decidable from the published row, and this
one is chosen because it keeps the branch the row explains.

**DD-5 — the minor axis rounds half away from zero.** `R0279` is graded
Medium as a plain float-to-int and is read as a use, not traced. `pkg/sim` admits
no floats. Half away from zero is the choice, computed as
`(2*num + den) / (2*den)` on the magnitude with the sign applied after, so that
the two directions round symmetrically. The term it decides is at most the stop
distance and only ever moves the destination one cell, and the destination is
clamped afterwards.

**DD-6 — the cover block takes the corpse rule from the acquisition.** The only
published corpse rule in this neighbourhood is `AI-ACQUIRE-002`'s: candidates
below 1 health are parked, and the parked list is taken whole where nothing living
survives. It belongs to the acquisition routine and not to the block build, which
is unread on that point. This build applies the same rule to the block, so that
"which candidates exist" has one answer in this package rather than two. A
defender therefore attacks a body only where nothing living stands within 5 cells
of the unit it protects.

**DD-7 — the standing acquisition reuses `candidateCost` under stand-ground.**
0096 DD-2 already made this substitution and stated its terms: the law's
acquisition discards its pick unless it is within reach, and the reach-vetoing
scorer is the half this build can honour. The alternative is a second scorer that
would have to agree with `candidateCost` on every world. The pick's ordering is
therefore this package's own cost ordering, not `AI-ACQUIRE-002`'s
nearest-then-turn ordering; the two agree on which candidates are admissible and
can differ on which of two admissible candidates is taken.

**DD-8 — the candidate list for one actor is `candidates`' body with one member.**
`groupSight` already takes a member list, and a one-member list is one actor's own
sight. The diplomacy decider is that actor. This is written as `actorCandidates`
beside `candidates` rather than by constructing a synthetic `aiGroup`, because an
`aiGroup` carries an owner and a group id that an actor-layer decision has no use
for and that `groupState` would then be asked about.

**DD-9 — a pursuit is the busy test.** `AI-FOLLOWTAB-113` reduces the inner table
to: `ord+0x08` in {5, 6, 8, 9} means the escort is on a pursuit or a cast and does
nothing further; anything else falls to the crowding check. This package
represents a pursuit as an attack order and has no actor-layer cast, so the test
is `HasAttackTarget` read after the engagement attempt. It is read after, not
computed inside `coverEngage`, because the law reads the order byte whatever wrote
it — including a pursuit the escort was already on.

**DD-10 — the close order and the step-away both drop the victim.** Both write
`ord+0x08`, which is the same byte a pursuit lives in, so a pursuit cannot survive
either. `clearAttack` is called directly; `releaseAttack` is not, and stays pinned
to `decide` alone as `release_test.go` requires.

**DD-11 — the close order ends at the actor pass and not mid-walk.** The law
services order 4 every sub-tick and stops at the stop distance. This build decides
on the actor pass, which runs once per script cycle. A unit crosses a cell in more
ticks than a script cycle takes at every speed this build produces, so an escort
overshoots its stop distance by at most one cell before the next pass ends the
walk. What ends it is that every within-range path writes over the order: an
engagement writes the pursuit, and an acquisition that scores nothing clears the
order outright, which is this build's stand-in for the idle turn the law writes
there.

## Superseded and cut

**SC-1 — actor state 0xc gets no arm, and its body is built anyway.** The
standing acquisition is `R0022`, which is both the 0xc arm's own body and
what the two escort arms fall to. It is built here, as `acquireStanding`, and
wired to the escort arms alone. The state stays without a case for a reason this
story cannot discharge: `AI-TICK-008` gates the per-actor machine on the group's
order byte being 0, and this build's actor pass has no such gate — it dispatches
on the state alone. Actor state 0xc is what the NAMED unit of every script attack
and every script escort takes, and it is also a unit a player or a drive can order
to walk. Giving it an arm here would let the actor pass overwrite that order every
script phase, which is a behaviour change no claim in this story's set is about
and which nothing in this slice witnesses. A later story adds the case and the
gate together.

**SC-2 — the defender's heal fork is not built.** `AI-FOLLOWHEAL-118`: a defender
carrying a spell book casts spell 6 on the unit it protects before it fights for
it. This build has no actor-layer cast path at all — the cast state `0xd` has no
arm — so there is nothing for the fork to select. The decoded fall-through is
exactly what is built: nothing cast falls straight through to the cover
engagement. The cost is that a defending caster fights instead of healing, and it
is bounded by there being no defending caster in any world this build can produce
from a shipped map without an authored spell book.

**SC-3 — the idle turn is not built.** `AI-ORDER-039`'s arm `0xb` re-picks a
facing under two gated conditions. This build's escort with nothing to acquire
ends its walk and keeps its facing. The facing half is a separate behaviour with
its own claim and its own randomness, and no requirement here reads a facing.

**SC-4 — no per-sub-tick re-aim.** DD-11 states the terms.
