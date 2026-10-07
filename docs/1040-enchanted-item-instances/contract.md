# Story `1040` — enchanted item instances, authored loot, and shop generation

**Dispatch-ready seat draft, rewritten 2026-08-24.** Implementation master `a901fe15` pins research
`49b154de`, which contains all three prerequisites. The lane copies this to
`implementation/docs/1040-enchanted-item-instances/contract.md` and commits that contract before
production code.

## Result

An enchantment belongs to an item instance. The enchantment remains attached while the item is
equipped, carried, dropped in a sack, picked up, moved between characters, bought, sold, saved, and
loaded.

An eligible enemy mage receives the enchanted staff authored by the shipped `Humans` row. The mage
casts through that staff. On death, the same staff instance enters the corpse sack under the original
death-drop rules. A party member can pick it up, equip it, and cast the same spell at the same power.

The Magic Items shop shelf generates enchanted armor and weapons through the decoded selection,
budget, retry, probability, and price rules. Differently enchanted equipment remains different shop
cells and different carried objects. Equal-code Potions merge here only when ordered effects and
stored price also agree. This deliberately differs from ROM1's effect-blind Potion equality so an
imported enchantment cannot be destroyed. Selling an enchanted item uses the price stored on that
instance, including its enchantment contribution.

Authored mission loot and actor stock construct the effects named by their map records. The item
keeps those effects through every inventory and persistence path.

Mission 40's `Treasure!` has a unit value of 10,000 instead of 0. It has no invented enchantment.

## Owner scope and story size

Owner directive, 2026-08-23: enemy mage staves, possible dragon drops, mission 40's Treasure, shop
enchantments, enemy enchantments, and enchanted mission sacks belong to the same story.

This story has more than five behaviours, reaches hashed simulation state, changes the serialized
byte form, and touches all nine domains. It would normally be split. It remains one implementation
story because one item-instance representation must cross every producer and consumer atomically.
Landing a producer before the container, drop, shop, or save paths would create an item whose effect
is silently removed at the next boundary. The research is split into three independent experiments;
the implementation is not split.

## Facts measured for this draft

The current implementation represents carried items as `ItemStack{Code uint16, Count uint32}`,
ground sacks as `[]uint16`, equipped slots as item codes, and shop rows as one code, price, and count.
The map loader resolves a staff's `{castSpell=...}` suffix into `Entity.WeaponSpell` and
`Entity.WeaponSpellLevel`. The equipped item itself still carries only its code. A corpse therefore
drops the code and loses the spell attachment.

Both lawful installs contain 46 `Humans` equipment cells naming a staff with a `castSpell` suffix.
The powers use 18 values from 1 through 99 (`ITEM-CASTSTATE-056`). The four `Dragon` rows instead
name `Flame Thrower`. `ITEM-SUIT-035` gives `Flame Thrower` a `sutableFor` value of 0, and
`ITEM-DEATH-012` states that a weapon with this value is not moved into the death sack. `EXP-0227`
found no Dragon staff or enchantment producer in its enumerated authored, reference and death
populations; computed image targets remain outside that Medium negative.

The EN and RU copies of `40.alm` have the same SHA-256. The ground record at cell `(96,84)` contains
item code `0x0e1f`, class 14, row 31. Its type-9 link is 0. `MagicItems[31]` is
`Quest Item31` with parameters `[10000, 1]`; the EN item-name table names code `0x0e1f`
`Treasure!`. The value is a base data value. The record does not attach a type-9 effect.

## Published research gate

All three prerequisite experiments landed after one independent adversarial pass each. A Medium or
Unknown result remains Medium or Unknown here; the implementation contract does not promote it.

### `EXP-0225` — item effect object and equipment-cell grammar

`EXP-0225` publishes `ITEM-EFFGRAM-070` through `ITEM-EFFSAVE-077`. It establishes:

- the effect-list element layout, order, identity, copy, equality, and serialization rules;
- the complete `{...}` equipment-cell grammar used by shipped `Humans` and `Units` rows;
- every shipped effect key, its operand widths and signs, and its equip, unequip, recompute, and
  combat consumers;
- the weapon-owned `castSpell` attachment and its relation to the general effect list;
- the two `ITEM-STACK-003` stackability tests: an empty effect list or literal item-kind value 3;
  the Potion prefix writes 3, so equal-code Potions remain stackable with effects and equality
  returns before comparing those effects;
- the behaviour of unknown, duplicate, invalid, and over-budget effects;
- the representation found in original saves and the rule used when an item is cloned or split.

