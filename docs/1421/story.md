# The map editor places, moves and changes structures and sacks

## Intent

In the first game's map editor (`mapedit`) an author moves, clones, deletes
and changes structures, adds, moves, clones and deletes ground sacks, and sets
the items and gold of a sack and the items of a unit's stock, with Undo, Redo
and Save As. The saved map plays: loaded as a mod's guest mission, its World
has the moved structure at its new cells, blocking them, with its old cells
free, lacks the deleted structure, and has the added sack, whose item the hero
picks up. Base: public main `0129b0a5`, merged with `165270ed`; knowledge pin
k275, main's pin after the merge. This is item I2 of the map editor audit at
`f3083ea9`.

## Authority

Owner direction for the editor and its operations. Structures (type 4):
`ALM-OBJ-019`, `ALM-OBJ-034`, `ALM-CLS-036`, `ALM-CLS-037`, `ALM-OBJ-062`,
`TERR-STRUCT-090`, all High for the fields used. Sacks and stock (type 8):
`ALM-SACK-065` High, `ITEM-OWNED-028`. The structure fields +0x0c, +0x0e and
+0x12 have no named reader in the engine; they are copied and not offered.
The choices the claims leave open are `DIV-3133` (structures) and `DIV-3134`
(sacks and stock); `DIV-618` and `DIV-3117` are updated.

## As built

### One section table

A map edit has one builder: `pkg/mapedit.Editor`, reached only through the
window host `mapInspectionEdits.change` and the window seam
`ui.InspectionEdits`. `pkg/mapedit/section.go` holds one table with a row per
record kind. A row names the record type, the type-0 count word, the X and Y
offsets, the grammar that measures one record, and the hooks a grammar needs.

| Kind | Type | Count word | X, Y | Grammar | Hooks |
|---|---|---|---|---|---|
| unit | 6 | +0x24 | +0x00, +0x04 | 70 bytes | clone renumbers the unit id (the 1419 rule) |
| structure | 4 | +0x20 | +0x00, +0x04 | 20 bytes, 28 when kind is 0x21 | a field write re-shapes the 0x21 extension |
| sack | 8 | +0x2c | +0x08, +0x0c | 16- or 20-byte head (format version 989) plus 10 bytes per element | a record with a non-zero owner (a stock) has no cell |

The operations `Count`, `Record`, `Position`, `Place`, `CloneRecord`, `Move`,
`Delete`, `Fields`, `FieldValue` and `SetField` take a kind and an index and
read only the row. Each walks the section by its count word and refuses a
section that does not consume its payload exactly. Every operation is one
edit, so one Undo step. The unit methods (`PlaceUnit`, `CloneUnit`,
`MoveUnit`, `DeleteUnit`, `SetUnitField` and the rest) are one-line wrappers
over the unit row and write the same bytes as before: story 1419's model,
host and release tests pass with their assertions unchanged. Their calls
through the window seam name the generic methods (`Clone("Unit", …)`); that
rename is the only change to those tests.

### Field rows

| Kind | Key | Bytes | Range | Check at the operation |
|---|---|---|---|---|
| unit | owner, class, definition, hp | as `DIV-3117` | as `DIV-3117` | as `DIV-3117` |
| structure | kind | u32 +0x08 | 1..255 | the installed `structures.reg` holds the ID |
| structure, kind 0x21 only | width, height | byte +0x14, byte +0x18 | 0..255 | range only; 0 and 0 select the table footprint |
| ground sack, 20-byte head | gold | u32 +0x10 | 0..2147483647 | range only |
| sack or stock | itemN | low u16 of element N's code word | 0..65535 | 0 removes the element; else the installed item-name table names it |
| sack or stock | additem | a new 10-byte element | 0..65535 | the installed item-name table names it |

