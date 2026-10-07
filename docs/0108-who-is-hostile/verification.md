# Verification — 0108, a blow decides who is hostile, not only the map

Stage 4 of the story: T1, T2, the fallout the plan did not predict, and the evidence.
Everything below was run in the lane worktree at the branch's last commit, on a clean tree.

## What landed

| Commit | Trailer | What |
|---|---|---|
| `50eb913` | `SDD-Task: 0108-who-is-hostile/T1` | `turnHostile` on the relation type, and its tests |
| `28d3ec7` | `SDD-Task: 0108-who-is-hostile/T2` | `flipOnBlow`, the call in `resolveBlow`, and its tests |
| `e67c471` | none — a pipeline stage | six landed tests that read one-way hostility off who acquired |
| `ec86947` | none — a pipeline stage | AC-4's untouched half, asserted rather than inferred |
| this one | none — a pipeline stage | `verification.md` |

No commit carries a `Co-Authored-By` trailer. Checked by hash, not by report:

```
$ git log --format='%h [%(trailers:key=Co-Authored-By)]' f3ecf15..HEAD
ec86947 []
e67c471 []
28d3ec7 []
50eb913 []
```

Deletion set over the whole branch is empty:

```
$ git diff --diff-filter=D --name-only f3ecf15..HEAD
$
```

## SC-1 — the gate

Run on a clean tree at the branch's last commit, exit codes captured directly.

```
go build ./...                             0
go vet ./...                               0
gofmt -l $(git ls-files '*.go')            0, no output
go test -trimpath -count=1 ./...           0, 31 packages ok
bash scripts/check-no-game-assets.sh       0   "clean (tree scan)"
bash scripts/check-doc-budget.sh           0
bash scripts/check-sdd-audit.sh            0   FAIL set empty
```

`-trimpath` is not optional on this machine: Defender quarantines one test binary without it.
The audit's note/warning **count** is meaningless from a lane — a worktree has no `builds/` — so
only its FAIL set is quoted.

## AC-1, SC-2 — the census, both roots

Taken through this tree's own reader (`game.StartMission` → `mapload.FromALM` → `Relations.Byte`)
by a throwaway command that is **in no commit**: it was written into `cmd/xcensus/`, run, and
deleted before the first commit of this stage. Ordered off-diagonal cells only, over slots 1..N
where N is the map's own roster length.

|  | maps | ordered | flippable | already hostile | locked |
|---|---|---|---|---|---|
| `gameversions/en` | 28 | 678 | 257 | 395 | 26 |
| `gameversions/ru` | 28 | 678 | 257 | 395 | 26 |

The two roots agree, and agree per map as well: the whole 28-line report is byte-identical
between them (`diff` returns nothing). AC-1 is met.

**A finding, and it is about `spec.md` rather than about the code.** `spec.md`'s *Why* paragraph
says "51 of 198 ordered off-diagonal pairs are flippable and 3 are locked". Those are exactly the
numbers of the **first nine** campaign maps in mission order — 10, 20, 30, 31, 40, 41, 50, 51, 60,
which sum to 198 ordered, 51 flippable, 3 locked. The corpus does not stop there; it runs to
mission 151, 28 maps. The stage-1–3 census was a prefix of the corpus, not the corpus. The clause
is left standing as written, because it is not false about what it measured and the direction of
its claim is unchanged — the real figure is **larger**, 257 of 678 rather than 51 of 198. Anything
downstream quoting 51 should quote 257.

SC-2's own requirement — a throwaway reader in no commit, three counts, the roots agreeing — is
met by the table above.

## AC-8, SC-3 — the tenth mission, both roots, before and after

`TestTheTenthMissionIsDrivenToAWin` skips without `AGAINROM_ASSETS`, so it is green in every gate
and red whenever actually run. It was run four times, with the asset root passed in and never
compiled in.

| | EN | RU |
|---|---|---|
| before (`f3ecf15`) | `outcome lost at tick 272` | `outcome lost at tick 272` |
| after (`ec86947`) | `outcome lost at tick 272` | `outcome lost at tick 272` |

All four traces are identical line for line, including the waypoint line:
`waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (39,41), Chebyshev 20,
after 272 ticks`. The stage-1–3 prediction — that mission 10's flippable pairs are not crossed by
a blow on the drive — holds. Nothing here was tuned toward the milestone and it did not move.

## The fallout the plan did not predict — a finding

`plan.md`'s FR-6 paragraph says "**This is the one place the change can reach a landed test**",
meaning two same-owner entities in a world built with no relation. That is wrong in both halves.
No test in the suite has that shape, so FR-6's second arm reached nothing; and **six** landed tests
were reached by FR-4's ordinary two-way write instead.

Every one of the six named one direction hostile and then inferred the other's blankness from
behaviour: the victim never fought back, the placement nobody was hostile to acquired nothing.
That was a free consequence of the relation being inert through combat, which is what this story
removes. Mechanism, identical in all six — the struck side's cell flips on first contact, its
group notices the striker at the next decision, and, these fixtures always hitting, it kills the
attacker, whose order goes with it. The assertions were failing on a corpse.

