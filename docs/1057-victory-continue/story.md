# Victory and Continue

## Player result

A campaign win opens the decoded 384x180 success panel. Victory is focused
first. Return and the first control choose Victory. Escape and the second
control choose Continue. Continue closes the panel and leaves the completed
world running.

The campaign End Quest confirmation starts with Victory, not Change Map and
Victory together. Victory is disabled before Continue and enabled afterwards.
A standalone map retains Change Map. Immediate Victory and the later End Quest
Victory reach one one-shot `FinishMissionWithRoster` boundary.

## Authority

`MISSION-VICTORY-029` through `MISSION-VICTORY-036` establish the success-panel
geometry, labels, input results, phase-2 End Quest gate, Continue reachability,
simulation continuity and load-time latch reset. `DIV-099` retains the
research-silent menu choices. `DIV-407` records the Unknown success-panel
reappearance after loading a save taken after Continue.

## Touched surfaces

- `pkg/ui`: success composition, control hit testing, input choice and End Quest rows.
- `pkg/game`: outcome-session latch, campaign completion wiring and install word 154.
- Persistence: no field was added; the simulation outcome remains the persisted fact.

## Proof

- `TestSuccessPanelGeometryAndInputStateTable`
- `TestCampaignVictoryContinueStateTableAndOneCompletionBoundary`
- `TestLoadedCompletedWorldResetsDelayedVictoryPermission`
- `TestSuccessContinueUsesDialogsLocalIndex154`
- `TestOutcomeChildCreationPlaysCompleteAndFailedSelectorsOnce`
- `TestReleasePauseMenuVisitsEveryDestination`
- `TestReleasePauseMenuCoversFullActionPopulation`

The EN and RU release pair reads Victory and Continue from each installed
dialog table. It exercises disabled Victory before Continue, enabled Victory
after Continue, and the standalone Change Map path.
