# The map editor edits mission messages one way in both games

## Result

The map editor lists, edits, checks and saves the mission messages of both games the same way. The pages window of a
first-game map now lists the map's pages with the actions that show each, as the second game's lists its sections.
Both games refuse a letter the game cannot show or would show as another, and a NUL, in the same words. A
second-game action that shows a message now offers New dialog, which adds the next free event section, points the
action at it and opens its text. Saving gives the same files as before.

## Authority

- Owner: «Как рекомендуешь» on the architecture audit at `7dfe5ea99`, row 1: mission texts one path in both games.
- R2-ENGINE-049 (Medium): the campaign reads the mission text file numbered by the scenario location.
- R2-ENGINE-050 (Medium): an ordinary event selects its section by `event%d` through a first-substring lookup;
  IDs 250, 253, 254 and 255 take other branches.

## Rules

1. The pages window lists the map's pages in both games, each with the actions that show it, and its number opens it.
2. Both games refuse a letter the font cannot show, a letter the text file's serve shows as another, and a NUL, alike.
3. New dialog adds a message at the first free number, points the action at it, and one undo takes back both.
4. A second-game new message is a section at the text's end; 250, 253, 254, 255 and shadowed keys are skipped.
5. A second-game new message opens the pages window at its text; a first-game one opens the message window.
6. An edited text saves to the same files as before; a text not edited since the open writes no text file.
7. New dialog is refused on an unreadable mission text; a copied trigger's page keeps its source bytes.

## Code

| kind | its one builder | change |
|---|---|---|
| map format edit row | `pkg/game/mapeditrow.go` editRows | each format's `messageStore` row replaces `eventText` |
| workspace text fields | `pkg/game/mapmessagestore.go` pageFields, setPageField | one body over page files or sections |
| message targets | `pkg/game/mapmessagestore.go` messageTargets | one list; New dialog in both games |
| mission text check | `pkg/game/mapmessagestore.go` checkPageText, withPage | one check and one write |
| new message | `pkg/game/mapmessagestore.go` addMessage, freeMessage | the store's number and fresh text |
| mission text save | `pkg/game/mapedittestplay_files.go` writeEditedMapPages, writeMapInMod | `mapCompanions` carry the texts |
| install text source rows | `pkg/game/installtext.go` textSource rows | the editor reads `missionTextSource` |
| message open and create | `pkg/ui/mapeditormessage.go` newMessage | a host with no message window opens the page |

## Proof

| test | rules |
|---|---|
| `TestFirstMapPagesWindowListsItsPagesAndTheirReaders`, `TestSecondMapEditorShowsAndEditsTheMissionTextSections` | 1 |
| `TestBothMapEditorsRefuseALetterTheFontCannotShowAlike` | 2 |
| `TestSecondMapEditorCreatesAMessage`, `TestMessageNewPageBytesAndHistory`, `TestMessageWizardAndRefusal` | 3, 4, 6 |
| `TestNewMessageWithoutMessageWindowOpensItsPage`, `TestPageTargetOpensThePagesPromptAtItsText` | 5 |
| `TestMessagePopupNewApplyCancelAndBack`, `TestMessageOutOfOrderPopupCancel` | 5 |
| `TestSecondMapEditedMissionTextSavesIntoTheMod`, `TestSecondMapEditedMissionTextFollowsTheTargetMod` | 6 |
| `TestSecondMapEditorNewDialogNeedsAReadableMissionText`, `TestTriggerCopyKeepsTheBytesOfACodePagePage` | 7 |

Release tests at the release: `TestReleaseSecondMapEditorOpensEveryMissionText`,
`TestReleaseSecondMapEditedEventTextShowsInTheGame`, `TestReleaseSecondMapEditorRefusesTextTheGameShowsAsOtherLetters`,
`TestReleaseTriggerMessageEditsAndUndo`.

Review report: review/story1778-mission-texts-one-path/RESULT.md.

## Open debt

- The first game's text code keeps the stored bytes, so its serve check finds no letter there today.
- The second game has no message window; its messages are edited in the pages window.
