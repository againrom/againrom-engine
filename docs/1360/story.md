# Enemy card knowledge

An enemy creature's mission card and tooltip draw each field group from the player's knowledge level of that creature, and the level grows with hero-typed kills. The Diary count is written to SAV, restored from SAV (native and original), carried to the next mission of the campaign, and never changed by Shift+F4.

## Authority

Public pin k196 (`c2166832`). `UNIT-145` gives the gates per level, `UNIT-146` the level sources, `UNIT-147` the kill count, writer and cap, `UNIT-148` the Diary lifetime, `SAV-1172`/`SAV-1173` the Diary in the SAV Player tail, `SAV-1125`/`SAV-1126` the packet-fed Human values, `TEXT-UI-036` as narrowed. Owner direction: implement the count as the original does, write it to SAV, restore it from SAV; Shift+F4 shows every card for display only. Unknowns (first read after LOAD, town path, level of Humans) take the smallest rule consistent with the claims and each has a DIV row.

## As built

Level L of a unit, from the local player's point of view:

| Unit | L |
|---|---|
| owned by the local player | 7 |
| creature (type 64..80, Units row 64 or above, face 1..4) | `min(count>>1, 7)` of the local Player's Diary count for its row |
| any other unit (non-owned Human or hero) | 0 (`DIV-2448`) |
| any unit while Shift+F4 is on | 7, display only |

Gates drawn at level L or above: Health and Mana 1; Sight and Speed 2; attack and damage 3; armour and defence 4; Body, Agility, Mind, Spirit 5; resistances and their heading 6; skills and their heading 7; the exactly-7 captions (Spellcaster, Armor piercing) at 7. Weapon and Worn draw at 7 and Swing at 3 (`DIV-2453`). XP, weight and cell are always drawn. The card and the tooltip read one subject, so they agree.

The Diary writer runs where the victim's kill credit is processed. A victim adds one to the Player Diary entry of its Units row when the credited killer exists, is hero-typed (type 33..63), has health of 0 or more and is owned by the local player. A Humanoid victim above row 63 and a victim with no row add nothing. The count rises while it is 16 or less (cap 17) and the remainder word falls from 1024 while nonzero. A downed body that falls below zero and leaves in one decay pass is counted before it is removed (`DIV-2449`). The row of a placed creature comes from install-derived placement rows set at map load and carried across every decode, like the rules; a restored actor uses its source binding row.

The Player Diary lives in the existing `savedDiaries` World state, so the SAV producer writes it from current state and a cold LOAD restores it; no wire field is added. A world with no kill carries no Player entry, so existing hashes are unchanged. An original SAV's Diary is read by the existing importer and the level follows from it at the first frame (`DIV-2450`). Staging that rebuilds a world (original living-actor import, holdings) now carries the install-derived rows.

A won mission stores the Player Diary on the town; the next mission opened from that campaign starts with it. A town SAVE writes the town's Diary into the Player record (`projectTownKnowledge`) and town LOAD, native and original, reads it back (`savedPlayerDiary`). The placement rows of the Diary come from `alm.Map.AuthoredUnits`, the units as decoded, so a load that withdraws a placement keeps the same rows. A new campaign starts empty (`DIV-2451`). Scope limits and the cheat producers are `DIV-2452` and `DIV-2454`. `DIV-2080` and `DIV-2178` are updated.

## Proof

Tests in `pkg/sim`, `pkg/ui`, `pkg/mapload` and `pkg/game`, run on EN and RU installs where they read an install:

- `TestReleaseEnemyCardUnitsTableHasTheAdmittedRows`: the Units table has 53 admitted rows over 14 types with no (type, face) collision, found through the install's own search.
- `TestReleaseEnemyCardLevelFollowsTheDiaryCountPerRow`: counts 0..17 on a Giant Bee row (mission 10, 11 placements) give the card level `min(count>>1, 7)` on every instance, level 0 on another row, level 7 on the hero; the card statement is unchanged inside a level and grows at each step.
- `TestReleaseEnemyCardShiftF4ShowsEveryGroupForDisplayOnly`: level 7 while on; with it off the card, World hash, Diary and mission state equal the values before the chord.
- `TestReleaseEnemyCardKnowledgeSurvivesSaveAndLoad`: two hero kills give count 2 (level 1); the SAV carries 2; a cold LOAD restores 2 and level 1; the next two kills continue to 3 and 4 and level 2. Loss controls: a SAVE written after the Diary is dropped loads at 0; the same SAV with its Diary zeroed in the file loads at 0.
- `TestReleaseEnemyCardOriginalSAVDiaryLevels`: game0002-bigsack and game0017-victory (save corpus, skipped when absent) load with creature levels `{0:15, 1:10, 3:9, 4:5, 7:46}`, each equal to an independent nibble table built from the SAV's counts and the install's (type, face) rows, and equal in the viewer's entity data. Loss control: a world without the Diary reports level 0.
- `TestReleaseEnemyCardKnowledgeCarriesToTheNextMission`: count 5 at a mission's return opens the next mission with 5; a new campaign opens with 0.
- RED: the same tests at base `c9362810` fail with level 7 on every enemy card and count 0 after hero kills (EN and RU).

The cold-LOAD test fails if the original living-actor import drops the install-derived rows, and the kill test fails if a body that leaves in the pass that takes it below zero is not counted.

## Open debt

- A mission left without a win carries its kills nowhere, and town entry and exit are not established (`DIV-2451`).
- Levels above 7, `#Chicken` and `#modify +knowledge` (`DIV-2454`).
- Another player's hero kills, summoned bodies without a row (`DIV-2452`).
- Level source of non-owned Humans and heroes, first read after LOAD, teardown tick, gates of the remaining card lines (`DIV-2448`, `DIV-2450`, `DIV-2449`, `DIV-2453`).
- `DIV-2455` is unused.
