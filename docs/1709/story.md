# The map editor edits interactive dialogues

## Result

A mapmaker chooses, on the Start dialog action, between the old message page and a dialogue with replies. A dialogue
with replies gets a name, and a button opens its window, where the nodes, their texts in two languages and up to four
replies each are edited. A reply has a next node or an end, an only-once mark, a hidden or disabled look, one condition
and one outcome, edited with the same rows as the trigger window. The mod window lists the map's dialogues.

## Authority

- Owner: «the map editor edits interactive dialogues»: Start dialog chooses Legacy or Interactive, the map saves as
  `.almx`, the dialogue file is written once by one writer and reads back through the one loader.
- DIV-4998: the interactive dialogue is Againrom's own.

## Rules

1. Start dialog offers Legacy or Interactive; Interactive names a dialogue and opens it with New or Edit dialogue.
2. Back to Legacy leaves the name unused, and a map with nothing interactive left asks to fold to `.alm` at save.
3. The window adds, removes, selects and reorders nodes, the first node starts, and each text has two languages.
4. A reply has next node or End, once, hidden or disabled, one condition and one outcome from the wizard's own rows.
5. Apply refuses in words: a fifth reply, a missing next node, a bad node name, an incomplete outcome, a bad mark.
6. A text in one language only is allowed; a dialogue a trigger starts is not deleted.
7. Save mod refuses a dialogue that breaks a check or that a trigger starts and no file holds.
8. The mod window's Dialogues page lists dialogues with nodes and users, with Open dialogue and New dialogue.
9. Apply writes the dialogue file beside the map; the file reads back equal; Save and Publish carry it.
10. The mill mod built in the editor plays on both languages: a pick sets its variable, gold is paid once.
11. The window fits a full HD screen.

## Code

| kind | its one builder | change |
|---|---|---|
| window description | `popupSpec`, `layoutPopup`, `screens/dialogue.json` | one more window as data |
| condition and outcome rows | the wizard's `wizItems`, `wizParam`, `wizardScope` | a reply's rows, a source choice |
| mod window page | one row of `modScopePages` and `workspaceScopes` | the Dialogues page |
| dialogue file | `mod.ParseDialogue` and `mod.WriteDialogue` | the one reader and one writer |
| mod save and publish | the project writer, `missionPackageFiles` | dialogue files ride the project |
| text face | the editor's one text face | Russian text is typed only where drawn |

## Proof

| test | rules |
|---|---|
| `TestDialogueWindowOpensFromTheStartDialogRow` | 1 |
| `TestSwitchingBackToLegacyKeepsTheNameUnusedAndFoldsTheMap` | 2 |
| `TestMillDialogueBuiltThroughTheEditor` | 3, 4 |
| `TestDialogueWindowRefusesAFifthReply`, `TestDialogueWindowRefusesAMissingNextNode` | 5 |
| `TestDialogueWindowRefusesAnEmptyAndATwiceUsedNodeName`, `TestDialogueWindowRefusesAnIncompleteOutcome`, `TestDialogueRefusesAOnceMarkPastTheScriptVariables` | 5 |
| `TestDialogueWindowTakesATextInOneLanguageAndRefusesAUsedDelete` | 6 |
| `TestSaveModRefusesAnInteractiveActionWithNoDialogueFile` | 7 |
| `TestDialoguesPageListsOpensAndStartsDialogues` | 8 |
| `TestDialogueWriteReadsBackAndUndoRestoresTheMap`, `TestMillModSavesItsDialogue` | 9 |
| `TestMillModPublishCarriesTheDialogueFile` | 9 |
| `TestMillModPlaysOnBothLanguages`, `TestMillModSecondReplyPaysGoldOnce` | 10 |
| `TestDialogueWindowFitsAFullHdScreen` | 11 |

Release tests at the release: the mill mod played headless on rom1-en and rom1-ru.

## Open debt

- Russian text is shown read-only where the install's font has no Cyrillic; an own editor face waits for an owner call.
