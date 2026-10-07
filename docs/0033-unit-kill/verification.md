# Verification — health in the world

## The gate

Run unpiped, on Windows with `-trimpath`, over the whole tree at the last task commit.

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"

ok  againrom/pkg/sim              3.417s
ok  againrom/pkg/ui               1.608s
ok  againrom/pkg/game             1.059s
ok  againrom/pkg/mapload          0.380s
ok  againrom/pkg/render/terrain   0.605s
... 28 packages, no FAIL, 905 top-level tests and 2739 including subtests
check-no-game-assets: clean (tree scan)
docs/0033-unit-kill/analysis.md     5554 /  7168 bytes  ok (77%)
docs/0033-unit-kill/provenance.md   6956 /  7168 bytes  ok (97%)
docs/0033-unit-kill/spec.md        13307 / 13312 bytes  ok (99%)
docs/0033-unit-kill/plan.md        13304 / 13312 bytes  ok (99%)
docs/0033-unit-kill/tasks.md      T1..T9 each ok, legend+traceability ok
gofmt -l: nothing
```

Twenty-two tests were added over four new files in `pkg/sim` and `pkg/ui`, plus rows in
`pkg/render/terrain`, `pkg/game` and `pkg/mapload`.

## The four digest pins (SC-4)

Each was computed **outside this tree** by a third FNV-1a written from the published constants,
over bytes assembled by hand from the format table — never captured from the encoder — and each
is then cross-checked in-tree against `hash_test.go`'s second FNV. `pinBytes` and `rtfBytes` were
re-transcribed by hand at the new 34-byte record; the two tick-1 digests were assembled from the
prose description each already carries (header, grid, one record, the stored route), which the
tests pin independently cell by cell.

```
                   version 4              version 5            bytes
