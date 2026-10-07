# Story `1040` — enchanted item instances

This is the canonical as-built specification. `contract.md` records the measured premise and the
owner scope before implementation.

Base `a901fe15`, merged master `5f91f6f5`, research pin `49b154de`. The pin did not move. Simulation
form 61 belongs to this story.

## 1. Canonical identity

`sim.ItemInstance` owns one packed item code, the original item-kind byte, an ordered list of
`ItemEffect{Kind, Mode, Operand}`, and signed stored price. Effect order and duplicates are state.
Every public reader and transfer deep-copies the effect slice.

`ItemInstance.HasEnchantment` is the one read-only presentation seam. It answers true exactly when
the effect list is non-empty. Price alone is not an enchantment.

`sim.ItemEqual` implements the decoded stack predicate. It compares code first. An ordinary item
with no effects is stackable; two enchanted equipment instances compare their ordered effects.
Transaction price is outside ROM1 equality, but a container or shop cell also compares its stored
price because one quantity cannot preserve two values.

Potion kind 3 is the deliberate exception to ROM1. This build merges equal-code Potions only when
their ordered effects and prices also agree. That preserves the real enchanted Potion imported from
`game0002.sav`; `DIV-369` records the difference. `foldContainer`, shop shelf folding, transfers and
pickup all use the same identity.

Code arrays remain compatibility projections for old callers and the pre-61 form. They are derived
from canonical instances and are never used to rebuild a populated canonical record.

## 2. Authored producers

`mapload.ParseItemCell` separates the item name from its first brace block and preserves every
accepted effect in source order. It recognises the 49 decoded kind names, the permanent,
single-use, charges, duration and continuous modes, signed scalar operands, damage ranges, spell
ids and spell power. An unknown or malformed element is counted and rejected without losing its
accepted siblings. An indeterminate numeric conversion returns an error instead of inventing a
value.

Human and Unit equipment resolution constructs the complete instance. Actor stock and ground loot
consume ALM type-8 elements and their one-based type-9 links. A zero link produces a plain instance;
a valid link appends the linked ordered effects. Type-9 records are decoded leaves rather than raw
marker bodies.

The same construction is used by generated party equipment, starting inventory, party carry,
mercenary and NPC placement, fallback starting-weapon materialisation and script-created plain
items. The latter remain deliberately plain where the script supplies only a code.

Class-14 items take their signed value from `MagicItems`. Mission 40's `0x0e1f` Treasure therefore
has stored price 10,000 and an empty effect list.

Original-save state-0 item effects become ordered `ItemEffect` values. Unsupported piece states are
reported and not rewritten as a plain effect.

## 3. Containers and transfers

Carried stacks hold a complete instance plus quantity. All twelve equipment slots, every ground
sack, `Stock`, `PartyMember`, `Carry`, shop shelf and table cells carry complete instances.
Compatibility code fields are synchronized at their boundary.

These operations move or clone the complete value:

- construction, party cloning, mission start and campaign carry;
- the school's first training, when it materialises a member's `Carry`;
- equip, unequip, slot displacement and character-to-character transfer;
- manual ground drop, whole-container drop, death pour, same-cell sack merge and pickup;
- script give, take and drop operations;
- shop shelf/table/pack moves, split, cancellation, buy, sell, sorting and regeneration;
- original-save replacement, save, load and explicit old-form migration.

Death preserves carried order, then slot 2, slot 1 and slots 3 through 12. NPC suppression deletes
the complete loadout as before. Suitability remains the gate for whether authored equipment exists
to drop; no Dragon item is invented.

## 4. Equipped-effect consumers

Every rearm begins from effect-free data and folds equipped slots in slot order and effects in stored
order. The shared fold updates Body, Mind, Reaction, Spirit, maximum health, maximum mana,
regeneration, hit chance, physical damage, defence, absorption, speed, rotation speed, sight,
elemental protections, fighter or mage skills, elemental secondary damage, and the decoded general
damage arms. Health, mana and teach-spell effects use their separate published pool/book consumers.

Effect kinds 44 through 48 are one byte-width `(base, spread, selector)` value. Base is the low
operand byte, spread is the next byte and selector is `kind - 44`. Each occurrence replaces the
whole value in equipped-slot and stored-effect order. A later Water effect therefore replaces an
earlier Fire effect, the reverse order leaves Fire, and a repeated kind keeps only its later pair.
There is no five-component sum.

Equip, unequip, load and rearm replace derived state rather than accumulating it. Placement-time
speed and sight use the same authored modifiers. The combat resolver consumes the resulting
secondary component and reduces it through exactly the selected Fire-through-Astral protection.
The component proceeds when the physical hit succeeds or when the physical base/spread pair is
empty; a miss with a nonempty physical pair suppresses both. Damage and hit-roll draws retain their
existing order, and the secondary spread is drawn only after that gate. This makes a generated
ranged weapon on a legal Body-15 fighter, whose ranged fold leaves the physical pair empty, apply
its kind-44..48 item effect even when the ordinary roll misses. The ordinary regeneration step
consumes health and mana regeneration.
Kinds diagnosed as no-ops remain no-ops.

Weapon casting has one persisted provenance tag:

