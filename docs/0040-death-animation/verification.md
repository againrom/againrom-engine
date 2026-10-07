# Verification — the death animation: the fall, and the body it leaves

Task commits, oldest first: `8cc3eb8` (T1), `04e280c` (T2), `b99480d` (T3), `d0471a0` (T4),
`1334a64` (T5) — the shas after the rebase onto `origin/master`; before it they were `a405497`,
`f71db42`, `6ee1aa3`, `9a6f734`, `e3b803a` off base `b040859`, which is what the figures below
were measured on and re-measured after. Five trailered commits, each id once, and no other
commit of this story's carries a trailer: `9696c7e` holds the Stage 1–3 artifacts, `ea9b078`
the tasks.md correction recorded under *Contract repaired*, and this file none. Submodule pin
frozen at research `f35be34` throughout — the rebase brought no pin change with it — and
`pkg/sim` is untouched by this story: `git diff --name-only b040859..HEAD -- pkg/sim` was empty
before the rebase, and the eight commits it replayed onto are another lane's.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11, measured 2026-07-31. Every criterion
below but AC-11 is a unit test over hand-written descriptors, hand-built bundles and hand-built
worlds — **no test reads a game install, opens a window or reads a clock** — and no binary
ships.

## Gates

Measured on the rebased tree — the one pushed. The same chain was green on the pre-rebase tree
at 270 Go files, 955 tests and 2699 counting subtests; the differences are another lane's.

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 272 files)
$ go list ./... | wc -l
28                                           28 packages: 25 with tests, 3 without
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 965 tests, 2884 counting
                                              subtests; 25 packages ok, 3 without tests)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh             (0040 rows; whole-tree run exits 0)
  analysis 7012 / 7168   provenance 7123 / 7168   spec 13157 / 13312   plan 13140 / 13312
  tasks.md T1..T5 901 916 884 1310 606, all under 1400; legend+traceability 1124 / 1200
  plan <= 1.2 x spec ok (13140 <= 15788); tasks <= 1.2 x plan ok (5741 <= 15768)
$ sh scripts/check-sdd-audit.sh --story 0040-death-animation      before this file
check-sdd-audit: 5 trailered commit(s) for 0040-death-animation in ac6bd87..HEAD checked
FAIL 0040-death-animation: every task in tasks.md has landed and there is no verification.md
$ git log --format='%(trailers:key=SDD-Task,valueonly)' <base>..HEAD | sed '/^$/d' | sort | uniq -c
      1 0040-death-animation/T1 ... 1 0040-death-animation/T5      (5 ids, each exactly once)
$ git log --format='%B' <base>..HEAD | grep -ci co-authored-by
0
$ git diff --stat <base>..HEAD -- pkg cmd internal | tail -1
 12 files changed, 1172 insertions(+), 43 deletions(-)   (pkg/sim: none of them)
```

**The FAIL set at push time is EMPTY**: the full unscoped sweep exits 0 over the rebased tree.

```
$ sh scripts/check-sdd-audit.sh                   after rebasing onto origin/master
check-sdd-audit: ok (1 note(s)/warning(s), none enforced)     the warning is builds/, untracked
```

It was not empty for most of this story's life, and the set it was is recorded because a gate
that is red for somebody else is exactly the thing a lane can start ignoring. At the base this
branch was cut from, `b040859`, and at every measurement before the rebase, the sweep printed
two lines and neither was this story's:

```
FAIL 0045-two-tier-search: nothing in verification.md witnesses: <three acceptance ids>
FAIL 0045-two-tier-search: nothing in verification.md witnesses: <four success ids>
```

That was another lane's landed contract revision whose evidence had not landed yet; it landed
before this branch was pushed, and rebasing onto it is what cleared them. The id lists are
elided in the fence and given here instead, qualified by the story that owns them —
**0045 AC-12, AC-13, AC-14** and **0045 SC-12, SC-13, SC-14, SC-15**.
The witness gate counts a bare id as a claim by the story whose file it sits in, so pasting
another story's ids unqualified into this one would make them this story's to witness — which
is exactly the failure that gate exists to catch, arriving through a quotation. The
qualification has to sit on the id's own LINE to be read as one, which is how this file first
came out red.

This story's own FAIL — the one in the gate block above, which is what a story with every task
landed and no evidence file looks like — is cleared by this file, which carries no trailer.

## Witnesses

Every id and the named thing that answers for it.

```
FR-1 AC-1  SC-1   pkg/data/anim_test.go TestAnimDyingSlotIsTheDyingBlocksOwnLength
      — five hand-written classes: both layout pairs, a one-frame fall, the absent -1
      sentinel and its resolved-0 twin. Each asserts the derived slot AND the block-sum
      identity DyingBase + D*DyingSlot == TailBase against the tail base the function
      derives independently, so a slot taken off the wrong scalar moves one side only.
      TestAnimBasesStridesAndTotals now reads the slot at both pairs (2 and 4) beside the
      bases it already pinned.
