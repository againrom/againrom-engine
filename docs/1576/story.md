# Every speaker field picks on the map; each placeable record lists in one palette

## Intent

The map editor's bar shows three palettes of placeable records: «Постройки» (Structures),
«Объекты» (Objects) and «Деревья» (Trees). Before, every one of the 75 installed object classes
listed in both Objects and Trees, and the 16 decoration structure kinds listed in Structures. Now
each record lists in exactly one palette by the owner's rule, a placed record keeps its record
kind, placement writes the same bytes from the new palette as from the old one, and a palette
category with no record says so in EN and RU. The Trees window keeps both strokes: Forest, which
plants on open cells only, and One kind, a stroke of one tree class that writes the class on every
covered cell, over objects, roads, water, structure footprints, the drop cell and units. The One
kind stroke matches the object brush's row of that class at engine `5f63a70d7`; Forest is
unchanged. Its rows come from `objectRowsIn(true)` and it commits through `brushCommit` to
`objectStroke`.

Every speaker field already picks the speaker by a click on the map through the one unit pick;
this story's census names each place and holds it with a test.

## Authority

- Owner rule (story brief): «Постройки» holds what can be selected, destroyed or switched, plus
  the bridges; «Объекты» holds decoration and no tree; «Деревья» holds the trees; no record lists
  in two palettes; a record the data does not settle lists in «Постройки», named with the field
  that fails; a placed record keeps its kind.
- Claims the palette table cites: TERR-STRUCT-105 (VariableSize marks the two wooden bridges),
  UNIT-153 and UNIT-STRUCTHIT-138 (a Usable class is a hover hit and the target of the use
  order), MISSION-065 and AI-SELECT-122 (a plain click selects a destructible structure),
  TERR-STRUCT-210 (Indestructible classes with positive maximum health), REG-OBJ-047
  (DeadObject names the class a destroyed object leaves).
- Census, EN = RU, base `7401537ae`, map `scenario/10.alm` of each original: before, Structures
  66 kinds, Objects 75 classes, Trees 75 classes, 75 records in two palettes; after, Structures
  50 kinds, Objects 16 kinds and 33 classes, Trees 42 classes, none in two palettes. The census
  file is the lane's `CENSUS.md`; the release test below repeats its counts.
- Unsettled kinds, listed in Structures: 8 Well 1 (Indestructible with Buildings maximum health
  1000), 27 Magic Symbol, 38 Horisontal Bridge, 48 Teleport Pad, 52 Vertical Bridge, 53 Campfire
  (Flat: no field marks a fixed bridge).
- The map editor is our program: no divergence row.

## As built

| kind | builder |
|---|---|
| palette of a structure kind and an object class | `pkg/game/mapeditorpalettes.go`: `structurePalette`, `objectPalette`, `structurePalettes`, the one table; the Add step's `InspectionNewStep.Palettes` and the object list's `BrushObject.Palette` carry it as data |
| maximum health of a structure kind | `pkg/mapload/structures.go` `StructureMaxHealth`, which `Structures` now also seeds `Field42` from (same entry, same position) |
| structure kind step of the Add window | `pkg/game/mapinspectionadd.go` `NewRecordStep` (`kind`): the choices unchanged, plus their palettes |
| object list rows | `pkg/game/mapinspectionbrush.go` `objectBrushRows`: each row's palette from the object registry |
| the Structures grid and the Objects grid of structure kinds | `pkg/ui/mapeditorplace.go` `placeGridOf`, `placeShown`, `placeIn`: the one place grid; in the object brush's one-kind mode it shows the Objects palette's structure kinds, and a press places copies through the one Add window (`placeChoose`) |
| object brush rows and the Forest grid | `pkg/ui/mapeditorpalettes.go` `brushObjectRows`, `objectRowsIn`; `pkg/ui/mapeditorforest.go` `forestKinds` (Trees palette only); `setForest` starts a switched list on its clear row |
| what a palette holds | `pkg/ui/mapeditorpalettes.go` `PaletteHolds` (headless reader) and `emptyPaletteNote` (the four empty-category words) |
| Esc after placed copies | `pkg/ui/mapeditorcancel.go` `cancelPopup`: a placing the object brush began ends back in the brush (`addRun.brush`) |
| a speaker pick on the map | `pkg/ui/mapeditorparampick.go` `targetSpeaker`, unchanged |
| words | `pkg/words/text/{en,ru}/strings.toml`: `editor.palette.nostructures`, `noobjects`, `nodecoration`, `notrees` |
| static guard | `internal/archtest/editorpalettes_test.go` |

