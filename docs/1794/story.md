# The map registry sets the speed multiplier and the sight shift

## Result

Both games read the speed multiplier and the sight shift from the map registry of the loaded root or of a mod.
A ground unit's step rate takes the world's multiplier, and the computer units' sight takes the world's shift.
A missing key or file keeps 8 and 7, so a shipped install plays as before: same rates, same sight, same World hash.

## Authority

- Owner: card o-speed-multiplier, «Как рекомендуешь»: both keys from map.reg in both games; code values stay defaults.
- `R2-ENGINE-1472` (High keys and defaults; file values Unknown); `R2-ASSET-383` (High: shipped values equal defaults).
- `TERR-FOG-088` (High): the first game's fog keeps the compiled shift 7; only the computer units' sight reads the key.
- Divergence: DIV-5051 states the values and the load refusal. A map file's own override is Unknown; none is built.

## Rules

1. `data.OriginalMapRegistry` reads SpeedMultiplier and ScanShift; a missing key or file holds 8 and 7.
2. The values reach the World through `sim.BuildRules` (Path, Scan), set by `mapRegistry` beside the route keys.
3. `rateOf` takes the multiplier as an argument; `stepRate` passes the world's. No package value holds it.
4. The computer units' sight takes the world's shift and the tables built for it; the fog keeps shift 7.
5. A multiplier outside 1 to 255 or a shift outside 1 to 8 is refused at load, naming the key; MODS.md says so.
6. A multiplier of 4 halves the step rate against 8; a shift of 6 or 8 changes the computer units' sight, not the fog.

## Code

| kind | its one builder | change |
|---|---|---|
| map.reg reader | `data.OriginalMapRegistry`, `game.mapRegistry` | two keys and the bounds |
| build rules | `sim.BuildRules` | `PathFinding.SpeedMultiplier`, `Scanning` |
| rate function | `rateOf` | a multiplier parameter |
| sight function | `marchInto` over `sightReader` | the AI reader takes the world's shift |

## Proof

| test | rules |
|---|---|
| `TestOriginalMapRegistryReadsSpeedMultiplierAndScanShift` | 1 |
| `TestMapRegistryCarriesSpeedMultiplierAndScanShiftInBothGames` | 1, 2 |
| `TestSpeedMultiplierIsTheWorldsOwn` | 3, 6 |
| `TestScanShiftMovesTheAISightOnly` | 4, 6 |
| `TestOriginalMapRegistryRefusesValuesThatBreakTheArithmetic` | 5 |
| `TestModMapRegistryRefusals` | 5 |
| `TestShippedScanShiftIsTheCompiledShift` | 1 |

Release tests at the release: the rate and sight tests on the English and the Russian roots of both games.

## Open debt

- Whether a map file overrides the two keys in the original is Unknown (DIV-5051).
- Whether the second game's fog reads ScanShift is not shown; the fog keeps 7 in both games.
