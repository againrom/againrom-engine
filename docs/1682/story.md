# A mod sets a second-game spell's numbers by the first game's tables

## Result

A mod for the second game can now give a spell its own power, damage, range, duration and effect strength by power,
in the same tables and the same file the first game's mods use. A cast of that spell follows the mod's table, and the
book shows the same numbers. A table the spell cannot use is refused with the spell, the line and the reason. Without
a mod nothing changes in either game.

## Authority

- Owner: «Spells in mods»: «Formulas stay factor tables, extended to the second game; no free formula language until
  a spell needs one.» This story is that fourth part, after `docs/1673/story.md` and `docs/1678/story.md`.
- `DIV-2364`: the formula tables, now read in both games.
- `R2-ENGINE-022`: the second game's Invisibility lasts 16 ticks per power.
- `R2-ENGINE-025`: the second game's Stone Curse draws its duration.

## Rules

1. A row's power, damage, range, duration and magnitude tables set a second-game cast's numbers under that game's spell.
2. Every second-game spell number, the last one included, holds its own tables and the global ones.
3. Invisibility reads the duration table on its base of 3 in both games; Poison Cloud reads it in the second game.
4. The second game's Stone Curse refuses a duration table; a global one leaves its duration unchanged.
5. A table a spell cannot use, and an entry or a size past its limit, is refused naming the mod, the file and the line.
6. Without a mod every second-game spell number is the one computed before, and the first game's are unchanged.
7. The modding guide states the keys once for both games, with their limits and refusals.

## Code

| kind | its one builder | change |
|---|---|---|
| spell formula set | `pkg/rules/spellformula.go` SpellFormulaSet, WithSpellFormulas | sized to the larger game profile |
| formula freeze | `pkg/mapload/modspell.go` SpellFormulas | keeps the second game's last spell |
| duration reader | `pkg/sim/spellvalues.go` spellLastingTicksUnder | the table before Invisibility's power rule; not Stone Curse's draw |
| duration table rows | `pkg/sim/spellvalues.go` DurationTableRule | the rows the point duration reads the table for |
| mod check refusal | `pkg/mapload/modspell.go` modSpellFormulaRefusal | asks the row's arm in its game, not its number |
| spells mod kind and merge | `pkg/mod/spells.go` | unchanged |

## Proof

| test | rules |
|---|---|
| `TestSecondGameFixtureTablesSetTheCastNumbersAtTwoPowers` | 1, 3 |
| `TestSecondGameRowsReadTheFormulaSetUnderTheirOwnNumbers` | 1, 2, 3, 4 |
| `TestTheFormulaSetHoldsEveryNumberOfEitherGame`, `TestASpellTableWinsOverTheGlobalOneAndReadsClamped` | 2 |
| `TestSecondGameFormulaRefusalsNameTheRowFieldAndLimit` | 2, 4, 5 |
| `TestSecondSpellTablesFixtureCarriesItsTablesOnTheSecondGameOnly` | 1 |
| `TestSecondGameSpellNumbersWithoutAModAreTheCommittedOnes` | 6 |
| `TestDurationFactorIsThousandthsOfSixteenthsAboveHundred`, `TestFormulasWithoutTablesAreTheOriginalArithmetic` | 3, 6 |
| `TestModsDocStatesEachFormulaKeyWithItsLimitsAndRefusal` | 7 |

Release tests at the release: `TestReleaseModFormulaTablesReachTheCastTheBookAndTheSave`,
`TestReleaseWithoutFormulaTablesTheCastKeepsTheOriginalNumbers`, `TestReleaseModSpellEditsReachACastAndTheSave`,
`TestReleaseSecondGameAttachedEffectsActAsTheirOwnArms`,
`TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD`,
`TestReleaseSecondGameSpellbookPopupStatesTheCastPair`, `TestReleaseSecondGameMissionTwentyOneMageCastsIceMissile`.

## Open debt

- The second game's Stone Curse duration draw and an area's life have no table; a spell that needs one needs a new
  table element (owner: no free formula language until a spell needs one).
