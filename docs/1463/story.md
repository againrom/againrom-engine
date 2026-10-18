# Script parameter choices

## Intent and authority

The node popup and the E prompt use one host choice builder for unit, group,
player, structure, item and variable parameters. The owner adopted item 2 of
the editor scripts census. The editor interface is owner-directed.

`TRIG-PARAM-030` (High for reference dispatch and item offset),
`TRIG-ITEMBIND-121` (High for the low-word item transform), and
`MISSION-SLOT-008` (Medium for the shared variable/check register array)
bound the wire interpretation. No original-editor interaction is inferred
from these claims.

`TRIG-STORE-002` (High) bounds the register file to 100 slots.
`TRIG-COND-003` assigns Constants slots, `TRIG-ACT-004` identifies the
variable writes, and `TRIG-CMP-006` compares shared slots. `SAV-711` bounds
the highest compiled slot by the authored condition count even when a check
is rejected. Its complete LOAD execution remains Unknown. The partial
retraction of `TRIG-COND-003` concerns check 22 movement, not Constant slots.

## As built

The host builds named choices from map records and the installed item catalog.
Unit and structure identities must be positive and unique. Unit ids below
10001 open placed unit records. Hero ids 10001..11000 use the existing
"hero N" word and have no Open; referenced higher ids also have fixed rows.
Items use the installed code minus 0xe18 modulo 65536 and have no record popup.
An item row shows that parameter value and the item name, without the catalog
code. Player rows show "Player 1: Self" with one number. The host retains the
record name for both choice formatting and trigger summaries; summaries keep
the pre-existing kind, number, name and cell format.

Each resolvable object parameter has an Open link. Groups open the existing
member popup. Membership remains edited on the unit. A stack returns through
record popups with Back. Navigation with unapplied changes asks for Apply or
Cancel. A fresh selection clears the stack. The current number remains a
choice when no named record resolves it.

Apply uses the existing atomic field set. Picking changes the popup buffer.
Cancel discards it. The E prompt writes through the existing field setter.
No ALM layout, simulation state or game-save field changes.

Variable choices use the register reader landed in `a475071c`. The host also
supplies the slot's meaning for a changed operation. Variables use their
Constant's label or Variable N. New variable selects the lowest register in
0..99 that no node addresses and that is outside every authored condition
position, including Constants. With no free register there is no New variable
row. Existing Constant and used-register rows remain available. Picking writes no
Constant or map byte; Apply changes only the node parameter. Open selects the
same virtual variable row as the Scripts list, including its Constant link.

## Proof

`TestScriptParameterChoicesAndTypedBytes` checks each host choice population,
shared E labels, pick-before-Apply isolation, identical picked/typed bytes and
Undo. `TestScriptParameterOpenBackStack` drives two nested Opens and two Backs.
`TestScriptParameterOpenKindsAndCancel` covers unit, group, player and structure
destinations. `TestScriptParameterUnknownValueAndHeroHaveNoOpen` covers hero and
item navigation limits and unchanged unresolved values.

`TestScriptVariableChoicesAndTypedBytes` checks the shared variable choices
and wire equality. `TestScriptNewVariableAndOpenStack` checks New/Cancel and
navigation through the virtual variable row and its Constant.
`TestScriptNewVariableSkipsConditionSlots` checks Constant-only populations of
3, 99 and 100 conditions, including omission when all slots are occupied.
`TestScriptParameterSummaryUsesRecordNames` and
`TestTriggerPopupReadsAsTheRule` compare whole summary lines.
`TestScriptParameterChoicesAndTypedBytes` compares whole choice labels and
uses catalog-code-prefixed item fixtures.
`TestEditorLargeGroupKeepsBackInsidePopup` checks Back bounds with 32 members
at window heights 800 and 480; the button row is included in the row budget.
`TestScriptVariablePickerUsesHostRegisterSlot` moves the register after a unit
reference and checks that the picker edits the host reader's slot.
`TestScriptUnknownTagDoesNotBecomeVariable` keeps raw ALM tag 10 numeric,
without a link, both with the host catalog and without one. The view-only
variable kind never assigns meaning to an unknown wire tag.
The trigger summary also keeps its raw number, even when a variable has that
same value.

Installed current-value picks cover 28 maps per root. Each map also logs its
New variable register and checks that it is unaddressed, below 100 and outside
the condition positions. Every item row's displayed number equals its value.
EN counts are U 431,
G 267, P 170, S 16, V 111, I 80. RU counts are U 430, G 267, P 168, S 16,
V 111, I 80. Save As bytes equal each source after all its current picks.
On `scenario/60.alm`, both roots offer U 1084 (including 1000 fixed hero
rows), G 35, P 6, S 39, V 9 (including New variable), I 416. Changed picks
save and reopen for every kind; the compiled node retains the selected
reference or register, including the item's low-word offset transform.

Builder-arm loss controls on reviewed head `7f8e0659` remove one arm at a
time. U/G/P/S/I each fail their
subtest of `TestScriptParameterChoicesAndTypedBytes`; V fails
`TestScriptVariableChoicesAndTypedBytes`. Their receipts remain in the notes.

Correction loss controls restore Constant admission, catalog-code labels and
full choice labels as summary names separately. They fail
`TestScriptNewVariableSkipsConditionSlots`,
`TestScriptParameterChoicesAndTypedBytes/item`, and
`TestScriptParameterSummaryUsesRecordNames` respectively. The player-label
control fails `TestScriptParameterChoicesAndTypedBytes/player`.

Installed EN/RU witnesses are
`TestReleaseMapEditorParameterCurrentPicksPreserveBytes`,
`TestReleaseMapEditorParameterChangedPicksSaveReopenCompile` and
`TestReleaseMapEditorParameterFrames`. Receipts, frames, loss-control results
and final gate results are kept outside the repository in
`review/story1463-script-params/`.

## Open debt

The correction refreshes EN/RU item and player drop-downs and trigger summaries
alongside the existing unit, variable and Open/Back frames. Inspection and
final-commit gate receipts are retained in the external notes directory.

Original-editor picker and navigation interaction remains Unknown (DIV-3625).
Message and spell parameters remain numeric by owner direction.
