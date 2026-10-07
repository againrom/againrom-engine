# Plan — 0099

## What already exists, and what has to be built

Opcode 6 already compiles, reports and dispatches on `Args[0]` (`pkg/sim/script.go`,
`groupOrderSupported` and `cmdGroupOrder`). Group orders are stored state on `groupAI`
(`pkg/sim/engage.go`), carried by the byte form and the digest, and `decide` already returns
untouched on an order outside the five (`validGroupOrder`). So **FR-4's behaviour ships**; what
does not ship is order 0 surviving a round trip.

Nothing per-actor exists. The whole of the new layer is: one field group on `Entity`, one pass, one
dispatch, one arm.

## Where each clause lands

| Clause | Site |
|---|---|
| FR-1, SC-2 | `groupOrderSupported` and `cmdGroupOrder`, `pkg/sim/script.go` — one value added to each |
| FR-2 | already there: `cmdGroupOrder`'s `HasGroup` guard and `groupsNamed`'s empty answer |
| FR-3, FR-5, FR-6 | new `cmdGroupPatrol`, `pkg/sim/script.go`, beside its four siblings |
| FR-4 | `validGroupOrder`, `pkg/sim/binary.go` — 0 admitted; nothing in `engage.go` changes |
| FR-7 | new `actorPass`, called from `pkg/sim/step.go` after `w.engagementPass()` |
| FR-8 | new `armPatrol`, in a new file `pkg/sim/actor.go` |
| FR-10 | `clearFelled`, `pkg/sim/step.go` — one call added beside the four already there |
| FR-11 | the entity normalisation loop in `NewWorld`, `pkg/sim/world.go` |
| FR-12, FR-13 | `pkg/sim/binary.go`: `formatVersion`, the record layout comment, encode, decode |

**A new file, `pkg/sim/actor.go`**, holds the state constants, the pass, the dispatch and the arm.
It is a file rather than a corner of `engage.go` because the two layers are the thing this story
separates: `engage.go` is the group layer, and a reader who wants to know what a member does when
its group has stopped deciding should not have to read past 1000 lines of group decision to find
it. It is also what makes SC-1 cheap to check.

## The shape of the actor layer

```
const (
        actorStateGuard  uint8 = 0xb  // the initialiser's own default; no arm here
        actorStatePatrol uint8 = 0xa  // the one arm this build has
)

func (w *World) actorPass() {
        for i := range w.entities {
                if !w.entities[i].Alive() {
                        continue
                }
                switch w.entities[i].ActorState {
                case actorStatePatrol:
                        w.armPatrol(i)
                }
        }
}
```

The `switch` with no `default` is the seam (D-1, SC-1). A state with no case reaches nothing and the
entity is left in every field — which is FR-7's second sentence, and is what makes deleting
`case actorStatePatrol` leave a compiling, behaviour-free pass.

The loop walks `w.entities` in slice order, which **is** ascending id: the slice is kept sorted by
id and the whole step loop already relies on that. No sort is added.

## The arm

```
func (w *World) armPatrol(i int) {
        w.clearOrder(i)                       // FR-8.1 — target, stall and stored route
        e := &w.entities[i]
        if e.X == e.legX() && e.Y == e.legY() {
                e.PatrolLeg ^= 1              // FR-8.2 — a two-node ring
        }
        e.TargetX, e.TargetY = e.legX(), e.legY()
        e.HasTarget = true                    // FR-8.3
}
```

Three points about it.

**The arrival test is position equality, not `arrived`.** `arrived` is the group layer's own test
(`!HasTarget && Transit == 0`) and it cannot be used here, because step 1 has just cleared
`HasTarget` on every patroller — every one of them would read as arrived. The law's own test is on
the cell, and the cell is what an actor standing on its waypoint has in common across a crossing.

**The advance is `^= 1` and not a search.** That is D-2, and D-2's argument is what licenses it. A
comment at the site carries the argument, not just the conclusion.

**No clamp.** D-5 put it at the command, so both ring cells are in bounds by construction and the
destination this writes is too. The clamp `issueGroupDestination` performs is for a distribution
that offsets each member; nothing here offsets anything.

## The command

`cmdGroupPatrol(in ScriptInstant)` sits beside `cmdGroupCommandedMove`. It does **not** reuse
`issueGroupDestination`: that function distributes a formation about a centroid, sets a group rate
term and clamps — three behaviours this arm does not have. The law's command gives each member no
destination at all; the arm gives it one on the same tick. Sharing the body would have meant
passing three flags to turn its own content off.

