# The first game follows the answered fidelity checks

## Result

First-game shops and maps price an armour or a shield that carries a spell as the original does: the spell's
power joins the item's other magic and pays one surcharge, while a weapon's spell keeps its own price. Experience
from blows, kills and casts is scaled by Mind in the original's floating-point arithmetic, so an award that lands just
below a whole number is one point lower. The other checks match the engine, keep it under a named divergence row,
or wait for the owner.

## Authority

- Owner: «Yes: a story for the first game after 1775, of the same shape».
- SHOP-158 (High bodies, Medium shown price): an Armor or Shield sums a cast's power times Magic row 41 into N.
- HERO-209 (High), amending HERO-SKILLGATE-074: Mind scaling in binary64, truncated toward zero; no kill history.
- MAGIC-473 (High stores, Medium in play) with MAGIC-ITEMKILL-117 (High tails); MAGIC-476 (High arms, Medium fold).
- MAGIC-475 (High branches): matches but for DIV-442, an owner ruling.
- MAGIC-474 (High census) and MOVE-205 (High extractor, Medium in play): match.
- ITEM-256 (High order, Medium tail insert) and ITEM-257 (High arms, Medium mana cap): match but for two Unknowns.

## Rules

1. A first-game armour or shield counts spell power in its magic points and costs at most 9,999,999.
2. When a first-game weapon carries a spell, the spell is priced apart from the magic points, as before.
3. When a first-game blow, kill or cast pays experience, Mind scales it in binary64 and the result truncates.
4. When a first-game mage's kill byte holds a damage kind, the kill reads it as a spell id.
5. When a second-game item is priced or a second-game award is scaled, it is unchanged.
6. When Prismatic Spray ranks secondaries, strict minimum scans pick them after the primary, up to the cap.
7. When a walker takes the head of a near route, its next step follows a new near search.
8. When a unit dies holding items, the sack takes the pack, then the shield, weapon and worn slots, at its tail.
9. When a well with a charge is used, the charge is spent even at full health or without mana.

## Code

| kind | its one builder | change |
|---|---|---|
| item value | `pkg/mapload/itemeffects.go` storedItemPrice | armour and shield: cast power enters the points |
| shop price | `pkg/shopstock/enchant.go` Enchant.Apply | Base.CastInPoints from the candidate's item kind |
| kill award | `pkg/sim/skill.go` awardSkill, skillAwardProfile | first-game row: mind = mindBinary64 (mindScales) |

## Proof

| test | rules |
|---|---|
| `TestArmourAndShieldSumACastsPowerIntoThePoints`, `TestShopArmourCastPaysTheSurchargeOnItsPower` | 1, 2 |
| `TestShopArmourCastEntersThePointsAndAWeaponCastIsPricedApart`, `TestStoredItemPriceUsesThePublishedEffectControls` | 1, 2 |
| `TestFirstGameKillAwardScalesMindInBinary64`, `TestBinaryMindGainEqualsTheFloat64Expression` | 3 |
| `TestFirstGameKillsOfOneKindAreNotCut` | 3 |
| `TestFirstGameKillReadsTheDamageKindAsASpellId`, `TestSecondGameSpellTailKeepsTheDamageKind` | 4 |
| `TestArmourAndShieldCastPriceByEdition` | 1, 5 |
| `TestXPGainAgainstTheRealValuedForm`, `TestFR7MindScalesTheGainToTheseExactNumbers` | 5 |
| `TestPrismaticSprayTakesTheFoesNearestTheCasterUpToTheCap`, `TestPrismaticSprayCapFollowsPower` | 6 |
| `TestPrismaticSprayFallsBackToBodiesWhenTheGroupSeesNoLivingFoe`, `TestPrismaticSprayEqualDistancesFallToListOrder` | 6 |
| `TestAStoredRouteIsSearchedAgainAfterSixteenNearSearches`, `TestABlockedListedStepIsSearchedAgainOnlyPastTheDynamicCounter` | 7 |
| `TestTwoFullLoadoutsFellingOnOneCellConcatenateEachInFR2Order`, `TestADropOntoAStandingSackAppendsAtItsTailAndLeavesItsGoldAlone` | 8 |
| `TestStructureUse1150FountainPoolsAndCharges` | 9 |

Release tests at the release: none. Evidence: the lane's review report.

## Open debt

- DIV-5228 (the school a mage's spell or weapon kill trains) and DIV-5229 (a protection spell on a monster).
- HERO-207's strict decay stages wait for a torn-down record apart from the stage; the review report says why.
- DIV-442 (no Fire Ball after a blow to -10 or below) waits for the owner against MAGIC-475.
- DIV-5230 (a death drop's insert index) and DIV-5231 (a mana well's gift to a unit without mana).
- ITEM-256 against DIV-4946 (a unit's purse in its death sack) and ITEM-257 against DIV-4009 wait for the owner.
