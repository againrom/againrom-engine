# Map editor group selection, copy and paste, Russian brush words

## Intent and authority

Owner direction: the editor's brush words and key help read in the install's
language; the passability fills either stay readable at the smallest zoom or
stop there, by a stated rule; several records can be selected on the map,
moved, deleted, copied and pasted, also into another map of the same session,
each as one undo step. The editor's layout and keys are the owner's; the
original editor is evidence of what map bytes mean, not of its keys. No DIV
row: no game-visible behaviour changes.

## As built

### Words

- `pkg/words/text/{en,ru}/strings.toml` hold 71 new ids, each in both tables:
  `editor.terrain.*` (the ten `mapedit.Palette` rows; RU Трава, Земля, Песок,
  Трещины, Цветы, Камни, Саванна, Горы, Вода, Дорога),
  `editor.brush.raise`/`lower`/`status`/`noobject`/`object`, `editor.keys.*`
  (the title and every line of the status help and the F1 key list) and
  `editor.selection.*` (the group's status notes).
- `mapedit.Palette` keeps its English names; the window words a palette row
  through `terrainWord`. The objects list's clear row and a class without a
  DescText are worded by the window (`objectWord`); the host sends the clear
  row as code 0 with no name.
- The key table names each help line by its word id (`helpLine`).
  `ui.EditorHelpLines` is the English set; `ui.EditorHelpLinesIn(lang)` the
  install's. `TestReleaseMapEditorHelpLinesFitTheRail` measures the install's
  lines in the install's font.
- Russian lines are shortened to fit: every F1 line within the 300 px rail and
  the keys help within the 1264 px status strip at 1280x800, measured in the
  RU install's font (the first RU drafts measured up to 622 px a line and
  1487 px for the keys help). Where the terse form drops a word, the drop is
  in RU only: the brush lines "Тащить: рисовать / [ ]: размер" leave out
  "T: выкл." (the F1 line "T,H,B: кисть" names the keys).
- The F1 key list showed each row as "W keys map" (the report window's label
  of a map-wide finding), so no help line was readable in it on either
  install. A row whose rule is the key list's now shows its line.
- Stays English on both installs: object class names (the registry's
  DescText, which the RU install also ships in English; `classdump` on
  `gameversions/ru`), unit class names on the status line, the host's error
  texts inside a refusal note (for example "structure footprint 38,38 3x2
  leaves the map"), and every editor word outside the brushes, the key help
  and the group notes that was English before.

### Passability at the smallest zoom

Rule: the fills stay readable. They are drawn at every zoom; the grid still
stops below `GridMinCellPixels`. The GPU path draws the fills in batches of at
most 16384 cells (one `DrawTriangles` call each, 16-bit indices) instead of
one call per cell. The headless CPU path (`paintPassability`) is unchanged.

### Group selection

- Shift + drag on the map draws a rectangle, outlined cell edge by cell edge
  over the ground's heights; on release every unit, structure and sack whose
  cell lies inside is selected. A drag without Shift pans as before.
- Shift + click adds the unit, structure or sack under the pointer to the
  group, or takes it out; a group starts from the record already selected.
- Every member is drawn gold with its footprint. While several are selected
  the record panel stays closed, the status line shows "N selected" and the
  help switches to "Drag: move all / Del: delete all", "Ctrl+C: copy /
  Ctrl+V: paste", "Esc: clear the selection".
- Dragging any member moves the group by that member's offset; the preview
  moves them all. Del deletes the group. Esc clears it. A click on empty map
  clears it; so do opening a map and any other edit.
- Ctrl+C copies the group, or the one selected unit, structure or sack.
  Ctrl+V pastes at the cell under the pointer: the top-left of the copied
  cells lands there and each record keeps its offset and its fraction within
  the cell. The copy is a snapshot of the source map's model, kept for the
  session, so a paste works after another map is opened.
- Each move, delete and paste is one undo step (`mapedit.Editor.Mark` and
  `Squash` around the whole group).
- Every existing key keeps its action. Ctrl+C and Ctrl+V were unbound;
  Shift had no editor binding.

### What a paste copies, and the rule for references

- Each record is pasted through the clone of its kind:
  `mapedit.Editor.CloneFrom(src, kind, index, x, y)`. `CloneRecord` is
  `CloneFrom` with the editor as its own source, so a single Clone writes the
  bytes it wrote before.
- "The same map" is decided by the source name the copy was made under (the
  map the window opened), not by the editor model: a paste after another map
  was opened and the first one opened again is a paste into the same map.
- Into the same map a paste equals one Clone per record, byte for byte:
  identity ids are renumbered, a unit's stock and its recipes are copied, a
  ground sack shares its item recipes. The paste reads the records from the
  copy's snapshot through `mapedit.Editor.CloneFromCopy`, which keeps a sack
  item's recipe link wherever the map still holds that recipe's bytes at that
  index, and copies the recipe otherwise.
- Into another map the same, and a ground sack's item recipes are copied into
  the target, since the target's recipe numbers name other records. A record
  whose layout differs under the target map's grammar is refused.
- Scripts and groups. Identity ids are renumbered by the clone. Owner
  direction: no script of the target map may address a pasted record. So in
  a paste into another map each non-zero group id of the pasted units takes a
  fresh group id, one above the highest group id any unit of the target holds
  or any Target_Group parameter of its script names
  (`mapedit.Editor.HighestGroup`), in the clip's order; units of one source
  group share their new id. Group 0 stays 0. Into the same map the group id
  is kept, as Clone keeps it.
- Owner: a unit or structure whose owner slot the source roster holds and the
  target roster lacks is refused ("unit 1: this map has no player 2"). A slot
  neither roster holds is copied as Clone copies it.
- The status note says "Pasted N of M", when recipes were copied "recipes
  copied: K", and when group ids were renumbered "groups renumbered: K" (RU
  "перенумеровано групп: K").

