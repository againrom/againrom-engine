# Story 1415: one shop builder for both games

Status: done. The item name resolver, the second game's hero bag, the one deal core with each
game's rules as data, the one open-deal SAVE walk, mod stock on a second-game shelf, and one shop
page and controller for both games' shop rooms.

## Intent

Owner observation on architecture audit row 4: the two games' shops do not differ for the
player; only the stock may differ, and one party is in the shop. The owner wants one shop builder.
Owner direction on resume: first the second game's hero bag as a one-member party pack of the
engine's one item instance, then the builder, so the shop's customer side is one code, the
party's packs.

## As built

### Item display name resolver

`mapload.ItemDisplayName(table, code, saved, fallback)` names an item for the player in both
games: the item's saved name, else the install's shipped name for its code, else the caller's
composed name. `mapload.ShippedItemName` is the one reader of the shipped table. Routes on it: the
world's `itemName`, the party weapon restore, the map loader's current weapon, the map
inspection sack lines, record choices and item-code check, and the second shop page's item name.
`TestItemNameOneRoute` fails when another non-test file reads `.NameFor(`; the cheat factory's
name-to-code lookup is the one allowed reader.

### The second game's hero bag

The second campaign holds a party of one member, the hero (`pkg/game/secondparty.go`). His bag is
that member's `CarriedItems`, one `sim.ItemInstance` per unit. The shop deals with the bag: a buy
inserts units, an offer moves a stack's units into the sale, undo and an unpriced sale put them
back, the page lists the bag folded into stacks by the shop's merge rule.

The bag holds at most 8192 units (`maxSecondBagUnits`), the SAV producer's bound: party
validation, the campaign document and an older hero list refuse more. This is an engine choice
(DIV-3043): no claim gives the original bag's size; 8192 is twice the older list's 4096-entry
bound, so an older document with a 4097-unit stack still loads, and a full bag's campaign
document stays near 4 MB, under the 16 MiB SAV bound. The shop page holds the bound before gold
moves: a BUY whose units would pass it is refused like a short purse (the keeper's refusal and
`buy-short`), so an accepted trade never leaves a bag SAVE refuses.

| Part | Engine form | Claim | Settled |
|---|---|---|---|
| Bag | one member's pack of item instances; buy inserts the shop's item | `R2-ENGINE-402`, `R2-ENGINE-404` | yes |
| Armour ID | material<<12, part<<8, class<<5, row | `R2-ENGINE-327`, `R2-ENGINE-403` | yes |
| Weapon and shield ID | the armour form | none | no: DIV-3043 |
| Magic item ID | 0x0e00 plus row | none | no: DIV-3043 |
| Book | ID 0x0e80 plus spell; teachspell effect (tag 42) with the spell | effect `R2-ENGINE-414`; ID none | effect yes, ID no |
| Enchantment | castspell effect (tag 41), spell low half, level high half of the value | `R2-ENGINE-403`, `R2-ENGINE-414` | yes (level half Medium) |
| Type byte | armour 1, weapon 2, other kinds 0 | `R2-ENGINE-333` for 1 and 2 | shield, magic, book no |
| Reach a mission | the bag does not reach a mission | `R2-ENGINE-406` Medium, carry back Unknown | no: DIV-3043 |

Codes are lossless over the shipped tables: rom2-en armour rows 1..30 take parts 4..12, shields
part 2, weapons part 1 (one row has part -1 and no ID form), magic rows below 97, spells below 35.

Saved form: the campaign document writes the hero through `captureCurrentParty`, the party member
producer of the first game's current session document, as `Party` beside `Shop`; the shop writes
only its records. A document written with the shop's `Hero` list loads that list into the bag once
the install's tables are known (`arriveLoaded`, or the shop's entry); while it waits, a capture
writes it back unchanged. An open deal is written undone; the live deal stays open (DIV-3045).

### The one deal core

`pkg/game/shopdeal.go` holds the deal both games trade through: `shopDeal[T, S]` is four shelves
and the table, `shopPlace` one place (item, whether the customer owns it, the shelf it came off).
The first game's `Shop` and the second game's `secondShop` embed it; `ShopPlace` is the first
game's instance. Take, put, join, take back, undo, the totals, buy, sell, the emptied entries and
the SAVE walk are one code. What differs is a `shopRules` value, each game's data:

