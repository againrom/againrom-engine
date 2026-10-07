# 0138-stacks — verification

Windows 11, Go 1.26.1, research submodule pinned at `e384c30`. Both installs read:
`gameversions/en` and `gameversions/ru`. Every command below was run on a clean tree.

## The deliverable — four items off two bodies land in two slots

`missionrun -mission 10 -attack p0:u20 -attack p0:u19 -take p0:30:61 -take p0:26:59 -ticks 12000`,
**byte-identical on both roots**:

```
mission 10  scenario/10.alm  80x80  36 entities
party p0 skills before: [0 10 0 0 0 0]
attack 1  p0 -> u20 : FELLED it after 204 ticks, victim at -5 hp, attacker facing E
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12) "Soft Boots" defence 1 absorption 0
  before, carried: none
  ground (30,61): 0801007(slot 1) "Common Wood Club" 1012028(slot 12) "Soft Boots" defence 1 absorption 0
attack 2  p0 -> u19 : FELLED it after 18 ticks, victim at -4 hp, attacker facing NW
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12) "Soft Boots" defence 1 absorption 0
  before, carried: none
  ground (26,59): 0801007(slot 1) "Common Wood Club" 1012028(slot 12) "Soft Boots" defence 1 absorption 0
take 1  p0 <- (30,61) : took it
  taker carried: 0801007(slot 1) "Common Wood Club" 1012028(slot 12) "Soft Boots" defence 1 absorption 0
take 2  p0 <- (26,59) : took it
  taker carried: 0801007(slot 1) "Common Wood Club" [x2] 1012028(slot 12) "Soft Boots" defence 1 absorption 0 [x2]
outcome undecided at tick 286
party p0 skills after: [0 10 0 0 0 0]
```

Two bodies, four items, **two elements**. Take 1 leaves one club and one pair of boots; take 2 adds
a second of each and the container does not grow — the counts do (AC-4, FR-6). The count is
bracketed because `formatItem`'s row ends in an armour piece's own two numbers since 0136, so a
bare suffix read as a quantity of the absorption.

**The count observed on a real install is 2, not 3.** Mission 10's reachable set holds exactly two
clubmen: `u21`, `u57` and `u58` carry the same loadout but were never reached inside 30 000 ticks,
and mission 20's units drop nothing at all. The mechanism does not distinguish 2 from 3 — the
counts of 3, 4 and 70 000 are witnessed by the tests below — but no drive against either install
produced a third identical drop, and this file does not claim one.

## The gate

```
go build ./...                                   clean
go vet ./...                                     clean
go test -count=1 -trimpath ./...                 34 packages ok, 0 FAIL
gofmt -l $(git ls-files '*.go')                  printed nothing
bash scripts/check-no-game-assets.sh             check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh                 exit 0
bash scripts/check-hotfix-ledger.sh              check-hotfix-ledger: ok
```

`gofmt` was read for output rather than chained: it exits 0 whether or not it names a file, and on
this branch it did name two — both written with CRLF by a scripted edit, corrected in place.

## Acceptance criteria

