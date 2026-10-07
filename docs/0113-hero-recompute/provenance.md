# Provenance — 0113-hero-recompute

Pin: research submodule at `54e154c` (`master` at the story boundary).

## What is decoded, and at what confidence

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `HERO-ORDER-014` | High | The spine. One virtual method, and the four points where the order is load-bearing: caps first, pools before the skills are restored, the block cleared between defence's inputs and defence's write, the equipment fold between the bare values and the clamps. FR-4. |
| `HERO-CAP-015` | High | Step 0, destructive, `min(stat, 50 + modifier)`, 100 the ceiling, 50 what an untouched statistic reads. Already in `capStat`; FR-5 extends it to all four. |
| `HERO-HP-005` | High | The health maximum's three steps, the health-column gate, the class multiplier, and that every intermediate is truncated and stored narrow so the truncations compound. FR-7. |
| `HERO-MP-006` | High | The mana maximum, the same three steps on Spirit, the inverted class multiplier, the non-class-conditional `Spirit x 2` base, and the zero-column arm. FR-8, FR-16. |
| `HERO-CLASS-013` | High | Exactly two class-conditional edges inside the derive, both on the pools' experience term. FR-7, FR-8, DD-2. |
| `HERO-XP-010` | High for the two tables and both drivers; **partially retracted** | Three fields — the level, the per-slot experience dword at `actor+0x1cc + 4i`, and the recomputed total — and two routines between them: `R0899` level to experience, `S(n) = ftol((1.1^n - 1) x 1000)` cached per slot and summed, and `R0905` the inverse. The retracted clause is the "nothing increments `+0x130`" enumeration; what stands is that a hero's total is recomputed, not accumulated. FR-6. |
| `HERO-SKILL-009` | High | Six slots at `actor+0xa8 + 2i`; slots 1 to 5 restored from a base copy with a bonus added and clamped to `[0,100]` twice, **slot 0 in neither loop** but in the experience sum. FR-6a, AC-13. |
| `HERO-DERIVE-034` | High | The damage pair and to-hit: the spread computed and the base copied from it, the skill terms' asymmetry, the three constants. FR-9. |
| `HERO-STATDMG-036` | High | Only Body reaches damage and only Body and Reaction reach to-hit — an exhaustive read census inside the sole writer. FR-9. |
| `HERO-BARE-037` | High | The unarmed character is the same routine with the active-skill gate closed. FR-9. |
| `HERO-RESIST-012` | High | The five elemental protections share one source, Spirit halved truncating toward zero; the five damage-kind resistances are never re-derived; the clamp `clamp(min(spirit/2 + 70, p), 0, 100)`. FR-11, FR-15, AC-2, AC-8. |
| `HERO-ARMOUR-018` | High for the field identification; **Medium** for the sheet reconciliation; the `УРОН` line **retracted** | Defence is the block's first word and absorption its second. FR-10. |
| `HERO-FOLD-033` | High | The fold adds fourteen fields: defence, absorption, six protections, six damage-kind bytes. DD-3, DD-8. |
| `HERO-FOLD-035` | High | Eight plain `ADD`s and one assignment; **no multiply anywhere on the path from an item to a damage number**; no armour or shield routine contributes to the damage modifier pair. FR-13. |
| `HERO-CLAMP-030` | High; **Medium** on "no blow in play reaches the negative path" | The damage-kind byte is read unsigned by the resolver and its slot 0 is the index an attacker with no melee weapon uses. DD-8. |
| `HERO-SHEET-038` | High | The character sheet prints `[base, base + spread]`, so a figure here is directly comparable with what the owner reads on screen. |
| `HERO-KILL-027` | High; **Unknown** which skill absorbs a gain | Located, not built: the gain path exists and attaches to the skills, which is why the experience sum is the only experience this story writes. |
| `HERO-EQUIP-017` | — | Read for the weapon's construction; nothing taken beyond the confirmation that a weapon's numbers are already resolved before they reach the fold. |
| `HERO-COMBAT-011`, `HERO-DERIVE-034` | — | Read to confirm the eight numbers already in `Combat` need no widening. |
| `ITEM-DEF-002` | **Unknown** on the armour column | The named gap: which `Data.bin` column fills `Armor::Equip`'s `item+0x52` and `Shield::Equip`'s block at `shield+0x50`. FR-14, DD-4. |
| `DAT-SCHEMA-007`(a) | — | Armors, Shields and Weapons share one 18-title array, so the missing fact is the fill routine, not the titles. |
| `MAGIC-RESIST-006` | — | A resistance applies to the elemental component only; this build has no elemental damage, which is why the derived set is a loader value. P-1. |

## What is ours by choice

- The Go names `Profile`, `EquipMod`, `Loadout`, `Derived`, `CombatBlock`, and every panel label.
- Three panel rows rather than eleven, and the space-separated five-number form each states.
- The panel field numbers 34, 35 and 36, following this repo's own allocation rule.
- Placing speed and sight after the pools. Nothing constrains their position but the caps.
- `Derived.ClampPools` as a method rather than two fields on the value.

## What is open

- **Which shipped column fills an armour's or a shield's contribution.** The whole armour arm of the
  fold carries zero until it is answered. No value is invented for it.
- **Which archetype sets the class flag.** `HERO-CLASS-013` names `actor+0x4c` bit 2 the
  fighter/mage flag and reads the pools as `fighter ? 2 : 1` and `fighter ? 1 : 2`; the same claim
  records `R0842` setting that bit when the actor has a nonzero `manaMax`, which points the
  other way. No caller in this tree sets the flag, so nothing here depends on the answer.
- **What health and mana columns a generated campaign hero is born with.** Unanswerable from the pin:
  he has no shipped row. This is why the two pools are produced and not wired, and why a party
  member's health is still `SpawnHP`.
- **The margin between the original's `log` and ours.** `pow11` carries a measured margin for its own
  chain; no equivalent measurement exists for the pools' logarithm, and one is owed by whoever wires
  the pools to an entity.
- **Which skill a gain is credited to** (`HERO-XP-010`, `HERO-KILL-027`), and **the arithmetic of
  `R0905`**, the experience-to-level inverse. Both belong to the gain and loss paths, which no
  term of this recompute calls. The inverse is named as a seam rather than derived algebraically: a
  plausible closed form is not a decoded one, and a fit through an unread model is the failure
  `HERO-ARMOUR-018`'s retraction was recorded for.

## What was removed

Nothing. `Hero.Derive`, `Hero.Speed` and `Hero.Sight` keep their names and their contracts; only
their bodies move into the one implementation.
