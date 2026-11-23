# Second-game mana after a cast, spell flight time, and towers checked against the original

## Result

In the second game a mage casting from the book or a scroll gets his tripled mana restoration back 81 ticks
after he has turned to the target and begun, as the original does, not after his wind-up and recovery end.
Flying spells take the straight distance to the target, so a diagonal shot lands later; Lightning and
Prismatic Spray keep their fixed wait. The towers of mission 72 keep their reach, one cast every 16 ticks,
free casting and removal with the building, now confirmed by research. The first game plays as before.

## Authority

- Owner: «продожаем по списку» on the epic «ROM2: полная игра», triage E3d.
- `R2-ENGINE-1565`: the second game's cast stores the bare session counter as the activity stamp.
- `R2-ENGINE-543`, `R2-ENGINE-926`: that cast routine runs once the caster faces the target.
- `R2-ENGINE-1567`, `R2-ENGINE-1568`: a flying spell waits its Euclidean distance over its speed.
- `R2-ENGINE-1262`: Lightning and Prismatic Spray set that wait to 10.
- `R2-ENGINE-1560`, `R2-ENGINE-1562`, `R2-ENGINE-1563`: a tower's reach, schedule and removal.
- `R2-ENGINE-1564`: the mana restoration effect reaches only a mage.
- `DIV-4918` closed; `DIV-4718`, `DIV-4977` narrowed; `DIV-5081` written for Prismatic Spray's stamp.

## Rules

1. When a second-game mage casts from the book or a scroll, tripled restoration returns 81 ticks after he faces the aim.
2. When a second-game flying spell is cast, it waits the straight distance over its speed; Lightning waits 10.
3. When a tower looks for a target, it takes units within its record's radius on both axes, at any spell range.
4. When an item's mana restoration is worn, only a unit with a mana pool gains it.
5. When a first-game unit casts or a first-game spell flies, the result is as before.

## Code

| kind | its one builder | change |
|---|---|---|
| cast activity stamp | `pkg/sim/activity.go` startSpellAction | rule row `castStamps`; the second game stamps the payment or wind-up start |
| cast delay by distance | `pkg/sim/spelldelivery.go` spellDeliveryRemaining | rule row `flightsBySpeed`; every second-game flight row |
| game profile | `pkg/sim/gameprofile.go` newGameProfile | `castStamp`, `flight` set for the second game |
| building caster | `pkg/sim/buildingcaster.go` buildingCasterAct, removeBuildingCastersOn | unchanged; pinned |
| mana restoration effect | `pkg/sim/effect.go` effectLanding | unchanged; pinned |

## Proof

| test | rules |
|---|---|
| `TestCastStampByGame` | 1, 5 |
| `TestFlightWaitByGame` | 2, 5 |
| `TestBuildingCasterReachIsTheRecordSquare`, `TestBuildingCasterRunsOnceOnTheTurnPhase` | 3 |
| `TestBuildingCasterCostsNoMana`, `TestBuildingCasterRemovedThroughBothHitPaths` | 3 |
| `TestManaRestorationEffectReachesOnlyAMage` | 4 |

Release tests at the release: none.

## Open debt

- `DIV-4718`: the built cast unit's wind-up, power and mana, and removal by script or other spells, wait a run.
- `DIV-4977`: the second game's strike term and spell-weapon conversion wait a claim.
- `DIV-5081`: Prismatic Spray's stamp waits a claim reading its cast path.
