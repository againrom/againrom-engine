# A new multiplayer character is made by the one generator and stored

## Result

In both games New Character in the multiplayer character list opens the same character generator the campaign uses.
Accept saves the new character in the original game's own character file format, in the player profile's folder for
that game, and shows the list again with the new character selected. Cancel returns to the list and saves nothing.
An existing character file is never replaced. The campaign's own new game opens as before.

## Authority

- Owner (DIV-2369): multiplayer milestone 1 includes a character generator reached from the player list.
- Owner (DIV-5194): character files live in the player profile, one folder per game, in the original's format.
- `MENU-169`, `R2-ENGINE-1840`: New Character runs the campaign's creation pages.
- `HERO-213`: the first game's AlCh write order and short form; `R2-SESSION-513`, `R2-SESSION-517`,
  `R2-ENGINE-1853`: the second game's sections, E's chain and pads, and a new file's sections.
- DIV-5240, DIV-5241: what a new file holds where the claims are silent. DIV-5242: names, keys and pads.

## Rules

1. Accept on the New Character row opens the generator; Accept on a character row opens nothing.
2. Cancel in the generator returns to the list and writes no file or folder.
3. Accept in the generator writes one new file and shows the list with that file's row selected.
4. The first game's file is the short AlCh form of the name and four statistics; the second game's is A, B, C and E.
5. A written file reads back through the store's reader as the character the generator made.
6. A taken file name is skipped and its file kept; with no free name the write is refused and the generator stays.
7. The campaign accept keeps its start: the second game shows the town, the first opens the mission.

## Code

| kind | its one builder | change |
|---|---|---|
| generator | `pkg/game/chargen.go` generatorBegin over the generatorStart rows | the character start, as a start row and a target |
| character store | `pkg/game/characterstore.go` characterStoreRow | each row's file name and record; storeCharacter publishes through hostfile |
| file name builder | `pkg/game/characterstore.go` characterStoreRow fileName | first `char%04d.chr`, second `1%d.a2c` |
| record writer | `pkg/formats/alch` Write, `pkg/formats/a2c` Write | the writers beside the readers, one per format |
| list page | `pkg/ui/listrole.go` characterListRole | its accept opens the store's generator on the New row |
| engine words | `pkg/words/text/*/strings.toml` | the two refusal lines |

## Proof

| test | rules |
|---|---|
| `TestNewCharacterRowRunsTheStoresGenerator` | 1, 2, 3 |
| `TestNewCharacterIsGeneratedStoredAndListed` | 1, 2, 3, 4, 6 |
| `TestStoredCharacterReadsBackAsGenerated` | 4, 5, 6 |
| `TestCharacterStartLeavesTheCampaignStarts` | 7 |
| `TestWriteShortFileMatchesThePageFile`, `TestWriteThenReadEqualsTheInput` (alch) | 4, 5 |
| `TestWriteNewCharacterMatchesTheClaimsFile`, `TestWriteThenReadEqualsTheInput` (a2c) | 4, 5 |
| `TestReadRefusesBrokenSections` | 5 |

Release tests at the release: `TestReleaseInstalledCharacterFilesReadAndWriteBack`.

## Open debt

- MP-6 map entry: the server loader needs section D, which a new second-game file lacks (DIV-5241).
- MP-6 map entry: the stored first-game file keeps no sex, class, skill or weapon (DIV-5240).
