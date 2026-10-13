# The map editor paints objects

## Intent

The author paints the static objects of a map (the type-3 layer: trees, stones,
fences, bones) as they paint terrain. The kind is the grid brush; its one
builder is `mapedit.Brush` with its `Plane`. The objects layer is the third
plane value of that builder.

## Authority

`ALM-EDITDISP-279` (High / Medium, k301): the original's Objects tool stores one
byte per cell and writes no loot record. `ALM-CLS-035`: the byte is the class
code, resolving to `objects.reg` section index `code - 1`; 0 is no object.
Layout, radius and key are ours. Differences: `DIV-3274` (disc and clear row
against one cell per click), `DIV-3275` (the list). `DIV-3276` to `DIV-3281`
are returned unused.

## As built

- `mapedit.PlaneObject` joins `PlaneTile` and `PlaneHeight`. `Brush.Value` on
  that plane is the class code (0 clears). `Paint` writes the code over the
  disc into the type-3 payload and nothing else, as one edit; a stroke that
  changes no byte records nothing; a map without a type-3 record, or a code a
  byte cannot hold, is refused with nothing written. The model does not know
  which codes name a class: that is the host's list.
- `ui.BrushStroke` carries `Plane ui.BrushPlane` (terrain, height, objects)
  where it carried `Height bool`. `ui.InspectionBrush` gains `BrushObjects()`
  beside `BrushTerrains()`. The host maps the window's plane to the model's in
  one table (`brushPlanes`) and refuses an objects stroke whose code is not on
  its list.
- The list (`objectBrushRows`): the row "No object" (code 0), then every class
  of the installed `objects.reg` whose frame the game draws, in placement-byte
  order, named by `DescText` (or "Object <code>"). On EN and RU: 75 classes;
  7 registry classes have no art and stay off the list (the fire variants,
  placement bytes 2, 5, 8, 11, 14, 17, 20). The dead variants are offered.
- The window: mode `brushObjects` and the key B (`objects`) are rows of
  `editorKeyRows` and the brush tables; the help line of the keys screen reads
  "T, H, B: brush" and the objects screen has its own line. Entering and leaving
  go through `toggleBrush` and `leave`. `TestMapEditorModesHoldEveryKeyAndGesture`
  is re-recorded: a column and three paths for the new key, and the keys help
  line in every probe.
- Fix found by the modes hotfix: with the report open and a brush active the
  panel draw indexed the brush palette by the report's row; the draw now
  follows the row count's own order (report first).
  `TestMapEditorPanelDrawsAReportLongerThanTheBrushPalette` panicked before.
  The recorded "panic" cases (37 cells of the table) now record a panel hash.
- Passability: a type-3 cell that is non-zero is the `scenery` arm of
  `mapload.Passability` (ground-blocking); the mission start (`CellPlanes`) and
  `OriginalTerrainPlanes` read the same plane. A painted object blocks exactly
  as an installed one: both are the same byte in the same plane, and the saved
  map loads through the same path. The map check's `unit-terrain` and
  `drop-cell` rules read `mapload.Passability`, so a unit or drop cell on a
  painted object is judged with no new rule row.
- The view: the editor document is rebuilt from the model bytes, so the viewer
  draws the painted classes through the same static-object path as an installed
  map; undo and redo refresh the view because the document compare includes the
  overlay.

## Proof

- `pkg/mapedit`: the stroke writes the class code over the disc on the type-3
  payload only; the clear row writes 0; undo restores the bytes; an unchanged
  stroke records nothing; a map with no type-3 record refuses. On main the
  package does not compile with these tests (`PlaneObject` is undefined).
- EN and RU, `TestReleaseMapEditorObjectStrokeChangesOnlyTheTypeThreeCells`:
  load `scenario/10.alm`, paint one stroke, Save As to a temporary file: 13
  bytes differ, all type-3 cells (payload at file offset 19932), and the
  editor's save changes nothing else.
- EN and RU, `TestReleaseMapEditorPaintedObjectsLoadAsAModMissionAndBlockTheWalk`:
  a band of seven painted objects in a guest mod mission loads at its cells,
  blocks ground, and the party leader walks around it; the unedited map walks
  through (loss control).
- `TestReleaseMapEditorPaintedObjectBlocksLikeAnInstalledOne`: every installed
  object cell blocks; a painted one blocks; a drop cell on it is reported.

## Open debt

- The original's editor may offer a subset of the registry through
  `InMapEditor`; no claim states it (`DIV-3275`).
- The 64 head-of-buffer residue cells of `ALM-CLS-051` are not touched.
