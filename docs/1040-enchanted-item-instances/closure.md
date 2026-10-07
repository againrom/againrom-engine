# Story `1040` — closure

As-built evidence. `spec.md` owns behaviour; this file owns the aspect matrix, shipped witnesses,
research reconciliation and remaining review surface.

Base `a901fe15`, merged master `5f91f6f5`, research pin `49b154de`, simulation form 61. The pin did
not move.

## Result

An enchantment is now part of one item instance from its authored or generated producer through
equipment, inventory, sacks, shops, campaign carry and persistence. Its consumers affect live
combat and derived state. Equal-code enchanted equipment remains distinct; Potion effects and
stored value are retained under the owner policy.

The integration witness is both roots' real `100.alm` `M_Brigand3`: authored Lightning-15 staff,
enemy cast, death sack, party pickup, equip, form-61 save/load and party cast. The negative witnesses
are `NPC_Scrakan` suppression and every shipped Dragon placement. A second both-root witness builds
the real Magic-shelf `Staff` `0x812d` with fixed draws and proves its exact spell and transaction
price through purchase, equip, form-61 reload, cast and resale.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | The equipment grammar accounts for 2,386 cells, 260 braced cells, 266 accepted effects and one malformed nested cell per root. ALM decoding accounts for all 28 maps, 176 type-8 records, 177 type-8 elements, 199 type-9 records and 74 valid links per root. Class-14 stored values include mission 40's 10,000 Treasure. |
| Runtime state | PASS | `ItemInstance` is canonical in carried stacks, twelve equipped slots, sacks, Stock, party/carry records and shop cells. Readers deep-copy effects. `HasEnchantment` is the single read-only presentation predicate. |
| Simulation | PASS | Complete identity controls stacking and hash. Equip/rearm folds ordered effects without accumulation. Kinds 44..48 replace one `(base, spread, selector)` triple; combat consumes only that selected protection component after a physical hit or whenever the physical pair is empty. A nonempty physical pair miss suppresses both. Regeneration consumes both rates, and drop/pickup moves the same instance. Potion retention is explicit `DIV-369`. |
| Player input | PASS | Existing equip, unequip, pack/doll/table drags, buy, sell, cancellation, ground drop and pickup move complete instances. Fallback starting-weapon materialisation preserves all pre-existing canonical carried and equipped values. |
| AI | PASS | Human staff placements obtain their cast from the equipped item. Enemy AI uses the validated pair. Non-carriable authored cast weapons are tagged `Innate`; Dragons keep innate Flame Thrower without an invented drop. |
| UI/HUD | PASS | Item information and shop cells consume stored price and distinguish effect identity. The generated Staff shows full value 25,061 on the shelf and buy path and 12,531 on the player table and sell path. Mission 40 Treasure reports 10,000. Purple-star animation is an explicit later interaction and is not implemented here. |
| Triggers/scripts | PASS | Code-only script item opcodes create plain instances and every existing give, take and drop path preserves the complete value it receives. Script-coverage code/count views are read-only projections; canonical sacks remain in its observation. |
| Inventory/equipment | PASS | Construction, folding, split, clone, equip, displacement, unequip, transfer, death and pickup share complete instance identity. Death suppression and suitability retain their previous gates. |
| Persistence/save-load | PASS | Form 61 appends canonical item state, source tags and derived item fields; decode cross-checks legacy projections and refuses an invalid secondary selector or nonzero reserved padding. Hash changes with kind, mode, operand, order, price, location and secondary selector. Form 60 migrates to exact None/Item/Legacy states and a zero secondary triple, while current non-carriable producers write Innate. Party gob and original-save imports round-trip effects. |
| Campaign/session | PASS | Generated heroes, mission load, party clone/carry, roster join, school training's lazy Carry construction, original-save restore, town trade and the next mission preserve canonical instances. Shop regeneration intentionally uses the retained deterministic seed (`DIV-370`). |
| Shipped content | PASS | Both roots cover all equipment/ALM populations, 49 eligible and 27 suppressed Human staff placements, ten forced-cast shop triples and exact generated Staff `0x812d`, four Dragon definitions and 21 placements, mission 40 Treasure, Witch Potion import, linked ground/stock and zero-link loot. |
| Interactions with existing mechanics | PASS | Carry weight remains code-derived; effects interact with base recompute, pools, skills, protections, secondary damage, regeneration, casting, price clamp, shop sort/fold and same-cell sack merge. No code projection can overwrite a populated canonical instance. |

No aspect is `GAP`. The star animation, post-pin text fidelity and ROM1-exact shop reseeding are
outside this contract.

## Exact shipped witnesses

The install-gated release test walks every campaign ALM entry rather than a selected mission list.
On each root it prints:

```text
equipment cells=2386 braced=260 effects=266 rejected=1
maps=28 type8=176/177 type9=199 linked=74 records (74 elements)
Human staff placements=49 eligible + 27 suppressed
innate cast placements=3
Dragons=4 definitions/21 placements
```

The positive chain locates `100.alm` unit 97 at `(14,107)` by authored id and verifies the complete
staff effect before any action. A production enemy cast creates Lightning 15. Death pours the same
effect into the sack; pickup and equip preserve it. The world is marshalled and unmarshalled before
the party cast, so the last cast is a persistence witness as well as a transfer witness.

`150.alm` unit 13 at `(14,103)` dies without adding its staff to the sack population. The Dragon
census kills every placement and proves the Flame Thrower code count does not increase. It also
checks the source is `Innate`, not `Item`.

