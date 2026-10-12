# The map editor adds, deletes and changes units

## Intent

In the first game's map editor (`mapedit`) an author clones a selected unit
onto a chosen cell, deletes a selected unit, and sets a selected unit's owner
slot, class, definition id and hit points, with Undo, Redo and Save As. The
saved map plays: loaded as a mod's guest mission, its World has the clone,
lacks the deleted unit and carries the changed fields. Base: public main
`f3083ea9` (release 0.118.0), knowledge pin k267. This is item I1 of the map
editor audit at `f3083ea9`.

## Authority

Owner direction for the editor and its operations. The record and its fields:
`ALM-UNIT-018` and `ALM-UNIT-040` (the 70-byte type-6 record and its offsets),
`ALM-CLS-038` (class key +0x08, secondary key +0x0a, overriding definition id
+0x10), `ALM-OWN-039` (owner +0x14, a 1-based type-5 slot), all High;
`ALM-UNIT-048` (Medium: +0x40 is the unit id, +0x42 the group id), disclosed
under `DIV-618`. The choices the claims leave open are `DIV-3117`.

## As built

### One builder

A map edit has one builder: the byte model `pkg/mapedit.Editor`, reached only
through the window host `mapInspectionEdits.change` and the window seam
`ui.InspectionEdits`. The move, Undo, Redo and Save As paths are unchanged.

- `Editor.CloneUnit(i, x, y)` appends a copy of record `i` with new X and Y
  words and a new unit id (see Clone identity); one edit of the same three
  parts as `PlaceUnit` (record, type-6 size word, type-0 count word).
- `DeleteUnit` is the existing model operation.
- Field edits use one table, `pkg/mapedit/unitfield.go`, read by one setter
  `Editor.SetUnitField(i, key, v)`. A later field (structures, sacks, stock
  in later items) is a row, not a method.

| Key | Window label | Offset | Width | Range | Check |
|---|---|---|---|---|---|
| owner | Owner slot | +0x14 | u32 | 1..4294967295 | 1..roster size (1 when type 5 is absent, `ALM-REQ-056`) |
| class | Class | +0x08 | i16 | -32768..32767 | an installed Units row (key at or above 0x1a, with the record's secondary key) or Humans row (below) resolves it |
| definition | Definition id | +0x10 | u32 | 0..4294967295 | 0, or an installed Humans server id |
| hp | Hit points (-1 = default) | +0x20 | i16 | -32768..32767 | range only; -1 derives health from the definition (`UNIT-PLACE-034`) |

The model checks the range; the host (`checkUnitField`) runs the reference
check on the edited record against the install's tables and refuses the edit
with a message. The model and history stay unchanged on a refusal. A row a mod
adds to the tables is not consulted.

### Window

| Input | Operation |
|---|---|
| C, then a click on a map cell | clone the selected unit onto that cell; the clone becomes the selection; Esc cancels |
| Delete | delete the selected unit |
| E | open the field prompt for the selected unit, prefilled with the first field's value |
| Tab, in the prompt | next field, prefilled with its value |
| Enter, in the prompt | set the field |

The help line reads "Drag unit: move / C: clone / Del / E: fields", and
"Clone: click a cell / Esc: cancel" while a clone waits for its cell. Each
operation without a selected unit answers "Select a unit first.". A clone
keeps the source's fraction within the cell, as a move does. A clone onto a
cell outside the map is refused.

### Clone identity

A clone keeps every stored byte of its source except X, Y and the unit id:
the group id, flag word, +0x18 index, skill and resistance vectors and the
rest are copied unchanged (`DIV-3117`). The unit id is one above the highest
of every type-6 unit id and every unit id a `Target_Unit` parameter (script
parameter type 4, values below the hero band 10001) of the type-7 script
names. The script is read by `Editor.Map().Script()`, the existing alm reader.
So a script `Target_Unit` names at most one unit, and a unit deleted earlier
leaves its script id unclaimed instead of handing it to the next clone. A new
id past 65535 is refused; a script that does not decode refuses the clone.

## Proof

- `pkg/mapedit/unitauthoring_test.go`: the clone's 70 bytes against a
  hand-built expectation, the only other changes being the type-6 size word
  and the count word, Undo and Redo byte-exact; each field writes only its
  bytes and reads back through `alm.Open`; refused clones and fields write
  and record nothing; the table's rows lie inside the record.
- `pkg/game/mapinspectionunits_test.go` drives `ui.MapEditor` keys and clicks
  over the real host with a synthetic table: a click without C clones
  nothing (loss control); clone, delete and four field edits, each refused
  value leaving the bytes unchanged, then Undo through all steps and Redo back,
  byte-exact at every step.
- `TestReleaseMapEditorUnitAuthoringPlaysAsModMission` (EN and RU): opens the
  installed `scenario/10.alm` in the window's host, picks three interior
  creatures with unique unit ids, Save As writes the edited map over the
  `reniesta` mod's assembled map in a temporary mods folder, and the mod's
  guest mission 25 opens through `MissionOpenerWith`. Asserted: the clone's
  entity at its cell with the source's class, type, maximum health and group;
  no entity for the deleted unit id; the changed unit's owner, class and hit
  points in the World and its definition id in the built map (a creature's
  class key outranks its definition id, `ALM-CLS-038`, so the id is unread by
  the World); one entity more and one fewer than the loss control. Loss
  control: an unedited Save As of the same map, through the same route, plays
  all three source units and no unit with the clone's id. Both roots picked
  records 4, 5 and 6, clone cell 17,13, owner 5, class 69, definition id 1.
- `TestCloneUnitNeverTakesAnIDAScriptTargetUnitNames` (`pkg/mapedit`): a
  hand-built map with unit ids 1 to 3 whose script names 3, the hero-band value
  10001 and the integer 50; delete id 3, clone another: the clone is 4, equal
  to no type-6 id and to no scripted unit id. Loss control: with the old rule
  (one above the highest type-6 id) the clone takes 3 and the test fails.
- `TestReleaseMapEditorCloneNeverTakesADeletedScriptedUnitID` (EN and RU):
  through the window host on the installed maps, picks by rule the first map
  whose highest unit id a `Target_Unit` parameter names and whose lower ids
  are contiguous (`scenario/100.alm` on both roots: record 112, id 245), deletes
  it, clones record 0, Save As; the clone is 246 and `mapload.ScriptUnits` binds
  no script id 245. Loss control: with the old rule the clone takes 245 and the
  test fails on both roots.
- `mapedit` does not ship in the release archive (the release workflows build
  `cmd/againrom` and `cmd/starter`), so `CHANGELOG.md` takes no line.

## Open debt

- The checks use the install's tables; a class or definition a mod adds is
  refused.
- No record is added to a map without a type-6 record; clone needs a source.
- The unit id, group id and the stored fields with an unknown authoring role
  are not editable in the window.
- Structures, sacks and stock (audit item I2) and a validation report before
  save (I3) are later items.