Speaker places, each reaching the message popup's speaker row and its one pick (`targetSpeaker`);
none lacked the pick:

1. the Action row's Open link of an existing message;
2. the trigger wizard's Configure of a dialog item;
3. Details of an action row, then its Open link;
4. a message field's «New message» choice;
5. Scripts, Add, Message;
6. a record field with `NewMessageChoice` (the same `beginMessage` path as 4).

## Proof

Focused tests, at 1920x1080:

- `pkg/ui/mapeditorpaletteholds_test.go`: `TestMapEditorEachPlaceableRecordListsInOnePalette`,
  `TestMapEditorDecorationStructureIsFoundByTheSearchAndTheTree`,
  `TestMapEditorEmptyPaletteSaysSoInPlainWords` (EN and RU),
  `TestMapEditorObjectsGridPlacesTheAddWindowsRecord`.
- `pkg/game/mapeditorpalettes_test.go`: `TestEditorPaletteTableSettlesEachRecord`,
  `TestEditorPalettesHoldTheTablesRecords`, `TestEditorPaletteRoutesWriteTheSameBytes` (the
  fence from the Objects grid writes the Add window's bytes and stays a structure record; Forest,
  one kind, solid, one cell, writes the object brush row's bytes; opening every palette and
  saving with no edit writes the opened bytes).
- `pkg/game/mapeditpalettetrees_test.go`: `TestEditorTreesOneKindStrokeWritesTheObjectBrushBytes`
  (the Trees One kind stroke of the oak writes the same bytes as a direct object stroke on a cell
  holding another object, a road, water, a structure footprint, the drop cell and a unit).
- `pkg/ui/mapeditortreesonekind_test.go`: the Trees window shows the One kind button and the tree
  rows inside the map area at 1920x1080, EN and RU.
- `pkg/game/mapeditorspeakerplaces_test.go`: `TestEverySpeakerPlaceOffersTheOnePick` (EN and RU;
  the five routes above; the lady refused with the same cause, the fighter written as speaker).
- `internal/archtest`: `TestEditorPalettesAndSpeakerPickHaveOneBuilder` and its self-test
  `TestEditorPaletteGuardSeesASecondBuilder`: no source outside `pkg/ui` and the palette table
  names a palette, and no source outside the parameter pick asks `UnitSpeaker`.

The palette tests fail to compile on the base tree (`7401537ae`, exported with `git archive`, the
new files copied in). The speaker test passes on the base tree: it is a census of a behaviour
that already held.

Release test, not run in the lane:
`pkg/game.TestReleaseMapEditorEachPaletteRecordListsInOnePalette` (population row) counts the
installed palettes, checks that each record lists once and that the six unsettled kinds list in
Structures. The changed release tests `mapeditforest_release_test.go`,
`mapeditobjectart_release_test.go` and `mapeditobjects_release_test.go` read the Trees and
one-kind lists. The next release runs them on EN and RU.

Goldens: commit `f7f9228e2` re-recorded 61 of 112 rows of the map editor modes golden: column 9
(`objects`) for this story's objects probes (the modes host offers no decoration structure kind,
so the object brush window gains the row «No decoration structure lists here.»), three objects
probes in more columns, and the three story 1575 rows (`full/add-script`, and column 38 of
`full/triggers` and `full/scripts`). The correction moves no golden row: the modes probes never
open the Trees window's One kind mode, and the ordinary modes walk, the readable table, the popup
placement transcript and the side panels golden pass unchanged.

## Open debt

- Out of scope: the one-click trigger, the «Сценарий» palette list, the «Условия» and
  «Переменные» lists, speaker records a mod adds, the second game's installs, and any change to
  what the game does with a structure.
- The bushes (Bush 1..3 and their dead classes, 6 classes) list in Trees: they carry DeadObject or
  FireObject -2 as the trees do, and no field tells a bush from a tree.
- The MCP forest validation checks only that a class is offered (`forestKinds`); it does not name
  the palette.
- The passability release test's `brushWindow("objects", 1)` now paints the one-kind list's first
  class, Big stone 1 (code 22), not a tree.
- In the focused RU tests only the editor's words switch; host-supplied labels («New message»,
  «No speaker») stay EN.
