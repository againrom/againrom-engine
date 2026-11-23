# A dialogue reply holds a trigger's condition and action lists, edited row by row

## Result

In the first game's map editor, a dialogue reply now takes as many conditions and actions as a trigger does. The
reply is offered only when every one of its conditions holds, and when the player picks it, its actions run in the
order the map maker set. The Dialog window shows every condition and every action of the open reply, and each row can
be added, removed or moved with Alt and the arrow keys. A long reply scrolls inside the window on a full HD screen.

## Authority

- Owner: «надо сделать неограниченное количество действий в рамках выбора варианта реплики (не менее того сколько
  можно затолкать в триггер)».
- Owner: «хотелось бы по сути функционал триггера прямо в диалог воткнуть. то есть прямо даем настраивать условия
  (давай без ИЛИ пока что), настраивать действия». This replaces the ruling «a reply has one condition and one outcome».
- Limit: the extended trigger's own limits, 64 conditions and 64 actions, from the one script limits table (seat).

## Rules

1. A reply holds at most 64 conditions and 64 actions; a file with a 65th of either is refused at that line.
2. Every dialogue file reads as before and writes back byte-equal when unedited; a long list may wrap over lines.
3. A reply is offered only when all its conditions hold (no OR); picked, its actions run in order, then the once mark.
4. The Dialog window shows every condition row and every action row of the open reply.
5. Add condition, Add action, each row's Remove and Alt+Up/Alt+Down edit the rows; a 65th row is refused with a note.
6. The window's right column scrolls apart from the list when its rows pass the height; the window fits 1920x1080.
7. In any data file a wrapped list saves as one pair: edited it is replaced whole, dropped it leaves no line.

## Code

| kind | its one builder | change |
|---|---|---|
| script limits | `pkg/scriptprofile/profile.go` `Extended` | the reply's bounds; `MaxDialogueWhen`, `MaxDialogueDo` gone |
| dialogue file reader | `pkg/mod/dialogue.go` `readDialogueReply`, `dialogueList` | bounds from `scriptprofile.Extended` |
| TOML subset reader | `pkg/mod/toml.go` `parseTOMLWith` | an open array runs over the next lines (`continueList`, shared with `mod.toml`; `scanLine` is the one scanner) |
| dialogue file writer | `pkg/mod/dialoguewrite.go` `WriteDialogue` | unchanged: one line per list |
| editor host reply write | `pkg/game/mapinspectiondialogue.go` `writeDialogueInto` | refuses past `scriptprofile.Extended` |
| condition and action rows | `pkg/ui/mapeditorwizardscope.go` `wizardLists`, `wizardItemScope` | one list builder for trigger and reply |
| wizard row edits | `pkg/ui/mapeditorwizard.go` `wizardRowButton`, `moveRow` | full-list refusal; a row moves among its kind |
| Dialog window | `pkg/ui/mapeditormessage.go` `messageSpec`, `pkg/ui/screens/dialogue.json` | all rows, add rows, column top |

## Proof

| test | rules |
|---|---|
| `TestDialogueReplyHoldsAnExtendedTriggersLists` | 1, 2 |
| `TestDialogueListsRunOverLines`, `TestDialogueWrappedListsSaveAsEdited`, `TestRowWriterTreatsAWrappedListAsOnePair` | 2, 7 |
| `TestMillDialogueOpensAndWritesUnchanged`, `TestMillLongReplyRoundTripsThroughSaveMod` | 2 |
| `TestMillLongReplyRunsItsActionsInOrder`, `TestMillModPlaysOnBothLanguages` | 3 |
| `TestDialogReplyShowsAllItsRows` | 4 |
| `TestDialogReplyRowsAddRemoveMoveAndRefuse` | 5 |
| `TestDialogReplyRowsScrollInsideFullHD` | 6 |

Release tests at the release: none.

## Open debt

- The right column has no scroll bar of its own; the wheel and the focus move it (BACKLOG «Mods: global variables»).
