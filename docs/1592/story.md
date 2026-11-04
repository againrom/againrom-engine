# Traps and cell spells on the editor map

## Intent

Show and edit cell spells where they stand. Lay a rectangle of traps and remove
the stroke together, without extending the ALM format.

## Authority

Owner direction: traps, unit spells and casting structures must be visible and
configurable on the map. The editor is authored; the original editor establishes
its wire output. Rectangle trap strokes are an editor tool, disclosed in
DIV-4484. Untouched and type-9-only writes preserve unrelated bytes (DIV-4485).

Pinned `ALM-EDITT9-328` (High) establishes the three constructors and the per-cell
type-9 wire record. `ALM-SAVE-255` (High / Medium, amended) establishes save-time
loot position and ownership rewriting; this story does not claim a new native
runtime observation. No install was read or written.

## As-built behaviour

`inspectionEnchantment` gives known cell spells a cell, a size and a named
marker. Traps carry their two joined endpoint cells. Unit spells and casting
structures use their bound object's cell and cannot move independently. An
unbound entry remains marked at its stored cell and keeps the existing read-only
field policy. Spatial crosses, joins and endpoint outlines use `markerLines`;
record names use the shared marker-label composer. A cell click uses the existing
selection path and opens the field popup. A trap drag writes `triggerx/triggery`
through `SetFields`, leaving source, target and opaque words unchanged.

The shared category table adds Traps and spells to the bar and tree. Its palette
lists spatial type-9 entries and uses the existing Add window. The constructor
choices come from `cellSpellPalette` in `mapeditorpalettes.go`.

Trap area is available in Tools and that palette. The existing area gesture
draws an inclusive cell rectangle; the Add window asks its spell and power.
Each cell calls `addOn`, whose type-9 add is `Editor.AddSpellFrom`, then the
existing field writer. The rectangle is one Undo step. Source and target start
at each new trap's cell.

`GroupSpellRecords` remembers exact record spans in session-only history.
Splices shift these spans; field edits retain membership; removed members leave
the set. Clone, Squash, Undo and Redo keep the corresponding membership snapshot.
Delete on any member, or Remove trap set in the palette, deletes the surviving
members in descending file order through `deleteObject`, as one history entry.
Identical records outside that stroke are independent. Opening saved bytes
starts with no sets. No set identifier or extra byte is written to ALM.

`Editor.SaveBytes` returns unchanged bytes for an untouched map or when only
the type-9 frame differs from the opened stream. Other edit mixtures retain
the existing `alm.SaveForm` policy. This avoids unrelated unit/loot rewriting
for the edits this story adds.

## Proof

Synthetic ordinary tests, without installs:

- `TestEditorTrapMapSelectionMovementAndFields`: click and popup, trigger drag,
  field-only byte changes in the model and serialized output, Undo.
- `TestEditorSpatialRecordMarkerAndJoinedCells`: unselected cross, two joined
  cells, hit testing and the spell name reaching the common text painter.
- `TestEditorObjectSpellsFollowOwnerWithoutWriting`: owner movement relocates
  each marker without changing its type-9 bytes; independent movement refuses.
- `TestEditorCellSpellPaletteUsesTreeSelection` and
  `TestEditorCellSpellPaletteOffersAllKinds`: three constructors and three
  spatial kinds, no item recipes, existing tree/list selection and Add.
- `TestEditorTrapAreaMatchesSingleAddsAndDeletesOneStroke`: six cells equal
  six single Adds, exact serialized bytes, stroke creation/removal Undo/Redo.
- `TestEditorTrapAreaKeepsIdenticalOutsiderAndEditedMember`: field edits retain
  membership and deletion preserves an identical outsider.
- `TestTrapStrokeMembershipSurvivesEditsIndicesAndHistory` and
  `TestTrapStrokeMembershipDropsRedoBranch`: shifting indices, section splices,
  clone isolation, member deletion, Undo/Redo and abandoned branches.
- `TestEditorTrapAreaRefusalAndUntouchedSaveBytes` and
  `TestEditorTrapAreaCancelledDragAndDialogWriteNothing`: invalid input is
  atomic; previews and cancellation write nothing; untouched output is equal.

The focused model history, menu, key-table, palette and synthetic UI golden
tests pass. `go build ./...` passes after each implementation commit.
Plain-view and side-panel goldens include the added bar entry and tree branch.
Final gate receipts are returned in the lane's RESULT.md.

## Open debt

Release witnesses are not run in a lane. The next release needs both EN and RU:
`TestReleaseEditorNewTrapOnANewMapCastsInTestPlay`,
`TestReleaseEditorPlacedTrapCastsInTestPlay`,
`TestReleaseEditorCellSpellFieldsKeepEveryMap`,
`TestReleaseEditorCellSpellFrames`,
`TestReleaseEditorOffAnchorCastingRecordEditsAndCopies`,
`TestReleaseMapEditorOwnerFrames`,
`TestReleaseMapEditorEachPaletteRecordListsInOnePalette`,
`TestReleaseInstalledMapsKeepTheirBytesThroughTheSaveForm` and
`TestReleaseDeletingAGroundSackKeepsTheOtherSacks`.

The seat owns the full map-editor modes walk at landing. Its pixel hashes must
be reconciled with the added palette and menu entry. No full walk was run here.
Stroke membership intentionally ends when the map closes. Native reload and
test-play effects of the new area route remain unverified in this lane.
