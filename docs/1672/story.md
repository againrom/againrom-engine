# Guest characters made in the mod window

## Result

The mod window has a Characters page. A modder chooses a game character to start from,
types the new character's English and Russian names, and chooses a face. The page also
changes a game character's name or face. Missions offer a new character immediately;
their guest members keep their own levels and equipment.

## Authority

- Owner: "Guest characters made in W"; editor behaviour is owner directed.
- Owner: explanations belong to field hints, labels use plain words, and empty pages show one first step.

## Rules

1. Characters is a page of the mod window in English and Russian, with a name and face for each character.
2. Adding a character copies a chosen game character's appearance and class and creates its name in EN and RU.
3. A game character's name or face is changed through the character edit form.
4. Delete removes a character unless a mod mission uses it as a guest; the refusal names that mission.
5. Every edit is one Undo step and stays in the draft until the project's save writes it.
6. Missions offer new characters immediately and build each guest at the level and with the equipment its mission sets.
7. The title and every field and button explain their purpose behind a (?) mark in English and Russian.
8. Labels use plain words; an empty page shows only Add character.

## Code

| kind | its one builder | change |
|---|---|---|
| characters file | `DataKind.Write` / `writeCharacters` / `writeRows` | source-preserving writer beside `ParseCharacters`; row identity by source line from `numberRows`, shared with the items writer |
| mod window page | `modScopePages`, `workspaceScopeRows`, popup rows | Characters scope and fields |
| popup help | `modPageWords`, `objectFieldRows` | Characters title help and field meanings use the shared (?) marks |
| game character choices | `identityChoices` | people from the existing identity list |
| faces | `figureFaceChoices`, `composeFigure` | existing face choices shared with the page |
| generated text and strings | `makeTextKey`, `newModText`, `kindFiles.putTexts` | names through the shared strings writer; a rename writes only the typed language; new keys from `mod.RecordKey` |
| draft, Undo and project files | `editKind`, `changeWorkspace`, project writer | characters drafted and admitted with text |
| guest party | `guestParty` | unchanged; receives the editor's character through the draft context |

## Proof

| test | rules |
|---|---|
| `TestCharactersWriterRoundTripsAndPreservesSource` | 2, 3, 5 |
| `TestModWindowShowsCharactersThroughTheScopeTable` | 1 |
| `TestModCharactersPageWritesNamesFacesAndGameEdits` | 1, 2, 3, 5 |
| `TestModCharactersPageNameKeepsOtherTextUses` | 2, 3, 5 |
| `TestModCharactersPageDeleteRefusalUndoAndInvalidEdits` | 4, 5 |
| `TestModCharactersPageGameFaceEditKeepsTheGameName` | 3, 5 |
| `TestModCharactersPageGameRenameChangesOnlyTheTypedLanguage`, `TestModCharactersPageNewCharactersTakeTheRecordKeyRule` | 3 |
| `TestCharactersWriterDeleteOfTheFirstRepeatKeepsTheOthersLines`, `TestItemsWriterDeleteOfTheFirstRepeatKeepsTheOthersLines`, `TestOneRecordKeyRuleMakesTwoDistinctKeysForOneStem` | 2, 4 |
| `TestModCharactersPageGuestsBuildAtMissionLevelsWithEquipment` | 6 |
| `TestModCharactersPageFieldsHavePlainLabelsAndHints` | 7, 8 |
| `TestModPagesSayWhatTheyAreForBehindTheTitleMark`, `TestModCharactersPageTitleAndRowsHaveHelp` | 7 |

Release tests at the release: `TestReleaseGuestPartyBuildsCharactersAtTheirLevels`,
with an editor-made character on rom1-en and rom1-ru. Map editor modes walk is owed at landing.

## Open debt

- The seat runs the fast gate, publishes the candidate and obtains its one fresh review.
