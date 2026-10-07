# 0152 - campaign continuity and town surfaces

**Intensity:** spec-anchored / retrospective. **Terrain:** brownfield.
**Threshold:** High for canonical purse and death-treasure state; Medium for
campaign restoration and presentation.

## Problem

The baseline at `a2214db` did not preserve the original primary character's
identity and loadout reliably, did not admit an original between-mission save
to town, and did not separate permanent heroes from temporary mission allies
on the inventory surface. It also left the campaign purse, Quest Documents,
death gold, town dialogue, shop visibility, cloak order, staff damage, and
Mission Complete input boundary incomplete.

## Terms

- **Permanent hero:** a non-mercenary player character retained by the campaign.
- **Temporary ally:** a selected map actor that is not a permanent hero,
  including the mission-20 temporary roster and mercenaries.
- **Primary hero:** the one permanent hero marked `StartingHero`.
- **Doll box:** the selected unit's nonblank picture: its composed figure,
  portrait, or current world frame, in that order where available.
- **Party inventory surface:** a selected party member's own carried-item pack
  and worn-equipment boxes.
- **Selected presentation:** the selected actor's own nonblank composed figure,
  portrait, or current world frame.
- **Campaign document:** the Quest Documents item, presented through the
  primary hero.
- **Live staff damage:** the interval that the currently equipped staff's
  weapon-borne spell would release, shown from its base through base plus
  spread.
- **Staff spell characteristics:** the item spell's powered damage where it
  damages, its exact pre-target duration where it lasts, the raw weapon-release
  range, and Prismatic Spray's owner-defined ray count.
- **Reniesta:** the English display name of permanent companion NPC22. On a
  Russian installation, NPC22 keeps the lawful localized display name while
  retaining the same companion identity, female-mage class, and equipment.

## Functional requirements

**FR-1 - Restore the decoded original-save boundary.**

An original save MUST restore the named primary character's name, sex, class,
appearance, figure, equipped items, carried stacks, and their saved
quantities. A loaded Naira MUST remain Naira's female identity and appearance;
loading or selecting another actor MUST NOT substitute a default male fighter.

The persistent party MUST retain the primary character and the decoded
permanent Humans rows, while excluding temporary mission-20 allies and
mercenaries. An original save between missions MUST enter a fresh town with
the decoded party and participant purse. It MUST NOT claim to restore completed
missions, consumed offers, or available gates.

A newly generated primary hero MUST begin as Danath. Typed names MUST preserve
case and be limited to ten bytes.

**FR-2 - Keep party inventory ownership and selected presentation distinct.**

Every selected unit, including a permanent hero, temporary ally, enemy, or
non-party actor, MUST show that unit's own nonblank doll-box picture. A
selected party member, including a permanent hero, temporary ally, or
mercenary, MUST additionally present that member's own pack and worn-equipment
boxes, appearance, carried items, experience, skills, and weight. A non-party
enemy or other non-party actor MUST NOT gain party-inventory authority.
Selection is presentation-only: it MUST NOT transfer appearance, equipment,
carried items, experience, skills, weight, or other simulation state between
characters, nor retain a stale permanent-hero doll after another unit is
selected. Returning to a loaded primary hero MUST retain that hero's loaded
appearance.

**FR-3 - Install the mission-30 companion as a real permanent hero.**

When mission 30 activates the town with `AddHero = 22`, it MUST install the
applicable companion exactly once. For the male-primary path, that companion is
NPC22, called Reniesta on the English installation: the female mage with her
install-localized display name, identity, tavern dialogue speaker, junction and
map figure, staff, dress, cloak, spells, carried state, and saved state. The
opposite primary-sex path MUST install Fergard with his own identity.

The companion is a non-primary, non-mercenary permanent hero. The temporary
mission-20 roster MUST remain detachable and MUST NOT become a permanent party
member merely through selection.

**FR-4 - Keep the primary campaign surface singular.**

Only the primary hero's hero-inventory surface MUST display the shared
participant purse and Quest Documents. A companion may pick up a sack: ordinary
items remain with that companion, while purse credit and Quest Documents route
to the primary surface. If no valid primary surface exists, the campaign-value
transfer MUST be atomic and leave the sack unchanged.

**FR-5 - Carry, show, and preserve campaign gold.**

A fresh campaign participant purse MUST start at 100. A fresh mission MUST
start with the town purse; an original mission save MUST start with its decoded
participant purse; completing a mission MUST carry its live purse to town
before the normal scenario payment. The shipped mission-20 scenario payment is
zero. As an owner-authored divergence, the mission-20 campaign-transition
reward seam MUST then add 500 to the participant purse and make the resulting
balance visible. The seam MUST make that 500 amount configurable; it MUST NOT
reinterpret the shipped scenario payment.

