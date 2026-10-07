# Shop potion kinds through city SAVE

## Intent and authority

Every potion kind the EN and RU shops sell is used in the city, survives two F2 SAVE and cold LOAD cycles, and has the same next-mission state as an uninterrupted control. Owner rules: SAV is the only format with one producer; a field is written from current state; nothing refuses a save.

Claims: MAGIC-CONSUME-142 (modes, caps, the mage gate on mana), MAGIC-CONSUME-143 (shared id zero), MAGIC-CONSUME-144 (stored remaining timer, applied modifier, one removal), MAGIC-ATTACH-016. SAV-1116 covers the derive that folds the modifier words. Original-runtime reload of a city-drunk potion stays Unknown.

## As built

The installed shop sells six potion rows, identical on EN and RU (`shopPotionStock`): Health Regeneration (kind 8, +100, 960 ticks), Medium Healing (+30 health), Big Healing (+100), Mana Regeneration (kind 11, +100, 960), Medium Mana (+30), Big Mana (+100). The other installed potions (Antipoison, Fighter and Mage Bonus, attribute potions) are not sold.

No engine defect was found. The existing production path already carries all six: the one-shot kinds change the pool words in the Human, the timed kinds write the applied regeneration into the Human modifier word (offset 10 health, 14 mana) and the remaining timer into the current city policy record, and mission entry attaches the record once. A mission SAVE under a carried timed potion also writes the modifier word, so cold LOAD and expiry agree (no `savedModifierEffects` entry was needed for the two regeneration kinds).

## Proof

`TestReleaseCityPotionKindsF2SAVAndMissionExpiry` runs each kind as a subtest on a native arrival and on the original city (SHA-pinned `2026-08-15/game0010.sav`). It buys through the shelf, table and Buy, uses the potion from the pack, reads each ordinary SAV (effect record, Human health, mana and regeneration words, empty Human Effects list), cold-loads twice, enters the next mission and compares World hashes every tick with the uninterrupted party: 961 ticks for timed kinds (expiry at tick 960, one attachment, none at 961), 64 for one-shot kinds. Mana kinds use Reniesta, a mage; health kinds use the first member. For the original city the one-shot kinds start from a wounded pool (health 20 or mana 10, set in the carried record).

Loss control: for every kind with an observable effect, the SAVE is written from the member as it stood before the drink while the uninterrupted city keeps the drink. The SAV read and the next-mission observation must both differ. `TestReleaseCarriedPotionMissionSaveKeepsModifierWordThroughColdLoadAndExpiry` covers the mission-SAVE route for both timed kinds.

| Kind | Origin | Before | After the drink (every SAV, cold LOAD, mission tick 0) | Expiry |
|---|---|---|---|---|
| Health Regeneration | native, original | health regeneration 0 | 100 | removed once at tick 960 |
| Mana Regeneration | native, original (mage) | mana regeneration 0 | 100 | removed once at tick 960 |
| Medium Healing | original | health 20 | 50 | none, no attachment |
| Big Healing | original | health 20 | 120 | none |
| Medium Mana | original | mana 10 | 40 | none |
| Big Mana | original | mana 10 | 110 | none |
| Medium Healing, Big Healing, Medium Mana, Big Mana | native | full pools | unchanged, one potion consumed and paid | none (DIV-1691) |

EN and RU results are identical line for line. Receipts: `review/story1260-sav-m8/focused-en.txt`, `focused-ru.txt` (22 passing subtests each, 4 native one-shot kinds have no drop control because they change nothing). `internal/gatedtests`, `internal/storyguard` (comment bytes not raised) and `internal/divledger` pass, as does the ordinary `pkg/game` package test.

## Open debt

- DIV-1691: a native city member carries no wounds, so healing and mana potions change nothing there.
- DIV-1692: the timed record lives in the engine's city policy; the Human's Effects list is empty at a city save point. Original encoding is Unknown.
- Original-runtime acceptance of a drunk potion waits for an owner kit. DIV-1693 and DIV-1694 are unused.
