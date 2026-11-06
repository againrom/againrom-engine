# The map editor's variables table

## Intent

Give the map maker one window that lists each variable the map's scripts
use and the triggers that use it. The window opens from the top bar's
Window menu («Окно»). Map editor work counts zero toward a release.

## Authority

Owner remark 8 asks for a tool in the Window menu that shows the variables
in use and the triggers that name them. The owner's mockup is a window
«Таблица Переменных» with four columns: «Номер переменной», «Значение»,
«=, >, < и так далее», «Название триггера». The brief fixes the title as
«Таблица переменных» (EN "Variables table"). The owner's ruling of one
builder per kind applies. The map editor is this project's own program, so
no claim is needed. The value an action writes follows TRIG-ACT-004:
action 3 is `slot[p0] = p1`, action 8 is `slot[p0]++`. The base is
`49430e7a3`.

## As-built behaviour

The Window menu has a fifth row, Variables table, after Report panel. It is
a switch row of the toolbar table (`pkg/ui/mapeditortoolbar.go`, key
`variables`, menu `window`, no toolbar icon) and carries the open mark while
the table shows. It is disabled with no map open. A press runs the unlisted
key row `variables` (`pkg/ui/mapeditorkeys.go`), which `toggleVariables`
answers: it opens the table, or closes it when it shows.

The table is the list popup (`popList`) on the list kind `variables`.
`listKind` returns it while the records list shows filter
`editorVariablesFilter` (12). Opening goes through `showRecords`, so the
table takes the list popup's cancel checkpoint, its bar-window release and
its placement. Esc and the close button close it as they close a records
list: the checkpoint taken at opening comes back, including the selection.
The popup's corner is the list popup's one corner, shared with the records
lists and palettes; its size is kept under the window key `variables`.

The columns and their words:

| Column | EN head | RU head | Cell |
| --- | --- | --- | --- |
| 1 | Variable | Номер переменной | The register, then two spaces and its label where the label file gives one |
| 2 | Value | Значение | A check: the other side of its pair, a constant's number or else the other check's title (`nodeTitle`). Action 3: the parameter after the variable's. Action 8: 1 |
| 3 | Comparison or action | =, >, < и так далее | A check: the pair's comparison read from the check's own side (`cmpWord`, over `cmpWordOf`); a check on the right of its pair shows the mirror (`mirroredCmps`, as `readCondition` reads a pair). An action: its name (`opName`) |
| 4 | Trigger | Название триггера | The trigger as the Scripts list writes it (`scriptRowText`): "Trigger N  name" |

Rows run in register order. Within a register they run in trigger order,
and within a trigger in pair order (left side before right) and then in
action slot order. A variable the Variables list lists but no trigger uses
has one row with the register cell only and empty value, comparison and
trigger cells. This is the lane's choice: the Variables list lists every
labelled register and every addressed register, so the table names the same
registers. With no variable the popup shows the note `editor.variables.none`.

A press on a row runs `chooseListed` with the row's record: the trigger's
record, or `-2-i` for the row of variable `i` with no use. That is the route
the Scripts list's rows and the tree's rows end in. The rows are built from
the document on each frame, so a wizard Apply, an Add, a Delete, an undo or
a redo shows on the next frame.

### Kinds and builders

| Kind | Builder | What changed |
| --- | --- | --- |
| Window (list popup) | `listSpecBase`, `listRows`, `listTitle`, `listChoose`, `closeList` in `pkg/ui/mapeditorwindow.go` | One more list kind, `variables`, as data in each dispatch; `variableTableSpec` fills the spec |
| Popup layout and paint | `popupSpec`, `popupFlowOf`, `popupFit`, `layoutPopup`, `drawPopup` in `pkg/ui/editorpopup.go` | `popupSpec.ListColumns` (head, share) as data: each head wraps to its column over the list |
| List painter | `listBox`, `drawListBox` in `pkg/ui/widgetscroll.go` | `listBox.Columns`: a row's tab-separated cells, each clipped to its column, and a rule between columns; with no Columns a row draws as before |
| Script reader (variables) | `scriptVariables` in `pkg/ui/mapeditorscriptauthor.go` | Each variable records its uses (`scriptVariableUse`: trigger, pair or action slot, side, node); writers and readers are built from the same loop, so the Variables list's rows are unchanged |
| Menu row | `toolButtons` in `pkg/ui/mapeditortoolbar.go`, `menuItems` | One row as data |
| Key row | `editorKeyRows` in `pkg/ui/mapeditorkeys.go` | One unlisted row as data |
| Selection route | `chooseListed` | Unchanged; the table calls it |
| Words | `pkg/words/text/{en,ru}/strings.toml` | `editor.variables.{title,register,value,how,trigger,none}` |

No second window builder, script reader, paint path or selection route was
added. The list popup docks only for the report; the table stands floating,
as the records lists do, with the side panels docked or floating.

### Divergences

None. The table shows what the editor's own model reads. DIV-4566 and
DIV-4567 are returned unused.

## Proof

Fixture-only; no install was read.

