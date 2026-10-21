# Map editor painting, brush and map border

## Intent and authority

Owner direction: painted Sand, Land and Stones showed hard cell-sized edges; the brush size changed
only on `[` and `]` and was not shown; the brush had one shape; the 8-cell impassable border was
not visible. The story fixes the edges, adds a visible size with its controls, a Circle and a
Square shape for every brush plane, and a border view.

Authority for the terrain rule: `TERR-243` (the fixer rewrites each differing 3x3 neighbour from a
pair table and writes `base(cc)` where the table has no entry) and `TERR-252` (Medium: the pair
table's words are a family base plus a state code of four corner occupancies). No claim lists the
state code of each word, so the words of each state are read from the installed tile planes
(`DIV-3199`). Authority for the border: the game's own rule, `mapload.InBorderRing` (8 cells),
which the editor reads through the passability arm `ui.PassBorder`; no constant is added.

## Defect classes and fixes

The old stroke derived each touched cell's word from the classes of its eight neighbours through a
table fitted to the installed planes (`mapedit.Derive`). Every class below followed from that: the
rule read cell classes, while the tile pictures join at shared corners.

| Class | Seen | Cause | Fix |
|---|---|---|---|
| Pure cell beside pure cell | A painted block's outer cells met the neighbours' pure words with a straight edge | The table gave the pure word to a cell on a strip's secondary side | Every corner of a painted cell takes the class in every cell that shares it; each neighbour takes the word that shows its corners |
| No strip between two classes | Sand beside Stones, Mountain beside Land: one side blended, the other met a third class's pure cell | The neighbour of an unpaired class was read as a hub class on one side only | The corner farthest in strip steps from the painted class moves one step nearer until every cell is a word's; the inserted ring is `base(c)` where `TERR-243` names it (`DIV-3200`) |
| Water | Hard edge at a shore | Land cells ignored water neighbours | Water and Land are one family, so shore words are written on both sides |
| One-sided blending | Straight seams at Mountain and Cracked strokes | Only the strip's primary side took transition words | The corner rule writes both sides of every shared corner |

Flowers and Grass have no strip, and the installed maps join them with none: 32,187 shared EN corners
show Flowers beside Grass. The stroke treats the two as one at a corner (`DIV-3821`), so Flowers
over Land shows the Grass and Land strip, as `base(Flowers)` = Grass gives in the original.

## Reach of a stroke

A stroke changes terrain only in its footprint and the 4 cells around it. The bound is the
transition rings the corner rule needs: the two classes most strip steps apart, Mountain and
Flowers, need 3 inserted rings (Stones, Land, Grass), so `mapedit.Reach` = 3 corner rings, counted
on the square distance from the corners of the painted cells. A corner farther out never moves
(`TileFix.set` refuses it), so no word changes more than `Reach`+1 = 4 cells from the footprint.
`Reach` is computed from the strip table, not written as a constant.

Inside the bound, a corner one strip step from the painted class moves only in the first ring
around the painted cells. Three classes that strips join pairwise (Cracked, Stones, Land) have no
word that shows all three; unbounded, the rule moved one corner after another along a Stones/Land
or Stones/Mountain seam, and a one-cell Cracked click on `scenario/31.alm` rewrote 2,359 words up
to 34 cells away and opened 272 closed cells (review R1). A cell the rule leaves with corners no
word shows takes the installed word that differs from its corners at the fewest corners, its own
word winning a tie (`nearestQuad`), and `TileFix.Seams` counts it. That leaves a hard corner at
the junction; the alternative was the walk.

Over the sweep below (four installed maps per root, every palette class, sizes 0, 1, 3, 8, both
shapes, seven to eight positions):

| Root | Strokes | Farthest rewritten word | Strokes past the bound | Cells with the nearest word | New hard corners | Unpainted class changes |
|---|---|---|---|---|---|---|
| EN, bounded | 2,000 | 4 | 0 | 778 in 205 strokes | 3,480 | 57,656 |
| EN, unbounded | 2,000 | 63 | 184 | 0 | 1,903 | 129,047 |
| RU, bounded | 2,480 | 4 | 0 | 2,288 in 236 strokes | 8,267 | 69,772 |
| RU, unbounded | 2,480 | 82 | 233 | 0 | 4,414 | 136,611 |

"New hard corners" counts, around every rewritten cell, corners where neighbouring cells show two
classes and did not before (`hardEdges`). The bound roughly doubles them and halves the class
changes. The review's click now rewrites 25 words, changes 12 unpainted cells' class, reaches 3
cells and opens 5 cells, all within 3 cells of the click (EN and RU alike). The click on the base
commit `f8e8606c` rewrote 5 words. Instrument: `TestReleaseMapEditorStrokeStaysNearTheBrush`;
unbounded rows from the same test with the bound removed by an overlay
(`review/story1481-editor-brush/loss/`).

## Rule comparison over the installed maps

Population: every cell of the installed maps (EN 38, RU 34) at least 8 cells from the edge whose
eight neighbours include another class. Each rule predicts the cell's four corners from its
neighbours; the share is the cells whose installed word shows the predicted corners.
Instrument: `TestReleaseTerrainRuleOverInstalledMaps` (corner rule) and, for the removed table, the
same counter run on the base commit with `mapedit.Derive` as predictor.

| Root | Corner rule | Corner rule, Flowers read as Grass | Fitted table |
|---|---|---|---|
| EN | 407,922 of 445,331 (91.6%) | 425,207 (95.5%) | 345,005 (77.5%) |
| RU | 306,300 of 333,009 (92.0%) | 318,415 (95.6%) | 258,041 (77.5%) |

Per family (EN, Flowers read as Grass), the corner rule leads on every strip, by 4 to 39 points
(Flowers/Savanna 77.8 against 38.8, Grass/Land 93.4 against 61.3, Savanna/Land 92.7 against 66.0, Stones/Land 98.1 against 67.7,
Mountain/Stones 98.7 against 80.7, Water 97.9 against 93.9). The table leads on pure cells of Road,
Cracked, Savanna and Mountain by 0.3 to 0.6 points (RU 0.4 to 1.1); it is not kept there, because a cell
from the table and its neighbour from the corner rule need not share their corners. RU gives the
same order. Logs: `review/story1481-editor-brush/measure_both_{en,ru}.log`.

## Controls as built

- Size: the brush list's first row shows the size with `-` and `+`; Alt and the mouse wheel step
  it by one per notch; `[` and `]` still work. All three change one value, 0 to 16
  (`mapedit.MaxRadius`, from the host). The plain wheel still zooms.
- Shape: the list's second row, Circle (default) or Square, for the terrain, height, objects and
  passability brushes. Circle is the old disc; Square is every cell within the size on both axes,
  and a Square height stroke fades with that distance. Size and shape are fields of
  `mapedit.Brush`, and its one cell walk (`Brush.Footprint`) gives the stroke's cells and the
  marker's (`DIV-3203`).
- Marker: the outline of the exact cells the stroke will touch.
- Help: each brush's help starts with "Alt+wheel or - +: size", then the brush's own line with
  "[ ]: size". Where the status strip cannot hold both (the passability view naming every arm of
  the cell), it drops the size line and keeps the brush's own. The Circle and Square buttons carry
  their own names; a shape line does not fit the strip.
- Border: the map view dims the border cells and draws a line at the playable edge;
  View > Map border turns it off and on, on by default (`DIV-3822`). With the passability view on,
  its own fills replace the dimming and the status line names the border.

## Proof

- `pkg/mapedit`: `TestStrokeOfEachClassPairLeavesNoSeam` (every palette class over every other,
  sizes 0, 1, 3, both shapes), `TestStrokeInsertsTheBaseClassWhereNoStripJoins` (the fallback
  insert), `TestStrokesAcrossThreeClassesStayNearTheBrush` (three-class corners and 200 random
  strokes: no word past `Reach`+1 cells, no new hard corner unless `TileFix.Seams` counts a cell;
  the scripted Cracked stroke between Sand and Stones now leaves its junction as 6 disagreeing
  corner pairs where the walk had left none, the pin that moved),
  `TestThreeClassJunctionStaysNearTheStroke` (a one-cell Cracked stroke on a Stones band's edge in
  Land), `TestBrushShapeCells` (sizes 0, 1, 3, 16), `TestSquareHeightStrokeFadesOnTheSquareDistance`.
  Loss control: the three stroke tests, run on the base commit with the old derivation, fail
  (for example Sand over Grass, size 3: 77 corners disagree). With the bound removed (an overlay
  of `corner.go`), both reach tests fail: the junction walks 17 cells, the scripted strokes 8 and 9.
- `TestCircleStrokesKeepTheirSavedBytes`: height, object and passability strokes with Circle at
  sizes 0, 1, 3, 8 and 16 give the bytes the base commit gives.
- `pkg/ui`: `TestMapEditorBrushSizeControlsMoveOneSize` (buttons, Alt+wheel, brackets, bounds; the
  plain wheel zooms), `TestMapEditorBrushShapeButtonsReachTheStroke`,
  `TestMapEditorBrushMarkerOutlinesTheHostCells` (loss control: a host with a one-cell walk),
  `TestMapEditorBrushHelpNamesTheSizeControls`, `TestMapEditorBorderViewShowsTheBorderArm`.
- Release, EN and RU: `TestReleaseMapEditorStrokeStaysNearTheBrush` (the review's click through
  the window, and the sweep above: no word past 4 cells, no closed cell opened past 4 cells in
  `mapload.Passability`, every written word drawn, every painted cell its class; with the bound
  removed it fails on 184 EN and 233 RU strokes), `TestReleaseMapEditorStrokesLeaveNoSeam` (eight class pairs through the
  window, no new hard corner, undo restores the bytes), `TestReleaseMapEditorBorderIsTheGameBorder`
  (the view equals `mapload.InBorderRing` on the first and last installed map and on a 48x32
  map), `TestReleaseMapEditorBrushFrames` (owner frames, kept outside the repository).

## Open debt

- `DIV-3199`: the word of each corner state comes from the installed planes, not from a claim.
- `DIV-3200`: Land beside Flowers and Cracked beside Mountain insert another class than `base(c)`;
  a stroke stops at 4 cells and leaves hard corners at three-class junctions there.
- `DIV-3821`: Flowers and Grass are joined at a corner on the installed maps' evidence only.
