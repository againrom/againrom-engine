# A multiplayer map is entered with the stored character

## Result

In both games accepting a character in the multiplayer character list opens the multiplayer map choice of that game.
Cancel returns to the list with the same character selected. Ok starts the chosen map with that character alone as the
party: its name and statistics, the money its file holds, and in the second game its sex, class and experience.
The map is played on this computer only; it saves nothing and leaves the character file and the campaign saves
as they were. The campaign's own new game opens as before.

## Authority

- Owner (DIV-2369): multiplayer milestone 1 is a working character list, generator, map choice and map entry.
- Owner (DIV-5196): no connection step; the character list goes straight to the map choice.
- `R2-ENGINE-1843`: a join places the party at the map's drop list. DIV-5246: the first game takes the same rule.
- `R2-SESSION-514`, `R2-SESSION-517`, `HERO-213`: what each file holds. DIV-5247: what the entry takes from it.
- `R2-ENGINE-1842`: the join's strength gate. DIV-5248: not applied.
- `R2-ENGINE-1841`: the server writes the hero back. DIV-5249: the entered map saves and writes back nothing.

## Rules

1. Accept on a character row opens the game's multiplayer map choice; Accept on New Character opens the generator.
2. Cancel or Escape in the map choice shows the character list again with that character's row selected.
3. Ok in the map choice opens the chosen loose map with the stored character as the one party member.
4. The party has the file's name, statistics and money; in the second game also its sex, class and experience.
5. The same character entering the same map twice starts equal worlds.
6. The entered map is no campaign mission: SAVE is refused and the character file is not written.
7. A map outside the choice, a name outside the store and a statistic over 255 are refused.
8. The campaign's new game starts the same world as before.

## Code

| kind | its one builder | change |
|---|---|---|
| list page | `pkg/ui/listrole.go` characterListRole | a character row opens the map list role over the store's maps |
| map choice | `pkg/ui` mapListRole, `pkg/game/mapchoice.go` MultiplayerMapChoice | the store's Maps seam |
| character store | `pkg/game/characterstore.go` characterStoreRow | each row's whole-file read; one bounded file reader |
| party value | `pkg/game/chargen.go` generatorParty | fed a generator result read from the file |
| session | `pkg/game/mapsession.go` mapSession | new: character, party, purse and map |
| map opener | `pkg/game/frontend.go` requestOpener, `pkg/game/difficulty.go` prepareFreshGame | a request may carry a session |
| map entry | `pkg/game/missionentry.go` enterMission | a session's request admits its loose map, hires and advances nothing |
| map entered | `pkg/ui/listrole.go` enterOpened | the Load page and the map choice enter an opened map through it |

## Proof

| test | rules |
|---|---|
| `TestCharacterRowEntersAMultiplayerMapWithTheStoredCharacter` (both games) | 1, 2, 3, 4, 6 |
| `TestMultiplayerMapEntryIsDeterministic` (both games) | 5 |
| `TestMapSessionRefusals` | 7 |
| `TestCampaignNewGameWorldIsUnchanged` (both games, base hashes) | 8 |
| `TestNewCharacterIsGeneratedStoredAndListed`, `TestNewCharacterRowRunsTheStoresGenerator` | 1 |

Release tests at the release: none.

## Open debt

- MP-7: the session's end, the character write-back and a second player (DIV-5249, DIV-2369).
- MP-7: the stored items (sections D and F) and the join's own construction (DIV-5247); the strength gate (DIV-5248).
