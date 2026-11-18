# One scroll in every editor window

## Result

Every scrolled area of the map editor now scrolls the same way. The wheel
works anywhere over the scrolled content or its bar, a click on the bar's
track moves one page toward the click, and dragging the thumb makes the
content follow it. The bag window's item grid, which answered only the wheel
over its cells, now answers all three.

## Authority

- Owner: «скролл у нас везде должен работать одинаково! делаем унифицированный скролл везде».

## Rules

1. The wheel over a scrolled area or its bar moves it, including the rows beside a grid.
2. A click on a bar's track moves the area one page toward the click.
3. Dragging a bar's thumb moves the area and the thumb follows the pointer.
4. Thumb position and page size come from the same area data for every area.
5. A list's wheel moves its highlight; a window or grid wheel moves its first row.

## Code

| kind | its one builder | change |
|---|---|---|
| editor scroll area | `MapEditor.scrollAreas` | every wheel and bar is data of one area, routed by `scrollWheel` and `scrollPointer` |
| editor grid section | `MapEditor.gridArea` | the item, forest, relation, mod and prompt grids share one area |
| editor popup list | `listArea`, `dropdownArea` | lists, menus and drop-downs are areas that move a highlight |

## Proof

| test | rules |
|---|---|
| `TestBagGridWheelOverTheGridMovesTheTop` | 1, 5 |
| `TestBagGridWheelOverTheWindowRowsMovesTheTop` | 1 |
| `TestBagGridTrackClickMovesOnePage` | 2, 4 |
| `TestBagGridThumbDragMovesTheTop` | 3, 4 |
| `TestEveryEditorScrollAreaAnswersWheelTrackAndThumb` | 1, 2, 3, 4, 5 |
| `TestEditorListsClampAndThumbFollowsPointer` | 3, 5 |
| `TestMapEditorIdentityGridWheelStaysInTheChosenBlock` | 1 |

Release tests at the release: none.

## Open debt

- The tree, menu and prompt-list areas are built by the same builder but the walking test has no fixture for them.
