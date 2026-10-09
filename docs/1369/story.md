# ROM2 spell table

## Intent

The ROM2 spell table holds 35 entries. Entry 30 has no params, so the loader
refused the whole table and every ROM2 map that names a spell stood blocked.
The story1368 census counted 41 such maps; maps 10 and 21 were blocked by
nothing else. This story loads the table on both ROM2 roots, resolves every
named spell to a rule, and lets a ROM2 unit cast in ordinary play. The ROM1
table, the ROM1 spell rules and ROM1 hashed state do not change.

Base: `e592b27e` (game 0.96.0). Knowledge pin: k204.

## Authority

| Part | Claims |
|---|---|
| table shape: 29 spells, five blank trailing entries | R2-ENGINE-019 |
| spellbook indexed by spell id from 1 | R2-ENGINE-017 |
| Distribution 5 forces mode 2 and duration 0 | R2-ENGINE-022 |
| apply arms grouped by id; shared protection, Shield, Haste, Slow, Bless and Curse magnitudes | R2-ENGINE-024 |
| Poison Cloud magnitude and duration | R2-ENGINE-024 |
| Stone Curse Effects string parses to no kind (Medium) | R2-ENGINE-025 |
| Heal and Drain Life draw U[1, spread]; Drain Life cut by Astral resistance; Summon level and kind roll | R2-ENGINE-026 |
| Ice Missile and Diamond Dust on the generic damage path | R2-ENGINE-024, R2-ENGINE-028 |

ROM1 claims apply only where R2-ENGINE-024 states the rule is shared. Where a
claim is Medium or silent the row runs the smallest rule and carries a DIV row.

## As built

- `data.LoadSpells` drops a trailing run of blank entries (empty name, no
  params). A blank entry before a written row is still refused.
- `sim.SpellRule` gains `Arm` and `Second`. `Arm` is the first-game arm a row
  runs: 0 means the row's own ID, `sim.ArmNone` means no singular arm. Neither
  field is in the binary spell record. Every sim site that compared a spell ID
  with a first-game ID now compares the row's arm. A first-game row keeps Arm
  0, so every converted comparison reads the same value as before.
- `pkg/sim/secondspell.go` holds the ROM2 id-to-arm table. Ice Missile (5),
  Blizzard (7), Diamond Dust (16) and Summon (25) have no arm.
  `mapload.SpellRules` assigns arms and reclassifies the damage pair by arm
  for a ROM2 table. The ROM2 binary state reader reassigns arms on load.
- Under `Second` four arms take their ROM2 formula: Heal adds U[1, spread]
  with a spread of 0 still drawing 1; Drain Life is (base + U[1, spread]) cut
  by the target's Astral resistance before the health cap; Poison Cloud
  scales its magnitude by (P + 45) / 45 and takes its duration from the
  Duration column; Invisibility lasts P << 4 at application and in the
  spell record the book shows (R2-ENGINE-022). Stone Curse with no effect
  kind lands and attaches nothing (DIV-2620).
- An attached effect keeps its row id. Every rule that reads an attached
  effect by spell asks for the row running the arm (`World.armSpellID`): a
  first-game table answers the arm itself, a second-game table the row, or
  no id when no row runs the arm. The swept sites are listed below.
- A creature casting a row with no arm aims by the row's columns: a unit
  target aims at the victim, an area at the victim's cell (DIV-2619).
- `sim.SpellRuleLands` refuses a staged area with no stage program. Blizzard
  is such a row (DIV-2617); the census reads the same predicate.
- `pkg/game` point targeting, self-cast and the Teleport fog check read the
  arm, as do the presentation reads of attached and area effects listed
  below.

## Swept effect lookups

`pkg/sim` and `pkg/game` were searched for every lookup of an attached
effect, area effect or cast event by a literal or first-game spell id
(`hasAttachedSpell`, `attachedMagnitude`, `removeAttachedSpell`,
`effectIndex`, `HasEffectSpell`, `.Spell ==`/`!=`, `switch` on a spell id,
and the named spell constants).

Changed to the arm:

| Site | Reads |
|---|---|
| `pkg/sim/effect.go:80` | Stone Curse action gate (movement, attack, cast) |
| `pkg/sim/effect.go:100`, `pkg/sim/sight.go:461`, `pkg/sim/sight.go:470`, `pkg/sim/engage.go:831` | Invisibility sight and acquisition |
| `pkg/sim/combat.go:385`, `pkg/sim/structurecombat.go:41`, `pkg/sim/spell.go:1136`, `pkg/sim/spelldelivery.go:187`, `pkg/sim/usescroll.go:307` | Invisibility cancellation on attack, book admission, direct cast and scroll |
| `pkg/sim/combat.go:840`, `pkg/sim/combat.go:842` | Bless and Curse in the physical roll |
| `pkg/sim/effect.go:409`, `pkg/sim/sourceeffect.go:30` | Bless and Curse annihilation |
| `pkg/sim/spellformula.go:63` | Invisibility duration, applied and recorded |
| `pkg/game/effectmark.go:57`, `pkg/game/effectmark.go:133` | effect marks; stone frame hold |
| `pkg/game/world.go:4954` | stone and translucent draws |
| `pkg/game/world.go:4510`, `pkg/game/world.go:4569` (via `cellEffectsByArm`) | Wall of Fire ambience; Light, Darkness and Wall of Fire lighting |
| `pkg/game/fog.go:29` | Teleport fog reveal |
| `pkg/game/spellbolt.go:172` | Heal and Drain Life bursts |
| `pkg/game/spellsound.go:191` | script Lightning bolt |

