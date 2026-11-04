# Bridge brush

## Intent

Paint and erase wooden bridge cells in the map editor's brush window. One
pointer stroke is one edit and one undo step. Keep the existing structure
record codec and the drawing shared with the game.

## Authority

The owner requested the Bridge tool, approved the existing wooden bridge
picture and chose a minimum of 3 by 2 cells, including erase remnants.
`TERR-STRUCT-105` at main's knowledge k420 establishes the class-33 rectangular
nine-patch representation. It establishes neither the original editor's tool
nor its minimum size. `DIV-4510` records the authored tool and cover rule;
`DIV-4511` records the owner-directed floor without asserting an original
threshold. Both reserved divergence IDs are used. The knowledge pin is unchanged.

## As-built behaviour

Tools / Bridge and the Bridge toolbar button open the existing brush window.
Its title is Bridge in English and Мост in Russian. Left drag paints; right
drag erases. Brackets and the size buttons change the existing radius. The
stamp is rectangular, with width max(2r+1, 3) and height max(2r+1, 2). At the
map edge its origin shifts inward to retain that size. The window names the
minimum and both mouse gestures through the engine words tables.

The stroke gathers only bridge records its own footprint meets or touches,
including corner contact. A record touching a met record is not thereby met.
Its cell set is the met records' union with the stroke cells added or removed.
Unmet bridge rectangles are fenced in the model. The host supplies other
structure kinds' attached cells through mapload.StructureCells and the brush's
Fenced data; it uses the existing footprint resolver. No stored rectangle
takes a fenced cell. The mission's structure pass is unchanged.

`rectangleCover` in `pkg/mapedit/rectanglebrush.go` owns the disjoint width-first
patch cover. Scan 3-by-2 patches top to bottom, left to right. A full unassigned
patch grows right, then down, through unassigned cells. Each extent is at most
255; where a terminal strip is below the minimum, shorten the preceding
rectangle to leave a minimum-size strip when possible. Mark the rectangle's
cells assigned before continuing. No rectangle overlaps another or fills a
hole. Cells no remaining full unassigned patch can hold are dropped. This is
a deterministic greedy rule, not a maximum-area or minimum-count solver.

The smallest nonrectangular union of full 3-by-2 patches has rows `###.`,
`####`, `.###`. The first rectangle keeps six cells and drops the other four,
although every input cell belongs to a full patch. A disjoint cover cannot
keep all ten: one rectangle cannot describe the shape and two need at least
twelve cells. Greedy allocation can also lose cells a different partition
would keep: rows `######`, `######`, `...###` retain the top twelve and drop
the last three; a left 3-by-2 and right 3-by-3 partition keeps all fifteen.
DIV-4511 discloses both losses beyond the owner's size floor.

For radius 0, centres (10,10) through (14,10) leave one rectangle
[9,16) by [10,12). An existing 9-by-3 rectangle [5,14) by [10,13), erased at
(9,10) and (9,11), leaves [5,8) by [10,13) and [11,14) by [10,13).
The L witness leaves [9,16) by [10,12) and [9,12) by [12,16).
Join-down leaves [5,14) by [10,13) and [11,14) by [13,19).

Exact rectangles retain their records first. Other overlapping rectangles
reuse records in file order. Growth and shrinkage preserve identity, owner,
placement, fractional position and reserved bytes; only the anchor and size
fields that changed are written. Unmet records retain every byte. Removal
uses the existing deleteObject / Delete rule, including attached cell spells.
New records use owner slot 1 and NewStructureRecord, Place, then SetFields.
The existing Clone / Squash and Paint applier make the result atomic and give it one undo
step. Delete retains its existing file-order compaction: removing an earlier
record shifts a later surviving record's index. An isolated resize keeps its
index. Opening or saving without the tool performs no bridge cleanup.

| Kind | One builder |
|---|---|
| Brush mode and window | brushMode and the existing brush window in pkg/ui/mapeditorbrush.go; existing toolbar, menu and key tables carry the new entry |
| Stroke, footprint and undo | BrushStroke, Paint and BrushCells; mapedit.Brush.walk receives RectangleBrush data; the existing history applier and Squash own undo |
| Structure record | NewStructureRecord, Place, Move, SetField / SetFields and Delete; the host's deleteObject retains its reference rule |
| Rectangle cover | rectangleCover in pkg/mapedit/rectanglebrush.go, called only by rectangleStroke through Paint |
| Editor and game drawing | terrain.StructurePlacements, unchanged |
| Editor words | pkg/words/text/en/strings.toml and ru/strings.toml through e.word |

