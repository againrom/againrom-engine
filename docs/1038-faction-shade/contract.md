# Story `1038` — faction shade on world bodies

**Contract, seat, 2026-08-23.** Base `e3466269`, research pin `d7ee0c6`. Branch
`story/1038-faction-shade`, worktree `wt-impl1038`.

**Divergence ids reserved: `DIV-353` through `DIV-360`.** If this range is spent, the lane stops
and asks the seat. It does not allocate from its worktree, which the shared allocator cannot see.

## Result

Map-placed humanoids are drawn with the faction shade selected by their decoded owner slot. The
selection reaches the production world-body blit, not a parallel test renderer, and it works with
both preserved EN and RU installs. Unit classes which do not use the shared human palette keep
their existing class and tier palettes.

The pointable result is a composed frame from a real campaign mission: a shaded humanoid's indexed
body pixels resolve through the owner-selected table from `graphics/units/humans/human.pal`.
A both-install campaign census proves the same path for every shipped mission which contains an
eligible map-placed humanoid.

## Authority and confidence

The implementation follows the pinned research rows below. The confidence applies to exactly the
claim made by each row; a Medium conclusion is not promoted because a High mechanism feeds it.

| Row | What this story takes from it | Confidence used here |
|---|---|---|
| `PAL-RULE-021` | A map-created faction's owner shade is the 1-based ordinal of its ALM type-5 roster entry; the runtime fallback is the player's low nibble. | High for the mechanism; Medium for the shipped-single-player reach. |
| `PAL-FIRST-022` | The first creation for an owner index fixes that slot; later encounters do not rewrite it. | High. |
| `PAL-JOIN-023` | A join-like path derives the same low-nibble fallback. | High for rule A; Medium for the exact reconnect label. |
| `PAL-BLIT-024` | The selected palette object is consumed by all five world-body sprite blits, on both drawable-bit branches. | High. |
| `MAGIC-STONEDRAW-084` | Stone Curse overrides a `Palette == 0` owner table with one fixed neutral table before the body blit while `drawable+0x15a <= 2`. | High for the two gates and override arguments; Medium that both decoded shade forms render grey. |
| `REG-UNITS-050` | Drawable field `+0x15a` is the corpse stage: 0 live, 1 fallen, 2 first bone, and 3–4 the later bones. | High. |
| `UNIT-SPRITE-042` | Map-placed human bodies take the global-table side of the body-sprite selection. | High for the branch mechanics; the global table's contents and index remain Unknown. |
| `PAL-OWN-007` | `human.pal` contains sixteen consecutive 0x400-byte owner tables and is the owner-colouring resource. | High for the byte layout and mechanism. |
| `PAL-SLOT-015` | Owner slots are a sixteen-entry cycle; slot 0 is blue, slot 1 lime-green, slot 2 red, and slots pair eight apart. | High for the bytes and slot relation. |
| `PAL-BAND-016` | Only registry classes with `Palette == 0` use this owner-shaded human body family. | High for the class count and changed-index mechanism; Medium for visual colour words. |
| `PAL-SHADE-012`, `PAL-SHADE-013`, `PAL-FIGURE-014` | The palette object is a full 256-entry table, and the paper doll is a separate presentation path. | High for the mechanics; the amended/retracted wording is not carried forward. |

`UNIT-PICT-035` is the direct sprite-layout dependency of `UNIT-SPRITE-042`; it supplies the picture
pair and table relation but does not identify the global table's contents. That remaining Unknown
is not rounded into the owner-shade rule.

## The build at the base

Re-measured at `e3466269` on 2026-08-23:

- ALM unit owner values are already decoded and transported unchanged through `alm.Unit.Owner`,
  `sim.Entity.Owner`, `game.entityDraws`, and `ui.MapEntity.Owner`.
- `pkg/game/units.go` loads class and tier palettes but does not open
  `graphics/units/humans/human.pal`.
- `pkg/render/terrain.StaticFrame` owns the indexed pixels and 256-entry palette consumed by the
  canonical RGBA blit. Every world body therefore still resolves through its class frame's base
  palette, regardless of owner.
- BACKLOG row 12 says the faction shade is decoded but not applied. Its older research-open wording
  is stale: `PAL-RULE-021` and its dependencies now answer the owner-slot mechanism.