Left as they are:

- Fire Ball (`pkg/sim/celleffect.go:401`, `pkg/sim/savedcontinuation.go:417`,
  `pkg/sim/spelldelivery.go:54`, `pkg/game/spellbolt.go:167`): row 2 runs arm
  2 in both tables, and no other row runs arm 2.
- Wall of Fire legacy decode (`pkg/sim/castbinary.go:270`): a first-game save
  format.
- Second-game script layers (`pkg/sim/rom2script.go:156`,
  `pkg/sim/rom2script.go:187`): already keyed by second-game ids.
- Prismatic Spray rays (`pkg/sim/binary.go:1535`, `pkg/mapload/modspell.go`):
  set only by mods; mod spell checks stay under DIV-2622.
- Area overlay pictures (`pkg/game/spellbolt.go:711`), item spell text
  (`pkg/game/iteminfo.go:158`) and every picture lookup by id: DIV-2622.
- Potion effects use spell 0 and are not spell rows.

## Divergences

DIV-2616 (shared arms), DIV-2617 (Blizzard), DIV-2618 (Summon), DIV-2619
(creature aim), DIV-2620 (Stone Curse), DIV-2621 (Slow book slot), DIV-2622
(presentation keyed by first-game ids), all in `docs/divergences/rom2.md`.

## Proof

Focused tests:

- `pkg/data`: `TestLoadSpellsStopsBeforeTrailingBlankEntries`,
  `TestLoadSpellsRefusesABlankEntryBeforeAWrittenRow`.
- `pkg/sim`, attached-effect consumers: `TestSecondBlessLeavesItsTargetFreeToMove`,
  `TestSecondInvisibilityHidesItsTargetUntilItAttacks`,
  `TestSecondBlessAndCurseAnnihilate`,
  `TestSecondBlessAndCurseControlPhysicalDamage`,
  `TestSecondShieldDoesNotActAsCurse`,
  `TestSecondInvisibilityRecordDurationIsItsAppliedDuration`,
  `TestSecondArmsSurviveAColdBinaryLoad`,
  `TestFirstGameEffectLookupsKeepTheirIDs`.
- `pkg/sim`: `TestSecondGameArmsFollowThePublishedTable`,
  `TestSecondHealDrawsOneToSpread`, `TestSecondDrainLifeIsCutByAstralResistance`,
  `TestSecondPoisonCloudAndInvisibilityTakeTheirOwnFormulas`,
  `TestSecondStoneCurseWithNoEffectKindLandsAndAttachesNothing`,
  `TestSpellRuleLandsRefusesAStagedAreaWithNoProgram`,
  `TestCreatureAimOfARowWithNoArmFollowsItsColumns`.
- `pkg/mapload`: `TestSecondGameRulesClassifyTheDamagePairByArm`.

Install witnesses, each run on the EN and RU ROM2 roots:

| Test | Result on both roots |
|---|---|
| `TestReleaseSecondGameSpellTableLoads` | 35 collection entries, 29 rules, every row carries its arm |
| `TestReleaseSecondGameMissionTenMageHealsTheHero` | mission 10's mage casts Heal (row 24) on the hero through the ordinary engage; health 138 to 145; 5 mana paid |
| `TestReleaseSecondGameMissionTwentyOneMageCastsIceMissile` | mission 21's mage casts Ice Missile (row 5) at the attacking hero for 3 mana; health 145 to 123 |
| `TestReleaseSecondGameAttachedEffectsActAsTheirOwnArms` | an ordinary book cast of the installed Bless leaves its target free to move; the installed Invisibility hides its caster from an enemy; its record at P70 is 1120 ticks, the applied duration |
| `TestReleaseSecondMissionTenHealSaveContinuation` | after the mage casts, a named SAV and cold LOAD keep his mana, book and the table's arms; both worlds run 300 identical steps holding 13 further casts |

## Census movement

Instrument: `TestReleaseSecondGameSupportCensus` on the EN and RU ROM2 roots.
Both roots move alike.

| Total | before | after |
|---|---|---|
| maps blocked by `spell` | 41 | 5 |
| maps with no blocker | 1 | 3 (10, 20, 21) |

The five remaining maps (53, 87, 92, 102, 110) name Blizzard only. With unit
books now loading, the maps name 21 spell IDs; none names Summon or Slow. All
other census totals are unchanged.

## Open debt

- Blizzard's cell program is Unknown; the row is refused at landing.
- Summon's creature names and statistics are Unknown; the row is refused.
- Slow (29) has no book slot; the engine book holds ids 1 to 28.
- Stone Curse's random duration is not modelled; it lands and attaches
  nothing.
- Durations past the field width saturate rather than wrap.
- Bless and Curse keep the first-game 0..99 draw; `R2-ENGINE-028` (Medium)
  reads 0..100 (DIV-2616).
- Captions, bolt, burst and area pictures, the area overlay, item spell
  text, the Control Spirit corpse pick and mod spell checks still read
  first-game ids (DIV-2622).
