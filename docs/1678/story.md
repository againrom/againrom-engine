# A mod turns spells off, moves them in the book and sets the book's size

## Result

A mod can now take a spell out of the game: the spell leaves the book of the player's mage and party, and no shop
sells its book or scroll. A mod can move a spell to another cell of the book and show fewer cells. Map units still
cast what the map gives them. A spell a hero already knew is only hidden: the save keeps it, and without the mod the
book shows it again. Without a mod nothing changes, in either game.

## Authority

- Owner: «Spells in mods»: a book position is display order; a spell is stored under its name; "off" covers the book
  of the player's mage and party and the shop's books and scrolls; map units cast as the map sets them; a spell known
  before the mod is hidden, not erased. This story is the second of four, after `docs/1673/story.md`.
- `AI-SPELLIDENT-286`: the book's 24 cells and their spells; `SHOP-CONSUME-073`: the shop's book and scroll admission.

## Rules

1. A spells row may set off or on and a book position; the global book size is 1..24; a value outside is refused.
2. A refusal names the mod, the file, the line and the limit; each new field is set by one mod.
3. Under the mod the book of the player's mage and party shows only spells that are on, in the mod's order and size.
4. Under the mod no shop stocks a book or scroll of a spell that is off; the rest stock by the usual rules.
5. A map unit's book, mask and cast list stay the map's under the mod.
6. A known spell turned off is hidden: the save under the mod equals the save without it and shows it without the mod.
7. Without a mod the book rows, shop admission, catalog and save bytes are those of the game, on both games.
8. Under the mod no player path selects, arms, casts or autocasts a spell that is off, though the hero keeps it.
9. An explicit on is judged against the cell the edits leave, whether in one row, two rows or two mods.

## Code

| kind | its one builder | change |
|---|---|---|
| spell catalog profile | `pkg/rules/spellcatalog.go` GameProfile | book size; Edited applies off, position and size, tests an on after all edits |
| spells mod kind and merge | `pkg/mod/spells.go` ParseSpells, `pkg/mod/kindread.go` readSpells | `off`, `book_position`, `book_size` |
| spells applier | `pkg/game/modspell.go` setModSpells | refuses through `pkg/mapload/modspell.go` ModSpellProfile |
| book rows | `pkg/data/spellpresentation.go` SpellPresentation | unchanged ground; `pkg/game/spell.go` spellBookOrder orders cells |
| book icon slot | `pkg/data/spellicon.go` | unchanged; a moved cell draws its icon (`pkg/ui/spellbook.go`) |
| shop admission | `pkg/game/shopconsumables.go`, `pkg/game/shop.go` | reads the edited profile (SpellGameProfileOf) |
| player castable-spell test | `pkg/ui/spellbook.go` bookHolds | quick slot, C key and saved selection ask it; the game drops an autocast it does not hold (`pkg/game/spell.go`) |
| SAV book writer | the one save producer | unchanged: the book is stored by spell number |

## Proof

| test | rules |
|---|---|
| `TestGameProfileIsTheBookTheGameDraws`, `TestEditedSwitchesMovesAndSizesTheBook` | 1, 7 |
| `TestEditedRefusesNameTheEditAndTheLimit`, `TestModSpellProfileRefusalNamesTheRow` | 1, 2 |
| `TestParseSpellsReadsTheSwitchPositionAndBookSize`, `TestSpellSwitchRefusalsNameTheLineAndTheLimit` | 1, 2 |
| `TestSpellSwitchFixtureGivesTheSwitchPositionsAndSize`, `TestSpellSwitchFieldsAreOneModsEach` | 1, 2 |
| `TestSpellSwitchRefusalsNameTheRowAndTheLimit` | 2 |
| `TestSpellSwitchBookRowsOnBothGames`, `TestAMovedBookCellDrawsItsOwnIcon` | 3, 7 |
| `TestSpellSwitchShopStocksNoBookOrScrollOfAnOffSpell` | 4, 7 |
| `TestSpellSwitchLeavesAMapUnitsCastList` | 5 |
| `TestSpellSwitchHidesAKnownSpellAndTheSaveKeepsIt` | 6, 7 |
| `TestSpellCatalogProfilesReproduceTheReplacedTables`, `TestShopBookAndCaptionSpellListsKeepTheirAnswers` | 7 |
| `TestAnOffSpellInAQuickSlotIsNotCast`, `TestAnOffSavedSpellIsNotCast`, `TestSpellSwitchStopsAnAutocastOfAnOffSpell` | 8 |
| `TestEditedOnFollowsTheMovedCell`, `TestAnExplicitOnFollowsTheMovedCell` | 9 |

Release tests at the release: `TestReleaseShopStockCensusWithoutModsIsShipped`, `TestReleaseSecondShopStockMatchesMain`,
`TestReleaseSecondShopBookIcons`, `TestReleaseOwnerShopScrollStoredValues`,
`TestReleaseSpellbookPopupComposesFromSpellsTxtAndInstalledLabels`, `TestReleaseModSpellKeysEditTheInstalledTable`,
`TestReleaseModSpellEditsReachACastAndTheSave`.

## Open debt

- A book above 24 cells and up to 28 waits for the book art; above 28 waits for the SAVX book record.
- A spell the game's book has no cell for cannot be turned on or placed; its icon waits for a later story.
- The spells kind stays kept: two mods still cannot set one field; `load-after` does not order it yet.
- A custom spell table keeps its own book; off, position and size reach the book only on the game's own catalog.
