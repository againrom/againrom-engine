# Verification — the deterministic walking skeleton

Task commits, oldest first: `f087f5b` (T1), `3b1b6c5` (T2), `cb84d3b` (T3), `4c3e799` (T4),
`8752b2f` (T5), `b9c0155` (T6). Base `f20677a`; four document commits before T1, one of them a
mid-flight contract correction (`98563db` — FR-2 said "mutates" where it meant "advances").
Submodule pin frozen at research `909b330`.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Evidence at `b9c0155`. No developer run
against a lawful install appears below, because this story ships nothing that could make one — see
*The build*. Every number here comes from the synthetic suite.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 439 tests pass, 21 packages ok, 5 without tests)
      of them in this story's three packages: 68 tests, 122 counting subtests
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh             (exit 0; every 0019 artifact under the tight ceiling)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 40 trailered commit(s) in ac6bd87..HEAD checked
  [41 notes/warnings; on 0019 exactly one, the builds/ warning below. The rest are pre-existing,
   unenforced, and all on stories 0000-0013. Elided rather than pasted: their text names ids, and
   an id pasted into this file would be counted as a witness of ours.]
warn 0019-walking-skeleton: verification.md exists, builds/0019-walking-skeleton/README.md does
   not (builds/ is untracked - warning only)
check-sdd-audit: ok (41 note(s)/warning(s), none enforced)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' f20677a..HEAD | sed '/^$/d' | sort | uniq -c
      1 0019-walking-skeleton/T1  ...  1 0019-walking-skeleton/T6     (6 ids, each exactly once)
$ git log f20677a..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on one line — `0019-walking-skeleton: every task
in tasks.md has landed and there is no verification.md`. This file is what clears it; that is the
stage's own acceptance, and the commit carries no trailer. The remaining `warn` is the build
convention's, and it is correct: there is no build directory, and there should not be.

## Witnesses

Every id, and the named thing that answers for it. All of it runs windowless, reads no game install
and consults no clock (FR-12).

