# Map editor: placing objects, selecting them and groups

## Intent

A map maker places units and structures from a picture grid, selects objects
with a box on the map, moves them with a visible silhouette, and builds the
groups a trigger names without typing a group number into each unit.

## Authority

Owner direction:

1. Units and structures are chosen from an icon grid with thumbnails. The
   chosen one looks pressed, its settings open at once, and copies are placed
   by clicks or by holding the button.
2. A left click with no tool draws a selection box, for the open tab. Moved
   objects show translucent silhouettes until the release.
3. A group is the group number on one player's units. "To group..." offers an
   existing group or New (the first free number); "Remove from group";
   Ctrl+1..9 assigns the selection to that group and 1..9 selects it.
4. A Groups window lists groups per player with number, member count and
   thumbnails; a click selects the members and moves the camera to them; a
   cross removes the group.
5. Members of the chosen group show an outline and a number badge.
6. The window names the triggers that use a group and warns before a used
   group is emptied.
7. Other players' units in a selection are skipped with a note. A unit holds
   one group. Groups are numbers only.

Claims:

- `ALM-UNIT-048`: the unit record's `+0x42` word is the group id; it is not
  unique per unit.
- `ALM-TRIG-046`: a `Target_Group` parameter (type code 2) names a group by
  that word.
- `TRIG-MAPGROUP-043`: instants 32 and 33 act on every member of the group.
- `TRIG-151`: the 28 scenario maps name 169 group ids; 4 ids are authored for
  two owners on each root.

## As built

### Picture grid and placing

- `pkg/ui/editorpopup.go` gains one grid widget, `popupGrid`: the caller hands
  in cells (picture, pressed, ticked), the side of a cell, the number of rows
  shown and the first row; the popup builder lays it out, draws it, scrolls it
  and reports which cell a press hit. Rows can also carry small pictures
  (`popupRow.Icons`). Nothing in the widget knows about units.
- The Units and Structures tabs show the grid above their record list
  (`pkg/ui/mapeditorplace.go`). The pictures are the Add window's class
  previews (`NewRecordPreview`: a unit's standing sprite, a structure's
  portrait).
- A press on a cell shows it pressed and opens the Add window on that class at
  once, with the cell pick armed and the window in its repeat mode. Each click
  on the map adds one copy. A held drag adds one copy on each new cell it
  crosses. A second press on the pressed cell or Esc ends the placing.
- Undo is per placed object: each copy is one `AddRecord` and one undo step,
  including each copy of a held drag.
- The Add window's Group kind is removed; groups are made from a selection.

### Selection box and move

- A plain left press on the map with no tool, brush or record under it starts
  a selection box (`boxPointer`). On the Units, Structures and Sacks tabs the
  box takes that tab's kind; on the Groups tab it takes units; elsewhere it
  takes every kind a selection takes. A press and release without movement
  selects the record under the pointer, as before.
- A left drag on the map no longer pans in the editor. The middle button and
  the arrow keys pan.
- Dragging a record, or a selection by one of its records, draws the records
  as translucent sprites at the pointer cell until the release
  (`dragPlacement`, sharing the sprite path the move pick uses). The release
  writes one move, as before.
- A selection of more than one record opens a selection window with a count
  per kind, the group numbers in it and To group..., Remove from group and
  Delete. Delete deletes the whole selection as one edit; Ctrl+C, Ctrl+V and
  the existing drag act on the selection as before.

### Groups

- A group is the group word of each member unit (`ALM-UNIT-048`). The host
  writes it with `SetUnitsGroup` (`pkg/game/mapinspectiongroups.go`) as one
  undo step, through the same field edit the unit popup's group field uses
  with the model's history entries squashed, so one Undo restores the bytes
  from before the action and Redo the grouped bytes. It skips units that
  already carry the number. It refuses a record that is
  not a unit, units of two players and a number a unit of another player
  carries. `FreeGroup` is the lowest number no unit carries. Saving writes
  the unit records; reading takes the word back with the map.
- The UI side is `pkg/ui/mapeditorgroups.go`.

