# The map editor writes what the original's editor writes

## Intent

A map the editor saves is a map the original's editor would write from the same
content. The story compares each statement the claims make about the original
editor's reader and writer with what `pkg/mapedit` and `pkg/formats/alm` write,
and either matches it or records one divergence row with the reason. The story
started at public main `fc95e10b` (pin k289) and moved the pin to k296.

## Authority

Owner direction (the first game's map editor is built completely; the original's
editor is evidence of what it writes). Claims read from the pin: `ALM-SAVE-252`
to `ALM-SAVE-256`, `ALM-NEWMAP-257`, `ALM-NEWMAP-258`, and the amended
`ALM-SAVE-239`, `ALM-SAVE-240`, `ALM-NEWMAP-236`. Rows: `DIV-3248` to `DIV-3255`
and the amended `DIV-3208`. The height limit
(126 for a new map, 127 for the brush) is unchanged; the owner decides it.

## As built

### New map players (`ALM-NEWMAP-258`)

`alm.NewMap` writes two type-5 records: id words 1 and 2, side words 0 and 1,
scalar 0, names `Self` and `Enemy`, and a sixteen-word relation row of 2 on the
record's own slot and 1 on the other fifteen. Before, the scalar was 5000, the
first name was empty and the row held 0 outside the two players.

### The save form (`ALM-SAVE-255`, `ALM-SAVE-256`, `ALM-SAVE-239`)

`alm.SaveForm` returns the bytes the original's save leaves, for a file of
version 990 or above:

- A unit whose owner matches no type-5 record is owned by player 1.
- A unit's loot link outside 1 to the loot count is written as 0, and a loot
  record a unit links takes that unit's id as owner.
- A loot record with an owner takes the owner unit's position; one whose owner is
  no unit moves to the unit with id 1, which links to it.

`Editor.SaveBytes` is that form of the session's bytes. The three file writers
(Save As, Save into a mod, the blank-map write) call it; the session, the dirty
flag and undo keep the session's own bytes.

### Deleting a sack (`DIV-3255`)

`Editor.Delete` of a loot record sets every unit's loot link to it to 0 and
lowers each link above it by one, as part of the same edit (one Undo restores the
exact bytes). Placement and clone append, and Move writes a position, so none
shifts an index. No claim states the original's rule; the choice is `DIV-3255`.

### Kept as they are

- Scripts (`ALM-SAVE-252`): the model is raw-backed, so every action node, condition
  node and trigger is written whole at the offset it was read from.
- Type 9 (`ALM-SAVE-253`, `ALM-SAVE-254`): every entry is kept and the record size
  equals the bytes (`DIV-3248`, `DIV-3249`).
- Uncleared bytes (`ALM-NEWMAP-257`): zeros (`DIV-3250`).
- The version word (`DIV-3251`), loot with owner 0 (`DIV-3252`) and a map with no
  player (`DIV-3253`) stay as the rows state.

## Proof

- `pkg/formats/alm`: `TestNewMapPlayersMatchTheClaim`, `TestNewMapUnwrittenBytesAreZero`,
  `TestSaveFormBindsAnUnmatchedOwnerToPlayerOne`, `TestSaveFormLootFollowsItsUnit`,
  `TestSaveFormMovesOrphanLootToUnitOne`, `TestSaveFormLeavesOlderFilesAlone`,
  `TestSaveFormKeepsScriptNodesAndTriggersWhole`,
  `TestSaveFormKeepsEveryType9EntryAndItsSize`. Each carries a control that fails
  on the earlier behaviour or shows the comparison can differ.
- `pkg/mapedit.TestDeleteLootRenumbersUnitLinks` (delete below, at and above a
  linked index; Undo and Redo exact) and `pkg/game.TestReleaseDeletingAGroundSackKeepsTheOtherSacks`
  (EN 30 maps, 60 deletions; RU 28 maps, 56): the ground count falls by one and the
  units' stock keeps owner, position and contents. Both fail without the rule
  (17 failing deletions on each of EN and RU). `TestSaveFormAppliesFromVersion990Up`
  states that a 991 file takes the form.
- `pkg/mapedit.TestSaveBytesWritesTheOriginalsFormAndKeepsTheSession`.
- `pkg/game.TestReleaseInstalledMapsKeepTheirBytesThroughTheSaveForm`: 38 of 38 EN
  and 34 of 34 RU installed maps equal their installed bytes through the form,
  with a control that binds an off-roster owner to player 1. The earlier
  `TestReleaseInstalledMapsSaveUnchangedWithoutAStroke` and
  `TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes` still pass.
- `TestReleaseMapEditorNewMapPlaysAsModMission` starts the new map as a mod
  mission headlessly.

## Open debt

The original's save also changes the open map; the engine changes the written
file only (`DIV-3254`). `DIV-3248`, `DIV-3251`, `DIV-3252` and `DIV-3253` wait
for claims on the helper results and constructor values they name.