A structure kind set to 0x21 adds an 8-byte zero extension; a kind leaving
0x21 drops it. A move, clone or field edit of a structure whose footprint, as
`mapload.Structures` builds it from the installed tables, leaves the map is
refused, as is a sack or clone cell outside the map. A refused edit writes and
records nothing.

### New records

A clone copies every byte of its source at the new cell and keeps the cell
fraction; only a unit clone takes a new id. A new sack is a ground record:
owner 0, words at the cell centre (fraction 128,128, as all 133 ground sacks
on the 28 `scenario` maps of each root), no gold, one element with a zero
second word and no type-9 link. It is appended after the last type-8 record.

### Window

| Input | Operation |
|---|---|
| drag a unit, structure or sack | move it; a press on several records moves the selected one, else a unit before a sack before a structure |
| C, then a click on a map cell | clone the selected unit, structure or sack onto that cell; the clone becomes the selection |
| S, then a click on a map cell | prompt "Item code for the new sack"; Enter adds the sack and selects it |
| Delete | delete the selected unit, structure, sack or stock |
| E, Tab, Enter | field prompt over the selected record's rows |

The help line reads "Drag: move / C: clone / Del / E: fields / S: sack". A
stock is selected from the list; a move or clone of it is refused. After an
edit the window keeps its viewer and takes the new structure and sack layers.

## Proof

- `pkg/mapedit/records_test.go`: each structure operation against an
  expected buffer built by splicing the fixture independently (move: the two
  coordinate words; clone: the 28 source bytes at the new cell plus the size
  and count words; delete: the 20 bytes plus the two words); the 0x21
  re-shape both ways; a sack placed, its gold set, an element appended,
  removed and changed, every step read back through `alm.Open` and
  `Map.Loot`; refused values, a stock move or clone and a section that does
  not walk write and record nothing; the short head offers no gold; Undo and
  Redo byte-exact through every sequence.
- `pkg/game/mapinspectionrecords_test.go` drives `ui.MapEditor` keys, clicks
  and drags over the real host with a synthetic table: drag, refused
  off-map move, C clone (a click without C clones nothing), refused and
  accepted kind, Delete, S sack (a click without S adds nothing, an unnamed
  item refused), gold and an added item, a stock item, a stock clone
  refused, sack delete; then Undo through every step and Redo back,
  byte-exact. A drag on a unit standing on a structure's anchor moves the
  unit.
- `TestReleaseMapEditorStructuresAndSacksPlayAsModMission` (EN and RU): opens
  the installed `scenario/10.alm` in the window's host, picks by rule the
  smallest blocking structure on open ground with no unit or sack within four
  cells and the nearest shift along X onto open ground (both roots: record 2,
  kind 13, from 11,61 to 25,61), deletes record 0 (anchor 23,65), adds a sack
  of item 0x1106 at the open cell nearest the hero (15,64), Save As over the
  `reniesta` mod's assembled map, and opens the guest mission. Asserted on the
  World: the moved structure at its new anchor with its 2x2 footprint, none at
  the old anchor or the deleted anchor, one structure fewer; the sack with its
  item; the hero ordered to it carries one more of the item; the hero ordered
  onto each blocking cell of the new footprint never stands there and reaches
  every old cell. Loss control: an unedited Save As through the same route
  has the structure at its old anchor blocking its blocking cells, the
  destination cells reachable, the deleted structure present and no sack on
  the cell.
- `mapedit` does not ship in the release archive at the final commit (the
  release workflows build `cmd/againrom` and `cmd/starter`), so
  `CHANGELOG.md` takes no line.

## Open debt

- The checks use the install's tables; a structure class or item a mod adds
  is refused.
- The structure fields +0x0c, +0x0e and +0x12 are not editable; a clone shares
  its source's +0x12 script id, and a deleted structure's id may stay named by
  a script. A validation report before save is item I3.
- A sack's owner word is not editable: a ground sack cannot become a stock,
  and no stock is added to a unit that has none.
- A record cannot be added to a map that has no record of its type.
