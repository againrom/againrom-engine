# Verification — the plane a map describes, and what it costs

Run on Windows 11, Go 1.26.1 (windows/amd64), the `research/` submodule at its
frozen pin `8c92427` and not bumped, **no game install present anywhere on the
run**. Every test below builds its own cells.

## The gate

Run unpiped, output redirected and read back; `-trimpath` because Windows
Defender quarantines an untrimmed test binary on this machine.

```
$ go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
  sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
  sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
                                                                    exit 0

go build, go vet and gofmt print nothing when clean, and did.
go test: 25 packages "ok", 3 with no test files, 0 FAIL. In full:
  cmd/{againrom,almtool,classdump,mapview,regtool,terraintool}
  internal/{archtest,notices,synth}
  pkg/{data,game,mapedit,mapload,sim,ui,vfs}
  pkg/formats/{alm,reg,res,spr16,spr256}
  pkg/render/{camera,frame,menu,terrain}
  no test files: cmd/restool, cmd/sprtool, pkg/render
(per-package timings are the only figures that drift between runs and are
 left out for that reason; every other figure below is this run's.)

check-no-game-assets: clean (tree scan)

  (the budget sweep prints one block per story; this story's rows:)
docs/0036-terrain-passability/analysis.md      6487 /  7168 bytes  ok (90%)
docs/0036-terrain-passability/provenance.md    7148 /  7168 bytes  ok (99%)
docs/0036-terrain-passability/spec.md         13244 / 13312 bytes  ok (99%)
docs/0036-terrain-passability/plan.md         13285 / 13312 bytes  ok (99%)
docs/0036-terrain-passability/tasks.md T1      1227 /  1400 bytes  ok (87%)
docs/0036-terrain-passability/tasks.md T2       843 /  1400 bytes  ok (60%)
docs/0036-terrain-passability/tasks.md T3      1128 /  1400 bytes  ok (80%)
docs/0036-terrain-passability/tasks.md T4       781 /  1400 bytes  ok (55%)
docs/0036-terrain-passability/tasks.md T5       618 /  1400 bytes  ok (44%)
docs/0036-terrain-passability/tasks.md T6       748 /  1400 bytes  ok (53%)
docs/0036-terrain-passability/tasks.md (legend+traceability)
                                                859 /  1200 bytes  ok (71%)
docs/0036-terrain-passability/verification.md  9197 /  9216 prose  ok (99%)
0036-terrain-passability: plan <= 1.2 x spec  13285 <= 15892 bytes  ok
0036-terrain-passability: tasks <= 1.2 x plan  6204 <= 15942 bytes  ok

check-sdd-audit: 160 trailered commit(s) in ac6bd87..HEAD checked
warn 0036-terrain-passability: verification.md exists,
     builds/0036-terrain-passability/README.md does not (warning only)
check-sdd-audit: ok (44 note(s)/warning(s), none enforced)
```

## Every criterion, and what witnessed it

All in `pkg/mapload`; each name is one test function.

| id | witnessed by |
|---|---|
| **AC-1** | `TestEachArmBlocksItsOwnCellAndNothingElse` — six interior cells, one per given; the sixth lies past the end of BOTH planes, so "nothing at all" is literal |
| **AC-2** | `TestGroupSevenBlocksExactlyAtBlendLevelThreeOrMore` — all 56 pairs, 35 blocking, the straddling pair named |
| **AC-3** | `TestTheRejectedSubCellsTheUnwrittenGroupsAndTheIgnoredBits`, 17 probes |
| **AC-4** | `TestTheWaterRangeIsReadWhole` — three words the corpus carries 0 of |
| **AC-5** | `TestAnyNonzeroSceneryCodeBlocksAndZeroDecidesNothing` — the four codes, code 0 on a blocking word, and the open-word control that gives the fifth case its meaning |
| **AC-6** | `TestTheBorderCoversTheOuterEightRingsAndNothingElse` — 24x24 cell by cell, then 16x24 and 24x16 |
| **AC-7** | `TestTheDerivedPlaneIsCarriedHashedAndReadBack` — see *The digest pin* |
| **AC-8** | `TestTheWorldCarriesTheDerivedBytesAndOnlyThisPathDerivesOne` — four worlds read through their byte forms |
| **AC-9** | `TestAUnitCrossesAtTheGapAndNeverStandsOnWater` (arrival) and `TestAChannelWithNoGapEndsTheOrderInTheTickThatFindsIt` / `TestACrossingHeldByABodyIsWaitedOutAndGivenUpOnTheSixteenth` (the two give-ups) — **and see *The contract and the tree disagree*** |
| **AC-10** | `TestTheDerivationIsTotalOverEveryMapShape` (nine shapes) and `TestEveryMapShapeBuildsAUsableWorld` (seven, each marshalled, read back and ticked) |
| **AC-11** | **not run here** — see *What was not run* |
| **AC-12** | **not run here** — see *What was not run* |
| **P-1** | AC-1's whole-byte comparison over every cell of every fixture, and every world above being one `sim.NewWorld` accepted — a byte above bit 1 is refused rather than masked, so a plane it takes is a plane inside the bound |
| **P-2** | AC-6's counts: the air bit on 512 of 576 cells and on none of the 64 interior ones |
| **P-3** | AC-10's shapes; no shape raised, panicked or returned another length |
| **P-4** | `TestTheDerivationReadsTheExtentAndTheTwoPlanesAlone`, and AC-8's fourth map — differing in altitudes, units, objects, triggers, groups and name — deriving the same plane |
| **P-5** | AC-7's digest, pinned to a number derived outside this tree, and AC-9 in both routing modes over the same plane |

