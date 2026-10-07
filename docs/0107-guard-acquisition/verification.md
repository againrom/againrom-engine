# Verification — 0107

Two tasks landed, then one verification-stage commit that carries no trailer. The method is
mutation, not reading: every clause below was checked by breaking the line that implements it and
watching a named test go red. Two mutants survived; both are named, one was equivalent and one was
not, and the one that was not is closed by a test written at this stage.

## The gate

Run on a clean tree at `1f85643`, the last task commit, exit codes captured directly rather than
through a pipe — and **re-run whole at `d36f866`**, the verification commit, because that commit
adds the test that closes M11 and so is not covered by the first run. Both runs are the figures
below.

```
go build ./...                              EXIT=0
go vet ./...                                EXIT=0
gofmt -l $(git ls-files '*.go')             EXIT=0, no output
go test -trimpath -count=1 ./...            EXIT=0   31 packages ok
bash scripts/check-no-game-assets.sh        EXIT=0   "clean (tree scan)"
bash scripts/check-doc-budget.sh            EXIT=0
git diff --diff-filter=D --name-only origin/master..HEAD   (empty)
git log --format='%(trailers:key=Co-Authored-By)' origin/master..HEAD   (all empty)
```

`-trimpath` is not optional locally: Windows Defender quarantines one test binary without it.

## The mutants

Each row is one line reverted in production code, the whole `pkg/sim` suite re-run, and the file
restored. `KILLED` names one of the tests that went red.

```
M1  FR-1        row = 0 -> row = row                        KILLED TestAGroundMemberOfReachAboveOneScoresAFlier (+1)
M2  FR-2        the inside arm, d = 1 -> d = d              KILLED TestTheRewrittenDistanceTermIsFlatThenRises (+2)
M3  FR-2        the outside arm, d = d+1-reach -> d = d     KILLED TestTheRewrittenDistanceTermIsFlatThenRises
M4  FR-6        the refusal reads the PRE-rewrite distance  KILLED TestStandingGroundWithReachAboveOneTakesUpToItsReach
M5  FR-4        the ground conjunct dropped from the fold   KILLED TestAReachAboveOneFlierKeepsItsOwnColumnUnderTheOrdinaryOrder
M6  FR-5        the stand-ground disjunct dropped           KILLED TestTheSameFlierFoldsOntoTheImmobileColumnUnderStandGround
M7  DD-3        the two arms replaced by max(1, d+1-reach)  SURVIVED -> equivalent, see below
M8  FR-7        the veto return deleted                     KILLED TestAGroundMemberNeverTakesAFlier (+6)
M9  FR-4        the fold never fires at all                 KILLED TestAReachAboveOneGroundCandidateFoldsOntoTheImmobileColumn (+1)
M10 FR-3        the row guard widened to m.Reach >= 1       KILLED TestAGroundMemberOfReachAboveOneScoresAFlier (+3)
M11 FR-3        the rewrite guard widened to m.Reach >= 1   SURVIVED -> NOT equivalent, see below
```

**M7 survived and it is genuinely equivalent.** `max(1, d+1-reach)` agrees with the two arms on the
whole input domain, and the domain is closed: `d` is narrowed to a byte immediately above, so
`0 <= d <= 255`, and `reach >= 1` for every entity any path can present (DD-4). Inside the guard,
`d <= reach` gives `d+1-reach <= 1` so the max is 1, and `d > reach` gives `d+1-reach >= 2` so the
max is that. No test can tell the two apart and none should be written to try. DD-3 does not claim
a behavioural difference — it claims that the closed form would hide which of the two arms a
future divergence came from, which is a claim about the next reader and not about this build.

**M11 survived and it was NOT equivalent — it was unwitnessed.** Widening the distance rewrite's
guard from `m.Reach > 1` to `m.Reach >= 1` left every other test in the package green. At a reach
of exactly 1 the two arms are the identity for every separation but **zero**: the inside arm would
rewrite a separation of 0 to 1. A separation of zero looked unreachable and is not — `Domain.layer`
in `world.go` puts air movers on a second occupancy plane from ground and ghost ones, so a flier
and a ground unit may stand on one cell, and a flier is exactly the member the preference table
does not veto against a ground candidate (`preference[3][1]` is 1). The widened guard would score
that pair 384 where the contract scores it 0 — a strict win turned into a tie with any candidate
one cell out, broken the other way by list order.
`TestAReachOneMemberReadsThePlainDistanceAtZeroSeparation` builds that pair, asserts the plain
term, and fails on the mutant. It is the same shape as `0106`'s M9b and was found the same way.

