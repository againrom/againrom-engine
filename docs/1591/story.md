# A placed unit's name, speed, sight and facing, in the map editor

## Intent

The map editor's unit window overrides a placed unit's name, speed and sight in both of its views, and chooses where the unit looks when it is placed. A click on the Scenario entry shows the scenario's records and opens no trigger. A shipped map, which uses none of this, loads, plays and saves as before.

## Authority

- Owner, verbatim: "В редакторе надо разрешить в обоих видах окна переопределять Имя, Обзор, Скорость".
- Owner, verbatim: "При клике на сценарий не должен автоматом открываться первый из триггеров".
- Owner, verbatim: "Хочется иметь возможность также настраивать куда смотрит юнит при установке (но скорее всего только юнит! Так у других объектов скорее всего нет 8 вариаций)".
- Owner ruling: editor stories count 0 toward a release.
- `ALM-UNITREAD-227` (High / Medium): no holder of a type-6 runtime record reads unit file +0x1c, +0x22, +0x26, +0x28, +0x2a, +0x30, +0x33, +0x34 or +0x35. Decides that +0x30, +0x33 and +0x34 can carry the three record overrides without changing what the original does with a shipped map.
- `MOVE-119` (High): the actor base constructor sets facing and wanted facing to 0x40 plus one rand() draw over 0..0x80. Decides that an authored facing must still spend that draw.
- `ALM-UNIT-048` (Medium): +0x40 is the unit id a script names. Decides the key of the names page.
- Divergence rows: `DIV-4481` (name, speed and sight) and `DIV-4482` (facing), in `docs/divergences/ui-map-editor-units.md`.

## As-built behaviour

### Carriers

Census of every type-6 record under both first-game roots (instrument: a temporary, uncommitted test in `pkg/formats/alm` that walks every `*.alm` file and every `*.alm` entry of every `*.res` archive): EN 38 maps, 8094 records; RU 33 maps, 3991 records; no map refused. Bytes +0x30, +0x33 and +0x34 are 0 on every record of both roots.

| Field | Carrier | Values |
|---|---|---|
| speed | type-6 record +0x30, u8 (`alm.UnitSpeed`) | 0 = the kind's; 1..255 |
| facing | type-6 record +0x33, u8 (`alm.UnitFacing`) | 0 = the constructor's draw; 1..8 = north, then clockwise to north-west |
| sight | type-6 record +0x34, u8 (`alm.UnitSight`) | 0 = the kind's; 1..19 cells (`alm.SightCells`, the rings the sight march walks) |
| name | line `<unit id>=<name>` of the mission page `units.txt` | 1..32 characters, no control character; unit id 1..65535 from +0x40 |

The names page has one line per named unit, ordered by unit id, each ending in CRLF. A line that is not `<id>=<name>` with an id of 1..65535 and a non-empty name names nothing; the first line for an id holds. The editor holds the page in UTF-8: Save As writes it to `<stem>.text/<language>/units.txt` beside the map, and a mod package carries it among its mission texts. A mod serves it at `main/text/battle/m<N>/units.txt`, encoded in the install's text bytes (`EncodeInstallTextFile`), as it serves `event01.txt`. The game reads it from that address with the font-bytes codec (`unitNamesSource`).

### Editor

- The unit window's full and plain views, floating and docked in the right panel, in English and Russian, show Name, Speed, Sight and Facing. Name, Speed and Sight are overrides: an empty input shows the kind's value, a value set is listed among the map overrides, and an emptied input clears the override. Facing is a choice of the original's draw and the eight directions.
- The plain view stands Name and Facing in its first block and Speed and Sight on the card where the card draws them. A host that gives no Name field still shows the card's name in that row.
- A value outside its range is refused with the range (speed 0..255, sight 0..19, name 1 to 32 characters) and writes nothing.
- A name and the record edits of one Apply are one Undo step: the edit stages the workspace (`workspaceStage`) and `change` adopts it with the model.
- The Add window asks a new unit's facing after its group; no choice keeps the draw.
- The editor's map draws a unit with an authored facing in that direction's idle frame.
- A click on the Scenario entry of the top bar, its key F2 and the record tree's Scenario branch show the scenario palette and select no record; `chooseFirstOf` no longer selects the first scenario record. It clears the selected record when that is none of the scenario's records, and a pasted group stays selected, so Del still deletes the group while the Scenario list shows.