FR-1 SC-1   pkg/game/units_anim_test.go TestUnitAnimMirrorsEveryFieldOfTheDerivation —
      REFLECT-DRIVEN over a class with every phase scalar distinct and both tracks
      distinct, so no field can come out right by coincidence; it binds the next field
      added as well as this one. TestUnitAnimCarriesTheDyingSlot re-reads the identity
      through the render tier's own type at both layouts and at an absent phase.
FR-2 AC-2  SC-2   pkg/game/corpse_test.go TestTheCorpseLinkIsOneHop — a bundle holding a
      sibling namer, a self namer, a dangling namer, a TWO-HOP shape (13 -> 10 -> 11) and
      a frameless class; each judged by pointer identity, read back through the bundle
      type a loader hands over. The two-hop case is asserted twice: that 13 links to 10,
      and separately that it does NOT link to 11, which is where a chain walk would land.
FR-2 SC-2   TestLinkingTouchesEveryClassAndReplacesNone — no class added, dropped or
      replaced, frame counts unchanged, and exactly one class left unlinked (only one
      names a class the bundle lacks). TestLinkingIsOrderIndependentAndIdempotent — nine
      runs leave identical links, and a dying entry naming no class mints none.
FR-3 AC-3  SC-3   pkg/render/terrain/deathanim_test.go TestADyingFrameIsHeldForTwoTicks —
      every tick from 0 to 2L-1 against this file's own transcription of the block rule,
      plus the boundary said twice: the block's last frame is first reached at tick
      2(L-1) and is NOT reached one tick earlier.
FR-3 AC-4 P-4  SC-4   TestPastTheRunTheLastDyingFrameIsHeld — all eight octants at ticks
      2L, 2L+1, 100, 5000 and 2^30: the answer is the block's last frame every time, and
      every answer is asserted to lie inside [base + oct*L, base + oct*L + L).
FR-3 AC-5  SC-5   TestTheDeathSelectionTakesTheAnimatedBlocksDirectionRule — both layout
      pairs over all eight octants, plus the two rules stated as counts (0 of 8 mirror at
      D 8, 3 of 8 at D 5) so they cannot both be one rule.
FR-3 AC-5  SC-5   TestTheDeathSelectionRefusesWhatItCannotDraw — six refusals (no dying
      block, an absent phase clamped to none, a short sheet, a sheet of no frames, a
      negative count, a base below zero), each carrying frame 0 and no mirror; and the
      guard's bound said from both sides — the sheet of exactly last+1 frames answers,
      the sheet of last does not.
FR-3   TestANegativeCountIsTheFirstFrame and TestTheDeathSelectionIsTotalAndRepeatable —
      1440 input combinations over D, slot, octant and tick, each asked twice: no panic,
      equal answers, and no answered index outside the frame count.
FR-4 AC-6 FR-5  SC-6   pkg/game/death_test.go TestAKilledUnitPlaysTheCorpseClassesFallAnd
      ThenHolds — a hand-built world, a class whose corpse is a DIFFERENT class with its
      own sheet and its own block constants. Alive it draws its own art; from the tick
      the blow lands it draws the CORPSE class's art at the corpse sheet's own indices,
      one frame per two ticks, then holds — re-read 40 ticks later.
FR-5 AC-7  SC-6   TestDownedAndDeadBothFall — one entity damaged to exactly zero and one
      killed outright, on one tick: both report their own life state at the seam and both
      draw the same fall at the same indices.
