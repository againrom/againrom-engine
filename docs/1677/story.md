# Map editor windows built from one element tree, fitted and centred

## Result

Every map editor window is laid out, painted and hit-tested from one element tree in one layout pass. A window
taller than the editor scrolls its rows, then shortens its list, then folds its notes behind the title's question
mark, so it fits at 640x480. Question, notice and list windows first open in the middle of the editor; the Add
window keeps its place near the top. Every tip is one element: it shows after 300 ms, a question mark's at once.
A choice row's list opens inside the window, and every button is as wide as its caption, at least 96 pixels.

Every editor colour is named in one ink table that the painter and the web page mockups read. The unit window's
compact detailed view is a window description with the game's card and doll beside three field columns, with
English and Russian web page mockups; the editor does not show it yet.

## Authority

- Owner: «Надо посмотреть тщательно на редактор и собрать некий UI Framework. То есть унифицировать все экраны».
- Owner, on the first place of a window: «По дефолту да, но с возможным переопределением».
- Owner, on the tip delay: every tip after 300 ms, the (?) mark at once (answered «as recommended»).
- Owner, on the compact view mockup: «Макет выглядит хорошо, но где кукла и окно статов?»
- Owner: the editor is pinned by no table, golden or walk test.
- Owner ruling «Editor UI framework»: the map editor alone moves onto the one layout builder; the starter and the
  game screens stay out. The game's dialog keeps its own description.

## Rules

1. Every editor window laid out through the element tree has the rectangles the flat window spec had.
2. A window taller than its bounds scrolls its rows first, then shortens its list, then folds its notes.
3. At 640x480 every editor window stays inside the editor window.
4. A window first opens centred in the editor; the Add window opens at its own place; one store gives every corner.
5. The compact unit view stands its fields in three columns at 1920 wide, skills and gear folded, closed at first.
6. The compact unit view stands the game's card and the doll at the game's size beside its field columns.
7. A row tip shows after 300 ms, a (?) text at once and, if taller than the editor, scrolls; one function places tips.
8. A choice row's list opens inside the window, and a press on a row drawn over its box chooses the row.
9. Every editor button takes its width from one measure: its caption plus pad, at least 96.
10. A typed row draws through one edit field; every editor picture stands through one picture element.
11. Every editor colour is named in one ink table, which the painter and the page writer read.
12. A window description reads back the same from its window tree and from its page.
## Code

| kind | its one builder | change |
|---|---|---|
| editor window layout | `pkg/ui/editorpopup.go` `layoutWindow` | over the tree of `editortree.go`; height fit in `giveWay` |
| window placement | `pkg/ui/mapeditorpopupstore.go` `centreCorner`, `keptCorner` | homes and prompt corner removed |
| tip | `pkg/ui/editortip.go` `layoutTip`, `drawPopupTip` | one placer and painter, the delay in the editor |
| drop-down | `pkg/ui/mapeditorobject.go` `rowDropdown`, `drawEditorDropdown` | area list and menu through it |
| game dialog | `pkg/ui/dialog.go` `Dialog`, `layoutDialog`, `paintDialog` | outside the editor; `drawPopup` is the editor window painter; `internal/archtest/dialogdescription.go` guards both |
| button, edit field, picture | `popupButtonWidths` via `fitButtons`; `editorinputview.go` `typedRows`; `editorpicture.go` `drawEditorPicture` | hand widths removed; icons, portraits, previews, the doll |
| ink table | `pkg/ui/editorink.go` `editorPopupInk` | page reads it; `internal/archtest/editorink.go` guards |
| compact unit view | `pkg/ui/screens/unitcompact.json`, `popuppage.go` | card and doll column; page reads the tree |

## Proof

| test | rules |
|---|---|
| `TestEditorTreeLaysOutAsTheFlatSpec`, `TestEditorTreeLaysOutModWindowsAsTheFlatSpec` | 1 |
| `TestEditorWindowGivesWayInsideItsBounds` | 2 |
| `TestEditorPagedWindowsKeepInsideA640x480Editor` | 3 |
| `TestPromptsFirstOpenInTheMiddleAndTheAddWindowAtItsOwnPlace`, `TestEditorPopupPlacementIsOneStore` | 4 |
| `TestUnitCompactViewGroupsFieldsInColumns` | 5 |
| `TestUnitCompactViewStandsCardAndFigureBesideColumns` | 6 |
| `TestEditorRowTipWaitsTheDelay`, `TestEditorHelpMarkShowsAtOnce`, `TestControlLayoutHasOneDescriptor`, `TestFoldedNotesTipReadsInsideTheEditor` | 7 |
| `TestChoiceRowDropdownOpensInsideTheWindow`, `TestOpenDropdownListIsHitBeforeItsBox` | 8 |
| `TestQuestionButtonsTakeTheOneMeasure` | 9 |
| `TestTypedRowIsOneEditField`, `TestEditorPictureFitsItsBox` | 10 |
| `TestPageInksAreThePaintersTable`, `TestEditorInkIsOneTable` | 11 |
| `TestWindowDescriptionRoundTripsThroughTheSpec`, `TestWindowDescriptionRoundTripsThroughThePage` | 12 |
Release tests at the release: none.

## Open debt

- The map search list and its typed field, and the brush preview, keep their own drop-down, field and picture.
- The graph, passability and capture files keep 12 colours of their own, listed in the ink scan.
- The editor shows the compact unit view only after the unit window's screen step.
