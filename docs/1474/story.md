# Map editor terrain tools and passability view

## Intent and authority

The editor's terrain, height and objects brushes were reachable only by the
keys T, H and B, and passability had neither a view nor a tool. Owner
direction: a View mode that washes passable cells semi-transparent white and
blocked cells semi-transparent red, as a complement to the grid. The editor's
layout is the owner's; the original editor is evidence of what map bytes mean,
not of its layout. No DIV row: no game-visible behaviour changes.

Claims read through the pinned claim tool: ALM-GRID-012 (tile word: bits 0-9
the index, bit 13 the impassable flag, one of three tile-word arms that block a
cell) and ALM-TERR-016 (the block plane is a bitmask derived from type 1 and
type 3, with the eight-cell border; a ground mover tests bit 0).

## As built

- Top bar: a "Brushes" menu (RU "Кисти") after View. Rows Terrain, Height,
  Objects, Passability and No brush, the current one checked. A row turns its
  brush on and leaves a brush already on as it is; No brush turns the brush
  off. The entry is disabled with no map or a read-only document. The keys T,
  H and B keep switching their brush; Q switches the passability brush (Q was
  free: `pkg/ui/mapeditorkeys.go` and the key map in `MapEditor.Update` bind C,
  Del, E, S, D, G, T, H, B, [, ], N, U, J, K, I, P, A, F1, F2, F5, M, X, W, L,
  O, F and R).
- Terrain list: each row of `mapedit.Palette` shows a swatch beside its word:
  the cell the row's base word resolves to (`terrain.Resolve`) in the
  install's tileset, copied at runtime by `MapInspector.terrainSwatches` and
  kept for the session. The list is the one picker (`Picker`, drawn by
  `drawListBox`); the picture is a new data field of its row, `PickerRow.Icon`,
  which the list box draws square at the row's height before the text. No
  second list widget exists.
- View menu: "Passability" (RU "Проходимость") below "Grid", off by default.
  On, every cell the terrain pass draws gets a fill, white at alpha 0x40 for an
  open cell and red at alpha 0x60 for a closed one, drawn after terrain,
  objects and structures and before every editor line, so the grid lies over
  it. The passability brush shows the same fills while it is on.
- Red set: `InspectionDocument.Passability`, filled by the host from
  `mapload.GroundArms(m, table)`. That function reads the plane and the
  footprint occupancy of the one derivation `PassabilityWith` runs
  (`deriveTaken`), which is the plane `mapload.Planes` hands the world
  constructor. A cell is red exactly where that plane's ground bit is set. The
  host rebuilds the document after every edit, undo and redo, so the fills
  follow every stroke, record edit and structure move.
- Status line, with the view or the passability brush on: "Passable", or
  "Blocked:" and the arms (border, Mountain, Water, flag, object, structure).
  A cell a structure footprint attached names the structure alone, since the
  footprint overrides the terrain arms there.
- The status strip keeps its line within 1280 px: when the whole line is
  wider than the strip, the keys help drops its leading lines and keeps the
  last, "F1: all keys / F2: scripts". Q is named on its own line of the F1
  key list ("Q: passability brush"), not on the status line.
- Passability brush: rows "Block" and "Unblock" (RU "Закрыть", "Открыть").
  A stroke is one `mapedit.PlaneImpassable` brush: it sets or clears bit 13 of
  the tile word of each disc cell and writes no other bit, cell or record;
  the tile-group mask reads bits 0-9 only. One stroke is one undo step.
  Unblock on Water, Mountain, an object, a structure or the border leaves the
  cell red, and the status line names the arm that keeps it.
- Height: with the height brush on, the status line shows "Height N", the
  altitude byte of the cell under the pointer. The editor already shows
  height: the document viewer is built displaced (cells lifted by the altitude
  plane) and lit (corner-interpolated shading from the altitudes). The
  viewer's `flat` and `unshaded` fields are never set by the editor; only
  tests set `flat`. `adoptTerrainEdit` carries both across a terrain edit.

Unchanged: game screens, World hashes and SAV digests (no `pkg/sim`, save or
game-screen code changes; `deriveTaken` returns the plane `derive` returned);
the bytes an existing terrain, height or object stroke writes; the bytes of a
map opened and saved with no edit.

## Proof

- `pkg/mapload/groundarms_test.go`: a 40x40 fixture with every arm and a
  structure that closes two cells and opens two. Arms are nonzero exactly
  where `PassabilityWith` sets bit 0; control: with no table the footprint
  cells read their terrain arms again.
- `pkg/mapedit/impassable_test.go`: Block and Unblock change only bit 13 of the
  disc cells (every changed byte differs by 0x20 in a tile word's high byte);
  control: a flagged cell outside the disc keeps its flag; two undos restore
  the two strokes; a value other than 0 and 1 is refused.
- `pkg/ui/mapeditorpassability_test.go`: the Brushes menu enters each mode and
  turns the brush off (control: a disabled entry opens nothing); T, H, B and Q
  switch; the passability rows send 1 and 0; the view is off by default, fills
  red and white, names arms, and draws nothing when off; the height readout
  shows only with the height brush; the terrain rows carry swatches and the
  list box draws them (control: a row with no swatch draws none).
- Release, EN and RU installs: `TestReleaseMapEditorPassabilityIsTheWorldsGroundPlane`
  checks every installed map (EN 38, RU 34) against `PassabilityWith` and two
  maps (`scenario/10.alm` and the first other map whose structures move the
  plane) against the world `mapload.FromALMWith` builds; control: the plane
  without the table disagrees with that world.
  `TestReleaseMapEditorPassabilityFollowsEveryEdit` drives the window on
  `scenario/10.alm`: a Water stroke, undo, redo, an object stroke, Block
  (one byte changed, by 0x20), Unblock back to the original bytes, Unblock on
  Water (still red, status names Water) and a structure drag with its undo.
  `TestReleaseMapEditorPassabilityFrames` writes the owner frames.
- `TestReleaseMapEditorStatusFitsWithPassability`, EN and RU at 1280x800:
  the whole status line fits with the pointer on a cell in the keys help and
  each brush's help, view off and on; with the view naming five arms the
  keys help keeps its F1 line. Control: that whole line is wider than the
  strip in the same font.
- `pkg/ui/testdata/mapeditor_modes.golden` is regenerated: the bar gained an
  entry and the F1 key list a line for Q, so every recorded chrome hash
  moved. The "ui:menu-fit" gesture finds View > Fit by key; its cell equals
  the "fit" key's cell in 87 of 103 rows (base 86 of 101). Against the base
  golden every other column is a relabelling or a split (the new Brushes
  entry tells an editable document from a read-only one); no two states the
  base told apart merge.
- Gates on the merge `79fb69685` with main `d66eb78e` (Ebitengine v2.10.5,
  the spell-weapon reach hotfix, pin k334): the release set passed on EN and
  RU, 1314 gated tests each, with 0 failures and one skip,
  `TestReleaseActivePauseRenderedWorld`, under `AGAINROM_HEADLESS_ONLY`;
  mission scenarios 76 of 76; `go test -trimpath -count=1 ./...` 82 packages
  ok; storyguard and archtest pass; `check-no-game-assets.sh` clean. The six
  owner frames per install re-render byte-identical.

## Open debt

- The terrain names in the list come from `mapedit.Palette` in English on both
  installs, as before; the height rows "Raise"/"Lower" and the help lines are
  English too.
- The fills cover cells the terrain pass draws; at the camera's smallest zoom
  they cover the whole view and the grid is not drawn (`GridMinCellPixels`).
