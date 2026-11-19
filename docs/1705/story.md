# Slow is a castable spell of the second game

## Result

A hero loaded from an original second-game save keeps Slow in his spell book with the cost and range the save
stores, and he casts it. The two Necromancer creatures whose spell list holds Slow keep it and cast it. A map can
place Slow, and a book of Slow teaches it to a mage who does not know it, in a mission and in the town shop.
The first game's books, saves and worlds are unchanged.

## Authority

- Owner: every spell a second-game map names resolves to a rule and casts in play (DIV-2621, now closed).
- `R2-SESSION-237`: the EN game0005 and game0006 books hold spells 1 to 29. `R2-ENGINE-017`, `R2-ENGINE-019`:
  the book is indexed from 1 and Slow is id 29. `R2-ENGINE-619`: Slow has data, an apply arm and cast id 66.
- DIV-4174 and DIV-4369 keep what research does not state: the book page cell and the original's own casts.

## Rules

1. When a second-game LOAD holds Slow in a hero's book, he keeps it with its saved range and cost and casts it.
2. When a Units mask names Slow, the creature's book holds it with the spell row's range and cost; it casts.
3. When a map placement names Slow, the cell is kept; the first game still drops it.
4. When a mage who does not know Slow reads a book of Slow, he learns it and the book is used up.
5. When a book holds Slow, SAVE and the cold LOAD keep it; a spell past 29 is refused.
6. When a first-game book or world is saved, its bytes and hash are those of before.

## Code

| kind | its one builder | change |
|---|---|---|
| spell table | `pkg/rules/spellcatalog.go:95` SpellProfile | read per game, no new row |
| book | `pkg/sim/spellbook.go:34` BookIDs, `:51` Slot, `:63` BookSpellCount | one slot more, as data |
| book bytes | `pkg/sim/bookbeyondbinary.go:11` section 140 | written only when a book holds Slow |
| book SAV path | `pkg/game/savbookcurrent.go:14` projectCurrentBookGraph | 29 refs; format bound named |
| creature casting | `pkg/mapload/spell.go:137` CreatureBooks.add, `:26` unitSpellbook | mask and cell reach 29 |
| spell effects | `pkg/sim/spellarms.go:473` SecondGameSpellArm | unchanged; Slow runs its arm |
| game branch | `pkg/sim/spellbook.go:63` BookSpellCount | the profile's spell count |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondOriginalBookKeepsAndCastsSlow` | 1, 5 |
| `TestMaskBookCreatureCastsSlow` | 2 |
| `TestBookSpellCountWalksEveryIDOfBothGames` | 2, 3 |
| `TestSecondGameWorldReadsABookOfSlow`, `TestSecondGameMageLearnsABookOfSlowInTheShop` | 4 |
| `TestBeyondSlotBookSectionIsWrittenOnlyForASpellPastTheSlots` | 5, 6 |
| `TestTheSecondGameBookHoldsSpellTwentyNine`, `TestShopStateRecord` | 5, 6 |

Release test at the release, on the EN and the RU install: `TestReleaseSecondOriginalBookKeepsAndCastsSlow`.

## Open debt

- DIV-4174: the book page shows no cell for Slow (no slot-table row) and the original's own casts of Slow are
  unresearched; a computer caster never chooses Slow (`R2-ENGINE-1247`).
