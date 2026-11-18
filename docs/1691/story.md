# The editor's variables table on the one window framework

## Result

The map editor's variables table is now laid out by the one window builder in English and Russian. It is as wide as
its columns need, within the screen, and every column head shows whole. A trigger name too long for the screen takes
the room the other columns leave, ends in "..." and shows whole in a tip. The title's (?) says that Value is the
number in the trigger's condition or action, not the variable's current value. Rows and presses are as before.

## Authority

- Owner: the T6 answers on the variables table mockup (cards o-t6-q1 and o-t6-q2), both (a), binding.
- Owner: «The map editor targets screens of 1920x1080 and above»; no layout or test for a smaller window.

## Rules

1. The table is as wide as its columns need, within the screen less 16; no fixed 680.
2. Every column head shows whole on one line, measured and painted in the heading face.
3. A cell too long for its column ends in the cut mark and shows its whole text in the tip after the shared delay.
4. When the cells do not fit, the columns that fit keep their natural width and the cut column takes the room left.
5. The head stays Value («Значение»); the title's (?) says it is the number in the condition or action, not the value.
6. A row's press opens its trigger, an unused variable's row its variable window; rows and their words are unchanged.
7. The table lies inside a 1920x1080 screen, pages when its rows outgrow it, and its rows are hit as laid out.

## Code

| kind | its one builder | change |
|---|---|---|
| column list | `pkg/ui/editorpopuplist.go` listColumnWidths | natural column needs; cut columns share the room left |
| list-window fit | `pkg/ui/mapeditorlistfit.go` listFit | the table's columns, notes and title as data (`listColumnsFit`); cells and heads by advance |
| window layout | `pkg/ui/editorpopup.go` placeWindow | heads in the heading face over the measured columns |
| list painter | `pkg/ui/widgetscroll.go` drawListBox | each cell fitted to its column with the cut mark |
| tip | `pkg/ui/editorpopup.go` popupTipAt | a cut list cell's whole text through layoutTip |
| variables table | `pkg/ui/mapeditorvariabletable.go` variableTableSpec | data only: columns, help, whole rows, `listFit` for the width |
| description reader | `pkg/ui/popupdesc.go` build | list columns in a description; the page shows heads and cells |
| words | `pkg/words/text/{en,ru}/strings.toml` | the table's help |

## Proof

| test | rules |
|---|---|
| `TestVariablesTableLongTriggerTakesTheRoomLeft` | 1, 4 |
| `TestVariablesTableHoldsTheLayoutProperties` | 2, 3, 6, 7 |
| `TestVariablesTableHelpSaysWhatValueIs` | 5 |
| `TestVariablesTableWidthComesFromListFit` | 1 |
| `TestPopupListCellTipUsesTheSharedDelay` | 3 |
| `TestVariablesTableOpensFromTheWindowMenuWithItsColumns`, `TestVariablesTableRowSelectsItsTriggerDockedAndFloating` | 6 |
| `TestDescriptionListFewRowsKeepTheirNaturalColumnWidth`, `TestDescriptionListColumnsFitPageAndTipAtEditorTarget` | 1, 2, 3 |

Release tests at the release: none.

## Open debt

- The bottom note shows only the empty state; a visible-row count note waits for an owner decision.
