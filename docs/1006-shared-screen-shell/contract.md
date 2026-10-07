# 1006 — shared screen shell

## Result

The tavern, skill school, shop and detailed character-generation stage use one screen shell. The
shell has a left content region and a right column. Each screen supplies its own upper-right
controls. The town-room lower-right region shows the selected party member through one shared
character-panel path. Detailed character generation uses the same composition but shows only its
preview doll on the right and its full statistics in the symmetric left panel.

The tavern immediately hires or returns complete mercenary squads. The school selects a skill and
buys one level through a separate Train control. The shop keeps all behaviour shipped by 0157 and
1005, including all interactive-doll behaviour. Its independent Book toggle replaces the trade
table with the selected member's learned spells and restores the unchanged table when disabled.
Character generation keeps the existing pre-create and detailed flows, including validation; only
the detailed stage moves to the shell. Town mission dialogues and mission offers remain available.
The shared town-room hero controls provide previous/next member selection and a doll/statistics mode.

The owner can verify the result in `builds/current/` on both lawful installs by entering all four
surfaces, changing the selected party member in every town room, hiring and returning a squad,
buying a selected skill level, switching the shop between its unchanged table and spellbook, trading
through two party members, using the interactive doll, and completing character generation through
the unchanged pre-create and migrated detailed stages.

## Owner directives

The following requirements are authored behaviour. They are not statements about ROM1:

- Member arrows work in every town room. Doll/statistics shows the full character panel. A room may
  enter in doll mode; persistence of that mode between rooms is optional.
- Tavern cells select on one click. A double-click immediately toggles hire or return for the whole
  squad. Hire uses the same toggle path. Talk and dialogue NPC conversations are repeatable. EXIT
  shows the current purse and only leaves.
- A school skill click selects the purchase and displays its price. Train shows the selected price,
  spends it and raises that skill by one. Train is disabled when the purse is insufficient. General
  is not offered. The visible order matches detailed character generation. EXIT shows the purse.
- The shop has one Book toggle. It is independent of doll/statistics and replaces the trade table
  with the selected member's learned spells. Member changes rebind all member-owned shop surfaces,
  while the trade table remains unchanged for cross-member transfers.
- Only detailed character generation uses the shell. Its right side contains the preview doll
  without member arrows or a statistics toggle. Its left side contains the full character panel.
  Its upper controls are Accept, Reset and Back.

## Claims

The claims were read from research pin `fcdb02e`. All listed claims are active unless stated
otherwise.

| Claims | Confidence and status | Contract use |
|---|---|---|
| `TOWN-086`, `TOWN-087`, `TOWN-088`, `SHOP-FIGURE-041` | High, active | One lower character panel is constructed once and borrowed by the shop, tavern and school |
| `TOWN-094` | High, active | The three upper-right widgets are distinct; shop and school extend 16 pixels into the left region |
| `SHOP-SHELF-047` | High, active | Shop shelf hit and draw rectangles remain distinct during migration |
| `MERC-TYPE-001`, `MERC-HIRE-003` | High, active | Mercenary types, whole-squad hire and dismiss semantics |
| `MERC-SHELF-002` | High / Medium / Unknown, active and partly retracted | The offer intersection and permanent unlock list; the retracted early-visibility reading is not used |
| `MERC-PRICE-004` | High / Medium, active | Mercenary price formula and the unresolved alternate price mode |
| `MERC-CMD-007` | High / Unknown, active | Town tavern command payloads; map-placed taverns remain unresolved |
| `TOWN-061`, `TOWN-017`, `TOWN-019`, `TOWN-067` | High, active | School draw order, two-panel hit dispatch, per-slot enable flags and class assignment |
| `TOWN-018` | High / Medium, active | Five mage and five fighter skill columns and their shipped art |
| `TOWN-068` | High / Unknown, active | Shared five-slot geometry and unresolved visual row order |
| `HERO-SKILLBUY-076` | High / Unknown, active | Skill purchase mutation, price and slot bounds; its former producer residual is resolved below |
| `TOWN-GENERAL-106`, `TOWN-GENERAL-107`, `TOWN-GENERAL-108`, `HERO-GENERAL-086` | High, active | Five school choices, no General, exact ROM1 visual mapping, price-query producer and Train producer |
| `SESS-HERO-013`, `TEXT-CHARGEN-028` | High, active | Four hero choices, the name prompt and the pictorial pre-create controls |
| `UNIT-GATE-014` | High, active and amended | Three difficulty choices; the caption is owner testimony, not a decoded string |
| `HERO-STAT-001`, `HERO-COST-002`, `HERO-BUY-003` | High, active | Four stats, cumulative point cost, bounds and click semantics |
| `HERO-BUDGET-004` | High / Unknown, active | The 140 total budget and the server-side out-of-range residual |
| `HERO-CHARGEN-082` | High, active | Detailed reset preserves identity and skill while rebuilding the preview |
| `HERO-CHARGEN-083` | High / Unknown, active | Detailed Back returns to pre-create; the next Forward discards detailed edits |
| `HERO-CHARGEN-084` | High / Medium, active | Ordered final-name validation and localized refusal text |
| `HERO-CHARGEN-085` | High / Unknown, active | Actor creation occurs only after successful validation; the original rejection destination is unresolved |

