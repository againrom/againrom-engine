# Every list page role is a data row of the one list page

## Result

Nothing the player sees changes. The Load page, the character list and the multiplayer map choice look and behave as
before in both games and write the same files. Each of them is now one row of the list page's role table, so a later
page of the same kind is a new row and not new branches. The save list and the character list read their folders
through one lister.

## Authority

- Owner: «Как рекомендуешь» to «roles of the list page as data, before MP-3».
- Architecture audit at `7dfe5ea99`, row 3 of the widened kinds: the role chose bodies by a flag in seven places.

## Rules

1. The Load page, the character list and the map choice draw the same page and take the same input in both games.
2. The map choice deletes no row, by key or button.
3. A list page role is one row: page, rows, Ok, answer on close and delete seams; no page function asks the role.
4. The saves and the characters are two rows of one folder lister: folder, extensions, file reader and order.
5. Saves and character files list as before: the same files, order and skipped count; the listing writes nothing.

## Code

| kind | its one builder | change |
|---|---|---|
| list page | `pkg/ui/savepage.go` loadPage, `pkg/ui/save.go` openListPage, `stepPage` | reads the open role's row |
| list page role | `pkg/ui/listrole.go` listRole | new row table: load, characters, maps |
| profile file lister | `pkg/game/profilefiles.go` listProfileFiles | saveScan and characterStoreRow are its rows |
| list page words | `LoadWindowWords`, `CharacterPage`, `MapChoiceWords` | unchanged |

## Proof

| test | rules |
|---|---|
| `TestListPageOpensARoleRow` | 3 |
| `TestMapChoiceDeletesNoRow` | 2 |
| `TestListPageRolesAreRows`, `TestListPageRoleScanNamesEachBranch` | 3, 4 |
| `TestProfileListerListsTheRowsPattern` | 4, 5 |
| `TestSharedScannerKeepsEveryCallersRows`, `TestCharacterStoreListsEachGamesFolder` | 5 |
| `TestMapChoiceSelectOkAndCancel`, `TestMapChoiceDrawsEachCaptionAtItsOwnButton` | 1 |
| `TestCharacterPageIsTheGamesData`, `TestCharacterPageListsTheStoreAndDeletesInTheFirstGame` | 1 |
| `TestMultiplayerOpensTheCharacterListInBothGames` | 1 |

Release tests at the release: `TestReleaseMultiplayerOpensTheCharacterList`,
`TestReleaseCharacterStoreDecodesTheInstalledCharacterFiles`.

## Open debt

- The save page keeps its own step (`stepSaveScreen`) and branches on its save state; it is not a row yet.
- The save list does not report files it leaves out; the character list does (audit row 3, Medium).