pinDigest          0x66b2f30bc889dee4  -> 0x4835b2e24f792300     168
rlxTick1Digest     0x0f0c9ef3dfd6074b  -> 0xf34baf1219fd7838     113
hybTick1Digest     0xb4f18ad97f3021a3  -> 0x1ca4f71915ecc6de     192
rtfDigest          0x79e69cc8097021b9  -> 0x6fc8c538db18c544     180
```

**None is recomputable by arithmetic from another, and the reason is structural rather than
observed.** FNV-1a is a left fold: the digest of a prefix determines the digest of any extension
of that prefix, and nothing else about one form's digest is recoverable from another's. The four
version-5 forms differ from each other at offset 1 (the tick) and from their version-4
predecessors at offset 0, so no one of them is a prefix of any other — checked as an assertion in
the derivation, not assumed. Every earlier value therefore had to be discarded and the new one
hashed from its own bytes.

## Witnesses

| id | what ran |
|---|---|
| AC-1 | `sim.TestEachPairIsExactlyOneOfTheThreeStates` — the eight pairs, each with the count of predicates that held |
| AC-2 | `sim.TestTheDamageLadderIsWalkedRungByRung` — health and state asserted at each of eleven rungs |
| AC-3 | `sim.TestABlowThatFellsAUnitClearsItsWholeOrder` — 4 cases, order checked as target, coordinates, stall and route |
| AC-4 | `sim.TestEveryNoOpBlowLeavesTheWorldWhereAQuietTickLeavesIt` — 9 cases by digest and by state, with a control |
| AC-5 | `sim.TestACorpseFreesItsCellAndADownedUnitDoesNot` (4), `…InTheTickItDies` (4), `TestAUnitThatIsNotAliveIsGivenNoTargetAtAll` (4) |
| AC-6 | `sim.TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth`, `TestMarshalledBytesArePinned`, `TestThePinnedDigestIsFNV1aOfThePinnedBytes` |
| AC-7 | `sim.TestUnmarshalRefusesEveryVersionButFive` (256 values), `…AWellFormedOlderStream` (6 streams), `…AndLeavesTheReceiverExactlyAsItWas` (40 cases), `…AMalformedRouteAndLeavesTheReceiverAsItWas` (12) |
| AC-8 | `sim.TestTheHealthPairIsCanonicalInFormAndDigest`, `TestTheCorpusCrossesADecodeAtEveryTick` |
| AC-9 | `mapload.TestEveryMapBuiltUnitIsBornAtTheSpawnHealth`, `TestNoOtherWayOfBuildingAWorldSetsEitherHealthField`, `TestFromALMBuildsOneEntityPerUnit` |
| AC-10 | `game.TestTheSeamCarriesTheWorldsOwnLifeStateAndHealthPair`, `ui.TestAnEntryArrivesCarryingWhatItWasHandedAndNothingIsRecomputed` |
| AC-11 | `ui.TestATapTakesTheLivingAndNeverTheDead` (4), `TestABoxTakesEveryCoveredUnitThatIsNotDead` (3), `TestADeadIdIsSkippedByBothReadersAndStaysInTheSet`, `TestTheTwoKeysIssueOneBlowPerMarkedUnitAscending` (3) |
| AC-12 | `terrain.TestHealthBarRectsGeometry` (6), `TestHealthBarFillIsMonotoneAndClampedIntoItsGround`, `ui.TestOnlyTheLivingWithAHealthSystemCarryABar`, `TestABarTakesTheReliefOffsetItsUnitsMarkTakes` |
| AC-13 | **not run.** Manual, and it needs a lawful install — see *Not witnessed* below |
| P-1 | structural, and witnessed twice: AC-1's table counts the predicates that hold, and `TestTheThreeStatesArePairwiseExclusiveAndJointlyTotal` sweeps 81 pairs including both int32 extremes. The "no field could disagree" half is `sim.TestTheCanonicalWorldsFieldSetsArePinned`, which refuses a `Dead` bool as a field nobody declared |
| P-2 | AC-2, AC-3, AC-5, AC-7 — every blow's clearing checked as all four parts of an order, and the decoder's refusal of a target on a unit that is not alive |
| P-3 | **sampled, not proved.** AC-4's nine no-ops and AC-7's refusals, each compared to a quiet tick by digest and field for field |
| P-4 | `sim.TestTheCorpusCrossesADecodeAtEveryTick` — a corpus walked to quiescence under all three kinds, cut at every one of 31 ticks, each cut stepped to the end beside the original, compared by digest and by whole form |
| P-5 | **sampled, not proved.** `game.TestASnapshotPushMovesNoWorldFieldAndNoDigest`, `frontend.TestABlowBecomesACommandInTheQueueTheOrdersUse`, and AC-11's whole-set comparisons |
| SC-1 | AC-1 over its eight pairs, each expectation hand-written, plus the exclusivity sweep |
| SC-2 | AC-2 asserted at all eleven rungs; AC-3's cleared order checked as four parts |
| SC-3 | AC-4, each no-op a case of its own, every one compared by digest — the undefined kind and kind 255 included |
| SC-4 | the four pins above, the offset table extended and still a partition, `pinBytes` re-transcribed, a whole version-4 world refused beside the 256-value sweep |
| SC-5 | AC-5's two cells each tested in both directions, six ticks per run |
| SC-6 | AC-8's corpus, every later tick |
| SC-7 | AC-9 through the exported copy; AC-10 entry by entry |
| SC-8 | AC-11's set compared whole before and after; AC-12's bar at both ends and in the middle, its offset compared as a point |
| SC-9 | the mutants below |

## Mutants

Each was applied to **production** code, run over the whole tree, reverted, and the tree confirmed
byte-identical against a saved copy (`cmp`) before the next.

```
mutant                                          file             tests killed
version byte left at 4                          sim/binary.go          18
death test widened to health <= 0               sim/world.go           21
damage clamped at zero                          sim/step.go             3
occupancy seed skips downed as well as dead     sim/route.go            4
occupancy plane seeded before the commands      sim/step.go             5
dead filter dropped from the tap                ui/command.go           3
dead filter dropped from the box                ui/command.go           4
dead filter dropped from presentSelected        ui/command.go           3
blow keys read before the gesture               ui/app.go               1
blow keys read the raw selection                ui/command.go           6
chip's floor of 1 removed                       game/world.go           1
fill ratio taken before the multiplication      terrain/overlay.go      3
bar drawn over the dead                         ui/overlay.go           3
two bar passes swapped                          ui/overlay.go           1
```

**No survivors, and none declared.** SC-9's five are the first, second, third, fourth and sixth
rows; the other nine attack the state model where the brief asked — the occupancy phase boundary,
the chip amount, the ratio, the key ordering — or complete a pair the plan named only one half of.

Three mutants were **strengthened rather than reported thin.** The seed-skips-downed mutant killed
3 until the AC-8 corpus was made to send a mover onto the same cell while its occupant was downed
and again after it was dead; it then killed 4. The freed-one-tick-late mutant killed 1 until the
in-the-tick case became four arms — onto the cell, past it, the order ahead of the blow, and the
blow that ends a downed unit — and then killed 5. The bar-over-the-dead mutant killed 1 until a
case was added over entries whose `Life` byte **contradicts** the pair beside it, which is the only
shape that can tell reading the seam's answer from re-deriving it; it then killed 3.

Three rows stand at one kill and are reported as they are. Each is a single test written for
exactly that decision, with a control beside it: the key-ordering case has a companion showing the
same tap issues nothing without a key; the chip-floor case is one comparison over the whole queue
covering three floor-sensitive maxima (100, 25, 4 and 0); the pass-order case reads the pass slice,
which is the one place the draw order is observable at all.

## Not witnessed

**AC-13 was not run.** It is the manual criterion and it needs a lawful install: a real map, a
group boxed and ordered, chipped with L until one is downed, then K. Nothing here substitutes for
it, and nothing here claims to. What would witness it is the owner running the story's build
against an install and reporting what the screen did.

**P-3 and P-5 are sampled, not proved,** and the wording is the contract's own. What was measured
is nine no-op commands, fifty-two refused forms and three front-end reads; what is not measured is
the universal statement over every command and every form.

The **downed unit's blocking** is disclosed in the contract and holds here: a body has no exit but
a further blow, so a corridor of downed units is a wall no order dissolves. `AC-5`'s "ordered past
a downed unit" case is that wall, measured.

The **move loop's own alive test** is reached by no command — the arm that applies a move-to
refuses to set a target on a unit that is not alive, and a blow clears the order it fells a unit
out of — so `TestTheMoveLoopRefusesToAdvanceAUnitThatIsNotAlive` builds that state through the
fields, which is what these tests being internal is for. It is a second line, and it is tested as
one rather than left unreachable.