```
FR-1  AC-3  P-4  SC-1            pkg/sim/world_test.go, pkg/sim/hash_test.go
      TestNewWorldOrdersByAscendingID · TestNewWorldCopiesTheSliceItWasGiven
      TestNewWorldRefusesDuplicateIDs · TestNewWorldZeroesTheResidueOfAClearedTarget
      TestEntitiesHandsOutACopy · TestSnapshotIsIndependentOfTheWorld
      TestDigestsAgreeAtEveryTickWhateverTheInsertionOrder
      Three entities handed over in both orders; the two worlds compare equal as slices, and
      their digests agree at tick 0 and at each of 8 stepped ticks. Writing through the slice
      Entities() returned leaves the world byte-identical, snap() being shown first to be able
      to see such a write at all.

FR-2  AC-2  P-5  SC-1            pkg/sim/world_test.go, pkg/sim/step_test.go,
                                 pkg/mapload/fromalm_test.go
      TestWorldExportedMethodSetIsPinned · TestReadersDoNotMutateTheWorld
      TestStepIgnoresACommandNamingAnAbsentEntity · TestStepLaterCommandForTheSameEntityWins
      TestSteppingALoadedWorldLeavesTheMapUnchanged
      *World's exported method set is read back off the type through reflect and compared to a
      pinned list of six. The sweep calls every method not declared a writer and compares a deep
      snapshot; a method taking arguments fails the sweep rather than being skipped, so a new
      writer cannot slip past it. UnmarshalBinary is the one declared writer — FR-2's second and
      last write path. P-5's map half: a loaded world stepped 4 ticks leaves an independently
      built second map deep-equal, with a guard that the world did move.

FR-3  FR-4  AC-1  P-1  SC-2      pkg/sim/step_test.go
      TestStepWalksOneCellPerTickAndClearsTheTargetOnArrival · TestStepAppliesCommandsBeforeMoving
      TestStepClearsATargetNamingTheEntitysOwnCell · TestStepReadsNoBoundsField
      TestStepAdvancesTheTickExactlyOnceAndDrawsNoRandomness · TestStepMutatesTheWorldInPlace
      TestStepIsAPureFunctionOfStateAndCommands · TestStepIsIndependentOfInsertionOrder
      Six walks, each path written out cell by cell by hand and cross-checked against the test's
      own max(|dx|,|dy|): straight on x, straight descending on y, diagonal, diagonal then
      straight, both axes negative through the origin, and one target outside the 8x6 bounds
      whose whole path is off-grid. Arrival clears the target, leaves no residue, and three idle
      ticks after it move nothing. FR-3's ordering clause has no behavioural witness — disclosed
      below.

FR-5  SC-3                       pkg/sim/rng_test.go, pkg/sim/world_test.go
      TestRNGPinnedSequence · TestRNGOneSeedOneSequence · TestRNGStateFollowsFromSeedAndDrawCount
      TestRNGZeroSeedIsNotAFixedPoint · TestNewWorldTakesItsRNGStateFromTheSeed
      Four draws from seed 0 pinned as literals — SplitMix64 is published, so they are
      recomputable by someone holding no code of ours. Seeds 1 and 2 agree at 0 of 64 draws.
      state == seed + k*gamma checked for 3 seeds over 32 draws each, which is what makes eight
      bytes enough to carry the generator.

FR-6  AC-6  SC-4                 pkg/sim/hash_test.go
      TestFNV1aAgreesWithItsPublishedVectors · TestHashIsFNV1aOverExactlyTheByteForm
      TestEveryFieldChangesTheDigest
      The test's own FNV-1a is checked against published vectors before it is used to measure
      anything, then agrees with Hash() on four worlds including an empty one. AC-6 runs 11
      single-field changes — tick, X, Y, id, a target coordinate, a target set, a target
      cleared, the RNG state, both bounds fields, an added entity — each differing from the
      unchanged digest AND from all ten others.

FR-7  AC-5  AC-9  P-2  P-6  SC-4  pkg/sim/binary_test.go
      TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth · TestMarshalledBytesArePinned
      TestThePinnedBytesDecodeBackToThePinnedWorld · TestMarshalRoundTripsAndReMarshalsIdentically
      TestAnInterruptedRunContinuesIdentically · TestUnmarshalReplacesTheWholeReceiver
      TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas · TestUnmarshalRefusesEveryVersionButOne
      TestAReachedTargetEncodesLikeATargetNeverSet · TestTheFormFollowsTheSliceLengthNotItsCapacity
      TestTheCountFieldIsFourBytesWide
      Offsets and widths are written out by hand from the contract's table and shown to cover
      every byte of the form. AC-9 is 19 spoiled forms — nil, empty, five truncation lengths
      including one whose shortfall divides evenly by the record size, one byte over, versions 0
      and 2, four bad counts including 0xffffffff, descending ids, duplicate ids, two
      out-of-range presence bytes — each on a receiver populated and stepped, each refused with
      the receiver byte-identical, plus the unspoiled control that must be accepted. All 256
      version bytes are tried and exactly one is taken. AC-5 marshals at tick 4 of an 8-tick
      schedule that places two further orders after the cut, resumes into a fresh world, and
      reaches the uninterrupted run's digest; re-marshalling the resumed world is byte-identical.

AC-12                            pkg/sim/binary_test.go, pkg/sim/hash_test.go
      TestMarshalledBytesArePinned · TestHashIsPinned · TestThePinnedDigestIsFNV1aOfThePinnedBytes
      A 3-entity world at a fixed seed: 92 pinned bytes transcribed by hand from the contract's
      table, and the digest pinned beside them. The third test recomputes FNV-1a over the pinned
      bytes and requires the pinned digest, so the two pins check each other rather than both
      falling out of one run of this package. The pinned world is given out of id order with one
      entity carrying coordinates for a target it does not have, so the pin also holds the sort
      and the zeroing of a cleared target's residue.

FR-8  FR-9  AC-4  AC-10  AC-11  P-3  P-7  SC-5    pkg/sim/run_test.go
      TestRunRecordsOneFramePerAdvancedTick · TestRunAdvancesExactlyTicksTimes
      TestRunFramesCarryTheWorldsTickNotTheScheduleIndex · TestRunKeepsNoReferenceToTheSchedule
      TestTwoRunsOfOneScheduleAgree · TestReplayReproducesTheRunsFinalDigest
      TestReplayReproducesOnlyItsOwnRunsInitialWorld · TestReplayOfAnEmptyLogChangesNothing
      TestReplayRefusesALogThatIsNotThisWorlds
      TestReplayLeavesTheMismatchedFrameAndEveryLaterOneUnapplied · TestALogHasNoByteFormOfItsOwn
      A 10-tick run over a 7-entry schedule: one entity turned around twice mid-walk, one tick
      carrying two opposite orders for one entity, ticks carrying nothing, and three tail ticks
      past the schedule's end — so a command dropped, added or reordered changes where somebody
      ends up. AC-4 runs it twice from independently built worlds. AC-10 replays into a second
      world built from the run's own arguments, never a struct copy. P-7 is measured at a
      mid-log mismatch: the world is compared against the state the last good frame left.

FR-10  AC-7  SC-6               internal/archtest/determinism_test.go, dag_test.go
      TestCheckSimDeterminismNamesEachViolation · TestCheckSimDeterminismReportsInFileOrder
      TestCheckSimDeterminismEmptyFileSet · TestSimSourcesAreDeterministic
      TestLoadSimSourcesSkipsTestFiles · TestLoadSimSourcesMissingPackage
      Fourteen evaluator cases, each banned thing caught on its own: os, time, math/rand,
      math/rand/v2 and os/exec (nested under a banned root), a declared float64 field, a float32
      used only in a conversion, complex128, a float literal located to its line, an imaginary
      literal, an unparsable source — and four that must stay clean: a clean source, banned names
      appearing only in a comment and a string, three look-alike paths (math/bits, math/random,
      oshelper, timeline), and the test-file exclusion. An empty file set is a violation, so a
      loader gone hollow reads as a failure. AC-7 then runs the live scan over pkg/sim's
      production sources: no finding, over a non-empty file set, with a further assertion that
      some source still names math/rand in prose — without that, the scan would no longer be
      distinguishable from a text grep. The renamed dag_test case, "stdlib in sim is not a DAG
      violation (CheckSimDeterminism catches these)", keeps its imports and its empty expectation.
      No document still defers the behavioural half:
      $ grep -ril 'defer' AGENTS.md docs/ARCHITECTURE.md pkg/sim/doc.go internal/archtest/dag_test.go
      (no output)

FR-11  AC-8  P-5  SC-7          pkg/mapload/fromalm_test.go, internal/archtest/dag_test.go
      TestFixtureMapIsBuiltFresh · TestFromALMBuildsOneEntityPerUnit · TestFromALMIsIdenticalAcrossLoads
      TestFromALMSeedsFromTheConstant · TestSteppingALoadedWorldLeavesTheMapUnchanged
      TestLiveTreeClean
      Five units on a 40x24 map: the usual 0x80 low byte, raw zero, low bytes 0x04/0xFF (7+255/256
      is still row 7, so the fraction is dropped and not rounded), the far corner, and one in the
      top half of the uint32 range where the unsigned shift and a signed division stop agreeing.
      Expected cells are hand-written literals, never u.X>>8 recomputed. The slice order matches
      neither ascending X nor row-major order, so ids from a sort would not be these ids. Two
      loads of two independently built maps agree as slices and as digests. TestLiveTreeClean is
      where "pkg/sim MUST NOT import it or any other againrom package" is actually enforced.

FR-12  SC-8                     go list, internal/archtest
      TestLiveTreeClean (its CheckSimTests half) · TestCheckSimTests
      $ go list -f '{{.ImportPath}} test:{{.TestImports}} xtest:{{.XTestImports}}' \
          ./pkg/sim ./pkg/mapload ./internal/archtest
      againrom/pkg/sim       test:[bytes reflect testing]   xtest:[]
      againrom/pkg/mapload   test:[]  xtest:[againrom/pkg/formats/alm againrom/pkg/mapload
                                             againrom/pkg/sim reflect testing]
      againrom/internal/archtest test:[os path/filepath strings testing] xtest:[]
      pkg/sim's test imports are held to the standard library by a check that runs — CheckSimTests
      over the live tree, with TestCheckSimTests showing it rejects a cross-tier and an external
      import. pkg/mapload's are not; see below. No file, window or clock appears in either set,
      and the whole suite ran green above with no game install present.
```

