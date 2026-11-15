# Items are made and changed in the mod window

## Result

The mod window has an Items page. A modder adds an armour row by its key and a weapon row from its picture,
changes the numbers of a shipped item by its name, switches a shipped or added row off, and allows or forbids a
material pair. A bad value is refused on the page in the loader's own words. Every change is undone, redone and
saved with the map's other edits. The new item is offered to the party, sold in a first-game shop and, for a mod
made for both games, in a second-game shop, and the published mod needs no script line.

## Authority

- Owner card n-modwb-10: the mod window's Items page.
- Owner: a mod is a layer over the base catalog: it adds, changes fields by key and switches a record off;
  nothing is deleted. The engine reads only the merged catalog. Mod item rules stay per game.
- Owner: shop stock is never stored in a save; there is one shop in both games.
- Owner: the fixed tree is `data/<kind>.toml` and `assets/<kind>/`.

## Rules

1. The Items page lists the rows the mod adds, its changes of shipped rows and its material masks.
2. An armour row is added by key, standing in for the install's first sold armour row of a mod slot.
3. A weapon row is added from its sprite; the picture's name is its key and its base is the first weapon row.
4. A shipped row is changed by its key in a change record; an empty number leaves the shipped number.
5. A change switches a row off: an added row takes no item code; a shipped row keeps its numbers, no shop sells it.
6. A material record allows its pair with the sprite imported into it, or forbids it.
7. A value the items reader or the item layer refuses is refused on the page in their words and makes no step.
8. A sprite is held to the item layer's picture rule and stored under `assets/items/`.
9. Every page edit is one draft step that Undo and Redo walk, also past a save; Ctrl+S writes it through the mod save.
10. The items kind's writer keeps the file's other lines and comments; an unchanged file is not rewritten.
11. Party equipment offers a page-made weapon before a save.
12. The first game's stock offers a page-made shop item; a mod for both games has it in a second-game shop.
13. Publish packs the items file and its sprites; the packed mod passes the data check with its script unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| mod data kind | `pkg/mod/kinds.go` `dataKinds` | `items` takes `writeItems` beside `readItems` |
| row writer | `pkg/mod/missionwrite.go` `writeRows` | `pkg/mod/itemwrite.go` gives it the items records |
| items reader | `pkg/mod/items.go` `ParseItems` | `off` in `[[change]]` |
| item layer | `pkg/game/moditemlayer.go` `setModItems` | off patches; drafted sprites through `modItemSprite` |
| W page | `pkg/ui/mapeditorworkspace.go` `modScopePages` | row `items`; fields in `pkg/game/mapeditormoditems.go` |
| mod file import | `pkg/game/modassetimport.go` `importModFile` | an edit rides in the import's own step |
| mod context | `pkg/game/mapeditormodcontext.go` `openModContext` | `draftItems` reads the drafted file by its kind |
| editor mod document | `pkg/game/mapeditormod.go` `document` | the snapshot holds the items file and its sprites |

## Proof

| test | rules |
|---|---|
| `pkg/mod` `TestItemsWriterKeepsTheBytesOutsideTheFieldsItChanges` | 10 |
| `pkg/mod` `TestItemChangeOffIsReadByTheItemsRule` | 5, 7 |
| `pkg/ui` `TestModWindowShowsTheItemsPageOfItsHost` | 1 |
| `TestItemsPageAddsChangesAndSwitchesOffRows` | 1, 2, 3, 4, 5, 6, 9 |
| `TestItemsPageRefusesAFieldByTheReadersRule` | 7 |
| `TestItemsPageImportsASpriteByTheLayersRule` | 7, 8 |
| `TestItemsPageWeaponIsOfferedToTheParty` | 3, 11 |
| `TestItemsPageModPublishesAndShopsOfferIt` | 9, 12, 13 |

Release tests at the release: the items release tests on rom1-en and rom1-ru, and the shop witness on rom2-en and
rom2-ru (owed, not run).

## Open debt

- The page edits the fixed items file only; records an older mod reads from another file are shown nowhere.
- A row's layer, anchor and figure fields and a material's name are kept but not edited on the page.