## The contract, clause by clause

**FR-1** — `candidateCost` computes `row := lawDomain(m.Domain)` and overwrites it with the
literal `0` when `m.Reach > 1`. M1, M10.

**FR-2** — the rewrite, two arms in that order under the same guard. M2, M3.
`TestTheRewrittenDistanceTermIsFlatThenRises` tabulates every separation from 0 to 8 at reach 4
and asserts the term is 1 through 4 and `sep+1-4` after, never below 2 outside the reach.

**FR-3** — both halves. The row half by M10, the distance half by M11 and the test written for it.

**FR-4** — `col` folds to 0 only for a ground candidate under the ordinary order. M5 (the fold
made too wide) and M9 (the fold removed) bracket it from both sides;
`TestAReachAboveOneFlierKeepsItsOwnColumnUnderTheOrdinaryOrder` is the negative case.

**FR-5** — the same fold for any domain under stand ground. M6.

**FR-6** — the refusal moved below the rewrite. M4 restores the old reading by evaluating the
refusal against a copy of `d` taken before the rewrite; the AC-4 test goes red on the reach-4 rows
and stays green on the reach-1 rows, which is also what proves that fixture is on the stand-ground
path at all. **The ordering is derived rather than quoted and is graded Medium** — the comment at
the statement says so in those words, and nothing this story is measured on depends on it, because
the guarding order has no such refusal.

**FR-7** — the veto is unmoved and still above everything both tasks add. M8 takes seven tests
with it, six of them landed before this story. Its consequence is AC-2.

**FR-8, FR-9** — `git diff --stat origin/master..HEAD` touches `pkg/sim/engage.go`,
`pkg/sim/world.go` (a comment only) and `pkg/sim/engagereach_test.go`. `binary.go` was never
opened; the byte-form version is still 24 and this story's allocated version is unspent. No field
was added. The determinism source scan in `internal/archtest` is green unchanged (P-2).

## Acceptance

**AC-1 — the census, both roots.** Through this tree's own loader, at `1f85643`, mission 20 on
`gameversions/en` and `gameversions/ru`, identical figures on both:

```
57 entities. 23 carry a reach above 1:
  18 ground movers, slot 3, reach 4, sight 6, in groups 3/4/5/10/11/12/13
   5 fliers,        slot 4, reach 4, sight 6, in groups 2 and 8
slot 3 and slot 4 are MUTUALLY HOSTILE   (Relations.Hostile(3,4) and (4,3) both true)
nearest hostile pair on the map, of any kind: 10 cells
  entity 44 (slot 3, ground, reach 4) at (111,93)  ->  entity 5 (slot 4, air, reach 4) at (107,103)
```

The pair is real and the veto is not hypothetical. **The separation is what matters and it was not
measured before this stage:** every unit on that map carries a sight of 6, and the nearest hostile
pair stands 10 cells apart, so at load *no* candidate list on mission 20 is non-empty and no scorer
runs at all. Mission 10's figures are `analysis.md`'s and were re-read unchanged: four ground units
of reach 4, no fliers.

**AC-2** — `TestAGroundMemberOfReachAboveOneScoresAFlier`: the same pair scores `scoreSeed` at
reach 1 and 192 at reach 4, and 192 is `preference[0][3]`'s arm — row 1's air cell is the table's
zero and could not have produced any finite value.

**AC-3, P-1** — `TestTheRewrittenDistanceTermIsFlatThenRises`, above.

**AC-4** — `TestStandingGroundWithReachAboveOneTakesUpToItsReach`: reach 4 takes a candidate at 4
and refuses one at 5; reach 1 takes one at 1 and refuses one at 2.

**AC-5** — the landed suite is green with **no pin re-taken**. No test file other than the new one
was edited; `git diff origin/master..HEAD --stat` shows no change to `binary_test.go` or to any
digest fixture. Nothing in this story adds state, so a moving digest would have been a stop; none
moved.

**AC-6 — the tenth mission, both roots, before and after.** Four runs of
`missionrun -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3`:

```
BEFORE en   waypoint 1 STOPPED BY THE WORLD DECIDING, short of (39,41), Chebyshev 20, 272 ticks
            outcome lost at tick 272
BEFORE ru   identical
AFTER  en   identical
AFTER  ru   identical
```

