# 0162-wear-rule — verification

Base `4b6e4d7`, research pin `e1b27fd`. Machine: Windows 11, Go per `go.mod`. Two lawful roots were
read for the census, `gameversions/en` and `gameversions/ru`; neither is in the repository.

Master moved twice during the story and both landings were merged in. `470f5cf`
(0160-dialogue-dress) at `efb2381`, and `4dc62c8` (0161-spell-art) at the tip. Neither merge
produced a conflict and the submodule pin is unchanged at `e1b27fd` through both.

The one file both other stories and this one changed is `pkg/game/world.go`. Each landing's diff
against that file was read rather than taken on the auto-resolve. 0160 added speaker state and
0161 added a `projectiles *terrain.EffectSet` field, both in the `mapWorld` struct declaration
around line 285; this story's changes are `enqueueEquip` and `wearAllows` around line 1780. The
regions do not overlap and neither addition reads the sutableFor column or the party member's mage
flag. 0160 also touched `pkg/data/itemcode.go`, adding a `FigureDir.Female` method beside the
`Mage` method this story does not use, and `pkg/mapload/spawn.go`, which still spells
`weaponCarrySlot` as its own constant.

Every result below was re-measured on the tree with both merges in it.

## Nothing was watched on a screen

No windowed game was launched for this story. Every result below is a test or a command's output.
The two enforcement sites are witnessed through the same functions the running game calls —
`mapWorld.enqueueEquip` for the equip interaction and `townScreen.ShopScreen` for the shop cells —
but that is a test driving the dispatch, not a person seeing a grey cell.

## Gate

```
go build ./...                                     clean
go vet ./...                                       clean
gofmt -l $(git ls-files '*.go')                    prints nothing
go test -trimpath -count=1 ./...                   all packages ok
scripts/check-no-game-assets.sh                    PASS
scripts/check-doc-budget.sh                        PASS
scripts/check-sdd-audit.sh                         no FAIL (one note: no tasks.md)
scripts/check-sdd-audit-selftest.sh                PASS
scripts/check-hotfix-ledger.sh                     PASS
```

The research submodule's own `scripts/check-*.sh` were run at the pin and pass; the pin is unchanged
by this story.

## Script-gap census — unchanged, and expected to be

```
$ go build -o /tmp/mr ./cmd/missionrun
$ AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
17
$ AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
11
```

`pipeline/milestone-baseline.txt` records for `en m10`: 1 groupcmd sub-command 11, 2 groupcmd
sub-command 15, 13 instant op 2, 1 instant op 20 — 17 nodes. For `en m20`: 11 instant op 2. Both
numbers are what master carried before this story. This story adds no script opcode and moves
neither number. Its result is in `builds/current/`: a mage character can no longer put on plate
armour, and the shop draws what he cannot use on the grey background.

## The shipped census, measured

`TestTheShippedCensusMatchesTheWearRule` (`pkg/game/wearrule_test.go`) reads `world.res::data/data.bin`
through the front end and partitions every row of the three equipment collections. It passes on both
roots:

```
$ AGAINROM_ASSETS=<againrom>/gameversions/en go test -trimpath -count=1 -run TestTheShippedCensus... ./pkg/game/
ok      againrom/pkg/game       1.017s
$ AGAINROM_ASSETS=<againrom>/gameversions/ru go test -trimpath -count=1 -run TestTheShippedCensus... ./pkg/game/
ok      againrom/pkg/game       0.967s
```

Measured, identical on both roots: Armors 30 rows, 19 fighter-only, 9 mage-only, 2 both. Shields 9
rows, all fighter-only. Weapons 27 rows, 19 fighter-only, 2 mage-only, 3 both, 3 neither.

## Disagreement with ITEM-WEAR-056, named

`ITEM-WEAR-056` states that weapon row `rem` "carries no parameter array at all, so the slot is
absent and both bits stay clear". Measured here, that row's parameter array is present and cell 15
holds -1, on both roots:

```
Weapons[23] "rem" = -1
```

The instrument is this build's own `databin` reader over `world.res::data/data.bin`, the same reader
`pkg/mapload/spawn.go` already uses; `spawn.go`'s `carriable` doc records the same cell as "-1 on the
removed row `rem`", from 0136, so this is a second reading of one cell by an instrument that existed
before this story. The claim's counts are otherwise reproduced exactly. Under the instruction-level
rule of `ITEM-WEAR-055` — bit 0 and bit 1 copied independently — -1 reads as usable by both, which
is why the weapons split above says 3 both rather than 2 both and one absent. No shipped behaviour
turns on it: no path in this build resolves that row. This is a question for research, not a
correction made here.

## Witness table

