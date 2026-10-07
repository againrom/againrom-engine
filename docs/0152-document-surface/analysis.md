# Analysis

## Scope examined

The comparison baseline is implementation commit `a2214db`. The working tree
adds original-save party restoration, campaign purse continuity, creature gold,
town presentation, party-surface selection, figure composition, and staff
damage presentation. The inspected paths include original-save decoding,
mission construction and completion, canonical world encoding, town dialogue,
inventory composition, and UI input dispatch.

## Baseline observations

At `a2214db`, original-save restoration did not retain the loaded primary
character's complete identity, figure, loadout, and stack counts, and a save
between missions could not open the town. The party model did not distinguish
permanent heroes from the temporary mission-20 roster for campaign surfaces.

The selected-unit sheet had no explicit rule separating any selected unit's
doll picture from party-member pack and worn state. The campaign purse and
Quest Documents were not constrained to the primary hero surface.
Mission completion did not carry the live purse back before the mission payment.
Death gold, the mission-complete gold notification, and the decoded gold visual
were absent from the required vertical path.

The shipped mission-20 scenario payment is zero. The owner's required +500 is
therefore an authored campaign-transition reward, not a decoded scenario
payment; it needs a named seam rather than a reinterpretation of source data.

Town entry exposed list controls rather than four square regions. Shop and
school offers lacked the same portrait dialogue used by the tavern; the shop
showed only a fixed first five inventory positions. Mage cloak layers followed
the ordinary order. The sheet could expose a staff's physical interval instead
of the interval used by its live spell release. A release from the click that
completed a mission could reach the just-opened town.

## Working-tree observations

The current implementation models a permanent hero separately from a
mercenary or temporary ally. Every selected unit has its own doll-box picture:
a party member uses its composed figure where available, while another unit
uses its own portrait or current world frame. Every party member has an
individual pack and worn-equipment boxes; only the primary hero presents
campaign gold and documents. A non-party enemy never inherits the previously
selected hero's doll or party-inventory authority.

Original-save restoration retains the loaded primary character's identity,
including Naira's sex and appearance, rather than rebuilding a default male
fighter. It restores a between-mission save into a fresh town with its decoded
party and purse only. The exact town-progress boundary remains explicit.

Mission 30 installs the real NPC22 path for the applicable primary-hero choice.
The companion is called Reniesta on EN and keeps the lawful localized display
name on RU. It has its own identity and starting equipment from the
tavern/junction/map construction path. The mission-20 temporary roster does
not become permanent merely because an actor is selected.

For a staff attack, the one `Damage` value is the live weapon-spell interval.
It is not a separate physical row and it is not a placeholder physical value
such as `0-0` or `2-2`. Non-staff weapons retain their ordinary live damage
interval.

The new Mission Complete release latch consumes exactly the release half of
the click that entered town. It does not suppress later independent town input.

## Evidence limits

The asset-free unit tests construct synthetic saves, maps, archives, entities,
and UI input. They establish code-level invariants but cannot establish a
result from lawful GOG data. Separate environment-gated release tests exercise
the EN and RU data, the preserved Naira saves, and the save labelled `666`.
Neither class of automated test establishes an interactive desktop result.
The commands, input hashes, observed restored state, and remaining live checks
are recorded in `verification.md`.
