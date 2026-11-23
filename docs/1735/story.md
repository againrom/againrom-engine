# The map editor checks a second-game map, shows its problems and test-plays it

## Result

On a second-game map the editor now checks the map and shows a report and a problems mark. Saving a map that holds an
error asks first, as on the first game. Test play starts the edited second-game map in the second-game session. Checks
that rest on facts only the first game states are not run, and the report names them.

## Authority

- Owner: «Как рекомендуешь» on group G7 of the second-game editor gap list (check, problems mark, test play).
- Claim-free: no new fact about the original is stated; the rules run are those that hold for any map.

## Rules

1. When a second-game map is checked, the rules marked as holding there run and the report lists the others as not run.
2. When a second-game map holds an error, the problems mark shows and saving asks before it writes.
3. When test play starts on a second-game map, the second-game session opens the mission the map replaces.
4. When a first-game map is checked, saved or test-played, every rule and refusal is as before.

## Code

| kind | its one builder | change |
|---|---|---|
| map checker | `pkg/game/mapcheck.go` mapRules | rows gain a flag for second-game maps; skipped rules named |
| edit host seams | `pkg/game/mapeditrow.go` editRows | second row offers checker, error save and test play |
| test play | `pkg/game/mapedittestplay.go` testPlayMission | mission row chosen by the edit row |
| campaign service | `pkg/game/campaignservice.go` | campaign and mission numbers per game |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapCheckRunsOnEveryInstalledMap` | 1 |
| `TestReleaseSecondMapPlantedErrorIsReportedAndSaveAsks` | 2 |
| `TestReleaseSecondTestPlayBuildsTheEditedMapHeadless` | 3 |
| `TestReleaseFirstMapPlacementSavesAsBefore` | 4 |

Release tests at the release: `TestReleaseSecondMapCheckRunsOnEveryInstalledMap`,
`TestReleaseSecondMapPlantedErrorIsReportedAndSaveAsks`, `TestReleaseSecondTestPlayBuildsTheEditedMapHeadless`.

## Open debt

- none
