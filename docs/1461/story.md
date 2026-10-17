# Trigger areas edited on the map

## Intent

The map editor draws each trigger's areas on the map. An author now changes
them there: dragging a handle of an area of the selected trigger rewrites the
cell, box or radius its condition or action names, and drawing a box on the
map creates a new box condition for a trigger the author picks. The story
started at engine main `62118bd2`; the pin does not move.

## Authority

Owner direction: an area's edges and corners resize the cell, box or radius
its node names; a drawn box becomes a new area condition (check 2) and asks
which trigger takes it, a new trigger first and the default; the record drag
is the one drag builder; the trigger choice uses the existing popup and
drop-down; every word is in `pkg/words` in EN and RU; each edit is one Undo
step; a node more than one trigger names, or a constant several pairs compare
with, gives a notice the author can cancel; values out of range are clamped
or refused; untouched maps keep their bytes.

Claims, read from the pinned snapshot:

- `TRIG-CHECKGEOM-130` (High): check 2 holds when `p0 <= x <= p2` and
  `p1 <= y <= p3`, an inclusive box; reversed bounds never hold. Check 3
  compares the Chebyshev distance with its own `p2`; check 7 stores the
  distance for a pair to compare.
- `TRIG-DIST-014` (High / Medium): every check distance is Chebyshev; the
  editor's "circle" is a square of side 2r+1.
- `TRIG-COND-003`, `TRIG-CMP-006`: the check opcodes; the six comparison
  codes `==` `!=` `>` `<` `>=` `<=`, the three pairs ANDed.
- `ALM-TRIG-046` (Medium): parameter type codes, X and Y plain tile
  coordinates.
- `TRIG-BIND-010`: what the binder keeps and drops.
- `TRIG-EDITVAL-119` (High / Medium): the original Trigger Manager stores
  node ids, one of six comparison codes per row and the once flag; it says
  nothing of drawing an area on the map.
- `TRIG-HEROORD-075`: a `Target_Unit` of 10001 is hero ordinal 0, the first
  player's primary character.

No claim describes how the original editor edits an area on the map; the
handles, the drawn pair's form and its defaults are this project's
(`DIV-3605`).

## As built

`mapload.ScriptAreas` now names, for each area, the parameter slots its
cells come from (`Slots`) and the condition and slot that hold a square's
radius with its bias (`RadiusNode`, `RadiusSlot`, `RadiusBias`). The editor
carries them in `ui.InspectionArea` as document records.

Handles (`pkg/ui/mapeditorareaedit.go`, drawn by `areaLines` in
`mapeditorareas.go`) show on the areas of the selected trigger. A press near
one starts the record drag (`editorRecordDrag.area`); the preview follows the
pointer and the release writes one record through `InspectionFieldSet` as one
Undo step. Esc during the drag, or a release without movement, writes
nothing.

| Handle | Shown on | Writes | Bounds |
|---|---|---|---|
| Box corner | the four corners of a check-2 box | the node's X and Y slots of that corner (X, Y or X2, Y2) | each cell clamped to 0..min(W,256)-1 and 0..min(H,256)-1; a corner stops at the opposite corner, so X <= X2 and Y <= Y2 |
| Box edge | the middle of each side | the one slot of that side | as the corner |
| Radius | the middle of each side of a square | check 3: its own radius slot; checks 7, 15, 16: slot 0 of the constant on the other side of the pair, written as radius plus bias (1 for `<` and `>=` with the check on the left, mirrored when the constant is on the left, else 0) | the Chebyshev distance from the centre to the pointer's cell, 0..255; the centre does not move |
| Cell | a cell a node names | the node's two X and Y slots | clamped as the box |

A square around a unit (check 6) and any area whose parameters are not plain
X/Y slots show no handle.

Before a write, the editor counts the references to the record. A node
that more than one trigger names gives the notice "Triggers T1, T3 use this
node." A radius constant that more than one pair compares with gives "The
radius comes from constant N, which pairs in triggers ... compare with (K
pairs)." Change all writes; Cancel and Esc write nothing.