| Action | Route | Result |
| --- | --- | --- |
| To group... | selection window or Groups window button | a question lists New (group N) and the player's groups; OK writes the units to the chosen number |
| Remove from group | selection window or Groups window button | writes 0 on the selected units of the first grouped unit's player |
| Ctrl+1..9 | key | writes the selection to that number |
| 1..9 | key | selects the group's members and centres the view on them; a missing group gets a note |
| Group row click | Groups window | selects the members, opens the group popup and centres the view on the members |
| Cross | Groups window row | writes 0 on every member |

- Units of another player than the group's (or, for a new group, than the
  first unit's) and records that are not units are left out, and the bottom
  line names how many were skipped and why.
- The Groups window lists each player's groups: number, member count, the
  member pictures and a cross; the chosen group is marked. Under a row
  that a script names, a line lists the triggers and nodes that use the group
  (the group popup's Used by builder). The window scrolls with the wheel.
- Members of the chosen group draw the existing member outline and a badge
  with the group number above each member.
- An edit that would leave a group with no units while a trigger or script
  node names it opens a warning naming the users; the edit happens only on
  "Empty it", and Cancel changes nothing. Group actions and every delete
  route ask: the Delete key and the selection window's Delete button on a
  selection, and the Delete key on the one selected unit.
- Remove from group on units of several players removes the group from the
  units of the first grouped unit's player and notes the others as skipped.

### Builders reused

The popup builder for the grid, the Groups window, the selection window and
both questions; `Picker` for the To group list; the Used by rows; the area
label drawing for badges; the Add window's class previews and the record
previews for pictures; the move pick's sprite path for silhouettes; the Add
window's repeat mode for placing; the selection marquee for the box. New
popup part: the grid, used once here and meant for reuse.

## Proof

Fixture tests (`pkg/ui/mapeditorgroups_test.go`) drive every route through
the pointer or a key and read the host calls: grid choice and placing by
click and by hold, the box per tab, silhouettes until release, Delete on a
selection, the Groups window rows, a row click that chooses and frames, To
group with New and with an existing group skipping another player's units,
Ctrl+number, the warning before Remove empties a used group, the same warning
before Delete by the key, the selection window's button and the selected
unit (Cancel writes nothing, "Empty it" deletes), and Remove from group
skipping another player's units. `pkg/game/mapinspectiongroups_test.go`
checks that one Undo after assignment, removal, Ctrl+number and the Remove
button restores the exact earlier bytes and Redo the grouped bytes.

Release tests on EN and RU (`pkg/game`):

- `TestReleaseMapEditorGroupIsTheGroupATriggerActsOn`: two units of player 4
  of the guest mission go into a new group through the Groups window, and the
  start trigger's patrol order becomes instant 32 on that group. The saved
  map reopens in the editor with two members and the game's reader gives both
  units the group. In the headless run both are off the map after the start
  trigger. Loss control: the same map with only the first unit grouped leaves
  the second on the map.
- `TestReleaseMapEditorDeleteUsedGroupAsksFirst`: on `scenario/10.alm`,
  Delete of both members of group 1, which a script names, asks first by the
  key and by the selection window's button; Cancel keeps the bytes, "Empty
  it" deletes, Undo restores the bytes and Redo the deletion.
- `TestReleaseMapEditorNewGroupNumberIsMapWide`: a new group takes a number
  no unit of any player carries.
- `TestReleaseMapEditorPlacingAndGroupsWithoutAnEditSaveUnchanged`: four
  shipped maps opened, used through the grid, the box, the Groups window and
  a group key, and saved without an edit keep their bytes;
  `TestReleaseInstalledMapsSaveUnchangedWithoutAStroke` covers every shipped
  map.
- `TestReleaseMapEditorObjectsAndGroupsFrames` writes the owner's frames when
  `AGAINROM_EDITOR_FRAMES` is set.

No file outside the editor changes; game frames, World hashes and SAV bytes
are untouched.

## Open debt

- The host refuses a group number that another player's units carry, while
  four shipped scenario ids are authored for two owners (`TRIG-151`). Those
  maps open and save unchanged; the editor only refuses to add to such a mix.
- The grid is on the Units and Structures tabs only; sacks keep the Add window.
