# One Dialog window for plain and interactive dialogues

## Result

A map maker now edits every dialogue in one Dialog window. It opens from the Configure button of a Start dialog action
and from the mod window's Dialogues page. The lines stand in a short list on the left; the selected line is edited on
the right: speaker, text, hero variants by sex or class, and what happens after the line. A line may offer up to four
replies. The editor stores the dialogue as message pages while no line has a reply choice, and as a dialogue file when
one has, without asking.

## Authority

- Owner: «После реплики выпадающий список», the list takes at most a fifth of the window, and only one reply is open.
- Owner: texts are one field in the install language; other languages already in the file stay untouched.
- Owner: «Главное чтобы игралось на текущей игре»; the English version is low priority.

## Rules

1. The window lists the lines at no more than 20 percent of its width and shows only the selected line.
2. A new line is added with the plus row; lines are reordered by their grip or Alt+Up and Alt+Down; the first starts.
3. After the line offers Next line, Reply choice and End dialogue; a reply choice shows up to four replies.
4. Only the selected reply is open; each reply has text, transition, outcome, condition and hidden or disabled.
5. A dialogue with no reply choice is written as message pages, as before, and the map stays in the older form.
6. A reply choice writes the dialogue file and the map needs the script form; removing the last reply folds back.
7. The speaker and the hero variants of a line live in the dialogue file and in the message page tags, and survive both.
8. A hidden reply is left out when its condition fails and a disabled reply is shown greyed, as set.
9. An existing map and the mill mod open and play unchanged; Apply refuses a broken dialogue in plain words.
10. Only a file with the install's language alone folds back; with another language's text, or no reply, it stays.

## Code

| kind | its one builder | change |
|---|---|---|
| message and dialogue window | `pkg/ui/mapeditormessage.go` messageSpec | two columns, list and one line |
| dialogue rows | `pkg/ui/mapeditordialogue.go` dialogRows | after the line, replies as rows |
| popup layout | `pkg/ui/editorpopup.go` popupSpec | first column share, tall text row cap |
| dialogue file | `pkg/mod/dialoguewrite.go` WriteDialogue | speaker, hero variants, next line |
| dialogue play | `pkg/game/dialoguereplies.go` | speaker face and hero text of a node |
| dialogue host | `pkg/game/mapinspectiondialogue.go` | checks and file round trip |

## Proof

| test | rules |
|---|---|
| `TestMessageWindowListTakesAFifth` | 1 |
| `TestPlainDialogMadeInTheWindowIsWrittenAsPages` | 2, 5 |
| `TestAReplyMakesTheDialogueAFileAndRemovingItFoldsBack` | 3, 6 |
| `TestDialogueWindowOffersNoFifthReply` | 3 |
| `TestMillDialogueBuiltThroughTheEditor` | 4 |
| `TestSpeakerAndHeroVariantsSurviveBothStores` | 7 |
| `TestDialogueReplyConditionIsWrittenHiddenOrDisabledAsSet` | 8 |
| `TestFailingConditionHidesOrGreysTheReplyAsItsFieldSays` | 8 |
| `TestMillDialogueOpensAndWritesUnchanged` | 9 |
| `TestDialogueWindowRefusesAReplyChoiceWithNoReply` | 9 |
| `TestDialogueFileWithOtherLanguageSurvivesRemovingEveryReply` | 10 |
| `TestHandWrittenReplylessDialogueFileSurvivesApply` | 10 |

Release tests at the release: `TestMillModPlaysOnBothLanguages`, `TestMillModSecondReplyPaysGoldOnce`.

## Open debt

- An existing dialogue file cannot be picked from a trigger; the owner decides how a trigger names a file.
