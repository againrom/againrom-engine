# Verification — typed data classes from the graphics registries

Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` → `plan.md`
§Design decisions; `Tn` → `tasks.md`; `ALM-*`/`REG-*` → the `research/` claim ledger.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. **Two runs:** the first pass, 2026-07-27 at
`research/` `e61153d`, and the *Absent-everywhere defaults* **revision**, 2026-07-28 at `9c01af7`. The
automated evidence reads no game install: `pkg/data` names no filesystem package in any file, tests
included, and every `cmd/classdump` test writes synthetic bytes into `t.TempDir()`. The developer runs
used the owner's lawful GOG install, its root on the command line and **not recorded here**, and their
output — converted game data — went outside the repository. **No class field value, description string
or sprite path appears in this file**, only counts and the two value spaces a claim publishes.

## Gate results

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go') | wc -l
0
$ go test -trimpath -count=1 ./... | grep -c '^ok'
19                                     (+ 7 packages with no test files, 0 FAIL)
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ bash scripts/check-no-game-assets.sh --history
check-no-game-assets: clean (history scan)
$ bash scripts/check-sdd-audit.sh
check-sdd-audit: 18 trailered commit(s) in ac6bd87..HEAD checked
warn 0016-data-classes: verification.md exists, builds/0016-data-classes/README.md does not
check-sdd-audit: ok (41 note(s)/warning(s), none enforced)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 302a04a..HEAD | sed '/^$/d' | sort | uniq -c
      1 0016-data-classes/T1
      1 0016-data-classes/T2
      1 0016-data-classes/T3
      1 0016-data-classes/T4
      1 0016-data-classes/T5
      1 0016-data-classes/T6
      1 0016-data-classes/T7
      1 0016-data-classes/T8
      1 0016-data-classes/T9
```