`EXP-0225` enumerates the complete shipped equipment-cell population. Each root has 2,386 cells,
260 braced cells and 266 accepted Effects. The combined count is 520 braced cells, not 520 per root;
the malformed nested cell makes 522 opening-brace characters across both roots. Each root has 46
Human cells and two Unit cells with `castSpell` attachments (`ITEM-EFFPOP-071`).

### `EXP-0226` — Magic Items shop effect generator

`EXP-0226` publishes `SHOP-EFFPOOL-061` through `SHOP-EFFSEED-072`. It establishes:

- the effect candidate population and all class, item, tier, and budget admission rules;
- the effect payload produced for each candidate and the cost added to the item price;
- the exact random draw order, bounds, retries, and reject-and-redraw behaviour;
- the inclusive additional-effect gates `50/101` and `25/101`;
- the interaction between the `2 * ceiling - basePrice` budget and the 9,999,999 price clamp;
- whether re-entering, replaying, or loading town reseeds before generation.

The complete two-root shelf pool has 367 triples: 193 Armor, 36 Shield and 138 Weapon. The selector
reaches 273 weighted rows and 36 kinds. Generator-family exclusivity remains Medium because the
published absence search is scoped to direct calls. Exact reseed timing and intervening draws remain
Unknown.

### `EXP-0227` — authored enchantments and death-drop corpus

`EXP-0227` publishes `ALM-EFFREC-071` through `ALM-M40-074` and `ITEM-AUTHCAST-086` through
`ITEM-PRODUCER-091`. It establishes:

- the type-9 record grammar and the exact operation performed by a type-8 element's 1-based link;
- whether the link applies identically to ground sacks and actor stock;
- every linked type-8 element in every shipped EN and RU campaign map;
- the join from the 46 Human staff rows to shipped placements, owner, template name, NPC suppression,
  death eligibility, and resulting dropped item;
- all Dragon placements and every possible equipment, stock, ground-loot, script, and death producer
  that could create a staff or another item at the dragon's death;
- mission 40's class-14 row 31 price and effect state, including the zero type-9 link;
- the general price rule for positive and negative `MagicItems` values.

Each root has 74 valid campaign links. The shipped single-player campaign has 49 death-eligible
Human-staff placements and 27 `NPC`-suppressed placements. Four Dragon rows and all 21 campaign
placements per root carry only the innate `Flame Thrower` with suitability zero. The enumerated
authored and static-reference producer populations contain no Dragon staff or enchantment death
producer; computed image targets remain outside that Medium negative. Mission 40's `0x0e1f` is a
plain class-14 row-31 item with value 10,000 and link zero.

## Behaviour contract

### B1 — one canonical item instance

The simulation has one canonical item-instance value. It contains the base item code, an ordered
effect list of exact `(kind u8, mode u8, operand u32)` values, and signed stored price/state where a
published producer supplies it. Duplicates and order are identity. Container quantity belongs to the
stack or cell that owns instances; it is not hidden inside the effect payload.

Weapon casting has one tagged source: `None`, `Item`, `Innate`, or `Legacy`. For `Item`, the equipped
instance's cast-spell effect is authoritative and any `Entity.WeaponSpell` pair is a validated derived
cache. For `Innate` and `Legacy`, the tagged pair is authoritative and no item effect is invented.
Exactly one source is active. Equip, unequip and rearm replace the tag and source atomically, so a
derived pair cannot disagree with the equipped weapon.

Copies returned by public readers cannot alias simulation state. Canonical ordering is the original
ordering. The implementation must not sort or deduplicate effects unless a research claim requires
it. The story assigns no semantics to original `Item+0x47` or the transient Effect field not carried
by the published serialized form.

### B2 — every container and transfer preserves the instance

Carried inventory, all equipped slots, ground sacks, actor stock, party records, shop rows, drag
state, and original-save import paths that already import an item carry the complete item instance.
This includes the parallel `mapload.PartyMember` and `mapload.Carry` forms, shop table and drag
projections, party gob fields, and original-save `sav.Piece` conversion; none remains a code-only
shadow that can later overwrite the canonical instance.
The following operations preserve the effect payload and price:

- spawn and party construction;
- equip, unequip, displacement, and transfer between characters;
- manual ground drop and pickup;
- death drop and same-cell sack merging;
- script give, take, and drop operations where the authored operation supplies an item;
- shop generation, buy, sell, drag cancellation, shelf folding, and sorting;
- campaign transition, party cloning, save, load, and migration;
- the school's first-training lazy `Carry` construction, including carried and worn canonical
  instances as well as their code projections.

A transfer that changes only location must not reconstruct the item from its code.

