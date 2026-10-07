# 1231 — creature combat state in SAV

## Intent

Fresh mission 140 lost the two `Bat_Sonic.4` actors' held Sonic Beam, combat
words and spellbook when Againrom wrote SAV. Their original `rood.sav` records
carry those fields. The result applies to Units definitions generally, including
custom rows, and survives cold LOAD, subsequent ticks and a second ordinary
SAVE. Creature colour is a separate owner control.

## Authority

Pinned `UNIT-STREAM-001`, `HERO-DMG2-029`, `UNIT-SPELL-007`,
`ITEM-DEATH-012` and `UNIT-DIFF-002` name the stream, class flags, book and
death rules. The owner's `rood.sav` provides the mission-140 record comparison;
`savedoverbat.sav` shows the former output loss. Neither SAV is tracked.

## As built

- A resolved Units row keeps its weapon in the living worn slot, including a
  weapon with `Suitable=0`. Its resolved equipment operands and optional cast
  spell reach current state. On terminal death that weapon remains with the
  actor; a carried weapon still drops. Human-row starting equipment retains
  its existing policy.
- The Units `Spell 1..3` cells build the current book from resolved Spells
  definitions. The first spell cell controls book presence. Dragon and Daemon
  force a book. The top tier sets the five non-General skill words to 30; the
  General word receives the authored ToHit before difficulty and equipment.
  The constructor writes book and always-hit class flags from current state.
- The ordinary mission SAV producer reads these current values. It does not
  select a writer based on whether the world came from an original SAV. No new
  persisted simulation field or save format was added.

## Proof

- `TestReleaseCreatureInnateCombatSAV` on EN and RU starts mission 140 and
  checks map IDs 1098 and 1099 in the first SAV and after cold LOAD, 32 ticks
  and a second SAV: exact `U4C=0x12`, `UA6`, Sonic Beam `F40=0xf118`, and
  book slot 16 (wire spell ID 17, range 6, cost 5).
- Synthetic Units rows test a held innate weapon, an ordinary carried weapon,
  a row without the suitability column, a negative suitability value, their
  combat values and terminal loot. Another custom row checks always-hit,
  constructed class flags, top-tier skills and a spellbook; a plain creature
  remains the control.
- `TestReleaseEnchantedItemProducerPopulation` checks the shipped EN/RU Units
  population: Dragons hold Flame Thrower without a cast suffix, while
  Catapult/Ballista casts come from their held item. Terminal Dragon death
  retains that weapon and adds no Flame Thrower to ground loot.

## Open debt

- The three authored spell probability cells and their original AI threshold
  consumer are not represented by this SAV book result. They remain an AI
  scheduling obligation under `UNIT-SPELL-007`.
- A custom spell outside the current 28-slot book or without a decoded Spells
  definition leaves a present but empty book, without refusing world creation.
- A pre-fix SAV that already lost a creature's held weapon and book does not
  reconstruct those bytes from its row on LOAD. The fresh construction path
  and subsequent saves carry them.
- The owner's one-byte original runtime time-flow controls drew both mice
  black. The earlier brown input remains unexplained; colour is unchanged.
