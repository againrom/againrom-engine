# Read map details, objects and findings in their panels

## Result

The Map panel measures its labels, fits between the side docks and scrolls beneath its title when shortened.
The description occupies one row, with its full text on hover and a link that opens the editor.
Long names in Objects and scenarios show in full on hover.
The Report dock leaves room for up to three findings and keeps save instructions visible.
Finding details move behind (?) when they cannot fit.

## Authority

- Owner: «Т5, 9, 10, 11» and «10, 15, 16 тоже берем».
- The owner chose one linked description row, a minimum height for up to three findings and visible save instructions.

## Rules

1. The description takes one linked row and opens the map description editor.
2. The Map panel measures its labels, fits between the side docks and pages beneath a fixed title and help mark.
3. A clipped description or object name shows its full text in a tip.
4. Clicking an object row selects the object.
5. The Report dock keeps room for up to three findings and the count when dragged down.
6. Details that cannot fit move behind the report's title help, where the empty dock also keeps its hint.
7. Clicking a finding selects it and takes the map to its record.
8. When a save needs confirmation or is refused, the Report body shows the outcome at the minimum height.

## Code

| kind | its one builder | change |
|---|---|---|
| Map panel | `panelSpec`, `naturalWidth`, `layoutPopup` | Measured labels and fitted values within the map area |
| Objects tree | `treeSpec`, `drawListBox` | Existing picker rows with a shared clipped value tip |
| Docked report | `dockReportContent`, `reportHintSpec`, `layoutPopup` | Measured minimum, visible outcome and foldable details |
| Field and list tips | `popupTipAt`, `layoutTip`, `drawPopupTip` | Shared clipping geometry and full text |
| Description previews | `popupdesc.go`, `layoutPopup`, `popupPage` | Shared paint and tip geometry for the preview |
| Captions and help | `pkg/words` | English and Russian report captions and help |

## Proof

| test | rules |
|---|---|
| `TestMapDescriptionIsOneLinkedValue` | 1 |
| `TestMapPanelPagesWithFixedTitleAndFullValueTip` | 2, 3 |
| `TestNaturalMapPanelFitsBetweenTheDocks` | 1, 2, 3 |
| `TestObjectsPanelClippedMemberTipAndSelection` | 3, 4 |
| `TestReportDockDragKeepsThreeFindings` | 5, 6, 7 |
| `TestShortReportReservesOnlyItsFindings` | 5 |
| `TestDockedSaveOutcomeStaysVisibleAtMinimum` | 6, 8 |
| `TestDescriptionPanelPreviews` | 2, 3, 5, 6 |

Release tests in EN and RU include `TestReleaseMapPanelFrames` and `TestReleaseMapEditorObjectsAndGroupsFrames`.

## Open debt

- Installed fonts and the live report resize and hover appearance need the EN and RU editor panel witnesses.