L, or View > Draw trigger area (L), arms the draw tool (status line: "Drag a
box on the map for a new trigger area; Esc cancels."). The drag draws a box
clamped to the map; the release opens "New trigger area" naming the box,
with a Trigger drop-down (New trigger first and selected, then each trigger
as `T<n> <name>`), Create and Cancel. Create calls
`InspectionAreaAuthor.AddBoxArea` (`pkg/game/mapinspectionareas.go`) as one
edit:

- a check-2 node with the next condition id: unit 10001 (type 4), X, Y, X2,
  Y2 (types 5, 6, 5, 6) the box's inclusive corners;
- the pair `box == constant 1`, the box on the left, as 9 of the 10 shipped
  box pairs (EN); a constant 1 that names one node and that no set-variable
  action (3, 8) or variable check (19) addresses is reused, otherwise a new
  constant (opcode `0x10002`, int type, value 1) takes the next id;
- a new trigger ("Trigger", once) when New trigger is chosen, else the
  chosen trigger's first empty pair. A trigger with three pairs is refused
  in the prompt ("Trigger N already has three condition pairs.").
- The script's 100-register and 5000-id limits and a register a variable
  already uses refuse the edit; the prompt shows the reason and stays open.

The new trigger is then selected and shows its area and handles.

## Proof

Unit witnesses (`pkg/game/mapeditorareaedit_test.go`, a synthetic 40x40 map
with three triggers):

| Test | What it proves | Loss control |
|---|---|---|
| `TestAreaHandlesRewriteTheirParameters/box_corner`, `/box_edge`, `/box_edge_clamped`, `/radius`, `/cell` | each handle writes the bytes `SetFields` writes for the same values on a twin editor; one Undo restores the map byte-equal; Redo writes it again | per handle, below |
| `TestAreaDragCancelledWritesNothing` | Esc during a drag, a click on a handle, showing areas, selecting, and the draw tool cancelled with Esc write nothing | - |
| `TestAreaSharedNodeAsksBeforeTheChange` | a shared radius constant (T2, T3) and a shared action node (T1, T3) ask first; Cancel writes nothing; Change all writes the `SetFields` bytes; one Undo restores | - |
| `TestAreaDrawnBoxGoesToTheChosenTrigger` | Create with New trigger writes the bytes of the node, the pair and the trigger placed directly through `mapedit`; Undo restores; the drop-down attaches a box to T2's empty pair; a full trigger is refused; Cancel writes nothing | - |
| `TestAreaDrawnBoxAddsItsConstantWhenNoneIsFree` | with no reusable constant 1 the pair compares with a new constant after the box | - |
| `TestScriptAreaNamesTheSlotsItComesFrom` (`pkg/mapload`) | the slots, radius node, slot and bias of each shape | - |

Loss controls, each run on the committed tree with one arm removed and then
restored; the results are in the story's notes directory:

| Arm removed | Failing test |
|---|---|
| corner: the left/right X update in `areaDragTo` | `TestAreaHandlesRewriteTheirParameters/box_corner` |
| edge: the clamp of the left side against X2 | `TestAreaHandlesRewriteTheirParameters/box_edge_clamped` |
| radius: the bias added on release | `TestAreaHandlesRewriteTheirParameters/radius` |
| cell: the cell update in `areaDragTo` | `TestAreaHandlesRewriteTheirParameters/cell` |

Release witnesses (`pkg/game/mapeditorareaedit_release_test.go`), EN and RU:

- `TestReleaseMapEditorAreaDragsSaveAndReopen`: on `scenario/100.alm`
  trigger T8, the box (26,123)-(27,124) dragged by its far corner to
  (28,125); on `scenario/10.alm` trigger T3, the square at (36,51) radius 3
  dragged to radius 4 through its shared constant (notice, Change all). Each
  map is saved, reopened with `alm.Open`, and `mapload.ScriptAreas` shows the
  moved area; `mapload.CompileScript` builds the script and its check holds
  the new value. A box drawn on `scenario/100.alm` and created as a new
  trigger is read back as that trigger's box and compiled.
- `TestReleaseMapEditorAreasUntouchedMapsSaveTheirBytes`: every installed map
  (EN 38, RU 34) opened, its areas shown (EN 394, RU 393), its first trigger
  selected, drawn, and saved with Save As writes the installed bytes.
- `TestReleaseMapEditorAreaFrames`: the handles, a drag in progress, the
  armed draw tool, the trigger choice, its open drop-down and the shared
  notice, with every glyph kept outside an open prompt or list.

## Open debt

- Moving a whole box or a square's centre by dragging its inside is not
  built; the handles only resize, and a cell handle moves the cell.
- A drawn box always names hero ordinal 0 (10001). On a multiplayer map every
  ordinal resolves to 0, so the condition reads the first player's hero; the
  unit is edited in the node's fields afterwards.
- The shared notice counts trigger references; a radius constant that a
  set-variable action also writes is not named in the notice.
