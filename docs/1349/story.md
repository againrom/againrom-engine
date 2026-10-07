# Spell formula tables

## Intent and authority

A mod's `data/spells.toml` can replace the spell formulas, per spell and for all spells, with tabulated values frozen into `Rules` at load: the caster `power`, the damage factor, the range bonus, the duration factor and the magnitude arms. Power above 100 works. This is step 5 of the spell mod plan (inventory `docs/1299/inventory.md` on branch `story-1299-spell-mod`). Owner rulings: without a mod the game is byte-identical (world hashes, SAV bytes, tooltips, book); skill training stays the original spell's; a refusal names mod, file and line. Differences from ROM1 are `DIV-2364` to `DIV-2367` in `docs/divergences/mods.md`.

The same story fixes the book caption of Prismatic Spray, which printed `min(level/20+2, 7)` and now prints `SpellRule.RayLimit`, as the tooltip does. Unmodded text is byte-identical.

## Table form

A plain TOML integer array, not a Starlark table. A table is already a frozen `Rules` value, it needs no interpreter call at load or per cast, it takes `"@setting"` entries like every other key, and a mod author reads and diffs it as data. A Starlark table would add a second evaluation seam for no formula a 256-entry array cannot state. Live per-cast hooks stay step 6.

## As built

Keys, in `[[spell]]` tables, and the first four of them in `[global]`.

| key | indexed by | entry domain | meaning of an entry |
|---|---|---|---|
| `power` | caster skill level plus Mind, 0 to 510, up to 511 entries | 0 to 255 | the caster power |
| `damage_factor` | power, up to 256 entries | 0 to 30000 | thirtieths: damage is `min * entry / 30` |
| `range_bonus` | power, up to 256 entries | 0 to 255 | added to the row's range, the sum saturates at 255 |
| `duration_factor` | power, up to 256 entries | 0 to 1000000 | thousandths of the row's duration |
| `magnitude` (row only) | power, up to 256 entries | -32768 to 32767 | the arm's magnitude |

An index past the last entry reads the last entry. A row table wins over a `[global]` table. `[global]` has no `magnitude`, because the arms differ in meaning. Two mods setting one table of one row or of `[global]` are refused naming the first. The refusals are in `mapload.modSpellFormulaRefusal`: `damage_factor` needs a damage or heal pair, `range_bonus` a range above 0 (or a `range` key on the row), `duration_factor` a spell with a power-scaled duration, `magnitude` a magnitude arm. A table length or entry outside its domain is refused at the key's line.

Where each reaches the game. `rules.SpellFormulaSet` is frozen by `Rules.WithSpellFormulas`; `FrontEnd.Table.Rules` receives it from `SetMods` and `SetModSpells`, whichever runs last, and `mapload.TableRules` gives it to the world. The sim reads it through `...Under(r, rule, ...)` functions that return the original arithmetic when no table is set.

- Power: the cast, the book record, the popup and the saved book roots.
- Damage factor: direct, area and delivery damage, the tooltip and the popup.
- Range bonus: the cast admission, the book slot range, the popup, and the saved-root check, which accepts any reach the table can produce.
- Duration factor: point effects, the SAV record and the popup. Stone Curse and Invisibility read the table whole; Invisibility keeps its base of 3.
- Magnitude: the protections, Shield, Haste, Freezing Cloud, Slow, Poison Cloud, Light, Darkness, Bless and Curse, in the cast and in the caption (`spellCaptions` now take the rule and its characteristics).

Persistence. The tables are not in the world byte form and not in the SAV. A LOAD rebuilds them from the mod set the mark requires, and a game without the mod refuses the SAV.

## Proof

- `pkg/rules`, `pkg/sim`, `pkg/mod`, `pkg/modrt`, `pkg/mapload`, `pkg/game` (`*formula*_test.go`): domains, clamping, refusals with line, duplicates across mods, `@setting`, each formula reaching its consumer through the world, the book range cache following a range table, the tables absent from the byte form, `SetMods` and `SetModSpells` in either order, the popup, the captions and the ray rule.
- Release, EN and RU (`pkg/game/modformulas_release_test.go`, gated): a fixture mod written inside the test sets Heal (`power`, `damage_factor`), Bless (`power`, `duration_factor`, `magnitude`) and Fire Arrow (`range_bonus`). Heal restores exactly the tabled amount at power 150; Bless attaches with magnitude 77 and the tabled ticks; the popup states each number; the SAV Spell records hold the tabled reach; a cold LOAD under the same mod reproduces the popups, the Bless effect, the reach and the heal; a game without the mod refuses the SAV. Loss control: the installed game shows the original numbers. The reach is read from the world's characteristics and book slot, not from a cast at a unit, because the mission caster's sight does not cover the reach; the cast-time range check is proved in `pkg/sim`.
- Unmodded identity: the pinned world hashes, the full `go test` and the release suites pass unchanged.

## Open debt

- The original's behaviour for powers above 100 and for the tabulated forms was not observed.
- The area life formula, Control Spirit and Fire Sacrifice are not tabled.
- The popup folds a damage above the in-flight byte as before (`DIV-2327`).
- `range_bonus` is refused by a heuristic (a range above 0 or a `range` key on the row).
- Not built: live per-cast hooks (step 6).
