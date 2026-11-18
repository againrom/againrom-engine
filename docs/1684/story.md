# The editor's field prompts on the one window framework

## Result

The map editor's windows that ask for one value (a record field, a map path to open or save, the new map size, a
recipe's item, a speaker's name, the forest density, a random seed) and its message window are now laid out by the
editor's one window framework in English and Russian. Each is titled by what it asks for; what to type and the keys
are behind the title's (?) mark. A field with choices shows a drop-down, a recipe's item a grid of pictures, and a long
message shows whole. What each field accepts, Enter, Esc and where the value goes stay as they were.

## Authority

- Owner: the seven T14 answers on the field-prompt mockup, binding.
- Owner: «one builder per kind»; the system file dialog stays the first route for a path.

## Rules

1. A field with choices shows a choice row: a drop-down, or a picture grid when every choice has a picture.
2. A record field's prompt is titled by the plain view's word for the field ("Gold", «Золото»).
3. When the system file dialog did not open, the typed path window opens and its note says why.
4. The save window's "Texts inside the map" is a labelled switch row with the .almx rule behind its (?).
5. The new map size is one field, 80x80; its help states sides from 32 to 256 in steps of 8.
6. The line "Map and history retained" shows only under an error that left the map as it was.
7. A message longer than six lines shows whole; the window body pages when taller than the window.
8. A prompt's title holds no key; its help ends with the keys once.
9. Every caption, instruction, button text and editor message of these windows comes from the words table.
10. Each window lies inside the screen at 640x480, 800x600 and 1920x1080 and its rectangles hit as laid out.

## Code

| kind | its one builder | change |
|---|---|---|
| prompt window | `pkg/ui/mapeditorwindow.go` promptSpecBase | title, help and rows from the prompt's words |
| prompt rows | `pkg/ui/mapeditorprompt.go` promptRows | choice row, picture grid, labelled path and switch |
| prompt drop-down | `pkg/ui/mapeditorprompt.go` promptDropdown | the area trigger's and a field's choices |
| choice list | `pkg/ui/editorchoices.go` openChoiceList | one list for the object window and the prompt |
| picture grid | `pkg/ui/editorpopup.go` popupGrid | the recipe item's pictured choices |
| message window | `pkg/ui/mapeditorwindow.go` promptSpecBase | message rows that page; kept line from keptError |
| field word | `pkg/ui/mapeditorplain.go` plainLabel | the field prompt's title |
| words | `pkg/words/text/{en,ru}/strings.toml` | prompt names and helps, sixteen editor messages |

## Proof

| test | rules |
|---|---|
| `TestFieldWithChoicesShowsAChoiceRow` | 1 |
| `TestRecordFieldPromptTitleIsThePlainWord` | 2 |
| `TestPathFallbackNoteSaysTheDialogDidNotOpen`, `TestEditorDialogFailureKeepsTypedRoute` | 3 |
| `TestSaveWindowSwitchIsALabelledRow`, `TestSaveWindowSwitchPutsTheTextsInsideTheMap` | 4 |
| `TestNewMapSizeIsOneField`, `TestNewMapSizeHelpStatesTheSideRule` | 5, 8 |
| `TestKeptLineOnlyUnderAKeptError` | 6 |
| `TestALongMessageShowsWhole` | 7 |
| `TestMapEditorModesReadableTable` | 8 |
| `TestFieldPromptBuildersHoldNoEnglishLiteral`, `TestMapEditorMessageBoxShowsTextOnceConverted` | 9 |
| `TestFieldPromptsFitTheirWindowAtThreeSizes` | 10 |

Release tests at the release: none.

## Open debt

- none
