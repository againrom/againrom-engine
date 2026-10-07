# SAV M10 owner kits

## Intent and authority

Build the two owner kits that carry the M8 gap audit's original-game obligations: a city file and a combat file on each of EN and RU. The owner loads each file in the original game, acts, resaves, and the resaves are loaded here. Authority: `pipeline/SAV-ENDGAME.md` M10, the owner rule that every chooser label reads "<number> <change>", and the owner ruling that a field is written from current state through one SAV producer. No ROM1 behaviour is asserted.

## As built

The kit files come from ordinary engine state and the ordinary F2 SAVE route. Nothing is patched in bytes.

Kit A, city (`game9601` EN, `game9602` RU). Walk the native campaign to chapter 120 through `Won` for each earlier main mission, cold LOAD the city, then:

1. Buy the installed Health Regeneration potion through shelf, table and Buy, and drink it on the hero.
2. Train the cheapest mage school slot for the companion mage, then buy the merchant book of the first spell she lacks (spell 2).
3. Press the tavern squad button for the Human squads 6, 14 and 13 and the siege squad 2, return squads 6 and 2, hire both again.
4. F2 SAVE.

Result: 14 party members, hired squads `type 2 x1, type 6 x4, type 13 x4, type 14 x3`, an active regeneration record (kind 12, +100, 960 remaining), the companion Fergard knowing spells 1, 2, 6, 12, 18. Chapter 120's tavern offers all four squad types, so the owner can return and hire them in the original.

Kit B, combat (`game9603` EN, `game9604` RU). Native arrival at mission 30, the companion mage Reniesta buys the merchant books of spells 2 and 3, mission entry, then:

1. The hero attacks and kills the nearest creature (a body).
2. The hero drops his first pack item through `DropCarried` (a new Sack beside him; the map already holds four).
3. Reniesta casts spell 3 at a cell (a standing area effect that outlasts the sequence).
4. Reniesta casts spell 2 at a cell five or more cells away; the tick the delivery enters flight, F2 SAVE.

Result: 5 Sacks, 5 bodies (four were on the map), 1 area effect, 1 spell delivery in flight. The delivery is the engine's flight object; it has no projectile record in the SAV (DIV-1185). A projectile record exists only for a flight loaded from an original file.

Labels (typed through the F2 dialog, encoded by `encodeSaveLabel`, so the same route as a player label):

| file | label | covers |
|---|---|---|
| `game9601` EN | `9601 город EN` | lowercase Cyrillic on EN |
| `game9602` RU | `9602 Город RU` | uppercase Cyrillic letter on RU |
| `game9603` EN | `9603 Ёлка EN` | `Ё` and Cyrillic on EN |
| `game9604` RU | `9604 Бой RU` | uppercase Cyrillic letter on RU |

The writer refuses `Ё` on RU (DIV-1339 collision set), so no RU file carries it. The owner types `Ё` in the original's SAVE dialog on RU in the README's step; that run is the open acceptance.

Engine fix. After a native city is written and loaded cold, school Train for a companion mage did nothing and posted no message. `trainOriginalCityFighter` compared the loaded member with the replayed baseline, and only `SpellbookPresent` differed (the loader leaves it unset, the baseline sets it). The comparison now takes that flag from the member with the other live views. One line in `pkg/game/originalhuman.go`.

## Proof

- `TestReleaseCitySchoolTrainsNativeMageAfterColdLoad`: native arrival, F2 SAVE, cold LOAD, Train debits the price and raises the skill, second F2 SAVE and cold LOAD keep both. Without the line the Train does nothing and the test fails.
- `TestKitOwnerCityM10` and `TestKitOwnerCombatM10` build each file, write it to `AGAINROM_OWNER_KIT_OUT` (a temporary directory when unset), cold LOAD it through the main menu in a fresh front end and assert the claims above. City: next action opens the shop, lists the tavern's hired squads and enters mission 120 with 14 members and 8 ticks. Combat: 96 ticks after LOAD the delivery has landed and the area effect still stands.
- `TestKitOwnerReceiveM10` loads any SAV named by `AGAINROM_OWNER_KIT_LOAD` in its own process and prints the same fields. The kit's `receive-resaves.sh` runs it over the kit files and the owner's resaves.
- Receipts and hashes: `review/owner-sav-m10-kits/`.

## Open debt

- Original-game acceptance of all four files is Unknown until the owner runs them.
- Kit C (a mission-150 save before the victory) is not built. With the final script unit (188) wounded to 5 HP and the hero beside it, a 600-tick run after cold LOAD with an Attack command dealt no damage and the unit's HP rose from 5 to 2000. The cause was not investigated, so a state one blow from victory is not shown to be reachable by ordinary action.
- Chapter 120 is reached through `Won` for each earlier main mission, not by playing the missions (the same controlled prerequisite the roster and group witnesses use).
- DIV-1695 to DIV-1698 are unused.
