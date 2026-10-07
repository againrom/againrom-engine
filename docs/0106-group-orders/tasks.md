# Tasks — 0106

Three tasks, in order; each leaves the tree green. All three are in `pkg/sim` and nothing outside it
is touched. Read `plan.md`'s matching section first — it carries the reason for every decision
below, and a task entry states only what to do.

| Task | FRs | DDs | ACs | Ps | SCs |
|---|---|---|---|---|---|
| T1 | FR-1, FR-3 | DD-1, DD-2 | AC-7, AC-9 | P-3 | SC-4 |
| T2 | FR-2 | DD-3 | AC-8 | P-1 | SC-4 |
| T3 | FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10, FR-11 | DD-4, DD-5, DD-6, DD-7 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-10, AC-11, AC-12 | P-2, P-4, P-5 | SC-1, SC-2, SC-3, SC-5 |

The AC column says which criterion a task's own tests are expected to satisfy; witnessing them is
the verification stage's, not a task's.

## T1 — the post on the record, and the constructor that writes it

**Files:** `pkg/sim/world.go` and a `_test.go` beside it.

`Entity` gains `PostX, PostY int32`, placed beside the patrol ring, documented at the field: what a
post is, that every actor has one, that there is no unset value, and DD-1's reason for its being
per-actor rather than per-group.

The constructor writes it from the entity's own `X, Y` **unconditionally**, at the site that already
writes `ActorState`, before any normalisation that could move a coordinate. Whatever a caller's
entity value carried is overwritten (DD-2). No fault shape, no `patrolFault` clause, no clamp: a
post is legal on a dead entity, on an entity in no group and under any order (P-3).

Do not touch the byte form in this task — the record grows in T3, and a form written here would need
re-pinning twice.

**Done when:** AC-7 and AC-9 hold as unit tests over constructed worlds, `go test ./pkg/sim/...`
passes, and no test reads an install (SC-4).

## T2 — the stance commands anchor

**Files:** `pkg/sim/script.go` and a `_test.go` beside it.

`cmdGroupGuard` and `cmdGroupStandGround` each write every **living** member's post to that member's
current cell, over every group record their id names. `cmdGroupGuard` already walks its living
members for the notice base; reuse that walk. `cmdGroupStandGround` needs the walk added and keeps
its existing single-line behaviour beside it.

Write the anchoring in **both** arms rather than hoisting it into `cmdGroupOrder`'s dispatch: the
two arms that anchor are the two that carry the write, so a third arm gaining it would be a visible
edit. No other sub-command — swarm, move, swarm 2, patrol — gains anything.

The write takes the member's cell with no test on whether it is moving, engaged or idle (DD-3). A
member that is not alive is not written.

**Done when:** AC-8 holds, a mid-crossing member anchors at the cell it holds, `go test
./pkg/sim/...` passes, and no test reads an install (SC-4).

## T3 — the walk home, the form, the digest

**Files:** `pkg/sim/engage.go`, `pkg/sim/binary.go`, and the `_test.go` files beside them; the
digest pin in `pkg/sim/hash_test.go`.

`engage.go`: one new method over a group's members, called from `decide` at **both** exits the guard
stance reaches — the empty-candidate branch and the foot of the per-member loop — and from nowhere
else (SC-3). Per member: with a victim, untouched; with none and off its post, `clearOrder` then its
post as target, held; with none and on its post, untouched. The stand-ground arm gains nothing.

Update `engage.go`'s file header: the paragraph saying this build has neither the walk home nor the
turn is now true of the turn only, for DD-4's reason and not a missing field's.

`binary.go`: `formatVersion` 24. Two `int32` on the entity record's tail, after the reach byte; the
layout comment gains a row and the record length grows by 8. The post is carried whole — refuse
nothing, fold nothing, clamp nothing (DD-7).

Re-take the pinned bytes and the pinned digest **together** from the same world, last.

**Done when:** AC-1 to AC-6, AC-10, AC-11 and AC-12 hold; deleting the new method leaves the package
compiling (SC-3); no path added here draws from `w.rng` (SC-1); `git diff --stat` shows no file
outside `pkg/sim` (SC-2); and the full local gate is green (SC-5).
