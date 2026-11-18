# The editor's message and variable windows on the one window framework

## Result

The map editor's message window and variable window are now laid out by the one window builder in English and
Russian. The message window is as wide as its messages ask, every message's text shows whole and a long dialog pages.
The x and + buttons keep their marks and say in a tip what they do, and a hero whose dialog ends shows no speaker and
no text box. The variable window names the variable in its title, shows the shared register as a link with its own
(?), and says at the bottom when no trigger reads or changes the variable.

## Authority

- Owner: the T4 answers on the message and variable window mockup, binding; question 3 answered (b), the others (a).
- Owner: «The map editor targets screens of 1920x1080 and above»; no layout or test for a smaller window.

## Rules

1. The message window takes its natural width from its parts, within the screen less 16; no fixed 660.
2. Every message's text shows whole or scrolls in its field by wheel and bar; the window stays in the screen.
3. A message's x and the page's + keep their marks; each says in its tip what it does, from the words table.
4. The variable window's title names the variable; a shared register is a link row with its own (?).
5. A variable no trigger reads or changes says so in a note at the bottom of its window, not in a row.
6. One term in every editor word: a message is "message" («сообщение»), the page a "dialog" («диалог»); no card.
7. A cell where the hero's dialog ends shows its hero and «Нет варианта», no speaker, no text box, no typed text.
8. Each window lies inside a 1920x1080 screen, its labels whole and its parts hit as laid out.

## Code

| kind | its one builder | change |
|---|---|---|
| object window layout | `pkg/ui/editorpopup.go` layoutPopup | a Whole multi-line row grows to its text or the room left, then scrolls |
| window natural width | `pkg/ui/editorpopupnatural.go` naturalWidth | the width a window's rows, title, buttons and notes ask |
| note lines | `pkg/ui/linebreak.go` wrapNotes | the question, message and variable notes break here |
| message window | `pkg/ui/mapeditormessage.go` messageSpec | natural width, whole texts, tips, the end cell |
| variable window | `pkg/ui/mapeditorscriptauthor.go` variableSpec | named title, shared link row, unused note |
| object window paging | `pkg/ui/mapeditorwindow.go` pageObjectRows | one paging rule for the record, message and variable windows |
| row button tip | `pkg/ui/editorpopup.go` popupTipAt | a row button's tip from the row's ButtonTips |
| words | `pkg/words/text/{en,ru}/strings.toml` | tips, marks, helps; message and dialog in place of card |

## Proof

| test | rules |
|---|---|
| `TestMessageWindowTakesItsNaturalWidth` | 1 |
| `TestMessageWindowShowsEveryTextWhole`, `TestMessageWindowLongTextsStayInsideAndScroll`, `TestMessageWindowTextBarScrollsTheText` | 2, 8 |
| `TestMessageWindowButtonsSayWhatTheyDo` | 3 |
| `TestScriptsListVariableLinksAndSharedRegister`, `TestVariableWindowHoldsTheLayoutProperties` | 4, 8 |
| `TestVariableWindowSaysUnusedInANote` | 5 |
| `TestMessageWordsSayMessageAndDialog` | 6 |
| `TestMessageWindowEndCellHasNoSpeakerAndNoText`, `TestMessageWindowTypesOnlyIntoShownText` | 7 |
| `TestMessageWindowHoldsTheLayoutProperties` | 8 |

Release tests at the release: none.

## Open debt

- An end cell has no text box, so the editor no longer offers to type a variant for that hero; the owner decides.
- The question windows keep their own row measure beside naturalWidth; folding it in waits for a later framework task.
