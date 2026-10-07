# Verification — map-editor E2: the headless edit / document model

Task commits, oldest first: `e343bc9` (T1), `9c1e3cc` (T2), `5f88cbd` (T3), `e0c4ec7` (T4),
`7c22f60` (T5), `9c239cf` (T6), `65b5399` (T7). Base `86e5cac`, the task list. Seven untrailered
commits sit in the span, every one docs-only and belonging to another story — six authoring
`docs/0026-unit-collision/` and `4a041c5`, the 0001 revision's evidence — disclosed, not a
violation. Submodule pin frozen at research `e7602bb` throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. The gate ran in a detached worktree at
`65b5399`, so it measured that tree rather than whatever else was loose beside it. Every figure
comes from synthetic byte streams built in test code; no game install is read. No runnable binary
ships here, so the story owes no `builds/` directory and the audit's line about one is a warning.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 606 tests, 2063 counting subtests,
                                              24 packages ok, 4 without tests)
      pkg/mapedit         38 tests, 461 counting subtests
                          (14 of the subtests are FuzzEditScript seed entries)
      pkg/formats/alm     32 tests, 114 counting subtests   (26 + 6 added by T1)
      internal/archtest   11 tests,  36 counting subtests
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0025-mapedit-model
  analysis.md 6260 / 7168      provenance.md 6660 / 7168
  spec.md 12905 / 13312        plan.md 12668 / 13312
  tasks.md T1..T7 all under 1400; legend+traceability 720 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 76 trailered commit(s) in ac6bd87..HEAD checked
FAIL 0025-mapedit-model: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 86e5cac..HEAD | sed '/^$/d' | sort | uniq -c
      1 0025-mapedit-model/T1  ...  1 0025-mapedit-model/T7  (7 ids, each exactly once)
$ git log --format='%B' 86e5cac..HEAD | grep -ci co-authored-by
0
$ sh scripts/check-doc-budget.sh docs/0025-mapedit-model/verification.md
  verification.md (prose) 7930 / 9216 - fenced blocks free
```

That one FAIL is what a story with every task landed and no evidence file is supposed to look like.
This file clears it and carries no trailer. The same commit also corrects two comments in
`pkg/formats/alm` — comments only, so `vet`, `gofmt` and the suite read the tree the same either
way.

## Witnesses

Every id and the named thing that answers for it.

```
AC-1  SC-1                     pkg/mapedit/editor_test.go
      TestNewAcceptsExactlyWhatOpenDocumentAccepts - acceptance agreeing with
      alm.OpenDocument on the fixture and on the rejection fixtures, an error and a nil
      model, never a partial one. TestFreshModelExposesTheInputAndItsView - Bytes()
      equals the input and Map() equals alm.Open of it, over all four fixture variants.
      TestTheModelSharesNoBufferWithItsCallerOrWithItself - the caller's slice mutated
      after the load and a handed-out buffer overwritten; SC-1's independence half.
      pkg/mapedit/unit_test.go TestUnitCountAndRecordReadTheFixtureExactly - SC-1's
      returned-record clause, which landed under T5 and not T3 (see below).

SC-2                           pkg/mapedit/fixture_test.go
      TestFixtureIsAcceptedInBothOrdersAndIsByteStable, TestFixtureDecodesToItsDeclared-
      Values, TestFrameWalkAgreesWithTheDocument (all ten records of each order, against
      alm.Document.RecordPayload), TestFixtureDistinctiveValuesArePresentAndPairwise-
      Distinct, TestComparatorOwnCases - in-region change accepted, out-of-region change
      and reordering rejected.