## Known unknowns and expected divergence work

- `TOWN-GENERAL-107` and `TOWN-GENERAL-108` resolve the school price-query and Train producers.
  No producer divergence remains.
- `TOWN-GENERAL-106` establishes ROM1's visual-to-stored-slot order as `1,2,4,3,5`. The owner
  requires detailed character generation's `1,2,3,4,5` order. `DIV-121` records that conflict.
- The tavern's within-chapter unlock timing and alternate price mode are unresolved
  (`MERC-SHELF-002`, `MERC-PRICE-004`). A choice needed for shipped play is recorded as `UNKNOWN`.
- Research establishes shared borrowing only for the lower panel. Using the shell for detailed
  character generation is the owner's authored composition and is recorded as `DEVIATION`.
- The landing reconciles existing rows `DIV-015`, `DIV-016`, `DIV-017`, `DIV-018` and `DIV-020`.
  It does not duplicate them. A real shared character widget is expected to close `DIV-017`.

Research Unknowns are ledger work, not implementation holds.

## Aspects

| Aspect | Verdict before implementation | Scope |
|---|---|---|
| Data | Applies | EN and RU screen art, text, mercenary and skill data |
| Runtime state | Applies | Active room, selected member, immediate hires, school selection and chargen draft |
| Simulation | N/A | No Sim Core field, command or digest changes; implementation must stop and reclassify if this proves false |
| Player input | Applies | Mouse, keyboard, wheel, drag and modal routing |
| AI | N/A | No unit decision changes |
| UI / HUD | Applies | The shell and all four surfaces |
| Triggers / scripts | Applies | Existing town mission offers and dialogues remain connected |
| Inventory / equipment | Applies | Shop containers, mercenary equipment and the shared doll |
| Persistence / save-load | Applies | Committed hires, training, purse and hero changes survive save/load |
| Campaign / session | Applies | Town entry, room transitions, party membership and detailed chargen |
| Shipped content | Applies | Campaign town content on both preserved roots |
| Interactions with existing mechanics | Applies | 0157 trade, 1005 doll, wear rules, dialogue modal and character creation |

## Domains

**Assets**, **Client**, **Town & Economy**, **Party, Items & Heroes**, **Campaign & Scripts** and
**Persistence**.

## Out of scope

The town square and world map. Map-placed taverns. New shop animations, tips or merchant poses beyond
the reconciliation of existing divergence rows. Pre-create composition changes. New
character-generation rules. Training General, whose ROM1 server semantics are decoded but which the
owner explicitly excludes from the school surface.
Sim Core, combat, AI, mission scripts and the serialized sim byte form.