Mission 40's sack at `(96,84)` contains `0x0e1f`, price 10,000 and no effect. Both roots' Witch in
`game0002.sav` imports a three-count `0x0e06` Potion with effect kind 8, mode 1, operand
`0x03c00064`, and price 50.

Both roots expose ten forced-cast weapon triples in the installed shop table. Fixed draws select
`Staff` `0x812d`, base price 333, and construct Lightning 15 from the exact budget 33,300 and
power maximum 19. Its kind-41 operand is the spell id in the low word and signed power in the high
word. The independent price formula gives cast contribution 24,728, stored and buy value 25,061,
and sell value 12,531. Production shelf and table cells show those values and the Lightning-15
effect. The instance then survives buy, pack, equip and form 61, establishes source `Item`, emits a
weapon cast and returns intact to the weapon shelf after sale. A plain same-code `0x812d` control
stays effect-free at base value 333 and remains distinct even with an equalised test price.

## Deterministic and migration witnesses

Fixed-draw shop tests cover one, two and three effects; both optional gates; forced staff casting; weighted selection and
retry; capacity and budget refusal; ordinary and cast price formulae; cast early return; discarded
optional casts; the outer attempt bound; clamp; distinct identity; shelf/table split and merge;
buy, sell, cancellation and regeneration. Alias tests mutate every returned effect slice and prove
the model is unchanged.

Independent state tests cover effect order and duplicates, plain and enchanted equality, Potion
retention, price outside item equality but inside a container cell, carried/equipped/sack moves,
death and pickup, public-reader copies, hash sensitivity and form-61 byte round trips.

The secondary-damage witnesses cover Fire then Water, Water then Fire, repeated-kind replacement,
an unrelated-effect control and later equipment-slot replacement. A live `Rearm` preserves the one
triple. A headless production-seam witness takes a fighter-slot-1 Short Bow from the real shop-pool
and fixed-draw generation functions, equips it on a legal Body-15 fighter, and proves the ranged
fold leaves physical base/spread zero and `AlwaysHits` false. On the forced ordinary miss, each of
kinds 44..48 still reaches exactly its selected Fire-through-Astral protection. Resolver controls
pin secondary-only miss at three draws and positive damage; physical-plus-secondary miss,
physical-only miss and empty/no-secondary at two draws and zero damage; and the ordinary hit path
at three draws with both components. Constructor, `SetCombat`, `SetDerived` and form decode refuse
selector 5 atomically. Form 61 round-trips the triple, hashes a selector-only mutation and rejects
nonzero reserved padding that could otherwise hide a second component. Form 60 migrates to the zero
triple.

The school's first successful training now constructs `Carry` from the canonical carried and worn
instances before it updates experience. Distinct effects and prices survive in both halves, code
projections agree, and mutations of the member's pre-carry slices do not alias the captured state.

Form-60 fixtures cover an occupied `M_Brigand3` Lightning-15 staff becoming `Item`, a nonzero pair
with an empty slot becoming `Legacy`, and a zero pair becoming `None`. A current synthetic
non-carriable cast weapon becomes `Innate`. Party gob and original-save state-0 effects provide the
two persistence edges outside `sim.World`.

## Research and divergence reconciliation

- `ITEM-EFFGRAM-070` through `ITEM-EFFSAVE-077` supply the item/effect grammar, copy, equality,
  consumer and save rules. `ITEM-EFFDISP-075` supplies the ordered equip/remove walkers and the
  kind-44..48 installers. `HERO-MOD-016` establishes the one live base/spread/selector triple and
  its assignment rather than coexisting selector state. `HERO-DMG2-029` establishes
  the shared resolver's selected-protection consumer and its physical-hit-or-empty-pair gate. The
  implementation preserves the combined widths, order, duplicate state and last-write semantics.
- `SHOP-EFFPOOL-061` through `SHOP-EFFSEED-072` supply selection, budgets, draw order, prices and the
  clock-seed boundary. Generation implements the deterministic rule set; `DIV-370` records the seed
  policy difference and remaining reseed Unknown.
- `ALM-EFFREC-071` through `ALM-M40-074` and `ITEM-AUTHCAST-086` through `ITEM-PRODUCER-091` supply
  the link grammar, Human placement population, Dragon negative population and Treasure value.
- `DIV-235` is CLOSED: complete identity now reaches every container, including school training's
  lazy Carry constructor. `DIV-322` is amended: shop cells now distinguish complete effects, while
  the backend/display folding mechanism still differs.
- `DIV-369` records the deliberate effect-and-price-aware Potion rule. No extra Dragon divergence
  exists because the owner did not direct a new Dragon drop after the negative corpus result.

The later item-text experiment is not cited as authority. It landed after the frozen pin and its work remains a
later story.

## Gates

The candidate gate population is the repository Go build/vet/test chain, format, asset and citation
checks, divergence structure/retraction reconciliation, pin-forward check, both 58-test lawful-root
release runs, both scenario runs, milestone census and preserved-install verification. The exact
results are recorded at the candidate commit before push; no gate is satisfied by a skip.

## Review surface and stopping

The independent review receives all nine domains and these remaining surfaces:

1. every authored, generated, imported and code-only producer;
2. every carried, worn, sack, party, shop and drag container and transfer;
3. stacking, recompute, pools, combat, regeneration, casting, death and suppression;
4. form 61, hash, form-60 migration, party gob and original-save import;
5. exact EN/RU equipment, ALM, mage, Scrakan, Dragon, Potion and Treasure populations;
6. ledger truth, explicit post-pin text exclusion and the future star-animation seam.

No surface is pre-declared covered by review. The ceiling is three passes. The chain stops at the
first pass with no P finding and an empty remaining list. This lane does not self-review and does
not merge or rebuild `builds/current`.