Instant 23 and gold-sack pickup MUST credit the participant purse with
32-bit wrapping. Eligible creature deaths MUST create or merge sacks from
their canonical death-gold inputs. Gold pickup MUST issue a quantity notice.
The primary inventory MUST show gold with the decoded EXP-0008 sprite.
Death-gold inputs and purse state MUST survive canonical byte-form round trip
without changing historical-form decoding.

**FR-6 - Provide the town and dialogue surfaces.**

The town square MUST expose distinct rectangular TAVERN, SHOP, SCHOOL, and
GATES controls. Tavern, shop, and school offers MUST use their shipped dialogue
text and each part's speaker selector with the corresponding portrait; an empty
or missing payload MUST not be replaced with authored dialogue. The visible
controls within the town rooms MUST be buttons.

Town dialogue MUST be an overlay. Enter, Escape, and its visible OK button
advance or close it as applicable; hover and click input behind it MUST not
reach town controls. The final accepted dialogue part MUST consume its offer
through the town offer path. The old competing dialogue control is absent.

Entering the shop MUST show a read-only inventory line. It MUST expose every
primary-pack item through five 80-by-80 cells, persistent left/right arrows,
clamped scrolling, and item hover popups. A release anywhere in the shop strip,
including an empty cell or spent arrow, MUST not activate a town row. The shop
MUST NOT buy, sell, move, split, drag, or equip an item.

**FR-7 - Compose mage cloaks in program order.**

For a mage wearing a slot-8 cloak, the primary cloak layer MUST be behind the
body and the secondary layer MUST be the final foreground layer. The map figure
and party-member inventory doll MUST use the same order.

**FR-8 - Report live damage and isolate the Mission Complete release.**

For a staff whose current attack is replaced by a weapon-borne spell,
character generation, the unit sheet, and the equipped-item tooltip MUST show
one `Damage` interval containing live staff damage. The interval MUST use the
same live spell-release arithmetic as the equipped staff. They MUST NOT show a
separate physical row or physical placeholder values such as `0-0` or `2-2`.
The tooltip MUST state `Magic`, the spell's useful characteristics, and the
raw weapon-release `Range`; it MUST NOT append the staff's unused physical
to-hit or defence. A damaging ordinary staff spell states `Casts <spell>` and
its powered `Damage` interval. Stone Curse instead states `<spell> <seconds>
seconds` from its exact pre-target duration and does not expose a level or
power. Prismatic Spray states `Casts <spell>`, powered `Damage`, `Rays N` where
`N = min(power / 20 + 2, 7)` by owner ruling, and `Range`. A kind-41 enchanted
shop staff MUST use the same spell-specific composition from its stored spell
and power and MUST NOT repeat a raw `Cast ... power ...` effect line. Its price
line reads `Value N`, without a leading `#`. Reniesta's equipped Wood Staff in
lawful save `666` MUST therefore show `Casts Fire Arrow`, `Damage 5-10` and Fire
Arrow's range, not physical `Damage 0-0`. A headless snapshot of the same live
world MUST expose the same worn-item tooltip lines. A unit without such a staff
MUST continue to show its ordinary physical damage interval, to-hit and
defence.

The release half of the pointer click that activates Mission Complete and
enters town MUST be swallowed. A later independent release MUST retain normal
town behavior.

## Acceptance criteria

| ID | Given | When | Then | Evidence class |
|---|---|---|---|---|
| AC-1 | a lawful original save containing Naira and the GOG Humans data | it is loaded and another actor is selected and deselected | Naira's female identity, figure, equipment, carried stacks, and stack quantities remain her own | real-data integration and live GUI |
| AC-2 | the lawful save labelled `666` with its mission-20 roster | it is loaded | the primary party and purse restore; temporary allies are excluded; the disclosed town-progress boundary is visible | real-data integration |
| AC-3 | two permanent heroes, a temporary ally, a mercenary, and an enemy with distinct state and presentation | each is selected | every selected unit shows its own nonblank doll-box picture; each party member exposes only its own pack and worn-equipment boxes; the enemy has no party inventory; no state transfers | unit and live GUI |
| AC-4 | a loaded primary hero with a changed selection history | the primary is selected again | its loaded appearance, equipment, carry, experience, skills, and weight are unchanged | unit and live GUI |
| AC-5 | mission 30 and both primary-sex choices on EN and RU lawful installs | town activation is repeated | the applicable real companion is installed once with its own identity, dialogue portrait, tavern/junction/map figure, staff, dress, cloak, and separate state; EN NPC22 displays Reniesta and RU preserves NPC22's localized display name | integration and live GUI |
| AC-6 | a companion, a gold-and-document sack, and a valid primary hero | the companion picks up the sack | ordinary items remain with the companion; purse and Quest Documents appear only through the primary hero | unit and integration |
| AC-7 | a fresh campaign, an original mission save, eligible creature death, sack pickup, instant 23, and mission 20 completion | each gold boundary occurs | fresh purse is 100; purse continuity, death gold, pickup notice, the EXP-0008 gold sprite in primary inventory, and the visible owner-authored configurable +500 transition reward follow the stated rules and round-trip canonically | unit, real-data integration, and live GUI |
| AC-8 | tavern, shop, and school offers plus a missing payload | their rooms are opened and a dialogue is advanced | shipped text and per-part portrait appear in an overlay; underlying hover/clicks are swallowed; acceptance consumes exactly once; missing content stays silent | integration and live GUI |
| AC-9 | a primary pack longer than five items | the shop is entered, arrows are used, and each cell is hovered or released | every item is reachable; arrow limits clamp; popup index matches the shown cell; no strip release activates a row or mutates inventory | unit and live GUI |
| AC-10 | a mage with distinct body and cloak pixels | map figure and hero doll are composed | primary cloak is behind the body and secondary cloak is foreground in both | unit and real-data integration |
| AC-11 | a damaging staff, Stone Curse staff, Prismatic Spray staff, kind-41 enchanted shop variants, lawful save `666` with Reniesta's equipped Wood Staff, and a non-staff weapon | generation, sheet, shop/equipped tooltip, and headless snapshot are observed while the spell is resolved | ordinary damaging spells state `Casts`, powered `Damage` and `Range`; Stone Curse states exact seconds and range; Prismatic Spray states powered damage, ray count and range; no dedicated magic description contains raw `#` or `power`; Reniesta shows Fire Arrow and `Damage 5-10`, not `Damage 0-0`; non-staff damage remains physical | unit, real-data integration, and live GUI |
| AC-12 | the exact source invocation `go run ./cmd/againrom` | Mission Complete is activated with one pointer click | its matching release cannot activate a town control; a later independent release can | live GUI |

