# Story `1046` — joined actors become complete heroes

This is the canonical as-built specification. `contract.md` preserves the dispatch-time AMBER
boundary; it is not rewritten. The dependency is resolved at research pin
`be95a8b482cfed678625b5a7e4c8fa29c17264e6` by `HERO-JOIN-120`..`128`. The implementation base is
`b4e119ef8d91a9371cfcdca5b590e2f7503f51ee`; implementation masters `2e744f8c` and `8f6e68e3` were
merged normally. Simulation form 61 remains current.

## 1. Producer population and exact rows

**FR1.** The shipped persistent-companion population is five producer origins and ten conditional
source variants:

| Origin | NPC | Source rule |
|---|---:|---|
| mission-30 town `AddHero` | 22 | female primary selects server 28 `PC_Fergard`; male selects 29 `PC_Reniesta` |
| mission-40 map unit 6, group 14 | 25 | fixed server 42 `PC_Paladin` — Brian |
| mission-70 map unit 151, group 105 | 23 | female primary selects 30 `PC_Danath_2`; male selects 31 `PC_Naira_2` |
| mission-100 map unit 245, group 113 | 24 | male/female fighter selects 36/37; male/female mage selects 34/35 |
| mission-140 map unit 443, group 16 | 26 | fixed server 43 `PC_Elf` — Rood Glaen |

The composed selector is `26 + selector + 4*floor(mission/40)`. Fixed registry ids bypass that
formula. EN and RU choose the same rows. Only localized display names differ.

**FR2.** `NPCDefs.CampaignServerID` owns this selection. Campaign map construction asks it before a
person is minted. `CampaignNPCMember` gives town `AddHero` the same definition-id placement,
Humans-row parser, item-instance resolver and roster constructor used by a map NPC. Standalone and
synthetic map starts keep their tier-zero fallback when no campaign server id can be resolved.

**FR3.** A selected Humans row is one atomic payload: permanent non-primary player-character
identity, type and class, sex and figure, profile, four hero statistics, six skill levels, six
derived construction XP values, known-spell mask, carried instances and twelve worn slots. Each
item instance retains code, kind, ordered effects and stored price. An ordinary effect's stored
price uses the aggregate Magic-row point cost; `castSpell` uses its spell scalar and power. The
result is clamped at 9,999,999. This closes the difference between a base item price and the live
effect-adjusted price.

Brian starts from `PC_Paladin`: B/R/M/S `41/39/25/21`, skills `[0,25,3,0,0,1]`, XP
`[0,9834,331,0,0,100]`, aggregate 10,265, no spells or carried items, and seven worn instances in
slots 1, 6, 7, 8, 9, 10 and 12. Their codes are `0201106`, `0206110`, `0207116`, `0208119`,
`0209122`, `0210126`, `0212130`; their stored prices are 4,800, 2,700, 1,020, 4,200, 1,800, 1,200
and 2,400, with no effects.

Mission-70 Naira starts from `PC_Naira_2`: B/R/M/S `39/41/22/26`, skills `[0,24,5,0,0,38]`, XP
`[0,8849,610,0,0,36404]`, aggregate 45,863, no spells or carried items, and nine worn instances.
The enchanted instances are `0901220`, to-hit +22, stored price 91,488; `0504201`, air protection
+16, stored price 37,245; and `0212129`, fire protection +19, stored price 52,822. The other six
codes and prices are `0505202`/3,200, `0506208`/4,480, `0207116`/1,020, `0508218`/8,000,
`0509220`/3,200 and `1110124`/16. The base `PC_Naira` aggregate 1,593 is not reachable from this
producer.

## 2. Live handover and ordinary hero surfaces

**FR4.** The simulation remains the handover producer. After each reported step, the map driver
asks `BoundarySurvivors(SelfSlot)` for living self-owned Human actors and joins only roster-backed
actors not already named by the mission's runtime id list. `CarryRosterIDs` returns the same actor's
live skills, XP, container and equipment with the parallel runtime id. No reconstruction, fresh id
or name-specific patch occurs. The mission party and `Start.IDs` change before rearm, notices or the
viewer projection can observe the step.