```
for _, gi := range w.groupsNamed(in.Group) {
        g := &w.groups[gi]
        members := w.groupLivingMembers(g.owner, g.group)
        g.order = orderNone                                   // FR-3
        for _, mi := range members {
                e := &w.entities[mi]
                w.clearOrder(mi)                              // FR-5, with
                e.clearAttack()                               //   the two
                e.clearGroupSpeed()                           //   clears beside it
                x, y := w.bounds.clamp(in.Args[1], in.Args[2])
                e.ActorState = actorStatePatrol               // FR-6
                e.PatrolHeadX, e.PatrolHeadY = e.X, e.Y
                e.PatrolTailX, e.PatrolTailY = x, y
                e.PatrolLeg = patrolLegTail
        }
}
```

`orderNone` is a named constant beside the five in `engage.go`, with the comment that already stands
there — that 0 is what a group carries before anything installs one — amended to say that a command
now installs it deliberately. Naming it is worth one line: `g.order = 0` at a call site is the one
spelling a reader cannot tell from an oversight.

The loop over `groupsNamed` rather than one record is the four siblings' own shape and is kept for
their reason: the script's group parameter names a raw id, not an (owner, group) pair.

## The byte form

The entity record grows from 92 to **110** bytes: `ActorState` (1), `PatrolHeadX/Y`,
`PatrolTailX/Y` (16), `PatrolLeg` (1), appended as a **tail** so no offset before them moves. The
group record's `order` byte is unmoved and only its accepted set widens. The layout comment in
`binary.go` — which is the form's actual specification — gains a paragraph in the voice of the ones
above it, and the version constant gains its own.

**Validation is one predicate, `patrolFault(e Entity) error`,** called from both the constructor and
the decoder. That is the shape `transitFault` already has, and it exists for the reason that one
does: the constructor normalises and the decoder refuses, so the two must agree on what is wrong,
and a second copy of the rule is a second place for them to stop agreeing. It answers FR-13's four
cases; the constructor calls it only to know **what** to normalise, and normalises rather than
returns.

Encode/decode are the mechanical part. The one thing to get right is that **decode does not go
through `NewWorld`** — it never has — so FR-11's unconditional constructor write does not stamp a
decoded patroller back to guard. Verified by AC-3, which is a round trip and not an inspection.

## The tick

One line in `stepWorld`, in the `scriptPassPhase` arm:

```
w.scriptPass(tr)
w.engagementPass()
w.actorPass()      // new
```

That is D-4. The comment block above the switch already argues the script/decision order; the
actor pass is added to it in one sentence rather than gaining a block of its own, because it is the
same argument applied once more: the layer that hands an actor over runs before the actor acts.

## Evidence

The unit witnesses go in a new `pkg/sim/patrol_test.go`. Three of them are not unit tests and are
called out because they are what the story is judged by:

- **AC-2 — the two villagers walk.** `TestPatrolWalksTheMissionTensPatrollers` in
  `pkg/game`, beside the mission-10 drive already there, skipped without an asset root like its
  neighbour. It starts mission 10, steps a bounded number of ticks with **no commands**, and asserts
  that entity 33 and entity 34 each leave their placement cell and return to it. Recording their
  extreme positions in the failure message is what makes a regression readable.
- **AC-1** is `almtool script` output, reported in `verification.md`, not a test.
- **AC-6** is two `missionrun` invocations on two roots, reported in `verification.md`. **It is not
  an assertion and must not become one.**

**Landed tests this story changes, and how.** Named here because the reconciliation is not optional:

- Anything pinning `formatVersion == 20` or a 92-byte entity record — `pkg/sim/binary_test.go`,
  and any pinned digest anywhere in the tree. Every pinned digest changes, because the record
  widens. They are re-pinned from a run and the re-pin is stated in `verification.md`.
- `TestTheCandidateCountNarrowsToAByte` and the 0096 order tests refusing order 0
  (`pkg/sim/grouporder_test.go`, `pkg/sim/binary_test.go`) — the refusal of 0 is **superseded by
  FR-4** and those assertions invert. Everything they say about the other five is unchanged.
- `pkg/sim/scriptgroup_test.go`'s report of unimplemented sub-commands, which names 14 today.
- `TestTheTenthMissionIsDrivenToAWin` **is not touched.** If it changes state, that is a finding
  for `verification.md`, not an edit.

## Risks

- **A pinned digest is re-pinned to hide a real change.** The mitigation is that every re-pin is
  accompanied by the round trip (AC-3), which fails on a form that does not decode to itself.
- **The arm re-searches a route every full tick.** Two patrollers on mission 10, one search each per
  16 sub-ticks. Measured rather than assumed: the mission-10 drive's wall time is reported before
  and after.
- **The clear of the group order makes two villagers stop reacting to anything.** That is FR-9's
  disclosed divergence and it is the story's largest behavioural risk. AC-6 is how it is caught.