## Mutants, re-run in this seat

Every survivor the tasks disclosed was re-run here rather than taken on report. The mutation is
applied to the tree, `go test -trimpath -count=1 ./pkg/sim ./pkg/mapload ./internal/archtest`
decides, the tree is restored, and two controls are included so a survivor cannot be read as a
suite that fails to run.

```
M1  the move phase walked in DESCENDING id order              SURVIVED
M2  a frame's commands applied TWICE inside one Step          SURVIVED
M3  mapload.Seed's literal changed by one                     SURVIVED
M4  the header length guard weakened to headerLen-1           SURVIVED (argued equivalent)
M5  the frame appended AFTER the step rather than before      SURVIVED (argued equivalent)
K1  NewWorld sorts descending                                 KILLED   (13 tests, 5 files)
K2  step1's axis sign flipped                                 KILLED   (7 tests, 2 files)
M6  Step called twice per frame inside Run                    KILLED   (6 tests)
```

## What no test sees

- **FR-3's ordering clause has no behavioural witness anywhere in this story.** M1 walks the move
  phase backwards and the suite stays green, because moves never interact: no entity reads another,
  so any permutation yields the same state. The clause holds *by construction* — the constructor
  sorts, the step walks that slice — and the sort is witnessed (K1 kills). The ordering itself is
  confirmed only by reading the code, and becomes load-bearing at the first mechanic where two
  entities contend for one cell.
