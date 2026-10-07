# Story `1046` — joined actors become complete heroes

**Dispatched 2026-08-24.** The exact implementation base is
`b4e119ef8d91a9371cfcdca5b590e2f7503f51ee`; the pinned research commit is
`40ac9aff7bb716602d3b89960f8fc562d7ff9f81`. The lane and branch were unused at dispatch. The
owner's two required witnesses are Brian joining in mission 40, Treasure, and Naira joining in
mission 70, Monk Warrior. They must enter the party as complete ordinary heroes, not actors that
retain an ordinary-person payload behind a localized hero name.

The current Naira join receives aggregate XP 1,593 from the base `PC_Naira` row. The owner rejects
that value explicitly. It is a negative implementation witness, not evidence of the replacement.
This contract does not guess Naira's or Brian's original payload and does not treat current code as
evidence of ROM1.

## Result

At the first observable state after each shipped join, Brian and Naira have the exact original hero
identity, appearance, class, sex/body, aggregate XP, six skill levels, six skill-XP values, known
skills or spells, carried item instances and worn equipment for that join. Their ordinary hero
capabilities are available immediately: selection, character presentation, inventory, equip,
unequip and item use all operate through the same paths as for an ordinary roster hero.

That complete state survives save/load and campaign transitions. A completed or restored join does
not duplicate the actor, replace the payload with a base person template, discard equipment, or
leave a second non-party copy. Death, loss and mission-end culling apply the existing hero rules.

The two named actors are witnesses, not a two-call-site shortcut. Before implementation, the lane
enumerates every production producer of a persistent Hero join and every consumer that distinguishes
a hero from an ordinary Human. EN and RU shipped content must select the same semantic payload at
each corresponding producer.

## Authority and unresolved original fact

- `PARTY-JOIN-025`, `PARTY-ENDCULL-026`, `PARTY-PERSIST-028` and
  `PARTY-JOINCORPUS-029` establish the Hero join transfer and survival band.
- `PARTY-M20-030` through `PARTY-M20-032` distinguish an ordinary Human from an actor constructed
  with the Hero flag.
- `SAV-CAMPAIGN-084`, `SAV-CAMPAIGN-087` and `SAV-CAMPAIGN-088` establish Brian's mission-40 join,
  stable identity and progression-based upstream duplicate prevention.
- `DLG-DRESS-024` establishes Brian's mission-40 definition, Paladin body source and persistent
  seven-cell equipment payload.
- `SAV-HEROXP-063`, `SAV-HEROSKILL-064` and `SAV-HEROID-065` establish the aggregate/per-skill XP,
  level and per-object identity fields that a save carries. Their recorded Naira saves are later
  primary-hero states, not evidence of the mission-70 join payload.
- `ITEM-CARRY-015`, `ITEM-HUMEQ-030`, `HERO-APPEAR-042`, `HERO-APPEAR-044`,
  `HERO-CLASS-020`, `HERO-SKILL-009` and `HERO-EQUIP-017` establish item transport and live
  appearance/class derivation from equipped state.
- `REG-NPC-090` establishes the four composed Start records but does not establish a mission-stage
  Humans-row selector. `HERO-START-081`, `SESS-HERO-014` and the current pin likewise do not publish
  the complete immediately-after-join payload for Brian and Naira.

The exact progressive row or other original writer that supplies each joined hero's initial XP,
skills and equipment is therefore unresolved at this pin. A bounded research result must publish,
for every shipped persistent-Hero join producer, the live state immediately before and after
transfer: concrete source row or writer, aggregate XP, all six skill levels and XP values, carried
and worn item instances, class, sex/body and appearance. It must cover EN and RU, distinguish any
primary-hero-dependent composed branch, and separate join-time state from later earned progress.
This is an AMBER research dependency. Production code does not begin until it is resolved or the
owner explicitly changes the fidelity requirement.

