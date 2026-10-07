# Spell mod data keys and global multipliers

## Intent and authority

A mod changes spell values per spell and for all spells through `data/spells.toml`. This is step 1 of the spell mod plan (inventory `docs/1299/inventory.md` on branch `story-1299-spell-mod`, "Story split" row 1). Owner direction: MOD milestone item 4, and the seat rulings on `[num, den]` multipliers and on no live hooks in stages 1 to 5. The game without a mod is unchanged. ROM1 differences are rows `DIV-2326` to `DIV-2329` in `docs/divergences/mods.md`.

## As built

File and route. `data/spells.toml`, loaded by `game.data.add("data/spells.toml")` (`pkg/mod/spells.go`, `pkg/modrt/data.go`). The result is `modrt.Result.Spells`, stored by `FrontEnd.SetModSpells` in `mapload.ModContext.Spells`. `mapload.SpellRules` applies it to the loaded `data.Spell` rows before the `sim.SpellRule` conversion, so the world table, book learning, casts, staff and scroll casts, the spellbook popup and the SAV book records read one edited table. `-check` prints `againrom: spells edited=N global=M`.

Row keys, in `[[spell]]` tables. `target` names the row by its spell table name (an underscore reads as a space). Other keys are optional.

| key | column | domain |
|---|---|---|
| `mana` | mana cost | 0..32767 |
| `range` | max range | 0..255 |
| `radius` | area radius | 0..255 |
| `damage = { min, max }` | damage pair (the heal amount on Heal) | each 0..255, min at most max, not both 0 |
| `duration` | spell duration term | 0..4095 |
| `area_duration` | area lifetime term | 0..4000 |
| `effect = { kind, magnitude, mode, duration }` | the row's first effect record, any subset | magnitude -32768..32767, duration 0..4095, kind and mode from named lists |

`[global]` keys: `damage_mul`, `heal_mul`, `mana_mul`, `range_mul`, `radius_mul`, `duration_mul`, each `[numerator, denominator]` with numerator 0..1000000 and denominator 1..1000000. Rules:

- `damage_mul` scales the pair of every row with a pair that does not heal, Drain Life included; `heal_mul` scales Heal's. `duration_mul` scales `duration`, `area_duration` and the effect duration. No multiplier touches an effect magnitude.
- A row key is final for its field; a multiplier scales the fields no row key set.
- The product is rounded toward zero and clamped into the domain of its field, never refused. A row key outside its domain is refused.
- Classification (damaging, restorative) follows the pair after every edit. A row key may not edit the pair of a row that has none, and may not empty a pair; a multiplier that empties one clears the flag.
- Any integer may be written `"@name"`, the integer setting `name` of the same mod, so `-mod-setting` and the settings screen reach every key and multiplier part.

Refusals name the mod, file and line: unknown table or key, a value outside its domain, a bad ratio, a missing or non-integer setting, and (from `SetModSpells`, which reads the install's table) an unknown target, a pair edit on a row without one, and an effect kind with no mode. Two mods may not set one global or one field of one row; the second is refused naming the first.

Domains. Each ceiling is what the saved book slot (mana as a signed 16-bit word, range as a byte), the area radius byte and the 16-bit duration and magnitude words hold, so a SAV written under a mod states every value exactly. Mark format 2 of the inventory carries values beyond those widths; no key reaches beyond them, so the mark format is unchanged and no SAV format changed.

Book and save. A book slot copies range and mana from the table when the spell is learned (`LearnBookSpell`, map constructors), and a loaded SAV keeps the slot's stored values. A mod's values reach new games and newly learned spells; a SAV written under a mod holds them and loads only under the same mod set (`DIV-1802`).

## Proof

- `pkg/mod`: the file, each key, `@setting`, each refusal with its line.
- `pkg/modrt`: settings reach rows and multipliers, refusals name mod, file and line, two mods setting one field or one global, a mod without the file.
- `pkg/mapload`: each key changes its column alone (field count against the installed row); multipliers round toward zero and clamp; row keys are final; unmodded table equals the installed table; every effect name resolves; Poison Cloud keeps its duration word shift; world and book take the edited rows; the domain ceilings survive the book slot and `BookSlotValues`; a cast spends the edited mana, reaches the edited range, rolls the edited damage and covers the edited radius in a simulated world.
- `pkg/game`: `SetModSpells` refusals, `SetMods` keeping the edits, the spellbook popup stating the edited values.
- `cmd/againrom`: the launch refusal and the `-check` line.
- Release, EN and RU (`pkg/game/modspell_release_test.go`, gated): the installed table under a fixture mod (written inside the test) equals independent arithmetic on the installed columns for every row, columns no key names are unchanged, a second unmodded front end equals the installed table, the popup states the edited values, an in-mission cast spends the edited mana, the SAV's Spell record holds the edited range and mana, the cold LOAD under the same mod restores them, and a game without the mod refuses the SAV.
- Unmodded identity: the existing pinned world hashes and the EN and RU release tests pass unchanged; the seam returns before touching a row when no edit is declared.

## Open debt

- `damage` saturates at the payload byte for rows delivered by a queued transport once the power factor is applied; the popup states the unsaturated record (`DIV-2327`).
- `effect` kind, mode and magnitude reach only the rows whose arm reads the row's effect record; magnitude formulas stay code until the tabulated formula story. A radius beyond the map edge wraps and draws one burst (`DIV-2328`).
- The original game's load of a SAV with edited book values was not observed; no owner check kit is prepared in this step (`DIV-2329`).
- Not built here: ray rule, target filters, large-radius cell walk, formula tables, live hooks (stories 2 to 6 of the plan).
