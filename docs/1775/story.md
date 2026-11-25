# The second game follows the answered fidelity checks

## Result

Second-game shops price armour, shields and weapons with magic effects as the original does, and a price effect
sets the price. A negative protection now raises spell damage and one above 100 stops it whole. A mage's spell
kill trains the skill the original picks, and a weapon's spell other than Fire Ball needs the target to stay
above zero health.

Corpses reach each decay stage strictly below its health threshold, a hero's body stops at -61 health, and in
the Russian version a dead unit's items are lost on a cell whose sack already holds 1024 items. The other checks
match the engine or keep it under a named divergence row.

## Authority

- Owner: «И для второй игры перенос EXP-2208 в движок».
- R2-ENGINE-1824 (High expression, Medium prices), amending R2-ENGINE-1764 (High bodies): the tag 1 replace,
  the armour and shield cast's level word in S, and trunc((1.5^(S/70)+1) x S x 50).
- R2-ENGINE-1825 (High order and strictness; Medium hero classes); R2-ENGINE-1826 (High bodies, Medium credit).
- R2-ENGINE-1828 (High order and merge, Medium links); R2-ENGINE-1829 and R2-ENGINE-1830 (High order, Medium
  for other arms); R2-ENGINE-1831 (High order); R2-ENGINE-1766, 1827, 1156 and 1157 match.

## Rules

1. When second-game equipment is priced, ordinary effects and an armour's or shield's cast sum to one size S.
2. When second-game equipment holds a tag 1 effect, its price is that effect's value.
3. When a second-game spell or blow meets a protection word, the word is unbounded and the damage is at least 0.
4. When a second-game mage's spell kills, the kill reads the damage kind as the spell, as the original.
5. When a second-game corpse ages, each stage begins strictly below -10, -20 and -40.
6. When a second-game body of class 0x21..0x24 ages, it stops at -61; the Russian build raises one below to -61.
7. When a first-game corpse ages, its stages begin at -10, -20 and -40, as before.
8. When a Russian second-game death bag meets a sack of 1024 entries or more, its items are lost, its gold kept.
9. When a second-game weapon Spell other than Fire Ball rides a blow, it fires only on a target left above 0.
10. When a second-game well is used, one use is spent whatever the unit's health, mana or class.
11. When Prismatic Spray, a walk, a kill award or a Human speed runs in the second game, it is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| equipment price | `pkg/secondshop/shop.go` Price, `fill.go` effectsPrice | 1.5^(S/70); armour cast in S; tag 1 replaces |
| spell damage roll | `pkg/sim/spellrow.go` spellRows, `spell.go` secondSpellDamage | the protection word unbounded |
| blow resolver | `pkg/sim/combat.go` resolveBlow, blowRule | second-game row: signedProtection |
| kill credit tail | `pkg/sim/itemattribution.go` tailSpell | spell row data: tailSpell |
| corpse ladder | `pkg/sim/world.go` decayLadder, `gameprofile.go` decay | second-game row: strict, heroFloor |
| corpse walk | `pkg/sim/step.go` decayWalks, nativeDecayStep | hero floor; the build's raise |
| build rules | `pkg/base/base.go` Profile, `pkg/sim/buildrules.go` BuildRules | HeroBodyRaise, DeathSackCap for RU |
| death sack pour | `pkg/sim/sack.go` pourSack, deathSackRefuses | the four death pours ask the bound |

## Proof

| test | rules |
|---|---|
| `TestSecondShopPriceAddsOneSurchargeOnTheSummedSize`, `TestSecondShopArmourCastEntersTheSummedSize` | 1 |
| `TestSecondShopTagOneEffectReplacesTheEquipmentPrice` | 2 |
| `TestSecondSpellDamageReadsTheProtectionWordUnbounded`, `TestSecondBlowMagicPartReadsTheProtectionSigned` | 3 |
| `TestSecondGameSpellTailKeepsTheDamageKind` | 4 |
| `TestSecondGameCorpseStagesBeginBelowTheirThresholds` | 5, 7 |
| `TestSecondGameHeroBodyStopsAtMinusSixtyOne`, `TestRussianBuildRaisesAHeroBodyToMinusSixtyOne` | 6 |
| `TestHeroBodyRaiseSurvivesASaveRoundTrip`, `TestCorpseRulesBelongToTheRussianSecondGameBuild` | 6, 8 |
| `TestDeathBagMergesNothingIntoASackOfTheBuildsBound` | 8 |
| `TestSecondGameRiderNeedsTheTargetAboveZeroAfterTheBlow` | 9 |
| `TestSecondGameWellSpendsAUseWithoutABenefit` | 10 |
| `TestPrismaticSideTargetsStayWithinTheRangeBound`, `TestSecondGameKillsOfOneKindPayFullTwiceThenTheCut` | 11 |
| `TestHumanSpeedAppliesTheOverloadBeforeTheModifier`, `TestABlockedListedStepIsSearchedAgainOnlyPastTheDynamicCounter` | 11 |

Release tests at the release: none. Evidence: the lane's review report.

## Open debt

- DIV-5208, 5209, 5210, 5211, 5212, 5213, 5214, 5215: one Unknown each, with the original run that settles it.
- DIV-4946 (purse joins the death gold) and DIV-442 (no Fire Ball after a blow to -10 or below) wait for the owner.