FR-6 AC-8 P-3  SC-6   TestNoUnitStopsBeingDrawnOnAccountOfDying — the three refusals as
      WORLDS rather than as calls (a corpse class holding no frames, one with no dying
      block, one the bundle does not hold) plus a unit with no art. The push holds four
      entries for four entities, each of the three is drawn from its OWN class's sheet
      with no death frame, and the artless one keeps the square.
FR-7 AC-9  SC-7   TestABodyKeepsTheDirectionItDiedFacing — walked west four cells (read
      off the world's own cells), then killed; for twelve ticks after, its step is zero
      and its frame is the WEST slot of the dying block at the expected phase.
FR-4 P-2   TestASnapshotBuiltTwiceAnswersTheSameDeath — over ten ticks spanning the death,
      four builds per tick compared entry by entry.
FR-4   TestTheDeathClockIsStampedOnceAndCarriesNoPerEntityOffset — two entities of
      different id killed on one tick start their falls at the SAME frame (the live
      selection's per-entity offset is not applied), the clock holds exactly two entries,
      the stamps do not move over twenty further ticks and twenty further builds, and a
      world where nothing dies holds none.
FR-8 AC-10 P-1  SC-8   pkg/game/drawn_invariance_test.go TestOneCommandStreamReachesOne
      DigestWhateverIsDrawnBetweenItsTicks — the stream now carries a survivable blow, a
      blow to exactly zero and a kill, so one entity passes through all three life states
      in one run. Two driven legs (one call a tick; five calls a tick at four different
      remainders) are compared against a HEADLESS leg assembled from the table itself:
      canonical byte form and digest equal at every one of the 13 tick indices. Death's
      own non-vacuity is asserted beside it — all three life states crossed the seam, and
      each driven leg's death clock holds exactly one entry, so the agreement is not one
      about a memory that stayed empty.
P-3   AC-6, AC-7 and AC-8 read together: every entity in each of those pushes is drawn by
      exactly one of the three paths and none by none — the four-entry count in AC-8's
      world is the completeness half.
```

## Mutants — SC-9

Both applied to `pkg/render/terrain/unitanim.go`'s phase expression, one at a time, each run
over the whole tree and reverted (`git diff` clean afterwards, tests green).

```
1. deathPhaseTicks 2 -> 1                      the fall plays twice as fast
   --- FAIL: TestADyingFrameIsHeldForTwoTicks     (pkg/render/terrain)
   ok  pkg/data  ok  pkg/game  — no other package moved
2. the upper clamp deleted                     the held frame walks past the block
   --- FAIL: TestPastTheRunTheLastDyingFrameIsHeld  (pkg/render/terrain)
```

Beside them, a green-but-hollow check on T4's dispatch: the death branch disabled
(`if false && ...`) fails four of the six death witnesses — the fall, the downed/dead pair, the
frozen facing and the clock — and leaves the never-vanish one green, which is the right shape,
that one being about the fallback.

## Not run

**AC-11 (manual)** — the game against a lawful install, a unit killed and watched. Not run
here: no window, no install and no binary in this lane's evidence. It reads FR-3, FR-5 and FR-7
together in the running picture, all three of which have automated witnesses above; what it
would add is the only thing tests cannot say, that the result looks like a body falling.

**A corpse drawn from a real sheet.** Every death frame above is selected from a hand-built
sheet, so the block constants are this repo's own arithmetic and not a shipped class's. The
corpus fact the pin carries — that every shipped class names a dying class and every target
resolves with a positive dying phase count — is cited, not re-measured.

## Contract repaired

`60207c2` amends T4's file list and one fence, and nothing else. T4 adds a memory the snapshot
build writes, and one older test assembles a partial driver by struct literal whose own comment
says it reproduces the constructor's state before the tick-0 push; a build that writes a map
that literal does not hold panics on the first entity. The entry's permission list gains that
test file and its fences say the memory list is the only thing that may change there. No
requirement, design decision or task boundary moved, so it is editorial and started no revision.

## Conclusion

The contract is met for what it claims and no more. The fall and the frozen last dying frame
are drawn from the corpse class's own sheet at the decoded cadence, a body keeps its facing,
nothing that dies stops being drawn, and no value this story adds reaches the world: byte form
and digest agree with a headless run at every tick index of a stream that kills. What is not
built is the decay past the first stage, and it is not built because it is not a drawing — the
evidence ledger records what it would take and which lane owns it.
