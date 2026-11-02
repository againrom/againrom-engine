# New terrain tiles in a mod

## Intent

A mod adds terrain tile groups 13 to 31. Each group is one row of the mod's `data/terrain.toml`,
naming its land kind, its name in the map editor's terrain list and an optional minimap colour,
and four strips under `files/common/` at the tile loader's paths:

```toml
[[group]]
group = 13
kind = "grass"
name = "Snow"
minimap = "#e8f0f8"
```

with `files/common/graphics/terrain/tile4-04.bmp` to `tile4-07.bmp`. The check holds each new
strip to the loader's strip rule and refuses a strip with no row and a row with no strip, naming
the mod and the file. The game draws the group, walks it by its row's kind and shows it on the
minimap; a map naming a group no active mod adds is refused at load naming the group and the map.
The map editor with the mod open lists the groups and paints their whole cells; the saved words
and mask carry them; Publish packs the strips and the data file. The format is documented in
`docs/MODS.md`, "New terrain tiles".

## Authority

- Owner direction (story brief): a mod may add groups 13..31 with their strips and their land
  kind as mod data, one row per group; a mod that adds a group changes game state.
- TERR-TILE-291 (High / Medium): a tile word picks slot 4*((w&0x1fff)>>6)+((w>>4)&3) and frame
  w&15; the loader holds 128 slots, 32 groups. TERR-TILE-292: a set mask bit builds its four slots
  from tile paths; a path no archive supplies is fatal.
- ALM-TILE-336 (High): the installed maps use groups 0..12. ALM-TILE-338: the paths of groups
  13..31 are absent from the installs.
- TERR-TILE-293 (High / Medium): the classifier and raw water test see w&0x3ff, so a group g>=16
  takes the class, cost and block of group g mod 16. Againrom gives a mod group its row's kind
  instead (DIV-4413).
- ALM-SAVE-238 and ALM-TILE-337: the original editor's save clears bits 10..12 and caps +0x18 at
  0x1fff. Againrom keeps the words and the mask bits of the mod groups (DIV-4412).
- TERR-TILE-294 (frames 14 and 15 of a 14-frame strip read past it), TERR-TILE-295 (a null slot
  is read) and TERR-TILE-296 (the minimap lookup is unbounded): Againrom bounds a strip at 14
  frames and refuses at load a cell naming a frame its strip lacks, so neither read is reached.
- The load refusal is stated in `DIV-4412`'s implemented behaviour: where the original stops with a
  fatal error on an absent strip (TERR-TILE-292), Againrom refuses the map before play, naming the
  group and the map; both stop the map from running. No other row states it.

## As built

