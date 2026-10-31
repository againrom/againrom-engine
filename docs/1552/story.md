# The second game's town shop stocks, sorts, prices and captions as the original

## Intent

The owner made six remarks from four shots of the second game's shop: the shelf order is wrong;
amulets and the like are never stocked without magic; a book's caption reads a number; a scroll's
caption names neither effect nor spell; a cloak's caption says it casts Lightning, where clothing
only raises attributes; prices do not fit their plates. Owner: «сравните с rom1, все по идее 1 в
1». The story applies the original's shop claims of knowledge snapshot k390 and, where a claim
leaves a question open, the first game's behaviour with a DIV row.

## Authority

- Owner, in the brief: each remark matches the claims or one DIV row states the remaining
  difference; where a claim leaves a question open, the first game's behaviour with a DIV row;
  the first game's shop does not change; the second game's shop is profile data of the one shop.
- Claims: `R2-ENGINE-709` (shelves and stock), `R2-ENGINE-710` (sort keys), `R2-ENGINE-711`
  (enchant), `R2-ENGINE-712` (caption), `R2-ENGINE-713` and `R2-ASSET-182` (price, font2,
  plates), `R2-ENGINE-714` (cell paint order), `R2-ENGINE-715` (EN and RU equal);
  `R2-ENGINE-264`, `R2-ENGINE-265`, `R2-ENGINE-267` (admission, fill, prices);
  `R2-SESSION-215` (observed first views). First game: `SHOP-154`, `SHOP-MAGIC-007`,
  `SHOP-144`, `SHOP-148`.

## As-built behaviour

### Kinds and builders

| Kind | Builder |
|---|---|
| item enchant | `shopstock.Enchant` (`pkg/shopstock/enchant.go`); the first game's rules in `pkg/game/shopenchant.go`, the second game's in `secondshop.Tables.enchant` |
| shelf order | `shopstock.SortBy` (`pkg/shopstock/sort.go`); the first game's keys in `shopSortShelf`, the second game's in `secondshop.Tables.Sort` |
| stock | `shopstock.Generate` with the second game's profile, `secondshop.Tables.Profile` |
| item caption | `itemInstanceInfoLines` (`pkg/game/iteminfo.go`), the code it names by `itemTemplateCode`, the form it describes by `shopController.described` |
| magic item decoder | `mapload.ItemInstanceFromCode` |
| price font | `shopController.ShopScreen`, `InstallResources.missionCardFont` |
| bag form | `secondshop` `unit`, `Unit`, `FromInstance` and `enchantForm` |

The first game's enchant moved unchanged into `shopstock.Enchant`; its wrappers keep their
signatures. Its stock, prices and shelf captions on EN and RU are byte-equal to the base over
six ceilings and forty seeds. Of the 4096 codes 0x0000..0x0fff decoded bare, 28 codes,
0x0e81..0x0e9c, which no first-game item row names, now read "Book" where they read a number.

### Stock

An enchanted category runs the one enchant with the second game's data: data.bin group B as
the effect table, the cast lists {1, 10, 11, 18, 26, 5, 16} and {18, 26}, the level field as
the capacity, a cast's level within the capacity left, each effect priced on its own, the cap
19,999,999. A weapon no fighter wields takes one cast; any other item takes one to three
effects by its part's weight column, and group B gives a castSpell weight only in the two
weapon columns, so no armour or shield casts a spell. The values the claims leave unread are
the first game's (DIV-4329). The item's effects are its bag form (DIV-3043).

### Shelf order

Each category list is sorted once when its draws end: the fighter bit first, the row, then the
price, for armour, shields and weapons; a magic item or a book by its price (DIV-4328). The
sort is stable.

### Caption

A book's caption is its school book's name joined to its spell, "Book of Fire Arrow", RU
"Книга с заклинанием Огненная стрела". A cell describes an item in its mission form, so a
scroll's and a potion's effect comes from its MagicItems string: "Scroll of Lightning" with the
spell's lines. An enchanted armour lists its attributes and casts nothing. The damage line form
and a scroll's stat line are the first game's (DIV-4330).

### Scrolls in a mission and older saves

The item decoder reads a scroll's MagicItems string as it reads a potion's, so a bought scroll
enters a mission with its castspell effect and casts there, and a new game's starting scroll
0x0e1c holds that effect in town and in the mission. Before, a scroll held no effect and could
not be cast. Saves written before this change hold their scrolls with no effect. The LOAD keeps
them so: such a scroll cannot be cast in the mission a mission file holds it in, nor in the next
mission after a town file's LOAD. At the next town's arrival a potion or scroll with no effects
comes home to the bag in its bag form, as before, and from the bag it enters every later mission
with its spell (DIV-4191).

### Price

A cell's price draws in font2, the generation screen's text font, on the plaque its digit count
chooses. In the first game that font is the tip font, so its shop does not change.

### Remark on amulets

The claims stock unenchanted armour rows of every admitted part: town 1's Armor category admits
material 15 caps, hats, capes, robes, gloves and shoes (`R2-ENGINE-264`, High), and rings and
amulets are armour rows (`R2-ENGINE-709`). The story follows the claims; whether the original
shows those rows plain is the owner's question.

## Proof

- Focused ordinary tests, each failing on the base and passing after:
  `TestSecondShopArmourAndShieldsCastNoSpell` (the base does not build it: it has no group B
  table), `TestSecondShopShelfSortsByTheClaimsKeys`, `TestSecondShopCellCaptionNamesTheSpell`,
  `TestSecondShopClothingCaptionListsItsAttribute`, `TestSecondShopPriceDrawsInTheTextFont`.
- Older saves, each failing on the first candidate and passing after:
  `TestKeptUnitTakesAnEffectlessScrollHome`, `TestSecondBagEffectlessScrollOfAMissionFileComesHome`
  (a mission file with a bought scroll of no effects: LOAD, victory, the scroll in the bag at
  town 2), `TestSecondBagEffectlessStartingScrollOfATownFileComesHome` (a town file whose hero
  pack holds 0x0e1c with no effects: LOAD keeps it, the next mission takes it as saved, the
  arrival puts it in the bag). The files are fixtures written by the engine, not install data.
- The first game's shop tests pass unchanged.
- `TestReleaseSecondShopStockMatchesMain` pins the new stock digest; the lane computed it on
  EN and RU with a probe, not with the release test.
- Headless frames of Plagat's shop at `-seed 1`, EN and RU, base and candidate, are lane
  evidence outside the repository.
- Release tests the next release runs on both roots: the second-game shop stock, page, caption,
  town shop and book icon tests, the bag's mission tests (`TestReleaseSecondBag*`), the
  original town LOAD tests (`TestReleaseSecondOriginalTownLoad*`), and the first game's shop
  shelf, price, plaque, quantity, unpriced, price line, enchantment, cast retry, order and
  starting staff tests.

## Open debt

- DIV-4328: the qsort's equal-key order, the magic shelf's order, and the client's display
  order, which `R2-SESSION-215`'s ascending first views suggest differs from the server keys.
- DIV-4329: the enchant's value helpers.
- DIV-4330: the damage line's colon and a scroll's stat line.
- DIV-4191: a scroll a save from before this change holds cannot be cast in the mission it is
  LOADed into, or the next mission after a town LOAD; the LOAD does not repair it.
- Kaarg's and the Druids' town shops are not framed; only town 1 is.
