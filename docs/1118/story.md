# Editor unit movement and Save As

The editor now completes the owner-requested loop: move a placed unit, Undo,
Redo, Save As, and reopen the new ALM in a fresh editor.

## Behaviour

- Dragging a unit previews its destination marker, then commits one cell move
  on release. Original file-order identity selects the record, independent of
  catalogue filtering. Ground dragging still pans. Escape, a rail crossing or
  leaving the canvas/map cancels; a click or same-cell move adds no history.
- The retained byte model owns history. Movement preserves both coordinate
  fractions and all bytes outside the selected record's first eight bytes.
  Undo/Redo rebuild the inspection projection: units, raw details, section
  previews, warnings and unit/group trigger targets remain consistent. Camera,
  filters, selection and the selected detail viewport survive an edit.
- The existing rail carries Undo, Redo, Save As and Saved/Modified state.
  Ctrl+Z, Ctrl+Y or Ctrl+Shift+Z navigate history; Ctrl+S opens Save As.
  The path field starts empty and Enter writes only an explicitly named new
  `.alm` file. Long paths retain their editable tail and caret; errors occupy
  separate bounded rows. Escape cancels the prompt.
- Save As writes model bytes, not a decoded-map re-encoding. It rejects an
  existing destination, source alias, either preserved EN/RU install, and
  archive-identified installations. Physical path and identity checks cover
  case aliases, dot segments, symlinks and Windows junctions. Device names and
  alternate-stream targets are refused. No directories
  are created. Failure preserves document, checkpoint and history.
- A successful Save As updates the source/checkpoint without erasing history.
  Undo away from saved bytes becomes Modified; Redo back becomes Saved. Opening
  another map or closing a dirty editor is refused until Save As or Undo has
  resolved the changes. Malformed opens never replace the current document.

## Authority and scope

`ALM-UNIT-040` establishes the 70-byte unit record and coordinate DWORDs;
`ALM-UNIT-048` distinguishes unit and group identity. `ALM-REQ-055` establishes
physical record framing and last-wins duplicates. `ALM-ORD-057` does not license
reordering: this feature preserves the complete physical stream instead.
These claims establish data, not original-editor gestures or presentation.
The interface remains owner-directed under `DIV-618`; trigger inspection's
separate `DIV-626` remains open. No new divergence number is needed.

Only unit anchor movement is editable. This is not simulation, pathfinding,
collision validation, unit placement/deletion, trigger editing, structure or
sack movement, terrain editing, overwrite-save or original-editor fidelity.
There is no discard-changes action in this slice. External concurrent filesystem
mutation is not a supported writer transaction; exclusive creation prevents
replacement, but this is not a hostile-filesystem race defence.

## Proof

`mapinspection1118_test.go` independently walks ALM headers and compares every
byte, including overridden type-6 sections, unknown records and trailers.
It drives pointer movement, keyboard history and the explicit path prompt into
host persistence, then opens a fresh editor. UI probes cover original identity,
preview, retained camera/filter/selection, markers, ground pan, disabled buttons
and non-overlapping long-path/error layout at all supported minimum sizes.
Writer tests exercise real temporary symlinks and Windows junctions, source hard
links, case/dot aliases and both synthetic preserved language roots.

`TestReleaseMapEditor1118MoveUndoRedoSaveAsReopen` drives the same editor on EN
and RU archive and loose maps, checks every ALM byte independently, checks raw
unit-ID trigger references, and compares the actual composed canvas across
Undo and fresh reopen. This is CPU terrain plus retained bodies and the actual
rail, not GPU readback; projected shadows are omitted. No native window or
original game is driven. Exact candidate results are in `verification.md`.
