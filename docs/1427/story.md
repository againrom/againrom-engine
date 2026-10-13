# The map editor creates a new map from scratch

## Intent

In the first game's map editor the author creates a new map of a chosen size,
puts a unit on it, and saves it into a mod; the saved map opens in the editor
as any installed map does and plays as a mod mission. The story started at
public main `49d36217` (release 0.120.0); the pin moved from k275 to k287.

## Authority

Owner direction (map editor, priority; audit item I8). Claims read from the
pin: `ALM-REQ-055`, `ALM-REQ-056` (the record set the game accepts and its
defaults), `ALM-NEWMAP-236` and `ALM-NEWMAP-237` (what the original editor
writes into a new map), `ALM-SAVE-238` (the group mask and tile normalisation),
`ALM-GRID-012` and `ALM-GRID-013` (the tile and height planes), `TERR-246` (the
fill and the height), `ALM-GRP-041` (the roster record), `ALM-UNIT-040`,
`ALM-TAILU16-082` and `UNIT-PLACE-034` (the unit record), `MISSION-DROP-002`
(the drop cell and its fallback), `TERR-PLACE-208` (a unit in the border ring
is deleted). Rows: `DIV-3207` to `DIV-3212`; `DIV-3213` and `DIV-3214` are
returned unused.

## As built

### One constructor

`alm.NewMap(spec)` (`pkg/formats/alm/newmap.go`) builds the bytes from data
tables and returns `OpenDocument` of them, so the model opens the result by
the one route and no second map model exists. `mapedit.NewMap` wraps it as an
edit session. Each row of the defaults names its claim or its divergence row.

| field | bytes written | source |
|---|---|---|
| sides | W, H at type-0 `+0x00`, `+0x04`: 32 to 256 in steps of 8 | `ALM-NEWMAP-237` |
| type-0 scalars | `+0x08` `0x3f490fda`, `+0x0c` 360, `+0x10` 21, `+0x14` 63, `+0x1c` 2, `+0x20` to `+0x2c` 0, `+0x70` 4, `+0x74` 1 | `ALM-NEWMAP-236` |
| group mask | `+0x18`: `1 << ((tile & 0x3ff) >> 6)`, plus `0xf00` above `0xff` | `ALM-SAVE-238` |
| name | `Multiplayer Map`; the rest of the 512-byte text region is 0 | `ALM-NEWMAP-236`, `DIV-3208` |
| tile plane | one word in every cell, Grass `0x11` by default; nine fills listed | `TERR-246` |
| height plane | one byte in every cell, 64 by default, at most 126 | `TERR-246` |
| scenery plane | 0 | `TERR-246` |
| roster (type 5, 152 bytes) | two records: colour 1 / 2, `+0x04` word 0 / 1, scalar 5000, name empty / `Enemy`, relation rows 2 on the diagonal, 1 between the two, 0 elsewhere | `ALM-NEWMAP-236`, `DIV-3208` |
| types 9, 8, 7 | 4 zero bytes, empty, 12 zero bytes (three zero counts) | `ALM-NEWMAP-237` |
| types 4, 6 | empty records | `DIV-3207` |
| file header | magic, 20, dataSize `4*W*H+0x48`, 10 records, version 990 | `ALM-NEWMAP-237`, `ALM-WRITER-099` |
| record headers | tag 7, hdrLen 20, `+0x10` word 0, order 0,1,2,3,5,4,9,8,6,7 | `DIV-3207` |

The two empty records make the unit, structure and sack operations of the one
section table work on a new map unchanged. A side of 16 or less is border ring
throughout (`TERR-PLACE-208`: a unit placed there is deleted at load), so the
lower bound is the claim's 32 and a 16 by 16 map is refused. The brief's
16 by 16 release test runs on 32 by 32 for this reason.

### A unit from nothing

A new map holds no unit to clone, so `Editor.NewUnitRecord` builds one type-6
record (`DIV-3209`): the cell centre, the class key, the secondary key of the
first installed definition row with that class key, owner slot 2 when the
roster has two entries, the four u16 words at `+0x20` to `+0x26` as 0xFFFF,
every other byte 0, unit id one above the highest in use. `MapInspector.AddUnit`
checks the class against the install and places the record with `PlaceUnit`.

