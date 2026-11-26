# Both games read the route values from the install's map data

## Result

In both games the six route values of the map data's Path Finding section now steer the walk and the pursuit: how
often a stored route and a near step are searched again, how far the searches may look past the straight line, and
when a unit aims at the end of its route or at its head. A mod or an edited install that changes a value changes how
units walk; an install without changes plays exactly as before. A key the file lacks, or holds as text, keeps the
value the game has always used. The speed multiplier and the sight shift are still fixed.

## Authority

- Owner: «Как рекомендуешь» on reading the route values of the map data in both games, the old values as defaults.
- `MOVE-PARAM-006`, `MOVE-208`, `MOVE-TERM-003`: the keys, defaults and their use in the first game.
- `R2-ASSET-383`, `R2-ENGINE-1472`, `R2-ENGINE-1761`: the second game's keys, shipped values and walk bodies.
- DIV-5280: the second game's use of the two scan values; DIV-5051 now records what stays fixed.

## Rules

1. A key of the Path Finding section is read under its stored name, cut to 15 characters.
2. A missing key, or one that is not an integer, holds its shipped value; both installs ship the shipped values.
3. The route-end goal and the pursuit's re-search and head removal use the section's values in both games.
4. A stored route is searched again past the static refresh rate, a listed near step past the dynamic one.
5. The far and near search budgets add the static and the dynamic scan-ahead value.
6. The values are world input: not in the world hash or the save, so a save loads under the opening install's values.
7. An unmodified install of either game gives the same mission hash as before.
8. A StaticIsntNeeded below 3 is used as written; the route-end index stops at the last route cell.

## Code

| kind | its one builder | change |
|---|---|---|
| map.reg loader | `pkg/data/originalterrain.go` OriginalMapRegistry | reads the six keys beside the cost table |
| map.reg file read | `pkg/game/table.go` mapRegistry | one read for both games; costs stay first-game only |
| route-end goal | `pkg/sim/pursuitsearch.go` routeEndGoal | takes the values as a parameter |
| search budget | `pkg/sim/route.go` generationBudget | slack from the values |
| world input | `pkg/sim/buildrules.go` BuildRules | carries the values by the build rules route |

## Proof

| test | rules |
|---|---|
| `TestOriginalMapRegistryReadsTheStoredPathFindingNames` | 1 |
| `TestOriginalMapRegistryKeepsTheShippedValueOfAMissingPathFindingKey` | 2 |
| `TestRouteEndGoalFollowsThePathFindingValues`, `TestAPursuitReadsStaticIsntNeededFromThePathFinding` | 3 |
| `TestAWalkReadsStaticRefreshRateFromThePathFinding`, `TestAWalkReadsDynamicRefreshRateFromThePathFinding` | 4 |
| `TestGenerationBudgetFollowsTheScanAheadValues` | 5 |
| `TestAWorldWithNoPathFindingRunsTheShippedValues` | 2, 6 |
| `TestReleaseShippedPathFindingKeepsTheMissionHash` | 7 |
| `TestAStaticIsntNeededBelowTheLookAheadKeepsTheRouteIndexInsideTheRoute` | 8 |

Release tests at the release: `TestReleaseShippedPathFindingKeepsTheMissionHash` (rule 7),
`TestReleaseModdedPathFindingReachesTheDefinitions` (rules 1, 2, 3), each on one install of each game.

## Open debt

- The speed multiplier and the sight shift stay compiled constants (DIV-5051).
- A mod cannot ship its own map data: install files of a mod take only pictures, sounds and texts.
- The second game's use of the two scan values is Medium (DIV-5280).
