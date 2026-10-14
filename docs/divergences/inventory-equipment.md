# Divergences -- Inventory equipment

1 row(s). Part of the split described in `docs/DIVERGENCES.md`.

## Divergences

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-3364 | inventory / the Spell of a spell weapon taken off | - | `ITEM-187` (High / Medium): removal (`R0853`) deletes the Weapon's Spell and writes 0 at `+0x80`; the next equip builds a new Spell from the `castSpell` Effect | An actor with source operands (one loaded from an original SAV) deletes the Spell when the Weapon is taken off (`pkg/sim/sourceequipmove.go`, `replaceObjectSpell(&item, SourceItemSpell{})`). An actor without them (one of a session this engine started or wrote) keeps it: the saved-object unequip (`unequipSavedNative`) and the plain unequip move the Weapon to the pack with its Spell, and a SAVE writes that carried Weapon with its WeaponSpell. Worn again on these routes, it carries the kept Spell | FIDELITY-DEBT | Owner direction for this change kept the engine's removal as it was | An engine change that deletes the Spell on every removal route | OPEN |
