# Arms and drop navigation

## Intent and authority

The owner requested G panel links to the scripts already edited by the arm and
drop routes. The editor is authored here. The ALM layout stays unchanged.
`TRIG-DROP-013` identifies action 0x10002 as the drop table holder; its amended
fallback and array-index clauses do not define an editor gesture. Knowledge pin
k326 remains unchanged. DIV-3657 records the interface Unknown.

## Behaviour

G lists Win, Lose and Protect arms, their existing add rows, and each drop action's
cell. Selecting an arm keeps its condition selected and shows the same node
popup as the base revision. Open beside an existing arm, or E on the selected row, opens its trigger
through openObjectRecord.
An independent Protect condition opens its condition popup. Drop opens its action
popup. The trigger and node builders are unchanged. Back uses objectBack and
restores the row's own record selection, popup visibility, popup scroll and list
scroll. No opened trigger remains selected or highlighted behind G. After Apply,
Back rebuilds the panel from the
current document. Nested node navigation uses the same stack.

Enter picks a unit or structure for an arm or add row. Enter on a drop row opens
its action. D still picks the drop cell. The host's Goals population and ordinal
mapping are unchanged; DropGoals appends display rows only in the UI. No model
writer, simulation, save format, version or asset changes.

## Proof

`TestArmsPointToTheirScriptRecord` checks two Win links, Lose and independent
Protect. The same named cases fail with the base host source because their links
are absent. `TestArmOpenAndBackPreservePanel` checks each arm kind and Drop at row
25 of a scrolled panel. `TestArmOpenLossControls` rejects missing and wrong links.
`TestArmPopupAndScriptsEditHaveSameBytesAndUndo` changes Fire once through both
popup entry routes, compares bytes, returns after Apply and undoes the edit.

`TestArmRowPopupMatchesNodeSelection` checks all six popup lines and the selected
condition for Win and Protect against direct node selection, matching the base
review witnesses. `TestArmOpenBackDeleteMatchesRow` checks the same popup and
selection after Back, Win deletion refusal and Protect condition deletion, with
identical bytes and messages before Open and after Back.
`TestArmBackClearsTriggerAreaHighlight` compares area lines before Open and after
Back. `TestArmBackRestoresPopupState` checks visible and hidden popup state and
popup scroll. `TestArmBackRestoresManualScrollAfterPanelRebuild` places row 25 at
the top of a 60-row panel, closes G while the trigger is open, and checks its
rectangle after Back rebuilds G. This manual position differs from selection-follow.
Temporary overlays restoring R1's hidden flags, R2's old Back branch, and a no-op
scroll restore each fail their named regression tests. The overlays are absent
from the acceptance gates; their logs are recorded in PROGRESS.md.

`TestArmRoutesByteHashes` pins the base host's SHA-256 outputs for the synthetic
goalMap fixture, and compares Enter/click and D/click against those host bytes:

| Route | SHA-256 |
|---|---|
| Retarget Win to unit 13 | 4cd407eb0f2d0dca0f91998e8e45ac5ba8606a54e359c0b14754a60ef689af4d |
| Add Protect on unit 10 | ae7c136581a8ad208f5a91edb8c434fbe006a975ab051938975cfb1ec9f8c895 |
| Move Drop to 8,9 | 15a378fbd7aa98b907f9626355dd33c80ca346a41a1be55304faef4bfb944756 |

`TestReleaseMapEditorArmsOpenAndKeepBytes` independently enumerates links from
installed ALM triggers and conditions, opens every arm and drop holder, checks
Back and saves byte-equal. It logs per-root populations.
`TestReleaseMapEditorArmsFrames` renders the G panel and an opened trigger on
campaign map 131, using installed EN and RU resources. Both are in the gated-test
manifest. Fixture proofs do not establish install validation.

## Evidence and open debt

Gate receipts, per-root populations, loss controls and reviewed frames are kept in
the seat's `review/story1467-arms-open/` directory. Final results are recorded in
PROGRESS.md there against the candidate SHA. The GPU window witness
TestReleaseActivePauseRenderedWorld is excluded under the owner's desktop fence.
That exclusion is not a GPU presentation pass. No push or landing is performed
from this worktree.

The editor divergence ledger was split by the trigger-area subsystem to make room
for DIV-3657. DIV-3605's row text is unchanged in ui-map-editor-areas.md.

The non-returning review notes remain bounded debt: Drop rows display full raw
coordinates (the game uses low bytes), shared arm conditions link to the first
trigger, and add-row translation matches host text. Neither installed root has
a Drop coordinate above 255. Independent link checks in the review supplement
the release test's mirrored arm-enumeration rule.