Repaired in `e67c471` on a reading of what each test is *for*, and no rule was adjusted to spare
one:

- Where the one-wayness is scaffolding and not the subject, the reverse cell is now **authored as
  locked (bit 1)**, which declines a flip, so the test goes on measuring the release, the cycle or
  the digest under the relation it intended:
  `TestAHostileGroupStartsAFightNobodyOrdered`,
  `TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued`,
  `TestTheAlwaysScoringPopulationMatchesTheGoldenDigest`,
  `TestACommandedAttackIsNotPrivilegedUnderGuard/the exempt stance keeps it`.
- Where the flip **is** the subject — a loaded map, the thing a player watches — nothing is locked.
  `TestALoadedMapFightsWithoutBeingTold` and `TestAStartedMissionFightsWithoutBeingTold` proved the
  store asymmetric by watching who acquired at tick 64; they now read the two bytes off the matrix
  **before a tick**, a stricter form of the same claim, and assert the retaliation rather than
  forbid it.

`tasks.md` gave T2 a four-file list that could not contain this repair, so it is a stage commit
with no trailer rather than a task.

## AC-2, AC-3, AC-4, AC-5, AC-6 — the acceptance the tests carry

**AC-2** — `TestAStruckFlippablePairEndsHostileAndTheVictimsGroupAcquiresTheStriker`: both cells
carry bit 0 after the blow, every other byte of the 50×50 matrix is checked through `Byte`, and
the struck slot's group then acquires the striker where it saw nothing before. Killed by M3, B
and A.

**AC-3** — a locked pair struck and declining. At the cell,
`TestTurnHostileGatesOnTheLowTwoBits`'s `locked` and `hostile and locked` rows; struck **in both
directions in turn**, `TestFR6BothArmsOfTheDiagonal`'s first subtest, where two entities of one
slot exchange blows and both calls name the loaded 2, which does not move. At the world,
`TestAHostileGroupStartsAFightNobodyOrdered` and
`TestAFriendlyRelationDoesNotEndAnAttackAlreadyIssued` after the repair above: the locked side
never acquires the other across 400 and 100 ticks of being struck. All go red under M1.

**AC-4** — the asymmetric pair. `TestALoadedMapFightsWithoutBeingTold` and
`TestAStartedMissionFightsWithoutBeingTold` author `[2][1]` hostile and leave `[1][2]` flippable,
then read both bytes off the matrix at the end: the flippable direction has gained bit 0 and the
hostile one is **byte-identical**, by which point both sides are swinging, so "whichever of the two
swung" is covered rather than assumed. That second half was inferred until `ec86947`; it is
asserted now. Killed by M3 and M4.

**AC-5** — `TestNothingFlipsOnAMissAnOutOfReachSwingOrANamelessParty`, three subtests: a miss, a
swing out of reach, a party naming slot 0. Both arms of FR-6 are `TestFR6BothArmsOfTheDiagonal` —
a loaded map's forced 2 left alone, a world named no relation turning the slot self-hostile. The
second is killed by A and C; see M5 for what the slot-0 arm does not witness.

**AC-6** — `TestAnAbsorbedBlowAndABlowOnADownedVictimBothFlip`, two subtests: damage entirely
eaten by absorption still flips, a victim at 0 health flips the same. M4's `damage entirely
absorbed` kill is the whole of FR-1's "before absorption".

## AC-7 and SC-5 — the form

`pkg/sim/binary.go` was never opened. `formatVersion = 24` is the same integer it was, and this
story's allocated **25 is unspent** — left for the story that adds the remembered attacker, which
does need per-entity fields. Every byte-level pin in `binary_test.go` and every record length it
names is unchanged and green, which is what says no record grew.

One digest **did** move: `TestTheAlwaysScoringPopulationMatchesTheGoldenDigest`,
`0x6ac739b05c2f394b → 0xa9005c2d631dcee9`. Its cause is the fixture, not the form — the repair above authors one relation byte, `[3][2] = 2`, and the relation is the
byte form's last block. `binary_test.go`'s and `hash_test.go`'s digests do **not** move, their
fixtures never taking a blow. The new number was recomputed from a run of this tree, not guessed.
That a flip moves a digest with nothing new to encode is AC-7's third clause, and P-1 and P-5
below are killed by that same digest.

## SC-4 — the mutation battery

Each mutation was applied to the landed source, `./pkg/sim ./pkg/mapload ./pkg/game` run, and the
source restored with `git checkout --`. **A line was not read to decide whether it is witnessed;
it was broken, and what went red was recorded.**

| # | Mutation | Killed by |
|---|---|---|
| M1 | the gate drops its bit-1 half — locked cells flip | `GatesOnTheLowTwoBits/locked`, `FR6BothArms/a loaded map's forced 2 is left alone`, + 4 more |
| M2 | the write assigns instead of OR-ing — bits above 1 lost | `GatesOnTheLowTwoBits/clear and clear: flippable`, `TouchesOnlyItsOwnCell`, `NoRunOfTurnHostileCallsLowersAByte` |
| M3 | only attacker→victim is written | `AStruckFlippablePairEnds…`, `AnAbsorbedBlow…` (both), `ALoadedMapFights…`, `AStartedMissionFights…` |
| M4 | the call moved **below** the absorption test | `AnAbsorbedBlow…/damage entirely absorbed`, `ALoadedMapFights…`, `AStartedMissionFights…` |
| M5 | the slot-0 guard removed from `flipOnBlow` | **SURVIVED — equivalent, see below** |
| A | `turnHostile` made a total no-op | 7 tests over `pkg/sim` and `pkg/mapload` |
| B | forward direction dropped, reverse written twice | `AStruckFlippablePairEnds…`, `AnAbsorbedBlow…` (both) |
| C | the write sets bit 1 instead of bit 0 | 7 tests over `pkg/sim` and `pkg/mapload` |
| D | the gate drops its bit-**0** half | **SURVIVED — equivalent, see below** |
| P-1 | the flip consumes one draw from the stream | `TheAlwaysScoringPopulation…` |
| P-5 | the flip also writes an entity field (`Stall++`) | `TheAlwaysScoringPopulation…` |