**The prediction on record held**: mission 10 is unchanged, line for line, on both roots. Both
reach terms are inert there — the interceptor is a reach-1 ghost mover and its victim a reach-1
ground mover — and the run still ends `outcome lost at tick 272`, the known state this story did
not introduce and does not repair.

**AC-7** — `TestThePinnedBytesDecodeBackToThePinnedWorld` in `pkg/sim/binary_test.go` decodes a
byte stream literal written before this story, header `0x18` = 24, and compares it field by field
against the pinned world. It is green and neither the version nor the record length moved.

## Mission 20 — what a player can actually watch

`analysis.md` predicted the story would bite here. It does, but **not without a player**, and the
distinction is the finding.

With nothing driven, mission 20 is static on both roots and on both sides of the change:
`missionrun -mission 20 -census -ticks 400` reports `0 of 57 unit(s) moved, 0 fell` before and
after. That is the sight measurement above: nobody sees anybody, so no candidate list is ever
non-empty and the scorer this story changed is never called.

Walk one flier into a bow-armed group's sight and the difference is immediate. Driving script unit
`u9` (entity 5, slot 4, air, reach 4) from (107,103) to (111,97), which is four cells from entity
44 of slot 3's group 13, on `gameversions/en`:

```
BEFORE   u9 walks there and nothing answers.
         movers: u7, u8, u9 — all slot 4. 57 entities at the end. u9 at (111,97), 15 hp.
AFTER    movers: u7, u8, u82, u83 — u82 and u83 are slot 3, group 13, GROUND, REACH 4.
         u83 takes 2 damage (60 -> 58 hp). 56 entities at the end and u9 is gone from the world.
```

The two bow-armed ground units of group 13 acquired the flier, shot it down, and took a blow doing
it. Before this story their preference row held a zero in the air column and they would not have
looked at it at any distance. This is the behaviour `spec.md`'s *Why* names, on a map that ships,
identical in shape on both roots.

## Properties

**P-1** — the table test asserts non-decreasing across the join as well as the two bands.

**P-2** — `internal/archtest`'s determinism source scan over `pkg/sim` is green and was not
touched; every term added is integer arithmetic on two entity records. M-none: the scan is its own
witness and reverting it is out of this story's scope.

**P-3** — `scoreSeed` has exactly its two roles in `engage.go`: the three selection loops' seed
(lines 453, 541, 606) and the two returns inside `candidateCost` (the FR-7 veto and the FR-6
refusal). Neither task added a third kind of use.

**P-4** — `candidateCost` is still the only scorer: one `func` definition, called from those same
three selection loops and from tests, and no second body was created (DD-1).

**P-5** — DD-4, and M7's equivalence argument above is its proof: no clamp is written and none is
needed, because `reach >= 1` closes the outside arm at 2.

## Design decisions

**DD-1** — one body and an order, re-affirmed; P-4 is its witness.
**DD-2** — the row choice and the column choice are two statements gated on two different
entities' reaches, with the reason written at the column choice.
**DD-3** — M7.
**DD-4** — the comment at the rewrite, and M7's domain argument.
**DD-5** — `minimalGuardRange` is untouched and still a constant; its existing comment already
carries the registry disclosure, so nothing was duplicated onto it.
**DD-6** — the notice circle is untouched: `noticeBase`, `noticeRadius`, `clipToNotice` and
`freezeGroups` do not appear in the diff at all.

## Two comments this story made false in the other direction, and fixed

Both were recorded in `analysis.md` before either task ran, and each was fixed as part of the
statement it sits on, because a code comment asserting what the rest of the tree does is a premise
rather than documentation.

`world.go`'s `Entity.Reach` said *"no caller of NewWorld yet has a way to name anything but 1"*.
The map loader has filled it from each placement's first resolving weapon since `0104`'s third
task; the AC-1 census above is that field, loaded. `engage.go`'s `preference` table and
`candidateCost`'s own doc comment both said the reach terms *"cannot fire here"* on the ground
that reach was a constant 1. Both cells went live in this story — the row in T1, the column in T2 —
and both sentences now say so.

## What is still open

The refusal's position relative to the rewrite (FR-6) is **derived, Medium**. Nothing measured
here depends on it, but a published row that gives the stand-ground body its own offsets would
settle it either way.

The turn cost is still zero and ties still fall to list order. On mission 20 that is now load
bearing in a place it was not before: a bow-armed group with two fliers equally far away picks the
lower entity id, and no row publishes the body that would decide it differently.

Mission 10 still ends `outcome lost at tick 272`. This story predicted that and did not change it.