| Rule | First game | Second game | Claim |
|---|---|---|---|
| Table places | 5 | unbounded | SHOP-TRAY-024; R2-ENGINE-266 two lists |
| Buy beyond the gold | refused whole (client guard) | items before the first unaffordable one bought | SHOP-TRAY-026; DIV-3065 (R2-ENGINE-269 vs R2-ENGINE-266) |
| Unpriced item sold | stays on the table | returns to the hero | engine (DAT-DOC-021 sentinel); R2-ENGINE-266 |
| Emptied shelf entry | leaves at take | quantity-0 placeholder until leave | R2-ENGINE-266, R2-ENGINE-375 |
| Undone shop item | the shelf it came off | its record's placement | R2-ENGINE-266, R2-ENGINE-333 |
| Sold item's shelf | by class (`shopShelfForItem`) | its record's placement | SHOP-SELL-010; R2-ENGINE-333 |
| Shelf merge and order | equal item merges, shelf sorted | Insert: stackable or placeholder merges, else appended | SHOP-DUP-028, DIV-319; R2-ENGINE-266, DIV-3042 |
| Lot joins a place | equal item joins (no join rule) | equal and stackable joins | R2-ENGINE-265 |

Both games credit a sale ceil(q x P / 2) a place and show (P+1)/2 a cell (SHOP-SELL-010,
R2-ENGINE-266/267). A buy stopped at an unaffordable place keeps that place and every later one.

The page stages through the same join rule (`shopController.shopJoin`): a lot taken from the
shelf or offered from the bag joins a table place, and the bag view folds units into one cell,
only where the item is equal and the deal's `join` admits it. In the second game a book or an
enchanted item is never folded, so each unit is its own place and the stop-at-first buy holds
per unit (R2-ENGINE-266, DIV-3065). The first game's rules set no join, so its page folds equal
items as before.

The first game's city shop decides through the same core: buy, sale and clear run `shopDeal.buy`,
`sell` and `clearTable` with `Shop.rules()` on a copy of the session's shop
(`pkg/game/cityshopmutation.go`). The city object graph only carries the decided places to the
packs, shelves and owners and checks each moved item's identity against the decision; gold moves
by the core's spend and payout.

### Open-deal SAVE

One policy in both games: the SAV is written with the deal settled on the capture's copy, and the
live deal stays open (DIV-1443, DIV-3045). `shopDeal.settle` walks the table from its last place:
a shop place goes back on its shelf, a customer place to its owner. The first game's owner step
moves the item through the city graph to its owners' packs (`resolveShopSnapshot`); the second
game's puts the stack's units back in the hero's bag in the order they were offered. Neither SAV
writes the stock.

### Mod stock on a second-game shelf

`secondshop.Tables.Stocked` holds the mods' stocked armour and shields (`secondshop.Stocked` reads
a mod item's code in the bag's item form). Every fill files them by placement after its draws;
they take no draw, so the stock stream is unchanged. The first game's mod stock is unchanged.

### One shop page for both games

