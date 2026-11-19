# Second-game kills count by creature kind and keep the hero's chosen skill

## Result

In the second game, kills of creatures that share one kind now count together on the hero's
list, and the list keeps its counts across a save and a load. The hero keeps the skill chosen
for him at the start as his main skill, through the party, a save and a load, so its training
is not cut like the other schools. A fighter credited with a spell kill, or already dying when
his victim falls, earns no skill for it, as in the original.

## Authority

- `R2-ENGINE-320`: the chosen skill becomes the hero's main skill (Medium: first of equal schools).
- `R2-ENGINE-480`: the award cuts every school but the main one; step 5 takes a spell skill only for a mage.
- `R2-ENGINE-1157`: the repeated-victim list keys on the kind word (Medium: the list roles).
- `R2-ENGINE-1158`: the kill route's two gates (Medium: +0x48 holds the killing spell; +0x68 is the award).
- `R2-ENGINE-319`: the main skill is the first word of the SAV experience block (Medium: bound reads).
- `R2-ENGINE-525`: the kind word of Units and Humans rows (Medium: a Units -1 is stored as 0xFFFF).
- `DIV-4154`, `DIV-4839`, `DIV-4844`, `DIV-4845` closed; `DIV-4997` holds what the claims leave open.

## Rules

1. Every second-game actor holds a kind word: a Units row's parameter 55, a Humans row's id, 21 for the hero.
2. A second-game hero's main skill is the chosen skill, or a template hero's first largest school.
3. The kind word and the main skill cross the party, the World form, a SAVE and a LOAD.
4. The repeated-victim list counts kills by kind word, before and after a SAVE and a LOAD.
5. A second-game kill pays no award to a killer below zero health or to a non-mage credited with a spell.
6. Map placement, party, hire and summon set both fields through the one actor constructor.
7. First-game awards, saves and World hashes are unchanged.
8. A mission SAVE with no kind section loads each kind word from the actor's record.

## Code

| kind | its one builder | change |
|---|---|---|
| actor constructor | `sim.NewActor` | takes the kind word and main skill from `mapload.RowActorKind` and `partyActorKind` |
| skill award | `World.awardSkill`, `World.processKillCredit` | reads the actor's main skill; the kill gates are award profile data |
| SAV record writer | `secondActorTemplateKey`, `secondMainSkillHeld` | the town writer writes both; a mission record writes U148 from `DisplayBacking`, and the kind word crosses a mission SAVE through the World form |
| SAV record loader | `admitOriginalActorRegistry`, `Entity.restoreValues` | read both back; an older save takes the main skill by the template rule, and with no kind section its kind word from the record's U148 |
| game branch | `secondGameSkillAward`, `Table.ActorKindWords` | the second game's profile and table data |
| World form | `World.appendActorKinds` | one section, written only when an actor holds either field |

## Proof

| test | rules |
|---|---|
| `TestSecondGameRowsGiveTheirKindWordAndMainSkill` | 1, 2 |
| `TestSecondGamePlacementsCarryTheirKindWordAndMainSkill` | 1, 2, 6 |
| `TestSecondGamePartyActorsCarryTheirIdentityAndMainSkill` | 1, 2, 3, 6 |
| `TestSecondGameSummonedCreatureCarriesItsKindWord` | 1, 6 |
| `TestSecondTownSaveKeepsTheHerosMainSkill` | 3 |
| `TestSecondGameVictimListCrossesSaveAndColdLoad` | 3, 4 |
| `TestReleaseSecondOldMissionSaveTakesKindWordsFromRecords` | 8 |
| `TestKillRouteCallsTheAwardForTheClaimedKillers` | 5, 7 |
| `TestFirstGameAwardsKeepTheirWorldHash` | 7 |

Release tests at the release: `TestReleaseSecondGameRepeatedVictimCrossesSaveAndLoad`,
`TestReleaseSecondOriginalFileKeepsTheHerosMainSkill`, on the EN and RU installs.

Evidence: `review/story1703-rom2-skill-award/RESULT.md`.

## Open debt

- `DIV-4997`: a mage's kill credited to a spell with no row, and the claim's two readings of a non-mage.
- `DIV-4840`, `DIV-4841` stay open.
