# Map editor terrain commands: random terrain and heights, terrain patches, starting height

## Intent

The map editor gains the whole-map terrain commands the original editor has: random terrain
(Ctrl+E) and random heights (Ctrl+Shift+R), each from a typed seed and each one undo step. A
selection box copies the tiles, heights and objects (trees and other cell objects) under it with
the records it selects, as one clip, and Ctrl+V pastes them at the pointer's cell, cut at the
map's edge, as one undo step. A new map starts at a chosen height, 0 to 126, 64 by default. A
View toggle (V) writes each height value on the map as the stored height minus 64. The game, its
frames, World hashes and SAV bytes do not change, and every editor operation that existed before
writes the same ALM bytes.

## Authority

- Owner direction: reproduce the mechanisms the original editor's help pages describe, in this
  editor's own layout (the brief of this story); the height numbers are an owner request added to
  it. The help pages are not evidence of game behaviour and no text of them is in the repository.
- `TERR-248` (active): the original's whole-map command visits x outer, y inner and redraws each
  cell's tile word among its look-alike row with the CRT `rand` (seed 1), keeping bit `0x2000`.
- `TERR-246` (promoted): a new map holds one tile word and one height in every cell, the height
  from a slider 0..126, 64 by default; its relief option adds `rand()/0xaaa - 6`.
- `TERR-GEOM-031`: the mesh vertex `(c, r)` takes height byte `c + W*r`, so a stored height
  belongs to the top-left corner of cell `(c, r)`.
- `ALM-EDITDISP-279`, `ALM-EDITPASTE-300` (active): the original copies a cell rectangle and its
  paste writes three cell layers (tile words, heights, object bytes) under three menu switches;
  no record is pasted.
- `ALM-SAVE-238`: rows 0 to 7 of the tile plane take the save's normalisation.
- Divergences in `docs/divergences/ui-map-editor-terrain.md`: `DIV-4013` (seeded source, the corner
  rule's look-alike table, the authored random-heights rule) and `DIV-4014` (one clip of all three
  planes with the records, no per-plane switches, object bytes replaced). The starting height
  matches `TERR-246` and takes no row.

## As built

### One writer per plane

`pkg/mapedit/brush.go` keeps the brush's plane writers as the only writers: `tileEdit` (tile
words, save normalisation of the top eight rows, group mask through `alm.TileGroupMask`),
`wordRuns`, `byteRuns` and `(Brush).heights` (cosine fall-off, clamp, row step of the height
brush). Every existing stroke goes through them; its bytes are unchanged.

### Random terrain and random heights

`pkg/mapedit/wholemap.go`. `RandomizeTerrain` visits x outer and y inner and draws one word per
cell among the words that show the same corner state in the same strip group, keeping bit 13 and
the other high bits, so no terrain or transition moves; then `tileEdit`. `RandomizeHeights` lays
`max(1, w*h/64)` height strokes through `(Brush).heights`: per stroke it draws x, y, a radius
3..8, a change 8..32 and its sign. Both take the caller's source; the window's host
(`pkg/game/mapinspectionterrain.go`) builds it from the typed seed with `random.NewGo`, the same
source the brush's random fills use, and runs each command as one `change`. The window
(`pkg/ui/mapeditorterrain.go`) asks for the seed (0 to 4294967295, the last seed plus one
offered), refuses another text on the message line, and names the seed and the rule on the
status line. Tools menu rows and the F1 key reference carry both commands.

### Terrain patch

`pkg/mapedit/patch.go`. `CopyPatch(x, y, w, h)` takes the three planes of a rectangle;
`PastePatch(p, at)` writes them at `at`, cut at the map's edge, as one edit. A plain drag on the
map that moved keeps its cell box (`selectionState.box`); Ctrl+C then calls the host's
`CopyArea`, which puts the patch and the box's records into the one clip. Paste runs the patch
first and the records' paste after it inside one `Mark`/`Squash`, so one undo step restores both.
A map with no object plane refuses a patch with objects.

### Starting height

The new-map question has an edit row, prefilled 64. The source `new/<W>x<H>` gains
`/altitude<n>` for any other height (before `/multiplayer`); `newMapSpec` reads it and refuses a
value outside 0..126 or a malformed one. `mapedit -new 32x32/altitude40` takes the same source.

### Height numbers

`pkg/ui/mapeditorheights.go`. V (View: Height numbers) toggles them, off by default. Each label
is the stored height minus `HeightLabelZero` (64, the new-map default), drawn in the editor's font
just inside the cell's top-left corner, the vertex the height belongs to. A cell narrower than
the widest label (`-64`) plus margins or lower than the font shows none; a label that would cover
a popup, an area label, a group badge or an earlier label is skipped. Labels are read from the
document each frame, so they follow every height edit at once. With the toggle off the chrome
draws nothing new.

### Words

EN and RU in `pkg/words/text/{en,ru}/strings.toml`: `editor.newmap.height`,
`editor.newmap.badheight`, `editor.tools.random*`, `editor.prompt.random*`, `editor.random.*`,
`editor.keys.randomterrain`, `editor.keys.randomheights`, `editor.keys.heightlabels`, `editor.view.heights`,
`editor.selection.area`, `editor.selection.terrain`.

## Proof

- `pkg/mapedit`: `wholemap_test.go` and `patch_test.go` (one seed one map, another seed another,
  planes kept, one undo step, the patch cut at the edge, objects pasted).
- `pkg/game`: `TestNewMapHeightWordsAreTheFormats`, `TestNewMapSourceCarriesTheStartingHeight`,
  `TestHostWholeMapCommandsFollowTheSeed`, `TestHostAreaCopyPastesTheTerrain`.
- `pkg/ui`: `TestRandomCommandsTakeTheTypedSeed`, `TestBoxCopyTakesTheTerrainUnderTheBox`,
  `TestHeightNumbersShowTheVertexHeightMinus64`, `TestNewMapQuestionTakesTheStartingHeight`, and
  the regenerated `testdata/mapeditor_modes.golden` (new help lines, the new-map question's row).
- Release, EN and RU (`pkg/game/mapeditwholemap_release_test.go`): the commands on
  `scenario/10.alm` through the window's keys, the patch with trees, the new-map heights 64, 0
  and 126 and the refusal of 127, the height numbers on, after an edit and off; each changed map
  is built by `mapload.FromALMWith` and plays 40 ticks.
- Unchanged: every existing editor test passes with its asserted values, including the brush,
  fill and forest release tests on shipped maps.
- Loss controls (a seed the source ignores, a patch that drops objects, a label offset of 0)
  make the named tests fail; the results are in the story's return.

## Item 1 census

The type-9 trap and casting-structure records stay carried byte-equal and are not edited here.
The census of their fields, the claims that decode each and the questions for the fields no claim
decodes are in the story's return to the seat.

## Open debt

- The original's 130-row look-alike table and its weights are not decoded into the repository;
  random terrain draws from the corner rule's table (`DIV-4013`).
- No claim reads the original's whole-map altitude command; the random-heights rule is ours
  (`DIV-4013`).
- Per-plane paste switches and adding objects over the old ones are an owner decision
  (`DIV-4014`).