AC-2  P-1  SC-3                grid half: pkg/mapedit/grid_test.go
      TestGridSetterWritesItsCellAndCarriesEveryOtherByte - tile, altitude and overlay,
      each on a fresh load, region diff through T2's comparator, the view's cell read
      back through the shipped decoder.
      type-0 half: pkg/mapedit/meta_test.go TestMetaScalarSetterWritesItsFieldAnd-
      CarriesEveryOtherByte, TestSetAngleWritesTheCallersBitsAndNotAFloatOfThem (a
      second signalling NaN, distinct from the fixture's), TestStringSetterRewritesIts-
      WholeFieldAndCarriesEveryOtherByte (the residue clause), TestStringSetterAcceptsA-
      FieldFullOfText.

P-6  SC-4                      pkg/mapedit/meta_test.go
      TestSettingAFieldToTheStringTheReaderGaveBackRestoresItsBytes - for every byte
      1..255 alone in a field, the model's own view is asked what it decodes to and the
      model is asked to write that back; both reject sets asserted exactly, 0x80..0xff
      for the name and {0x98} for the description, in both directions.
      pkg/formats/alm/encode_test.go TestFieldImageRestoresEveryByteTheReaderDecoded is
      the same sweep inside alm, with TestFieldImagesCarryTheirCodecsBytes, TestEncode-
      RejectsTextThatLeavesNoRoomForTheTerminator, TestEncodeRejectsRunesTheCodec-
      CannotRepresent, TestEncodeRejectsEmbeddedNULAsARoundTripFailure (the three causes
      asserted as distinct) and TestEncodeImagesAreFieldWidthAndOwnedByTheCaller.

AC-2  AC-3  P-1  P-2  SC-5     pkg/mapedit/unit_test.go
      TestMoveWritesTheTwoCoordinateWordsAndCarriesTheRest (the other 62 bytes carried),
      TestPlaceProducesTheImageTheTestBuiltItself (whole-file comparison against an image
      the test grew from the container contract), TestPlaceAppendsVerbatimAndCarries-
      EveryOtherByte, TestDeleteRemovesTheRecordAndShiftsTheLaterOnesDown, TestPlacing-
      ACloneOfAnExistingRecordIsTheSupportedWorkflow, TestPlaceThenDeletingItReproduces-
      TheOriginalImage - all over both record orders and over an empty and a populated
      type-6 roster, each step re-opened through alm.OpenDocument and written back.

AC-4  P-4  SC-6                grid_test.go TestGridSetterRejectsACellOutsideTheGrid;
      meta_test.go TestStringSetterRejectsWhatItsFieldCannotHold; unit_test.go
      TestUnitSetterRejections - bad indices -1, -70, n, n+1, 1<<20 against three
      history states (fresh, one accepted place, one place undone) and records of nil,
      0, 69 and 71 bytes. Each rejection asserts bytes, view, CanUndo and CanRedo
      unchanged; the third state is the only one that can see a setter truncate the
      redo log on its way to returning an error.

AC-5  AC-6  P-3  P-5  SC-7     editor_test.go TestUndoAndRedoOnAnEmptyHistoryAreNoOps,
      TestUndoRestoresAndRedoReappliesTheExactBytes, TestTwoMutationsUndoTwiceRedoTwice,
      TestAMutationAfterAnUndoDiscardsTheRedoHistory, TestAMutationThatChangesNoByteIs-
      StillRecorded; unit_test.go TestUndoAndRedoOfALengthChangingEdit and TestALength-
      ChangingMutationAfterAnUndoDiscardsTheRedoHistory. Every comparison is against a
      buffer the test cloned before the step it judges, never a recomputed expectation.

AC-7  P-2  SC-8                pkg/mapedit/fuzz_test.go
      FuzzEditScript - 14 seeds green under plain go test, no panic; the corpus read as
      opcodes over the four fixture variants, operands reaching one step past each
      accepted range on both sides. TestFuzzSeedsAcceptEveryMutationKind - the per-kind
      accept counts over the same slice, outside the target.

SC-9                           pkg/mapedit/arch_test.go
      TestNonTestSourceNamesOnlyItsDeclaredAlmIdentifiers, TestTheUnitSettersNeverName-
      TheTypeSixPayloadLength, TestTheScanReadsParsedSyntaxAndNotFileText, and the two
      deliberately-wrong-table tests; internal/archtest green with pkg/mapedit
      registered against pkg/formats/alm alone.
$ git diff --name-only 86e5cac..65b5399 -- pkg/formats/alm
pkg/formats/alm/doc.go            (+5 -1, the one Strings-paragraph sentence)
pkg/formats/alm/encode.go         (added)
pkg/formats/alm/encode_test.go    (added)
      alm.go and document.go untouched, and no consumer of alm changed anywhere in the
      span: the only other files are AGENTS.md, docs/ARCHITECTURE.md,
      internal/archtest/dag.go and pkg/mapedit/*.
```

## The mutation campaigns

Each task made its witnesses fail before trusting them. Counts as the task agents reported them:

```
task  mutants  killed  survivors
T1        8       7     1, closed inside T1
T2        9       9     0   (plus a self-found guard gap in the fixture table, closed)
T3       23      20     3, one disclosed and later closed by T5
T4       13      13     0
T5       26      25     1, unkillable behaviourally, closed lexically by T7
T6       14       9     5   (one first-run survivor closed by strengthening the corpus)
T7        9       8     1, reported and not fixed
```

Beyond the seeds, T6 reported 170,380 fuzzer-built executions with no crash. Re-run for this
record: 21 s fuzztime, 274,275 execs, PASS, no failing input, nothing written under `testdata/`,
tree clean after. The five findings below are the campaigns' real product.

**The ordering rule, unobservable at T3 and discriminating at T5.** Reversing `splice`'s forward
application order killed nothing in T3's suite, because every edit available then was a single
fixed-length part and the order was therefore unobservable — T3 measured that rather than assuming
coverage. T5 closed it: the forward-order mutation is red on six tests and the revert-order
mutation on three, and *every* failing subtest is a `type-6_first` one, with not a single
`type-0_first` case among them. That is exactly the discrimination the plan's adversarial pass
emitted the second record order for, and the prediction and the measurement agree.

**The count is `#type6`, and that is held by construction plus a scan.** Deriving the unit count
from `payloadSize/70` survived every behavioural mutation in T5's campaign and in T6's, because
acceptance requires `len(type6 payload) == 70·#type6`: the two expressions are equal on the model's
entire reachable state. A separating input needs a buffer whose container disagrees with itself,
which no public call produces and `New` rejects on sight, and manufacturing one needs an internal
test that DD-9 forbids. T7's `payloadSize` scan makes the mutant red and so discharges DD-5's
rejection clause — but by construction and a lexical scan, not by discrimination, which is the same
honest limit the determinism wall's source scan carries. Watching both `payloadSize` and
`recPayloadSize` was necessary: watching one leaves the header-word route open.

**The scan's own limit.** The same count read at the raw literal offset `0x08` names neither
identifier and passes every test in this repository. T7 reported it rather than fixing it, and it
is recorded in the determinism wall's own words: the scan proves no such name occurs there, not that
the value cannot arrive another way.

**Two mutants only the coverage test can see, which is why the split exists.** The index mapping
pinned to `-1` and the name builder always returning an unencodable rune are killed by
`TestFuzzSeedsAcceptEveryMutationKind` with `FuzzEditScript` green. That is the direct witness for
SC-8's split and the reason the per-kind counts live in an ordinary test: an input exercising only
two opcodes is not a defect, while a driver that has drifted into rejecting everything must still
go red. Of T6's five reported survivors, three — a transposed cell index, `SetName` writing into
the description field, `MoveUnit` writing four bytes — are killed by the per-setter comparators, one
is the count above, and one is a bounds check unreachable through any public call. This is DD-8's
framing measured rather than asserted: write-back is the identity for any stream `OpenDocument`
accepts, so the per-step re-open proves **acceptance**, and fidelity is the region comparator's
claim, made per setter beside it.

**A correction to the reason recorded for the U+0080..U+00FF case, and it is the orchestrator's.**
The comment in `pkg/formats/alm/encode_test.go` said that such a code point's UTF-8 bytes read back
through `decodeASCII` as themselves, "so the round trip cannot catch it" — true but not
distinguishing, and `encode.go` carried the same imprecision as `decodeASCII` being "the identity on
bytes 0x00..0x7f". `decodeASCII` is `string(cstr(b))` and validates nothing, so it round-trips
*every* NUL-free string, a raw `0x80` and Cyrillic included. Measured in this seat over all
16,843,008 strings of 1–3 bytes: `EncodeName` rejects 14,729,344 with `ErrFieldUnencodable`, and
14,582,016 of those the round-trip guard alone would have accepted. The two guards are orthogonal —
the scan owns the non-ASCII case, the round trip owns the NUL case — and what U+00E9 separates is
"not ASCII" from "not valid UTF-8": it is valid UTF-8 and still not ASCII, so a check written as a
UTF-8 validity test accepts it. Both comments are corrected in this commit; no behaviour changes.
The same sweep confirms the plan needs no correction: over those 16.8 M strings there are **zero**
inputs where DD-4's "any byte ≥ 0x80" and the implemented "any rune > 0x7f" disagree, so DD-4 is an
equivalent, less direct phrasing of what the code does.

## Disclosed notes

- **SC-1's returned-record clause is witnessed under T5, not T3.** T3's `Done when:` names SC-1,
  but `UnitRecord` did not exist until T5, so the independence of a returned record could not be
  measured there. T3 witnessed the buffer and caller halves; T5 witnessed the record half.
- **`PlaceUnit` returns `-1` on rejection.** `spec.md` names no value for it, and the tests assert
  only that a rejected place does not return something a caller could mistake for a placed record
  (`idx >= 0` is the failure). The literal is a contract detail the spec is silent on; nothing
  should be built on it.
- **T7's third test, `TestTheScanReadsParsedSyntaxAndNotFileText`, is in scope, not an addition.**
  The fence says the scan reads parsed syntax rather than file text; `doc.go` and `editor.go` name
  `alm.OpenDocument` in prose three times against one real call, so a text scan would differ, and
  without that case the fence had no witness at all. The difference is now asserted.
- **T3's other two survivors are equivalences, not gaps.** Opening the buffer in place instead of a
  copy is unobservable because `alm.Open` happens to copy everything it returns, and the explicit
  negative-coordinate check is redundant against the `uint64` conversion that already rejects it.
  Both readings were kept for reasons T3 records; neither is claimed as covered.
- **T1's count.** Its commit body enumerates seven mutations, one of them the survivor it then
  closed; the figure of eight above is the task agent's own report and includes the re-run.

## Revision — the absent-record model

```
$ git log --oneline ce1d503..HEAD            (base ce1d503; this file is the third commit)
22279ff feat(mapedit): a record the map does not carry is a rejection, not offset zero
fb116b8 docs(0025): the roster is what the stream carries, and an absent record is a rejection
$ git submodule status
 03a944824127cf9f715c88c5a9cb432b13e4be4a research (heads/master)   frozen, not bumped
$ git log --format='%(trailers:key=SDD-Task,valueonly)' ce1d503..HEAD | sed '/^$/d'
0025-mapedit-model/T8                        (one id, once; the other two carry no trailer)
$ git log --format='%B' ce1d503..HEAD | grep -ci co-authored-by
0
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 683 tests, 1599 counting subtests,
                                              25 packages ok, 3 without tests)
      pkg/mapedit         43 tests, 505 counting subtests
      pkg/formats/alm     36 tests
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0025-mapedit-model
  analysis.md 6990 / 7168      provenance.md 7096 / 7168
  spec.md 13067 / 13312        plan.md 13226 / 13312
  tasks.md T8 1311 / 1400; legend+traceability 767 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh                                     (before this commit)
FAIL 0025-mapedit-model: nothing in verification.md witnesses: AC8 P7
FAIL 0025-mapedit-model: nothing in verification.md witnesses: SC10
$ go test -trimpath -count=1 -run '^$' -fuzz FuzzEditScript -fuzztime 21s ./pkg/mapedit
PASS - 53,427 execs, no failing input, nothing written under testdata/, tree clean after
```

No declared overrun: the additions were paid out of the S-7 duplication `fb116b8` itemises.

```
AC-8  P-7  SC-10               pkg/mapedit/fixture_test.go
      TestThinRostersAreAcceptedAndMissWhatTheyDeclare - the five thin fixtures
      pinned before use: accepted by alm.Open and alm.OpenDocument, written back, the
      document's record count the header's own, and the independent walk agreeing with
      alm.Map.Present on type-3 and type-6. It also refuses a no-type-6 fixture whose
      #type6 is zero, so a zero unit count cannot pass by naming a zero word.
      editor_test.go TestNewAcceptsExactlyWhatOpenDocumentAccepts - agreement is exact
      again; the two former "stricter" witnesses (recordCount 3 and 9) are ordinary
      cases, the five thin rosters join them, and "New accepted a stream alm rejected"
      is asserted on its own line as the direction that must never happen.
      TestAThinRosterLoadsAndTakesTheEditsItCanTake - AC-8's load half over all five.
      grid_test.go TestSetOverlayRejectsAMapWithNoOverlayRecord - the restored grid
      branch, at cells the accepting cases use, with the manufactured W*H plane
      asserted present in the view first so the rejection is not a coordinate
      rejection wearing another name; SetTile and SetAltitude still accepted on the
      same map. unit_test.go TestAMapWithNoUnitsRecordHasNoUnitsAndTakesNoUnitEdit -
      the restored unit branch: count zero against a #type6 of 3, and read, move,
      delete and place each through rejectionChangesNothing.
      TestAnEmptyUnitsRecordIsNotAnAbsentOne - the contrast, stated separately.
```

**Restored, and one not.** type-3 and type-6 absence are back; type-1/type-2 absence stays
unreachable — `alm` refuses such a stream — so no case exists for it.

## The revision's mutation campaign

Production code, whole tree per run, md5 both sides.

```
mutant                                             file / md5 while mutated              result
M1  SetOverlay takes required(3), not optional     grid.go   21e078178762e18ae44c0f48fad8155b  killed
M2  locate leaves an absent id at the zero span    editor.go df16cc1a804c4047ad2c7a51c82d52d7  SURVIVED
M1+M2 together                                     both of the above                          killed
M3  locate walks a constant ten records            editor.go 68004a9db9ef26915b3ab0cf8df0f73f  killed
M4  unitCount drops the presence guard             unit.go   5379c987b1d9ba1ef6a1eb5d51f23cc2  killed
M5  PlaceUnit takes required(6), not optional      unit.go   6f2e58109059167fefd6c9c98b48e458  killed
M6  optional never reports an absent record        editor.go a3dcaf9517131d55ad437e9f42d95a45  killed

baseline, restored and green after every run       editor.go a5bbfa64c42f283ae8aac426541bc650
                                                   grid.go   63311f5fcc85651ad112fd47e11940a8
                                                   unit.go   f4b6abcdaec72b21eb37f4b08cdf185d
```

**M2 survived, as predicted.** The sentinel is a backstop, the presence flag the guarantee. M1
alone dies as `slice bounds out of range [:1282] with capacity 1280`, M1+M2 quietly on
`rejectionChangesNothing`'s "the call was accepted": the sentinel makes the defect loud, the flag
makes it unreachable.

**M3 killed narrower than predicted.** Predicted three rosters red, measured two: the
twelve-record roster (type-6 past the tenth) is skipped by the absence cases and was unexercised
by the load case, so losing its type-6 failed nothing — a real gap. Closed by asserting per roster
what it does carry; re-run, red on all three. The prediction was wrong about coverage, not
behaviour.

**`alm` changed by comment only**: `Groups`/`Present` on the type-5 default this reader does not
apply. The plan's now-false SC-9 clause was corrected in `fb116b8`.

## Contract findings

Nothing in `spec.md` (12905 B) or `plan.md` (12668 B) was found false at the implemented tree, and
neither is edited by this stage. T4 measured the one clause that looked like a candidate — DD-4's
byte-level phrasing of the name's reject set — and found it equivalent to the rune-level
implementation, which the sweep above re-confirms. The span `86e5cac..65b5399` holds exactly the
seven task commits, each trailered once, plus seven disclosed docs-only commits belonging to other
stories.