- `DIV-048` is OPEN FIDELITY-DEBT. Its exact revisit condition is an owner-colouring story plus a
  research answer for the map-faction shade ordinal. Both halves are present here; the row closes
  only after the production blit and both-install witness pass.

## Behaviour groups

**G1 — decode the owner palettes.** The Assets domain reads exactly sixteen complete BGRA palette
tables from `graphics/units/humans/human.pal`. Missing, short, long, or malformed data does not make
a body disappear: the existing base palette remains the cosmetic fallback.

**G2 — derive one presentation shade from the transported owner.** The owner value already carried
by the entity selects one of sixteen tables by its low nibble. This group adds no simulation field,
does not rewrite `Owner`, and does not change the byte-form or simulation hash.

**G3 — shade only the shared human family.** Registry classes with `Palette == 0` receive an owner
palette. Classes with a class/tier palette keep it. The choice is made after the production world
body has selected its class, tier, body replacement and pose, so those choices remain intact. A
Stone entry bypasses the owner table and stays on the client's fixed owner-independent neutral path
through corpse stage 2. Later bones resume the ordinary owner path even while the effect remains,
as the joined `MAGIC-STONEDRAW-084` and `REG-UNITS-050` rules require.

**G4 — consume the choice in the production world-body blit.** Every ordinary selected world-body
frame, including its lit path, resolves indexed pixels through the chosen table. Palette-specific
frame identity is preserved through the client texture cache; bodies owned by different factions
cannot alias one cached texture. Stone reaches the grayscale texture cache from the un-repaletted
base frame through its researched stage gate; later bones reach the ordinary owner-frame cache.

**G5 — prove shipped reach and a visible frame.** A release-gated census walks the real campaign on
both preserved installs, derives its expectation independently from the archived raw palette bytes,
and exercises the normal map-to-world-to-frame path. At least one real mission witness names its
owner slot and an indexed pixel whose composed RGBA value is the selected shade.

## Deliberate exclusions

- The paper doll, dialogue portraits, icons, cursors, terrain and structures have separate art
  paths. `PAL-FIGURE-014` keeps the doll out of this story.
- Network lobby, reconnect and multiplayer owner allocation are not new runtime features here.
  `PAL-JOIN-023` supports the selector's low-nibble rule; its Medium reconnect name is not a claim
  that this build implements that path.
- The Unknown global sprite table named by `UNIT-SPRITE-042` is not invented. This story changes
  palette selection on the body frame this build already selects. `DIV-353` records that the
  original's clear-side table has no identified counterpart in this build.
- No hashed simulation value, save format, AI decision, trigger, order, inventory state or campaign
  progression changes.

## Domains

**Assets** owns the raw sixteen-table resource and its validation. **Client** owns class eligibility,
owner-palette selection, frame identity, cache interaction and the production body blit. No other
domain is touched unless implementation discovers a real cross-domain dependency; that discovery
requires the contract to be amended before the dependency is changed.

## Closure owed

The landing records all twelve aspects as PASS, N-A or GAP. Data, runtime presentation state,
UI/HUD, campaign/session, shipped content and interaction with existing class/tier/body selection
are expected PASS. Simulation, input, AI, triggers, inventory/equipment and persistence are expected
N-A, with evidence that no hashed state or byte form changed.

`DIV-048` moves to the closed ledger only if G1 through G5 pass. `DIV-353` carries the unresolved
global-table sprite source from `UNIT-SPRITE-042`; any further mismatch is entered in the single
divergence ledger using the reserved range. The canonical `spec.md` describes the
as-built contract; `closure.md` records the aspect matrix, mutation witness, EN and RU campaign
counts, real mission composed-frame witness, milestone census and research reconciliation.

## Adversarial ceiling and stopping condition

**Three independent adversarial passes is the ceiling.** This is an ordinary two-domain
presentation story and does not change hashed simulation state.

**The chain stops at the first pass with no P finding, or when the review brief's remaining-surface
list is exhausted, whichever comes first.** P means player-visible or hashed-state-wrong and returns
the story. W means production is correct but the witness is blind; D means production and witness
are correct but a document is wrong. W and D are fixed or ledgered without another pass. If the
ceiling arrives with uncovered surface, that remainder becomes a separately scoped defect rather
than extending this story.