**FR5.** A new member enters the ordinary guarded/player-character set in that same tick. Its
equipment derives body, class and map art through `HeroAppearance`; its character subject is built
through the existing party panel path. The joined roster member's figure directory and face replace
the placement lookup before the post-step projection. A member selected while still foreign
therefore keeps its localized name, character card and fully equipped composed figure in the first
post-handover pane even while the inventory subject still names the previously selected party
member. The next ordinary frame refreshes the inventory subject from the retained selection.
Selection, map naming, character card, doll, inventory, equip, unequip and item use all address the
live world entity. Item commands run through the normal queued simulation command, post-step rearm
and projection paths.

## 3. Save, campaign boundary and terminal state

**FR6.** An in-mission save stores joined actors once, in the existing world bytes. `Snapshot.Party`
contains only the party prefix minted at mission entry. Restore mints that prefix to recover stable
map ids, substitutes the saved world, then discovers each already-owned roster actor and rebuilds
the live party surfaces before the first final projection. Joined skills, XP, effects, prices,
equipment, container, current pools and canonical hash therefore remain the saved world's values.
No byte-form or save-envelope field is added.

**FR7.** Mission completion carries the dynamic party and parallel runtime ids through
`FinishMissionWithRoster`. The existing boundary keeps actors in the researched Human persistence
band, refreshes live carry state and mints them as ordinary members in town and the next mission.
The same runtime id is never appended twice. Normal once-only trigger state and campaign
progression prevent a completed map producer from replaying; no name, row or companion-identity
guard is invented for forced replay. Dead candidates are excluded by `BoundarySurvivors`; loss and
already-member death retain the existing party rules.

## 4. Shipped witnesses

**FR8.** The release population test reads all ten exact rows from each lawful install and compares
literal source statistics, skills, XP, spell masks, figures and item counts. Separate exact Brian
and Naira checks compare every worn slot, item code, kind, ordered effect and stored price. The
production-route witness drives the mission-40 hero to Brian's trigger and the mission-70 hero to
Naira's trigger after selecting the mission-local actor while it is still foreign. In the first
tick that observes ownership it checks the dynamic party, localized player-character state, body
art, all twelve live equipment instances, figure identity, selected portrait, doll compositor and
composed character pane without changing the selection or inventory subject directly. The next
ordinary frame refreshes the inventory subject. The witness then performs a reversible equipment
interaction, encodes and decodes a save, restores it and opens the next campaign mission. The same
test runs independently on EN and RU without launching ROM1 or a GUI.

## Design decisions

**DD1.** Campaign row selection lives with `NPCDefs`, because the registry flags and mission tier
are data facts. Map and town producers consume one answer.

**DD2.** `rosterTemplate` remains the single exact row-to-member constructor. `CampaignNPCMember`
reaches it through a definition-id placement rather than copying its field graph.

**DD3.** Live handover is observed after `StepReported`, the statement that changes owner. It is not
predicted from a trigger latch or repaired from a display name.

**DD4.** The world is canonical live state. `mission.party` carries identity inputs and a current
boundary snapshot; inventory, equipment, combat and save bytes continue to read the world.

**DD5.** Save dedup is structural: the entry prefix is minted and the joined map actor remains only
in world bytes. It does not depend on a durable hero id the original does not have.

**DD6.** Dynamic party ids remain positional and parallel. `CarryRosterIDs` extends the existing
boundary result rather than adding a second actor-to-member transfer implementation.

**DD7.** Effect-adjusted price is constructed with the item instance, before any roster or world
consumer sees it. Tooltip, shop, serialization and persistence therefore share one stored integer.

**DD8.** Release expectations are literal outputs of EXP-0242. No expected payload is read back from
the selector under test, and EN and RU are separate executions.

## Bounds

No GUI or original executable is launched. No ordinary transferred low-type Human becomes
persistent. No new save form, generic progression system, original-current-HP constant or forced
replay identity rule is introduced. Brian may earn skill progress before transfer on a live route;
the transfer preserves that current state instead of resetting it to the source-initial literal.
No divergence id or research id is allocated by this story.
