# 1001 — Spell effects

## Result

Every shipped spell reaches one ordinary application path. Point casts change actor state instead of ending at presentation. Area casts paint their decoded cells, apply that same path at the decoded blast, ring, or cloud cadence, and resolve layer conflicts without stacking. Wall of Earth blocks ground and spirit movement while present and restores passability when removed.

The result is witnessed in headless runs loaded from real English and Russian campaign missions. The witness covers a lasting point effect, a repeated area pulse including friendly fire, and route availability before, during, and after Wall of Earth. The same release path also restores Reniesta's spell-bearing staff from the owner's original mission-40 save, loses the mission when the restored or freshly placed Reniesta dies, toggles autocast by right-clicking a spellbook cell with a moving dashed border, and admits Fire Arrow through one real wind-up and recovery cycle with one projectile, one application, and one starting swing.

The world-unit HUD shows health above the unit square and, for mages, a blue mana bar immediately below health but still above the square. Lasting point effects and live area layers remain visible for the lifetime held by simulation state. An idle mage heals only a living wounded permitted target in spell range, in deterministic owner, ally, then neutral priority, and stops as soon as no eligible target remains. Every successfully applied spellbook use trains the spell row's own school through the existing skill-accounting path. A level rise is visible immediately to spell power and the complete derived character sheet, without healing or refilling the caster.

A mage owns one action at a time. A physical or staff attack, its weapon-spell replacement or rider, a manual spell, armed offensive autocast, and idle Heal contend at one simulation admission boundary. The admitted action owns wind-up, one release, and recovery; rejected competitors spend and train nothing, emit nothing, preserve orders, and are selected again from live state only after the actor becomes idle.

An admitted caster faces the target cell before its one cast animation starts and retains that admitted direction for the action. A Heal that restores positive health emits one target-owned semantic event; the client turns it into one deterministic, fog-gated shower of rising instances from the shipped `projectiles/healing/sprites.16a` sheet around the healed hero. The decoded seven accessible phases, embedded alpha, palette, and centring are preserved; only the arrangement, rise, and lifetime are authored. The presentation consumes no simulation RNG and owns no saved state.

Original-save restoration also retains every decoded persistent companion, including the owner-reported Brian save, with identity, skill accounting, pools, worn equipment, and carried items intact. Mission-40 actors whose lawful equipment includes a weapon and shield expose those live slots to the same character-doll compositor used for ordinary party members; no name-based painted substitute is permitted.

The spellbook popup projects every actor-dependent characteristic from the selected mage's current simulation state on every display update. A school-level, Mind, effect, equipment, loaded-save, or selected-member change is visible immediately without closing the popup; flat row values remain flat. Every actor-directed spell requires current actor perception both at admission and at the final release transition. A released projectile is committed once and continues normally if the target later enters darkness, while no new cast may target it until it becomes currently visible again.

## Authority

The contract follows active research rows `MAGIC-EFFMODE-009`, `MAGIC-CEIL-013`, `MAGIC-ARM-014`, `MAGIC-EFFECT-015`, `MAGIC-ATTACH-016`, `MAGIC-TARGET-017`, `MAGIC-TRAIN-018`, `MAGIC-SING-019`, `MAGIC-AREATICK-036` through `MAGIC-FIREDIV-047`, `MAGIC-RING-048`, `UNIT-STREAM-001`, `AI-FILTER-001`, and `AI-GROUPSEE-068`, plus the active rows they cite. Their confidence and amendment state are preserved in the pinned research tree at `744214fe1f4b461cbd2d6766390b850ab15779ab`.

The owner directs the implementation of point effects, area harm, and Wall of Earth through `MAGIC-WALLBLOCK-045`. The owner also directs right-click autocast with a rotating dashed spell-cell border, mission loss on Reniesta's death in mission 40, restoration of her equipped staff's real combat values, one caster swing at the start of a travelling projectile, decoded cast wind-up and recovery, useful-only healing, simulation-owned range-aware autocast, persistent effect feedback, health and mana bars above a unit's square, spell-school training with immediate full-sheet recomputation, strict serial ownership of every attack and cast by one mage, facing the admitted target, shipped-sheet rising Heal stars, restoration of Brian from the identified original save, live weapon/shield doll layers on the owner-visible mercenaries, current-state spell popup values, and actor-local current visibility before release. The owner further rules that loss of visibility after release never cancels the committed projectile. The reported 0-0 staff, absent loss, undiscoverable autocast, repeated swing, rapid Fire Arrow, healthy-target healing, overlapping staff/Fire Arrow/Heal activity, absent Brian, total 10575, large item set, bare mercenary dolls, stale popup, and firing into darkness are observations to reproduce and decode, not evidence of ROM1 behaviour. A research absence does not remove an owner-directed subject. It is disclosed in the divergence ledger and implemented to the authored answer.

## Behaviour boundary

The slice owns:

- ordinary damage, healing, Drain Life, protections, Shield, Haste, Slow, Bless, Curse, Stone Curse, Invisibility, Prismatic Spray, Fire Sacrifice, Teleport, and Control Spirit where the decoded target and state exist in the simulation;
- lasting effect attach, refresh, annihilation, non-stacking, continuous cadence, expiry, and reversal;
- manual, unbidden, script, and weapon/item releases through one application path;
- blast, expanding ring, cloud, wall, Light, and Darkness cell sets, timing, layer conflicts, removal, friendly fire, and Fire Ball footprint division;
- passability derived from active Wall of Earth cells for movement domains 1 and 2, without a wall branch in route search;
- deterministic hashing and round-trip persistence of lasting actor effects and area phase, ownership, power, direction, and painted cells;
- original-save reconstruction of party identity, worn equipment, weapon spell and derived combat values through the same projection used by a fresh mission;
- mission-40 critical-character binding through original-save and fresh entry, so Reniesta's real death path loses while an unrelated party death does not;
- a right-button spellbook-cell autocast toggle that submits a simulation command, clears on a second press, coexists with manual casting, and draws a cell-bound dashed border whose phase advances;
- one manual-cast animation start at projectile launch, with no restart while the projectile remains in flight and no borrowed swing on area or instant releases;
- canonical book-cast progress: one admitted cast waits for the caster's decoded charge, applies and pays once at release, then observes decoded recovery before another manual or automatic cast can be admitted;
- one canonical actor-action admission guard shared by attack commands, weapon-spell attacks, manual casts, AI casts, offensive autocast, and idle Heal; a competing action neither replaces the actor's current victim/order nor leaves a queued target behind;
- spell-specific target eligibility and deterministic in-range enumeration, including idle Heal priority of the caster's own team, allies, then neutrals, without replacing an active combat or scripted order;
- useful-only Heal admission: a dead, healthless, hostile, full-health, or out-of-range target starts no cast, spends no mana, emits no event, and occupies no cooldown;
- fog-gated feedback whose lasting point marks and area activity are driven by the live effect records and disappear with them;
- health and clamped blue mana tracks above the actual unit square, following the same camera, relief, motion, footprint, visibility, and clipping path as the unit;
- successful spellbook application awards `round(manaCost/2)` to the installed row's `Sphere`, under the promoted hero/class gates and once per ordinary area application; refusals and item/weapon casts award nothing;
- one-level-at-most skill accounting, separate per-slot experience and stored level, followed in the same action by full derived-stat recomputation and deterministic current-pool clamping without refill;
- one admitted cast-facing write from the caster to the admitted unit/cell target, shared with oriented area geometry; self/same-cell casts retain their prior facing and refusals do not turn;
- one positive-Heal semantic observation and one deterministic target-local shower drawn from the installed healing sheet, anchored to the healed target's application cell, clipped and fog-gated with no saved or hashed particle state;
- generic restoration of every decoded persistent save member, including a non-primary companion's stable identity, total and per-skill experience, worn slots and item container, without duplicate or name-specific injection;
- live mission/NPC equipment projection into the shared doll composer, with weapon and shield independently resolved and layered for every applicable owner-visible actor;
- one canonical current-state spell-characteristics projection reused by simulation and popup, with no actor-dependent Client cache or copied formula; and
- actor-local current perception in the canonical cast eligibility predicate for manual, automatic, AI, idle-Heal, place, and target-directed weapon casts, checked at admission and release only. A released projectile's simulation flight and application are independent of later visibility, while Client rendering remains fog-gated.

Area movement-cost multiplication from `MAGIC-AREACOST-046` is adjacent but outside this contract. It remains a typed divergence because implementing a second mutable terrain plane is not required to produce the owner-named spell effects or Wall of Earth passability. `MAGIC-FIREDIV-047` is inside the contract.

No new spell art, sound, or copied game asset is produced. Heal feedback resolves picture 20 to the installed healing sheet through Assets and renders the published accessible phases without sending it through the deliberately suppressed travelling-projectile draw arm. Existing fog-gated presentation reads the resulting simulation state.

## Domains and closure

The touched domains are Assets, Sim Core, Combat & Magic, AI & Orders, Party, Items & Heroes, Campaign & Scripts, Client, and Persistence. Audio is not touched. The Assets boundary includes read-only original-save decode, the installed healing sheet, and lawful mission/NPC equipment; the Campaign & Scripts boundary includes mission-40 critical identity, companion membership, and loss evaluation; the Client boundary includes actual right-button routing, live popup projection, HUD damage projection, cast animation/event consumption, target-local Heal feedback, fog gating, and the real doll compositor. The Sim Core and AI boundaries own actor-local perception and refuse unseen actor-directed targets before any action side effect.

Data, runtime state, simulation, input, AI, UI/HUD, triggers, inventory/equipment, persistence, campaign/session, shipped content, and interactions with existing mechanics are all applicable. Landing requires a PASS for each; a known in-scope GAP fails the story. The evidence includes the owner's preserved mission-40 save as a read-only integration instrument and a synthetic unit fixture of its decoded shape; neither save bytes nor installed assets enter tests or history.

## Customisation seam

Installed spell rows remain the source of mode, magnitude, duration, distribution, radius, range, area lifetime, and damage. Engine-width truncation is retained where research establishes it. Collection row count and effect-list count are data-driven; fixed original limits are recorded with their width and with whether lifting them changes shipped bytes. No shipped table is rewritten.

## Divergence outcome

`DIV-001` closes when every applicable point arm has state and every release path consumes it. `DIV-002` closes when area effects reuse ordinary application on decoded cells and cadence. The known omission of `MAGIC-AREACOST-046`, any owner-authored mission/autocast behaviour not established for ROM1, and any positively established unresolved singularity remain typed in `docs/DIVERGENCES.md`; no mismatch may survive only in this folder.
