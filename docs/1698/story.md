# Map tabs and lists that fit the editor

## Result

Each map tab has its own close mark. Long tab strips page with the active
map in view. Menus and search results fit above or below their controls
and scroll when more rows are available. Long tab names and status text
show in full on hover, shortcut help sits behind the status question mark,
and the player legend fits inside the map.

## Authority

- Owner: «10, 15, 16 тоже берем».
- Owner: close the map named by the tab and show at most 13 menu rows at a time.

## Rules

1. A tab's close mark closes that map and asks before closing it with unsaved changes.
2. Tabs take their natural widths, page when needed, and show full clipped names in tips.
3. Menus show at most 13 rows and scroll to the remaining actions.
4. Menus, search results and choices fit below their control, above it or on a scrollable page.
5. The status stays on one line, opens its full clipped text on hover and puts shortcut help behind a question mark.
6. The View command toggles a legend whose player rows wrap and page inside the map corner.

## Code

| kind | its one builder | change |
|---|---|---|
| Tabs | `tabStrip.layout`, `tabStrip.draw` | Map tabs use natural width, individual close marks and paging. |
| Open choices | `dropdown.fitList`, `drawDropdown` | Menus, search results and choice lists share a bounded frame. |
| Status and legend rows | `layoutPopup`, `drawPopup` | A one-line status strip and titleless legend use the popup row layout. |
| Tip | `popupTipAt`, `layoutTip`, `drawPopupTip` | Tab controls and clipped status use the same tip and hover delay. |
| Description preview | `layoutChromeMockup`, `mockupPage` | The chrome preview uses the live layout and paints its rectangles. |
| Words | `words.Book` | English and Russian tab control tips. |

## Proof

| test | rules |
|---|---|
| `TestMapTabCloseTargetsItsOwnMap` | 1 |
| `TestEditorToolsMenuPagesAfterThirteenRows` | 3 |
| `TestEditorChromeFrameworkFitsAndHitsAtEditorTarget` | 1, 2, 4, 5, 6 |
| `TestOpenDropdownChoosesARowAboveItsAnchor` | 4 |
| `TestFittedDropdownOpensAboveOrPagesInsideWindow` | 4 |
| `TestChromeDescriptionFitsAtEditorTarget` | 2, 3, 4, 5, 6 |

Release tests at the release: `TestReleaseMapEditorStatusFitsWithPassability`,
`TestReleaseMapEditorGroupNoteFitsTheStatusStrip`.

## Open debt

- Verify the editor chrome at 1920×1080 with the English and Russian install fonts at release.
