# Hired siege squad as a Unit actor

## Intent and authority

The city SAV wrote a hired Catapult or Ballista as a Human actor on definition row 0. The original keeps that actor as an extra Human beside its own rebuilt Unit, so the owner saw two Ballistas in the party. Authority: `MERC-LEVEL-005` (High, types 1 and 2 are `Unit(0x198)` objects named by the literals "Catapult" and "Ballista"), the owner ruling that a field is written from current state, then a rule from evidence, then loaded bytes, then a constructor value, and two original resaves of the city kit that hold the original's own Ballista Unit.

## As built

Writer. `nativeCityDataConstruct` writes each hired siege member as a `Unit` object in the hire group (`pkg/game/nativecitysiege.go`). The Units row is the one named "Catapult" or "Ballista"; the statistic words, face, reach, sight, mana floor, attack words and the weapon come from `mapload.SiegeHireActor` (`pkg/mapload/siegehire.go`), which uses the same constructor and Units block as the mission start. The held weapon is written through the holdings writer (`cityMemberEquipment`). The party projection captures the member through the existing `Unit` branches.

Reader. `persistentOriginalCharacter` keeps a `Unit` whose row names a siege engine (`originalSiegeHire`); `restoredMember` builds it with `siegeHireMember`, the builder the tavern uses, so a loaded hire equals a fresh one. The restored member holds no worn items (`restoreFromState`), as a fresh hire; the weapon belongs to the Units row. A legacy Human actor with row 0 stays dropped, and `restoreHiredMercenaries` rebuilds the squad from the hire flag once per type.

Field table of the Ballista Unit (witness: original resave `game0009` object 18 and `game0014` object 97, equal in every row below):

| field | value | source |
|---|---|---|
| token row, type | Units row of "Ballista", its TypeID | Units row, witness row 27 |
| token publication mask | 0 | witness (hire constructor) |
| token map unit id | 0 | uninitialised in the witness (DIV-1719) |
| token size word, domain, face, class flags | Units row | witness equals row |
| state word (U50) | 0x0b | witness, hire constructor |
| statistics, speed, load, capacity, health | Units row and actor basis | witness equals row |
| experience, attack charge, relax, sight, reach | Units row | witness equals row |
| word 41 (U136) | 1 | witness, hire constructor |
| display backing (U148) | hire type | witness (2); Catapult 1 is DIV-1718 |
| mover | passability mask, bytes 5 and 0xff, speed | witness |
| insert index, container flag | 10000, 1 | witness, hire constructor |
| name, spellbook, experience block, equipment array | empty | witness; a Unit holds none |
| held weapon | the Units row's weapon, one effect, one weapon spell | witness semantics |
| uninitialised bytes | 0 | DIV-1719 |

## Proof

- `TestReleaseCitySiegeHireWritesUnitActor` hires both squads, writes the city SAVE, asserts the Unit rows, hallmark fields and weapon, then cold LOADs (one member per type, equal roster), writes the second city SAVE, enters the mission with both engines present, writes a mission SAVE and returns a squad as the loss control. It runs on EN and RU.
- `TestReleaseOriginalSiegeUnitRecognition` pins what the reader accepts.
- `TestReleaseOriginalCityResaveSiegeUnit` loads the original's resave, which holds the engine's Human row-0 actor and the original's Unit together, and requires one Ballista member. It reads `AGAINROM_OWNER_KIT_RESAVE_CITY` and skips when it is unset.
- `TestReleaseCityGroupsRepeatedHireCyclesF2SAV` now expects a `Unit` for a siege hire.

## Open debt

DIV-1717 (Human hire constructor values), DIV-1718 (Catapult witness), DIV-1719 (uninitialised bytes), DIV-1720 (mission writer and restored engine state). DIV-1710 is closed.
