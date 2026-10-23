# Map editor: Forest mode of the object brush

## Intent

A map maker plants a forest with one drag instead of one object at a time.
The object brush gains a Forest mode: it scatters a chosen mix of object
classes at a set density on empty ground only, and the right button thins
the same kinds.

## Authority

Owner direction (editor layout and controls take no divergence row):

1. A Forest mode of the object brush on the object plane scatters a chosen
   mix of classes at a set density (sparse, medium, dense, solid, or a typed
   per cent), random anew for each stroke, on the one `Brush` walk, with the
   brush's own size and shape.
2. It plants only in empty object cells, never on water, roads or
   structures, and leaves standing objects as they are.
3. The kinds are ticked in the picture grid; each ticked kind has an equal
   chance; no weights.
4. The right button with the same brush thins: it removes ticked kinds in
   covered cells at the set share and leaves other objects.
5. One stroke, left or right, is one Undo step.

The cell rule is the data the game already reads for movement:

- empty: the cell's type-3 byte is 0 (`TERR-PASS-049` (e): a non-zero byte is
  a static object);
- water: the tile index is in the raw water range, `(index & 0x300) == 0x200`
  (`TERR-PASS-049` (d), the block plane's water arm);
- road: the classifier gives class Road (`TERR-PASS-050`, strip group 12 at
  blend level 3 or more), through `mapload.TileClass`, the classifier the
  tile brush already reads;
- structure: a placed structure's footprint attaches the cell
  (`TERR-STRUCT-068`), open deck and doorway cells included, through
  `mapload.StructureCells`, the same occupancy the world's block plane is
  built from.

The tool writes only object class codes the object list already offers, so a
map it writes stays one the original reads.

## As built

### Model

- `mapedit.Brush` gains `Scatter` (`pkg/mapedit/brush.go:67`): the ticked
  classes, the share in per cent, `Thin`, the random source `Draw` and the
  host's per-cell structure plane `Fenced`. An object stroke with classes is
  a forest stroke (`scatterStroke`, `pkg/mapedit/brush.go:367`).
- It walks the cells of `Brush.Footprint` (`:131`, over the one `walk`,
  `:108`) once each, in row order. Planting: an open cell takes a class when
  `Draw(100) < Share`, and the class is `Classes[Draw(len(Classes))]`.
  Thinning: a cell holding a ticked class is cleared when
  `Draw(100) < Share`; every other cell is left.
- It refuses a share outside 1..100, a missing source or classifier, a class
  outside 1..255 and a structure plane of the wrong size, and then writes
  nothing. The stroke is one edit (`apply`, `pkg/mapedit/editor.go:180`), so
  one Undo restores its bytes.

### Host

- `paintStroke` (`pkg/game/mapinspectionbrush.go:129`) builds the scatter
  from the window's `BrushForest`: a fresh `random.NewGo` source seeded by
  `strokeSeed` (`:108`) for each stroke, the seam tests fix, and for a
  planting stroke the structure plane from `mapload.StructureCells`
  (`pkg/mapload/structures.go:305`) under the installed table.
- `Paint` (`:76`) refuses a ticked code the object list does not offer.
  `BrushObjectIcon` (`:89`) gives the grid each class's installed frame.

### Window

- The object brush's popup gains a Tool row (One kind, Forest). In Forest
  mode it shows a Density row with the share and a `%` button, which asks
  for a per cent 1..100 in the editor's prompt, a row of presets (Sparse
  10 %, Medium 25 %, Dense 50 %, Solid 100 %), a hint line and the 1490 picture grid (`popupGrid`,
  `pkg/ui/editorpopup.go:56`) with one cell per offered class; a press
  ticks or unticks the class, as does a press on its list row, which then
  reads `[x]` or `[ ]`. The default share is Medium.
