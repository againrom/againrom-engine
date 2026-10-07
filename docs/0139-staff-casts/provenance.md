# 0139 — provenance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (a weapon may carry a spell id and a level) | `MAGIC-SPELLHOP-023`, `MAGIC-ITEM-007` | High |
| FR-1a (the token names the spell, the `:n` is the level) | `MAGIC-SPELLHOP-023` — **open in the source**, see *Ours by choice* | — |
| FR-2 (the four-part trigger, and its two test sites) | `MAGIC-AUTOCAST-020` | High |
| FR-2a (the caster predicate is a mana pool) | `MAGIC-AUTOCAST-020` (`OR AL,0x6` gated on the streamed `ManaMax` column), `MAGIC-STAFF-022` | High |
| FR-3 (the strike is never called; the cast replaces it) | `MAGIC-AUTOCAST-020` | High |
| FR-4 (no to-hit roll, no absorption, no weapon damage) | `MAGIC-AUTOCAST-021` | High |
| FR-5 (the melee reach test does not gate the cast) | `MAGIC-AUTOCAST-021` | High |
| FR-6 (a state-`0xe` item is destroyed at release; a staff is not) | `MAGIC-AUTOCAST-020` | High |
| FR-7 (the level is the power, and the item power is unclamped) | `MAGIC-ITEM-007` (as amended by EXP-0066) | High |
| FR-8 (damage from power: `f = power/30 + 1`, min and max scaled) | `MAGIC-POWER-004`, `MAGIC-DMG-005` | High |
| FR-9 (no mana is charged) | `MAGIC-CAST-003` composed with `MAGIC-AUTOCAST-020` — see below | High, by composition |
| FR-16 (event-driven item training, slot selection and signed persistence) | `MAGIC-ITEMTRAIN-116`, `HERO-ITEMSKILL-096`, `ITEM-CASTSTATE-056`, `SAV-HEROXP-063`, `SAV-HEROSKILL-064` | High |
| FR-17 (surviving attribution, delayed death and serialized fields) | `MAGIC-ITEMKILL-117`, `MAGIC-ATTRGATE-118` | High for writers and gates; Medium for history-dependent recipient |
| FR-10 (the generator hands a caster a staff and only a staff) | `MAGIC-STAFF-022` | High |

**FR-9 is a composition and is recorded as one.** `MAGIC-AUTOCAST-020` states the arm's write order:
`actor+0x64`, `actor+0x68 = weapon`, `actor+0x54 = 0xd`, and *then* `R0268` validates.
`MAGIC-CAST-003` reads `R0268` whole and its second instruction is `if (caster+0x68 != 0)`
skip — "a cast from an item". The field the trigger writes is the field the economy tests, and it is
written first, so the subtraction is not reached. Neither claim states the conclusion; both legs are
High and each is an instruction-level reading, so the composition is carried at High and named here
rather than asserted in the contract.

## Ours by choice

| What the spec fixes | Why no source fixes it |
|---|---|
| FR-1a — the token before `:` names the spell and the number after it is the power | `MAGIC-SPELLHOP-023` states outright that which of `effect+0x40` (id) and `+0x42` (power) receives the suffix is **not established**; `R1000` was not read. The narrow reading is taken: a name cannot be a power and the live builder's argument is an id byte, so the name goes to the id and the number to the power. |
| FR-1b — `_` in a token stands for a space in a `Spells` row name | Nothing decoded says how the token reaches a row. The substitution is the smallest rule that resolves every shipped token; it is measured rather than assumed (AC-1). |
| FR-2b — the trigger additionally requires the id to name a row of the loaded table | The original allocates a `Spell` from any id byte. A trigger that armed on an id naming nothing would leave a caster who diverts from every strike and releases nothing — strictly worse than the swing it replaced. |
| FR-11 — the wind-up is the actor's own charge | `actor+0x134` is named as the wind-up length and no column is decoded for it. The charge is the one cadence number this tree holds for the actor and it is the staff's own row column, so a cast costs what the swing it replaced would have cost. |
| FR-12 — the cast's admission distance is the spell row's own max range | `MAGIC-AUTOCAST-021` says the admission changes hands and is `R0268`'s; it does not state the comparison. 0127 already admits a commanded cast by the row's max range in Chebyshev cells, and this reuses it rather than inventing a second rule. |

## Research correction

**FR-16 is now decoded rather than authored.** EXP-0191 confirms the owner's observed Stone Curse
case and supplies the missing cadence, amount, fighter, Poison, target and refusal boundaries. The
former per-release half-mana implementation is superseded.

The research correction matters because the former implementation paid a release-shaped mana-cost
award. ROM1 pays no such item award: separately accepted effects produce proportional damage
awards, and a later death consumes mutable attribution that can outlive the event which wrote it.
Class selection occurs only at the common sink.

## Open — deliberately assigned no meaning

- **`MAGIC-POWER-004`'s `+ power/30` cast-range term.** Not modelled anywhere in this tree,
  including 0127's commanded cast. It is zero for every shipped staff level below 30 and this story
  does not add it on one path only.
- **The `Delivery System` flight delay** (`MAGIC-CAST-003`: `distance / SpellEffectSpeed`, and Fire
  Arrow ships `2` and `200`). A weapon-borne cast lands on the tick it is released, exactly as a
  commanded one does.
- **`Distribution system`, area shapes, and everything a non-unit-targeting row does**
  (`MAGIC-TARGET-017`, `MAGIC-SHAPE-008`). A row that does not target a unit is refused, not shaped.
- **Elemental school, resistance and protection.** `SpellRule.School` is read only to name the skill
  a cast trains — a commanded one since the skill-award story, a weapon-borne one since FR-16. As an
  **element** it is still read by nothing: no resistance, no protection, no damage term, as 0127
  left it.

## Removed

Nothing was dropped from an earlier revision of this contract.