| id | witnessed by |
|---|---|
| **SC-1** | AC-1 and AC-5, every arm on its own interior cell, every byte compared whole against a whole-plane expectation, the five overlay codes each its own case |
| **SC-2** | AC-2, expected levels a hand table in the test — this test package is EXTERNAL to `mapload`, so the production table is not reachable from it even by accident |
| **SC-3** | AC-3 and AC-4; the ignored-bit probes ride a group-7 blocking word, which mutant M3 shows is the only word that discriminates |
| **SC-4** | AC-6, its border predicate written from FR-4's words with a literal 8 |
| **SC-5** | AC-7 — the split expectation, the external digest, the second FNV, the byte-for-byte trip back after a routing tick |
| **SC-6** | AC-8, AC-10 |
| **SC-7** | AC-9 in both modes, the channel and its gap inside the interior, the cell read against the plane after every tick |
| **SC-8** | seven mutants below, no survivors |
| **SC-9** | **not run here** — see *What was not run* |

## The digest pin, and why it is not a capture

`gfDigest = 0x578ed5f38a8547b1` was produced by a program **outside this tree**
that re-implements FR-2's three arms, FR-3 and FR-4 from `spec.md`'s text,
derives the 576 cells of T4's hand-built map, assembles the 648-byte form from
the offsets the byte form's own documentation states, and takes FNV-1a over it.
Its independently derived interior came out **cell for cell equal** to the
literal 8x8 table written by hand in the test:

```
$ python 0036-pin.py                     # outside the repo, no Go code read
interior rows 8..15, columns 8..15:
     0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01,
     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
     0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00,
     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
     0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
     0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
cells 576  ring(0x03) 512  blocked-ground 524  open 52
form bytes 648
digest 0x578ed5f38a8547b1
```

The test then pins two things that have to agree separately: `Hash()` equals that
number, and a **second FNV-1a**, written out in the test from its published
constants and checked against its published vectors first, equals it too over the
bytes this tree marshalled. Recorded from a run of the derivation, the pins would
have agreed with each other whatever the derivation did.

**No version bump.** The codec stays at 5; the grid section (cell count at +30,
cells from +34) has carried this since 0029, this story adds no field and no
section, and no file under `pkg/sim` was edited.

## Mutants

Each applied to production code, the whole tree run, the file restored by copy,
the restore verified by SHA-256 and `git status` confirmed empty afterwards. Six
are SC-8's; **M7 is added**, because the loader's own wiring is the mechanism the
owner will actually feel and SC-8 does not attack it.

```
mutant           edit                                        killed
M1  DD-3  blendMinimum 3 -> 2                            1 func
M2  DD-3  "s <= 13 &&" deleted                           1 func
M3  DD-2  g := (i>>6)&0xf -> g := int(word&0x1fff)>>6    1 func
M4  DD-5  borderDepth 8 -> 7                            12 funcs + 7 subtests
M5  DD-4  overlay != 0 -> overlay != 0 && overlay < 247  2 funcs
M6  DD-4  if a.border -> if a.border || a.water          6 funcs + 1 subtest
M7  DD-1  NewWorld(.., Passability(m), ..) -> .., nil,.. 8 funcs + 6 subtests

golden pkg/mapload/passability.go 618a4d38...199edad
golden pkg/mapload/fromalm.go     6be3d1d2...2e3db0d
   ... seven runs, `go test -trimpath -count=1 ./...`, exit 1 each time ...
restored pkg/mapload/passability.go 618a4d38...199edad
restored pkg/mapload/fromalm.go     6be3d1d2...2e3db0d
git status --porcelain:
(empty above means the tree is byte-identical to the commit)

M4 killed: TestAnyNonzeroSceneryCodeBlocksAndZeroDecidesNothing,
  TestEachArmBlocksItsOwnCellAndNothingElse, TestFromALMSeedsFromTheConstant,
  TestGroupSevenBlocksExactlyAtBlendLevelThreeOrMore,
  TestTheBorderCoversTheOuterEightRingsAndNothingElse,
  TestTheCensusReportsEveryNumberFR8Asks,
  TestTheDerivationIsTotalOverEveryMapShape,
  TestTheDerivedPlaneIsCarriedHashedAndReadBack,
  TestTheFixturePlaneIsTheChannelTheContractDescribes,
  TestTheRejectedSubCellsTheUnwrittenGroupsAndTheIgnoredBits,
  TestTheWaterRangeIsReadWhole,
  TestTheWorldCarriesTheDerivedBytesAndOnlyThisPathDerivesOne
M7 killed: TestAChannelWithNoGapEndsTheOrderInTheTickThatFindsIt,
  TestACrossingHeldByABodyIsWaitedOutAndGivenUpOnTheSixteenth,
  TestAUnitCrossesAtTheGapAndNeverStandsOnWater,
  TestFromALMSeedsFromTheConstant, TestOneDerivedCellMovesTheDigest,
  TestTheDerivedPlaneIsCarriedHashedAndReadBack,
  TestTheFixturePlaneIsTheChannelTheContractDescribes,
  TestTheWorldCarriesTheDerivedBytesAndOnlyThisPathDerivesOne
```