- `pkg/game/mapeditorvariabletable_test.go`,
  `TestVariablesTableListsEachUseAndFollowsEdits` (EN, RU): a synthetic map.
  Trigger "Alpha" compares the check on register 20 with the constant 10
  (code 2) in pair 1 and with the constant 3 (code 1) in pair 2, and sets
  register 21 to 42. Trigger "Beta" compares register 21 with 10 (code 0),
  adds 1 to register 22 and wins. The label file names register 40
  "Spare", which no node addresses. The table opened from the Window menu
  shows, in EN:

  | Variable | Value | Comparison or action | Trigger |
  | --- | --- | --- | --- |
  | 20 | 10 | is greater than | Trigger 1  Alpha |
  | 20 | 3 | does not equal | Trigger 1  Alpha |
  | 21 | 42 | Set variable | Trigger 1  Alpha |
  | 21 | 10 | equals | Trigger 2  Beta |
  | 22 | 1 | Add 1 to variable | Trigger 2  Beta |
  | 40  Spare | | | |

  and in RU «больше», «не равно», «Задать переменную», «равно», «Прибавить 1
  к переменной», «Триггер 1  Alpha», «Триггер 2  Beta». The table's
  registers equal the Scripts tree's Variables rows. A press on rows 4, 1
  and 6 selects Beta, Alpha and the Spare variable. The wizard adds "Add 1
  to variable Spare" to Beta and the Spare row reads `1 | Add 1 to variable
  | Trigger 2  Beta`; undo restores the rows. Delete of Alpha leaves `20 |
  | |`, Beta's two rows as "Trigger 1  Beta" and the Spare row, with the
  registers still equal to the Variables list; undo restores the rows.
- `pkg/ui/mapeditorvariabletable_test.go`:
  `TestVariablesTableOpensFromTheWindowMenuWithItsColumns` (EN, RU: the row,
  its mark open and closed, the title, the four heads laid out over their
  columns above the list, the rows, the registers against the Variables
  list, a second press closing it);
  `TestVariablesTableRowSelectsItsTriggerDockedAndFloating` (right panel
  docked and floating: a title drag moves it, it reopens at the dragged
  corner, a row press shows the same trigger window as `chooseListed`, the
  unused variable's row selects that variable, an edit shows on the next
  read); `TestVariablesTablePaintsEachCellInItsColumn` (every glyph of a cell
  clipped to its column, each head and each cell painted, a rule between
  columns); `TestVariablesTableReadsAPairFromTheCheckSide` (a constant-left pair
  for each code 0..5 shows the mirrored comparison on the variable, and a
  variable-against-variable pair shows its right variable the mirror of its
  left; the editor reads a pair no other way; `inspectScript` is in
  `pkg/game`, which `pkg/ui` cannot import, so the expected words are
  written out in the test); `TestVariablesTableOfAMapWithNoVariableSaysSo` (no variable: no
  row and the note); `TestListColumnStopsPartTheWidth`.
- `pkg/ui/mapeditormenus_test.go`: `menuRowKeys["window"]` gains
  `variables`, because the Window menu has a fifth row. No golden changed.
- Focused runs on the candidate: `./pkg/ui -run
  'Variable|Script|Trigger|Wizard|Menu|Toolbar|Popup|Panel'` ok;
  `./pkg/game -run 'Variable|Scripts|Trigger|Wizard' -skip '^TestRelease'`
  ok; `./pkg/words -run 'Russian|Tables|Shipped|EveryUsedID'` fails only on
  `TestEveryUsedIDIsInTheEnglishTable`, with the fifteen
  `editor.unit.groups.*` ids that fail identically on a `git archive`
  extract of the base; `^TestLiveTreeClean$` ok.
- `TestMapEditorModesHoldEveryKeyAndGesture` by `-run`, ordinary mode: it
  fails at the base extract and on the candidate with the same 27 rows. The
  table is not in the modes table.
- `internal/storyguard/baseline.go`: CommentBytes 10570526 to 10578159, the
  doc comments of the new code, named in its note.
- Release witnesses for the seat (EN and RU installs), not run in the lane:
  `TestReleaseMapEditorScriptVariablesSaveUnchanged`,
  `TestReleaseMapEditorScriptsListUntouchedBytes`,
  `TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem` in `pkg/game`.

## Open debt

- A check that reads a variable on both sides of a pair (two variables
  compared) gives two rows, one per register, each naming the other check's
  title as its value.
- A variable a node addresses through a slot no trigger names (a node in no
  pair and no action slot) has a row with no trigger, as an unused label
  has.
- The table has no sort or filter control; its order is fixed.
- Esc returns the selection the editor had when the table opened, as for a
  records list.

## Correction

- The comparison of a check on the right of a pair is mirrored (`variableUseWords`
  over `mirroredCmps`), so `10 > variable 20` shows "is less than" and the
  second row of `variable 20 < variable 21` shows "is greater than".
- A new trigger created while the table is open closes the table and selects
  the new trigger, as for any filtered list that does not hold the new record
  (`selectNewest`). Keeping the table open needs a table-aware selection
  outside `chooseListed`, `selectNewest` and the popup builder; left as is.