### Game

- Spawn: a creature's row speed and sight, and a Human's base speed and sight, are replaced by the authored values; equipment still adds to a Human's. The value enters at one point per kind: the creature spawn block (`spawnBlock.withTailStats`) and the Human derive (`data.DeriveHuman`, through `data.Hero.AuthoredSpeed` and `AuthoredSight`, which the roster template sets).
- Facing: `ConstructPlacementFacings` draws one value per placement in the original order, then gives an authored placement the byte `(d-1)*32` as facing and wanted facing. The random stream, the drop pick and every later draw equal those of the same map without the authored facing.
- Name: `loadMissionText` reads the names page when a mission opens or resumes. `mapWorld.drawnName`, the one name builder, gives the entity whose map unit id the page names that name as its last stage, so the card and the message lines that name the character show it.

### SAV

| Field | SAV home | LOAD |
|---|---|---|
| speed | the actor's current speed | restored as saved |
| sight | the actor's current sight | restored as saved |
| facing | the mover's facing byte | restored as saved |
| name | none | derived again from the names page by the entity's map unit id, which SAV keeps |

A save is never refused for an override.

## Proof

Focused tests, all passing on the candidate:

- `pkg/ui`: `TestScenarioBranchClickOpensNoTrigger`, `TestScenarioEntryOpensNoTrigger`; the Add window and plain view tests.
- `pkg/formats/alm`, `pkg/mapedit`: the carrier decode and the field table's ranges (`unittail_test.go`, `unitmotion_test.go`).
- `pkg/mapload`: `TestPlacementTailSpeedAndSightReplaceTheKindsValue`; `TestAuthoredFacingReplacesTheDrawnByteAndKeepsTheStream` (two seeds: the authored byte, the next placement's facing, the drop cell and `World.RandomState` equal those of the map without it; each direction 1..8 gives `(d-1)*32`).
- `pkg/data`: `TestRecomputeKeepsAnAuthoredSpeedAndSight`.
- `pkg/game`:
  - `TestEditorUnitSpeedSightFacingInBothViews` and `TestEditorUnitNameInBothViews`: EN and RU, docked and floating, plain and full; set, refuse, clear and Undo; the empty input shows the kind's value; the map's bytes do not move for a name.
  - `TestEditorUnitNameAndSpeedAreOneUndoStep`.
  - `TestEditorAddWindowAsksTheUnitsFacing`, `TestEditorMapDrawsTheAuthoredFacing`, `TestEditorViewsPlaceEveryHostField`.
  - `TestEditorUntouchedMapWritesByteEqual`: an opened map saved without an edit writes its bytes and no texts folder.
  - `TestEditorUnitOverridesSurviveSaveAsAndAMod`: all four overrides through Save As (`<stem>.text/<language>/units.txt`) and a mod package; the reopened package names the unit.
  - `TestModServesTheUnitNamesPage`.
  - `TestPlacedUnitOverridesReachTheGameAndSurviveSAV`: a creature placed with speed 37, sight 7, facing 3 and the name Bjorn shows the name and spawns with speed 37, sight 7 and facing byte 0x40; after SAVE and a cold LOAD all four hold, and the other unit takes no name.

Fixture-only tests do not claim install validation. The release tests that spawn placed units, draw the editor's unit window, save a placed unit or walk the map editor modes table run at the next release.

## Open debt

- A Human's authored speed and sight after SAVE. The SAV Human record has no home for the authored base. SAVE gives every actor a source actor load; after LOAD the next derive of that Human (equip, training, a potion) is `data.HumanState.derive`, which has no authored input, so speed and sight return to the derived values. The stored current values hold until that derive. Carrying a Human with a source load out of the mission rebuilds its hero from the source and drops the authored base; a Human that was never saved keeps it into the next mission. Fix: LOAD derives the authored base from the map's placement by map unit id into the derive input.
- A deleted unit's line stays in the names page. A later unit given the same id (one above the highest, when the deleted unit held the highest) takes that name.
- Units that share a unit id share a name; a unit with id 0 cannot be named.
- The names page is per language folder, as the mission's other pages are.
- `pkg/ui/testdata/mapeditor_modes.golden` is not rewritten in the lane: the Add window's facing step and the Scenario entry move the sampled modes walk, and the full walk (`MAPEDITOR_MODES_WRITE`) runs at landing.
