# Mods add items

## Intent and authority

A mod adds items and edits the fields of existing items with files only: `data/items.toml`, names in `text/<lang>/strings.toml`, an optional PNG sprite, and one Starlark call that loads the data file. Authority is owner direction (`review/mod-milestone/MOD-LAYOUT.md`, built on `docs/1267/story.md`). The item identity stays the 16-bit code (material, class, shape, row), so an added item takes a free row of its class table; extended identity is not in scope. ROM1 differences are rows `DIV-1861` to `DIV-1866` in `docs/divergences/mods.md`. Without a mod, every hash, SAV byte, scenario and release witness is unchanged.

## As built

Data file (`pkg/mod/items.go`, `toml.go`). `[[item]]` tables and `[[change]]` tables; the parser gained table arrays and quoted keys. An item row has `key`, `name` (a text key), `slot` (`body`, `shield` and the other slot names), `defence`, `absorption`, `weight`, `price`, `sprite`, `stand-in`, `for` (`any`, `fighter`, `mage`) and `shop`. A change names a shipped item by its English text and sets any of the four numbers. Every refusal carries `mod`, file and line: unknown slot, unknown key, duplicate key, number out of range, missing text key, bad stand-in form, a sprite path outside the mod folder.

Loading (`pkg/modrt/data.go`). The script calls `game.data.add("data/items.toml")` in `init`; the name is `add` because `load` is a Starlark keyword (`DIV-1864`). Only that file is loadable, once, and not after `init`. Text comes from the language of the base (`ru` for a Russian install, else `en`; Russian falls back to English per key; `modrt.LanguageFor`). `modrt.Result.Items` carries the rows and changes in load order. `cmd/againrom` calls `FrontEnd.SetModItems` after `SetMods`, and `-check` prints `againrom: items added=N changed=M` only when a mod carries items.

Code space (`pkg/game/moditems.go`, `pkg/data/rowoverlay.go`, `moditemrow.go`). A row takes the lowest free row of its class table (armour: row 31, one row; shields: rows 10 to 31; weapons: none, a new weapon is refused). `data.RowOverlay` extends a table's `Collection` and mask table without changing the shipped one; `Table.Armors`, `Shields` and `Weapons` are replaced, so every reader sees the added rows. The item is the Iron material in the Common shape and its stated numbers are those of that reference item: defence column = 4 x defence, weight and price factors 1. After the overlay is built, `ArmorFromCode`, `ShieldFromCode` and `ItemPrice` must return the stated numbers, otherwise the launch is refused (`cannot state defence N`). A `stand-in` (`[Shape] [Material] <row name>`, language independent, matched against the shipped row) names the shipped item the original format sees; an omitted shape and material take the first combination the row's masks allow. The code space being full refuses naming the class and the mod.

Shop route. `shop = true` appends one unit of the item to the armour shelf after the random draw, without consuming a draw (`Shop.generate`, `modShopStock`). No other route (a starting inventory, weighted stock) exists (`DIV-1862`).

Sprite (`pkg/game/moditemsprite.go`, `pkg/vfs/fs.go`). A PNG of at most 80x80 pixels becomes a one-frame 256-colour `.16a` stream at start-up (alpha to the level in sixteenths; colours reduced to 256; nothing visible refused) and is served as the item's `inventory/<code>.16a` through `vfs.FS.Overlay`. Without a PNG the icon aliases the stand-in's. The worn figure layers always alias the stand-in's (`DIV-1866`).

Saves (`pkg/game/modmark.go`). The `AgainromMods` leaf gains an optional `items` array (`object`, `code`, `row`); the mark format stays 1. SAVE writes the stand-in code into `F40` and the stand-in row into `T0C` of each item object holding a mod code and lists the object with its true values; the other fields of the object keep the true numbers (`DIV-1863`). LOAD under the same mod set verifies each listed object still holds the stand-in and writes the true code and row back before the document is read. LOAD without the mod, or under another set, refuses by the rule of `docs/1267/story.md`. An unmodded save has no leaf.

Example mod (`pkg/modrt/testdata/mods/heavy-armor`, test data): a Gambeson (body armour, defence 6, absorption 1, weight 6, price 120, stand-in Soft Mail, `shop = true`, a generated 80x80 sprite, English and Russian names) and a change that sets Chain Mail's weight to 20. Its code is `0x071f`.

## Proof

- Unit: `pkg/mod` (rows, changes, every refusal with file and line, name length, quoted keys, text fallback, slot order), `pkg/modrt` (example mod in both languages, `LanguageFor`, refusals, other files, second loads, load order), `pkg/data` (`RowOverlay`), `pkg/game` (allocation, full code space, stand-in forms, changes, number refusals, shop stock, `Shop.generate` with mod stock, icon conversion and refusals, mark round trip on a document), `pkg/vfs` (overlay), `cmd/againrom` (`-check`, a refusal).
- Release, one root at a time (`pkg/game/moditems_release_test.go`, gated by `AGAINROM_ASSETS`): through chargen, mission 10, a mission SAVE in mission 20 and its cold LOAD under the mod, then the town, the Gambeson is on the armour shelf at 120 with the language's name and a one-unit stock, its icon is the PNG's, and the figure layers equal the stand-in's; it is bought, the purse falls by 120, the town SAVE holds the stand-in in the ordinary fields and the item in the leaf, a cold LOAD under the mod restores the item and the purse, wearing it works and survives a second SAVE and LOAD; in a mission the equipped Gambeson changes defence, absorption and weight by the stated numbers, and a mission SAVE and cold LOAD restore it; LOAD without the mod refuses naming `heavy-armor`; an unmodded town SAVE has no leaf; the Chain Mail change sets weight 20 and leaves defence and absorption as shipped.
- Screenshot (offscreen): `review/story1272-mod-items/`.

## Open debt

- No new weapons: the weapon table has 4 free rows but no decoded row layout a mod can fill.
- Only one armour row is free on shipped installs.
- No starting-inventory route.
- A data-only mod without `main.star` carries no items.
- The worn figure layer is the stand-in's, not mod-drawn; the `layer` key is rejected.
- The saved defence, weight and price blocks of a stand-in object hold the true values, not the stand-in's; whether the original reader accepts that is Unknown.
- Original SAV-corpus saves were not loaded under a mod.
