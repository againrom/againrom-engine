# A multiplayer character keeps its gains for the next map

## Result

In both games leaving a multiplayer map through the game menu's exit, or taking its win, returns to the character
list with that character's row selected. The character file is rewritten there: in the second game E's statistics,
money and five skill experiences over the loaded document, every other section kept; in the first game the short
file's four statistics. The next map entered with that character in the same run starts with the party the first map
left, items included. Campaign play and saves do not change; a session map still saves no SAV.

## Authority

- Owner: «Создать персонажа мультиплеера, пройти карту мультиплеера в одиночку на сервере движка, прокачаться и
  продолжить с тем же персонажем»; MP-9: no connection step.
- `R2-ENGINE-1841`: the server writes the hero back from nine call sites; which event each serves is Unknown besides
  unit death. `HERO-214`: the first game exports from six sites; how often a map writes is Unknown. DIV-5258.
- `R2-SESSION-517`: E's fields. `R2-SESSION-516`: D and F record framing; the item word's engine meaning is unread.
  `HERO-213`: the short file holds the name and four statistics. DIV-5259.
- DIV-5260: the end returns to the character list; the original's next screen is unread.

## Rules

1. The game menu's exit from a session map shows the character list with the character's row selected.
2. At that end the file holds the live character's statistics, and in the second game its purse and skill experience.
3. Every field the store does not write keeps the loaded document's value.
4. A second end of an unchanged character writes identical bytes.
5. The next map of the same run starts with the carried party: experience, purse and items.
6. A rewritten second-game file alone gives a new run the experience and purse it holds.
7. The same route twice gives equal files and equal world hashes.

## Code

| kind | its one builder | change |
|---|---|---|
| character store and file writer | `pkg/game/characterstore.go` characterStoreRow.record, writeCharacter | record writes over a loaded document; writeCharacter replaces a listed file |
| hero carry | `pkg/game/citymissionobjects.go` carriedParty | named from currentMissionCityParty's first line; the session end calls it |
| session | `pkg/game/mapsession.go` mapSession | file, end and advance; the in-run carry by file name |
| party value | `pkg/game/mapsession.go` storedParty, carriedParty | the next entry takes the carry when the file holds the bytes it wrote |
| list page | `pkg/ui/listrole.go` chooseCharacterMap | the end shows the same list role again |
| map opener | `pkg/game/missionentry.go` enterMission | a session's advance is the session's |
| notice destinations | `pkg/ui/notice.go`, `pkg/ui/flow.go` goNoticeDest | NoticeLeave and NoticeToCharacters |

## Proof

| test | rules |
|---|---|
| `TestMultiplayerMapEndWritesTheGainsForTheNextMap` (both games) | 1, 2, 3, 4, 5 |
| `TestRewrittenSecondCharacterFileGivesTheGains` | 6 |
| `TestMultiplayerMapEndIsDeterministic` (both games) | 7 |

Release tests at the release: none.

## Open debt

- Items in D and F are not written (DIV-5259); first-game experience and purse live only in the run.
- Death, respawn and the second player (MP-8, DIV-2369); SAV on a session map (DIV-5249).
