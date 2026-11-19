# One element each for the map editor's bar, lists and tips

## Result

The map editor looks and answers as before on maps of both games. Every
drop-down list in it, from the menus and search hits to the object window,
the prompts and the Units tab's player list, answers a press the same way.

The menu bar, toolbar, search field, status line, player legend and tips
each keep their one shared element, and a check keeps a second from returning.

## Authority

- Owner: one builder per kind; «Редактор ром2 давай доделаем».
- Owner rule Q7: a tip shows after 300 ms, a (?) mark's help at once.

## Rules

1. A press on an open list's row chooses that row and closes the list.
2. A press on a drop-down's box opens or closes its list; a press on its scroll bar only scrolls.
3. A press beside an open choice list closes it and reaches what lies under it.
4. The menu bar, toolbar and search field are bars of the one bar element.
5. The status line and the player legend are windows of the one window element.
6. Every tip is laid out and painted by the one tip element, after 300 ms or at once for help.

## Code

| kind | its one builder | change |
|---|---|---|
| drop-down list | `dropdown`, `drawDropdown`, `dropdown.press` | five press handlers call `press` with their acts |
| hover tip | `layoutTip`, `drawPopupTip`, `editorTipDelay` | none; guarded |
| toolbar button row, menu bar, search entry | `popupBar` via `layoutPopup`, `drawPopupBar` | none; guarded |
| status and legend line | `statusStripSpec`, `playerLegendSpec`, `drawWindow` | none; guarded |

## Proof

| test | rules |
|---|---|
| `TestDropdownPressChoosesTogglesAndLeaves` | 1, 2, 3 |
| `TestFieldPromptListOpensChoosesAndCloses` | 1, 2 |
| `TestOpenDropdownChoosesARowAboveItsAnchor` | 1 |
| `TestMapEditorUnitsTabPlacesForTheChosenPlayer` | 1, 2 |
| `TestEditorToolsMenuPagesAfterThirteenRows` | 1 |
| `TestMapEditorBarSearchChoiceIsATreePress` | 1 |
| `TestEditorChromeFrameworkFitsAndHitsAtEditorTarget` | 5, 6 |
| `TestEditorFrameworkElementsHaveOneBuilder` | 4, 5, 6 |

Release tests at the release: none. Pixel equality before and after at 1920x1080 on both games is in the review
report.

## Open debt

- The trigger window's drop-down probes in `mapeditortrigger.go` wait for the trigger window's own lane.
