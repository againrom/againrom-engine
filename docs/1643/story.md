# The map editor copies a trigger

## Result

A map maker copies the selected trigger with one press: the trigger window's Copy, or Copy trigger in the Tools
menu. The copy stands right after the original, carries the original's name with a copy mark, and is selected.
It has conditions and actions of its own with the original's settings, so changing it later leaves the original
as it was. Each message the copy shows is a page of its own with the original's text. One Undo removes the copy
and its pages; Redo makes the same copy again. A trigger with a drop zone is not copied.

## Authority

- Owner: «добавить возможность копировать триггер -- именно просто создать копию, далее можно его поменять как
  обычный триггер. это удобство».
- Owner: a copied trigger's message gets a text page of its own; a trigger with a drop point is not copied.
- No original behaviour is involved; the map format is unchanged and no DIV row is needed.

## Rules

1. When a trigger is selected, Copy, Tools Copy trigger and the command copy_trigger make one copy, selected.
2. The copy has the original's pairs, comparisons, groups, fire-once flag and node settings on new node IDs.
3. The copy's name is the original's with the copy mark of the game's language, cut to fit the name field.
4. Editing the copy's conditions or actions leaves the original's trigger, conditions and actions byte for byte.
5. The copy is one Undo step back to the bytes before it; Redo writes the same bytes as the first copy.
6. The copy stands right after the original; later triggers move one index on with their bytes unchanged.
7. A map saved after a copy reopens to the same bytes, and in a mission the copy fires as the original does.
8. With no trigger selected, Copy trigger says to select one and changes nothing; with no map its row is off.
9. An original's slot word outside the comparison alphabet or the ID cap is in the copy as it is.
10. A node at ID 5000, a 64-byte label or an ID two nodes carry refuses the copy in EN and RU words, writing nothing.
11. A name that already ends with the copy mark keeps one mark.
12. Each message action of the copy shows a new page holding the text of the original's page at the copy.
13. A text written to either page later leaves the other; Undo removes the copy's pages; a saved map reopens with both.
14. A trigger with a drop action is refused in EN and RU words; the map, its pages and the history stay as they were.

## Code

| kind | its one builder | change |
|---|---|---|
| trigger record | `mapedit.Editor.NewTrigger` (`NewScriptTrigger`, extended encoder) | `CopyTrigger` asks it with the original's record, whose words stand unchecked as in `SetTriggerSlots` |
| condition and action node | `mapedit.NewScriptNode`, IDs from `NextScriptID` | `copyNodes` asks them |
| trigger order | `mapedit.Editor.Reorder`, the trigger move | the copy moves from the end to after the original |
| history, message page | `Mark`/`Squash`, `change` and its stage; `addMessagePage`, which `pointMessage` asks | one step; a page per message |
| command | `editorCommandRows` | `copy_trigger` row |
| key and Tools menu row | `editorKeyRows`, `toolButtons` | `copytrigger` rows |
| trigger window buttons | details and wizard button rows | Copy before Back |
| words | `e.word` word book | ten keys, EN and RU |

## Proof

| test | rules |
|---|---|
| `TestTriggerCopyStandsAfterTheOriginalWithNodesOfItsOwn` | 1, 2, 3, 5, 6 |
| `TestTriggerCopyFiresAsTheOriginalAndEditsAlone` | 4, 7 |
| `TestTriggerCopyKeepsAnExtendedTriggersPairs` | 2, 6 |
| `TestTriggerCopyNameFitsTheNameField` | 3, 11 |
| `TestTriggerCopyKeepsWordsOutsideTheAlphabet` | 5, 9 |
| `TestTriggerCopyRefusesInTheWordBook`, `TestTriggerCopyRefusesADropZone` | 10, 14 |
| `TestTriggerCopyShowsAPageOfItsOwn`, `TestTriggerCopyGivesEachMessageActionAPage`, `TestTriggerCopyPageReachesTheModsFolder` | 12, 13 |
| `TestMapEditorCommandsMatchTheWindowRoutes` (copy_trigger) | 1, 5 |
| `TestMapEditorCopyTriggerIsOneCommandRow` | 1, 3, 8 |
| `TestMapEditorModesHoldEveryKeyAndGesture` (four copytrigger paths) | 8 |
| `TestMapEditorMenuRowPressDoesWhatItsButtonDoes` | 8 |
| `TestEditorEditsGoThroughTheCommandTable` | 1 |

Release tests at the release: `TestReleaseMapEditorScriptsListUntouchedBytes`, `TestReleaseMapEditorScriptsListFrames`,
`TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`, `TestReleaseMapEditorTriggerAreaCensus`,
`TestReleaseMapEditorTriggerFrames`, `TestReleaseTriggerWizardOpensShippedTriggers`, `TestReleaseTriggerWizardFrames`,
`TestReleaseTriggerWizardMapPlaysTheTrigger`, `TestReleaseEditorCancelPreservesInstalledMapBytes`,
`TestReleaseMapEditorArmsOpenAndKeepBytes`. The frame tests draw the trigger window, which now has a Copy button.

## Open debt

- The copy's nodes hold 0 in the word after the ID, as `NewScriptNode` and the game's node constructor write it.
  The game reads that word for conditions (ALM-NODEREAD-232); in the installed maps 0 of 1477 EN and 0 of 1470 RU
  nodes hold another value. Slot names and trigger bytes +0x40..+0x7f have no reader in the game
  (ALM-NODEREAD-232, ALM-TRIGREAD-233) and are not carried. Whether the original map editor shows slot names is
  Unknown.