**M5 is equivalent, provably rather than plausibly.** `flipOnBlow`'s
`if a.Owner == 0 || t.Owner == 0 { return }` guards calls `relationIndex` already refuses — its
first clause is `if from == 0 || to == 0 || … { return 0, false }` — so `turnHostile(0, x)` and
`turnHostile(x, 0)` write nothing and do not materialise the matrix. No fixture can tell the two
apart because no world can. The guard is a stated redundancy, kept where the code knows it is
about to call twice. FR-2's second sentence is witnessed by behaviour, not by that line.

**D is equivalent for a structural reason worth keeping.** The write is `|= relationHostile`, an OR
of the very bit the gate's bit-0 half tests, so on an already-hostile cell it is a no-op whether
the gate declines it or not. That half becomes load-bearing only if the write is an assignment —
which is **M2, killed**. The pair says what neither says alone: the bit-0 half is defence in depth
against a future assignment, not the mechanism.

Both survivors were looked at against the shape this story could hide — a fixture already hostile,
or whose entities share an owner, either of which would make an arm of the gate dead code. Neither
is of that kind: M1, M2, C and D exercise all four low-bit shapes on a live fixture, and A shows
the whole method is reached.

## The properties

- **P-1** — the flip sits below both draws and consumes none. Witnessed by the mutation: inserting
  a single `uniform(4)` into `flipOnBlow` moves the golden digest. No landed test's rolls moved
  because of this story; the six that moved did so through the relation, not the stream.
- **P-2** — monotone. `TestNoRunOfTurnHostileCallsLowersAByte` runs twelve calls, both directions
  and the diagonal, and asserts no cell ever loses a bit it held. M2 kills it.
- **P-3** — idempotent. `TestTurnHostileTwiceEqualsOnceOnTheSamePair`.
- **P-4** — one place computes the offset. Structural, and checked rather than asserted:
  `grep -rn '\.cells\b' --include=*.go` over `pkg/`, `cmd/`, `internal/` returns, outside
  `relations.go` and tests, only `route.go`'s `routeScratch.cells`, an unrelated int field. Nothing
  else indexes the matrix.
- **P-5** — the relation is all a flip touches. Witnessed by the mutation: adding one entity-field
  write to `flipOnBlow` moves the golden digest.

## The requirements and the decisions

FR-1, FR-3 by M4 and by `TestAnAbsorbedBlowAndABlowOnADownedVictimBothFlip`'s downed arm. FR-2 by
AC-5's three no-ops, with M5's caveat. FR-4 by M3 and B, which drop one direction each and are
killed by different assertions. FR-5 by M1, M2 and C. FR-6 by `TestFR6BothArmsOfTheDiagonal`,
its two subtests killed by M1 and A. FR-7 by `TestTurnHostileOnASlotTheMatrixDoesNotHoldIsANoOp`,
which asserts an empty relation stays `nil`. FR-8 by AC-2's shape: nothing is re-issued on the
tick of the flip and the struck group acquires at its **next** decision. FR-9 by the form section.

DD-1 by P-4's grep — the low-two-bits test exists once. DD-2 by `grep -c flipOnBlow
pkg/sim/combat.go` being 1. DD-3 by `TestFR6BothArmsOfTheDiagonal`, impossible to write if a
same-slot guard existed. DD-4 by M3 and B being killed by *different* tests, which a symmetric
helper would have made indistinguishable. DD-5 by `formatVersion = 24`. DD-6 is a record of what
is not built; nothing in this tree implements any of it.

## Still open, and not chased here

Two items the stage-1–3 lane named for research: `AI-FILTER-001`'s invisibility term has no
published writer of the bit it tests, and `AI-SIGHT-092`'s second writer forces sight to 5 behind
a test on a field whose meaning is unpublished. `spec.md`'s D-1 and D-2 — a missed swing does not
flip, an absorbed one does — remain disclosed calls the corpus does not settle.
