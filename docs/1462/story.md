# One Scripts list

## Intent and authority

The bar Scripts entry and F2 open the same records list. Base:
`9e1f94387045cdda388a6b9876c688287f53d563`; the knowledge pin is unchanged.
Owner direction is census plan item 1 and owner decision 5 in
`review/editor-scripts/CENSUS.md`, corrected by owner decision 1 in
`review/story1462-scripts-list/CORRECTION-BRIEF.md` at reviewed head
`751d5a352a0cb95359e951059b6f73f590f3cb40`. The editor's layout is project policy;
the original editor is evidence of its writes only.

Claims read from the pinned snapshot:

- `MISSION-SLOT-008` (Medium): variables and compiled check results share a
  register array. A condition's authored array position gives the editor's
  register number; rejected runtime checks can shift compiled assignments.
- `ALM-NODEREAD-232` (High / Medium): a node carries a 64-byte label; the
  original loader reads it, and the binder uses it as text in failure logs.
  It does not establish the original editor's list layout.

## As built

Filter 7 is the Scripts filter; filter 9 and the Triggers bar entry are
removed. Both the bar and F2 call `showRecords`, then `listSpec` uses the
existing records list, Picker, scrolling and popup builder. No second list
window or Add flow is introduced.

The four headings always appear in this order. An empty heading says
`<heading>: none` (RU: `<heading>: нет`) on its own line.

1. Triggers, in authored order, with the trigger number and name.
2. Variables: every register addressed by action 3 or 8 or condition 19,
   in ascending register order. The row shows `Variable N` / `Переменная N`,
   plus the label of the Constant (`0x10002`) at that position when present.
   A non-Constant condition at that position is named as sharing the register.
3. Conditions no trigger names, in condition-array order, excluding the
   Constants already listed as variables.
4. Actions no trigger names, in action-array order.

The host exports the register through its existing `scriptVariable` reader:
the first int, X, Y or constant parameter, or register 0 if absent. The UI
does not decode operands. Variable rows are temporary inspection views,
including free registers with no condition record. Their read-only popup
lists the triggers that change the register and the triggers that read it,
once per trigger in each group, in authored order. Each is a `popupLink` to
the existing trigger popup. A Constant also has a link to its node popup.
Unattached addressing nodes create a variable row without inventing a trigger.
References use node IDs; variable addresses use array positions. Zero node
references are empty. No ALM record or persistent field is added.

Record rows select the same document record through `Select`; trigger and node
popups, their writers, Delete refusals, G, D and area handles are unchanged.
View > All records (RU: Вид > Все записи) exposes the existing unfiltered
records list, including raw sections and every node, preserving the former
Triggers raw-data route. Script records outside the decoded model, including
overridden duplicate section-7 records, retain generic record rows under their
Triggers, Conditions or Actions heading and remain under All records too.
An empty Scripts list retains `ScriptNotice`, including the reason that
section 7 is absent. Up/Down traverse records and variable views and skip headings. The selected trigger
remains the area selection. Reopening Scripts keeps an existing listed
selection, otherwise selects its first record. Add and A open the existing
Script wizard, with Action, Condition and Trigger kinds. Every new caption
is in `pkg/words` for EN and RU, without the unsupported Russian letter.

## Old routes

| Old route | New place |
|---|---|
| Bar Triggers, filter 7 | Bar Scripts, filter 7; Triggers heading |
| F2 mixed Scripts, filter 9 | F2 opens the same Scripts list as the bar |
| Trigger row | Triggers heading; same trigger popup |
| Addressed register, including a free register | Variables heading; read-only use popup with trigger links |
| Constant addressed as a variable | Variables popup links to the same condition popup |
| Condition/action no trigger names | Conditions/Actions heading; same node popup |
| Node named by a trigger | Trigger popup's existing pair/action Edit; View > All records exposes every record |
| Raw type-7 section in Triggers | View > All records retains every physical section, including decode failures and overridden duplicates |
| A or list Add | Script Add wizard; the bar list now offers all three script kinds |
| E on selected script record | Same raw field prompt |
| Delete on selected script record | Same writer and refusals; variable views have no writer |
| Trigger popup New, pair/action New, node Back | Same existing popup and wizard routes |
| G arms, D drop cell | Same panel and map pick |
| Map trigger area click and handles; L drawn area | Same trigger selection, handles and drawn-area flow |
| Read-only record text and links | Same inspector text and links; Scripts can also be opened read-only |

## Proof

