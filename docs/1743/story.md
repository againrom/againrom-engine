# Second-game Humans take their class thresholds at creation, the player's included

## Result

In the second game a player's Human now takes the class Withdraw and Wimpy at mission start, as a computer Human
does: a mage HPmax and HPmax/4, a ranged Human that does not fly HPmax/2 and 0, any other Human 0 and 0. The
player's Retreat setting still rewrites them by class when it is applied. The creature card's damage digits already
equal the published decode of the row's two bytes (Troll.5 480..735, Spider.5 45..2010; Medium). The first game is
unchanged.

## Authority

- Owner: adopt the research on combat and Human statistics (triage E2); R2-ENGINE-1605 to R2-ENGINE-1607, R2-ASSET-343.
- R2-ENGINE-1606 (High for the table and the one RU call site; Medium that a Human reaches it): the setter is the
  template loader's last call and tests only the Human kind, not the owner.

## Rules

1. Every second-game Human, a player's Human included, takes the class thresholds at mission start (DIV-4906).
2. A second-game creature's damage pair, and so its card, is the two-byte decode of its row (DIV-4913).

## Code

| kind | its one builder | change |
|---|---|---|
| class-threshold setter | `classThresholds` in `pkg/sim/withdraw.go` | the player-owner exclusion is gone |
| creature card damage | `ByteStoredDamage` in `pkg/data/unitdef.go`, read by `valuesOf` and the unit panel | none |
| strike (blow refusal) | `resolveBlow` in `pkg/sim/combat.go` | none; reach refusal matches, zero-maximum refusal kept |
| hero drawing layers | the hero appearance and paperdoll compositor | none; no new claim |
| skill award | `awardSkill` through `skillAwardProfile` in `pkg/sim/skill.go` | none; no new claim |

## Proof

| test | rules |
|---|---|
| `TestSecondPlayerHumansTakeTheClassThresholds` | 1 |
| `TestSecondComputerHumansTakeTheClassThresholds` | 1 |
| `TestASecondGameCreatureDealsTheDamageItsRowBytesHold` | 2 |

Release tests at the release: none.

## Open debt

- DIV-4912: the RU strike's refusal of a non-mage Human against movementType 3 is not built; a per-build value can
  carry it.
- DIV-4843: the zero-maximum refusal stays; the award's division by a zero maximum is unread.
- DIV-4274, DIV-4275, DIV-4840, DIV-4841, DIV-4086 stay Unknown; no new claim.