## Proof

Base: `068d8e618205283a5849ebc2381848e6133188e8`. The failing-test commit is
`ecea3d97886d75beb5d00dfabd01082c64cd03a9`. Each of its eight ordinary tests
failed before implementation: six reported no brush plane 4, the edge
footprint was one cell, and the toolbar lacked Bridge.

The focused synthetic witnesses in pkg/game/mapinspectionbridge_test.go are:

- TestBridgeStrokeAcrossOpenGround
- TestBridgeEraseSplitsAndDropsSmallPieces
- TestBridgeStrokeMergesTouchingRecords
- TestBridgeStrokeStoresAnLAsTwoRectangles
- TestBridgeStrokePreservesRecordBytesAndUntouchedSave
- TestBridgeStrokeIsOneUndoStep
- TestBridgeBrushCellsMatchMinimumAtMapEdge
- TestBridgeToolPointerPaintsErasesAndRefreshesView

They pass after implementation. They check the rectangle geometry, a dropped
2-cell-wide remnant, untouched and unrelated-brush save bytes, preserved
record identity and reserved bytes, byte-exact undo/redo, the actual left and
right pointer route and one drawn bridge through both the editor viewer and
LoadMapViewer. Synthetic art does not establish installed rendering.
TestRectangleBrushHasOneCoverAndRecordRoute guards the policy, cover caller
and Paint route with a static source scan. It also rejects the existing
structure-write APIs outside rectangleStroke in brush functions and recognizes
unqualified policy literals. It does not prove arbitrary alias or raw-byte
dataflow. Final baseline, fast and architecture receipts belong to the lane's
CORRECTION-RESULT.md.

The correction tests were committed before the production correction, starting
at d1fa065fa8cc92d10bfaea44e038ba3b175dc86e. TestBridgeStrokeKeepsWalkableCells
failed for L, two-stroke L, T, join-down and the touching pair; its straight,
corner and erase cases were passing controls. TestBridgeStrokeLeavesIndirectRecordBytes
failed on identity 8's changed position and height. TestBridgeStrokeAvoidsOtherStructures
failed with either structure order. TestRectangleCoverDropsUnassignablePatches
failed on the staggered patches. TestRectangleCoverKeepsLongBandsDisjoint failed
with the base cover overlaid from its Git blob. The corrected host witnesses
check zero Refused and Abandoned, open stored water cells, disjoint rectangles,
exact retained cells, no structure-overlap report and byte-exact Undo after
each stroke. The pointer witness now paints on free ground in its existing
fixture, away from the non-bridge structure whose cells are fenced.

The fast run exposed the old toolbar-size threshold and panel golden, plus
a shortcut label on a tool that has no keyboard binding. The threshold is
now 782 pixels, and the nine synthetic panel states record the additional
Bridge button. Focused toolbar, panel, key, menu and brush tests verify the
correction. The two unit-name tests refused by the sandbox are not retried.

## Open debt

The requested Panic! paint/erase pictures were not produced. The temporary
headless instrument was to read only scenario/81.alm and its picture data.
scripts/build-tools.sh stopped at mkdir with Permission denied before
compiling it. Lane rules prohibit a workaround. The instrument source and
failure receipt are in the owner-artifact directory, outside the repository;
its temporary repository files were removed.

The brief's universal index-preservation wording is limited by the required
Delete compaction described above. Script targets retain their identities;
a target of a removed record follows the existing deletion/report rule.
Original editor gestures and small-piece handling remain Unknown. Adjacent
bridge records each draw their own nine-piece frame, including the edges at
their joint, and each takes its own structureRectLift. Uneven ground can give
parts of one painted shape different lifts. The cover's additional losses are
described above; no globally optimal count or retained area is claimed.

No release test, full editor modes walk or installed witness ran in this lane.
The seat must walk MAPEDITOR_MODES_FULL=1 at landing and assess the recorded
table after the toolbar/menu addition. The next release must run EN and RU
map editor witnesses for the brush window, Structures palette and saved
structure records. RESULT.md names their tests. Versions and resources are
left for the landing. Main is already reconciled. The sole review returned
the story; this correction receives no second review. Push and landing belong
to the seat. Synthetic L and join-down placement pictures use the game's
StructurePlacements path and establish neither installed art nor unit routing.
The first return's fast and architecture receipts do not establish correction
acceptance. CORRECTION-RESULT.md records the correction candidate's gates and
any sandbox denials the seat must rerun outside the sandbox.
