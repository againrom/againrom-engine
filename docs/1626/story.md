# Story 1626: one original-file source boundary

## Intent

One table says where a state that comes from an original file is read from: a town's Player groups
and a mission's application state. A repair to reading or identity has one row to go to, and the
capture route reads through the same document read the first frame adopts. No SAV byte, RNG step,
World hash, frame, hit area, refusal or error text changes on any input.

## Authority

Owner direction on the architecture audit `5d474c65`, kind R1 (original-save LOAD and campaign
adoption), ruled a separate story («Берем те 7 видов, пытаемся достигнуть 13»): one source, read,
adopt and capture boundary; the second game's actor graph and session are the inputs, the first
game's city and Player inputs the reduced form; codecs stay.

## As-built behaviour

- `pkg/game/originalfilesource.go` is the boundary. `originalFileSources()` is the table, five rows,
  one per source: `current SAV document`, `file session and actor graph`, `city provenance` (the
  three town group sources), `decoded document` and `file state store` (the two mission sources).
  Each row's columns are the source's rules: `townHolds`/`townGroups`, `missionHolds`/
  `readApplication`, `frameHolds`/`frame`. A column a source does not state is nil. Rows are tried in
  order, so the town's selection is the first row that holds, unchanged from the previous `if`
  chain; a current file without readable provenance over no file session holds no row and keeps no
  group source, as it did.
- Operations: read `originalSource.readMissionApplication`, the mission LOAD's read moment, called
  once from `prepareMission`, and `readOriginalApplicationState`, the document's read; adopt
  `CampaignSession.adoptTownGroups` (the town arm) and `firstFrameSourceOf` (the first frame's
  source, through the table); capture `applicationCurrentRaw`, the current-to-original projection,
  in the boundary's file. `captureApplicationState` and `projectApplicationState` stay in
  `savapplication.go` and read a document through `readOriginalApplicationState`.
- Moved unchanged into the boundary's file: `readFileApplication`, `readOriginalApplicationState`,
  `originalApplicationStateOf`, `applicationActorIDs`, `applicationCurrentRaw`,
  `fileStoreFirstFrame`, `documentFirstFrame`, `documentApplication`, `firstFrameSourceOf` (now the
  table's walk), `cityGroupsFromLoaded`, `cityGroupsFromGraph`, `cityGroupsFromCurrent`.
  `cityGroupsOver`, `loadedPlayerGroups`, `writeCityGroups`, `reconcileCityGroups` and
  `pruneCityGroupsForSave` stay in `citygroupcarry.go`. `firstFrameSource` and its `firstFrame`
  stay in `firstframe.go`.
- Left apart: `applyOriginalStructuresCurrent` and `takeLoadedPack` (the brief keeps them
  unchanged and in place); `clampOriginalView` reads `/View/*` on the resave document, a patch of the
  output, not a source read; the codecs in `pkg/formats/sav`.

## Proof

- `pkg/game/originalsourcewitness_test.go` and `testdata/originalsourcewitness.golden` (numbers only)
  run over synthetic files of both games, recorded at the base `8a06417be` with the test added in
  the first commit of this story: a town's groups from a loaded first-game city, a current SAV, a
  second-game actor graph and a second-game engine SAV, the empty case and the file holding no
  source (session level: a current SAV's decoded source with an unreadable provenance); the mission
  read moment with a state store present, absent (a file with no world half) and refused (a pressed
  index outside the book, a missing `/View` leaf); the first frame and three captures from a state
  store and from a decoded document, the second game's engine SAV among them; the projection back
  into a document for a current, an absent and a local-only application state with its refusals; and
  the hash of each resaved SAV. The record is equal at the candidate. A planted fault fails it: the
  current row ignoring a provenance refusal, the store row's identity, the capture's pressed slot.
  Not detected: narrowing the store row's identity class from 1 to 2, because the fixture's hero
  binds class 2 or higher.
- `TestTownSourceTableChoosesOneRowPerFile` pins each row's selection and that a test never stands
  without its reader. `TestFirstFramesRestoreThroughOneRestore` keeps the first-frame owners.
- `internal/archtest` `CheckOriginalFileSources` (reference check through the type checker, so a
  function value and an import alias count): the read moment and each group source have one
  production site, the table; the document read has its four named callers.
  `GroupWordsSeams` names the boundary's file in place of `citygroupcarry.go`, because
  `cityGroupsFromGraph` moved. `worldcarriers.go` names the table in place of `prepareMission` for
  the document nil test.
- Census at `pkg` and `cmd`, non-test references (base to candidate), with sites outside the
  boundary: `readFileApplication` 1 outside to 0; `cityGroupsFromCurrent`, `cityGroupsFromGraph`,
  `cityGroupsFromLoaded` 1 outside each to 0; `loadedPlayerGroups` 2 outside to 0;
  `originalApplicationStateOf` 2 outside to 0; `documentFirstFrame`, `fileStoreFirstFrame`,
  `documentApplication` to the table and its rows; `firstFrameSourceOf` 1 unchanged, inside the
  restore. `readOriginalApplicationState` 4 production uses, 3 outside the boundary, named by the
  guard (capture and projection reading the document row's reader); `applicationActorIDs` 2, 1
  outside (the projection); `applicationCurrentRaw` 3, all outside (the three producers of a
  current document's leaves).

## Open debt

- The witness's identity row holds class-2 and higher bindings only (the fixture's actors). A
  store mission with a class-1 binding is covered by the release tests of second-game missions, not
  by this witness.
- The three document-side readers stay callers of `readOriginalApplicationState` rather than a
  table column: their read is the document row's own reader, and a named op would be the same
  function under a second name.
