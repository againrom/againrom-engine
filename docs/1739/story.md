# Second-game area spells checked against their published cell slots and rates

## Result

The second game's area spells strike what the research now states, and tests hold that. Blizzard strikes one cell of
its six by six window per pulse, 32 pulses, and hits a walker, a flier and a building on that cell alike. Light and
Darkness, Wall of Fire and Poison Cloud reach only walkers and ghosts, and Light and Darkness keep the sight lowered
between pulses (owner). English Acid Stream pulse 2 now strikes x = -1 twice and never -2 (R2-ENGINE-797, High).

## Authority

- Owner: adopt the research on area and cloud spells (triage E3b); R2-ENGINE-1530 to R2-ENGINE-1537.
- Owner, DIV-4834 and DIV-4835: «Тьма обязана снижать обзор».
- R2-ENGINE-1534: the original's Effect ends one update before each pulse; DIV-4835 keeps the owner's hold.

## Rules

1. A Blizzard pulse strikes the ground, air and structure occupants of its one cell, caster and allies included.
2. Blizzard draws its cell from the window -2..+3 on each axis and pulses 32 times, three updates apart.
3. A Light, Darkness or Wall of Fire pulse reaches ground-slot occupants only: walkers and ghosts, no flier or building.
4. A cloud pulses AreaDuration + ceil(floor(16P/10)/16) times, sixteen updates apart, in both games.
5. The shapes hold 21 cells (radius 2), 57 (radius 4), 10 or 9 (wall), 9 (Fire Ball), 25 or 21 (Acid Stream).
6. Light and Darkness hold the lowered sight at every tick of the cloud in both games (owner, DIV-4835).
7. A straight Acid Stream's pulse 2 strikes x = -1,-1,0,1,2 in the English build, -2..2 in the Russian (R2-ENGINE-797).

## Code

| kind | its one builder | change |
|---|---|---|
| area spell cell poll | `applyAreaCells` in `pkg/sim/celleffect.go` | none; clouds read the ground slot, others all three |
| occupant slot of a unit | `domainForCode` in `pkg/mapload/spawn.go` | none; movementType 1 and 2 ground, 3 air |
| spell rate | `tickCellEffect` and `meteorStormStage` | none |
| Light and Darkness reach and sight return | `spellPointDurationUnder`, `areaArms` | none; owner hold kept |
| Acid Stream straight cone | `acidStreamStage` in `pkg/sim/areaarms.go` | rows read from the build's profile (`base.Profile.StraightCone`) through the table |

## Proof

| test | rules |
|---|---|
| `TestSecondBlizzardStrikesTheGroundAirAndStructureSlotsTogether` | 1 |
| `TestSecondBlizzardDamagesEveryActorItStrikes` | 1 |
| `TestSecondBlizzardPulseScheduleAndCellDraws` | 2 |
| `TestLightReachesOnlyTheGroundSlotInBothGames` | 3 |
| `TestWallOfFireBurnsOnlyTheGroundSlotOnItsPulseCount` | 3, 4 |
| `TestSecondAreaShapesHoldThePublishedCellCounts` | 5 |
| `TestALightOrDarknessCloudHoldsTheSightBetweenPulses` | 6 |
| `TestAcidStreamStraightPulseTwoFollowsTheBoundRows` | 7 |
| `TestStraightConeRowsBelongToTheEnglishSecondGameBuild` | 7 |

Release tests at the release: none.

## Open debt

- The update rate outside a debugger, the RU placement body and the EN actor resolver's rejection bit are unread.
- A unit of s by s under a cloud takes one application per covered cell (Medium); Fire Ball's size byte is unread.
