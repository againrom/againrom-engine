# The first game's cast damage from the spell record

## Intent

A first-game cast deals the damage the original's arithmetic gives, and its
effect lasts the time it gives. Named result: KNOWN-DEFECTS B7, `DIV-2107`.
Base: public main `b61fc0a6`; the knowledge pin moves to k254.

## Authority

Claims at k254: `MAGIC-300` (the payload is the freshly scaled record pair),
`MAGIC-301` (byte narrowing, the unit resolver's draws, inclusive spread,
signed resistance with the 0.75 bias, the Fire Ball divisor, the building
rule), `MAGIC-302` (Fire Sacrifice, Poison Cloud, Heal and Drain Life arms),
`MAGIC-303` and `MAGIC-304` (lifetimes and the ordinary timer), `MAGIC-305`
(EN and RU share one executable and one Spells table), `TEXT-117` (the popup
may disagree with applied damage), `TEXT-096` (the record formulas). The
randomness ruling: the arithmetic that consumes a draw matches the original;
the draw comes from the World's one stream.

The second game's rows (`SpellRule.Second`) keep every path they had.

## As built

### Record pair

`SpellRecordDamage` (`pkg/sim/spell.go`): with f = power/30 + 1 in binary64 on
the power's low byte, the base is the low byte of trunc(dmin*f) and the spread
the low byte of trunc(dmax*f - base); a column at or below zero gives 0.
`spellDamageUnder` returns this pair for every first-game row without a
damage-factor mod table. Every consumer reads it: the unit resolver, the
building resolver, the Fire Ball divisor, the area clouds, the in-flight
payload SAVE writes, Heal, Drain Life and the weapon-spell projection. The
popup's `spellRecordDamage` (`pkg/game/spell.go`) calls the same function, so
the popup and the cast state one pair (`DIV-1882`, `TEXT-117`).

`pkg/sim` holds no floating point (`internal/archtest`). `pkg/sim/binary64.go`
is binary64 in integers: a 53-bit mantissa and an exponent, round to nearest
even on each operation, a 128-bit intermediate for products, quotients and
sums, truncation toward zero.

### Unit resolver

`directDamage` (`pkg/sim/spell.go`), for a first-game damage payload on a unit:

1. the physical step's spread draw on the payload's empty pair (an original
   stream draws nothing at zero, a seeded stream draws), the source's Bless
   roll R(100) and, unless Bless succeeded, its Curse roll, when attached; the
   accuracy roll R(200). None changes the elemental part;
2. a pair with base and spread both zero lands nothing and draws no spread;
3. otherwise v = base + R(spread), inclusive;
4. a nonzero school resistance, read as a signed word, gives
   (v*(100-r) + 75) / 100 (`ResistedSpellDamage`); the result lands at zero
   or above.

The building resolver (`applyStructureSpellAt`) and the Fire Ball divisor
(`MAGIC-FIREDIV-047`) were already the claims' rules; they now read the
record pair.

### Separate arms

| Arm | Before | Now | Claim |
|---|---|---|---|
| Heal 6 | base + R(spread) | base + 1 + R(spread-1); a zero spread still draws and heals base+1 | `MAGIC-302` |
| Drain Life 11 | base + R(spread) | base + 1 + R(spread-1), kept as a signed word, capped at target HP + 10 | `MAGIC-302` |
| Fire Sacrifice 4 | HP + mana clamped at 0, spread clamped at 0 | signed HP and mana words, base min(sum, 255), spread min(min(sum+power, 512) - base, 255), low bytes, no lower clamp | `MAGIC-302` |
| Poison Cloud 8 magnitude | magnitude*(power+30)/30 in integers | low word of trunc(magnitude*f) in binary64 on the full signed power | `MAGIC-302` |
| Poison Cloud pulse | (unchanged) | `poisonDamage` already matched: signed water resistance, 0.5 bias, no draw | `MAGIC-302` |

### Lifetimes

Stone Curse 20 on a unit: duration word times 100 less the signed Earth
resistance in signed 32 bits, divided by 100, kept as a word; zero becomes one
(`MAGIC-303`). Before, the resistance was clamped to 1..100 and a zero or
negative resistance left the word.

