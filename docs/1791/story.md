# A multiplayer monster group comes back

## Result

On a multiplayer map in either game, a monster group comes back once every member is dead. The wait runs 60 seconds of
game time from the last member's decay stage 3. Each member returns on its placed cell, or on the free cell nearest it,
with the health and mana of its template. A group with one member alive stays down. Campaign maps keep their dead
monsters. A multiplayer map is not saved, so the new Outpost state needs no save path.

## Authority

- Owner: BACKLOG MP-8, a multiplayer death is not a loss; the monsters come back.
- `R2-ENGINE-1818`, `R2-ENGINE-1867`: the Outpost restore copies template position, health and mana, and positions at
  radius 0. `R2-ENGINE-1865`: the return tick order. Grades as in the divergence rows.
- DIV-5300 to DIV-5304: the delay, the cell, the template branch, the first game and the groups returned.

## Rules

1. A session monster group returns after its last member reaches decay stage 3 and the profile's delay has run.
2. A group with one member standing does not return.
3. A returned member stands on its placed cell, or on the nearest free cell when that cell is taken.
4. A returned member has the health and mana of its template.
5. The delay is one profile field, the same in both games (60 passes); the first game runs the same pass.
6. A single-player map builds no session Outpost and its dead monsters stay down.

## Code

| kind | its one builder | change |
|---|---|---|
| map session | `pkg/game/missionentry.go` enterMission | a session entry declares the session Outposts |
| monster group | `pkg/sim/outpost.go` DeclareOutposts | `OutpostMap.Session` runs it in either game |
| placement | `pkg/sim/outpost.go` outpostPlacement, nearestFreeCell | unchanged; on the session path only, the id 0 member keeps its id |
| template build | `pkg/sim/createactor.go` createActor | unchanged |
| death stage | `pkg/sim/outpost.go` outpostDue | unchanged; the first game runs the pass in its pool phase |
| profile data | `pkg/sim/gameprofile.go` sessionReturn | new field, 60 in both profiles |

## Proof

| test | rules |
|---|---|
| `TestSessionOutpostsReturnAGroupOnceEveryMemberIsDown` | 1, 3, 4, 5 |
| `TestSessionOutpostsKeepAGroupWithOneMemberAlive` | 2 |
| `TestSessionOutpostsReturnOnTheNearestFreeCellWhenTheCellIsTaken` | 3 |
| `TestWithoutTheSessionRuleADeadGroupStaysDown` | 6 |
| `TestCampaignOutpostGivesAnIDZeroMemberANewID` | 3, 6 |
| `TestSessionMonsterGroupReturnsAfterTheDelay` | 1, 3, 4, 5 |
| `TestSessionMonsterGroupStaysDownWhileAMemberStands` | 2 |
| `TestCampaignMonstersStayDown` | 6 |

Release tests at the release: none.

## Open debt

- Map buildings (shop, tavern quests, town hall) are the next story.
- A SAV of a multiplayer map would need the Outposts and their countdown; no save path is added.
