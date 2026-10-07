# Item-borne damage-kind resistance

## Intent and authority

Decide whether the original lets an item or an effect raise weapon damage-kind resistance, and bring the engine and ledger to the answer. Authority, snapshot k115: `ITEM-EFFARM-146` (High for the 46 arm bodies, Medium for the effect-level statement), `ITEM-EFFKEY-147` (High), `ITEM-EFFSYM-148` (High), `ITEM-EFFSHIP-149` (Medium), `ITEM-ARMFOLD-033` (High / Unknown), `HERO-MODDK-161` (Medium), `HERO-DKIDX-162` (High / Unknown), `HERO-FOLD-033` (fold stands, item-or-effect consequence narrowed).

## As built

- The original has no item or effect writer of the damage-kind block `+0x10e..+0x113`. Its writers are the `+god` order, two spawn routines and the SAV archive copy.
- The engine already matches. `FoldWear` adds defence and absorption only. `ApplyItemEffects` routes kinds 21..25 to `Mod.Protection` and nothing to `Mod.Resistance`. No engine code changed.
- The damage path reads `Resistance[XPSlot-1]` for active slots 1..5 and no byte for slot 0. Shipped `Weapons` attack types are 1..5, 11 and -1; the engine's active slot is the attack type when it is 1..5 and 0 otherwise.
- `DIV-365` is closed with the claim grades above. `DIV-1659` cites `ITEM-EFFKEY-147` and `ITEM-EFFSYM-148` for the arm-to-word choice, which matches the engine's offsets for kinds 16, 19 and 21..24. `DIV-1835` records the slot-0 byte.

## Proof

- `TestNoItemEffectKindWritesDamageKindResistance` (`pkg/mapload/itemresistance_test.go`): every accepted effect key in every operand mode, fighter and non-fighter, leaves `Mod.Resistance` zero.
- Existing: `TestFoldWearSumsArmourAloneAndZeroesProtectionAndResistance`, `TestEveryWeaponKindSelectsItsOwnResistanceByte`, `TestReleaseMission90BladeResistanceChangesOnePhysicalBlow`.

## Open debt

- `DIV-1835`: a nonzero `+0xce` (active slot 0) carried by a SAV never reduces a blow.
- Unknown in the claims: values in a SAV-loaded modifier block, the unread `vt+0x50` callees, the `+god` origin, other `vt+0x4c` callers, the attack-type -1 row, and active slots 6..9 (none shipped).
