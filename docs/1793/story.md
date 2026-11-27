# A mod carries its own route and terrain values

## Result

A mod can now carry its own map registry file, and both games read the route values and the terrain costs from it.
A value the file leaves out keeps the value the original ships. A mod without the file changes nothing. A mod that
carries the file changes game state, so a save marks it as it marks a mod with a text file.

## Authority

- Owner: board card o-mod-mapreg, answer «Как рекомендуешь»: a mod ships its own map.reg, with a fixture mod test.
- Owner: the documentation states the route refresh limit 0..255.
- No divergence row: a mod file is an Againrom feature, and the original has no mods.

## Rules

1. The mod install-file table admits map.reg as one more row; no other .reg file is admitted.
2. The file is served at world/data/map.reg only and must parse as a registry, else the mod is refused by name.
3. Both games read the mod's [Path Finding] values through the one loader; a missing key holds its shipped value.
4. A mod that carries the file is not state-free; a mod with pictures and sounds only still is.
5. docs/MODS.md names the file, path, keys, missing-key rule and refresh limit (above 255 never, below 0 always).
6. The map editor lists the install's map.reg among the files a mod can replace, as it lists other admitted files.

## Code

| kind | its one builder | change |
|---|---|---|
| mod install-file admission | `pkg/mod/kinds.go` dataKinds | one row, install map registry |
| file kind | `pkg/mod/files.go` FileKind | FileRegistry: binary, changes state |
| install file serving | `pkg/game/modfiles.go` serveInstallFiles | one case: address and parse check |
| map.reg loader | `pkg/data` OriginalMapRegistry, `pkg/game/table.go` mapRegistry | unchanged |

## Proof

| test | rules |
|---|---|
| `TestInstallFilesAdmitOnlyTheMapRegistry` | 1 |
| `TestOnlyATextAndTheRegistryChangeState` | 4 |
| `TestModMapRegistryReachesBothGames` | 3, 4 |
| `TestModMapRegistryRefusals` | 2 |
| `TestModsDocListsFixedTree` | 5 |
| `TestFixedTreeEveryKindPacksAndKeepsBytesAndDigest` | 6 |

Release tests at the release: `TestReleaseModdedPathFindingReachesTheDefinitions` (loader on the install).

## Open debt

- The mod's file replaces the install's whole file; a missing key takes the shipped value, not the install's copy.
