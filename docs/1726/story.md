# One cell-slot builder for both worlds

## Result

A second-game mission now holds its cell slots from its start and moves them at each crossing as the first game's
mission does. A second walker who enters a layered cell whose slot another actor holds no longer resets the cell's
decayed step cost, and the cost stays decayed after he leaves. The slot table is written into SAV and a loaded mission
keeps it. The first game's rules are as before. A SAV written before this change has no table and fills it at its next
crossings.

## Authority

- MOVE-088 (High for the order and both slot branches): the release recomputes a cell only while its slot is held; the
  occupy recomputes only while its slot is empty; a taken slot refuses the occupy.
- SAV-CELLLEAVE-584 (High): the release clears any non-zero slot without comparing it with the leaving actor.
- Owner: one cell-slot builder for both worlds; DIV-4968 accepted until then.

## Rules

1. A mission start without planes holds its actors' cell slots from construction.
2. A crossing on a World without planes releases and occupies through the slot builder; a held slot skips the recompute.
3. A second walker entering and leaving a held cell leaves the decayed cost, with planes or without.
4. A slot row exists while a slot holds it; an emptied row is deleted.
5. The slot table is written into SAV as Cell actor edges with the flag SlotCarrier, not rebuilt at LOAD.
6. A mid-mission SAVE and cold LOAD keep every slot, and the loaded World crosses as the saved one does.
7. The slot table is not imported motion authority.
8. A script's removal frees an actor's slots and its return takes them, with planes or without.

## Code

| kind | its one builder | change |
|---|---|---|
| slot release and occupy | `releaseCellSlots`, `occupyCellSlots` in `pkg/sim/savedcellplanes.go` | recompute by world kind |
| crossing | `recomputeCrossing` in `pkg/sim/savedcellplanes.go` | footprint recompute branch removed |
| slot table | `cellRecords` rows by `createSavedCell`, `deleteEmptySavedCell` | rows without planes |
| SAV writer of the slots | `projectSavedActorMotions` in `pkg/game/savmotion.go` | writes the table; `SlotCarrier` |
| mission start | `newWorld` through `Terrain.StartSlots` | set in `pkg/game/missionentry.go` |

## Proof

| test | rules |
|---|---|
| `TestCrossingKeepsADecayedCellWhileASlotIsHeld` | 1, 2, 3 |
| `TestSlotTableSurvivesTheByteForm` | 4, 5, 7 |
| `TestSlotTableCrossesSaveAndColdLoad` | 5, 6 |
| `TestPresenceRemovalAndReturnMoveTheSlots` | 8 |

Release tests at the release: `TestReleaseMissionRoutesAndCostsMatchTheNativeReader` shows no walk difference on
mission 80 from the crossing recompute. Evidence: `review/story1726-cell-slot-builder/RESULT.md`.

## Open debt

- DIV-4968 narrows to a SAV without the table and the imported-mover loop of a World without planes.
