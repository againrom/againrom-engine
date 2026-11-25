# A second-game map carries the engine's extended script record

## Result

A second-game map can hold the engine's extended script record, the same one a first-game ALMX map holds.
The map loader, the editor model and the mission save read it for both games.
A second-game program with the record runs on the extended machine with the second game's own operations.
The condition «Unit left its cell» then holds while the chosen unit stands on another cell.
A second-game map without the record reads and saves as before.

## Authority

- Owner direction: the same extended record for the second game.
  A map with an extension is saved in the extended form, one without it as the original.
- DIV-4599 defines the extended script record; DIV-5204 the condition 0x4105.

## Rules

1. The second game's map reader takes the script record by the first game's reader; texts stay first-game only.
2. A second-game profile on the extended machine keeps the second game's operations and node rules.
3. A second-game extended program runs 0x4105 and keeps it across the world byte form round trip.
4. A repeating trigger fires each pass away from the cell, a one-shot trigger once.
5. The second game's condition table offers «Unit left its cell» and does not gain 0x4106.
6. The first game is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| Extended script record reader | `pkg/formats/alm/alm.go` walkRecordsIn | Format-row data `againromRecords` in `rom2.go` |
| Extended script record writer | `pkg/formats/alm/scriptrecord.go` AppendScriptRecord, FrameScriptRecord | AppendScriptRecord takes the map's format; FrameScriptRecord writes the first game's tag word and the second-game reader ignores it |
| Script profile | `pkg/scriptprofile/profile.go` For | One extended-machine overlay of each game's profile |
| Map script form | `pkg/mapedit/scriptform.go` Profile, ExtendScript, FoldScript | Format row holds operations; the format's own trigger row and tally |
| Map save extended-form choice | `pkg/game/mapinspectionscriptform.go` scriptSaveForm | Second row offers the script form seam |
| Script check runner and load | `pkg/sim/script.go` NewExtendedScript, `pkg/mapload/script.go` compileScriptFrom | Dialect argument |
| Mission save | `pkg/game/currentscriptprogram.go` compile; `pkg/sim/scriptbinary.go` unmarshalExtendedScript | Extended program of either dialect |
| Condition table rows | `pkg/scriptprofile/ops.go` secondChecks | Row 0x4105 |

## Proof

| test | rules |
|---|---|
| `TestExtendedRecordCompilesGroupsListsAndMisses` | 1, 2 |
| `TestUnitLeftCellRunsOnTheSecondGamesExtendedMachine` | 2, 3, 4 |
| `TestUnitLeftCellIsOfferedForBothGames` | 5 |
| `TestExtendedProgramKeepsItsLateChecksAcrossTheByteForm` | 3, 4 |
| first-game 1773 and 1774 tests | 6 |

Release witness on each second-game install: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.

## Open debt

- No fixture test drives the second-game wizard to a saved `.almx`.
  New trigger did not open the wizard on the second-game editor fixture.
- Extend and fold of a second-game script move the type-0 node count word +0x28; no focused test.
- No game-level SAV test of a second-game mission with the condition.
