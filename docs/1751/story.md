# A mod's global variables carry across its campaign and its saves

## Result

A mod's global variables now last for the whole campaign, not one mission.
A new campaign starts at the declared values; a won mission hands its values to the next one.
Losing, leaving or restarting a mission throws away what that mission wrote.
Town and mission saves keep the values; an original or older save starts at the declared values.
A guest chain that wins passes its values back to the party that waited for it.

## Authority

- Owner: «a mod's global variables carry across its campaign and its saves»; the store sits on the town.
- Owner: loss, exit and restart discard the mission's writes; a restart seats the entry values.
- `DIV-5108`: the globals, their seat, the World wrapper and now the campaign scope.

## Rules

1. When a campaign starts, its mod's missions see the declared initials.
2. When a mission starts, the World holds the values the campaign store holds for its mod.
3. When a mission is won, the campaign store takes the World's values, and the next mission starts with them.
4. When a mission is lost, left or restarted, the store keeps its values; a restart seats the entry values.
5. When the town is saved and loaded cold, the store is the one before SAVE.
6. When a mission is saved, loaded and restarted, the restart seats the entry values.
7. When a save has no globals record, as an original or an older save, the mission seats the initials.
8. When the declaration changed, a kept id keeps its value, a new id takes its initial, a dropped id vanishes.
9. When the record holds a mod outside the active set, its values are kept and written back unchanged.
10. When a guest chain wins, the waiting party's store holds the chain's values.
11. When a save holds no globals, its mod mark has the bytes it had before.
12. When the same campaign is replayed from the same start and commands, the globals and World hashes are equal.

## Code

| kind | its one builder | change |
|---|---|---|
| World carrier | `pkg/game/worldcarriers.go` worldCarrierSet | "World mod globals" seats the store; town points capture, restore, `townWin` |
| won-mission finisher | `pkg/game/winfinish.go` finishWin | runs `townWin` before the rows; a refused win puts the store back |
| guest return | `pkg/game/winfinish.go` guestHome | the stash restore keeps the chain's store |
| mod mark | `pkg/game/modmark.go` markDocumentForMods, applyModMark | `globals` per mod as `{id, value}`; at most 1024 per mod |
| SAV LOAD source | `pkg/game/originalrestore.go` decodeOriginalSource | the decoded town takes the mark's store |
| campaign store | `pkg/game/modglobals.go` modGlobalStore | the store on `Town`, its admission and its mark form |

`internal/archtest/worldcarriers.go` names the three writers of `Town.globals` outside the carrier set.

## Proof

| test | rules |
|---|---|
| `TestGlobalsCampaignWinCarriesToTheNextMap`, `TestGlobalsYardMapIsTheBuiltMap` | 1, 2, 3 |
| `TestGlobalsCampaignLossAndRestartDiscard` | 4 |
| `TestGlobalsCampaignTownSaveAndColdLoad` | 5 |
| `TestGlobalsCampaignMissionSaveLoadAndRestart` | 6 |
| `TestGlobalsCampaignOlderAndOriginalSavesTakeInitials` | 7 |
| `TestGlobalsCampaignChangedDeclaration` | 8 |
| `TestGlobalsCampaignMissingModKeepsItsValues` | 9 |
| `TestGlobalsCampaignGuestChainPassesItsValues` | 10 |
| `TestModMarkWithoutGlobalsKeepsItsBytes` | 11 |
| `TestGlobalsCampaignReplayIsEqual` | 12 |

Release tests on EN and RU: `TestReleaseGlobalsMillCarriesToTheYard` (2, 3),
`TestReleaseOriginalControlResaveUnchanged` (11).
The witness hashes are in the review report.

## Open debt

- The editor's node lists do not offer the global nodes yet.
- A second-game form of the globals waits for an owner decision (`DIV-5108`).
