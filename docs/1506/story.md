# One shop item and one shelf insert for both games' shops

## Intent

Both games hold a shelf entry and a deal place in one record, `shopitem.Item`
(`pkg/shopitem/item.go`); `game.ShopItem` and `secondshop.Item` are aliases of it. Every merge of an item onto a shop shelf in both games goes through one insert,
`shopstock.Insert` (`pkg/shopstock/shelf.go`), under a stack-key rule each game states as data:
the stock's draws, the multiplier's further rolls, the mods' stocked items, an item sold to the
shop, an item put back from the table, and a restock. Behaviour is unchanged in both games: the
same shelves item by item, quantity and order, the same prices and the same draws from the random
stream.

Tests change only where they spell the type, build a literal, or read a field that moved or was
exported (seat ruling after the owner's decision: one shop item and one rule). No asserted value,
count, order, price, byte, seed or stream position changed; no test was deleted or skipped.

## Authority

- Owner decision 1 on audit `pipeline/reviews/arch-audit-027ef24e.md`, rank 12 (shop item record
  and shelf merge): one shop item and one shelf insert.
- Owner ruling that the first game is a reduced second game; the seat rule "One builder per
  kind". Refactor only; no ROM fact moves. The rules keep their own citations: first game
  SHOP-DUP-028, `DIV-319` (order), `DIV-322` (stacking); second game R2-ENGINE-265, R2-ENGINE-266.

## As built

### The record

`pkg/shopitem/item.go` imports only `pkg/data` and `pkg/sim`; `pkg/game` and `pkg/secondshop`
both import it (`internal/archtest/dag.go`).

| Fields | Who sets them | Document |
|---|---|---|
| `ObjectID`, `Code`, `Kind`, `Effects`, `Price` (unit), `Count`, `Weight`, `WeightPresent`, `SourceEquipment` | both games: the item as the bag holds it | written as before by the first game's shop document |
| `CityID`, `CityOrigins`, `Owners` | first game: city identity and the pack slots a place came from (were unexported `cityID`, `cityOrigins`, `owners`) | `json:"-"`; saved by the shop's own SAV records as before |
| `Table`, `Row`, `Class`, `Material`, `Spell`, `Level` | second game: table identity (`Table` was `Kind`) | `json:"-"` |

`Item.Instance` and `Item.Clone` are the former `ShopItem` methods. `shopitem.Owner` is the former
`shopOwnerQuantity`, `shopitem.Table` the former `secondshop.Kind`; both old names are aliases.
The second game's quantity is `Count` (was `Quantity int`) and its unit price `Price int32` (was
`int64`; `Tables.Price` is at most 19999999). The bag unit's price is `min(Price, priceCap)` as
before. `secondshop.Item.Same` had no caller and is deleted; `Stackable` became the package's
`stackable`. The older hero list document keeps its own record `secondHeroItem`
(`pkg/game/secondtownshop.go`) with the old field names, so that document reads and writes as
before.

### The insert

`pkg/shopstock/shelf.go`:

- `Stack[I, K]`: `Key` (an item joins only an entry of an equal key), `Same` (refines the key;
  nil: the key decides), `Takes` (whether the entry takes the item's units; nil: always), `Add`
  (the entry with the units added), `Copy` (the entry an unjoined item opens; nil: the item),
  `Order` (files the shelf after an entry opened; nil: append order).
- `Merges`, `Find`: the first entry an item joins.
- `Insert(list, stack, items...)`: each item joins the first entry it merges with, else opens one
  at the end; `Order` runs once when any entry opened. The key index keeps a shelf of the
  multiplier at 1000 linear.

### The two rules

| Field | First game `shopShelfStack(t)` (`pkg/game/shopstockprofile.go`) | Second game `secondshop.Stacking` (`pkg/secondshop/fill.go`) |
|---|---|---|
| Key | city node, saved node, code, unit price | the item but its count and price: table, row, class, material, spell, level |
| Same | `shopItemEqual`: the key's fields and `sim.ItemEqual`, effect list included | key only |
| Takes | every equal entry | a stackable item, or any item onto a count-0 placeholder |
| Add | Count += | Count += |
| Copy | `ShopItem.Clone` | the item |
| Order | `shopSortShelf` by the table: class, kind, price (`DIV-319`) | none: append order |

A city-bound return (`cityShelfStack`, `pkg/game/cityshopmutation.go`) is the first game's rule
with the join `cityShelfJoins` (price aside, another city node): `Find` names the cell, the caller
merges it through the city object graph, and an item that joins none opens through `Insert`.

The second game's `put` (`pkg/game/secondtownshop.go`) keeps a shelf entry only when its bag unit
reads back through `secondshop.FromInstance` to a table item that has a bag unit again; any other
entry leaves the shelf at that `put` (a return, a sale or an undo filed on that shelf), as at base,
where `put` rebuilt the shelf through that read-back. The other entries keep their order and
counts, and the incoming item is then filed by `Insert`. `TestSecondShopPutDropsAnEntryWithNoReadBack`
pins it against the base formula.

### Sites before and after

| Before (base `cc247360`) | Present rule | After |
|---|---|---|
| `pkg/game/shop.go:123` `ShopItem`, `:143` `Instance`, `:148` `Clone` | the first game's item | `shop.go:124` alias of `shopitem.Item`; methods in `pkg/shopitem/item.go` |
| `pkg/game/shopsave.go:11` `shopOwnerQuantity` | a place's pack slot | `shopsave.go:12` alias of `shopitem.Owner` |
| `pkg/secondshop/shop.go:86` `Kind`, `:97` `Item`, `:112` `Same`, `:118` `Stackable` | the second game's item | `shop.go:88` alias of `shopitem.Table`, `:100` alias of `shopitem.Item`; `Same` deleted; `:104` `stackable` |
| `pkg/game/shopsort.go:162` `shopStackShelf`, `:183` `shopStacker`, `:190` `newShopStacker`, `:198` `add` | fold into the first `shopItemEqual` element by key, else append a clone; no order | `shopStackShelf` is `Insert` under `shopShelfStack(nil)`; `shopStacker` and `newShopStacker` deleted |
| `pkg/game/shopstockprofile.go:60` `order` (each shelf's `Finish`, books `:102`, mods' `Place` `:108`) | stack, then sort | `Insert` under `shopShelfStack(t)` |
| `pkg/game/shopstockprofile.go:117` multiplier `Join` | re-stack the shelf, add the further items, sort | the same `order` over the shelf and the further items |
| `pkg/game/shop.go:737` deal `put` | a count of 0 is refused; fold into the first equal entry without a sort; else append a clone and sort | the count-0 refusal stays in `put`; then `Insert` under `shopShelfStack(s.tbl)` |
| `pkg/game/cityshopmutation.go:527` `cityShopReturnToShelf` | first cell of another node `cityShelfJoins` admits is bound and merged through the graph; else append a clone and sort | `Find` under `cityShelfStack`; the merge stays; else `Insert` |
| `pkg/secondshop/fill.go:149` `Insert` | first `Same` entry taking a stackable item or a quantity-0 placeholder; else append | deleted; `Stacking` and `insert` |
| `pkg/secondshop/fill.go:62,68` category insert, stocked items | `Insert` | `insert` |
| `pkg/secondshop/deal.go:28` `Sell` | `Insert` | `insert` |
| `pkg/secondshop/bag.go:124` `stacks` | its own `Same && Stackable` scan for the stack index, then `Insert` | `Find` and `insert` |
| `pkg/game/secondtownshop.go:62` deal `put` | `secondshop.Insert` over the converted shelf | `shopstock.Insert` under `secondshop.Stacking` into a clone of the shelf itself, less the entries with no read-back; `secondShopItem` (`:88`) keeps the table identity beside the bag form, so the shelf is not converted |
| `pkg/game/secondtownshop.go:65` table `join` | `Same && Stackable(incoming)` | `secondshop.Stacking.Merges` (a table place never holds 0 units, so the placeholder clause never applies) |

## Rules outside the insert

- `shopDeal.settle` (`pkg/game/shopdeal.go`) appends a shop place to the shelf it came off with no
  merge: the SAVE policy, not an insert. Unchanged.
- The table joins (`shopFindMinePlace`, first game, code and price; the second game's `join`) join
  places on the table, not shelf entries.
- The city-bound merge writes the city object graph and can fail; it stays the caller's.

## Proof

Instruments are lane test files kept in the notes directory `review/story1506-shop-item/dump`,
copied into the candidate and into a `git archive` of base `cc247360`; they are not committed.

- Stock and deal dump, 200 seeds per root. First game: 14 shelf ceilings, multiplier 1, 2, 10 and
  1000 on every seed (11200 stocks), 5600 deal sequences of 60 steps; 40428 whole-entry takes.
  Second game: 1400 stocks (new game, new game with mods, late towns; every shop location), each
  with a 60-step deal; 2952 quantity-0 entries. Each line carries the shelf item by item, the
  prices, the stream position and the next draw. Candidate and base are byte-identical on every
  root: `en` and `ru` 4512925 lines (sha256 `345237f7…`), `rom2-en` and `rom2-ru` 586652 lines
  (sha256 `d47c439f…`).
- Town SAV after a scripted buy, sell, take and clear: byte-identical to base, with the shelf
  digests after every step and after a cold reload. First game 12118 bytes (`en` `82ea581f…`,
  `ru` `37298f60…`); second game `rom2-en` 14689 bytes (`6c03d935…`), `rom2-ru` 14691 bytes
  (`b3637f31…`).
- Loss controls, each a one-line change to one rule, then restored from the commit. Second game,
  placeholder clause dropped: `TestStackingPlaceholder` fails. Second game, `Takes` always true:
  `TestStackingPlaceholder`, `TestSecondShopPageKeepsUnstackableLotsApart`,
  `TestReleaseSecondTownShopBookPartialBuy` and `TestReleaseSecondShopStockMatchesMain` fail;
  first game passes. First game, `Same` nil: `TestReleaseShopStockMultiplierDefaultMatchesMain`,
  three book-shelf tests, `TestShopCellsKeepDistinctEffectsAndCloneAcrossEveryModelTransfer` and
  `TestReturnedItemLandsInOrder` fail; second game passes.
- The second game's shop release tests (21) pass on `rom2-en` and `rom2-ru`.
- On the record commit `4ec2cd55` the dumps (same 200 seeds per root, same line counts and
  sha256) and the four SAV witnesses are again byte-identical to base, and the three loss controls
  fail the same tests. The dump instrument's copy for the candidate differs from base's only in
  field spelling and in printing the second game's stock view in the text `%+v` gave the old
  record (`review/story1506-shop-item/dump/mkcand.py`).
- Test files touched by the record: 17 (13 in `pkg/game`, 4 in `pkg/secondshop`). Beyond type
  spelling, literals, renamed fields and integer conversions: `bag_test.go` compares a record with
  `reflect.DeepEqual` (the record holds slices and is not comparable);
  `secondshoppage_release_test.go` keys a seen-set by the record's `%+v` text for the same reason;
  `stockmain_test.go` prints the stock in the old record's `%+v` text, so its digest constant
  stays; `secondtownshop_test.go` builds the older hero list document through a helper of
  `secondHeroItem`.
- Gates on `e9111c8d` (base merged with `origin/main`): `gofmt -l`, `git diff --check`, `go vet`
  of the touched packages, `go test -trimpath -count=1 ./...` (83 packages ok),
  `scripts/check-no-game-assets.sh` and `pipeline/check-div-claims.sh` pass. The headless release
  set fails only on a `.map-pages-*` rename refused by Windows ("Access is denied"): `en`
  `TestReleaseMapEditorArmsOpenAndKeepBytes` and `TestReleaseMapEditorParameterCurrentPicksPreserveBytes`;
  `ru` `TestReleaseEditorCancelPreservesInstalledMapBytes`,
  `TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes` and
  `TestReleaseScriptSpellCurrentPicksPreserveEveryMap`. Rerun alone once, four pass;
  `TestReleaseMapEditorArmsOpenAndKeepBytes` fails again on the same rename, on another map. The
  map editor's save path is outside this change. The one skip is `TestReleaseActivePauseRenderedWorld`.
- Gates on `d4f49284` (the record and this document; `origin/main` `d7c4e9f9` already merged):
  the same gates pass, `go vet` adds `pkg/shopitem` and `internal/archtest`, 83 packages ok. The
  release set fails only on the same `.map-pages-*` rename: `en`
  `TestReleaseMapEditorAreasUntouchedMapsSaveTheirBytes`, `ru`
  `TestReleaseMapEditorArmsOpenAndKeepBytes`. Rerun alone once, the first passes; the second fails
  again on the same rename, on another map. The one skip is `TestReleaseActivePauseRenderedWorld`.
- Read-back drop in the second game's `put` (`8d99b7a9`). Test collections code armour row 3 as
  the shield part (no read-back) and row 4 as part 6; a shelf of row 3 x1 and row 4 x2 gives the
  base formula's shelf after one `put`, after `take(1)` and after the undo. Loss control, the drop
  removed: `TestSecondShopPutDropsAnEntryWithNoReadBack` fails, no other `TestSecondShop` test.
  `TestStackingFirstOfTwoPlaceholders` pins the merge into the first of two count-0 entries of one
  key; with `Insert` merging into the last match it fails and `pkg/shopstock` passes.
- Gates on `8d99b7a9`: `gofmt -l`, `git diff --check`, `go vet` and the ordinary `go test` of
  `shopitem`, `shopstock`, `secondshop`, `game`, `storyguard`, `archtest`, and
  `scripts/check-no-game-assets.sh` pass. Release `-run 'Shop|Stack|City'`: `en` ok (451 pass,
  8 second-game skips). On `rom2-en` the same pattern fails 52 first-game tests that need the first
  game's root, the same 52 by name as base `cc247360` run the same way; the second game's shop,
  stack and town tests (`-run 'Second.*(Shop|Stack|Town)|Stacking|TestInsert'`) pass but for two
  that need the first game's root (`TestReleaseSecondTownSaveLossControls`,
  `TestReleaseTwoTownSaveLoadsIntoTheSecondTown`).
- Not run: the second game's save-test groups through `pipeline/check-rom2-save-tests.sh`; the lane
  copy that drops its chain-ownership check was refused by a safety rule.

## Open debt

- The second game's bag codec stays in `pkg/game/secondtownshop.go`: `secondShopItem`,
  `secondShopItems` (drop an item with no bag unit), `secondItem`, `secondItems` and `stockOf`.
  They are not identities: a place the shared shop page makes from a pack unit
  (`shopFromPack`) carries only the bag form and owners, so `join`, `place`, the bag write and
  the stock views read the table identity back from the bag unit; and the views must stay in
  table form because tests compare them with a fresh `secondshop.Fill`. Making the shared page
  fill the table identity would remove `secondItem`; not done here.
- The older hero list document keeps its own record `secondHeroItem`; folding it into
  `shopitem.Item` would change that document's field names.
- A table row whose part, price or level makes its bag unit unreadable (for example an armour row
  coded as the shield part) is still drawn onto the shelf; it cannot be bought and leaves at the
  next `put` on that shelf. Whether the generator should refuse it is a separate question. Shipped
  `rom2-en` tables draw no such entry over records 0..3 and seeds 1..200 (review of `685746b8`).