### B3 — stacking preserves instance identity

ROM1 defines an item as stackable when its effect list is empty or its item-kind byte is literal 3.
Its equality first compares code, returns equal for two stackable objects without reading effects,
returns unequal for exactly one stackable object, and otherwise compares ordered effect lists.

This build deliberately narrows the Potion arm: equal-code Potions merge only when their ordered
effects and stored prices also agree. Plain identical Potions still stack. Effect- or price-different
Potions remain distinct, because the shipped `game0002.sav` carries a three-count `Potion Health
Regeneration` with a real Effect and the owner-directed item-instance invariant forbids discarding it.
`DIV-369` records this DEVIATION from ROM1's effect-blind Potion equality.

The shop must not collapse differently enchanted non-stackable equipment. Its cell identity carries
the complete instance and its transaction price, but price is not added to the decoded item-equality
predicate: `R0928` does not read it. A cell count represents instances admitted by this
story's preservation rule and the shop's transaction-price boundary. `DIV-322` is revisited because
generated stock and return-to-shop insertion use different original paths while this build folds the
backend shelf.

This story closes `DIV-235`: the corrected lazy training constructor is the last production
container constructor, and it preserves both canonical halves.

### B4 — effects apply from the equipped item

The story implements the complete consumer union reachable from the 16 shipped equipment keys and
the shop selector's 36 reachable kinds. Diagnosed no-op kinds remain no-op; the cast-spell and
teach-spell source kinds retain their distinct published roles. The contract does not promote all
50 dispatcher slots into executable equip effects.

Recompute restores the whole derived entity state from effect-free base values, then applies every
equipped instance once in stored order. Combat fields, Mind, Reaction, maximum health, speed, scan
range and protections cannot split across separate partial recomputes. Repeated equip, load, or
recompute must not accumulate an effect twice, and clamps occur only where a decoded rule clamps.

Kinds 44 through 48 share one ordered secondary-damage triple: low-byte base, high-byte spread and
the Fire-through-Astral selector `kind - 44`. Every occurrence replaces the whole triple, including
a later distinct kind; no two selectors coexist. Combat applies that one component against exactly
the selected protection. Its hit gate is the original third-component gate: a successful physical
hit admits the secondary component, and an empty physical base/spread pair admits it even when the
ordinary hit roll misses. A nonempty physical pair that misses suppresses both components. The
secondary spread draw occurs only after that gate. Construction, partial rearm and form decode
reject a selector outside `0..4`, and the form rejects residue that could encode a second component.

Weapon spell casting reads the equipped weapon instance. Mana suppression, spell id and power follow
the active research rows. No destructive item kind or post-cast item mutation is inferred without a
published consumer. The existing enemy AI uses the resulting weapon cast without a second authoring
path.

### B5 — enemy mage staff lifecycle

The equipment-cell parser constructs the complete authored staff instance for a Human mage. Both
roots' positive integration witness is `100.alm`, unit 97, owner `Enemies`, cell `(14,107)`, template
`M_Brigand3`, carrying `Rare Magic Wood Shaman Staff {castSpell=Lightning:15}`:

1. the mage spawns with the authored staff and spell;
2. the AI casts through the staff;
3. death moves the same instance into the corpse sack;
4. the party picks up and equips the staff;
5. the party casts the same spell at the same authored power;
6. save and load between steps 4 and 5 preserve the result.

The negative witness is `150.alm`, unit 13, cell `(14,103)`, template `NPC_Scrakan`, carrying
`Prismatic_Spray:99`. `ITEM-DEATH-012` suppresses its complete item drop, including the staff. The
story does not widen "enemy mage" into "every mage drops".

### B6 — Dragons do not gain an invented drop

Shipped Dragons keep their innate `Flame Thrower`. Its suitability is zero, so the death path does
not move it into a sack. The enumerated `EXP-0227` populations contain no separate Dragon staff or
enchantment producer. A shipped Dragon death is a committed negative witness over those populations;
the contract does not widen the Medium absence result to computed image targets. No owner-only Dragon
drop is added without a new explicit directive.

### B7 — mission 40 Treasure has value 10,000

Class-14 item pricing reads the decoded signed i32 `MagicItems` value. Mission 40's `0x0e1f` instance
reports a unit value of 10,000 in item information and the existing shop transaction path. It no
longer falls through the unpriced-class value 0.

The Treasure instance has no effect. Its type-9 link is zero. A price is not treated as evidence of
an enchantment.

The value-assignment routine performs no sign branch, but `ITEM-WEAR-058` gives `-1` a separate
downstream descriptor sentinel. The story preserves that sentinel and does not generalize it into a
blanket rule for every negative value without another claim.

