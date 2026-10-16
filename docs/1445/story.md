# Map editor top bar and popups

## Intent and authority

Owner direction: the map editor needs a clear flow. The map picker opens as
its own scrollable popup; every selected object opens a popup with its data,
easy to edit; a bar along the top opens popups for viewing and editing what is
selected (units, structures, groups and the rest). The trigger editor and
trigger areas on the map are a later story. Standing owner rulings: the editor
is Againrom's own, the original editor is evidence only of what it writes, and
one builder serves each kind. No ROM1 claim covers the editor's layout or
words; no divergence row is needed. Base engine main `cb15280e`, pin unchanged.

## As built

The right rail is gone. A top bar holds Map (drop-down: New map, Open...,
Open file..., Save, Save as..., Test play, Exit), Units, Structures, Groups,
Sacks, Players, Triggers, View (drop-down: drop cells, footprints, report
issues, grid, report, fit), Undo, Redo and Save. The map fills the window
between the bar and a one-line status strip of the same height, at scale 1.
The status strip shows the cell under the pointer, the map's saved state and
the mode's keys. Every keyboard accelerator still works.

Popups float over the map, move by their title bar and close by their close
button or Esc. At most one list popup and one object popup are open; the map
stays clickable. Choosing another object replaces the object popup's content.

- Map picker: open at start without `-map` and from Map > Open.... The
  new-map sizes stand under their own heading, then the installed maps. Wheel,
  Up/Down, PgUp/PgDn, Home/End, the scroll bar and Enter work; a double click
  or Open opens the map; File... opens the native path dialog.
- List popups: Units, Structures, Sacks, Players and Triggers list one row per
  record. A row selects the record, centres the map on it and opens its object
  popup. Groups lists the units' group field per player (group number, player,
  member count); a group row boxes its members on the map and opens a group
  popup whose member rows open each unit.
- Object popup: every field of the record's kind at once. Numbers and text are
  edit fields; a field with choices is a drop-down; Unit, Sack and Structure
  have a Cell row whose "On map" button arms a move by a click on the map; a
  data line naming a map reference is a link that selects the referenced
  record. Apply writes all changed fields through `InspectionFieldSet.SetFields`
  as one Undo step (`mapedit.Editor` Mark and Squash); Cancel reloads the
  record; a refused value shows its reason in the popup and changes nothing.
  Scenery and Section show their data read-only. Tab moves between fields,
  Up/Down steps a choice, Enter applies.
- While a field is focused, keyboard input goes to the field. A click on a bar
  button (Undo, Redo, Save) or a Map menu row first ends field typing as Esc
  does, then acts; the typed value is not applied, because Apply stays the
  only commit. The Exit row asks the window to close through
  `MapEditor.ExitAsked`, which `Update` honours.
- After every edit, Undo and Redo, the open group popup is rebuilt from the
  current document, keyed by the group's player and group number: members, map
  boxes and member links. A group left with no member closes its popup.
- The record's picture stands right of its data lines; data lines wrap at word
  boundaries.
- While a cell pick is armed (place, clone, move, drop, goal target), the
  object popup steps aside so the click reaches the map, and returns after
  the pick. The list popup stays.

Builders. `pkg/ui/editorpopup.go` is the one editor popup builder:
`layoutPopup` turns a popup description (title, typed rows, list, notes,
buttons) into rectangles that both `drawPopup` and the hit test read. It builds
the map picker, the list popups, the object and group popups, and the prompt
and message popup that replaced `modalLayout`. Frames, push buttons, edit
fields, list boxes and scroll bars come from the widget kit. The drop-down is
a new widget kind with one builder, `pkg/ui/widgetdropdown.go`, opening its
list on the kit's `Picker` model; the bar menus and every choice field use it,
and `dropdownStep` is the old Up/Down choice stepping. The editor's rail,
its row offset, wheel handling and `mapeditorlayout.go` are removed; lists use
one `Picker` model per content key.

Words. The editor's 52 `editor.*` words come from `pkg/words` in English and
Russian and follow the install's language.

Fonts. Titles, top-bar entries, menu rows and section headings use `font1`
(16x15), the engine default and the install's decorative face. List rows,
field labels and values, data lines, notes and the status strip use `font4`,
the documents font, loaded through `LoadFont(DocumentFont, FontCoverage)` with
`font1`'s byte selector. An install without `font4` keeps `font1` throughout.

## Proof

- `pkg/ui`: `TestDropdownOpensScrollsAndChooses`,
  `TestDropdownStepWrapsAsTheChoiceKeysDid`,
  `TestPickerSetRowsKeepsSelectionInside`,
  `TestPopupLayoutIsThePaintAndTheHitTest`, `TestPopupRowsScrollAndTheListSits`,
  `TestEditorMapPickerOpensByListAndMenu`,
  `TestEditorObjectPopupApplyIsOneUndoStep` (Apply, one Undo byte for byte,
  Cancel, refusal with its reason, Esc and close button),
  `TestEditorObjectPopupWithoutFieldsIsReadOnly`,
  `TestEditorGroupRowHighlightsExactlyItsMembers`,
  `TestEditorViewMenuSwitchesTheAnnotations`,
  `TestEditorGroupPopupFollowsUndoRedoAndEdits`,
  `TestEditorBarButtonsActWhileAFieldIsFocused` (Undo, Redo, Save),
  `TestEditorMapMenuRowsActWhileAFieldIsFocused` (each Map menu row), and the editor tests of stories
  1092, 1093 and 1118 rewritten for window coordinates.
- `pkg/game`: `TestSetFieldsIsOneEditWithTheFieldByFieldBytes` writes the same
  ALM bytes through `SetFields` and through the field-by-field setters.
  `pkg/mapedit`: `TestSquashFoldsTheEditsSinceAMarkIntoOneUndo`.
- Release, EN and RU: `TestReleaseMapEditorOwnerFrames` drives the installed
  editor through the six owner states and requires every glyph of the
  installed fonts to stay kept or blank, except text under an open drop-down's
  list. With `AGAINROM_EDITOR_FRAMES` it writes the owner frames.
- Release, EN and RU: `TestReleaseMapEditorGroupPopupFollowsUndoAndRedo`
  deletes unit record 0 of `scenario/10.alm`, opens player 2 group 2, then
  Undo, Redo and Undo; the popup lists, boxes and opens the group's true
  members each time.
- The editor flows of stories 1439 to 1441 (record and script authoring,
  Save As, packages, F5 test play and return) pass in the release set.

## Open debt

- Save and Save as open the same path dialog when no mod target is set.
- Host field labels, help lines and status-strip key hints, record titles,
  choice names, record details and host messages stay English in the Russian
  editor.
- Script argument coordinates are numeric fields, without a pick on the map.
- Popups open at fixed positions (list at the top left, object at the top
  right) and can cover map corners until moved.
- The E key still edits one field in the prompt popup, without the
  drop-down.
- The object popup scrolls by the wheel and has no scroll bar.
- `TestMapEditorCapturesItsInstalledFontAtScaleOne` admits the Hidden fate:
  with its synthetic test font 17 top-bar label glyphs show no cell. The
  installed fonts hide none outside an open list
  (`TestReleaseMapEditorOwnerFrames`).