### Refusals

Each record of a move or paste is placed or refused on its own, and the note
names the first two refusals and counts the rest:

- a cell outside the map ("unit 0: cell X,Y is outside the map");
- a structure footprint that leaves the map, or a clone or move the host
  refuses (the host's text after the record's name);
- a unit on a cell the edited map's passability plane
  (`mapload.PassabilityWith`) closes to the unit's movement domain, the
  domain the map load gives the placement (`mapload.UnitDomain`, the block
  `FromALMWith` builds) through `sim.Domain.BlockedBy`; a sack on a cell the
  ground plane closes. The note names the arms (`mapload.GroundArms`): "unit
  1: cell 32,30 is closed (border)". A ground unit is refused on every closed
  cell; a ghost or air unit (the shipped Bees, Ghosts, Sonic Bats, Dragons)
  only on a border cell, where the note names the border. Units and sacks do
  not shape that plane, so the placement runs once more without them. On
  `scenario/10.alm` a drag of a town group two cells right moved 1 of 3:
  "unit 23: cell 57,15 is closed (object); unit 27: cell 58,12 is closed
  (structure)"; a copy of Bee unit 11 pasted on its own cell 13,49 (Water) is
  placed, and ground unit 0 pasted there is refused "(Water)".

A delete refuses a record only when the host refuses it.

The host returns the note's lead ("Pasted N of M" and its extra parts) and
one note per refusal (`ui.SelectionEdit.Refusals`); the window composes the
status part. The whole line (`StatusLine`) names the first two refusals and
counts the rest. The strip (`statusFitted`) first drops leading help lines;
when the line is still wider than the strip it names the first refusal alone
and counts the rest ("N more", RU "еще N"), then cuts that refusal with "..."
to the room left beside the last help line ("Esc: clear the selection" while
a group is shown); only when no part of the refusal fits beside the help
does the help go.

Unchanged: game frames, World hashes and SAV digests (no `pkg/sim`, save or
game-screen code changes); the bytes of an unedited map; the bytes a single
Clone writes.

## Proof

