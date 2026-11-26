# A map entry is one request filled by the campaign or the session

## Result

Starting a mission, resuming a save or opening a multiplayer map plays as before in both games: same map, difficulty,
hires, advance notice and saved state. Inside, the map entry no longer asks whether a multiplayer session sent it. The
campaign or the session fills the request before entry, and the entry reads what it was given.

## Authority

- Owner: «Как рекомендуешь» on the architecture audit item 2, map entry as one request with no session identity test.

## Rules

1. A campaign mission and a session map both enter through the one entry body.
2. The entry body, the new-game opener and the request opener hold no test of a session pointer.
3. The map, difficulty, hire moment, loose-map decode and advance notice come from the request's fill.
4. A session map's won notice and leave end through the session's own end, not the campaign successor.
5. A campaign entry admits, hires, starts a guest and advances as before.

## Code

| kind | its one builder | change |
|---|---|---|
| map entry | `pkg/game/missionentry.go` enterMission | reads the fill; no session branch |
| entry fill | `pkg/game/missionentry.go` mapEntry | campaignEntry and sessionEntry fill it |
| guest seats | `pkg/game/frontend.go` campaignFill | the campaign's seats, run through the fill |
| difficulty choice | `pkg/game/difficulty.go` campaignDifficulty | unchanged; both fills call it |
| mission decode | `pkg/game` decodeMission | unchanged; the loose flag is fill data |

## Proof

| test | rules |
|---|---|
| `TestMapEntryReadsItsFillWhicheverFilledIt` | 1, 3 |
| `TestSessionEntryCommitsThroughItsOwnEnd` | 4 |
| `TestMapEntryFilesAskNoSessionIdentity` | 2 |
| `TestMissionEntryCommitsToTheSessionItIsGiven`, `TestMissionEntryDefersTheCommitToActivate` | 5 |

Release tests at the release: none.

## Open debt

- Both fills assign the live driver to the campaign session as before; a session-owned live driver waits for MP-7.