The smallest reproduction is the new synthetic list witness run on the base:
`TestScriptsListHeadingsAndRecordOrder`,
`TestScriptsListEmptyHeadingsSayNone`,
`TestScriptsListRowsKeepTheirPopupAndSelection`, and
`TestScriptsBarAndF2OpenOneListAndAddWizard` fail on the former mixed list.
The initial empty-heading test used omission; the final policy keeps a
one-line empty heading to show every heading on installed maps.

The synthetic map has one trigger, one referenced variable Constant, one
unused condition and one unused action. The order test has a subtest for
each heading. Record popup equivalence, heading clicks, Up/Down and area
selection remain covered. `TestScriptsListVariableLinksAndSharedRegister`
checks the Constant label, a shared condition register, a free register,
read-only rows and actual link clicks to the trigger and Constant popups.
`TestScriptsListHostVariableRegisters` checks actions 3 and 8 and check 19
with packed int, X, Y and constant operands, and the absent-operand default.
`TestScriptsListNoticeAndOverriddenRecords` checks the empty-list reason
and generic rows for records outside the model.
`TestScriptsBarAndF2OpenOneListAndAddWizard` checks both entry points and A;
the existing Add tests check the list's button too.
`TestScriptsListDeleteKeepsVariableRefusal` checks that Delete in a variable
view writes nothing, then requires the existing refusal on its Constant.

Loss controls remove each of the four section populations from the production
`scriptSections` result. Each corresponding
`TestScriptsListHeadingsAndRecordOrder/<heading>` must fail; the source is
restored after each run. Receipts are in the notes directory.

The reviewed Constant-only implementation failed `TestScriptsListFreeRegister`
on register 67: it displayed Variables: none. The correction passes it.
Its loss control excludes registers above the condition count from the
production variable builder; the named test fails. The source is restored
byte-for-byte. Receipts: `correction-repro.log` and `correction-loss-free.log`.

The modal golden records 101 paths by 66 actions (6666 cells). A control
overlay gives the candidate the former Triggers bar caption while keeping
its new list. It reproduces 6347 base cells; all 319 remaining differences
are Scripts, Triggers or Script Add paths/actions. No cell outside those
routes differs with the old caption. The initial list change regenerated
the golden because the new bar caption changes the panel hash in every path.
The correction preserves that table and its generic-record routes.

Release witnesses:

- `TestReleaseMapEditorScriptsListUntouchedBytes`: every installed map is
  opened. Each expected list record is clicked through its visible list row,
  its popup is required and cancelled, Up/Down run, and Save As is byte-equal
  to independently read installed bytes. Counts come from decoded installed
  ALM records and are checked against actual row selection. Every map also
  requires the exact actual list row count, including all four headings.
- `TestReleaseMapEditorScriptsListFrames`: `scenario/20.alm`, with all four
  headings in one frame (12 triggers, no addressed register, one unused
  condition, one unused action), the trigger popup from that list and its
  Script Add wizard. Additional frames show `scenario/151.alm` registers
  59..67 and variable 59's use popup, in EN and RU. EN and RU frames are
  rendered headless; no window runs.

The existing owner, trigger and Add frame witnesses enter through the
Scripts bar item. The first trigger is row 1, after its heading; the former
duplicate Triggers/Add-list frame is removed.

| Root | Maps | Triggers | Variables | Conditions | Actions | Popups cancelled |
|---|---:|---:|---:|---:|---:|---:|
| ROM1 EN | 38 | 421 | 52 | 173 | 162 | 808 |
| ROM1 RU | 34 | 418 | 52 | 172 | 157 | 799 |

Each measured root has 17 maps with addressed registers and 52 distinct
map/register pairs. Registers 32 and 33 on `scenario/60.alm` share
non-Constant conditions; the other 50 pairs are free registers. Neither
root has an addressed Constant. Synthetic tests cover its label and link.
Every one of the 72 installed maps saves byte-equal. No map, save or ALM
layout changes.

The release tests are registered in `internal/gatedtests/testdata/population.txt`.
Final receipts, including the complete EN/RU release set and the ROM2 EN
`^TestReleaseSecond.*(Save|Load)` control, belong in
`review/story1462-scripts-list/PROGRESS.md`; owner frames and crops are indexed
there by `INDEX.md`. The version files do not change.

## Open debt

`DIV-3617` records the owner-directed list layout. The existing authoring
defaults and refusals remain `DIV-3394`; this story changes no writer.
The original editor's corresponding list layout is not established by the
two claims read. Free registers have a read-only use popup and no node
record to edit. The inspector reports authored references, not current
runtime values or whether a trigger executes. No new persistent
representation is introduced for them.
