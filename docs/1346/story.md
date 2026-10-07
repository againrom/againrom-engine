# Spell target filters

## Intent and authority

A mod's `data/spells.toml` row can decide who a spell may target and hit: heal a hostile unit, choose which sides an area spell hits, and let a damaging spell name its caster. This is step 3 of the spell mod plan (inventory `docs/1299/inventory.md` on branch `story-1299-spell-mod`, "Story split" row 3). Owner rulings: a mod-allowed heal on an enemy only heals, with no diplomacy, reputation or hostility change and no friendly-act side effect; skill training stays the original spell's; the game without a mod is byte-identical. Differences from ROM1 are `DIV-2348` to `DIV-2351` in `docs/divergences/mods.md`.

## As built

Keys, in `[[spell]]` tables, next to the step 1 keys (`docs/1341/story.md`).

| key | domain | applies to |
|---|---|---|
| `heal_hostile` | `true`, `false`, or `"@setting"` with an integer setting of 0 or 1 | the row that heals |
| `self_cast` | as above | a damaging row, area rows included |
| `area_hits` | `"all"`, `"not_own"`, `"hostile"` | an area row |

`all` is the original and the default of every row. A key on a row without its arm is refused naming mod, file and line (`ModSpellRefusal`), and two mods setting one key of one row are refused naming the first, as for every other key.

Where each reaches the game. A row is a `sim.SpellRule`; the three fields are `HealHostile`, `SelfCast` and `AreaHits`. `mapload.SpellRules` sets them after the table is built, so every consumer reads one table.

- `heal_hostile`: the apply arm (`ordinaryEffectPayload`) no longer returns before healing for a hostile pair, which serves book, scroll, weapon and script casts; `pointAttribution` skips a hostile heal so no kill credit is written; the unbidden choice (`autoCastTarget`) admits a hostile hurt unit as tier 3, after own units, locked allies and neutrals. No relation, attacker notice or credit is written.
- `self_cast`: the two book-cast gates (`castSpell` release and `bookSpellRefusal` admission) no longer refuse a damaging row at its caster. The cast cursor already shows over any visible unit; what it cannot admit is decided by that one refusal, so a click is refused or cast as the row says. A weapon release still refuses its wielder.
- `area_hits`: `areaHitAllowed` filters each unit at the area walk (`applyAreaCells`) and at the walk of an area restored from a SAV (`applySavedAreaPayload`); the damage branch of the restored walk reaches the first.

Persistence. The filters are re-derived, not saved: the SAV writer is unchanged and holds no filter. A SAV loads only under the mod set its mark names, and the mod rebuilds the table. The world byte form (`MarshalBinary`) also carries the table, so the filters ride in bits 5 to 7 of the spell record's flags byte: bit 5 is `heal_hostile` on a healing row and `self_cast` on a damaging row, bits 6 and 7 are the `area_hits` value (0 all, 1 not_own, 2 hostile) on an area row. Every world written before has those bits zero, so it decodes unchanged and an unmodded world encodes the same bytes. `formatVersion` stays 95: the inventory's 96 is taken by the auto-healing continuation form, and no new record is needed. A bit set on a row without its arm is refused as an undefined bit. A queued delivery stores its rule as JSON and the loader refuses a record that does not re-encode to the same bytes, so the three fields carry `omitempty`: a delivery saved before has no filter keys and re-encodes identically, and a filtered delivery keeps its filters.

## Proof

- `pkg/sim` (`spelltargets_test.go`): a hostile and a neutral heal restore health, pay mana, leave the relation matrix, attacker notice and kill credit untouched; the unflagged heal is still refused and paid; the flag reaches a scroll and a weapon release; the unbidden choice takes no enemy without the flag, the lone hurt enemy with it, and the own ally before the enemy; a damaging self cast is refused without the flag and admitted, paid and damaging with it, leaving relations unchanged; each `area_hits` value hits exactly its sides among own, owner's ally, hostile and neutral units; a blast in flight keeps its filter across the byte form and ends with the same hash; the three fields round-trip the byte form, a rule without filters writes no filter key in a queued delivery, a world with filters encodes differently from one without, and a filter on a row without its arm is refused.
- `pkg/mod`, `pkg/modrt`, `pkg/mapload`: each key parses with its line, `@setting`, refusals, duplicate keys across mods, each key reaching its own row only, `false` changing nothing, and a key on a row without its arm refused at the key's line.
- Release, EN and RU (`pkg/game/modtargets_release_test.go`, gated): a fixture mod written inside the test sets the keys. A self cast of Fire Arrow hurts the caster; Heal on a wounded hostile unit raises its health, costs mana and leaves the participant hostile both ways; a SAVE and cold LOAD under the same mod rebuild the flags and cast the heal again; a game without the mod refuses the SAV. Fire Ball with `area_hits = "hostile"` hurts a hostile unit beside the party and neither the caster nor the ally; the SAV taken while the cast is in flight, loaded cold under the same mod, blasts the same way. Loss controls: the installed game refuses a damaging self cast, pays and does not heal a hostile heal, and the same blast without the key hits all three.
- Unmodded identity: the existing pinned world hashes, the full `go test` and the EN and RU release suites pass unchanged.

## Open debt

- The original's behaviour for a heal on a hostile unit, an area filter and a damaging self cast was not observed (`DIV-2348` to `DIV-2350`).
- A building on a covered cell is hit whatever `area_hits` says, and `self_cast` has no weapon-release form.
- Not built here: ray rule, formula tables, live hooks (stories 4 to 6 of the plan).
