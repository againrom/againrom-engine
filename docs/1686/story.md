# Map editor: the placing palette and the Players windows read whole

## Result

The placing palette picks a new unit's owner from one drop-down list instead of a wrapping row of player
buttons, and its (?) mark explains placing. The Players window's cells are as big as the palette's, each column
shows its player's whole name, and the legend names each relation with its picture and a (?) mark. A player's
window is titled with the player's number and name, shows every label whole, names control and relations in
words, leaves out the relation to itself and opens with its Data section folded.

## Authority

- Owner: the eight answers to the palette and Players screen questions: one owner choice; tabs stacked in
  the 400 window; 44 px cells at natural width; legend rows with name, picture and (?); the cross a word
  «x» with the hover line «Удалить игрока»; no relation to itself; control and relations as words with
  pictures; Data folded on open.

## Rules

1. The palette's owner row is one choice reading the chosen player; its list opens inside the editor window.
2. The palette keeps its 400 width; a block tab too wide for its share takes a line of its own.
3. The Players window's cells are the palette's 44 px, and the window is as wide as the matrix needs.
4. Each legend row names one relation, shows its picture and has a (?) saying what it means.
5. The delete cross reads the words table's «x» and its hover line says what it does.
6. A player's window lists its relation toward every other player and not toward itself.
7. A player's window shows control and each relation as words, a relation with its picture.
8. A player's window opens with its Data section folded; a press on the heading opens it.
9. A player's window is titled with the player's number and name, and every label is whole at 1920x1080.
10. Every caption, hover line and help of these windows comes from the words table, in Russian and English.
11. At 1920x1080 these windows and the delete question stand inside, presses reaching them.

## Code

| kind | its one builder | change |
|---|---|---|
| place palette | `placeGridSpec`, `placeOwnerRow` | owner row a choice; help moved to the title (?) |
| row drop-down | `rowDropdown` (`placeOwnerDropdown`) | the owner list is one more drop-down |
| relation matrix | `relationGridSpec`, `relationFit` | 44 px cells, whole heads, named legend, rows page |
| row head column | `placeHeadColumn` (`playerHeadRows`) | hand boxes and the 180 cap gone |
| object window | `objectSpec` (`playerFieldRow`, `playerTitle`) | player rows as words; Data a closed fold |
| question | `questionRowsSpec` | notes wrap to the window's text width |
| description page | `popupPage` | draws grid heads and cells |

## Proof

| test | rules |
|---|---|
| `TestPlacePaletteOwnerRowIsOneChoiceWhoseListOpensInside`, `TestPlaceOwner*` | 1 |
| `TestPlacePaletteBlockTabsStackInItsWidth` | 2 |
| `TestPlayersMatrixCellsAreThePalettesSide`, `TestRelationMatrixHeadsEachColumnWithItsRowHead` | 3 |
| `TestPlayersLegendRowsNameIconAndHelp` | 4 |
| `TestPlayersDeleteCrossIsAWordWithItsTip` | 5 |
| `TestPlayerWindowOmitsItsRelationToItself` | 6 |
| `TestPlayerWindowNamesControlAndRelationValues` | 7 |
| `TestPlayerWindowDataFoldIsClosedAtFirst` | 8 |
| `TestPlayerWindowTitleIsTheNumberAndName`, `TestPlayerWindowLabelsAreWholeAtFullHDSize` | 9 |
| `TestPlacePlayersBuildersHoldNoEnglishLiteral`, `TestEveryUsedIDIsInTheEnglishTable` | 10 |
| `TestPlacePaletteHoldsTheLayoutProperties`, `TestPlayersWindowHoldsTheLayoutProperties` | 11 |
| `TestPlayerWindowHoldsTheLayoutProperties`, `TestPlayerDeleteQuestionHoldsTheLayoutProperties` | 11 |

Release tests at the release: none. Review report: the lane's `review/editor-ui/t2/story/RESULT.md`.

## Open debt

- A player's control and relations stay read-only in its window; the Players window edits relations.
