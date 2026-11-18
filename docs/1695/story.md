# Read map editor lists at the width they need

## Result

Lists in the map editor open at the width their contents need. Long lists scroll while the title and buttons
stay put. Hover over a shortened line to read it in full. The Report window shows the result of the check or save
with its counts below the list.

## Authority

- Owner: «Т5, 9, 10, 11».
- Owner: one Report title for checks, saves with findings and refused saves, with the outcome and totals under the list.

## Rules

1. Records, Scenario, Traps and spells, Maps, Groups and Selection lay out their text and actions at natural width.
2. Paging keeps the title and buttons in place, and clicking a visible row or action reaches its usual target.
3. The floating Report keeps one title, shows the outcome and totals under the list, and keeps save confirmation.
4. Map editor keys keeps the full save line, which a tip shows after the usual delay when the line is shortened.
5. A record opens above the list that opened it. Pressing an exposed part of either window brings it forward.

## Code

| Kind | Its one builder | Change |
|---|---|---|
| List windows | `listSpecBase`, `groupsSpec`, `selectionSpec` through `layoutPopup` | Natural width and title help |
| Button row | `popupButtonWidths` | Scenario and Selection use the framework's measure |
| List row | `newListBox`, `drawListBox`, `listRowLead` | The widest row and its leading space contribute to width |
| Row tip | `popupTipAt`, `listCellTip`, `layoutTip` | The full row reaches the shared tip |
| Report data | `openReport`, `listSpecBase` | The outcome and totals are shown below the findings |
| Words | `pkg/words` | English and Russian title, outcome and key text |
| Description preview | `popupDesc`, `popupPage` | The layout supplies list and scrollbar rectangles |

## Proof

| Test | Rules |
|---|---|
| `TestEditorListWindowsLayoutAndActions` | 1, 2 |
| `TestEditorListButtonsUseNaturalWindowWidth` | 1 |
| `TestWindowedReportKeepsTitleAndShowsOutcomeWithTotals` | 3 |
| `TestScenarioRowOpensRecordAboveItsList` | 5 |
| `TestFloatingReportRowOpensRecordAboveItsList` | 5 |
| `TestEditorKeysFitTheFullSaveLineAndShareTheRowTip` | 4 |
| `TestDescriptionPlainListFitsPagesAndShowsFullRowTip` | 2, 4 |

Release tests at the release: `TestReleaseMapEditorPlacementPaletteFrames`,
`TestReleaseMapEditorInstalledTextIsCapturedAtScaleOne`.

## Open debt

- Installed English and Russian previews of the eight windows remain for the editor UI framework release checks.