### B8 — shop enchantment generation

The published direct-call population places the effect-generation stage on the Magic Items shelf.
The base item is selected first. Effects are then selected under the decoded budget and exact
inclusive gates. Failure frees the rejected candidate and redraws it under the decoded attempt
bound. The story does not turn the Medium direct-call absence result into image-wide exclusivity.

The generated item price is the base item price plus the decoded effect costs, with the existing
9,999,999 clamp. Buying and selling preserve that exact instance and price.

Given a fixed injected draw stream, the generator reproduces the decoded draw ordering, inclusive
bounds, rejection and early-return rules. The current implementation derives a repeatable seed from
campaign state, while `SHOP-RNG-008` establishes a clock-seeded original and `SHOP-EFFSEED-072` leaves
the exact reseed point and intervening draws Unknown. This story therefore retains the deterministic
policy and records the difference as a divergence. It does not claim cross-session RNG parity and
does not put a clock or `math/rand` inside `pkg/sim`.

### B9 — authored sacks and actor stock

The ALM consumer resolves every non-zero type-9 link under the grammar from `EXP-0227`. Ground loot
creates an enchanted item in the sack. Owned loot creates the same item in the named actor's stock.
The zero link creates a plain item.

The EN and RU shipped-content census must account, per root, for all 28 campaign maps, 176 type-8
records, 177 type-8 elements, 199 type-9 records and 74 valid positive links. Type-9 becomes a
first-class decoded leaf rather than remaining only a raw `TileMarkers` body. Unlinked records are
reported separately and are not silently assigned to an item.

### B10 — persistence, hashing, and migration

The item instance is part of canonical hashed simulation state. The byte form records exact widths,
effect order, counts, container order, equipped-slot position, and shop-independent item price state
where the decoded model requires it.

Story `1039` uses version 59 and story `1037` uses version 60. This story owns version 61. The lane
does not choose another version itself.

Migration from form 60 creates plain instances for bare item codes and preserves the untagged weapon
pair without guessing its provenance. A zero pair becomes `None`. A non-zero pair with a materialized
weapon slot becomes one cast-spell effect on that equipped instance and source `Item`. A non-zero pair
with an empty weapon slot remains byte-for-byte under source `Legacy`; no item is invented. New loaders
write `Item` when they materialize a carriable weapon and `Innate` only when their own weapon fold
removes a non-carriable cast source. Older supported forms pass through the existing upgrade ladder
to form 60 first. Decode rejects a double source.

## Twelve-aspect closure

| Aspect | Required closure |
|---|---|
| Data | Every shipped equipment suffix, MagicItems price, type-8 link, and type-9 record resolves or is reported as unsupported. |
| Runtime state | One item-instance representation is used by carried, equipped, sack, shop, drag, and party state. |
| Simulation | Effects, price-relevant identity, stacking, drop, pickup, and hashing use the instance. |
| Player input | Equip, unequip, pickup, ground drop, buy, sell, and drag preserve the instance. |
| AI | An authored enemy mage casts through its equipped item; no second spell source is required. |
| UI/HUD | Item information and shop cells show the correct value and distinguish behaviourally different instances. |
| Triggers/scripts | Existing item give, take, drop, and authored map-loot paths preserve the instance they receive. |
| Inventory/equipment | All slots and containers use the same equality, transfer, and recompute rules. |
| Persistence/save-load | Current saves round-trip instances; supported older forms migrate; imported item paths do not discard decoded effects. |
| Campaign/session | Town regeneration and mission transitions preserve or regenerate instances under their own rules. |
| Shipped content | Both roots cover mage staves, Dragons, mission 40 Treasure, shop pools, and every linked ALM item. |
| Interactions | Death suppression, suitability, carry weight, spell casting, shop sorting, price clamps, and same-cell sack merging remain correct. |

No aspect is provisionally N/A. A known in-scope GAP fails the story.

## Domains

All nine domains are touched: Assets; Sim Core; Combat & Magic; AI & Orders; Party, Items & Heroes;
Campaign & Scripts; Town & Economy; Client; Persistence.

The implementation must preserve the import DAG. Assets decode bytes without game rules. Item state
and deterministic mechanics live in Sim Core. Shop orchestration and any clock source remain outside
the determinism wall. Client reads state and submits commands; it does not mutate item instances.

## Required witnesses

The lane must commit independently authored value tests for ordered equality, duplicates, deep-copy
and no-alias behaviour, split/clone, the Potion policy, price versus equality, and hash sensitivity
for effect kind, mode, operand, order, price, container and equipped slot. It must mutation-kill an
effect deletion at Stock, Carry, Sack, ShopItem, ALM link, death pour, binary encode, hash, rearm and
original-save import.