Compared and unchanged: the area counters 240 + trunc(16*power/10) and
320 + trunc(16*power/10) (`areaLife`); the cloud tick, which decrements before
it pulses on a multiple of 16 including zero and expires at an entry counter of
zero (`MAGIC-304`); the attached-effect timer's 9600 guard; Poison Cloud's
continuous counter 128. Not adopted, each a row: Freezing Cloud's and Light's
inner counters (`DIV-3003`), the timer's continuous pulse above 9600 and its
zero wrap, the binary64 `pow` boundaries and the staged-area totals
(`DIV-3004`).

### Hashed state that moves

- Every first-game damage application on a unit takes two more seeded draws
  (three on a Bless or Curse source), and a zero pair one fewer.
- Applied damage moves where binary64 rounds below an integer (installed
  Lightning and Prismatic Spray 5/15 at powers 20, 32, 40 and 94, Wall of
  Fire 1/3 at power 40, measured by the release witness's census), where a
  product passes 255 (modded columns only), and on a negative resistance.
- Heal and Drain Life land one more on their lowest draw.
- Stone Curse lasts longer on a negative Earth resistance.

Tests repinned with this cause: `TestAreas1164FinalZeroAndFootprintVisits`,
`TestCadence1045AreaReleaseRebindsTheBookSweepAfterEveryRemovalShape`,
`TestFireballDividesEachByteComponent` and
`TestFireballComponentDivisionNativeCastContinuation` (the resolver's two
physical draws; the Fire Ball table was recomputed from independently reduced
SplitMix words), `TestACastSpendsManaOnceAndDamagesTheVictimByAnExactAmount`,
`TestAVictimExactlyAtMaxRangeIsInReach` and
`TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks` (the roll and the
recovery jitter are later words), the four Drain Life tests (base + 1),
`TestCurrentPoolsFollowRuntimeMutations` (Heal fixture 5/5 heals 6).
`TestCurrentDeliveryKeepsClockAboveOrdinaryTickWidth` marks its native rows
second-game, as the cold World already did.

Release tests moved with the same causes:
`TestReleaseMilestone2CurrentProducers1162/P2_Unit` (the weapon rider's
spread draw is a later word: the rider lands 9, not 7);
`TestReleaseModFormulaTablesReachTheCastTheBookAndTheSave` (a mod table's
zero-spread Heal heals its value plus one, `DIV-3002`). Mission 10's ordinary
route in `TestReleaseLoadedCreaturesKeepTheirSheets` and
`TestReleasePlacedSpeakerKeepsHisWornSetAfterLoad` reaches the hero's fight
17 ticks sooner on the moved stream; the hero turns to another attacker,
fells it and stands idle, so the route's `kill` clicks the target again when
the hero has no attack order, as a player would.

## Proof

- `TestSpellRecordDamageEqualsBinary64` (`pkg/game`): every damage byte
  0..255 against every power byte 0..255, integer form against float64
  computed outside `pkg/sim`: 16777216 inputs, none differ.
  `TestPoisonMagnitudeEqualsBinary64` and
  `TestResistedSpellDamageEqualsBinary64` cover the other two forms.
  `TestSpellRecordDamageLeavesTheIntegerFormula` is the loss control.
- Focused tests with loss controls (`pkg/sim/castdamage_test.go`): byte
  narrowing, binary64 rounding, signed resistance, the draw order, the
  building, Heal, Fire Sacrifice, Poison magnitude, Stone Curse.
- Release witness `TestReleaseCastDamageFollowsTheSpellRecord`, EN and RU:
  mission 10's generated mage at power 20 casts Fire Arrow 1 and Lightning 13
  96 times each on a target with resistance -20. Fire Arrow: record 6..13,
  landed 7..16, both ends reached. Lightning: record 8..24, landed 10..29, both
  ends reached, and the integer formula's top 25 (30 resisted) never lands.
  Stone Curse at power 20 on Earth resistance -40: word 262, applied 366.

## Open debt

`DIV-3000` to `DIV-3004`; the popup's own fold differences stay in
`DIV-1882`.
