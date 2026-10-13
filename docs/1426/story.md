# The map editor paints terrain and heights

## Intent

In the first game's map editor the author paints a terrain class with a brush
and raises or lowers heights with a brush. Painted cells get transitions toward
their neighbours. The saved map plays as a mod mission: painted cells carry
their class's movement cost and painted heights reach the height plane. The
story started at public main `49d36217` (pin k287).

## Authority

Owner direction (map editor, priority). Claims: `TERR-242` (the terrain tool
stores a palette word and runs the fixer on the cell), `TERR-243` (the fixer
rewrites differing neighbours), `TERR-244` (the fixer keeps the class),
`TERR-245` (the height tool: 0..127, row step bound 31), `ALM-SAVE-238` (save
normalisation), `ALM-GRID-012`, `ALM-GRID-013`, `TERR-PASS-049` and
`TERR-PASS-050` (the class of a tile word). Rows: `DIV-3199` (the fitted
table), `DIV-3200` (hub reading and water), `DIV-3201` (variant by position),
`DIV-3202` (impassable flag and save normalisation), `DIV-3203` (radius),
`DIV-3204` (height step, fall-off, row bound), `DIV-3205` (viewer rebuild).
`DIV-3206` is returned unused.

## As built

### One brush operation

`mapedit.Editor.Paint(Brush, []Cell, classifier)` (`pkg/mapedit/brush.go`) is
the one stroke for both planes. A `Brush` is data: plane (`PlaneTile`,
`PlaneHeight`), value (a class, or a signed height change), radius, fall-off
(`FalloffNone`, `FalloffCosine`). A stroke is one edit, so one undo step with
exact bytes. A stroke that changes nothing records nothing.

| stroke | bytes it writes |
|---|---|
| tile | type-1 tile words of the disc and of its 8-neighbourhood, as runs; the top 8 rows when the stroke changes them or the last cell; the type-0 tile-group mask (`+0x18`) when it changes |
| height | type-2 height bytes of the disc, as runs |

A tile stroke names no other plane and no record. A height stroke names the
height plane only.

### Transitions

`mapedit.Derive(class, around, x, y)` is the one function (`pkg/mapedit/terrain.go`);
`shapeTable` (`pkg/mapedit/terraintable.go`) is its data: for a cell on the
primary side of a strip, on the secondary side, and for water, one shape per
mask of the eight neighbours of the partner class. The strip pairs, palette
words and hub classes are data in `terrain.go`. `pkg/mapedit` imports only
`pkg/formats/alm`; the classifier is injected, and the host passes
`mapload.TileClass`. The table is regenerated from the installed planes with
`AGAINROM_ASSETS=<root> AGAINROM_FIT_TABLE=<path> go test ./pkg/game -run
TestFitTransitionTable`.

### The window

`pkg/ui/mapeditorbrush.go`. T switches terrain mode, H height mode (the same
key leaves it, as does Escape). The rail lists the palette (terrain mode) or
the rows Raise and Lower (height mode) and shows a status line. `[` and `]` set
the radius (0 to 8, default 2). A press on the canvas starts a stroke, the
cells the pointer crosses join it, and the release commits it through the
optional host seam `InspectionBrush` (`BrushTerrains`, `Paint`). The marker
outlines the cells under the brush. Help lines: "T: off / Drag: paint / [ ]:
size" and "H: off / Drag: shape / [ ]: size", both within the 300-pixel rail
(`TestMapEditorBrushHelpLinesFitTheRail`). The host is
`pkg/game/mapinspectionbrush.go`.

### The report

Unchanged. Save As and Save into a mod call the story 1423 `Report` on the
painted bytes; no rule or checker was added.

### Installed maps

Open and save without a stroke writes the installed bytes
(`TestReleaseInstalledMapsSaveUnchangedWithoutAStroke`, every installed map).

## Proof

Focused: `pkg/mapedit/brush_test.go` (a stroke writes only the planes it
names, the transition rule on synthetic neighbourhoods, undo and redo, loss
controls), `pkg/ui/mapeditorbrush_test.go` (one drag, one stroke; the view is
kept; mode switch; help width).

EN and RU release tests (`pkg/game`):

- `TestTerrainTableWordsClassifyAsTheTableSays`: every word the table writes
  classifies as the cell's class.
- `TestReleaseTerrainRuleOverInstalledMaps`: the rule against the installed
  planes (a measurement; floors 70 percent shape and 90 percent pure, not a
  tuning target). EN: 859,768 cells; 471,539 with a differing neighbour, 367,099
  (77.9 percent) with the installed shape; 388,086 with a uniform neighbourhood,
  374,263 pure. RU: 353,874 with a differing neighbour, 275,516 (77.9 percent).
- `TestReleaseTerrainTableHeldOut`: fitted on even-numbered maps, scored on
  odd-numbered ones: EN 70.3 / 87.8 / 94.1 percent, RU 71.0 / 88.1 / 93.7
  (primary side, secondary side, water).
- `TestReleaseMapEditorPaintedTerrainChangesWhereAPartyWalks`: paints a
  7-cell mountain band across a 10-cell walking line of the guest mission with
  pointer strokes in the window, saves into a temporary mod and starts the
  mission headless. The leader walks around the band; grass painted over the
  band is crossed. Other planes and units are unchanged.
- `TestReleaseMapEditorPaintedHeightsReachTheHeightPlane`: a radius-0 raise
  of 8 changes the saved height plane by exactly 8 at that cell and
  `mapload.Height` reads it.
- `TestReleaseTerrainAndHeightStrokesUndoExactlyAndReport`: three installed
  maps; undo and redo are byte exact; the report finds no error on the painted
  map.
- `TestReleaseInstalledMapsSaveUnchangedWithoutAStroke`.

## Open debt

- The original fixer's pair table and variant table are not in the claims;
  `DIV-3199` and `DIV-3201` stand until a claim names them.
- The height tool adds a step per stroke; the original sets an absolute
  anchor height (`DIV-3204`).
- A stroke rebuilds the viewer (`DIV-3205`).
- Words outside the table (shapes the installed maps hold that the table does
  not name) are left alone beside a stroke, not rewritten.
