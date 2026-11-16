# Map editor windows as data, with pages both ways

## Result

The map editor's trigger window is now written down once as a plain description: its title, rows, row kinds,
words, value bindings and buttons. The editor builds the window from it and shows and behaves exactly as before
in both languages. A new tool turns the description into one self-contained web page at a chosen screen size and
language, with the window at the corner, bounds and row count the editor gives it at that size, and reads a page
back into a description. The same tool paints the window as a picture with the editor's own painter.

## Authority

- Owner: «Надо посмотреть тщательно на редактор и собрать некий UI Framework. То есть унифицировать все экраны,
  типа рулбука. Параллельно еще сделать html конверсию или типа того, чтобы собирать мокапы»; on the page
  «Если есть двусторонняя возможность давай сделаем»; scope: the map editor.
- Owner: the editor is pinned by no table, golden or walk test.

## Rules

1. A window description names row kinds and word keys, never text, and holds no map data.
2. A description read into the popup spec and written back is the same description.
3. A page is read back from its embedded description, the same one; an edit of the visible part alone is not read.
4. Every element of a page sits at the rectangle the popup layout gives it, at 640x480, 800x600 and 1920x1080.
5. The trigger window built from its description equals the one built before: rows, kinds, labels and rectangles.
6. A page carries no install byte, image or font file; its words come from the one word table.
7. The picture comes from the editor's one popup painter and paints only inside the window.
8. The window's corner, bounds and rows are the editor's (`placedCorner`, `popupBounds`, `objectRowsShown`).
9. The tool writes nowhere the engine's install write fence refuses.

## Code

| kind | its one builder | change |
|---|---|---|
| editor popup window | `pkg/ui/editorpopup.go` `layoutPopup` | unchanged; a description's `build` gives its `popupSpec` |
| window description | `pkg/ui/popupdesc.go` `popupDesc.build` | new: rows, `when` and `each` over bindings |
| trigger window | `pkg/ui/mapeditorwizard.go` `wizardSpec` | rows from `screens/triggerwizard.json` |
| trigger window bindings | `pkg/ui/mapeditorwizardscope.go` `wizardScope` | new: the wizard's state as bindings |
| mockup page | `pkg/ui/popuppage.go` `popupPage` | new: reads `popupLayout`, inks from `editorPopupInk` |
| popup picture | `pkg/ui/editorpopup.go` `drawPopup` | unchanged; `EditorMockup.Picture` calls it |
| mockup tool | `cmd/editormockup` | new; `-out` via `game.RefuseInstallWrite`, place via `placedCorner` |
| word table use | `pkg/words` `TestEveryUsedIDIsInTheEnglishTable` | also reads the window descriptions |

## Proof

| test | rules |
|---|---|
| `TestWindowDescriptionsNameKindsAndWords` | 1, 6 |
| `TestWindowDescriptionRoundTripsThroughTheSpec` | 2 |
| `TestWindowDescriptionRoundTripsThroughThePage` | 3, 6 |
| `TestPageElementsAreTheLayoutsRectangles` | 4 |
| `TestTriggerWizardFromItsDescriptionIsTheWizardBefore` | 5 |
| `TestTriggerWizardDescriptionButtonsAreTheWindowsActions` | 5 |
| `TestMockupPictureIsThePopupPainters` | 7 |
| `TestEveryUsedIDIsInTheEnglishTable` | 1, 6 |
| `TestMockupPlacesTheTriggerWindowAsTheEditorDoes` | 8 |
| `TestToolRefusesWhereTheInstallFenceDoes` | 9 |

Release tests at the release: none.

## Open debt

- The other editor windows still build their rows in code; each moves onto a description in its own story.
- The multiplayer hero warning row has no before-and-after witness: no fixture holds a multiplayer hero map.
