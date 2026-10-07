# Story `1046` — closure

As-built evidence. `contract.md` is the historical dispatch; `spec.md` owns current behaviour. This
file owns aspect closure, the producer-to-consumer census, shipped witnesses and research
reconciliation.

Research pin `be95a8b482cfed678625b5a7e4c8fa29c17264e6` resolves the former AMBER dependency.
Production commits are `07bf6054`, `00ea68d7`, `7cd7d731` and `5994c652`; normal implementation
master merges are `ffbcf026` and `c1b9d53f`. Simulation form 61 remains current.

## Result

All five shipped persistent join origins now select their exact campaign-tier Humans row before
construction. The ten sex/class variants carry their full statistics, skills, XP, spellbook and
item instances. Brian is `PC_Paladin`, not a name over an ordinary-person row. Mission-70 Naira is
`PC_Naira_2`, aggregate XP 45,863, not base `PC_Naira` at 1,593. Enchanted worn items retain their
effect-adjusted stored prices, including Naira's 91,488 bow, 37,245 ring and 52,822 boots.

A map transfer becomes an ordinary party hero in the same tick that the simulation changes owner.
Its mission-local figure identity is installed before the projection, so a preselected actor has
its localized card, body art and fully equipped composed figure in that tick. The same live actor
supplies the later inventory subject, equipment commands, save bytes and campaign carry. A save
mints only the entry-party prefix and keeps joined actors in the saved world, so restore neither
drops nor duplicates them. The production EN and RU routes both drove Brian and Naira through join,
inventory, reversible equipment, save/load and the next mission.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `CampaignServerID` enumerates five origins and ten source variants. `CampaignNPCMember` and map placements share `rosterTemplate`. Literal EN/RU tests compare every row's statistics, skills, XP, spell mask, figure and item count. |
| Runtime state | PASS | `syncJoinedHeroes` observes post-step ownership, appends the roster-backed live actor and parallel id once, and updates the running `Mission` before rearm or projection. |
| Simulation | PASS | Skill levels, per-slot XP, spell mask, item effects, stored prices, combat rearm and canonical world bytes remain on the same entity. No form or hash rule changes. |
| Player input | PASS | Joined actors enter the guarded/player-character population. Production-route tests select them and drive ordinary queued unequip/equip commands. |
| AI | N-A | No AI discriminator or policy changes. A transferred actor follows the existing player-owned group and command rules. |
| UI/HUD | PASS | Body and class derive from live equipment. A foreign actor selected before handover retains its selection, localized name and card; `syncJoinedHeroes` installs its roster figure identity before the same-tick push. On both lawful roots, fixed Brian, both `npc23` branches and all four `npc24` branches produce a pane pixel-identical to the exact joined figure with all twelve live equipment slots. The two Naira panes differ from both Danath and naked-Naira controls. The prior inventory subject remains unchanged until the next ordinary frame binds inventory to the retained selection. |
| Triggers/scripts | PASS | The four map producers remain simulation instants 19/22. The town producer constructs through the same exact-row path. Brian and Naira are exercised through their shipped trigger routes. |
| Inventory/equipment | PASS | Brian's seven and Naira's nine exact worn instances, empty containers, effects and stored prices are literal witnesses. Reversible interactions preserve the complete instance. |
| Persistence/save-load | PASS | Joined actors are stored in world bytes, omitted from the snapshot mint prefix, rediscovered after world substitution and present once after encode/decode/restore. Form 61 is unchanged. |
| Campaign/session | PASS | Completion carries the dynamic party through `FinishMissionWithRoster`; Brian enters mission 70 and Naira mission 100 once with their live XP and equipment. Normal progression prevents producer replay. |
| Shipped content | PASS | Both lawful roots select the same ten rows and numeric payloads. Separate EN and RU executions complete both production routes. |
| Interactions with existing mechanics | PASS | Runtime-id dedup, Human-band cull, dead-candidate exclusion, loss, already-member death, item commands, rearm, notices and legacy synthetic starts retain their existing rules. |

No in-scope aspect is `GAP`.

## Producer census

| Producer | Shipped aliases and branches | Construction | Runtime transfer |
|---|---|---|---|
| town mission 30 `AddHero=22` | Fergard for a female primary; Reniesta for a male primary | `CampaignNPCMember` selects server 28/29 and calls `rosterTemplate` | town activation consumes the grant once |
| mission 40 `npc25` | Brian for every primary | fixed server 42 `PC_Paladin` | unit 6/group 14/player 2, instant 19/22 handover |
| mission 70 `npc23` | Danath for a female primary; Naira for a male primary | server 30/31 at tier 1 | unit 151/group 105/player 6, instant 22 handover |
| mission 100 `npc24` | Fergard, Reniesta, Danath or Naira from primary class and sex | server 36/37/34/35 at tier 2 | unit 245/group 113/player 7, instant 22 handover |
| mission 140 `npc26` | Rood Glaen for every primary | fixed server 43 `PC_Elf` | unit 443/group 16/player 5, instant 19/22 handover |