Re-run for the revision at `9947209` (T11):

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go') | wc -l
0
$ go test -trimpath -count=1 ./... | grep -c '^ok'
19                                     (+ 7 packages with no test files, 0 FAIL)
$ bash scripts/check-no-game-assets.sh && bash scripts/check-no-game-assets.sh --history
check-no-game-assets: clean (tree scan)
check-no-game-assets: clean (history scan)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 302a04a..HEAD | sed '/^$/d' | grep -c 0016
11                                     (T1…T11, one each, over 30 commits)
$ git log 302a04a..HEAD --format='%B' | grep -ci co-authored-by
0
```

Eleven trailers, each id exactly once; the story's six docs commits carry none. `check-doc-budget.sh`
passes for both passes — the revision's contract text was paid for by compaction, `spec.md` at
13312 / 13312 and `plan.md` at 13308, chain intact. `check-sdd-audit`'s **witness** half was red from
each pass's spec commit to its evidence commit, an added `AC` or `SC` having nowhere to be witnessed
until this stage; the trailer half stayed green.

## Automated criteria

```
$ go test -trimpath -count=1 -v ./pkg/data ./cmd/classdump | grep -E '^(--- PASS|ok|FAIL)'
--- PASS: TestAKeyNoSectionSetsTakesItsDefault
--- PASS: TestAWrittenZeroIsNotADefault
--- PASS: TestTheOtherTwoRegistriesKeepTheZero
--- PASS: TestEveryDefaultRowNamesAnIntKeyOfItsOwnTable
--- PASS: TestFileDoesNotInheritInObjectsButDoesInUnits
--- PASS: TestAllIsNumericOrderAndByIDIsSparse
--- PASS: TestScalarOverriddenByAWrittenZero
--- PASS: TestArrayInheritsOnLengthAndNothingClearsIt
--- PASS: TestScalarsChainWhileArraysTakeOneHop
--- PASS: TestParentZeroIsARealReference
--- PASS: TestUnitInventoryRoundTrip
--- PASS: TestUnitInventoryRoundTripThroughAParent
--- PASS: TestObjectInventoryRoundTrip
--- PASS: TestStructureInventoryRoundTrip
--- PASS: TestTheCasesThatAreValuesAndNotErrors
--- PASS: TestKeyTablesPinnedToTheirStructs
--- PASS: TestDescriptors
--- PASS: TestFindSection
--- PASS: TestReadKeySeparatesAbsentPresentAndWrongKind
--- PASS: TestFoldBoundary
--- PASS: TestSpritePathsAreExactStrings
--- PASS: TestSpriteBaseUsesTheResolvedFile
--- PASS: TestSpritePathEmptyWhenNoFileResolves
--- PASS: TestEveryMalformedCaseIsRejected
--- PASS: TestTheTwoReadingsThatMustLoad
--- PASS: TestTheLengthRulesAcceptWhatTheContractAllows
--- PASS: TestEveryLengthRuleNamesAKeyOfItsOwnTable
ok  	againrom/pkg/data
--- PASS: TestClassifyPutsARecordInOneOfFourIndependentBuckets
--- PASS: TestClassifyWidensTheClassKeyWithItsSign
--- PASS: TestClassifyReadsOnlyTheThreeFields
--- PASS: TestBucketTokens
--- PASS: TestDumpPrintsAllThreeCollections
--- PASS: TestRunRejectsAWrongArgumentCount
--- PASS: TestMissingRegistryFails
--- PASS: TestSweepCountsDivertedRecordsAndExitsZero
--- PASS: TestSweepFailsOnAnUnresolvableDirectRecord
--- PASS: TestSweepCountsATypeThreeCodePastTheRegistry
--- PASS: TestSweepFailsOnAnUnresolvedTypeFourReference
--- PASS: TestSweepDoesNotDeDuplicateSources
--- PASS: TestSweepSkipsASourceItCannotDecode
--- PASS: TestRunRejectsAMalformedSweepInvocation
ok  	againrom/cmd/classdump
```

**41** tests, no failure: the first pass's 36 plus the revision's five, which run first above. Each
criterion is carried by named tests in the file its task wrote — one each, named for it, for
**SC-1**, **SC-2**, **SC-3**, **SC-4**, **SC-6**. **SC-5** is `validate_test.go`'s four, **SC-7**
`sprite_test.go`'s three, **SC-8** `inventory_test.go`'s five, **SC-9** `classify_test.go`'s four plus
`sweep_test.go`'s three, beside which `TestSweepFailsOnAnUnresolvedTypeFourReference` carries FR-3's
type-4 exit code; `keys_test.go`'s five pin DD-1 and DD-2's tables, `go test ./internal/archtest` is
DD-7's row. **SC-12** with **AC-10** and **P-6** is `defaults_test.go`'s first two, **SC-13** with
**AC-11** its `TestFileDoesNotInherit…`, and its other two pin the scope limit and the table's
bijection. `TestParentZeroIsARealReference` is the one existing test whose expected value moved.

That closes **AC-1** (SC-1), **AC-2** (SC-2), **AC-3** (SC-3), **AC-4** (SC-4), **AC-5** with **P-1**
(SC-5), **AC-6** with **P-3** (SC-6), **AC-7** (SC-7), **AC-8** with **P-2** (SC-8), and **P-4**, whose
two guards are SC-1's and SC-2's subject. **SC-11**, measured directly:

```
$ go list -deps ./pkg/data | grep '^againrom'
againrom/pkg/formats/reg
againrom/pkg/data
```

No other `againrom/` package and no non-stdlib dependency at all; with no filesystem package named
anywhere in `pkg/data`, **P-5**'s "loading performs no IO" is structural rather than asserted — nothing
in the package could open anything.

## Developer-run evidence — AC-9, SC-10

`classdump` built from this tree with `go build -trimpath` outside the repository. Both invocations
exited **0**, stderr **empty**, each run three times with byte-identical stdout (`cmp`):

```
$ sha256sum <outside-repo>/classdump.exe
8eb09560f08ff19537893f366969ae1c35352ae0eef63268056ab2f1682a26d4
```

### (a) The print half — all that is quoted

```
units/units.reg: 34 classes
objects/objects.reg: 82 classes
structures/structures.reg: 66 classes
```

**34 / 82 / 66** — AC-9's first clause. Aggregates over the same 4 637 lines, counted not quoted:
34 / 82 / 66 class blocks, one `sprite` and one `shadow` line each, **0** empty paths, all 182 sprite
paths carrying their prefix and `.256`, all 182 shadow paths `b.256`, **0** a backslash. **7** object
classes resolve an empty `DescText` and load as values; **52** structures carry `AnimMask`, `AnimTime`
and `AnimFrame` all empty — the sentinel's nil case on shipped data.

*The tool prints `overlay =` where this run printed `shadow =` (renamed 2026-08-01: the `b` sibling
is an overlay layer, not a shadow — see `provenance.md`). The label moved; the 182 paths and every
count above did not, the string being computed identically.*

### (b) The sweep — totals, verbatim but for the root

```
totals sources=38 skipped=0
totals type3 nonzero=71099 resolved=71035 past-registry=64 type4 refs=3141 resolved=3141 unresolved=0 type6 records=8094 direct=6589 resolved=6589 unresolved=0 npc=11 def=1490 both=4 diverted-would-resolve=1505
```

**38 sources, 0 skipped, no `unresolved` line, exit 0.** Every type-4 and every `direct` type-6
reference resolved; the type-3 residual printed as a counted figure and did not move the exit code.
Sources are physical files, not de-duplicated (DD-6), which here did not bite: the 38 are **10 loose
`.alm` and 28 in `scenario.res`**, disjoint by name, so the totals read as a per-map census.

### (c) `R-6`'s first census

Nothing had counted this before. Measured: `direct` **6 589**, `npc` (`Flags` bit 0 only) **11**, `def`
(a live `DefID` only) **1 490**, `both` **4** — total **8 094**. So **15** records set `Flags` bit 0 and
**1 494** carry a live `DefID`, 1 505 diverting in all; `diverted-would-resolve=1505` says **every one
names a `units.reg` `ID` that exists anyway**, so what a divert changes is invisible here.

## Against `ALM-CLS-035` and its neighbours

| figure | the claim | measured here | |
|---|---|---|---|
| nonzero type-3 cells | `ALM-CLS-035`: 71 099 / 38 maps | 71 099 / 38 sources | **agree** |
| cells past the object array | `ALM-CLS-035`: 64, in `scn:131`/`scn:150` | 64 — **34** in `scenario.res::131.alm`, **30** in `::150.alm` | **agree** |
| type-4 records | `ALM-CLS-036`: 3 141, `kind ∈ 1..66` | 3 141 refs, 3 141 resolved | **agree** |
| type-6 records | `ALM-CLS-038`: 8 094, in `units.reg`'s `ID` set 8094/8094 | 8 094; 6 589 resolved + 1 505 would-resolve | **agree** |

Four figures touched, four agreements, the residual agreeing on **location** as well as size; nothing
disagreed, no figure was adjusted. **Two halves this run does not touch:** *where in a map* the 64
cells sit (the claim says rows 0..2; `classdump` prints no coordinates), and the 33 cells
`ALM-CLS-042` calls artless fire variants, counted here among the 71 035 **resolved** because sprite
existence is never checked — the contract (FR-3), not an oversight.

## The inheritance guards, on shipped data

The trap is decided on the corpus too — read off the raw registry with `regtool dump` and off the
loaded collection:

- **6** unit sections write `AttackDelay = 0`; **6** unit classes resolve it to `0`; **2** of the six
  also carry a `Parent`. A loader inheriting on a zero loaded field reads 4, not 6.
- Exactly **1** unit section stores `AttackAnimTime` as a zero-length string, and it has a `Parent`: it
  resolves to the parent's **7**-element track, unchanged. Reading `""` as "clear" leaves it nil.
- **16** of the 34 unit sections carry `Parent`; 23 unit classes resolve `ShootOffset` to nil and load
  as values (DD-8).

## The revision — absent-everywhere defaults

Same install, `classdump` rebuilt from `9947209` the same way. Exit **0**, stderr **empty**, stdout
byte-identical on a repeat run (`cmp`), diffed against the same command at the landed code:

```
$ sha256sum <outside-repo>/classdump.exe
a54af10d1a7540aad6c34bd4c5fa85580ace7b688c9fff45dfda9f964a04fd62
$ diff before after | grep '^[<>]' | awk '{print $2}' | sort | uniq -c
     108 FireObject        (54 lines each side)
      86 Parent            (43)
      66 DeadObject        (33)

