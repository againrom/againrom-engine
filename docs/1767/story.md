# A first-game cell with a Teleport trap moves the unit that steps on it

## Result

On a first-game map a Teleport trap now works: a walking unit whose step reaches the middle
of the trap's cell is moved to the trap's target cell and stands there for the reset. A unit
under a Move order then walks back to its destination (Medium). A target with no room leaves
the unit on the trap. Flying units are not moved; a trap of any other spell casts as before.
In the map editor such a trap reads as a teleport from its cell to its target.

## Authority

- Owner, on the reading of the first game's cell tail: «Да, в очередь».
- TRIG-CELLTAIL-035 (High): the entry caster skips spell 26; the arrival reader takes it and reads tail bytes 4 and 5.
- MOVE-129: the two footprint checks are High; the final relocation and reset are Medium; no original run.
- AI-535 (High / Medium): the reset's stores; state 0xc, the post at the cell; no store to the order's destination.
- AI-649 (High / Medium): the next group pass re-issues a Move member's walk with the same `ord+0xa`.
- TERR-229 (High), ALM-TRIG-049 (Medium), ALM-T9CONTROL-175 (High): the type-9 record writes the six bytes.
- UNIT-M10ENTRY-055 (High / Unknown): ground and ghost units take a cell record, air units do not.
- DIV-5176: the engine's order of checks, the next-tick move and the reset's extra stores.

## Rules

1. When a ground unit's step ends centred on a spell-26 cell, it moves to bytes 4, 5 and stands; a Move order resumes.
2. When the destination refuses the unit's footprint or is (0,0), the unit stays on the trap cell.
3. When an air unit is on such a cell, it is not moved.
4. When a cell holds another spell, entering it casts that spell and the unit is not moved.
5. When instant 25 writes spell 26, the tail is {26, power, 0, y, 0, y} and a LOAD keeps it.
6. When a first-game type-9 trap holds spell 26, the editor reads it as a teleport, adds and changes it.
7. When such a trap is saved, the type-9 record reloads to the same tail in a world.
8. When first-game action 25 is offered, it has no teleport form.
9. When the installed first-game maps are read, no campaign map holds spell 26; Cross.ALM (English) holds three.

## Code

| kind | its one builder | change |
|---|---|---|
| cell event rule | `pkg/sim/cellentry.go` cellEventBodies | event id from the game's vocabulary |
| game profile row | `pkg/sim/gameprofile.go` newGameProfile | first game selects the relocating body and the stand |
| script vocabulary | `pkg/scriptprofile/ops.go` vocabularies | first-game CellMove 26 |
| relocation | `pkg/sim/cellentry.go` relocateActor | unchanged, reached by the first game |
| one-unit stand | `pkg/sim/script.go` standScriptedUnit | unchanged, called after the cell move |
| cell tail repair | `pkg/sim/celltail.go` repairRelocatingCellEvents | skips the first game's encoder |
| type-9 record view and choices | `pkg/game/mapinspectioncasts.go` inspectionEnchantment, castSpellChoices | unchanged, read the first-game CellMove |

## Proof

| test | rules |
|---|---|
| `TestFirstGameTeleportTrapMovesTheUnitWhoseStepEndsOnIt`, `TestFirstGameTeleportTrapMoveOrderResumesAfterTheMove` | 1 |
| `TestFirstGameTeleportTrapRefusedDestinationKeepsTheUnit`, `TestFirstGameTeleportTrapToCellZeroQueuesNothing` | 2 |
| `TestFirstGameTeleportTrapLeavesAnAirUnit` | 3 |
| `TestFirstGameOtherCellSpellStillCasts` | 4 |
| `TestFirstGameInstantTwentyFiveTeleportKeepsItsBytes` | 5 |
| `TestFirstMapTeleportTrapAddedChangedSavedAndReloaded` | 6, 7, 8 |
| `TestReleaseInstalledFirstMapTeleportTrapCensus` | 9 |

Release tests at the release, EN and RU: `TestReleaseInstalledFirstMapTeleportTrapCensus`,
`TestReleaseInstalledMapsSaveUnchangedWithoutAStroke`,
`TestReleaseM10CellEntry1084UsesInstalledBindingPointerLightningAndNativeSave`.
Evidence: `review/story1767-rom1-teleport-trap/RESULT.md`.

## Open debt

- No original run has reached a Teleport trap cell; Cross.ALM's three cells are the witness to take (DIV-5176).
- The call after the reset and the occupy return are untraced (MOVE-129 Medium); not modelled.
- A destination of (0,0) is not moved; an instant-25 trap's destination x is 0, a cell at the map edge.
- No release test plays a unit onto Cross.ALM's traps.