The real-install closure must include:

- the complete printed equipment and ALM populations from B4 and B9, including the malformed token;
- the EN and RU `M_Brigand3` witness named in B5;
- the complete spawn, cast, death, pickup, equip, cast, save, and load chain where content permits;
- `NPC_Scrakan` proving suppression; all 49 eligible and 27 suppressed Human placements per root;
- all four Dragon definitions and 21 placements per root, with the negative result from `EXP-0227`;
- both roots' `game0002.sav` Witch carrying a three-count `Potion Health Regeneration`, code
  `0x0e06`, with Effect `(kind 8, mode 1, operand 0x03c00064)` preserved through import and stacking;
- mission 40 at cell `(96,84)`, code `0x0e1f`, value 10,000, zero attached effect;
- linked ground and actor-stock examples, the zero-link plain path, and every transfer boundary in B2;
- a fixed-draw shop witness covering one, two and three effects; null/cast early returns; independent
  `50/101` and `25/101` opportunities; discarded draws; the full attempt bound; normal budget debit
  versus early-return skip; price addition/clamp; distinct generated rows; returned-item merge; buy,
  sell, cancellation and regeneration;
- form-60 `M_Brigand3` with an occupied Lightning-15 staff becoming one `Item` source; a non-zero
  pair with an empty slot becoming `Legacy` without inventing an item; a current synthetic
  non-carriable cast weapon becoming `Innate`; form-61 World round-trip, Party gob and original-save
  state-0 effect.

The witness population is producer paths, not selected rows. Each tool prints the population it
covered and the blind spots it retains.

## Exclusions

- New enchantment keys not used by shipped content or the decoded shop generator.
- Original `Item+0x47`, the transient Effect field outside the published serialized form, and token
  states 8, 12 and 17 outside the equipment grammar. An imported unsupported state is reported; it
  is not silently rewritten to zero.
- A new owner-authored dragon drop if `EXP-0227` finds no ROM1 producer.
- Turning mission 40's Treasure price into an effect.
- Full restoration of the original game's world save beyond item paths this build already imports.
- An original-compatible save writer. That remains backlog row 9.
- Replacing the project's deterministic simulation with a clock source.
- Original item-description, identify and shop-text fidelity from the post-pin item-text research. It landed
  after this story's `49b154de` research pin was frozen and will be reconciled by the later UI-text
  story; `1040` owns value and distinct-cell correctness, not the original prose surface.

## Expected divergence work

- Close `DIV-235` when item identity reaches every container. The as-built correction includes the
  school's lazy `Carry` constructor. Record the owner-retention Potion exception as `DIV-369` and
  reconcile the shop backend/display split in `DIV-322`.
- Record the retained repeatable campaign-derived shop seed policy and the Unknown original reseed
  point as a divergence.
- Record any effect kind or original-save field that remains Unknown after the research gate. An
  Unknown that affects a shipped in-scope producer is a GAP and prevents landing.
- Record any owner-directed dragon behaviour that differs from the corpus, if the owner adds such a
  directive after `EXP-0227`.

The seat reserves `DIV-369` through `DIV-384`. The lane stops and asks if the range is spent.

## Review ceiling and stopping condition

The ceiling is three adversarial passes. Hashed state and domain count do not raise it. The initial
review surface is:

1. all authored and generated producers;
2. every container and transfer boundary;
3. equip, recompute, casting, death, and stacking consumers;
4. shop identity, price, generation, and UI;
5. byte form, hashing, migration, campaign transition, and original-save import;
6. EN and RU shipped content, including the named mage, Dragon, ALM, and mission 40 populations.

Each pass reports which surfaces are covered and which remain. The chain stops at the first pass with
no P finding and an empty remaining-surface list. W findings become ledger rows. D findings are fixed
in place without another pass. A P finding returns the story. If pass three still finds a P, the seat
cuts the unresolved surface into a separate story and records that `1040` was too large; it does not
start a fourth pass.

## Dispatch order

1. `EXP-0225`, `EXP-0226`, and `EXP-0227` landed after one review each.
2. Story `1037` landed with form 60.
3. Implementation master `a901fe15` pins research `49b154de` and reconciles the amended item claims.
4. `wt-story-1040` is open from that exact implementation master; form 61 is reserved here.
5. Push the exact story tip, run the fresh adversarial chain under the ceiling above, merge only the
   corrected reviewed tip, and verify the merge commit.