- `None`: no weapon pair and no equipped cast effect;
- `Item`: the equipped weapon's first kind-41 effect is authoritative;
- `Innate`: a non-carriable authored weapon supplied the pair;
- `Legacy`: a pre-61 nonzero pair had no materialized weapon from which provenance could be proved.

Construction, equip, unequip and rearm keep the tag, cached pair and equipped item consistent.
Player and AI casting use the same cached pair validated from that source.

## 5. Shop generation and transactions

The Magic shelf selects from the decoded armor, shield and weapon pool. A candidate receives one
required effect and the independent inclusive `50/101` and `25/101` opportunities for a second and
third. Selection preserves the decoded weighted endpoints, retries, capacity, interaction budget,
cast-spell early returns, rejected draws and the 9,999,999 price clamp.

A fixed injected draw source makes every rule testable. Production keeps the existing
campaign-derived deterministic seed. ROM1 starts from a clock seed and its exact reseed point remains
Unknown; `DIV-370` records the retained difference.

`ShopItem` carries the full instance and unit price. Shelf folds, table joins and return insertion
compare complete identity plus price. Differently enchanted equal-code equipment therefore occupies
different cells. Every shelf/table/public reader clones effects. Selling uses the stored unit price;
buying preserves that same item instance in the party pack.

`DIV-322` remains open only for the backend-versus-display mechanism: this build folds the shelf
model, while the original appends units and merges display rows. Its former inability to distinguish
enchanted instances is closed.

## 6. Persistence, hashing and migration

Form 61 retains the old sack, carried and equipment code projections, then appends canonical item
state. The appended section records each complete sack item, each carried stack and quantity, all
twelve equipped slots, every weapon-source tag, and item-derived regeneration, rotation and
secondary-damage state. Each item encodes code, kind, signed price, effect count, and every exact
kind/mode/operand tuple.

The secondary state encodes its one byte-width base, spread and selector followed by required-zero
padding inside form 61's reserved derived-state width. Decode rejects selectors outside `0..4` and
nonzero padding, so neither the API nor the wire form can admit coexisting school components.

Decode cross-checks every canonical item code against its old projection, rejects non-canonical
stacks, invalid source tags and disagreeing source/item pairs, and requires the section to be
consumed exactly. `World.Hash` hashes the complete form, including price, order and duplicates.
It also distinguishes secondary selectors when base and spread are otherwise equal.

The explicit form-60 migration constructs plain item objects for old codes. A zero weapon pair
becomes `None`. A nonzero pair with an occupied weapon slot becomes one kind-41 effect and `Item`.
A nonzero pair with an empty slot remains unchanged under `Legacy`; no item is invented. Current
non-carriable authored cast weapons are created as `Innate`. Earlier readable forms first pass
through their existing upgrade rungs and receive the same version-61 disclosure.

Party gob persistence carries canonical carried and equipped instances. Original-save import
replaces the live actor's stock with complete imported items and preserves effects through the same
form-61 round trip.

## 7. Shipped behaviour

On both roots, `100.alm` unit 97 (`M_Brigand3`) spawns with
`Rare Magic Wood Shaman Staff {castSpell=Lightning:15}`. The enemy casts, dies, drops that instance,
the party picks it up and equips it, a form-61 round trip preserves it, and the party casts Lightning
at power 15.

`150.alm` unit 13 (`NPC_Scrakan`) keeps its `Prismatic_Spray:99` staff suppressed on death. All four
Dragon definitions and all 21 campaign placements per root keep innate `Flame Thrower` and drop no
invented weapon. Mission 40 Treasure remains plain and valued at 10,000. The Witch in
`game0002.sav` imports three enchanted `Potion Health Regeneration` instances without an
effect-blind merge.

The installed Magic shelf has ten forced-cast weapon triples. On both roots, fixed draws select
`Staff` `0x812d` and encode Lightning 15 as one kind-41 effect. At a 1,000,000 ceiling its base
price is 333, forced budget is 33,300, maximum power is 19, cast contribution is 24,728 and final
price is 25,061. The shelf and buy views show 25,061; a player-owned table place and the sell commit
use 12,531. The exact instance survives shelf, table, pack, equip and form 61, and its attack casts
from source `Item`. A plain `0x812d` remains unenchanted at 333 and cannot fold with the enchanted
instance even when a test equalises their stored prices.

## 8. Presentation and open work

Item information and shop cells read stored price and complete identity. Story `1040` does not
implement the original description, identify or shop prose from the post-pin item-text research.

Hotfix `08595ab8` adds the custom `QOL-020` marker: three static purple pixel stars on every visible
pack, worn-slot and shop item icon whose `ItemInstance.HasEnchantment` result is true. Drag previews
reuse the marked icon. Plain instances keep the shared base icon unchanged. The marker is not a
claim about ROM1's animation.

The decoded ROM1 purple-star animation, asset and cadence remain later fidelity work. That consumer
must also read `ItemInstance.HasEnchantment` and must not inspect or mutate the effect slice itself.

Open ledger work after this story is `DIV-322`, `DIV-369` and `DIV-370`. `DIV-235` closes because
canonical per-object identity now reaches every container and transfer. No other id from the
reserved `DIV-369` through `DIV-384` range is spent.
