# Summon and Animate Dead reuse the fallen in the second game

## Result

In the second game a Summon or Animate Dead now brings back a unit that has finished dying
instead of always making a new one: the creature takes the place of the oldest finished unit.
Animate Dead takes the corpse that fell first among the corpses of the chosen kind on the target
cell, not the one with the lowest number. When Animate Dead brings the corpse itself back, the
creature has the plain statistics of its kind. The first game plays exactly as before.

## Authority

- R2-ENGINE-455: before building a unit, Summon walks the global unit list for a dead unit and reuses it.
- R2-ENGINE-780: Animate Dead tests kinds 2, 3, 4, each over the dead list head to tail; typeID 82..98 excluded.
- R2-ENGINE-781 (Medium for the pool): the pool can return the selected corpse; the copies then read the template.
- R2-ENGINE-782, R2-ENGINE-531: radius 0 at the clamped corpse cell; a failed placement keeps the corpse consumed.
- Closes DIV-4152, DIV-4429 and DIV-4430.

## Rules

1. In the second game a body joins the dead list when its teardown comes; an earlier teardown stands first.
2. Animate Dead takes, by kind 2, then 3, then 4, the first body of the dead list on the target cell.
3. A Summon or raised creature first takes the dead pool's unit, the lowest id among finished bodies, and keeps its id.
4. When the pool returns the consumed corpse itself, the raised creature has its template's statistics.
5. When the pool returns an older unit, the raised creature copies the consumed corpse, which stays at kind 5.
6. A failed placement builds nothing and leaves the taken unit gone; the consumed corpse stays at kind 5.
7. The dead list order enters the World byte form and hash and survives SAVE and a cold LOAD.
8. In the first game no body is ranked, no unit is reused, and every recorded World hash is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| unit construction | `createActor` | a profile's reused id replaces the next id; the actor goes in at its id's place |
| dead pool | `poolCreation`, `pooledDead` | the one source of a reused id, read by Summon and Animate Dead |
| corpse list | `corpseList`, `joinDeadList` | bodies ranked at teardown; `controlSpiritCorpseAt` walks the list |
| placement | `secondPlacement` | unchanged, radius 0 at the corpse cell |
| game branch | `gameProfile.deadList` | second-game profile data; the first game ranks nothing |
| World byte form | `formWrappers` | the dead list order, written when it is not id order |
| SAV dead-actor roots | `rootDeadActors`, `importSavedDeadList` | bodies written and read back in dead list order |

## Proof

| test | rules |
|---|---|
| `TestSecondAnimateTakesTheFirstCorpseOfTheDeadList` | 1, 2, 4 |
| `TestSecondAnimateReusesTheOlderPooledUnit` | 3, 5 |
| `TestSecondAnimateFailedPlacementLeavesTheTakenUnitGone` | 6 |
| `TestSecondAnimateRaisesEachCorpseKind` | 5 |
| `TestSecondSummonReusesTheIdOfAFinishedUnit` | 3 |
| `TestSecondSummonTakesThePoolInListOrder` | 3 |
| `TestSecondSummonFailedPlacementLeavesTheTakenUnitGone` | 6 |
| `TestSecondDeadRankByteForm` | 1, 7 |
| `TestCurrentSaveKeepsAnimatedCorpseAcrossColdLoad` | 3, 5 |
| `TestSecondGameVictimListCrossesSaveAndColdLoad` | 7 |
| `TestFirstGameTeardownAndRaiseKeepTheirRecordedWorld` | 8 |
| `TestFirstControlSpiritKeepsRecordedWorld` | 8 |

Release tests at the release, on EN and RU: `TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD`,
`TestReleaseConsumedCorpseSAVLoad`.

## Open debt

- DIV-4431, DIV-4432: the raised creature's lifetime and constructor tail wait on research.