- **A frame's commands applied twice inside one `Step` is undetectable** (M2). Last-write-wins makes
  a repeated move-to idempotent. That is true only while move-to is the sole command and stops being
  true for the first command that *accumulates*. Calling `Step` twice per frame *is* caught (M6).
- **`mapload.Seed`'s own value is the one unpinned constant in the story** (M3). Change the literal
  and every world the loader builds changes, every digest with it, and nothing turns red: the
  loader's seed test compares against a world built from that same constant. It was left unpinned
  deliberately — AC-12 pins a world `pkg/sim` constructs, not one the loader constructs, and a pin
  in `pkg/mapload` would make a legitimate `pkg/sim` encoding change fail in a package that did not
  change. The limitation is real either way and is recorded, not argued away.
- **FR-11's second sentence is enforced outside the package it is written about.** Nothing in
  `pkg/mapload` witnesses "`pkg/sim` MUST NOT import it or any other `againrom` package"; it rests
  entirely on `internal/archtest`'s live DAG test.
- **SC-8 outside `pkg/sim` is "read the imports", not "a check runs".** `CheckSimTests` covers
  `pkg/sim`'s tests alone. `pkg/mapload`'s test imports are listed above and were read by eye; no
  check would fail if a later test there opened a file or read a clock.
- **The behavioural scan's limit is lexical.** It proves no such import, type name or literal occurs
  in `pkg/sim`'s non-test sources — not that nondeterminism cannot arrive another way. A float
  reached through another package's untyped constant is invisible to it.
- **Coordinate wraparound is unhandled and unspecified.** Positions and targets are `int32` and the
  step neither clamps nor saturates, so an entity ordered at the extreme of the range would wrap.
  FR-4 permits off-grid deliberately, the wrap sits at coordinates nothing in this story can reach
  or produce, and it is named as a limitation rather than a defect.
- **Two survivors are argued-equivalent, not gaps.** M4 weakens the header guard by one byte, which
  the modulus check refuses one line later — the shortfall no longer divides by the record size. M5
  appends the frame after the step instead of before it; the frame's tick was already read, so the
  log is identical. Both are recorded as equivalent with the argument rather than counted as kills.
- **AC-12's pin is a change detector and is honest as one.** It says the encoding has not moved; that
  the encoding is *right* is said by the hand-written offset table and by AC-6's per-field digest
  sensitivity, and those are separate assertions.
- **The RNG is owned and never consumed**, so its only observable is the digest and the byte form.
  Nothing here exercises it inside a step.
- **No real map has ever been through `FromALM`.** The loader's fixture is an `alm.Map` literal, and
  nothing in the tree calls the loader outside its own test. A decoded shipped map has not been
  loaded into a world; what is claimed is the transform's arithmetic on positions restated from the
  format contract.
- **No performance measurement.** No criterion asks for one.

## The build

Confirmed independently in this seat: nothing outside the two packages imports either of them.

```
$ grep -rn 'againrom/pkg/sim\|againrom/pkg/mapload' --include=*.go . \
    | grep -v '^./pkg/sim/\|^./pkg/mapload/\|^./research/'
./internal/archtest/dag_test.go:104: ... {"pkg/game": {"againrom/pkg/ui", ..., "againrom/pkg/sim"}}
```

That single hit is a string inside a synthetic import graph, not an import. No `cmd/` binary reaches
the simulation core, so **0019 ships no runnable deliverable and `builds/0019-walking-skeleton/` does
not exist** — correctly, and that is why the audit's `builds/` warning fires above. The convention is
otherwise prospective, so the absence is written down rather than left to be read as an omission.

## Contract findings

Nothing in `spec.md` or `plan.md` was found false. Every FR clause was checked against the shipped
code and its witness, and the two places where the wording is weaker than it looks are disclosed
above rather than corrected: FR-3's ordering clause is unwitnessed behaviourally, and SC-8's second
clause is enforced by a check for one package and by eye for the other. `internal/archtest`'s DAG
allow map is unchanged by this story, and the import-graph check is green.
