# A dialogue can be an interactive conversation read from a file

## Result

A modder can make a trigger's dialogue interactive by naming a dialogue file beside the map. The file holds nodes of
text, each with up to four replies, and the words come in each language side by side. The player sees the replies as
numbered rows in place of the one button and picks one by number key, by the arrow keys and Enter, or by a click. A
pick can change gold or a variable, can lead to another node, and a reply marked once is not offered again, also after
a save and a load. A reply whose condition fails is left out or shown greyed, as its file says.

## Authority

- Owner: «interactive dialogues from a TOML file»: the trigger's dialogue instant chooses legacy or interactive, the map
  is then `.almx`, saves stay `.sav`, conditions and outcomes use the triggers' one evaluator and one executor.
- DIV-4998: the interactive dialogue is Againrom's own; the original shows a part with its one button.

## Rules

1. A dialogue file reads to its nodes and replies; a malformed file is refused with its file name and line.
2. A legacy dialogue reads, shows and hashes as before.
3. An interactive dialogue shows the replies its node offers; a failing condition hides or greys it as its field says.
4. Keys 1 to N, Down then Enter, and a click pick a reply; a greyed reply is not choosable; Escape does nothing.
5. A pick runs its outcome through one run-instant command, then pages on or closes; a replay hashes alike.
6. A once reply is not offered again, also after SAVE and a cold LOAD.
7. A map with an interactive dialogue cannot fold to `.alm`; its save is a `.sav`.
8. No save is possible while replies show.
9. A node or reply with no text in the game's language shows the other language's text; the file is not refused.

## Code

| kind | its one builder | change |
|---|---|---|
| dialogue panel | `Dialog.WithReplies`, `layoutNoticeBox` (`pkg/ui`) | reply rows and a greyed flag are data of the one panel |
| mod file parser | `mod.ParseDialogue` over the mod's one TOML parser | one more file kind in the mission tree |
| condition evaluator | `World.pairHolds`, shared by `ConditionHolds` and `triggerHolds` | one function, a reply asks it |
| action executor | `World.runInstant` through `runScriptInstant` | a reply runs the trigger's compiled instants |
| player command | `sim.RunInstant`, `sim.SetVariable`, `mapWorld.pending` | one new kind; the once mark is a variable write |
| `.almx` switch | `mapedit.FoldScript` refusing `LimitDialogue` | an interactive node keeps the map extended |
| SAV leaf and save gate | the extended state leaf, `flow.savePoint` | unchanged; marks ride the leaf, replies close the gate |
| game branch | none | no new check of the game's identity |

## Proof

| test | rules |
|---|---|
| `TestParseDialogueReadsTheFixtureToItsNodesAndReplies`, `TestParseDialogueRefusalNamesFileAndLine` | 1 |
| `TestMissionReadsTheDialogueFilesBesideItsMap`, `TestMissionRefusesAMalformedDialogueFileByFileAndLine` | 1 |
| `TestLegacyDialogueKeepsItsWorldHash` | 2 |
| `TestInteractiveDialogueShowsItsOfferedReplies`, `TestFailingConditionHidesOrGreysTheReplyAsItsFieldSays` | 3 |
| `TestReplyKeyChoosesThatReply`, `TestReplyClickNeedsPressAndReleaseOnTheSameRow`, `TestDisabledReplyIsNotChoosableByKeyOrClick` | 4 |
| `TestChoosingAReplyRunsItsOutcomeThroughTheCommand`, `TestRunInstantCommandRunsTheCompiledAction` | 5 |
| `TestReplayOfTheChoiceGivesTheSameWorldHash` | 5 |
| `TestOnceReplyIsNotOfferedAgainAndNextPagesTheNode`, `TestOnceReplyStaysUsedAfterSaveAndColdLoad` | 6 |
| `TestFoldScriptRefusesAnInteractiveDialogue`, `TestInteractiveMessageRaisesItsDialogue` | 7 |
| `TestSaveGateClosesWhileRepliesShow` | 8 |
| `TestMissingLanguageShowsTheOtherLanguageWithoutRefusingTheFile` | 9 |

Release tests at the release: none.

## Open debt

- The editor's dialogue window and the fixture mod are later stories; town talk shows no replies.
- The second game cannot carry the extended record, so only the first game raises an interactive dialogue.
- The world stops while a dialogue shows: a condition reads the last pass and an outcome lands on the next step.
