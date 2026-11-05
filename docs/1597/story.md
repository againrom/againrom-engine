# One stock reader of mod items for both shops

## Intent

Both games' shops place the mods' stocked items through one reader of the
merged mod item rows. Each game's stock policy is profile data. The clothing
layer lookup reads the same entry form. No player change: every shelf holds the
same mod items at the same prices, units and places, a layer is worn and drawn
as before, and hashed state and SAV bytes do not move.

## Authority

Owner direction: one builder per kind, and the multi-way allowance falls from
50 to 20 (audit-20 plan, rank R9, wave 5). Owner ruling: the first game is a
reduced second game; architecture work counts 0 toward the release. Owner
decision held open: mod weapons stay out of the second game's shop. DIV-1862
keeps the first game's restock chance. No claim changes; no DIV row is added.

## As-built behaviour

- `modStock` (`pkg/game/moditems.go`) reads `Table.Mods.Items` once per
  stocking. It skips an item without `Stock`. It classifies the code once by
  `secondshop.Stocked`, which now answers the code's table as its category for
  any table. It keeps the item when the policy carries that category and the
  stocking offers it, in the order the mods added the items. Each kept entry
  carries the code, the unit price and the second game's shop item.
- `modStockPolicy` is the profile. `firstModStock(seed, restocks, pool)` holds
  the first game's data: every category, shelf `ShelfArmour`, all items at town
  arrival, each item at a restock with the chance of a row of the shipped plain
  armour pool (`modStockOffered`, unchanged). `secondModStock` holds the second
  game's data: categories `{Armor, Shield}` at every fill.
- The first game's stock profile `Place` calls `modStock` with
  `firstModStock`, turns each entry into one armour-kind unit (`shelfItem`)
  and orders the shelf as before. `secondShopTables` fills `Tables.Stocked`
  from `modStock` with `secondModStock`; `secondshop`'s fill places those by
  record as before.
- The clothing layer lookup stays `mapload.LayerItem`, now a call of
  `mapload.LayerItemIn` over the same `[]mapload.ModItem` list. The mod mark's
  layer check (`applyModLayers`), which looped the list itself, calls
  `LayerItemIn`. The five `LayerItem` callers are unchanged.
- `internal/archtest/modstock.go` names every production function outside the
  named readers (the stock reader, the writer `setModItems`, the carry
  `SetMods`, `LayerItem`) that reads `X.Mods.Items`; every unnamed read of the
  mod context's `Items` (three save-path entries are named); every read of
  `Stock` or `Price` on a value bound to a mod item; every call of
  `secondshop.Stocked` outside the reader; every caller of the reader other
  than the two shops; and a reader count other than one.

## Proof

- `TestModStockRecordBothShops` (`pkg/game/modstockrecord_test.go`) was
  committed alone before the change and holds after it. With synthetic
  fixtures it records each game's shelves of a stocked armour, shield and
  weapon, an unstocked armour and a stocked layer at town arrival and after one,
  two and five restocks, and the layer lookup's answer for each code. Dropping
  the shield from the second game's set fails it.
- Focused tests: `./pkg/game -run
  'ModItem|ModShop|ModStock|ModLayer|Layer|ShopStock|SecondShop|SecondTownShop|Restock|ModMark'
  -skip '^TestRelease'`, `./pkg/secondshop -skip '^TestRelease'`,
  `./pkg/mapload -run 'Layer|ModItem'` (no tests match), `./internal/archtest
  -run 'ModStock|^TestLiveTreeClean$'`, `./internal/storyguard -run
  '^TestLiveTreeClean$'`.
- `TestCheckModStockNamesEachShape` plants one instance per shape.

## Open debt

- No release test ran. The next release runs, on both games, the mod item and
  shop release tests named in the lane result.
- Letting mod weapons into the second game's shop is an owner decision.
