# Plan

## Approach

Campaign membership remains separate from canonical map state. The campaign
metadata records permanent-hero, primary-hero, mercenary, and companion roles;
mission construction maps party members to their own entities and inventory
state. The selected actor feeds the doll-box presentation; party membership
permits pack and worn-equipment access, while a non-party selection does not.

Participant-owned campaign values are routed through the primary hero surface.
The canonical world owns live purse and death-treasure state. Town, panel, and
character-generation code read that state for presentation without recreating
the combat or campaign rules.

## Facts verified during planning

- At `a2214db`, original-save restoration and the selected-unit inventory did
  not provide the required permanent/temporary ownership boundary.
- The baseline canonical form had no entity death-treasure record and did not
  carry mission purse back to town before payment.
- The baseline town screen lacked the required dialogue overlay, complete
  shop-strip visibility, and exact click-release boundary.
- The baseline sheet had one damage field but did not bind a staff's value to
  the live spell-release path.

## Design decisions

### Party and original-save restoration

The original-save reader restores decoded permanent heroes and filters temporary
allies. It uses the loaded primary identity, including sex and appearance, as
the persistent source rather than minting a default party member. The
between-mission path enters a fresh town with only the decoded party and purse.
This serves FR-1 and AC-1 through AC-4.

Each party member supplies an independent carried set, worn set, figure,
experience, skills, and weight. Every selected unit supplies its own doll-box
picture through composed figure, portrait, or current frame. A selected party
member, permanent or temporary, receives its own pack and worn-equipment state;
a non-party unit does not. The rejected alternatives are clearing or retaining
the old hero doll for a temporary, enemy, or non-party selection: both hide the
selected actor and conflate presentation with inventory authority. This serves
FR-2 and AC-3 through AC-4.

NPC22 is installed from the chapter's AddHero data once, using the real
companion's character, figure, and starting equipment construction path. This
keeps NPC22's tavern dialogue, junction, map, and loadout identity coherent
instead of assembling a generic mage; Reniesta is its EN display name and RU
keeps its localized display name. This serves FR-3 and AC-5.

### Campaign values and canonical state

The participant purse begins at 100 for a fresh campaign. It moves through
fresh mission construction, original-save load, sack pickup, instant 23, and
mission completion as the same participant value. Mission completion copies it
to town before normal scenario payment. Mission 20's shipped payment remains
zero; a named configurable campaign-transition reward seam then adds the
owner-authored 500. This serves FR-5 and AC-7.

Quest Documents and purse presentation resolve through the primary hero only.
Companion pickup leaves ordinary items with the companion, then routes campaign
values to the primary surface atomically. The rejected alternative is a
per-companion campaign surface, which makes the displayed purse and documents
depend on selection. This serves FR-4 and AC-6.

Death-gold inputs are canonical entity state, encoded in a new fixed section so
future death outcomes remain stable across save and resume. Historical-form
decoding remains unchanged. This serves FR-5 and AC-7.

### Town and figure presentation

Town dialogue reuses shipped event text and the per-part speaker resolver. The
overlay owns keyboard and pointer input while it is visible. The shop retains
five visible cells but has a clamped offset and fixed arrows; it is read-only.
The release latch is set only by the click that completes a mission and is
cleared by its matching release. This serves FR-6 and FR-8, and AC-8, AC-9,
and AC-12.

Both figure composers use the mage program order: primary cloak behind body,
secondary cloak foreground. The map sheet, character-generation preview, and
equipped-item presenter obtain an equipped staff interval from the selected
live `World.WeaponSpellDamage` path. They format base through base plus spread,
which is the same arithmetic used by the staff release. The headless snapshot
uses the same item-tooltip lines. A weapon without a live weapon spell retains
its physical item interval. This serves FR-7, FR-8, AC-10, and AC-11.

## Risks

| ID | Risk | Mitigation |
|---|---|---|
| R-1 | Selection could mutate canonical state or a loaded hero's appearance. | Compare state before and after selection; exercise permanent and temporary selections. |
| R-2 | A temporary party member could lose its own presentation or pack/worn state, or a non-party selection could inherit either. | Test the doll box and party-inventory boundary separately for a temporary member and an enemy. |
| R-3 | A purse or document pickup could split campaign ownership. | Exercise companion pickup with gold, ordinary items, and Quest Documents. |
| R-4 | A canonical change could shift later bytes or historical decode. | Use fixed-width round trips and historical-form peels. |
| R-5 | Town overlay or Mission Complete input could leak a release to an underlying control. | Test the exact press/release sequence and a later independent gesture. |
| R-6 | A display calculation or headless snapshot could show staff physical damage rather than live spell damage. | Compare each staff interval with `World.WeaponSpellDamage` and the release arithmetic; kill the physical-fallback mutation with a `0-0` staff fixture and the lawful save-666 route. |

## Success criteria

The design supports FR-1 through FR-8 and AC-1 through AC-12. Synthetic tests
cover invariants and canonical state; lawful-install and live GUI observations
cover the data- and presentation-dependent acceptance criteria.
