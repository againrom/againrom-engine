# 0157 — verification

This file has two rounds. The first is the trade model; the second, from
FR-12 down, is the screen the owner ruled onto it after rejecting the first
round's row list.

## The result someone can point at

`pipeline/check-milestone.sh`'s script-gap census is **unchanged**, and that is
the correct answer: this story authors no script node and changes no compiler.

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 17
AGAINROM_ASSETS=<en> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 11
```

`pipeline/milestone-baseline.txt` carries m10 as `1 + 2 + 13 + 1 = 17` and m20 as
`11`. Both match.

The result is in `builds/0157-shop/`: the town's shop room now trades on the
original's own screen, with a picture on every item. The census is unchanged
after the second round as well, re-measured on the merge of master `3819cd3`:
m10 17, m20 11. Against the lawful EN root, winning into the town stocks the
merchant with 220 items at the chapter's own shipped ceiling.

```
AGAINROM_ASSETS=<en> againrom -check -mission 90
againrom: winning mission 90 leads to the town, chapter 30; tavern holds npc 22 -> 30;
shop holds 31; shop prices 0-1000; gold 70100; party 1;
merchant ceiling 1000, shelves ARMOUR 100 WEAPONS 100 MAGIC 20
```

A temporary probe on the same command, reverted before the commit, printed the
first three items of each shelf so the names and prices could be read:

```
ARMOUR  [Bronze Buckler 3] [Hard Leather Small Shield 88] [Hard Leather Large Shield 169]
WEAPONS [Wood Crossbow 333] [Iron Two Handed Sword 800] [Steel Spiked Club 384]
MAGIC   [Hard Leather Helm 13] [Bronze Bracers 25] [Bronze Pike 300]
```

Every price is under the ceiling of 1000, every name resolves out of the shipped
name table, and the magic shelf holds the union of the other two pools, which is
the unenchanted approximation provenance A-4 declares.

## The path from press to pixel, read rather than launched

The owner ruled that the code is self-sufficient and a screen is not to be
established by launching the game. Both directions were read end to end.

**Drawing the table.** `App.Draw` `case ScreenTown` (`pkg/ui/app.go:1906`) →
`a.drawTown()` → `townShopTable(a.flow.town)` (`pkg/ui/town.go`), which asserts
`TownShopScreen` and calls `AtTownShop()` then `ShopTable()` →
`townScreen.ShopTable()` (`pkg/game/shoproom.go`) reads `f.Shop.Table()`, builds
icons through `buildInventoryPack` from `graphicsPrefix + data.ItemIconPath(code)`
out of `Archives.Containers`, and fills the two totals and the purse →
`drawTownShopStrip(a.canvas, shopTable)` draws the caption and five cells, each
with its icon, its side mark and its count.

**Pressing BUY.** `App.Update` `case ScreenTown` (`pkg/ui/app.go:840`) →
`a.stepTown(in)` → `in.Enter` → `a.flow.chooseTown()` → `f.townList.Choose()` →
`f.town.Choose(i)` → `townScreen.Choose` `case roomShop` → `t.shopChoose(i)`,
which rebuilds the same row list and dispatches `shopActionBuy` → `t.shopBuy()`
→ `Shop.Buy(purse)` → guard, debit, `setShopPackItems`, `Town.gold -= spend` →
the returned `ui.TownAction` reaches `applyTownAction`, which calls
`refreshTown()` and sets `f.msg`. The next frame draws the new rows and prints
the sentence at `pickerMessageY`.

**Clicking a place.** `in.PrimaryReleased` → `a.place.WindowToFrame` →
`townShopStripContains(p)` → `townShopCellAt(p)` → `flow.clickShopTable(place)` →
`ShopTableClick` → `shopOffTable` → the place goes back to its shelf or to the
pack, through the same `applyTownAction`.

**The one hop a trace cannot check is geometry**, and that is where the defect
was: the 25-row list ran from y=40 to y=440 and the table sat at y=286, so the
room's rows drew over it and `RowAt` hit-tested rows behind it. The table moved
to y=364, `Picker` gained a per-list window, and two tests now hold the layout —
`TestTheShopListAndTheTableDoNotOverlap` checks the constants against each other,
and `TestTheShopRoomAtRestFitsTheWindowItIsDrawnIn` fails if the room ever
outgrows the space it is drawn in.

**Not witnessed on a running window.** Nothing in this story was seen on screen.
The game was not launched, per the owner's ruling.

## The four things

| Asked for | Built | Where |
|---|---|---|
| generated stock | yes | FR-1, FR-2, FR-3, FR-4 |
| a five-place table both sides trade across | yes | FR-5, FR-6 |
| buy and sell | yes | FR-7, FR-8, FR-9 |
| the merchant's inventory to take from | yes | FR-3, FR-10 |

Nothing he named was cut. What is missing inside them is the consumables shelf,
disclosed below.

## Requirements

**FR-1** the value window and **FR-2** the price: `pkg/data/shopstock.go`,
`ItemValue` and `ItemPrice`, over `ShopPool`'s walk of rows, shapes and mask
bits. `TestShopValueAndPriceDifferByTheRoundingAddend` and
`TestShopPoolAdmitsOnlyTheMaskedMaterialsInsideTheCeiling`.

**FR-3** three shelves cleared and refilled: `Shop.Generate`.
`TestShopGenerationIsSeededAndClearsBeforeItFills`.

**FR-4** filled on the way into the town: `FrontEnd.arriveInTown`, called from
`FinishMission`'s town arm, `originalsave.go`'s between-mission arm and
`Restore`'s town arm. `TestArrivingInTheTownStocksTheMerchant`.

**FR-5** the table and **FR-6** clearing it: `TakeFromShelf`, `PutOnTable`,
`TakeOffTable`, `ClearTable`. `TestTheTableHoldsFivePlacesAndStampsEachSide`,
`TestClearingTheTableSendsEveryPlaceHome`, `TestTheRoomRefusesASixthPlace`.

**FR-7** buy and **FR-8** sell: `Shop.Buy`, `Shop.Sell`, driven through the
screen by `TestTakingFromTheShelfAndBuyingPutsTheGoodsInThePack`,
`TestAPurseBelowTheTotalLeavesTheBuyButtonUnlit` and
`TestSellingCreditsThePayoutAndPutsTheGoodsOnAShelf`.

**FR-9** the screen's figure against the payout: `SellTotal` and `SellPayout`,
and the corner that names the payout beside them (`shopNote`).
`TestSellPaysTheStackAndTheScreenShowsThePerUnitFigure` and
`TestTheScreenStatesBothSellFigures`.

**FR-10** the room: `pkg/game/shopview.go`, `pkg/ui/town.go`'s seam,
`pkg/ui/app.go`'s draw and input arms. `TestTheRoomRectanglesChooseTheShownShelf`,
`TestClickingATablePlaceSendsItHome`,
`TestAClickOnATablePlaceCrossesTheSeamAndRebuildsTheList`.

**FR-11** no item tables: `maskTable`'s assertion and `ShopPool`'s refusals.
`TestShopWithNoTablesStocksNothing`, `TestShopPoolRefusesEveryTableItCannotRead`,
`TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft`.

**FR-12** the five regions: `pkg/ui/shopscreen.go`'s rectangle block, painted by
`ComposeShopScreen`; the resources are resolved by `pkg/game/shopart.go`.
`TestEveryShopControlIsInsideTheFrameAndDisjoint` and
`TestTheScreenComposesWithNoInstallAtAll`.

**FR-13** the three grids: `shopShelfCellRect`, `shopTableCellRect`,
`shopPackCellRect`, filled by `townScreen.ShopScreen`.
`TestTheCellFormulasAreTheDecodedOnes`, `TestTheShelfArrowsScrollByWholeRows`.

**FR-14** what one cell draws: `drawShopCell` and `shopPlaqueIndex`, over the
background `shopShelfCell` chooses. `TestThePlaqueIsChosenByTheDigitCount`,
`TestTheCellBackgroundStatesAffordability`, `TestTheScreenStatesBothSellFigures`.

**FR-15** the shelf arrows: `townScreen.scrollShelf` and `clampGridBase`.
`TestTheShelfArrowsScrollByWholeRows`.

**FR-16** the four rectangles in the room: `shopShelfPickRects`,
`townScreen.chooseRoomShelf`. `TestTheRoomRectanglesChooseTheShownShelf`.

**FR-17** the button panel: `shopButtonRects`, `townScreen.shopButton`, and the
four numbers and four guards `ShopScreen` fills.
`TestAPurseBelowTheTotalLeavesTheBuyButtonUnlit`,
`TestTheFourthButtonClearsTheTableAndLeaves`,
`TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft`.

**FR-18** what a click on a cell moves: `townScreen.ShopClick`'s three grid arms.
`TestTakingFromTheShelfAndBuyingPutsTheGoodsInThePack`,
`TestAPackItemOnTheTableComesBackWhenTheTableIsCleared`,
`TestClickingATablePlaceSendsItHome`.

**FR-19** the characteristics hover: `ShopCell.Info`, filled from
`itemInfoLines`, drawn by `drawShopHover` and resolved by `shopHoverCell`.
`TestEveryOccupiedCellCarriesItsCharacteristics`,
`TestTheHoverAnswersTheCellUnderThePointer`.

## Decisions

**DD-1** the shop is front-end state: `FrontEnd.Shop`, `pkg/game/shop.go`. No
file under `pkg/sim` changed and `formatVersion` is 46, unmoved — see P-3.

**DD-2** the split between `pkg/data` and `pkg/game`: `shopstock.go` prices and
admits, `shop.go` holds the shelves and the table.

**DD-3** `data.MaskTable` and `databin.Collection.EntryRaw`. `plainCollection` in
`shop_test.go` is a `data.Collection` that is not a `MaskTable`, and the shelf it
would have filled comes back empty.

**DD-4** `data.ComposeItemCode` is the one writer of the encoding; `ResolveWeapon`,
`ResolveShield` and `fillArmor` now go through it. The existing tests for all
three still pass unchanged, which is what shows the recomposition is identical.

**DD-5** one draw loop over a table of pools; the magic pool is the concatenation
of the other two.

**DD-6** the seed: `shopSeed(chapter, finished, ceiling)`.
`TestTheSeedMovesWithEveryCampaignInput`.

**DD-7** the ownership stamp is a bool and the origin shelf rides beside it:
`ShopPlace.Mine`, `ShopPlace.From`. `TestASoldItemLandsOnTheShelfItsClassNames`
covers the sell path's separate class rule.

**DD-8** both totals shipped: see FR-9.

**DD-9** every mutation answers a `ui.TownAction`, now from `ShopClick`.
`TestAClickOnATablePlaceCrossesTheSeamAndRebuildsTheList` shows the town is
refreshed after the click. **DD-10**'s windowed row list is superseded by DD-11:
`shopRoomPackPage`, `shopRoomShelfPage`, `shopRoomList`, `shopChoose` and
`ui.TownShopListRows` are all deleted.

**DD-11** the room is a drawn screen: `townScreen.Rows()` answers nil for
`roomShop`, `pkg/ui/app.go`'s `drawTown` writes `ComposeShopScreen`'s pixels and
returns, and the seam is `AtTownShop`/`ShopScreen`/`ShopClick`/`ShopScroll`.
`TestEveryRoomSaysWhatItIsAndCanBeLeft` was widened to allow the shop's empty row
list, which is the one place the removal is visible from outside the shop.

**DD-12** the art is loaded once, lazily: `FrontEnd.shopArt` sets
`shopArtLoaded` before it reads and answers the cache afterwards. Every field of
`ui.ShopScreenArt` is optional and `TestTheScreenComposesWithNoInstallAtAll`
composes the whole screen with all of them nil.

**DD-13** the view carries a class, not a picture: `ui.ShopCellBack` and
`ShopCell.Mine` are what cross the seam; `pkg/ui` maps them to
`ShopScreenArt.Back` and `ShopScreenArt.Plaque`. `shopPlaqueIndex` is integer
division and reads no float — `TestThePlaqueIsChosenByTheDigitCount`.

**DD-14** one hit test in a fixed order: `ShopControlAt`.
`TestEveryShopControlIsInsideTheFrameAndDisjoint` asserts both overlapping pairs
and which control wins each.

**DD-15** the hover reuses `itemInfoLines`:
`TestEveryOccupiedCellCarriesItsCharacteristics` compares the cell's lines with
that function's own answer rather than with a spelling of them.

**DD-16** three shelves in four rectangles: `shopRoomShelves`.
`TestEveryDivergenceIsStatedOnTheScreen` clicks the potion rectangle and reads
the refusal.

**R-1** and **R-2** are covered by `TestShopPoolRefusesEveryTableItCannotRead`
and by the pack tests, which read and write `Carry.Items` and
`PartyMember.Carried` without ever minting a `Carry`.

## Acceptance criteria

**AC-1** `TestShopValueAndPriceDifferByTheRoundingAddend`: 3 × 2.5 × 1.4 = 10.5,
window 10, price 11; a −1 price cell is admitted at ceilings 0, 1000 and 2^30.

**AC-2** `TestShopPoolAdmitsOnlyTheMaskedMaterialsInsideTheCeiling`: a mask of
`1<<0 | 1<<3` at shape 1 admits exactly materials 0 and 3; ceilings 49 and 0
admit nothing from a 50-coin row.

**AC-3** `TestShopGenerationIsSeededAndClearsBeforeItFills`: seed 7 twice gives
equal shelves, seed 8 gives different ones, and a second generation leaves 100 /
100 / 20 rather than 200 / 200 / 40.

**AC-4** `TestTheTableHoldsFivePlacesAndStampsEachSide`.

**AC-5** `TestClearingTheTableSendsEveryPlaceHome` and
`TestAPackItemOnTheTableComesBackWhenTheTableIsCleared`.

**AC-6** `TestBuyRefusesWhatThePurseCannotCoverAndDebitsExactlyWhatItCan` and
`TestAPurseBelowTheTotalLeavesTheBuyButtonUnlit`.

**AC-7** `TestSellPaysTheStackAndTheScreenShowsThePerUnitFigure`: quantity 3 at
unit price 5 pays 8 and shows 9.

**AC-8** same test: the sold stack is on the weapon shelf at count 3, and
`TestSellingCreditsThePayoutAndPutsTheGoodsOnAShelf` shows the pack empty.

**AC-9** `TestArrivingInTheTownStocksTheMerchant`: ceiling 1000 from chapter 30,
100 weapons; the same state stocks the same shop; winning the chapter moves the
ceiling to 5000 and the assortment with it. The save carries no shelf — no field
was added to `Snapshot` and `saveVersion` is unchanged at 1.

**AC-10** `TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft`: every
control of every kind is pressed at indices -1 to 6, nothing moves, and `Back()`
leaves the room.

**AC-11** `TestEveryShopControlIsInsideTheFrameAndDisjoint` and
`TestTheCellFormulasAreTheDecodedOnes`. Two decoded pairs overlap and the test
records which: the button panel reaches x 464..480 into the merchant's room, and
the up arrow's lower edge shares y=31 with the first row of shelf cells. The
gutter past the last table cell, at (432, 340), resolves to no control.

**AC-12** `TestTheShelfArrowsScrollByWholeRows`: the up arrow at the first row
does nothing, one press of the down arrow moves the base to 2 and the first cell
becomes the shelf's item 2, and two hundred further presses leave the base inside
the shelf with the first cell still occupied.

**AC-13** `TestTheRoomRectanglesChooseTheShownShelf`: the grid holds no cell
before any rectangle is clicked; the lower-left rectangle opens the weapon shelf
and the lower-right the armour shelf; a take off the shown shelf leaves the other
shelf's count unchanged.

**AC-14** `TestThePlaqueIsChosenByTheDigitCount` over 1, 10, 100, 1000, 10000,
100000, 1000000 and 9999999, plus a value past the clamp.
`TestTheScreenStatesBothSellFigures` shows a price of 5 printing 5 on the
merchant's side and 3 on the player's.

**AC-15** `TestTheCellBackgroundStatesAffordability`: at exactly the price the
background is `ShopBackAffordable`, one coin short it is `ShopBackItem`, and an
empty place is `ShopBackEmpty`.

**AC-16** `TestTakingFromTheShelfAndBuyingPutsTheGoodsInThePack` and
`TestClickingATablePlaceSendsItHome` cover the move and the return;
`TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft` covers the empty cell.

**AC-17** `TestAPurseBelowTheTotalLeavesTheBuyButtonUnlit`,
`TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft` and
`TestTheFourthButtonClearsTheTableAndLeaves`.

**AC-18** `TestTheHoverAnswersTheCellUnderThePointer` and
`TestEveryOccupiedCellCarriesItsCharacteristics`.

**AC-19** `TestTheScreenComposesWithNoInstallAtAll` composes a 640x480 screen
with a nil `Art` and a nil `Font`;
`TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft` builds the whole view
from a front end with no `Archives` at all.

## Properties

**P-1** no coin created or destroyed: `TestNoCoinIsCreatedOrDestroyed` runs a
take, a put, a buy and a sell and checks the purse against the two totals.

**P-2** no item created or destroyed: the shelf and pack counts are asserted
before and after every move in the four table tests, and the refused sixth take
and refused buy each leave both sides unchanged.

This property found the one defect the story would otherwise have shipped.
`Generate` clears the table before it refills the shelves — correctly, since
`SHOP-LIFE-013` makes generation a clear-and-refill — so a player who left the
shop with his own goods on the table would have had them destroyed by the next
homecoming. Leaving the shop room now clears the table first, which puts his
goods back in his pack and the merchant's back on his shelves.
`TestLeavingTheShopClearsTheTableSoARestockDestroysNothing` walks exactly that
sequence, including the restock afterwards.

**P-3** nothing reaches the simulation:

```
git diff --stat master..HEAD -- pkg/sim   ->  (empty)
grep -n 'formatVersion = ' pkg/sim/binary.go  ->  const formatVersion = 46
```

No `formatVersion` was requested or taken.

## Disclosures, in the words the screen uses

Four divergences are stated on the screen, not only in this file, and
`TestEveryDivergenceIsStatedOnTheScreen` reads the corner rather than a field.

- **The consumables shelf is missing.** The merchant's heading reads *"no
  potions or scrolls yet"*. `SHOP-POOL-021` builds that shelf from the Spells
  collection as item subtypes `0x2a` and `0x29`, and `SHOP-GEN-005` names six
  literal potions out of Magic Items. This tree has no item-code composition for
  either class and research publishes none, so the shelf is absent rather than
  guessed.
- **The stock does not change when a game is reloaded.** The heading reads *"his
  stock is the same after a reload"*. The original seeds the C runtime's `rand()`
  from a clock (`SHOP-RNG-008`), so its assortment was different every time; this
  build seeds from the chapter, the number of missions finished and the ceiling.
  A new homecoming still re-rolls.
- **Nothing on the magic shelf is enchanted.** With that shelf open the corner
  reads *"nothing on it is enchanted yet"*. `SHOP-MAGIC-007` grades what an
  effect is as Unknown.
- **The sell tally and the payout disagree on an odd unit price.** The SELL
  button prints the original's own tally, `SHOP-SCREEN-035`'s own number, and the
  corner reads *"he pays N for the lot"* wherever the two differ. This one is not
  ours — it is the original's own arithmetic (`SHOP-TRAY-027`), kept.

The second round's own divergences are `spec.md` D-1 to D-9. Five of them are
absences a player can see and cannot misread as a decode: the merchant is not
drawn in his room (D-2), the shelves do not flicker (D-3), the two shipped lines
of shop advice are not shown (D-4), the table picture is eight pixels short of
its region (D-5) and the potion shelf is empty (D-6). Three are this build's own
additions: the characteristics hover (D-1), the corner (D-7) and the wheel that
turns the pack (D-9). D-8 is that the screen is driven with the mouse and Escape.

**Nothing was watched on a screen.** No windowed game was launched for this
round. What is verified is the code: the rectangles are asserted against the
decoded numbers, and the whole screen was composed headless into an
`*image.RGBA` and written to a PNG by a temporary probe under `pkg/game`,
deleted before the commit. That probe read the preserved EN root and reported
every resource's decoded pixel size back:

```
shelf   (0,0)-(164,303)   table (0,0)-(472,87)   room  (0,0)-(288,288)
frame   (0,0)-(316,303)   menu  (0,0)-(176,238)  arrow (0,0)-(72,32)
back    (0,0)-(80,80)     plaque (0,0)-(60,10)
```

All eight match `SHOP-SCREEN-032`'s measured sizes, and all 32 entries the
screen names resolved — five region pictures, four buttons, four arrows, three
backgrounds and fourteen plaques. The composed PNG was read: the room, the rack,
the table, the pack row and the button column stand where the decoded rectangles
put them, the item pictures draw, and the plaques carry their numbers. A
screenshot of a running window would be a stronger claim than this file makes.

## Gate

```
go build ./...                     ok
go vet ./...                       ok
gofmt -l $(git ls-files '*.go')    (nothing)
go test -trimpath -count=1 ./...   ok, 39 packages, exit 0
scripts/check-no-game-assets.sh    clean (tree scan)
scripts/check-doc-budget.sh docs/0157-shop   ok
scripts/check-hotfix-ledger.sh     ok
git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD   -> no trailer on any commit
```

No `SDD-Task:` trailer is carried: there is no `tasks.md`, the slice was
implemented in one lane context, and every `FR` and `DD` is accounted for above.

The gate above was re-run after both of master's landings during this round were
merged in — `3819cd3` (0159) and `c56d13e` (a pin-only bump). The submodule shows
no leading character at `618797e`, and every `SHOP-SCREEN-` claim this round
reads is still active at that pin. The one file 0159 and this round both touched
is `pkg/game/frontend.go`, in different parts of `FrontEnd`: 0159 replaced
`FinishMission`'s body, this round added the two shop-art fields. No test here
pins a world digest.