`ui.ShopScreen` with its `ui.ShopLayout` is the one page; `shopController` (`pkg/game/shopcontroller.go`)
is the one controller, embedded by the first game's `townScreen` and the second game's
`secondCampaignScreen`. Each binds data and hooks, not methods: the deal and party selectors (nil
is the first game's session shop and party), the racks, the item value and label, and the room
hooks (shown, sound, scene and keeper event names, dress, leave, chose). A selector deal stays
outside the first game's city object graph.

The second game's shop room (`pkg/game/secondshoproom.go`) is the town screen's `ui.TownShopScreen`
while it shows; `AtTownSquare` is false then, and the room's scene runs through the shop interior
seam. Its pictures are the town description's shop room page art under the first game's entry
names (`R2-ENGINE-249`, per-town prefix for Kaarg and the druids); its scene draws the category
highlights, the keeper and the Kaarg fire as the page's interior. Category and keeper rectangles
are the room's `shelf/k` and `merchant` buttons: town 1 and Kaarg from `R2-ENGINE-354`, the druids
from DIV-3064. A category whose button condition fails (the druids' elven category while bank slot
770 is zero, `R2-ENGINE-259`) has no rectangle and cannot be chosen. The chosen rack writes the
deal's category, so the highlight follows it. The category is not saved: a LOAD restocks the
shop, and `restock` resets the category to the first one (DIV-3045). The grid page,
its rows, `town.GridSpec` and the town vocabulary's lists retired. DIV-3046 states the page and
names the missing hero panel.

The second game's table is unbounded, so its layout sets `ui.ShopLayout.TableScroll`: the wheel
over the table strip turns it (`ui.ShopWheelTable`) like the shelf and bag strips, the view's
`TableOffset` shifts the drag origin, the item stars and the tooltip key, and a newly staged place
turns the strip to show the last place. Every place stays reachable: a sixth place can be pointed
at, read and returned by its cell (DIV-3046). The first game's five-place table does not scroll.

### Stock items without a bag form

`TestReleaseSecondShopItemsWithoutBagForm` counts the candidates of every category of the four
new-game assortment records and of the druid record's fourth category with class 4, with both
picks of the magic phase-two rows: 219 distinct items on `rom2-en` and on `rom2-ru`, none without
a bag form (`secondshop.Tables.Instance`). A shelf therefore leaves out no item these records
admit. Records written by a later campaign event other than the druid conversation were not
counted.

## Proof

- `pkg/secondshop/bag_test.go` `TestBagUnitsRoundTrip`: each kind codes and reads back; a part -1
  row has no form; loss control: a unit with an object identity is refused.
- `pkg/game/secondtownshop_test.go`: deal, SAVE round trip with the hero through the party
  producer and malformed bag refusals, older hero list into the bag.
- Release, `rom2-en` and `rom2-ru`: `-run TestReleaseSecondTown` 38 tests pass in each, among
  them the town 1 deal, SAVE with a deal open, cold LOAD in a separate process and next purchase,
  and the same in towns 2 and 3.

- `pkg/game/shopdeal_test.go` `TestShopDealOneCoreBothGames`: the same core calls give five places
  and a whole-basket refusal under the first game's rules, and a stop-at-first buy, placeholders,
  an unpriced return and the leave drop under the second's; loss control: the second deal under
  the whole-basket guard refuses.
- `TestSecondShopModStockReachesShelf`: a fixture mod's stocked armour is on a shelf after a fill
  and is bought into the bag; loss control: the mod's unstocked item is on no shelf.
- Release on the deal core: `-run TestReleaseSecondTown` 38 pass on `rom2-en` and `rom2-ru`;
  first game `-run 'TestRelease.*(Shop|City|Sale|Buy|Sell)'` 92 pass on EN and RU
  (`TestReleaseSecondTownShopDealAndSave` skips there: no second-game root).

- `pkg/game/secondshoproom_test.go` `TestSecondShopPageTradesThroughTheOneController`: rack,
  one unit, whole lot, refused buy, undo, buy, offer and sale through the controller.
- Release, `rom2-en` and `rom2-ru`, towns 1, 2 and 3 on the one page:
  `TestReleaseSecondTownShopDealAndSave` and `TestReleaseSecondTownShopLaterTowns` (frames, buy,
  sell, SAVE with a deal open, cold LOAD in a separate process, next purchase),
  `TestReleaseSecondShopPageInput` (order, names, prices, tooltip, one unit and shift for the
  whole entry, UNDO, drags shelf to table and bag to table, BUY at P, SELL at (P+1)/2),
  `TestReleaseSecondTownShopKeepers` (the keeper's reaction to a category, a buy, a refusal and a
  sale, the idle episodes, the Kaarg fire), the druid gate, `TestReleaseSecondTownDruidStaleStockSave`,
  `TestReleaseSecondTownKaargDepartureStale` and `TestReleaseSecondTownRequestsTheSameSounds`.
  The page drives by pointer through `HeadlessShopPoint`. Frames were inspected on both roots.

- Review fixes, `pkg/game/secondshoppage_test.go`: `TestSecondShopPurchaseStopsAtTheBagBound`
  (a BUY past 8192 units is refused with gold, bag and table kept; loss control: without the
  page's bound check the bag passes 8192), `TestSecondShopOlderLargeStackLoads` (an older 4097-unit
  stack loads, is adopted and saves; 8193 is refused; loss control: the older list bounded at 4096
  units refuses the 4097 stack), `TestSecondShopPageKeepsUnstackableLotsApart` (two enchanted
  items and two books stage as separate places and a purse for one buys one; loss control: the
  join rule ignored folds them), `TestSecondShopPageTableScrolls` (six places, each cell checked
  against its place at each wheel turn, the sixth returned by its cell; loss control: the view
  filled from place 0 fails).
- Release on `rom2-en` and `rom2-ru`: `TestReleaseSecondTownShopFullBagRefusesPurchase`,
  `TestReleaseSecondTownShopBookPartialBuy`, `TestReleaseSecondTownShopTableScrolls` (wheel by
  pointer, cell names and prices per place, the sixth's hover lines, its return, BUY of five).
- First game's city decisions through the core: with `shopDeal.buy` disabled by an overlay,
  `TestCityShopBuySellAndClearPreserveDistinctRootQuantities`, `TestCityPurchaseJoinsTheRestoredItem`
  and `TestCurrentSaveDialogCityPurchaseTrainingAndColdContinuation` fail (they pass under the same
  overlay with the earlier city code); `sell` and `clearTable` disabled fail the city sale and clear
  tests. First-game equality with main: frames, SAV and export bytes equal on EN and RU.

## Open debt

- The hero panel (figure, doll, statistics) is not drawn on the second game's page (DIV-3046).
- The second game's tooltip lines are the engine's item lines for the bag code; no claim was read
  for the original's item information panel.
- DIV-3043 stays open, narrowed to the unpublished IDs and type bytes and the mission carry.