## Derived properties

**P-1 - invariant.** Permanent-hero selection does not alter canonical state
or another character's appearance, equipment, carry, experience, skills, or
weight.

**P-2 - invariant.** Exactly one primary hero presents the participant purse
and Quest Documents.

**P-3 - idempotence.** Re-entering town after NPC 22 is installed does not add
another companion.

**P-5 - negative-invariant.** A missing town payload neither substitutes text
nor changes offer state.

**P-6 - negative-invariant.** Without a valid primary hero, a campaign-document
transfer does not partially consume a sack.

**P-7 - negative-invariant.** A staff presentation and its headless snapshot
never expose a physical damage row or physical placeholder interval. An
equipped or item-enchanted staff tooltip names one spell and uses the applicable
damage, duration, ray-count and range lines without a duplicate raw effect,
level, power or leading-`#` price line; an ordinary weapon tooltip uses its
physical interval, to-hit and defence.

**P-8 - negative-invariant.** The Mission Complete click's matching release
does not activate a town control.

**P-9 - negative-invariant.** Selecting a non-party actor never substitutes a
stale permanent-hero doll or grants party-inventory authority. Selecting a
party member resolves only that member's own inventory state.

## Preservation and regression matrix

| Baseline surface at `a2214db` | Required preserved or corrected result | Acceptance criteria |
|---|---|---|
| Character generation name handling, ordinary weapon damage, and staff tooltips | Danath default and ten-byte case-preserving names remain; ordinary weapon `Damage` remains physical; authored and item-enchanted staff tooltips use the spell-specific damage, duration, rays and range composition; the headless snapshot retains the same lines | AC-1, AC-11 |
| Mission map, combat, and canonical historical forms | Existing map behavior and historical-form decoding remain; only documented canonical state is added | AC-7 |
| Party membership and per-member state | No selection operation copies state or changes loaded-primary appearance; every party member's pack and worn boxes remain its own | AC-1, AC-3, AC-4 |
| Any selected unit | Every selected unit keeps its own nonblank doll-box picture; a non-party unit gains neither stale hero art nor party-inventory authority | AC-3, P-9 |
| Town offer ownership and missing-content silence | Offer consumption remains single-path; missing content remains silent | AC-8 |
| Shop read-only boundary | Scrolling and popups add no trading or inventory mutation | AC-9 |
| Existing town input after a completed click | Only the exact matching Mission Complete release is swallowed; later input remains live | AC-12 |
| Map and inventory figure composers | Both continue to render a figure and agree on the corrected mage-cloak order | AC-10 |

## Constraints

| Choice | Alternatives | Result |
|---|---|---|
| Mission-20 reward | Treat 500 as shipped payment; add an undisclosed special case; use an authored configurable transition reward | The reward is an explicit owner-authored 500-gold divergence after the shipped zero payment. |
| Staff presentation | Show physical combat fields; dump raw effect fields; present useful spell characteristics | A staff tooltip states `Magic` and the spell-specific damage, duration, ray-count and range lines; ordinary weapons retain their physical fields. |

## Out of scope

- Original between-mission completion progress, consumed offers, and available
  gates are not restored.
- The shop has no trade or equipment operation.
- No claim is made here that a synthetic fixture proves the preserved GOG
  Humans data, original save `666`, or an interactive desktop result.
