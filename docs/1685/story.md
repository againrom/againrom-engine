# Map editor: the mod window's Spells page edits the mod's book

## Result

The mod window has a Spells page. An author picks a spell and changes its numbers, unfolds the globals, and
edits the mod's book by dragging spells between the catalog of both games and the book the game draws. A
drop swaps two book spells, places a catalog spell in an empty cell or takes a spell out, the note line
names what it wrote, and the game run from the mod shows the same book.

## Authority

- Owner: the nine answers of the spells page mockup (board cards o-sp16-q1 to o-sp16-q9): a spell without a
  cell may be placed (2); taking out writes off = true (3); a drop on a held cell swaps (4); the other game's
  spells are inert with a tip (5); fields and globals above the catalog (7); «Карта +» disabled,
  «Удалить» takes the chosen spell out (8).
- Owner: «the editor targets 1920x1080 and above»; the book holds up to 24 cells, the + beyond is inert with
  its tip; SAVX above 28 is card n-modwb-27.

## Rules

1. The chosen spell's fields and the globals' fold stand above the catalog.
2. The catalog lists both games' spells: the whole name as tip, a book spell ticked, the other game's inert.
3. A catalog spell without a picture shows its initials.
4. The book is the game's book painter's picture of the mod's book; each cell is the painter's cell.
5. A book spell dropped on another book cell swaps with it; both rows take the other's book_position.
6. A catalog spell dropped on an empty cell takes that book_position, also a spell without a cell.
7. A book spell dropped on a catalog cell, or «Удалить» with it chosen, writes off = true; «Карта +» does nothing.
8. The note line names the rows a gesture wrote.
9. A mod saved with a moved, an added and a removed spell runs with the book the page shows.
10. A chosen spell's number and a global ratio typed into the page are written to spells.toml.
11. At 1920x1080 the page stands inside the screen, labels whole, a press reaching each cell.
12. Every caption and tip comes from the words table, Russian and English, and no word id is unused.

## Code

| kind | its one builder | change |
|---|---|---|
| mod window page | `modScopePages`, `workspaceScopeRows`, `modPage` | one more entry, as Items |
| book picture | `composeOriginalSpellBar`, `bookCellRects` | put on `popupCanvas.Picture` by `modBook` |
| spell icon | `loadSpellAtlas`, `cutBookIcon` | catalog cells and book cells |
| catalog grid | `popupGrid` | Small cells, Ticked, Marks, inert Shut cells |
| drag and drop | `popupDrop`, `carryPress`/`carryMove` | new framework element; the page's rules are data |
| mod rows | `writeRows` (`writeSpells`), `ModSpellRefusal` | spells kind gets its writer and held draft |
| book rule | `SpellGameProfile.Edited` | a position admits a spell without a cell |
| control lookup | `editorControlRows` | `ControlPopupGridCell` |

## Proof

| test | rules |
|---|---|
| `TestSpellsPageCatalogTicksTheBookAndShutsTheOtherGamesSpells` | 1, 2, 3 |
| `TestSpellsPageBookIsTheGamesBookPainter` | 4 |
| `TestSpellsPageSwapsTwoBookSpellsByDrag` | 5, 8 |
| `TestSpellsPageAddsACatalogSpellIntoAnEmptyCell`, `TestEditedPlacesASpellWithoutACell` | 6, 8 |
| `TestSpellsPageTakesASpellOutByDragOntoTheCatalog`, `TestSpellsPageRemoveButtonTakesTheChosenSpellOut` | 7 |
| `TestSpellsPageBookIsTheBookOfTheSavedMod` | 9 |
| `TestSpellsPageWritesTheChosenSpellsFieldsAndTheGlobals`, `TestSpellsWriter*` | 10 |
| `TestSpellsPageFitsTheScreenAndEveryCellTakesItsPress` | 11 |
| `pkg/words` tests | 12 |

Release tests at the release: none. Review report: `review/mod-workbench/spells-page/story/RESULT.md`.

## Open debt

- A book wider than 24 cells and SAVX above 28 spells: card n-modwb-27.
- Spell pictures and their import: card n-modwb-25. Second-game maps in the editor: a separate story.
- A spells.toml value given by a setting (@name) shows the page's error line instead of the row.
