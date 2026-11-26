# A fallen multiplayer hero returns on Space

## Result

On a multiplayer map in either game, a hero's death no longer ends the map. The map goes on, the message line says the
hero has fallen and that Space returns him, and he waits until Space. Space brings him back at the cell of his last Shop
use on the map, else at the map's drop cell, or at the free cell nearest it. He returns with nine tenths of each skill's
experience, full health and mana, and nothing in his hands: what he wore and carried lies in a sack where he fell. A
death writes no character file; leaving the map after the return writes the returned hero. Campaign maps are unchanged.

## Authority

- Owner: BACKLOG MP-8, in multiplayer a death is not a loss; the hero returns with a partial experience loss.
- Owner: MP-7, a dead character does not change the file.
- `R2-ENGINE-1908`: Space sends the return request. `R2-ENGINE-1865`, `R2-ENGINE-1843`, `R2-ENGINE-1912`: the cell.
- `R2-ENGINE-1866`, `R2-ENGINE-1909`: nine tenths of five skill experience values, then health and mana refilled.
- `R2-ENGINE-1868`, `R2-ENGINE-1841`: the original writes the file at death (DIV-5297 keeps MP-7).
- DIV-5292 to DIV-5298: the wait, the screen, the cell, the penalty, the sack, the file and the first game.

## Rules

1. On a session map a fallen hero does not lose the map; he waits until Space, and with no Space he stays fallen.
2. Space returns him at the free cell nearest his last Shop use cell, else nearest the map's drop cell.
3. He returns with nine tenths of each of the five skill experience values and full health and mana.
4. He returns holding nothing; what he wore and carried lies in a sack on the death cell.
5. A death writes no character file; a map left with the hero fallen leaves it as it was.
6. The first game takes the second game's rule; a campaign hero's fall loses the mission as before.

## Code

| kind | its one builder | change |
|---|---|---|
| entry fill | `pkg/game/missionentry.go` mapEntry | sessionEntry sets heroReturn |
| defeat test | `pkg/game/world.go` guardedCharacterLost | with heroReturn, only a hero the world lost |
| hero placement | `pkg/sim/createactor.go` createActor, nearestFreeCell | keepsID reuses id 0 |
| command | `pkg/sim/command.go` ReturnHero | new kind and its constructor |
| held skill level | `pkg/sim/skill.go` skillAwardProfile.heldLevel | HeldSkillLevel and the return share it |
| Space key | `pkg/ui/app.go` Panels arm | asks the viewer's hero return sink first |
| character file | `pkg/game/mapsession.go` endSession, writeCharacter | no write while the hero is fallen |

## Proof

| test | rules |
|---|---|
| `TestSessionHeroReturnsOnSpaceAtTheDropCell` | 1, 2, 3, 4, 6 |
| `TestSessionHeroReturnsNearTheShopCell` | 2, 6 |
| `TestReturnedExperienceIsNineTenthsInThirtyTwoBits` | 3 |
| `TestSessionHeroReturnsAtFullHealthAndManaEachTime`, `TestFillReturnedPoolsFillsALivingHeroOnly` | 1, 3, 6 |
| `TestSpaceReturnsAWaitingHeroBeforeTogglingPanels` | 1 |
| `TestSessionDeathWritesNoCharacterFile` | 3, 5, 6 |
| `TestAFallenHeroLosesTheMissionOnlyWhenHeCanNoLongerBeHealed` | 6 |

Release tests at the release: none.

## Open debt

- Session game modes (arena keeps experience, modes 1 and 2 set skills to 70) wait for a modes item (DIV-5295).
- The second game's Shop and Inn on session maps wait for the map buildings story (DIV-5294).
- Monster return is out of scope.