The release population test executes all ten source variants on each root. The map loader's stable
unit ordering and roster map connect each map source record to its live actor. No other persistent
producer exists in the parsed shipped population established by `HERO-JOIN-120`.

## Consumer census

| Consumer class | Closed path |
|---|---|
| identity and ownership | `PartyMember.ID`, `CompanionNPC`, `PlayerCharacter`, world owner, runtime id list and guarded set |
| definition and appearance | exact Humans row, profile, figure dir/face, class, `HeroAppearance`, body art and world/doll equipment composition |
| progression and spells | six stored levels, six live XP integers, aggregate sum and `KnownSpells` on mint, save and next mission |
| item state | exact carried and worn `ItemInstance` values, aggregate/cast effect price, inventory subject, equip, unequip, item use and rearm |
| presentation | map draw, localized name, character subject, card, doll, worn boxes, pack and headless projection |
| persistence and hash | existing world binary and canonical hash, snapshot entry prefix, decode, world substitution and restored discovery |
| campaign boundary | dynamic `Mission.Party`/`Start.IDs`, `CarryRosterIDs`, `FinishMissionWithRoster`, town carry and successor mint |
| terminal and duplicate state | existing runtime-id skip, dead-candidate exclusion, Human persistence band, loss path and once/progression replay prevention |

No payload field is parsed only to be discarded. Presentation-only body inputs remain outside the
world form by design; their source member is rebuilt from the same exact roster row on restore and
campaign mint.

## Exact shipped witnesses

On each lawful root, the fixed payload test reads Brian's skills
`[0,25,3,0,0,1]`, XP `[0,9834,331,0,0,100]`, aggregate 10,265 and seven exact worn instances.
It reads Naira's skills `[0,24,5,0,0,38]`, XP `[0,8849,610,0,0,36404]`, aggregate 45,863 and nine
exact worn instances. It compares every occupied and empty slot, kind, ordered effect and stored
price.

The mission-40 route moves the primary to `(76,108)`. Brian may fight before the handover and earn
progress; the witness requires every XP slot to remain at least its source value, then preserves the
exact live transfer value through save/load and mission 70. The mission-70 route moves the primary
to `(38,108)`, satisfying the shipped nearest-player condition for group 105. Naira joins with the
literal source XP, her ring moves from slot 4 to the pack and back, and the same instance survives
save/load and mission 100. Both routes select the mission-local actor before handover. The first
iteration that observes player ownership retains the previous inventory subject and proves the
joined actor's localized name, hero/player status, body art, all twelve live equipment instances,
figure identity and exact composed character pane. A separate seven-variant witness repeats the
foreign-actor selection and GiveUnit step for fixed Brian, both `npc23` branches and all four
`npc24` branches. Its expected pane is composed from the exact roster figure, the complete live
equipment and the installed pane art without reading the Viewer's picture state. The next
production frame binds inventory to the retained selection; no direct subject switch is used.

## Research reconciliation

- `HERO-JOIN-120` closes the five-origin, ten-variant population and selector formula.
- `HERO-JOIN-121` and `122` identify Brian's fixed Paladin row and Naira's mission-tier row.
- `HERO-JOIN-123`..`125` supply literal progression, derived source values, item instances, effects
  and stored prices.
- `HERO-JOIN-126` establishes same-object transfer and ordinary permanent Human consumers.
- `HERO-JOIN-127` establishes town grant consumption and the absence of a forced-replay identity
  guard. The implementation therefore deduplicates only a runtime actor already in its parallel id
  list and relies on normal trigger/progression state for source replay.
- `HERO-JOIN-128` establishes EN/RU numerical equality and localized-name-only differences.

The implementation cites only active claims at the exact pinned commit. No divergence exists:
effect-adjusted price and same-object transfer follow research, while save mint-prefix dedup is an
againrom representation choice for preserving one actor, not a forced-replay identity rule. No
divergence id or research id was allocated.

## Remaining surface

The named surface is exhausted across all five producers, ten selection variants, exact Brian and
Naira payloads, first post-transfer party/UI state, seven joined-figure viewer branches, item
interaction, save/load, successor mint, runtime-id duplicate suppression and dead-candidate culling.
The viewer witness fails when the same-tick composed-human portrait push is removed and when a joined
face is changed. W-1 is closed in the witness and requires no open ledger row. Brian's current HP and
possible earned skill progress at transfer are deliberately not constants; the original preserves
the combat-lived actor. Forced replay beyond normal campaign progression remains outside the
contract and has no identity guard in either the research model or this implementation.
