# The character list shows the character files of both games

## Result

The Multiplayer character list now shows the characters stored in the player profile, one folder per game, by name and
newest first. The list ends with a New Character row, which does nothing yet. A file the game cannot read is left out,
and the line under the list says how many were left out. In the first game Delete asks the Load page's question and
then removes the chosen character file alone; the second game's page has no Delete. Opening the list writes nothing.

## Authority

- Owner: «работает кнопка мультиплеер, работает список игроков…»;
  «По образцу rom2 делаем rom1»; character files live in the player profile,
  one folder per game, never in the install, in each original's format (DIV-5194).
- R2-SESSION-022 (High for container, scramble, checksum; Unknown for field meanings): the `.a2c` framing and section A.
- R2-ENGINE-1840 (High for the list body; Medium that the last entry means a new character; Unknown the A +8 filter
  value and the last entry's text): the second game's list.
- MENU-168 (High for the body; Medium for the wildcard text and the folder; Unknown the +0x6c filter value) and
  HERO-213 (High for the `AlCh` order and sizes; Medium for the name text): the first game's list and file.
- MENU-ROUTE-163 (High): the first game's empty page shows one row New Character, and Delete.
- R2-ENGINE-1844 (Unknown for the panel controls): the second game's page shows no Delete.
- DIV-5192 filter, DIV-5193 last row, DIV-5195 order, DIV-2369 what the pages still lack.

## Rules

1. Each game lists the character files of its own profile folder by name, newest first.
2. A file that fails the magic or tag, the size or the checksum is left out and counted under the list.
3. The list ends with the New Character row; Accept on it, or on any row, stays on the page.
4. In the first game Delete asks the Load page's question and removes only the chosen file; Cancel keeps it.
5. The second game's page has no Delete, by button or key.
6. Opening or listing writes nothing and creates no folder; the install is not read for characters.
7. The Load page lists and deletes saves as before.

## Code

| kind | its one builder | change |
|---|---|---|
| character file decoder | `pkg/formats/a2c` ReadIdentity, `pkg/formats/alch` ReadHeader | new, one per format |
| character store | `pkg/game/characterstore.go` characterStoreRow, listCharacters | new; one row per game |
| game profile row | `pkg/game/gamefiles.go` gameFiles | the store row of each game |
| player profile folders | `pkg/game/runtimeprofile.go` runtimeProfileStores | the character folder |
| list page | `pkg/ui/savepage.go` CharacterPage, `pkg/ui/save.go` openListPage | store rows and the New row |
| file delete | `pkg/game/load_window.go` prepareFileDelete, listedFilePath; `pkg/ui/load_window.go` deleteSeams | the Load path serves a character file |
| launch wiring | `cmd/againrom/main.go`, `pkg/game/characterstore.go` ConfigureCharacters | the profile's store reaches the page |

## Proof

| test | rules |
|---|---|
| `TestReadIdentityDecodesAPageFile`, `TestReadIdentityRefusesBadMagicChecksumAndSize` | 1, 2 |
| `TestReadHeaderDecodesAPageFile`, `TestReadHeaderRefusesBadTagAndShortFile` | 1, 2 |
| `TestCharacterStoreListsEachGamesFolder` | 1, 2, 4, 5, 6 |
| `TestCharacterStoreAbsentFolderListsNothing`, `TestRuntimeProfileNamesTheCharacterFolder` | 1, 6 |
| `TestCharacterPageListsTheStoreAndDeletesInTheFirstGame` | 1, 3, 4, 5, 7 |
| `TestMultiplayerOpensTheCharacterListInBothGames`, `TestCharacterPageIsTheGamesRow` | 3, 7 |
| `TestReleaseMultiplayerOpensTheCharacterList` | 3, 7 |
| `TestReleaseCharacterStoreDecodesTheInstalledCharacterFiles` | 1, 2, 6 |

Release tests at the release: `TestReleaseCharacterStoreDecodesTheInstalledCharacterFiles`,
`TestReleaseMultiplayerOpensTheCharacterList`.

Base failures, witness counts and gate verdicts: review/story1771-mp-character-store/RESULT.md.

## Open debt

- The filter values app +0x3d8 and frame+0x3b8, the last row's text and the order's direction wait for an owner run
  (DIV-5192, DIV-5193, DIV-5195).
- Both installed `.a2c` names hold a `|` (Man|Man, Major|Home); whether the original shows the whole string is Unknown.
- The Delete question names a saved game; the original page's Delete was not pressed (DIV-2369).
- Creation from the New row, the writers, Rename and the portrait wait for the next multiplayer stories (DIV-2369).
