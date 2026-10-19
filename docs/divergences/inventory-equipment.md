# Divergences -- Inventory equipment

1 row(s). Part of the split described in `docs/DIVERGENCES.md`.

## Divergences

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-3364 | inventory / the Spell of a spell weapon taken off | - | `ITEM-187` (High / Medium): removal (`R0853`) deletes the Weapon's Spell and writes 0 at `+0x80`; the next equip builds a new Spell from the `castSpell` Effect | In the first game every removal route deletes a classed Weapon's Spell and leaves it holding none (`World.takeOffWeaponSpell`, `takenOffWeapon` in the town): the source-operand removal, the saved-object and plain unequip, the Weapon displaced by an equip into its slot or by a two-handed equip, a drop from the worn slot, a death's loot (saved-object and plain), and in the town the shop doll unequip, an unequip to the shop table and the city-topology unequip and wear-displacement, whose Item drops its Spell edge. A SAVE writes the carried Weapon without a WeaponSpell. The next equip builds a new Spell for any owner. The second game keeps the Spell as it stands | FIDELITY-DEBT | The engine follows `ITEM-187` on every removal route found | No remaining mismatch. A new removal route must delete the Spell | CLOSED |