- A left drag plants; a right drag thins (`brushPointer`,
  `pkg/ui/mapeditorbrush.go:471`). Size, shape and the outline are the
  object brush's own. With no kind ticked a stroke writes nothing and the
  window says to tick one. A typed share outside 1..100 keeps the share and
  says why.
- New code is in `pkg/ui/mapeditorforest.go`; the hooks are in
  `brushPopupRows`, `brushRowButton`, `brushCommit` and the list popup's
  grid press and wheel (`pkg/ui/mapeditorwindow.go`).

### Builders reused

The popup builder and its `popupGrid` with ticks; the editor's `Brush` walk
and `Footprint`; the object brush's size, shape, outline and stroke path;
the model's one edit per `Paint` and its undo stack; the editor prompt for
the typed per cent; `ListButtonRect` and `PlaceGridCellRect` for drivers;
`strokeSeed` for randomness. No second instance of a kind was added.

## Proof

Fixture tests (`pkg/game/mapeditforest_test.go`) drive the window by
pointer and keys over a 40x40 map with water, a road row, a 3x2 structure
and standing objects, on fixed seeds:

| Test | Shows |
| --- | --- |
| `TestMapEditorForestPlantsTheShareOnOpenCellsOnly` | Sparse 10.0 %, Medium 25.4 %, Dense 45.5 %, Solid 100 % and typed 37 → 36.1 % of 986 open cells (bound ±5 points); each of three ticked kinds within a third ±0.1; no plant on water, road, structure or a standing object; one Undo restores the bytes |
| `TestMapEditorForestDragIsOneUndoStep` | a five-point drag is one Undo and one Redo |
| `TestMapEditorForestRightButtonThinsTickedKindsOnly` | Dense thinning removes 430 of 865 ticked objects (49.7 %), only ticked kinds, unticked objects stay; one Undo restores |
| `TestMapEditorForestRefusesNoKindAndABadShare` | no kind ticked writes nothing; a typed 0 is refused with a message |
| `TestForestWordsSurviveEditorEncoding` | all 13 words the story added, EN at font selector 0 and RU at selector 1, encode without a substitution; it failed on `6129744b` for the RU `editor.brush.share.bad` (U+2014, drawn as "?") and passes with a colon there |

`pkg/mapedit/scatter_test.go` checks that one seed gives one forest, two
seeds two, a solid stroke over a full forest records nothing, and every
refused setting writes no byte or history.

Loss control: with the empty-cell test removed from `scatterStroke`,
`TestMapEditorForestPlantsTheShareOnOpenCellsOnly` fails on a standing
object overwritten (see the lane notes).

Release tests (`pkg/game/mapeditforest_release_test.go`, EN and RU):

- `TestReleaseMapEditorForestStrokeSavesAMapTheGameReads`: a dense stroke of
  four installed trees on `scenario/10.alm` plants only on open cells by the
  test's own reading; the map saved by Save As reopens in the editor with the
  same bytes; `alm.Open` gives the same object plane, `mapload.FromALMWith`
  builds a world and each planted cell blocks ground movers; one Undo
  restores the shipped bytes.
- `TestReleaseMapEditorForestWithoutAStrokeSavesUnchanged`: four shipped maps
  set up for Forest (kinds, every density, a typed per cent) and saved
  without a stroke keep their bytes; `TestReleaseInstalledMapsSaveUnchangedWithoutAStroke`
  covers every shipped map.
- `TestReleaseMapEditorForestFrames` writes the owner's frames when
  `AGAINROM_EDITOR_FRAMES` is set.

`pkg/ui/testdata/mapeditor_modes.golden` changes only in the object brush
rows: with the Tool row hidden the old table passes unchanged.

Only the editor and one exported reader (`mapload.StructureCells`, no
behaviour change) change; game frames, World hashes and SAV bytes are
untouched.

## Open debt

- The rule does not fence the party's drop cell or units' cells; the map
  report still flags a drop or a unit the forest closes.
- The grid shows every offered class, rocks and fences included; there is no
  "trees only" filter.