### The window and the command line

- The catalogue lists one row per size above the installed maps (`new/32x32`,
  `48x48`, `64x64`, `80x80`, `128x128`, `256x256`). N opens the catalogue at its
  first row. A click opens the map through `MapInspector.Open`, the one door
  of an installed map.
- U arms a unit on the next clicked cell; the class key is asked in the field
  prompt of the sack, and Enter adds the unit as one edit. The host must
  implement `ui.InspectionUnits`.
- The help is the editor's one list (`ui.EditorHelpLines`); the keys list gains
  a fourth row `N: new map / U: unit`, within the four-line reserve, and the
  armed unit pick has its own two lines.
  `TestReleaseMapEditorHelpLinesFitTheRail` measures every row with the
  installed font against the rail width of 300.
- `mapedit -new <W>x<H>` opens the window on the new map; with `-check` it
  prints the records and the report as `-map` does. `-new <W>x<H> -out <file>`
  writes the blank map to a new file through the fenced writer of Save As and
  prints the report; it writes whatever the report holds (`DIV-3211`).
- Save As of a new map has no input to protect. Save into a mod writes the
  new map to the mod's mission entry whose map file the folder does not hold
  yet and refuses when no or several entries are unwritten (`DIV-3210`). Both
  run the report gate first.

### The report

No new checker. The `drop-cell` rule (`DIV-3212`) also names a map with no
usable drop cell when the fallback square (30 to 100 on each axis,
`MISSION-DROP-002`) leaves the map or lands on blocked cells: a 32 by 32 map
without a drop cell reports `warning drop-cell`, with 5041 of 5041 cells lost;
a 128 or 256 map reports nothing, its fallback square being open ground. The
rule stays a warning, so a map without a drop cell can be saved and the party
then lands off the map.

## Proof

- `pkg/formats/alm` (`newmap_test.go`): each default against the claim's value
  written out in the test, the group mask for each fill, every side of the list
  opens and the document writes back its bytes, refused specs.
- `pkg/mapedit` (`newmap_test.go`): the session's bytes are the constructor's,
  reopening writes them back, a unit and a sack go in through the section
  table and undo returns the blank bytes.
- `pkg/ui` (`mapeditornew_test.go`): N opens the catalogue at the top, U needs
  a host that adds units, the class prompt calls `AddUnit` with the clicked
  cell.
- Release, EN and RU, through the production route:
  `TestReleaseMapEditorNewMapPlaysAsModMission` opens `new/32x32` in the
  editor window host, adds a unit, adds and undoes a second, and saves into a
  temporary mod (the file did not exist before). The saved file is the model's
  bytes; it reopens in a second editor with one unit and an empty report; Save
  As of the reopened map writes the same bytes. Mission 20 is then started
  headless from that map as the mod's replacing mission: the party stands on the
  drop cell (16,16), the unit stands on its cell with owner 2, a click on open
  ground queues one move order and the party walks to the clicked cell. Without
  the mod the same mission is the installed map.
  `TestReleaseMapEditorListsAndOpensEveryNewMapSize`,
  `TestReleaseMapEditorBlankWriteIsTheConstructorsBytes`,
  `TestReleaseMapEditorInstalledMapStillWritesItsInstalledBytes` (loss control:
  an installed map writes its installed bytes) and
  `TestReleaseNewWritesTheBlankMapAndPrintsItsReport` (`cmd/mapedit`).
- The release test adds the drop cell with the window's own operation
  (`SetDropCell`, which adds a drop action to a map that has none); the empty
  type-7 record of a new map takes it.

## Open debt

- A win. A mission's win needs a win arm (an instant-4 trigger); the editor
  retargets the arms a map has and creates none, so a new map plays and cannot
  be won.
- A new map with no drop cell is only a warning (`DIV-3212`); the owner may
  rule it an error by editing the rule row.
- Terrain and height of a new map are one fill; painting is story 1426.
- The unwritten bytes of the original's new map and the constructor values of
  its two players are not known (`DIV-3208`).
- Texts: a new map carries no event or tip text; the `texts` folder of the
  mission entry and `strings.toml` are written by hand.
