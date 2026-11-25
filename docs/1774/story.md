# Fire a trigger when a unit leaves its starting cell

## Result

A first-game map maker can choose a placed unit from a list or by clicking it on the map.
The condition remembers the cell shown when the unit was chosen. When the unit stands on another cell,
the trigger can run its usual actions. Returning to the starting cell makes the condition false again.

## Authority

- Owner direction: fire an event when the chosen character leaves the cell it stands on.
- TRIG-COND-003 places the original condition dispatch at opcodes 1 through 22.
- DIV-4599 defines the first game's extended script record. DIV-5204 records the new condition.

## Rules

1. Choosing a unit records its map ID and starting cell together. The row shows the cell after reopening the map.
2. The editor asks to extend a plain map. A map with the condition saves as ALMX and cannot fold back to ALM.
3. Saving an ordinary map without edits preserves its bytes.
4. At a script pass, walking or teleporting off the cell satisfies the condition. Returning clears it.
5. A dead unit uses its current cell. A removed unit with a terminal record uses its retained cell.
6. The trigger keeps its repeat setting. Saving and loading preserves its reference cell and fire-once latch.

## Code

| kind | its one builder | change |
|---|---|---|
| Condition and ready row | `pkg/scriptprofile/ops.go` firstChecks | Extended check 0x4105 with a unit, X and Y |
| Unit picker | `pkg/ui/mapeditorwizard.go` wizardChoices and `pkg/ui/mapeditorparampick.go` unitValueAt | Restrict the unit field to placements and write the placement cell |
| Script form | `pkg/mapedit/scriptform.go` UnitLeftCellLimit | The wizard, save and fold require the extended record |
| Trigger record | `pkg/formats/alm/scriptrecord.go` EncodeScriptRecord and DecodeScriptRecord | Existing record binds the condition and actions |
| Script pass | `pkg/sim/script.go` runCheck | Compare the resolved unit's cell with the two node operands |
| Mission save | `pkg/game/currentscriptprogram.go` captureCurrentScriptProgram | Existing check operands, results and latch |
| Editor words | `pkg/words` Book.Text | English and Russian condition, placement and form text |

## Proof

| test | rules |
|---|---|
| `TestUnitLeftCellEditorRoundTrip`, `TestUnitLeftCellIsOfferedOnlyForFirstGameMaps` | 1, 2 |
| `TestUnitLeftCellInAPlainMapRequiresConversion`, `TestUnitLeftCellIsInertOnPlainMachines` | 2 |
| `TestAnOriginalMapSavesWithoutTheExtendedRecord` | 3 |
| `TestUnitLeftCellWalkAndTeleport` | 4, 6 |
| `TestUnitLeftCellUsesTheRetainedCell` | 5 |
| `TestUnitLeftCellMissionSave` | 6 |

Release witnesses on EN and RU: `TestReleaseTriggerWizardMapPlaysTheTrigger`, `TestReleaseTriggerWizardFrames`.

## Open debt

- ROM2 needs an owner decision on the extended script form (DIV-4599, DIV-5204).