| kind | builder |
|---|---|
| mod data kind | `pkg/mod/kinds.go` `dataKinds` gains `terrain` (`data/terrain.toml`, key `group`, configuration, own layer), placed before `missions` so every mission's map check runs with the rows applied; `pkg/mod/terrain.go` `ParseTerrain` and `readTerrain` read the rows into `Data.Terrain` and refuse a group two mods add. The one game applier map `pkg/game/moddata.go` gains `terrain` → `FrontEnd.setModTerrain`, which sets `Table.Mods.Terrain` |
| install file service (strip loader input) | `pkg/game/modfiles.go` `serveInstallFiles`: a new path of a mod group's strip is held to `terrain.ReadStrip` and served; every other new path is refused as before. `pkg/game/modterrain.go` `modTerrain` pairs strips and rows of one mod and fills each row's frame counts |
| strip loader | `pkg/render/terrain/tileset.go` `LoadTileset` reads every strip through the new `ReadStrip` (8-bit, 32 px wide, whole cells, at most 14); `GroupStripPath` and `ModGroupStrip` are the path formula of `TilePath` for one group and column. No second loader: the game, the minimap and the editor read the strips through `LoadTileset` |
| land kind of a cell (classifier) | `pkg/mapload/passability.go` `classifyCosts`, `rawWater` and `cellArms` take the rows; `TerrainPassability`, `costPlane`, `Planes` and `OriginalTerrainPlanesWith` pass `Table.ModTerrain()`. A row's kind is data; no branch names a group |
| constructor planes at LOAD | `pkg/game/originalcellplanes.go` `readOriginalCellPlanes` takes the rows; `importOriginalCellPlanes`, `currentSpatialPlanes`, `constructCurrentBuildings` and the plane residue pass the table's rows |
| map load check | `pkg/mapload/tilegroups.go` `CheckTileGroups`, called by `FromALMRoster` whenever a table is given |
| map check rules (launch, `-check`, Publish, Test play) | the rule table in `pkg/game/mapcheck.go`: `terrain-planes` builds the planes with `OriginalTerrainPlanesWith` and the inspector table's `ModTerrain()`; `mission-build` runs after the `terrain` kind has applied |
| whole-map terrain edit | `pkg/mapedit/wholemap.go` `RandomizeTerrain` through `lookalikes`, which answers no look-alike for a word `opaqueWord` names (the corner rule's predicate), so Random terrain leaves every mod-group cell as it is |
| group mask writer | `pkg/formats/alm/tilegroup.go` `TileGroupMask`, the one mask writer, keeps the install rule and adds bit g for a mod group in use |
| map editor brush | `pkg/mapedit/brush.go` `Brush.Words` and `tileStroke` hand a group brush to `groupStroke` (`pkg/mapedit/groupbrush.go`), which paints whole cells through the same `tileEdit` and pattern pick (`patternIndex`) as the install's terrains, each cell taking one word among every frame of the group's four strips; `wordQuad` answers no corners for a mod group, so `TileFix` leaves its cells as they are |
| map editor terrain list | `pkg/game/mapinspectionbrush.go` `BrushTerrains` appends `modBrushTerrains`; `paintStroke` gives a group brush the row's words (`groupWords`); `MapInspector.SetMod` runs `useModTerrain`, which loads the mod's strips into the editor's tileset and its rows into the editor's table |
| minimap colour | `pkg/ui/minimaporiginal.go` `MinimapPaint.GroupColours`, filled by `minimapGroupColours` in `missionMinimapPaint` |
| Publish | unchanged: `PackMod` packs the whole mod folder and the admission check serves its files and data |

## Proof

Focused ordinary tests; each fails to build at `4bd09a258` (`ParseTerrain`, `TerrainPassability`,
`CheckTileGroups`, `GroupBrush`, `ReadStrip` and `modTerrain` do not exist there). Strips are made
by the tests; no install byte is read. The fixture mod is `pkg/modrt/testdata/mods/snow-field`
(rows for group 13 grass and group 31 mountain).

- `pkg/mod/terrain_test.go` `TestTerrainRowsReadAndRefuse`: rows read; a group outside 13..31, a
  second row, a missing field, an unknown kind and a bad colour refused with file and line.
- `pkg/game/modterrain_test.go` `TestModTerrainGroupsAreAccepted`: the mod with groups 13 and 31
  is accepted and marked state-changing; the tileset holds slots 52..55 and 124..127; a snow word
  draws a cell. `TestModTerrainCheckRefuses`: a strip 33 px wide, a strip of 15 frames, a strip of
  group 14 with no row and a row 31 missing column 3 are refused naming the mod and the file.
  `TestEditorPaintsAModGroup`: the editor's brush paints group 13; the words and mask carry it;
  the load check accepts the map with the rows and refuses it without; the terrain list holds
  Snow and Ice.
- `pkg/mapload/tilegroups_test.go` `TestModGroupCellsTakeTheirRowsLandKind`: a group 13 grass and
  a group 31 mountain cell take their rows' class, cost and block in the block plane, the cost
  plane and the constructor planes. `TestMapNamingAGroupWithNoPictureIsRefused`: a cell, a mask
  bit and a frame with no picture refused naming the group and the map.
- `pkg/mapedit/groupbrush_test.go` `TestModGroupBrushPaintsWholeCellsAndSaves`: whole cells,
  neighbours kept, mask bits 13 and 31, the saved form reopens with the same words and mask.
- `pkg/formats/alm/tilegroup_test.go` `TestTileGroupMaskKeepsModGroups`.
- `pkg/game/modgroupmission_test.go` `TestModGroupMissionPassesTheMapChecks`: on the cell-state
  fixture front, which carries a terrain cost table, a mod with the snow-field rows and a group 20
  row ships a mission replacing mission 10 on a map with group 13, 20 and 31 cells. The launch
  route (files served, `SetMods`, `SetModData`, whose `missions` applier runs the map check)
  accepts it, and `CheckModMap` over the table after the launch (the Publish and Test-play gate)
  gives no error. The same map under the mod without the group 20 row is refused naming
  `mission/map.alm` and terrain group 20. At `8b9dc1fd9` the launch refuses the first mod with
  `terrain-planes map: original terrain cell 1414 uses an uninitialized terrain pair`.
- `pkg/mapedit/groupbrush_test.go` `TestRandomTerrainKeepsModGroupCells`: cells of groups 13, 16,
  20, 24, 28 and 31 painted with `GroupBrush` stay as they are under `RandomizeTerrain` with a
  source answering n-1; the install's cells, a sand cell among them, take the words the same map
  without the mod cells gives. At `8b9dc1fd9` groups 16, 20, 24 and 28 change (`0x0401` to
  `0x0424` and the like).
- `pkg/modrt`: `TestDataLoadRefusesOtherFilesAndSecondLoads` lists `data/terrain.toml` among the
  loadable files; `testdata/merged-data.txt` holds the snow-field fixture (its rules and mark lines,
  folder and packed, EN and RU) and the `all` set's new mark. The `terrain` kind's move adds no line.
- `pkg/game/modterrainsave_test.go` `TestModGroupMissionSavesAndLoads`: a mission on the fixture
  map with group 31 and group 13 cells opens with their rows' planes, SAVEs, LOADs on a fresh
  front into the same words and planes, and both worlds stay equal for ten ticks; the same SAV
  without the mod is refused naming the map file and the group.

Release tests, not run in the lane (EN and RU installs): `TestReleaseCellPlanesSurviveSaveColdLoadAndTicks`,
`TestReleaseEveryMissionStartHoldsTheInstalledCellPlanes`, `TestReleaseOriginalCellPlanes1115NativeContinuation`,
`TestReleaseMapEditorPaintedTerrainChangesWhereAPartyWalks` and `TestReleaseMapEditorBrushFrames`
(`pkg/game`). They read the install's groups 0..12, whose route the change passes unchanged rows
to.

## Open debt

- The viewer's block plane (`pkg/game/mapload.go`), the original-save census
  (`pkg/game/originalsave.go`) and the original structure grid (`pkg/game/originalstructures.go`)
  read no mod rows; on a map with mod groups they classify those cells as the original would.
- `ExportCurrentSave` (`pkg/game/currentsave.go`) builds `terrainBase` from `mapload.Cost(m)`,
  which reads no rows, so a mod-group cell whose cost differs from the install reading is written
  as a terrain deviation row. The SAV round trip holds.
- The map editor keeps the previous mod's groups in its terrain list after a failed mod switch
  (`SetMod` whose load or strip check fails, or `openWorkspace`'s revert); the mod is then
  read-only, so no save is reached.
- A mod group of kind `water` blocks and walks as water but is not animated, makes no water sound
  (`pkg/ui/ambient.go`) and takes no shore blend.
- The editor's minimap is not given the rows' colours.
- A mod group's cells take no corner blend with their neighbours in the editor.
- No release test plays a mod with new groups on an install.
