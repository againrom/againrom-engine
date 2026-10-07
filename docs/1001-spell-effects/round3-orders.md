# round3-orders.md — fix round 3, R3-C1 (the stale route)

Branch `story/1001-r3-orders`, based on `efd89b8` (the pushed sha the second adversarial review ran
against). This document covers only R3-C1. The seat folds the proposed `spec.md` sentence and
`docs/DIVERGENCES.md` row into the canonical documents at the landing; this lane did not edit
`spec.md`, `closure.md` or `docs/DIVERGENCES.md`.

All commands below are run from the worktree root.

## What was wrong

`round3-review.md`'s R3-C1: a `KindMoveTo` or `KindGroupMoveTo` order that rewrites an entity's
target left the entity's stored route standing at the OLD target. `pkg/sim/step.go:441` wrote
`TargetX`/`TargetY` without touching `w.routes[i]`; `pkg/sim/group.go:320`
(`issueGroupDestination`) did the same. `decodeRoutes` (`pkg/sim/binary.go:2537`) refuses a
non-empty route whose last cell is not the entity's target, and the game's save is the marshalled
byte form verbatim (`pkg/game/save.go` → `pkg/game/resume.go`), so a save taken in that window was
written and could not be loaded.

An ordinary mover never shows the defect: the walk loop (`step.go`) runs the same tick the order
lands, and `subGoal` (`step.go:986`) already treats a route whose last cell is not the current
target as unusable and searches again, overwriting `w.routes[i]` before the tick ends. The defect
is reachable only when the walk loop does not run for that actor on the ordering tick, which it
does not for two reasons already in the code before this fix:

- **A book cast owns the actor through its wind-up** (`step.go:676`, `if _, casting :=
  w.bookCastIndex(e.ID); casting { continue }`) — new to this story, and R3-C1's own trigger.
- **A mover owing transit ticks for its current crossing** (`step.go:694`, `if e.Transit > 0 {
  e.Transit--; continue }`) — present at `origin/master` already. The review confirmed this
  independently in a `b51b439` worktree: 41 consecutive unloadable ticks at speed 6.

## The choice taken, and the one rejected

The brief's two options were fixing only the casting path (leaving the mid-transit path standing
and disclosed) or fixing the invariant at both target writers unconditionally.

**Option 2 was taken.** The fix at each writer is not a branch on "is this actor casting" or "is
this actor mid-transit" — it drops `w.routes[i]` whenever the write changes the entity's target,
which is correct regardless of which reason (or no reason) keeps the walk loop from reaching that
actor this tick. Testing for the casting case specifically and leaving the transit case as a
conditional the fix does not cover would have been *more* code for a narrower guarantee, not less:
the two skip sites share one invariant (a stored route's last cell equals the entity's target or the
route is empty) and one point of failure (the target writer that does not maintain it). Handling
both from the one site that writes the target is the smaller change.

Option 1 (casting path only) was rejected because the mid-transit path is equally reachable by a
player — any move order re-issued while a mover is between two cells, not a rare or synthetic
condition — and it is save-breaking by the same mechanism and the same decoder message. Leaving it
standing "disclosed" would still ship a game that can write saves it cannot load, on a path this
story did not create but does touch (both target writers) and can close at no additional cost.

The corollary the brief asks to be proven: this changes behaviour outside R3-C1's own path (the
mid-transit case, and in principle the ordinary case that `subGoal` already used to paper over), so
it needed evidence against the shipped campaign before being accepted. That evidence is below.

## What changed

One conditional at each of the two target writers, guarding a route drop:

- `pkg/sim/step.go`, `case KindMoveTo`: before writing `TargetX`/`TargetY`/`HasTarget`, if the
  order's cell differs from the entity's currently held target (or the entity held no target),
  `w.routes[i]` is set to `nil`.
- `pkg/sim/group.go`, `issueGroupDestination`: the same test and the same drop, against the
  clamped per-member cell each group member is actually sent to (post-formation-offset), before
  writing that member's `TargetX`/`TargetY`.

Both mirror the idiom `invalidateBlockedRoutes` (`pkg/sim/routeblock.go`) already uses for the wall
case: `w.routes[i] = nil` costs the mover one fresh search on whichever tick it next moves and
nothing else, because an empty route is always legal per `decodeRoutes` (`count == 0` skips the
target-match check entirely) and `subGoal` already treats an empty route exactly like a discarded
one.

Both `KindGroupMoveTo` and `KindGroupSwarmTo` reach `issueGroupDestination`, and so does the mission
script's own group march (`cmdGroupCommandedMove` → `issueGroupDestination`, `script.go:1985`), so
all three inherit the fix from the one write site; none of the three needed its own change.