M1, M2 and M3 each kill one test function, and that is precision rather than
thinness: `analysis.md` records that the corpus witnesses **0 cells** of any of
the three cases, so the only instrument that can exist is the synthetic word the
criterion writes, and there is exactly one criterion per decision by design.

## The instrument

`almtool pass` was run over a **synthetic** 40x30 map emitted by `internal/synth`
into a scratch directory outside the repo — not a game asset, and not a stand-in
for AC-11, which needs the shipped corpus. Every figure was re-derived by hand
from the fixture's own placements and agrees:

```
$ go run ./cmd/almtool pass <scratch>/census.alm
passability 40x30 = 1200 cell(s)
by value:   0x00 open 309   0x01 ground 27   0x02 air only 0   0x03 both 864
by mover:   ground blocked 891   air blocked 864
by arm (each with the other four absent, so these overlap):
  tile bit 13     1
  tile index 512..767 25
  strip group 7, blend level 3+ 1
  nonzero overlay code 2
  border, 8 cells deep 864
  arm total 893 against 891 cells blocking ground: 2 cell(s) of overlap
```

By hand: 40x30 is 1200 cells over a 24x14 interior, so the ring is 864; the
fixture blocks 24 + 1 + 1 + 1 = 27 interior cells, giving 891 and 309; and the
two cells of overlap are the water and the overlay it places under the ring.

## What was not run, and why

- **AC-11 and SC-9** — the census over a lawful install's shipped maps, read
  against the published corpus figures. This seat has no install. It is the
  story's only end-to-end check and the only one the derivation cannot pass by
  agreeing with itself, so it stays owed. What would witness it: `almtool pass`
  over the shipped `.alm` set and the three value counts, two blocked counts and
  five arm counts recorded beside the published figures.
- **AC-12** — the game on a lawful install, a group boxed and ordered across a
  lake, a mountain range and trees. Needs an install and a window.
- **`builds/`** — untouched by instruction; the build is made in the owner's seat.

## The contract and the tree disagree, on one clause

**AC-9's second half is wrong and I did not soften it.** It says that with the
gap closed the unit "holds cell and target and gives up on the 16th". Measured,
in both routing modes:

- closed by **water**, the unit holds its cell and the order is over in the
  **first** tick. The far search reads terrain and bounds, neither of which moves
  while a world is advanced, so a second far search from an unmoved cell is the
  same search — 0029's own AC-2 pins exactly this over "a wall with no gap in
  it", which is what a water channel with no gap is.
- closed by a **body standing in the gap**, the unit holds cell, target and route,
  its stall count rises one per tick to 15, and the order ends inside the
  sixteenth with no residue. This is the sentence AC-9 wrote.

Both are witnessed by tests that run. The correction is a contract question, not
an implementation one: either AC-9's given becomes an occupied gap, or its THEN
becomes 0029's rule. **`spec.md` was not edited** — it stands at 99 % of its
ceiling and the call is the owner's.

## Can the grid be turned off from outside?

**No, and the papers provide no way to.** FR-6 requires that no other path derive
or default a grid, and `FromALM` is the only path; there is no flag, no
environment variable and no build tag, and inventing one would be scope this
contract does not carry. What IS observable from outside is the plane itself,
through `almtool pass` above and through `mapload.Passability`, both exported.
Reversal is `git revert` of one commit — `16933d8` changes one argument.

## Conclusion

FR-1 to FR-8 are implemented and every criterion that can be witnessed without a
game install is witnessed by something that runs. Two manual criteria and one
success criterion are owed to a seat with an install. One contract clause is
wrong and is reported rather than absorbed.

**The foreclosure holds, at its stated size.** Only the `.alm` type-3 scenery
layer is baked; placed structures are not modelled, no footprint mask is
hardcoded and no bridge is special-cased. The 1113 shipped cells of deck and
doorway the original opens stay blocked, 908 of them decks — so on a lawful
install a map whose only crossing is a bridge has **no crossing**. That will look
like a defect to anyone who has not read C-1, and it is the disclosed price of
refusing to assert data this tree cannot read.