The bounded research question must not be seeded with the implementation's 1,593 value, candidate
progressive row names or inferred expected payloads. It may not consume or duplicate
`EXP-0232`, `EXP-0235`, `EXP-0236`, `EXP-0237` or their allocated SAV ranges.

## Current boundary

`mapload.rosterTemplate` can construct a joined actor with PlayerCharacter, Hero, class, body,
profile, spellbook, carried items and worn equipment. The composed `npc21` through `npc24` path,
however, selects only the four base `ChargenBase` rows. `CarryRoster` preserves live XP, skills and
item instances for actors that already crossed a mission boundary.

`Frontend.missionOpenerMode` currently starts the world before `canonicalizeJoinedRoster` edits the
mission Start roster. That helper localizes a name and derives body/class from equipment, but it
does not replace a full staged hero payload and cannot retroactively repair the already-created
live actor. Current join tests prove localized identity and a small equipment subset; they do not
prove the complete join payload or first-tick live appearance.

## Behaviours

### B1 — complete payload at every persistent-Hero join producer

Enumerate the full shipped producer population from mission Start definitions, composed party-NPC
records, transfer instants and any campaign restore path. At each producer, select the exact
research-published hero payload before the live world actor is minted. Do not add a name-specific
post-start patch or promote every ordinary Human to Hero. Primary-hero-dependent composed branches
retain their original branch selection.

The payload is one atomic value: stable actor identity and Hero flag; profile, class, sex/body and
appearance inputs; aggregate XP; six skill levels and six per-skill XP values; known skills or
spells; carried item instances and worn equipment. Tests compare every field. Brian and Naira must
not pass through a base-person default and be repaired partially afterward.

### B2 — immediate ordinary-hero runtime and UI behaviour

The first observable post-join tick exposes the joined actor through the ordinary party-hero
selection and presentation paths. The mission card, doll and world actor use the correct live
class/body/appearance derived from the joined equipment. Inventory opens on the actor's carried and
worn instances; equip, unequip and item use update the same canonical state and recompute appearance
through the existing hero paths. No duplicate presentation-only hero record may shadow the live
actor.

### B3 — persistence, campaign continuity and terminal states

Round-trip the complete joined payload through the existing save form and canonical hash. Save/load,
mission completion, town entry and the next campaign mission preserve identity, XP, skills, items,
equipment and derived appearance. Restore neither replays a completed join nor leaves a duplicate
non-party actor. Death, loss, mission-end culling and a full party use the already-published Hero
rules; this story changes them only if the producer/consumer census proves that a joined hero takes
a person-only branch.

### B4 — independent EN/RU shipped witnesses

Both lawful installs run headless production-route witnesses for Brian in mission 40 and Naira in
mission 70. Each witness names the selected producer and asserts the independently published exact
join-time aggregate XP, all six skill levels and XP values, carried and worn instances,
class/body/appearance and stable identity. The Naira witness must fail against 1,593. It then opens
inventory, performs a reversible equipment interaction, saves, loads and crosses one campaign
boundary without losing or duplicating the actor.

Expected values are literal research outputs or independently decoded fixtures, never values read
back from the same production selector under test. The witness report prints selected, executed and
skipped counts separately for EN and RU. ROM1 and its GUI are never launched.

### B5 — complete producer and consumer census

A committed census closes every production join producer and every hero/person discriminator:
party identity and ownership; Hero and player-character flags; definition and appearance
derivation; class/body/profile; XP and skills; carried and worn item instances; selection,
character card and inventory UI; equip/use actions; world and doll presentation; canonical hash;
save encode/decode; campaign restore and transition; duplicate suppression; mission-end culling;
death and loss. The census names EN/RU content aliases and conditional primary-hero branches.

An unenumerated in-scope producer or consumer is a GAP. A parsed field that never reaches one of
these consumers is also a GAP. The named population, not only the Brian and Naira tests, is the
completion boundary.

## State and domains

Touched domains:

- **Assets & Formats** — shipped Human, NPC, item and save data used to form the payload;
- **Party, Items & Heroes** — identity, XP, skills, inventory, equipment and appearance;
- **Campaign & Scripts** — join producers, progression, duplicate suppression and transitions;
- **Client & Presentation** — selection, character card, doll, inventory and world appearance;
- **Persistence & Replay** — save/load and canonical hashed state.

The story crosses hashed state and more than three domains, but is not split: the owner's invariant
is that a joined actor is immediately one complete hero. Splitting payload construction from its
live, UI or persistence consumers would deliberately ship a half-hero state. The five behaviours
above are one producer-to-consumer closure, not independently landable features.

## Twelve aspects

| Aspect | Contract verdict |
|---|---|
| Data | PASS — exact research-published join payload and complete producer census |
| Runtime state | PASS — one canonical live hero record exists before first observation |
| Simulation | PASS — XP, skills, equipment and canonical hash use the complete payload |
| Player input | PASS — selection, inventory, equip, unequip and use follow ordinary hero paths |
| AI | N/A — joining does not add an AI policy; the census must prove no person-only branch remains |
| UI/HUD | PASS — actor, card, doll and inventory present the live hero state |
| Triggers/scripts | PASS — every shipped persistent-Hero join producer is enumerated |
| Inventory/equipment | PASS — carried and worn instances are exact and interactive |
| Persistence/save-load | PASS — full payload round-trips in the existing form unless evidence requires otherwise |
| Campaign/session | PASS — transitions preserve identity and prevent duplicate joins |
| Shipped content | PASS — exact EN/RU Brian, Naira and full producer-population witnesses |
| Existing interactions | PASS — death, loss, culling, full-party and primary-dependent branches retain their rules |

The N/A AI verdict is conditional on the census finding no AI discriminator. A discovered in-scope
GAP fails the story.

## Exclusions and divergence discipline

- No guessed or owner-authored replacement for an original payload that ROM1 can answer.
- No GUI or original-game launch. Only static, headless and lawful-install-reading probes.
- No new generic chargen progression system unless research proves it is the join producer.
- No conversion of ordinary temporary Humans outside the persistent-Hero join population.
- No replay-time dedup redesign beyond the actual join and campaign producer population.
- No save-form increment unless the existing fields cannot round-trip an evidence-required value.
- No use of the already-owned `EXP-0235` through `EXP-0237` or their SAV ranges, and no duplicate of
  `EXP-0232`.

No divergence id is allocated at dispatch. The lane does not invent one. If research exposes an
owner-directed residual mismatch, or implementation must knowingly differ from published ROM1
behaviour, the lane stops and asks the seat for a range before editing `docs/DIVERGENCES.md`.

## Review ceiling and stopping condition

At most three fresh adversarial passes. Only P — player-visible wrong behaviour or wrong hashed
state — returns the story. W is fixed and ledgered without another pass; D is corrected in place.
The chain stops at the first pass with no P and an empty named remaining surface. A repeated finding
closes its whole class by enumeration. At the three-pass ceiling, any remaining player-visible or
hashed-state class is cut into a separately owned defect story rather than extending this chain.

## Dispatch and implementation gate

1. `wt-story-1046` and `story/1046-joined-heroes` were unused and were created from exact
   `b4e119ef8d91a9371cfcdca5b590e2f7503f51ee`.
2. The research gitlink is exact `40ac9aff7bb716602d3b89960f8fc562d7ff9f81` and contains the
   claims cited above.
3. The current save form remains 61. This contract does not consume a new form.
4. Contract commit precedes production code. Any later implementation-master advance is integrated
   only by a normal merge at a clean coherent boundary; no rebase or stash.
5. The AMBER research dependency in Authority is unresolved at contract time. The lane may commit
   this contract and perform read-only census work, but it must not change production behaviour or
   encode expected payload values until a published research result closes the dependency.
