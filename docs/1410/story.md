# The second game's shop page in town 1

## Intent

In the second game's first town a click on the shop opens the shop page
instead of nothing (DIV-2963). The player sees the stock by category, buys and
sells at the original's prices, and a SAVE after a deal keeps the bought
items, the gold and the shop's stock through a cold LOAD. Towns 2 and 3 are
not part of this story (seat ruling); DIV-2963 stays open for them. Base:
public main `6f9f622d`, merged with main at `b31f3518`; knowledge pin k260
from main.

## Authority

Second-game evidence only. Selectors and entry `R2-ENGINE-241`, `R2-ENGINE-249`;
the generic page, its stock, prices, item text and keeper `R2-ENGINE-263`
through `R2-ENGINE-270`; `R2-ENGINE-332` (data.bin columns); `R2-ASSET-069`,
`R2-ASSET-070` (`ShopFrame.256`); the SAV stock parameters `R2-SESSION-156`
(High) and `R2-SESSION-157` (current items across a LOAD Unknown). Choices
where a value is Unknown are rows DIV-3041 through DIV-3047 in
`docs/divergences/rom2-town.md`.

## As built

### Rules

`pkg/secondshop` is a leaf over a narrow `Collection` interface that the
second game's `databin` collections satisfy. `NewGameRecords` holds the
`Scenario.dll` ordinal-15 records (location 1 = town 1). `Candidates` decodes a
category mask and admits rows (`R2-ENGINE-264`); `Fill` draws a stock with the
`(rand()*n)>>15` index and the quantity form of `R2-ENGINE-265`, through the
random service's `ShopStock` stream. `Price` is
trunc(column 2 x material factor x class factor + 0.5) plus the enchant
effect price `trunc(50*S*(1+(S/70)^1.5))`, a magic item's column 0, a book's
spell column 21, capped at 19999999 (`R2-ENGINE-267`). `SaleCredit` is
floor(P*q/2+0.5) (`R2-ENGINE-266`); a cell offered for sale shows (P+1)/2.
`Buy`, `Sell`, `Take`, `Leave` and `Place` settle a deal and put a sold item
back on its category (`R2-ENGINE-333`). The price and level columns are
DIV-3041; the draw forms no claim states are DIV-3042.

### Session and SAV

`secondCampaign.shop` holds the four records, the shared stock, the location
it was drawn for, the hero's shop items and the open deal. Entry draws the
stock when category 0 is empty or the stock belongs to another location
(DIV-3045). The SAV's AgainromActions document carries it as
`currentSecondCampaign.Shop` (records, stock, location, hero items), written
from the current state with an open deal undone; a new game's shop is
omitted, and a document without it loads the new game's records and an
empty stock. The raw Scenario prefix bytes `0x1000..0x113f` are not written
(DIV-3044). Bought items stay in the campaign list as shop keys and do not
reach a mission (DIV-3043).

### Page

The page is a room of the one town composer: `rom2-town1.json` gives the shop
hotspot `click: {room: shop}` and a `shop` room with its scene (`ShopFrame.256`,
`ShopMain`, the shown category's shelf layer under a host condition
`shelf-0..3`, the keeper), its buttons (Undo, Buy, Sell, Exit) and three grids
(`stock`, `hero`, `info`). `pkg/town` gained `RoomSpec.Grids` and
`WhenSpec.If`; `pkg/game/secondtownshoppage.go` maps rows and paints the grids
as data. The first game's shop is unchanged. Presentation choices are
DIV-3046; the keeper stands still (DIV-3047).

Item names come from the shipped item-name table in the install's language:
`main.res` `text/itemname.txt` joined with `world.res` `data/itemname.bin`
(`R2-ENGINE-267`). The second game's file list names that key address, so
the definition table now carries the second game's item names for every
consumer. `secondshop.Tables.NameKey` is the chosen key form (DIV-3046). A
book's spell and an enchantment's spell come from data.bin.

## Proof

- `TestReleaseSecondShopTownOnePopulations`, `TestReleaseSecondShopTownOneFill`
  (`pkg/secondshop`, `rom2-en` and `rom2-ru`): the claim populations (57/13,
  44, 4 at 83/167/167/333, 160 enchant variants 649..1442, five books at 1000,
  23 phase-two rows); loss control slot 3 for slot 2. `TestSecondShopSettlement`:
  sale credit 125 against the client's 126, buy refusal, merge and placement.
- `TestReleaseSecondTownShopDealAndSave` (`pkg/game`, `rom2-en` and
  `rom2-ru`, in the chain's ROM2 Town group): the SHOP hotspot opens the page; all 133 non-book items town 1's
  record admits have a shipped name (swapping the part and class fields names
  2); buy, sell and a second deal change gold and items by the claimed amounts;
  SAVE, cold LOAD in a child process, shop, gold and the next deal equal the
  continuous run.
- Owner witness: `review/story1410-rom2-shop/witness/` (EN and RU frames, its
  `INDEX.md`).
- The first game's shop release tests run in the whole release set.

## Open debt

- Towns 2 and 3: the Kaarg page (fire, modulo-20 keeper, `R2-ENGINE-278`,
  `R2-ENGINE-279`, `R2-ENGINE-354`) and the druid page (category gate on bank
  slot 770, `R2-ENGINE-259`); DIV-2963 and DIV-2639.
- The keeper's idle, Yes and No episodes and the shop sounds (DIV-3047).
- The Scenario prefix bytes and the original's current stock across a LOAD
  (DIV-3044, `R2-SESSION-157` Unknown); campaign and timed refill (DIV-3045).
- Bought items in a mission: the second game's item code and effect encoding
  are unpublished and the engine has no second-game mission inventory
  (DIV-3043).
- Localized spell names for books and enchantments (`text/spell.txt`, DIV-3046).