| Id | Where witnessed | Result |
|---|---|---|
| FR-1 | `TestTheWearRuleAnswersTheWholeTable` (`pkg/data/wearrule_test.go`) | pass |
| FR-2 | `TestSuitabilityFromCodeReachesEachOfTheThreeCollections` | pass |
| FR-2a | `TestARowTooShortIsUsableByNeitherAndAMissingRowIsUnknown`, `TestAnUnreachableRowIsUnknownAndUnknownPermits` | pass |
| FR-3 | `TestAMageIsRefusedAFighterOnlyItemAndNothingMoves`, `TestAFighterIsRefusedAMageOnlyItem` | pass |
| FR-4 | `TestTheShopGreysACellTheShownMemberCannotUse` | pass |
| FR-5 | `TestTheSimulationAppliesNoClassRule` | pass |
| AC-1 | `TestTheWearRuleAnswersTheWholeTable`, eight cases stated one per row | pass |
| AC-2 | `TestTheShippedCensusMatchesTheWearRule`, both roots | pass |
| AC-3 | `TestAMageIsRefusedAFighterOnlyItemAndNothingMoves`: `pending` empty, `Carried(7)` unchanged after a tick, `Equipped(7)` all zero | pass |
| AC-4 | `TestASuitableItemStillEquips`, four permitted combinations | pass |
| AC-5 | `TestTheShopGreysACellTheShownMemberCannotUse` for both classes, plus `TestTheCellBackgroundStatesAffordability` (unchanged from 0157) | pass |
| AC-6 | `TestTheSimulationAppliesNoClassRule`: `sim.Step` equips a fighter-only code onto a mage subject's entity | pass |
| P-1 | `pkg/sim/binary.go` `formatVersion` is 46, unchanged; `git diff --stat 4b6e4d7..HEAD -- pkg/sim` is empty | pass |
| P-2 | `go test -trimpath -count=1 ./...` green with `AGAINROM_ASSETS` unset; the census test skips | pass |
| DD-1 | The predicate is in `pkg/data/wearrule.go` and its tests need no table, world or front end | pass |
| DD-2 | `data.SutableForColumn` and `spawn.go`'s `weaponCarrySlot` are separate constants, each with its own doc | pass |
| DD-3 | `TestARowTooShortIsUsableByNeitherAndAMissingRowIsUnknown` asserts both halves of the split | pass |
| DD-4 | `TestASutableForCellIsBitTestedAsItStands`: -1 sets both bits, 4 sets neither | pass |
| DD-5 | `TestAnUnreadableRowDoesNotRefuseAnEquip` and `TestTheSimulationAppliesNoClassRule` together fix the seam: the rule is at `enqueueEquip`, not in `pkg/sim` | pass |
| DD-6 | `ui.ShopBackUnusable` exists and `ShopCell.Occupied` is true for it, asserted in `TestTheShopGreysACellTheShownMemberCannotUse` | pass |
| DD-7 | `TestAnUnusableShelfItemIsNeverDrawnAffordable`: purse 1000000, background still unusable | pass |
| DD-8 | `shopUsable` reads `shopParty()[0].Mage`; the shop tests set that field and the cell flips | pass |
| SC-1 | `go test -run TestTheWearRule... ./pkg/data/` | ok |
| SC-2 | census run above, both roots | ok |
| SC-3 | `TestAnUnreachableRowIsUnknownAndUnknownPermits`, `TestAnUnreadableRowDoesNotRefuseAnEquip` | ok |
| SC-4 | `TestAMageIsRefused...`, `TestAFighterIsRefused...`, `TestASuitableItemStillEquips` | ok |
| SC-5 | `TestTheShopGreysACellTheShownMemberCannotUse`, `TestAnUnusableShelfItemIsNeverDrawnAffordable` | ok |
| SC-6 | full gate above; `formatVersion` 46; mission 10 and 20 counts re-measured | ok |

## Mutation check

Each enforcement site was disabled in turn and the suite re-run, to establish that the tests witness
the code rather than the fixture.

```
enqueueEquip's wear-rule guard disabled:
--- FAIL: TestAMageIsRefusedAFighterOnlyItemAndNothingMoves
--- FAIL: TestAFighterIsRefusedAMageOnlyItem
shopCellBack's unusable arm disabled:
--- FAIL: TestTheShopGreysACellTheShownMemberCannotUse
--- FAIL: TestAnUnusableShelfItemIsNeverDrawnAffordable
```

Both were restored before the gate above was run.

## Existing tests changed, and why

Four synthetic row builders now state the sutableFor cell: `eqWeaponRow` and `chargenWeaponParams`
(`pkg/game/equip_test.go`, `chargen_test.go`), `gaArmorRow` (`rearm_test.go`) and
`shopCollection.EntryParams` (`shop_test.go`). Before this story their rows were shorter than the
column, which now reads as usable by neither, so fifteen existing tests failed at their own setup.
Each builder was given the value 3, usable by either class, which is what four shipped rows carry
and which leaves every one of those tests asserting exactly what it asserted before.

One test's doc comment was corrected rather than its code:
`TestEquippingAMetalPieceIsAcceptedForAMagesOwnCharacter` (`rearm_test.go`) stated that the gate
holds "never a class, an archetype or a statistic to gate ON". Since this story the gate does read
one class bit. The test still passes and its subject is now sharper — a hero whose statistics look
like a mage's carries no mage flag and equips metal armour — so the comment names the flag and keeps
the rest.
