# Tasks — 0099

Two tasks, split representation from behaviour. T1 adds the field and the form and writes nothing
into either; T2 adds the writer, the pass and the arm. That order is what lets T2 witness every
behaviour through the real command instead of through a hand-built byte form.

`verification.md` and the build are pipeline stages, not tasks, and carry no trailer.

**Both tasks:** done when the repo's standing gate is clean and every AC in the task's row has a
named witness. AC-1 and AC-6 are measurements for `verification.md`, never assertions.

| Task | FRs | ACs | Ds |
|---|---|---|---|
| T1 | FR-4, FR-10, FR-11, FR-12, FR-13 | AC-3, AC-4, AC-5 | D-3 |
| T2 | FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8, FR-9 | AC-1, AC-2, AC-7, SC-1, SC-2 | D-1, D-2, D-4, D-5 |

## T1 — the actor state and the byte form

**Files:** `pkg/sim/world.go`, `pkg/sim/binary.go`, `pkg/sim/step.go`, `pkg/sim/binary_test.go`,
new `pkg/sim/actorform_test.go`, and the digest pins named below.

Add to `Entity`, as a tail: `ActorState uint8`, `PatrolHeadX/Y`, `PatrolTailX/Y int32`,
`PatrolLeg uint8`, and the constants `actorStateGuard = 0xb`, `actorStatePatrol = 0xa`. `NewWorld`
writes guard into every entity unconditionally; `clearFelled` clears the ring, the leg and the state
back to guard beside the four clears already there.

One predicate `patrolFault(Entity) error` answers FR-13 — the constructor calls it to know what to
normalise, the decoder calls it and refuses. `validGroupOrder` admits **0** beside the five.
`formatVersion` becomes **21**; the record grows 92 -> 110; the layout and version comments each
gain a paragraph in the voice of the ones above them.

**Landed tests that go red and are meant to:** every literal digest in
`pkg/mapload/{fromalm,gridform}_test.go`; the digest and offset pins in
`pkg/sim/{binary,commanded,groupform,hash,relaxation,routeform}_test.go`; every `92` record stride;
the 0096 order-0 refusal, which inverts. **Re-pin from a run, record old -> new in the commit body.**
A test red for any other reason is a stop-and-report, not an edit.

## T2 — the command, the pass and the arm

**Files:** new `pkg/sim/actor.go`, `pkg/sim/{script,engage,step}.go`, new
`pkg/sim/patrol_test.go`, `pkg/sim/scriptgroup_test.go`, one new test in `pkg/game`.

`actor.go` holds `actorPass` (living entities, slice order, `switch` on `ActorState` with **no
default**) and `armPatrol` (the plan's three steps; arrival is position equality, the advance is
`PatrolLeg ^= 1`, with D-2's argument at the site). Call `w.actorPass()` after `w.engagementPass()`
in `stepWorld`'s script-phase arm, and add one sentence to the comment that already argues that
ordering.

`groupOrderSupported` and `cmdGroupOrder` gain sub-command 14 together. `cmdGroupPatrol` sits beside
`cmdGroupCommandedMove` and does **not** reuse `issueGroupDestination`: it sets `g.order = orderNone`
(new constant beside the five) and, per living member, stops it, sets the state, builds the clamped
ring, makes the tail the leg.

**Landed test that goes red and is meant to:** `scriptgroup_test.go`'s report of unimplemented
sub-commands, which names 14 today. **`TestTheTenthMissionIsDrivenToAWin` and the loss predicate are
not touched** — if that test changes state, stop and report.

The `pkg/game` test starts mission 10 with an asset root (skipped without one), steps with **no
commands**, and asserts each patroller leaves its placement cell and returns.