| AC | Evidence |
|---|---|
| AC-1, AC-2, AC-3 | `TestAnAuthoredContainerFoldsAndReadsBackBothWays` — three of one code and one of another build two elements at counts 3 and 1, read back as four codes and as two elements |
| AC-4 | the drive above, and `TestTakingASackMergesIntoTheTaker` |
| AC-5 | `TestAFelledBodyDropsUnitsAndNotElements` |
| AC-6 | `TestGiveAllMergesIntoTheReceiver` |
| AC-7 | `TestEquipTakesOneUnitOffAnElement` |
| AC-8 | `TestEquippingTheLastUnitRemovesTheElement` |
| AC-9 | `TestAWorldHoldingACountedElementRoundTripsByteIdentically`, `TestACountedElementIsTwoWorldsFromTheSameElementAtAnotherCount`; **no pinned digest constant moved** — the whole T1 diff contains no 64-bit hex literal, and every `pre*Digest` in `binary_test.go`, `hash_test.go`, `routeform_test.go` and `budget_test.go` is untouched and green |
| AC-10 | `TestADecodedRecordNamingOneCodeTwiceFoldsRatherThanIsRefused`, `TestARecordNamingOneCodeSeventyThousandTimesFoldsWithoutLoss` |
| AC-11 | `TestRenderInventory/a_pack_element's_count_above_1_composes_differently_from_the_same_element_at_1_(0138_AC-11)` |
| AC-12 | `TestRenderInventory/a_counted_subject_with_no_font,_and_the_zero_subject_with_a_font,_both_compose_(0138_AC-12)` |
| AC-13 | the drive above, and `TestCarriedLineNamesElementsWithACountBesideAnyAboveOne` |
| P-1, P-3 | `TestFoldingIsIdempotentAndInvertsExpansion` |
| P-2 | `TestUnitsAreConservedAcrossTheActsThatMerge` |
| P-4 | `TestFoldingMergesIntoTheFirstPlaceAndDropsACountOfZero`, `TestTheCountFieldIsFourBytesWide` |
| P-5 | AC-12's two cases; `TestBuildInventoryPackStopsAtTheArraysLength` and `TestBuildInventoryPackRefusesANilSourceWithoutPanicking` for a container longer than the window; `TestARecordNamingOneCodeSeventyThousandTimesFoldsWithoutLoss` for a count no reader is sized for. The whole suite runs with `-count=1` and reports no panic |
| FR-10 | this story moved `formatVersion` by nothing and left the byte-form offset table untouched — the number on the merged branch is **39**, and it is another story's bump arriving through master; `TestUnmarshalRefusesACarriedCountThatOverrunsTheBuffer` still holds |

## What the assertions actually witness

Each line below was **reverted, the suite re-run, and the revert undone**; the named test went red
each time.

- the fold removed from `normaliseHoldings`, from `TakeSack`, from the give-all instant, from
  `decodeCarried`, and from `equip` — five separate reverts, five separate failures;
- `encode`'s carry-section preallocation put back to `len(codes)` — `MarshalBinary` panics, which is
  the failure a short buffer should produce;
- the corpse drop made to emit one code per element rather than per unit;
- `equip` made to take the whole element rather than one unit of it;
- `enqueueEquip` put back on `Carried` instead of `CarriedStacks`:
  `TestEnqueueEquipResolvesTheIndexAgainstElementsNotFlatCodes` failed with
  `pending = [], want exactly one command — element 1 is the sword`.

## The milestone did not move

`missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, the gate's own argv:

```
en	outcome lost at tick 224	census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
ru	outcome lost at tick 224	census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
```

Identical to `pipeline/milestone-baseline.txt` on both roots.

## Reconciliations after the merge with master

`0136` landed while this story was in the lane and its own `-wear` arm resolved an equip index
against `Carried`'s flat expansion — the same defect FR-9 required moving `enqueueEquip` off. It now
reads `CarriedStacks`. That is FR-9 reaching a caller that arrived after the tasks were written, not
new scope: no requirement changed. `enqueueEquip` also keeps `0136`'s widened `EquipTarget` gate
rather than this branch's older `EquipSlotFor`/`WeaponFromCode` pair.

`0135` landed after that, taking the byte form to version **39** for its own skill state and giving
the drive a `-tail` flag and two skill lines. Its version number, its census tail and its report
lines are its own; the carry section, the count and the fold are untouched by it. Both drives above
were re-run on the merged result and print the same ticks, the same counts and the same outcome.

## Limitations

- **No count above 2 was observed against an install**, for the reason stated above.
- **The drawn `x3` is witnessed by a pixel-difference test, not by a screenshot.** This build has no
  headless renderer for the inventory window, and one is not worth a story.
- **A container's load is not implemented.** `ITEM-STACK-003` gives it as `Σ weight × count`; this
  build carries no per-unit weight and no running load anywhere, so there was nothing to multiply.
- **Merging on every add is authored, not decoded**, as is treating every item as stackable — both
  are in `spec.md`'s disclosed divergences and in `provenance.md`.
</content>
</invoke>
