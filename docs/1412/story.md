# The second game's shops in towns 2 and 3 and the shop keepers

## Intent

In the second game a click on the shop of town 2 (Kaarg) and town 3 (the
druids) opens the shop page of town 1 with the town's own data: its art, its
assortment records, its keeper and, in Kaarg, the fire. The druid page offers
its fourth category only once bank slot 770 is set. The keeper of every shop
page animates. SAVE and cold LOAD keep the items, the gold and the stock in
towns 2 and 3. Base: public main `145d37d2`, reconciled with main `8013ec03`
(knowledge pin k262). The first
game is unchanged; town 1's deal is unchanged.

## Authority

Second-game evidence only. The generic keeper `R2-ENGINE-268` and its
triggers `R2-ENGINE-269`; the Kaarg centre and fire `R2-ENGINE-278`, its
keeper `R2-ENGINE-279`, its category rectangles `R2-ENGINE-354`; the druid
page and gate `R2-ENGINE-259`; the art `R2-ASSET-065`; the shop records of
locations 2 and 3 `R2-ENGINE-263`, `R2-ENGINE-330`; class 4 from the druid inn
`R2-ENGINE-331`, `R2-ENGINE-367`. Choices where a value is Unknown are rows
DIV-3061 through DIV-3064 in `docs/divergences/rom2-town.md`; DIV-3065 is
fidelity debt.

## As built

### One page, three towns

Each town's JSON (`pkg/game/towns/rom2-town{1,2,3}.json`) gives its shop
hotspot `click: {room: shop}` and a `shop` room of the one town composer. The
page code in `pkg/game/secondtownshoppage.go` reads rows, grids and buttons as
data and names no town. Kaarg's room places the centre at (164,0): its frame,
`ShopMain`, four highlights under `shelf-0..3`, four category rectangles as
shelf buttons, the keeper and the fire. The druid room draws `elven` while
`bank-770-set` holds and gives the fourth shelf button `"if": "bank-770-set"`;
`pkg/town` `ButtonSpec.If` hides a button, its row and its click while the
condition is false, and the page refuses to choose a hidden category.

### Keeper and fire

The keeper is the existing `priority` program, extended by data:
`idle-pick` draws which state the wait raises (Kaarg and druid: R(2) of a1 or
a2), `end: "all"` lowers every raised state at an episode end, `keep-index`
keeps the counter in the process across exit and entry, and the `cue` phase
plays a state's frame sound at its index (Kaarg: `Kman4.wav` at counter 1).
The wait is the new pick form `modulo`, 5000+1000*(rand()%5) ms, drawn on
every 100 ms step from the random service's shop-interior stream (`idle`
source). Page events `category-changed`, `bought`, `refused` and `sold` start
Yes, Yes, No and Yes in town 1 and a2 on `bought` and `sold` in towns 2 and 3.
The Kaarg fire is the new `transit` program: lite, burn and cicle concatenated
as cells 0..17 with loops [0,5] and [12,17]; choosing weapons runs it up to
the cicle loop and draws the selected cells, another category runs it back.
Clock and draw choices are DIV-3061; the Kaarg page DIV-3062; the triggers
DIV-3063; the druid page DIV-3064.

### Records and SAV

The campaign's auxiliary array was the shop records of locations 2 and 3
(`R2-ENGINE-330`); it is now one store, `secondCampaign.shop.records`.
Every published departure writes minima, draws, bounds and the category
masks; stages write the maxima; the druid inn's NPC 675 topic 78 ORs class 4
into record 3's category 3. A write that changes the record of the
location the stock was drawn for (the inn talk, a departure, a stage) sets the
stock's location to 0, which is no shop, so the next entry draws again with the
shop stock stream; a write that leaves the record equal changes nothing. The
SAV's campaign shop carries the records, the
stock, its location and the hero's items; the array is no longer written. A
document this engine wrote with the array rebuilds the records from its
bank's departures and stage. DIV-2963, DIV-3047, DIV-2437 and DIV-2630 close.

## Proof

- `pkg/town`: `TestPagePriorityPicksKeepsItsIndexAndEndsAll`,
  `TestPagePriorityIdlePickLossControl`,
  `TestPageTransitLoopsAndRunsBetweenLoops`.
- `TestReleaseSecondTownShopLaterTowns` (`rom2-en`, `rom2-ru`): Kaarg at stage
  30 and druid at stage 60 open the shop, stock inside each record's range, a
  buy and a sale at the claimed prices, SAVE with a deal open, the SAV's shop
  equal to the captured shop with a loss control, and a separate process that
  LOADs it and makes the same next deal. The druid subtest runs the gate: no
  fourth row, button, click or `elven` with slot 770 zero; after the inn talk,
  slot 770 is 2, the class-4 mask is set and the category is chosen. With slot
  770 = 1 the page offers the fourth category before the talk with nothing in
  it; after the talk the re-entered stock holds class-4 goods, and the same
  stock kept as drawn (the loss control) still holds none.
- `TestReleaseSecondTownDruidStaleStockSave` (`rom2-en`, `rom2-ru`): a SAV
  between the talk and the next entry keeps the stock stale; a separate
  process LOADs it, enters the shop and finds class-4 goods.
- `TestReleaseSecondTownKaargDepartureStale` (`rom2-en`): a Kaarg stock drawn
  at stage 30 is drawn again at the next entry after the mission 30
  departure; a departure that writes no record keeps it.
- `TestReleaseSecondTownShopKeepers` (`rom2-en`, `rom2-ru`): town 1 Yes to
  counter 12, No on a buy beyond the gold, Yes on a sale, idle Pose2-3 after
  5000 ms ending at counter 28; Kaarg and druid a1 or a2 as the pick draws, a2
  on a buy, nothing on a refusal; the Kaarg fire to its cicle loop and back.
  Each frame is probed where it differs from the resting page.
- `TestReleaseSecondTownKaargMaskCensus` and `TestReleaseSecondTownDruidMaskCensus`
  (`rom2-en`, `rom2-ru`): mask byte 144 reaches the shop hotspot, tip 233 and
  the door control that opens the page (Kaarg door 6, druid door 4); main's
  inactive shop byte fails them.
- `TestSecondInnTalkEffects` checks the class-4 mask;
  `TestCurrentSecondLegacyAuxiliaryArrayRebuildsRecords` the legacy array.

- Whole runs on the merge with main `8013ec03`: `go test -trimpath -count=1 ./...`
  passes; `check-release-tests.sh` on EN and RU passes (1008 of 1100 ran on each
  root, none failed) on the merge commit; after the two mask census fixes the
  ROM2 subject groups pass 89 declared subjects on
  `rom2-en` and `rom2-ru` with no skip or fail.

## Open debt

- A buy beyond the gold buys the affordable prefix; the original refuses the
  whole buy (DIV-3065).
- The generic and druid keepers' shop sounds are not claimed; the druid
  keeper's triggers and the Kaarg category callees are unread (DIV-3063).
- The Kaarg and druid highlight files and the druid rectangles are chosen
  (DIV-3062, DIV-3064).
- The Scenario prefix bytes and the current stock across the original's LOAD
  (DIV-3044); bought items in a mission (DIV-3043).
