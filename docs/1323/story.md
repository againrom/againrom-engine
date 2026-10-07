# Spell power up to 255

## Intent and authority

Owner direction: the spell power term clamps at 255 instead of 100, in the
ordinary game as well as under mods. Players reach power above 100 because the
effective school skill can pass 100 (DIV-2217) and because a Mind of up to 100
adds to a skill of 100 (`skill + Mind - 30` reaches 170 without any item).
Research states the original's clamp at 100 (`MAGIC-POWER-004`,
`MAGIC-CEIL-013`); the differences are DIV-2221 to DIV-2224.

Owner rulings for the durations:

1. Invisibility and Stone Curse above power 100 last `k * D(100) + D(P - 100k)`
   with `k = floor((P-1)/100)`. The Stone Curse cut by earth protection applies
   after.
2. Every other power-scaled duration keeps its law up to 255 and saturates at
   65535 instead of wrapping.
3. An attached effect with more than 9600 ticks left does not count down. The
   owner accepted the resulting permanent effects.
4. Area durations, magnitudes, ranges, damage, Teleport, Light, Darkness,
   Haste and Slow speed, Shield absorption, protection `P/2` (still held to 100
   per element), Bless and Curse, and the Prismatic Spray ray cap follow the
   larger power with their existing formulas.
5. Slow and Freezing Cloud leave a unit's speed at 1 at the least, at every
   power up to 255. The floor is `minEffectSpeed`.

## As built

- `pkg/sim/spell.go`: `spellPower` clamps 0..255 (`spellPowerMax`).
  `lastingTicks` reads the Q56 tables up to power 100 unchanged and the new
  Q64.64 samples of 1.025^p and 1.05^p for p 101..255
  (`pkg/sim/spellpowertables.go`, floored from exact rational arithmetic). A
  result above 65535 saturates at any power. `segmentedTicks` is the
  Invisibility and Stone Curse law; `spellLastingTicks` selects the law by row
  id for the cast, the popup record and the item tooltip.
- The popup record power (`spellRecordPower`) follows the cast power up to 255.
  A sum below 30 keeps the original byte fold and reads 100.
- `clampByte` replaces the byte truncation of the damage base and spread in the
  in-flight and area payloads, so a modded damage column saturates at 255
  instead of wrapping (DIV-2224).
- Item, scroll and script casts keep their own power source. Only the duration
  tables and the damage payload bytes changed for them.
- Poison Cloud's effect duration is the row's fixed column value and never
  followed power (`MAGIC-POISONINPUT-157`); it is unchanged. Its magnitude
  follows power.
- Bless and Curse magnitudes (`4P/5 + 20`, 224 at 255) are stored; no engine
  reader consumes them.
- `cmd/againrom/VERSION` 0.59.0.

Reference values, checked against exact rational arithmetic for every power
101..255 and eight column values: Invisibility 6312 at 100, 6862 at 150, 12624
at 200, 13326 at 255; Stone Curse (column 10) 1890, 2439, 3780, 4402;
Protection (column 30) 5670 at 100, 9291 at 120, 65535 at 200 and 255; Shield
and Haste (column 15) 2835 at 100, 15968 at 170, 65535 at 255.

## Proof

`pkg/sim`:

- `TestSpellPowerDurationsAboveHundredMatchTheExactLaw`: all powers 101..255,
  both laws, eight columns, against `math/big` rationals.
- `TestSpellPowerDurationsUpToHundredAreUnchanged`: columns 1..340, powers
  0..100, both laws, against a copy of the previous function (a result above
  the word saturates now).
- Saturation, reference values, segment law at 101, 200, 201 and 250,
  `TestSpellPowerClampsSkillPlusMindMinus30` to 255,
  `TestSpellRecordPowerFollowsCastPower`.
- `TestCastAtPower255AttachesTheSegmentedAndSaturatedDurations`: the book cast
  route at power 255 stores 65535, 13326, 4402, 65535 and 65535 for Protection,
  Invisibility, Stone Curse, Haste and Slow, and magnitude 18 for Haste and
  Slow. The Stone Curse cut test runs through 50 earth protection.
- `TestHasteAndSlowAtPower255MoveUnitsThroughTheMovementPath`: speeds 30, 12 and
  48, positions after 400 ticks ordered slow < base < haste, a speed-5 unit
  held at the floor still moves.
- Every pinned world hash and byte-form test in the package passes unchanged:
  the states they build never reach power above 100.

`pkg/game` release witnesses (EN and RU, in the gated population):

- `TestReleaseSpellPowerAbove100EffectsSaveAndColdLoad`: a mage with skill 100
  and the capped Mind casts Haste, Protection from Fire, Shield and
  Invisibility at power 120. Durations at attach are 4645, 9291, 4645 and 6439,
  SAVE goes through the ordinary producer, the cold LOAD holds the same effects
  and the same speed, the next 300 ticks leave live and loaded equal, and the
  second SAVE carries the Haste magnitude.
- `TestReleaseSpellPowerBoundEffectsSaveAndColdLoad`: a chargen mage with five
  worn school bonuses (skills 250, Mind 24, power 244) self-casts Haste,
  Protection from Fire, Shield and Invisibility. Attach words are 65535 for the
  first three and 13034 for Invisibility, Haste magnitude 17; after SAVE the cold
  LOAD holds the same skills, speed, spellbook and effects, and live and loaded
  stay equal for 300 more ticks.
- `TestReleaseSpellPowerBonusEquippedInPlaySavesAndLoads`: a chargen mage with
  trained skills 100, 80 or 60 equips a +30 school bonus in play, saves through
  the ordinary producer and loads cold; skills, spellbook and spell
  characteristics (power, range, duration) equal the live actor after five more
  ticks. Before the fix the trained-100 cases (Mind 24, 100 and 10) refused the
  LOAD; `TestReleaseSpellPowerBoundEffectsSaveAndColdLoad` only passed because
  its item is worn at mission open.
- `TestReleaseSpellDamageFitsTheEffectByteAtPower255`: no installed damaging row
  exceeds 255 at power 255 (largest 237, Meteor Storm).
- `TestReleaseSlowAtPower255KeepsTheSpeedFloor`: on the slowest (8) and fastest
  (35) installed unit, Slow and Freezing Cloud at power 255 leave speed exactly
  1 where the sum exceeds the speed, the unit transits at the cadence of an
  unslowed speed-1 unit, Slow stays at 65535 ticks past 3000 ticks, and at power
  120 both effects expire and return the exact original speed.

## Open debt

- No original observation exists above power 100. All values above are owner
  direction.
- A SAVE of a state at power above 100 writes durations and magnitudes inside
  the original's 16-bit words. The one projection is the in-flight damage byte
  (DIV-2224); the installed rows never reach it.
- Effects with more than 9600 ticks left never expire. Protection from about
  power 122 and Shield and Haste from about 150 stay until replaced; no dispel
  exists.
- The LOAD refusal of a book root above power 100 (`saved book root differs from
  current actor`) is fixed in this story: a native actor holds its skill clamped
  to the training cap until its worn bonus is applied, so the saved range byte may
  be the reach at any power up to the bound. The validation accepts a saved range
  between the derived reach and the reach at power 255 and compares every other
  field exactly. The check is looser for a corrupt range byte inside that band.
- The enchant shop's generated staff power stays capped at 100 (item power source
  unchanged).
- Bless and Curse magnitudes have no engine reader.
- The Protection popup caption shows the applied value, at most 100.