**The drop is conditional on the target actually changing, not unconditional on every write.** A
re-issued order naming the same cell an entity already holds keeps its route — relevant to a client
that emits a move command on every tick a movement key is held (`step.go`'s own comment on the
autocast sweep names this pattern), which would otherwise pay a fresh search every tick for no
behavioural reason.

## What was left standing

Nothing, for R3-C1 and for the mid-transit case beside it. Both are closed by this fix.

Not touched, because they are outside this finding and outside this lane's file ownership: R3-A1,
R3-A2, R3-B1, R3-B2, R3-B3, and every "minor finding" in `round3-review.md`.

## The fix does not disturb the shipped campaign

`bash scripts/campaign-sweep.sh` issues no player orders (unattended drive), so it exercises the
group writer only through script-driven group marches and the AI decision arms — not the player
`KindMoveTo`/`KindGroupMoveTo` path R3-C1 itself is about. It is still the available instrument for
"does this change disturb shipped, unattended play", and the comparison is exact:

```
AGAINROM_ASSETS=<againrom>/gameversions/en bash scripts/campaign-sweep.sh   # with the fix
AGAINROM_ASSETS=<againrom>/gameversions/ru bash scripts/campaign-sweep.sh   # with the fix
```

28 missions, 2000 ticks each, both roots: every row identical to the fixture the review's own
`b51b439` baseline used (**59 unsupported / 7375 reached**). The output table was captured with the
fix in place and again with both writer changes reverted (`w.routes[i] = nil` removed at both
sites, tree otherwise identical) — outcome columns AND the hash column, all 28 rows, are
byte-identical between the two runs on the EN root:

```
diff <(AGAINROM_ASSETS=<en> bash scripts/campaign-sweep.sh) <(same, fix reverted)
# empty diff
```

Two additional headless drives that DO issue player move orders, to reach past the sweep's own
unattended limit — compared the same way, fix present against fix reverted, output byte-identical
in both cases:

```
AGAINROM_ASSETS=<en> go run ./cmd/missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3
AGAINROM_ASSETS=<en> go run ./cmd/missionrun -mission 130 -census -mage -waypoint p0:30:30:5
```

