# A first-game trigger action teleports any unit from one cell to another

## Result

In the first game's map editor, a trigger can now run a Teleport action with a source cell and a target cell,
each typed or chosen by clicking on the map. In play, whenever that action runs, any living unit standing on
the source cell moves to the target cell, or to the nearest free cell when the target is occupied, with the
same relocation effect a Teleport trap shows. Repeating triggers run the action each time a unit steps onto
the source cell, maps that use it save as extended maps, and mid-mission saves round-trip.

## Authority

- Owner: «в rom1 редакторе и игре надо добавить такое же действие как в rom2 для телепорта из одной точки в
  другую, для триггера. юз кейс: триггер который многократно выполняется если кто-то наступил на телепорт
  и его телепортирует (любого персонажа)».
- DIV-5200: the first game's Teleport trigger action, its extended-script gate, and nearest-free-cell fallback.
- DIV-4599: the extended script record carried by `.almx` maps.
- MOVE-129, AI-535, DIV-5176: the first game's trap teleport relocation and stand reset.

## Rules

1. The trigger wizard and action popup offer Teleport with Source and Target cells, both pickable on the map.
2. When the action runs in play, any living unit on Source moves to Target by the trap teleport's move and effect.
3. When Target is taken, the unit moves to the nearest free cell; an empty Source or dead body moves nothing.
4. A repeating trigger with the action moves each unit that steps on Source, and a mid-mission save round-trips it.
5. A map using the action requires the extended script record and saves as `.almx`; a map without it keeps its bytes.
6. The second game's action table is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| first-game script actions | `pkg/scriptprofile/ops.go` `firstActions` | new row `0x4106` with source/target cells and wizard preset |
| unit move to a cell | `pkg/sim/cellentry.go` `finishCellMove`, `relocateActor` | trap move shared with `teleportCellOccupants` and `nearestFreeCell` |
| script instant dispatch | `pkg/sim/script.go` `runInstantObserved`, `scriptInstantSupported` | `ScriptInstantTeleport` arm |
| extended script gate | `pkg/mapedit/scriptform.go` `CountTeleportNodes`, `FoldScript` | `LimitTeleport` refuses fold and asks to extend in wizard and details |
| cell picking in editor | `pkg/ui/mapeditortrigger.go` `scriptCellPicked` | reused unchanged by wizard `wpCell` and node `cell0`/`cell2` |
| words tables | `pkg/words/text/{en,ru}/strings.toml` | English and Russian keys for the action, wizard row and form limit |

## Proof

| test | rules |
|---|---|
| `TestFirstGameWizardAndNodePopupOfferTeleportWithMapPick` | 1 |
| `TestFirstGameTeleportActionMovesAnyUnitAndFindsNearestFreeCell` | 2, 3 |
| `TestFirstGameTeleportActionRepeatsAndRoundTripsSave` | 4 |
| `TestFirstGameTeleportActionWritesAlmxReadsBackAndPlays` | 1, 4, 5 |
| `TestFoldScriptRefusesATeleportNode` | 5 |
| `TestSecondGameActionsUnchangedByFirstGameTeleport` | 6 |

Release tests at the release: none (synthetic maps; no installed first-game map holds action `0x4106`).

## Open debt

- First-game triggers lack an any-player unit-on-cell check; Always or nearest-unit works (story 1774 adds checks).
- Reading claim `R2-ASSET-386` via `knowledge` was blocked by sandbox permissions; verify in the lane if needed.
