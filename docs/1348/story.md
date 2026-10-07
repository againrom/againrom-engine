# Prismatic ray rule

## Intent and authority

A mod's `data/spells.toml` row for Prismatic Spray can raise the ray cap up to 100. This is step 4 of the spell mod plan (inventory `docs/1299/inventory.md` on branch `story-1299-spell-mod`, "Prismatic Spray with 100 rays" and "Story split" row 4). Owner rulings: rays never exceed the targets (the cap is only a maximum, as in the original spell, so 3 enemies in sight and a cap of 100 fire 3 rays); training stays the original spell's; the game without a mod is byte-identical. Differences from ROM1 are `DIV-2360` to `DIV-2363` in `docs/divergences/mods.md`.

## As built

Key, in a `[[spell]]` table for `Prismatic_Spray`: `rays`, an integer 1 to 100 or `"@setting"`. On any other row it is refused naming mod, file and line (`ModSpellRefusal`); two mods setting it are refused naming the first.

One rule, `SpellRule.RayLimit(power)` in `pkg/sim/spell.go`: with `Rays` set it is the cap, otherwise the original `min(power/20 + 2, 7)` clamped at 0. It replaces three copies: the selector's `uint8` limit (`prismaticVictims`), the tooltip's ray count (`weaponSpellCharacteristics`) and the ten-slot winner loop, which now scans to the limit and stops when no candidate is left. The victim list is the primary, then the caster group's candidates in the original score order, so a cap above the candidates fires one ray per candidate. The drawn figures and queued deliveries follow the victim list, so the draw needs no change.

Without the key `Rays` is 0 and every victim list, tooltip, delivery and byte is as before. The only change for an unmodded game is for a power at or below -40, which the old `uint8` limit wrapped; no shipped path produces one.

`mapload.applyModSpellTargets` sets `Rays` and zeroes the row's `Radius` (a point row's radius is never read). Persistence: in the world byte form a Prismatic Spray record with no area and flags bit 7 set holds the cap in the radius byte (`raysFlag`); the decoder refuses a zero cap, a cap above 100 and the bit on another row. A queued delivery's rule JSON carries `Rays` with `omitempty`. `formatVersion` stays 95.

## Training

The original awards `round(manaCost / 2)` to the school on every apply from a book and nothing for an item cast (`training.md`). The engine pays that award once per queued ray (`preparePrismatic`, `applyPrismaticItem`), and each ray's damage pays the ordinary damage award. Unchanged here, so a cast with a raised cap pays one award per ray: 12 victims pay 12 awards where seven rays pay seven. An item cast pays none.

## Proof

- `pkg/sim` (`prismaticrays_test.go`): the rule at powers -100 to 1000 and with a cap; the tooltip equals the rule with and without a cap; a cap of 100 over 12 foes selects the primary and all 12 in distance order, one ray per victim; 3 foes give 3 rays; a cap of 5 gives 5; the cap ignores power and the unmodded spray still stops at 7; byte form and delivery JSON round trip, a capped world encodes differently from an uncapped one, an uncapped rule writes no `Rays` key, and decoder and constructor refusals.
- `pkg/mod`, `pkg/modrt`, `pkg/mapload`: parse with line and `@setting`, bounds 1 to 100, duplicate keys, refusal on another row at the key's line, the key reaching its row only.
- Release, EN and RU (`pkg/game/modrays_release_test.go`, gated): a fixture mod sets `rays = 100` and `mana = 1` (the caster has 36 mana and the row costs 80). With 12 hostile units placed around the caster, 12 ray figures are drawn and every enemy is hit; a SAVE with the rays in flight, loaded cold under the same mod, hits all 12; a game without the mod refuses the SAV. Loss control: the same fixture without `rays` draws at most 7 figures.
- Unmodded identity: the existing pinned world hashes, the full `go test` and the EN and RU release suites pass unchanged.

## Open debt

- The original's behaviour for a ray list longer than its storage was not observed (`DIV-2360`).
- A ray figure is not capped; the viewer was measured only at 12 rays.
- Not built here: formula tables, live hooks (steps 5 and 6 of the plan).
