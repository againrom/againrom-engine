# A second-game hero carries the original's experience and record

## Result

A new second-game hero starts with the original's skills, experience, health and mana, and a hero who picks Shooting
or Astral starts at level 10 in it, as the original's producer leaves him. Every second-game hero keeps his experience
across SAVE and LOAD. The town save writes the hero's record as the original writes it: his template row and the
experience word template load leaves are now written too. The first game's heroes and saves are as before.

## Authority

- R2-ENGINE-290 (High for the arithmetic, predicates and weapon table; Unknown for item effects): the producer.
- R2-ENGINE-320: store order of the chosen skill and skill 5; template load writes +0x1c. R2-ENGINE-105: party identity.
- DIV-4981 and the owner: a stat cap limits potions only; a SAV or map value above it plays; items add above it.
- Owner: SAV rule order is current state, then a rule from evidence, then loaded bytes, then a constructor value.

## Rules

1. A new hero at a template's statistics has the claimed level 1 health and mana: 142, 123, 29 and 42; 157 and 99.
2. A new hero's skills are 0 but the chosen skill at 20 and skill 5 at 10; a chosen skill 5 holds 10.
3. A new hero's experience is the sum of his skills' experience, 7320, and his main skill is the chosen one.
4. The town save writes the hero's row, +0x1c, statistics, skills, experience, modifiers and defence as the original.
5. SAVE and LOAD keep each skill's experience of a generated hero and of an original's hero.
6. Each Human record of the original files holds at +0x1c the experience rule of his Humans row.
7. The first game's hero, saves and world hashes are as before.

## Code

| kind | its one builder | change |
|---|---|---|
| generated hero | `generatorParty` in `pkg/game/chargen.go` | template row kept; chosen skill before skill 5 |
| town SAV Human record | `secondFormHolders` in `pkg/game/savform.go` | hero's T1C by the row rule `secondRowExperience` |
| Human recompute | `data.DeriveHuman` | unchanged; the game profile's rules select pools and caps |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondNewHeroLevelOneSheet` | 1, 2, 3 |
| `TestReleaseSecondNewHeroRecordIsTheOriginals` | 4 |
| `TestReleaseSecondHeroExperienceCrossesSaveAndLoad` | 3, 5 |
| `TestReleaseSecondRowExperienceIsTheOriginals` | 6 |
| `TestChargen*`, `TestReleaseAMissionTenTurnHashMatchesAcrossBothLawfulRoots` | 7 |

Release tests at the release: the four `TestReleaseSecond` tests above. Evidence:
`review/story1719-rom2-hero-model/RESULT.md`.

## Open debt

- Item-effect tables and +0xe8 skill modifiers stay Unknown (R2-ENGINE-290): DIV-5001.
- The record's unsourced bytes, a new mage's UA0 and U6C, and a companion's T1C stay DIV-4305.