The first is the documented milestone drive (`pipeline/milestone-baseline.txt`'s own invocation);
the second is a mage-hero drive over a different, larger map. Neither of these two tools marshals
mid-drive, so they do not themselves witness the save/load defect — that is the round-trip tests'
job, below — they witness that the broadened fix changes nothing about where units end up or how
many fall.

`cmd/spelleffectcheck` (no `-mission` given, default 91) ran clean against the EN root with the fix
in place, printing the full 28-row spell table and the mission 91 controlled wiring line unchanged
from its known form.

## Witnesses

Every witness is a round trip: build the state through the production command path (`Step`), then
`MarshalBinary` and `UnmarshalBinary`, and require no error — the exact operation the game performs
at save and load. A field comparison would not show that the byte form itself is refused.

All four live in the new file `pkg/sim/staleroute1001_test.go`.

| Test | Writer | Precondition | Revert that reddens it | Failure message |
|---|---|---|---|---|
| `TestAMoveOrderOnACastingActorRoundTrips` | `step.go` `KindMoveTo` | a stored route stands, then a book cast begins, then a second `KindMoveTo` retargets the caster | remove the added conditional and `w.routes[i] = nil` at `step.go`'s `KindMoveTo` arm | `UnmarshalBinary refused the byte form the game would have saved: sim: entity record 0: its route ends at (0,8) and its target is (5,0)` |
| `TestAGroupMoveOrderOnACastingMemberRoundTrips` | `group.go` `issueGroupDestination` | same precondition, retargeted with a `KindGroupMoveTo` of one member | remove the added conditional and `w.routes[i] = nil` at `issueGroupDestination` | same message: `... its route ends at (0,8) and its target is (5,0)` |
| `TestAMoveOrderMidTransitRoundTrips` | `step.go` `KindMoveTo` | a slow mover (`Speed 1`) is mid-crossing (`Transit > 0`) with a stored route, then a second `KindMoveTo` retargets it | remove the `step.go` fix | same message |
| `TestAGroupMoveOrderMidTransitRoundTrips` | `group.go` `issueGroupDestination` | same mid-transit precondition, retargeted with `KindGroupMoveTo` | remove the `group.go` fix | same message |

Each revert was applied in isolation (the other file's fix left in place) and the affected pair of
tests reddened while the other pair stayed green, confirming each fix is independently load-bearing
at its own writer:

```
go test -trimpath -run 'TestAMoveOrderOnACastingActorRoundTrips|TestAGroupMoveOrderOnACastingMemberRoundTrips|TestAMoveOrderMidTransitRoundTrips|TestAGroupMoveOrderMidTransitRoundTrips' -v ./pkg/sim/
```

With `step.go` reverted alone: `TestAMoveOrderOnACastingActorRoundTrips` and
`TestAMoveOrderMidTransitRoundTrips` fail with the message above; the two group-writer tests pass.
With `group.go` reverted alone: the two group-writer tests fail with the same message; the two
`step.go`-writer tests pass. With both fixes in place, all four pass. With both reverted, all four
fail.

## Proposed `spec.md` sentence

Ready to paste, appended to the existing paragraph in "Actor action and cast admission" that ends
"The move spends nothing and refunds nothing." (currently `spec.md:123-127`):

> A move or group-move order that changes an entity's target also drops that entity's stored route,
> when the new target differs from the one already held. A non-empty stored route's last cell
> equals the entity's target at every tick boundary, which the byte form requires; an ordinary
> mover's own walk rebuilds an equivalent route on the same tick, but an actor whose walk does not
> advance that tick — a pending cast's wind-up, or an owed transit crossing — would otherwise leave
> the tick holding a route that still ends at the previous target beside the newly written one, a
> state the byte form refuses. A re-issued order naming the same target keeps its route.

## Proposed `docs/DIVERGENCES.md` row

Column order is the ledger's own:
`ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status`.

Allocated: `DIV-050`. `DIV-051` is not used and is returned; a duplicate of `DIV-028`'s own subject
(what a new order does to an actor mid-cast, already `UNKNOWN` there) did not need a second row, and
no other new fact surfaced that the ledger's five types cover — the fix is measured to change no
ROM1-observable outcome (the campaign sweep's hash column is identical with and without it), so it
is not itself a behavioural divergence; it is a defect in this tree's own byte-form consistency, and
the one open research question beside it is below.

| DIV-050 | sim / stored route consistency during a cast or transit re-order | Spells must have their effects, and a unit must obey a click (same mandate as `DIV-028`) | No claim decodes whether ROM1's own pathfinding model keeps a persisted "stored route" structure analogous to this tree's, or how its order machine would keep such a structure consistent with a rewritten destination while the actor's body is not advancing (a cast's wind-up, or a mid-crossing transit). `AI-ORDER-039` is silent on this | A move or group-move order that rewrites an entity's target unconditionally drops any stored route that no longer ends at that target, regardless of why the actor's own walk is not advancing this tick. This keeps `decodeRoutes`' own invariant satisfiable at every tick boundary, so the byte form the game saves always loads. Measured to change nothing else: the campaign sweep's hash column is identical across all 28 EN and RU maps at 2000 ticks with and without the fix, and two headless drives issuing real move orders (mission 10's documented milestone drive; a mage drive over mission 130) are identical too | UNKNOWN | Round 3 adversarial review, finding R3-C1: a move order landing on a casting or mid-transit actor left a stale route standing whose last cell no longer matched the rewritten target, which `decodeRoutes` refuses; the game's save is that byte form verbatim, so a save taken in that window could not be loaded. The casting path is new to this story; the mid-transit path was already present at `origin/master` (`b51b439`), measured there at 41 consecutive unloadable ticks at speed 6, and is closed by the same one-line fix at each writer rather than left standing | A claim describing ROM1's own route/order-machine consistency under a re-order while the actor's body is not advancing | OPEN |

## Gate

Run on the clean tree at `efd89b8` plus this lane's two files and one new test file:

```
go build ./...
go vet ./...
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

All green; `gofmt` printed nothing; `check-no-game-assets` printed `check-no-game-assets: clean
(tree scan)`.

## Not done in this lane

- R3-A1, R3-A2, R3-B1, R3-B2, R3-B3 and every minor finding in `round3-review.md`: out of scope,
  and three other lanes hold them.
- `pkg/sim/effect.go`, `pkg/sim/rearm.go`, `pkg/sim/celleffect.go`, `pkg/sim/route.go`,
  `pkg/sim/spell.go`, `pkg/sim/combat.go`, `pkg/game/iteminfo.go`: not touched, per file ownership.
- `spec.md`, `closure.md`, `docs/DIVERGENCES.md`: not edited; the sentence and row above are
  proposals for the seat.
- A registered `SDD-Task` trailer: none of this lane's commits carries one, since this fix is one
  change to one finding rather than a fanned-out story.
