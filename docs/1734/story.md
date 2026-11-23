# Players of a second-game map in the map editor

## Result

On a second-game map the one Players window lists the players, edits each player's name, colour and its relation
toward every other player, and adds a player. The two roster words no claim reads (+4 and +8) and a player's own
relation stay read-only, with a reason line. Deleting a player stays off there.

## Authority

- Owner: the second-game map editor epic, and «Как рекомендуешь» on the player group.
- Owner: «игроки должны управляться только одним вот этим экраном!» (the Players window).
- Engine rule «One builder per kind»: the Players window and the player record, the second game as data.

## Rules

1. A second-game map lists every type-5 record in the Players window, as the first game does.
2. A name, a colour slot (1..15) and a relation (0, 1, 2) edit, save and reload; no other byte of the roster changes.
3. The words at +4 and +8 are read-only: +4 has one flag bit read and +8 is unread (R2-ENGINE-1291).
4. A player's relation toward itself is read-only: the loader forces that word to 0x12 (R2-ENGINE-1291).
5. A new player takes the next colour, 5000 at +8 (R2-ENGINE-1293), hostile relations, 0x12 toward itself; at most 16.
6. A second-game player is not deleted: the owner words that name it are not renumbered.
7. An unchanged second-game map saves byte for byte equal; the first game's Players window is as before.

## Code

| kind | its one builder | change |
|---|---|---|
| player record | `KindPlayer` row in `pkg/mapedit/player.go` | second row: fields, locked words, blank row, check |
| capability of a format | `editRows` in `pkg/game/mapeditrow.go` | `fieldKinds`, `playerRecords`, `structureOwner` |
| add kinds of a game | `secondEditorTables` | the player |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapPlayerEditSavesAndReloads` | 1, 2, 3, 4 |
| `TestReleaseSecondMapAddsPlayer` | 5, 6 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 7 |
| `TestEditRowsHoldBothFormats` | 7 |

## Open debt

- The meaning of a relation value (0, 1, 2 shown as neutral, hostile, ally) and of word +4 is not read.
- The colour range 1..15 rests on the type-5 census of one map and the 16 diplomacy slots.
