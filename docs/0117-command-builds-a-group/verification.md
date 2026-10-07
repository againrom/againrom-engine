# Verification — 0117

Branch `0117-command-builds-a-group`, base master `84aa355`, research pin `87a256d`. Two trailered
commits, one per task, both verified in this seat on a clean tree.

## The defect, measured before and after

The reproduction is `TestATenthMissionHostileNeverHoldsItsOwnPostAfterBeingCommanded`
(`cmd/missionrun/commandgroup_test.go`), guarded on `AGAINROM_ASSETS` and skipped without it. It
starts mission 10, takes script unit 21 — a hostile at (36,51), owner slot 2, placed group 2, post
(36,51) — orders it five cells off its placement with one ordinary move command, and fails the
moment the unit is handed its own post as a destination.

Measured against the EN root:

```
with the two commandGroup call sites REVERTED:
  --- FAIL  tick 23: script unit 21 holds its own placement (36,51) as a destination
            after being commanded away to (38,49)
with them in place:
  --- PASS
```

The first line is the owner's report in one sentence. It is witnessed by reversion, not by reading
the assertion: the two calls were commented out, the package rebuilt, the test re-run, and the calls
restored. **AC-1.**

## The milestone

`bash pipeline/check-milestone.sh` — `ok`, both roots, `outcome lost at tick 272`, `census: 4 of 36
unit(s) moved, 1 fell, over 272 tick(s)`. That gate drives `builds/current`, which is master's
binary, so it says nothing about this branch on its own. The drive was therefore re-run from **this
worktree's** own `missionrun`, same argv (`-mission 10 -census -waypoint u21:56:21:3 -waypoint
p0:66:16:3`), against both installs: the four recorded lines came back **byte-identical**. The
milestone does not move in either direction under this story.

That is not a null result to be waved past. The drive commands two units, one of them the very
hostile the reproduction uses, so both now leave their placed groups mid-drive — and the outcome,
the tick and the mover count are all unchanged.

## Acceptance criteria

| | Witness |
|---|---|
| **AC-1** | `TestATenthMissionHostileNeverHoldsItsOwnPostAfterBeingCommanded`, above |
| **AC-2** | `TestACommandedGuardIsNotWalkedHomeButAnUncommandedOneIs` — the pair, in one world |
| **AC-3** | `TestACommandedActorsPlacedGroupIsUnchangedAndItsAliveCountHolds` |
| **AC-4** | `TestAGroupMoveBuildsOneCommandGroupWithARecordPerOwner` |
| **AC-5** | `TestACommandedTwiceActorKeepsTheSameIDAndNothingAccumulates` |
| **AC-6** | `TestNoCommandGroupIDEqualsAPlacedOrScriptNamedID` |
| **AC-7** | `TestASlotlessActorGainsNoCommandGroupOrRecord` |
| **AC-8** | `TestGroupRecordsStayAscendingAndRoundTripAfterOrders` |
| **AC-9** | `TestThePreviousVersionFormIsRefused` names 26 by its own number, beside `TestUnmarshalRefusesEveryVersionButTheCurrentOne` |
| **AC-10** | by deletion, run in this seat — below |

All but AC-1 are synthetic and run with no install present.

**AC-10, run here rather than taken on report.** Both `commandGroup` call sites were replaced by a
comment. `go build ./...` succeeded — the seam is a call, not a condition threaded through the
decision — and `go test ./pkg/sim/` failed in exactly seven places: the five new tests that assert a
nonzero command group, the re-pinned digest in `TestACommandedWorldRoundTripsAndHashesToTheSameDigest`,
and `TestAMoveCommandedGroupKeepsDecidingWhileItWalks`. Nothing else in the package moved. The two
tests that still passed under the deletion — AC-7's and AC-8's — hold vacuously without a writer,
which is what an absence claim and an ordering claim do.

## Properties

**P-1** — nothing added reads a clock, a random draw or a map iteration order. `internal/archtest`'s
source scan over `pkg/sim` is green, and `commandFloor`, `freeCommandGroup` and `commandGroup` are
slice walks over the world's own entities, script and records.

**P-2** — the only order a player order installs is `orderMove`, which `decide` already dispatches
through `moveArm`. No new order value is written anywhere.

**P-3** — `TestGroupRecordsStayAscendingAndRoundTripAfterOrders`, and structurally: `upsertGroup`
overwrites at an existing key and otherwise inserts at the ascending position, so a duplicate key
cannot be created.

## Scope claims

**SC-1** — `walkHome` is byte-for-byte unchanged: `git diff 84aa355..HEAD -- pkg/sim/engage.go`
touches no line of it, and its own tests are untouched. What changed is who is a guard.

**SC-2** — no non-test source outside `pkg/sim` reads `CommandGroup`. `pkg/mapload`'s byte-form
tests do, because they mirror the record's width; that is the form's business reaching them, not the
field's.

**SC-3** — a placed group's record is left standing when its last actor is commanded away. Nothing
reads a record no actor names, and `decodeGroups` already declines to police the correspondence.

## Gate, on the clean committed tree

```
go build ./...                          clean
go vet ./...                            clean
gofmt -l $(git ls-files '*.go')         empty
go test -trimpath -count=1 ./...        every package ok, no install present
scripts/check-no-game-assets.sh         clean (tree scan)
scripts/check-doc-budget.sh             EXIT=0
scripts/check-sdd-audit.sh              EXIT=0
git log --format='%(trailers:key=Co-Authored-By)' 84aa355..HEAD    empty
git diff --diff-filter=D --name-only 84aa355..HEAD                 empty
```

The note/warning **count** from `check-sdd-audit.sh` is meaningless from a lane worktree, which has
no `builds/`; only the FAIL set is comparable, and it is empty.

## The merge with master, and how the record was resolved

This story branched at `84aa355` and was merged forward twice before landing — first with
`4648096`, then with `e43dae0` — past four stories that also touched the byte form.

**Three stories widen three different places and none of them collides with another.**
`0112-container` added two SECTIONS, the carry block and the purse, between the sack section and the
script section. `0122-trigger-opcodes` appended four fields to the compiled CHECK record, taking its
width from 63 to 73. `0117` appends `CommandGroup` to the ENTITY record at `+145`, taking its width
from 145 to 149. They meet only in `binary.go`'s version constant and in the pins downstream of each
widening. Nothing in any of the three layouts had to be renumbered or moved.

**The version is 32, and it is this story's third number.** The field was written against 27, merged
forward as 27 when 26 landed, and lands at 32 because 31 landed first. That is worth saying plainly
rather than tidying away: a version number is allocation order and carries no claim about the form,
so what a merge has to establish is that the LAYOUTS do not collide — not that the number was right
the first time. FR-9 named 27 and the built form says 32; the requirement is the widening, and the
number is the seam it lands on.

Every backward-compatibility peel chain gains `strippedOfTheCommandGroup` at its **head**: it takes
a version-32 form down to 31, and 31's own peel — which on master is `strippedOfTheCarryAndPurse`,
because `0122`'s widening leaves a script-less pinned world's bytes untouched and moves only the
version byte — runs on its output. That ordering is the one thing in the resolution that would still
compile if it were backwards, so every chained expression in `pkg/sim/binary_test.go`,
`routeform_test.go`, `pkg/mapload/fromalm_test.go` and `gridform_test.go` was read in full, both
times, rather than trusted to the suite.

Every digest downstream of the widening was re-taken by running the test that prints it. **No
behavioural assertion changed** in either merge. `0123`'s repair of `remove`'s `carried` compaction
survives untouched — this story never reached that function, and `remove` still compacts all three
slices. `0123`'s own form-version canary in `pkg/sim/corpseloot_test.go`, re-pinned to 31 at
`0122`'s landing, ends at 32.

**Both witnesses were re-run on each merged tree**, not carried over. The reversion check gives the
identical result every time — `tick 23: script unit 21 holds its own placement (36,51) as a
destination` with the two calls out, PASS with them in — and the AC-10 deletion check fails in the
same seven places.

**The milestone drive, from the merged head's own binary**, both roots: `outcome lost at tick 272`,
`census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)` — the four lines
`pipeline/milestone-baseline.txt` records. The gate itself is taken in the orchestrator's seat
against a freshly built `builds/current`, because a stale one there measures a binary older than the
landings it is supposed to cover.

## What was falsified on the way in

Two premises this lane was handed were wrong and are recorded in `analysis.md`: the group order is
**stored** state frozen once at construction, not a per-tick derivation from the owner. The premise
that `walkHome` has no `underCommand` test was correct, and so was the reading that adding one would
fix nothing the owner can see — on arrival there is no command left for such a test to find.

No test of 0106's was asserting the defect. `TestAGuardsOwnOffPostDestinationIsReplacedByItsPost`
comes closest — it hands a destination-holding guard straight to `decide` and asserts the post
replaces it — but its own doc block already says that such a member is excluded by `aiGroups`, so it
was never a claim about a commanded unit. It passes unchanged.

## What this story did not build

The other order families `AI-CMD-033` names. An attack order leaves the actor in the group it was
in; so does every mission-script act. And nothing returns a commanded actor to its placed group —
that is FR-8, stated rather than filled in, and it is the disclosure this story owes: a unit the
player has ordered stays under the move order for the rest of the mission, engaging what comes
within reach and otherwise standing where it was sent.
