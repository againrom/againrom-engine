# The editor's question windows on the one window framework

## Result

The map editor's question windows (leave with unsaved changes, the notice before a test play or a mod save, the new
map mode, extending and folding the map's script, the trigger for a drawn box, a shared condition, moving units to a
group, a group left empty, the test play party and deleting a player) are now laid out by one builder in English and
Russian. Each is as wide as its own text, its title names the window, and what the choice means stands behind the
title's (?) with the keys. The windows show only the lines that state or warn, in one colour per window.

## Authority

- Owner: the T13 answers on the question-window mockup, binding; question 9 answered (a): a head line per group
  and a line per trigger or action that names it.
- Owner: «The map editor targets screens of 1920x1080 and above»; no layout or test for a smaller window.

## Rules

1. A window is as wide as its text: notes read at most 60 average letters a line, within the screen less 16.
2. A window's notes take one colour: the warning colour where they warn or refuse, the caption colour where they state.
3. The keys line ends the help of a window that has help; a window without help has no (?) for the keys alone.
4. A title names the window; no title asks a question.
5. The notice window shows the warnings only; what they mean stands behind the (?).
6. The new map window's caption keeps the range: "Starting height (0-126)".
7. The fold window shows the reason the map can be saved as an original map as a state line.
8. The drawn-box window's note is the box only; what the box becomes stands behind the (?).
9. The empty-group window shows a head line per group and a line per trigger or action that names it.
10. The delete-player window names the player ("Player 2: Monsters"); the consequences stand behind the (?).
11. Every caption, note, help and button text is a word; the full-checks refusal reads Russian in the Russian editor.
12. Each window lies inside a 1920x1080 screen, its labels whole and its parts hit as laid out.

## Code

| kind | its one builder | change |
|---|---|---|
| question window | `pkg/ui/mapeditorquestion.go` questionSpec | one builder from data for all eleven windows |
| question sizing | `pkg/ui/mapeditorquestion.go` question.spec | natural width, notes wrap inside the draw inset |
| question grid | `pkg/ui/mapeditorquestion.go` sizeGrid | the party grid's sizes, used by the window and the mockup |
| description reader grid | `pkg/ui/popupdesc.go` popupChain.grid | one grid form: a flowing grid or a column per item |
| mockup page grid | `pkg/ui/popuppage.go` newEditorMockup | one grid writer; the question path calls question.spec |
| script refusal words | `pkg/game/mapinspectionwizard.go` checkWords | the full-checks refusal keyed to the words table |
| words | `pkg/words/text/{en,ru}/strings.toml` | window names, helps, the keys line, notes |

## Proof

| test | rules |
|---|---|
| `TestQuestionWidthIsItsOwnText` | 1 |
| `TestQuestionNotesTakeOneInk` | 2 |
| `TestQuestionKeysLineOnlyBehindHelp` | 3 |
| `TestQuestionTitlesNameTheWindow` | 4 |
| `TestNoticeShowsTheWarningsOnly`, `TestThePartyWindowHoldsTheNoticeLines` | 5 |
| `TestNewMapHeightCaptionKeepsTheRange` | 6 |
| `TestFoldWindowShowsTheReason`, `TestAnExtendedMapSavesAsAlmOnlyWhenItFitsAndTheQuestionAgrees` | 7 |
| `TestAreaWindowNoteIsTheBox` | 8 |
| `TestGroupWarningHasALinePerUser` | 9 |
| `TestPlayerDeleteNamesThePlayer` | 10 |
| `TestQuestionBuildersHoldNoEnglishLiteral`, `TestFormQuestionForTheChecksIsInTheEditorsLanguage` | 11 |
| `TestQuestionWindowsHoldTheLayoutProperties`, `TestQuestionStandsWholeInTheMiddle` | 12 |

Release tests at the release: none.

## Open debt

- The description reader's grid shares one form with T14; T2's grid keys wait for story 1686 to land.
