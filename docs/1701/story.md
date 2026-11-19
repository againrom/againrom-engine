# A second-game town file keeps its hired companion by name

## Result

A town save of the second game that the original wrote, with Iglez in the party, loads with Iglez in the party under
his name, his own statistics and his kit, and the inn keeps the hires the save had chosen. Saving that town and loading
it again shows him the same. Dina in a Druids' town save shows her name too. Before, both loaded with a blank name.
A town save without them loads and saves exactly as before.

## Authority

- Owner ruling: an original town save loads with its complete state, and our town save writes a file the original
  loads into the same town.
- Owner remark: the hired companion of a town file (party identity 22) is not restored.
- `R2-SESSION-352`: the original stores every party member but the hero with an empty name.
- `R2-ENGINE-908`: the original fills an empty name with the character's line of the NPC name list.
- `R2-ENGINE-105`, `R2-ENGINE-106`, `R2-ENGINE-525`: the party identity word of a joined companion.
- DIV-4337 (narrowed), DIV-4187 (his name is the line of his identity).

## Rules

1. A town file holding a companion with an empty name lists him in the party under his identity's name line.
2. He keeps the file's identity, kind and statistics, and the inn keeps the file's hire selections.
3. The town save after that load writes him back; its load lists him again the same, name included.
4. A second-game save stores a companion named by his line with an empty name, as the original does.
5. A town file without a companion loads and resaves byte for byte as before; the first game is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| second-game SAV LOAD party | `restorePartyCharacters` (`RestoreParty`) | a companion with an empty name takes his identity's line |
| second-game town and mission SAV writer | `ExportCurrentSave` through `writeSecondTownHolders`, `secondFormHolders` | one more holder: a companion's `Name`, site `holderText` |
| party member | `mapload.CharacterMember` via `townInstall.character`; LOAD via `restoredMember` | unchanged |
| hire roster | `bankHires`, `hireOffers`, `departHires` | unchanged |

## Proof

| test | rules |
|---|---|
| `TestSecondTownFileNamesTheLoadedCompanion` | 1, 2, 3, 4 |
| `TestSecondTownSaveStoresTheCompanionUnnamed` | 4 |
| `TestSecondTownFileWithoutACompanionRoundTrips`, `TestSecondTownFileAfterMissionTwentyLoadsBack` | 5 |
| `TestReleaseSecondTownFileKeepsTheHiredCompanion` (owner file, skipped when absent) | 1, 2, 3 |

Release tests at the release: `TestReleaseSecondTownFileKeepsTheHiredCompanion`,
`TestReleaseSecondTownDruidsLoadKeepsEachPack`, `TestReleaseSecondOriginalMissionFileKeepsTheCompanion`, EN and RU.
Evidence: `review/story1701-rom2-town-hire/RESULT.md`.

## Open debt

- DIV-4337: the name line rests on a branch-candidate claim; the original's word for identity 22 is unread.
- DIV-4305: the companion's other record fields without a source.
- The owner loads the resave in the original.
