# QOL.md — deliberate improvements beyond ROM1

This ledger records conveniences the owner accepts or may accept that are not claims about ROM1
fidelity. It keeps three different questions apart:

- `docs/DIVERGENCES.md` says where againrom and researched ROM1 behaviour differ;
- `pipeline/BACKLOG.md` says which large ROM1 behaviours this build still lacks;
- this file says which optional or additive improvements againrom wants beyond ROM1.

A QoL row is not implementation authority by itself. `ACCEPTED` authorizes a later contract to
scope the feature; `CANDIDATE` asks for an owner decision first. A story that changes hashed state,
save form, economy or combat still follows the normal pipeline. If a QoL replaces original
behaviour rather than adding an opt-in route while keeping the original route available, it also
needs a divergence row.

The seat alone allocates `QOL-NNN` ids. Rows are append-only. A delivered item stays here as
`BUILT`; a declined or superseded idea stays with that decision so it is not proposed again as new.

Decision: `ACCEPTED`, `CANDIDATE`, `REJECTED`. Delivery: `OPEN`, `PARTIAL`, `BUILT`, `SUPERSEDED`.

| ID | Improvement | Decision | Delivery | Fidelity guard | Next decision or dependency |
|---|---|---|---|---|---|
| QOL-001 | Tactical pause: stop simulation, issue commands, then execute them in issue order on resume | ACCEPTED | BUILT | Pause is explicit; stopped frames run no simulation tick and change no digest. The original running route remains available | Built by `0041-world-clock`; its production witness queues stopped orders and applies them on the first resumed tick |
| QOL-002 | Speeds below and above ROM1, including a much faster game | ACCEPTED | BUILT | The original nine 8–32 tick-per-second speeds remain exact rungs and the default remains 16. Extension rungs exist only beyond their ends | Built by `0041-world-clock` and `0061-cadence-ladder`: the complete ladder is 1–1024 ticks per second |
| QOL-003 | Manual single-player merchant restock that forces stock regeneration | ACCEPTED | OPEN | Never automatic and never available to an unsynchronised multiplayer peer. One explicit action changes the same saved shop state ordinary generation owns | Decide price or time cost, cooldown, confirmation, RNG source and whether every town or only the current shop is rebuilt |
| QOL-004 | Several spells armed for autocast on one unit | ACCEPTED | OPEN | Manual casting and a single armed spell remain valid configurations. Selection is deterministic and part of canonical state if it affects combat | Specify priority, target rules, cooldown interaction, maximum count and conflict with a current player order; then allocate a save-form story |
| QOL-005 | Inventory workbench: useful grouping, filtering, sorting and less pointer travel | ACCEPTED | OPEN | Opening or changing a view never reorders canonical item instances. Any real move remains an explicit deterministic inventory command | Split into presentation-only filters, comparison, safety tags and bulk actions before dispatch; do not make one oversized inventory story |
| QOL-006 | Named equipment sets that equip a complete outfit in one action | ACCEPTED | OPEN | Applying a set is one atomic validated equipment command. Capacity, wear rules and item identity are checked before any slot changes; failure leaves the old outfit intact | Decide number and names of sets, where names persist, town versus combat availability and what happens to displaced gear |
| QOL-007 | Fighter weapon profiles: bow, crossbow, melee with shield and melee without shield | ACCEPTED | OPEN | This is an explicit equipment change, not an automatic class rewrite. Ranged profiles clear the shield; melee profiles restore only items still owned | Depends on `QOL-006`; decide whether switching in combat costs time, can interrupt an action, and may fall back when one saved item is missing |
| QOL-008 | Autocast policies inspired by Allods 2 | CANDIDATE | OPEN | Allods 2 is design inspiration, never evidence for ROM1. No behaviour is copied until the owner names the exact rule wanted | Owner to name concrete policies such as priority, health threshold, target class, recast threshold or area safety; fold accepted pieces into `QOL-004` |
| QOL-009 | Item lock, favourite and junk tags | CANDIDATE | OPEN | Tags do not change item stats or stacking. Locked items are excluded from sell, drop and bulk-transfer commands unless explicitly unlocked | Decide whether tags persist in a save or only in the local profile and whether equipment-set members are locked automatically |
| QOL-010 | Side-by-side item comparison with exact stat, resistance, spell and price deltas | CANDIDATE | OPEN | Read-only projection of the same recompute and price paths production uses; it never invents a second item calculator | Define comparison target for pack, ground loot and shop, and how unknown or conditional effects are shown |
| QOL-011 | Take-all and bulk transfer between sack, pack and shop | CANDIDATE | OPEN | Expands to a deterministic sequence of ordinary validated moves. Capacity or price failure reports the untouched remainder rather than deleting or duplicating items | Decide ordering, partial-success policy, confirmation and whether junk or locked tags participate |
| QOL-012 | Tactical planning overlay with queued waypoints, order preview and undo while paused | CANDIDATE | OPEN | Extends `QOL-001`; no queued plan enters simulation until resume, and preview state is not hashed | Decide per-unit queue depth, append versus replace gesture, undo scope and whether simultaneous party orders share one resume tick |
| QOL-013 | Quick save, quick load and rotating autosave checkpoints | CANDIDATE | OPEN | Uses the ordinary save envelope and never overwrites a named owner save without a visible retention rule | Decide slots, rotation depth, safe moments, ironman exclusion and whether quick load is available during combat |
| QOL-014 | Rebindable controls plus UI and text scaling | CANDIDATE | OPEN | Input and presentation profiles are local settings; simulation commands and original-size rendering remain available | Split key conflicts, controller support, integer UI scale, font scale and colour-accessibility overlays into bounded client stories |
| QOL-015 | Shop ergonomics: filters, affordability and equipability cues, remembered category | CANDIDATE | OPEN | Read-only presentation over current stock and party state. It neither regenerates stock nor changes prices | Start only after item identity and original UI text consumers land; keep `QOL-003` as the sole restock action |
| QOL-016 | Post-mission loot summary with explicit collect actions | CANDIDATE | OPEN | A summary never grants an item that is absent from the surviving ground/corpse state and never auto-collects without an owner rule | Decide whether it is informational only, whether collection obeys distance and capacity, and how inaccessible loot is represented |
| QOL-017 | Numbered unit control groups with camera recall | CANDIDATE | OPEN | Stores only local selection shortcuts; selecting a group submits no simulation command and does not alter party membership | Decide group count, replace/add gestures, dead-member cleanup and double-press camera behaviour |
| QOL-018 | Path, attack-range and spell-area previews before committing an order | CANDIDATE | OPEN | Read-only projection of the production pathfinder and targeting rules. The preview never becomes a second admission rule | Split movement, ranged line, area footprint and blocked-target reasons; every preview must be checked against the command's real result |
| QOL-019 | Configurable auto-pause on events such as enemy sighting, low health or completed cast | CANDIDATE | OPEN | Auto-pause only raises the existing `QOL-001` stop after the triggering tick; it neither rewinds nor changes that tick's deterministic result | Decide event list, per-character thresholds, cooldown against pause spam and whether a queued command suppresses a trigger |
| QOL-020 | Static three-star purple marker on every visible enchanted item icon | ACCEPTED | SUPERSEDED | The static mark is removed; plain code-keyed icons remain unchanged and the replacement remains presentation-only | Superseded by story `1059`, which implements the decoded procedural trail, Potion exception, measured surface census and per-surface clocks |

## Intake rule

Add a candidate when it is concrete enough to state an observable result and a fidelity guard. Do
not add generic wishes such as “better UI”. When the owner accepts a candidate, change only its
decision and sharpen its unresolved choices; implementation still starts from a normal contract.

Every landing updates `Delivery` and names its story or commit. A partial landing states the exact
remainder in the row rather than marking the whole idea built.