- `pkg/game/mapinspectionselection_test.go`, a 40x40 map with two players,
  two units, a 3x2 structure and a ground sack, driven through the window:
  - `TestMapEditorRectangleSelectsUnitsStructuresAndSacks`: 4 selected, then
    3 when the rectangle leaves the structure out.
  - `TestMapEditorGroupMoveKeepsTheOffsetsAndUndoesInOneStep`: every record
    at +3,+2, fractions kept, one undo restores the bytes. Control: the same
    drag without a group moves one unit.
  - `TestMapEditorGroupDeleteUndoesToTheSameBytes`: "Deleted 4 of 4", one
    undo gives the map's bytes back.
  - `TestMapEditorPasteEqualsOneClonePerRecord`: the pasted map equals
    `CloneRecord` per record at the same cells; into a second map opened in
    the window it equals `CloneFrom` per record; one undo reverts each paste.
    Control: the clones change the map.
  - `TestMapEditorPasteRefusesARecordByName`: the structure footprint and the
    border, scenery on the target cell ("unit 1: cell 16,10 is closed"), and
    a missing player 2. Control: a paste with room refuses none.
  - Mutation check: with the group's `Squash` removed, the move, paste and
    refusal tests fail.
  - `TestMapEditorPasteTakesAFlierOnACellClosedToGround`: under a table whose
    class 41 moves by air, unit 1 pasted on a scenery cell is placed ("Pasted
    4 of 4") and on a border cell refused. Control: the same paste under the
    table where class 41 is ground refuses it "(object)".
  - `TestMapEditorPasteInTheSameMapSharesASackRecipe`: the sack carries a
    recipe; copy, open another map, open the first again, paste: the bytes
    equal `CloneRecord` per record and no recipe is copied. Control: the same
    clip pasted into another map reports "recipes copied: 1".
  - `TestMapEditorPasteIntoAnotherMapTakesFreshGroups`: source units in group
    5, 0 and 5; the target's script names group 5 (a Win group arm). The
    pasted units hold 6, 0 and 6, none named by the script, and the note says
    "groups renumbered: 1"; one undo reverts. Control: pasted into its own
    map the groups stay 5, 0, 5.
- `pkg/mapedit/clonefrom_test.go`: a unit with stock and a sack with a recipe
  cloned into another map bring their recipes, the source stays byte-equal,
  each clone is one undo step. Control: within one map the sack's clone
  shares its recipe. `TestCloneFromCopySharesTheRecipeTheMapStillHolds`: a
  clone from a copy of the map equals `CloneRecord` byte for byte. Controls:
  `CloneFrom` the same copy copies the recipe; after the map's recipe changed,
  `CloneFromCopy` copies it.
- `pkg/ui/mapeditorselection_test.go`: rectangle selects exactly the records
  inside, in both drag directions (control: no Shift selects nothing); Shift
  click adds and removes; a group drag calls `MoveRecords` with the offset
  (control: one record moves alone); Ctrl+C, Ctrl+V in this and a second map,
  Del and Esc reach the host (control: single delete, paste with nothing
  copied); brush words RU and EN with a checker whose control rejects
  English; F1 rows Russian and shown as their lines; passability fills read
  at the smallest zoom; the batches hold every fill.
- `pkg/ui/mapeditorkeys_test.go`: every key the help names acts, including
  the selection screen.
- `pkg/ui/testdata/mapeditor_modes.golden` is regenerated: against the base
  golden only the help column changed, in 72 of 103 rows (the F1 list gained
  lines and their words moved to the table).
- Release, EN and RU: `TestReleaseMapEditorSelectionFrames` (new) drives the
  brush lists, the F1 list and a group on `scenario/10.alm` and checks each
  status in the install's words; `TestReleaseMapEditorHelpLinesFitTheRail`
  and `TestReleaseMapEditorStatusFitsWithPassability` now measure the
  install's words; `TestReleaseMapEditorObjectListOffersOnlyDrawableClasses`
  expects the unnamed clear row.
- Release, EN and RU, correction: `TestReleaseMapEditorPasteTakesAShippedFlier`
  (Bee unit 11 of `scenario/10.alm` on Water 13,49 pasted on its own cell is
  placed; ground unit 0 pasted there is refused with "(Water)", RU
  "(Вода)"); `TestReleaseMapEditorGroupNoteFitsTheStatusStrip` (1280x800, a
  two-unit group pasted on its own cells with three refusals of the widest
  wording the paste produces in the install's body font, "structure 99999:
  cell 9999,9999 is closed (border, Mountain, Water, flag, object)": the
  fitted line is 1261 px on EN and 1260 px on RU within 1264, names the cut
  first refusal, "2 more" and keeps the Esc line; control: the whole line is
  2756 px on EN and 3052 px on RU).
- Loss controls of the correction, one mutant each through `go test
  -overlay`: the domain check off fails the fixture flier test and the EN and
  RU flier release tests; the same-map decision off fails the same-map recipe
  test (and the group test's same-map control); `CloneFromCopy` made
  `CloneFrom` fails `TestCloneFromCopySharesTheRecipeTheMapStillHolds`; fresh
  groups off fails the group test; the note fit off fails the EN and RU fit
  release tests.
- Gates on `70c23410f`: `go test -trimpath -count=1 ./...` 82 packages ok;
  `gofmt -l` clean; `go vet` of the touched packages clean; `git diff
  --check` clean; `check-no-game-assets.sh` clean; storyguard and archtest
  pass (CommentBytes 9253065 to 9264855 for the new code). Release set with
  `AGAINROM_HEADLESS_ONLY=1`: EN and RU 1317 gated tests each, 0 failures,
  one skip, `TestReleaseActivePauseRenderedWorld`, by that variable. Mission
  scenarios 76 of 76. The second game's SAVE selection run by hand: EN 110,
  RU 110 declared subjects pass. `check-div-claims.sh` exits 0; no DIV row
  changed.

- Gates of the correction: `go test -trimpath -count=1 ./...` 82 packages
  ok; `gofmt -l` clean; `go vet` of `pkg/mapedit`, `pkg/mapload`, `pkg/game`
  and `pkg/ui` clean; `git diff --check` clean; `check-no-game-assets.sh`
  clean; storyguard and archtest pass (CommentBytes 9264855 to 9269247 for
  the new code). `TestReleaseMapEditor*` with `AGAINROM_HEADLESS_ONLY=1`: EN
  69 and RU 69 pass, none fails.

## Open debt

- Object and unit class names stay English on the RU install: the registry
  ships them so.
- A refusal note carries the host's error text, which is English on both
  installs.
- The group selection takes units, structures and sacks; trigger areas, the
  drop cell and script records are not selectable.
- Group 0 is kept in a paste into another map, by the owner's rule for this
  story. The script loader treats group 0 as a real group, so a script of the
  target that names group 0 also names a pasted unit of group 0.
- A structure's word at +0x42 (the structure field a script checks) is not
  renumbered in a paste.
- "The same map" is the source name: after Save As under another name the
  map counts as another map, so its groups are renumbered and a sack's recipe
  is copied. Two unsaved new maps of one size share one source name (`new/WxH`).
- The refusal note of a ghost or air unit names the border only; a cell that
  a structure covers is not refused to such a unit, as the plane does not
  close it.
