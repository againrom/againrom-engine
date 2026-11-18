# Map editor: the palette and brush windows read plainly and fit their contents

## Result

The Structures, Objects, Trees, Height, Terrain and Bridge windows carry their names as titles, with the
how-to behind the title's question mark, and open as wide as their words and names need. Objects no longer
shows the forest tool: Forest lives in Trees, where kinds are ticked in a grid of large cells and One kind
shows the tree list. Terrain shows the chosen ground as a large sample beside its list, and the Sacks
button opens the window that adds a sack at once.

## Authority

- Owner: the seven answers to the palettes and brushes mockup (review/editor-ui/t12/RESULT.md): Objects has
  no Tool row (1); registry names in both languages (2); Sacks opens Add directly (3); Forest ticks in the
  grid only (4); 80-pixel forest cells (5); no picture beside the Height list (6); the chosen ground beside
  the Terrain list (7).
- Owner: «the editor targets 1920x1080 and above».

## Rules

1. Structures is titled by its name, its how-to in the title's (?), the Ruined switch at the button row's left.
2. Objects has Size and Shape rows and no Tool row; its list shows the chosen object's picture beside it.
3. The Sacks bar button opens the Add window for a sack; a second press closes it.
4. Trees opens in Forest: kinds are ticked in an 80-pixel grid only, with Density, a count and Clear marks.
5. Trees' One kind shows the tree list with the chosen tree's picture and no grid.
6. Height is titled by its name, shows its four actions and no picture beside them.
7. Terrain shows the chosen ground as a large sample beside its list; it follows the choice.
8. Bridge is titled by its name, its Shape row reads Rectangle, its how-to in the title's (?).
9. Object and structure names stay the install's registry names in both languages.
10. Every caption, note, help and button text comes from the words table, Russian and English.
11. At 1920x1080 each window stands inside the screen, every label whole, a press reaching each part.
12. No window paints after the popup painter; the picture beside a list is the popup's own.

## Code

| kind | its one builder | change |
|---|---|---|
| palette and brush window | `listSpecBase` brush branch | every window is data from `brushWindowOf` |
| brush row | `brushRow` | rows by kind: Tool, Size, Shape, Strength, Fill, Density |
| list window width | `listFit` | the Players matrix and each brush window are its data (`relationFit`, `brushFit`); the mockup's grid head uses `sizeGrid`'s rule |
| picture beside a list | `popupSpec.ListPicture`, `drawPopup` | `brushPicture` feeds it; `drawBrushPreview` deleted |
| button row switch | `popupSpec.SwitchButton` | Ruined moves there from the window buttons |
| bar entry | `editorCategories`, `showRecords` | Sacks marked `addsAtOnce` opens Add |
| forest grid | `forestGridSpec`, `placeGridShell` | Structures' 80-pixel shell; `forestWindow` note and button |
| window description | `popupDesc` | list picture, switch and ticked cells from the mockup |

## Proof

| test | rules |
|---|---|
| `TestMapEditorStructureGridRuinedSwitchOnlyChangesCopies` | 1 |
| `TestMapEditorObjectsWindowLeavesForestToTrees`, `TestMapEditorObjectsBrushShowsTheHighlightedObjectsPicture` | 2, 4 |
| `TestMapEditorSacksEntryOpensTheAddWindow`, `TestMapEditorEveryAddRouteOpensTheAddWindow` | 3 |
| `TestMapEditorForestTicksInTheGridOnly`, `TestMapEditorForestGridCellIsEightyPixels` | 4, 5 |
| `TestForestClearMarksUnticksEveryKind`, `TestMapEditorTreesWindowShowsItsOneKindRows` | 4, 5 |
| `TestMapEditorHeightShowsNoPicture`, `TestHeightToolModeAndStrengthRows` | 6 |
| `TestMapEditorTerrainShowsTheChosenGround` | 7 |
| `TestMapEditorBridgeWindowShowsItsRectangle` | 8 |
| `TestMapEditorPlaceNamesStayTheRegistryNames` | 9 |
| `TestMapEditorBrushWordsAreTheInstallsLanguage`, `pkg/words` tests | 10 |
| `TestMapEditorBrushWindowsHoldTheLayoutProperties` | 11 |
| `TestMapEditorBrushWindowsPaintNothingAfterThePopup` | 2, 5, 6, 7, 12 |

Release tests at the release: `TestReleaseMapEditorForestStrokeSavesAMapTheGameReads`,
`TestReleaseMapEditorForestFrames`, `TestReleaseMapEditorAddWindowFrames`.
Review report: `review/editor-ui/t12/story/RESULT.md`.

## Open debt

- `question.spec` keeps its own width measure (other padding, notes and window clamp); it does not join `listFit` yet.
- The Add window's picture still paints after the popup (`drawAddPreview`, `drawAside`): T3 moves it.