DeadObject   -1 x54    + 21 distinct targets over 28 classes, 0 out of range
FireObject   -2 x21    -1 x61     nothing else
Parent       -1 x43    0 x2       ([Object40], [Object42])
```

**Three keys move and nothing else does** — 130 class fields of 4 637 lines, all in `objects.reg`, not
one in the other two blocks: the scope limit measured, not promised.

**SC-14 passes exactly.** `REG-OBJ-047` publishes `DeadObject` = `-1` on **54** and `FireObject` =
`{-2 ×21, -1 ×61}`, read from the bytes and not our loader; both now come out of it too. Before the fix
this run read `DeadObject = 0` on **33** and `FireObject = 0` on **54** — 33 + 21 = 54, 54 + 7 = 61,
the arithmetic that made the defect falsifiable in advance. `Parent` is the same shape unpublished: 43
classes carry none and read `-1`, the **two** naming `Object0` keep `0`.

**The `File` half changed nothing and this run does not demonstrate it.** Every shipped object class
writes its own `File`, so the 182 paths and the **0** empty ones stand and `-sweep`'s output is
**byte-identical** to the first pass's. AC-11 rests on one synthetic test and nothing in the corpus,
which is what the claim behind it says to expect.

## Limitations

- **No rejection path ran on real data.** All three shipped registries are well formed, so every
  *Validation* case, atomicity with them, is carried by SC-5's fixtures alone.
- **The census counts; it does not interpret.** What `Flags` bit 0 and a live `DefID` select is
  `ALM-CLS-038`'s reading; that `-1` is `DeadObject`'s "no dead form" is its *default* decoded, not the
  key's meaning. Key meanings stay unconfirmed (spec, *Key inventory*).
- **`File`'s non-inheritance has no witness on shipped data**, and `units.reg`'s and `structures.reg`'s
  own defaults are not implemented: undecoded at this pin, so both keep the Go zero and nothing here
  says what the engine does there.
- **Left out as data, not evidence:** every class field value, `DescText`, sprite path, every per-source
  census line but the two carrying the residual, every byte of the dumps. **No observation was made by
  a human at a screen**, and AC-9 requires none: it needs an install, which no test may touch.

## Conclusion

**AC-9 and SC-10 pass in full**: 34 / 82 / 66 classes print; the sweep resolves 3 141 of 3 141 type-4
and 6 589 of 6 589 `direct` type-6 references over 38 sources, exits 0 on an empty stderr, and prints
the 64-cell type-3 residual as the counted figure it is. `R-6`'s first census is `direct=6589 npc=11
def=1490 both=4`. **The revision closes AC-10, AC-11, P-6 and SC-12…SC-14**, `DeadObject` and
`FireObject` reproducing `REG-OBJ-047`'s value spaces from our own loader while the `File` half rests
on fixtures alone. Every automated criterion passes; against the claim ledger both runs agree on every
figure they touched.
