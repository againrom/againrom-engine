# A map editor toolbar and search row that fit

## Result

Search opens on a temporary row below the map tabs and the docked toolbar.
It lines up with File, and a hover shows the whole query when the field
cuts it. The toolbar tries smaller picture buttons when space runs short
and can move a whole group to a second row. Its buttons show their names
and any shortcuts on hover, in the editor's language.

## Authority

- Owner approved T15 and chose the temporary search row and the toolbar's two sizes and group wrap.
- The editor UI rulebook defines shared elements, text fitting, map room and hover tips.

## Rules

1. Search opens under File below the tabs and docked toolbar, and closing search frees the row.
2. The toolbar tries 36 px, then 22 px, then a second row of complete groups, preserving the map minimum.
3. At 1920×1080, bar captions and picture buttons fit, and a settled hover explains the picture or cut query.
4. Presses on bar entries, picture buttons and the search field use the displayed rectangles and existing actions.

## Code

| kind | its one builder | change |
|---|---|---|
| Bar | `layoutPopup`, `drawPopup`, `popupLayout.hit` | Lay out, paint and hit top bar, toolbar and search cells |
| Picture button | `drawIconButton` | Draw the same toolbar entries in the docked strip and floating window |
| Search field | `searchFieldRow`, `drawEditRow`, `editorClip` | Fit a query and preserve caret and selection while editing |
| Tip | `popupTipAt`, `layoutTip`, `drawPopupTip` | Pass the word, shortcut or full query through the settled hover |
| Floating toolbar | `floatSpec`, `placePopup` | Use the placement store and measure from the title and pictures |
| Description and page | `newEditorMockup`, `layoutPopup`, `mockupPage` | Read the same rectangles in the PNG and HTML previews |
| Words | `pkg/words` | Read the existing English and Russian captions and tips |

## Proof

| test | rules |
|---|---|
| `TestMapEditorSearchUsesTemporaryRowBelowChrome` | 1, 4 |
| `TestMapEditorToolbarFitsCompleteGroupsAsItGrows` | 2 |
| `TestChromePreviewUsesEditorRects` | 1, 3, 4 |
| `TestMapEditorChromeBarControlsUseLayoutHits` | 3, 4 |

Release tests in EN and RU: `TestReleaseMapEditorInstalledTextIsCapturedAtScaleOne`.

## Open debt

- Installed EN and RU font fit and appearance await the release witness and a chrome preview of each locale.
