# Contract — story 1048 tavern mercenary inspection

Open this story from exact implementation `cd65f373d62a875c1c66f461deeaab4cb1386779`
and exact research pin `23daf74f6ee83e5fee5474981faf7c276680cc9f`. Do not rebase or use stash.
At a clean committed boundary, merge the seat's member-name hotfix normally and adapt this story to
the one name drawn inside the mercenary card. The hotfix owns `MemberName`,
`townCharacterName`, the former separate DOLL name overlay and that overlay's hit exclusion/API
fields; this story does not edit or restore them.

## Outcome

The tavern begins with no mercenary candidate selected. A click on a bottom card selects that
candidate before Talk or Hire can operate. Only the selected card's miniature character animates.
The left column then shows that candidate's actual stats over a composed equipment doll. Every worn
item supplies the normal full item hover, while the whole inspection surface refuses equipment
click, drag, unequip, transfer and mutation.

The selected candidate's Talk button opens the existing modal dialogue composition with that
candidate's portrait and authored installed EN or RU text. The bottom cards keep their compact
geometry, price and `N/N` count/cap text, contain no squad name, and bring the production miniature
composer to the owner's supplied original-like scale, crop, alignment, palette, frame cadence,
background, border and text placement. The screenshot and shipped `manback.bmp`/`manbacktalk.bmp`
settle that geometry: six contiguous columns of 48×64 cards, each shipped background and sprite at
one-to-one scale, with price right-aligned at the top and count/cap left-aligned at the bottom over
the miniature. This replaces the build's current authored five-column 56×112 brown boxes.

## Authority, dependencies and domains

- Owner screenshots
  `owner-clipboard-7e314091-6b3f-44e0-8137-a603f682c74d.png` and
  `owner-clipboard-b58c2d8d-8c0f-4214-a636-9bd937a93669.png` are the visual and interaction
  authority where promoted research is silent.
- `MERC-TYPE-001`, `MERC-SHELF-002`, `MERC-HIRE-003`, `MERC-PRICE-004`,
  `MERC-LEVEL-005`, `MERC-CMD-007`, `TOWN-009` through `TOWN-014`, `TOWN-064`,
  `TOWN-065`, `TOWN-087`, `TOWN-136`, `TOWN-138`, `TOWN-214`, `TOWN-222`, `TOWN-282`,
  `DLG-WIN-001`, `DLG-LANG-010`, `DLG-NPCTAG-018`, `DLG-NPCTAG-019`,
  `REG-TEXT-037`, `REG-KIND-056` and `REG-SCN-059` bound the decoded data and existing
  presentation seams. None establishes click-to-select, selected-only animation or a read-only
  mercenary equipment doll.
- The lawful installed population is the thirteen offered types whose localized leaves are
  `text/inn/mercenary/npc01.txt` through `npc10.txt` and `npc12.txt` through `npc14.txt`.
  `npc35.txt` is not a mercenary candidate. The implementation resolves this finite mapping through
  existing resource, localization and dialogue readers; it does not embed translated prose.
- Touched domains: **Assets & Formats**, **Party, Items & Heroes**, **Town & Economy** and
  **Client**. Campaign data remains an input at its existing boundary. The story does not change
  simulation state, canonical hashing, byte forms or persistence.
- Amend `DIV-117` from a generic placeholder to installed candidate-authored Talk resolution and
  its disclosed malformed-data diagnostic. Reconcile `DIV-119` with the new selection precondition
  without changing its existing double-click command seam. `DIV-426` and `DIV-427` are reserved for
  owner-directed selection/read-only detail/selected-card animation and candidate-specific Talk
  presentation. Stop and ask if both are exhausted; do not allocate another id locally.

## Behaviour

1. The candidate producer is one ordered stream: available mission NPC offers followed by the
   current chapter's available mercenary types. Empty or unavailable producers create no card.
   Selection is explicit and stable by producer identity, never inherited from a former numeric
   index. Entry or re-entry to the tavern starts with no selection. Removing a producer clears its
   selection rather than selecting a neighbour. Hiring or returning a still-present type preserves
   its selection and refreshes its `N/N`, price, enabled state and detail.
2. A single primary click on an enabled or disabled bottom card selects it. Before selection, Talk
   and Hire are disabled. After selection, Talk and Hire resolve only that identity; Hire also
   observes the existing availability and affordability rules. The existing same-cell double-click
   seam may hire or return only after the first click has selected that cell. Empty card space and
   non-candidate panes do not select or mutate anything.
3. A selected mercenary projects the same current-level template, computed stats, equipment and
   item instances used by the production hire path. The left upper pane shows that identity's
   actual statistics. The lower pane uses the existing party figure/equipment compositor and slot
   geometry. Every occupied equipment slot routes to the existing full item tooltip lines and
   placement. Empty slots have no tooltip.
4. Candidate inspection is read-only. Pointer motion may change only hover presentation. Press,
   release, repeated click and drag over occupied or empty doll slots cannot arm inventory state,
   call a party/equipment mutation, transfer an item, change stats or create a sack. Tavern buttons
   and cards remain the only controls in this surface.
5. Only the selected bottom miniature advances through the shipped `sprites.16a` frame stream on
   the client presentation clock. Unselected cards remain on their canonical still frame. Changing
   selection freezes the old card and animates the new one; losing selection freezes all cards.
   Simulation time and hash do not consume this clock.
6. The miniature composer uses the shipped frame and palette through the existing art decoder. Its
   production draw call owns a six-column grid of contiguous 48×64 cells at the bottom of the
   centre pane, filled left-to-right and then upward. It draws `manback.bmp` or
   `manbacktalk.bmp` and the sprite frame at one-to-one scale with no crop, resampling or invented
   tint. Price is right-aligned at the cell's top edge; `N/N` is left-aligned at its bottom edge,
   both over the picture in the shipped card colour. It does not draw a squad name, an outer brown
   box or the retired `N for` form. The selected state changes only the rule this contract names:
   that cell advances its shipped animation frames while every unselected cell holds frame zero.
7. Talk resolves the selected mercenary type to its installed `text/inn/mercenary/npcNN.txt`
   payload, opens the existing modal, and obtains text and portrait through the existing
   localization/dialogue and portrait seams. All thirteen candidate paths resolve in each EN and RU
   root. Only a demonstrably missing or malformed external payload may show a concise diagnostic
   fallback; no valid shipped candidate takes that path, and the fallback is not translated prose.
8. Closing or exhausting the modal returns to the tavern without silently changing the selected
   identity. Leaving the room and entering again resets selection and presentation phase. New game,
   campaign replacement and candidate-population changes cannot retain stale identity, tooltip,
   animation or action ownership.

## Witnesses

- Production composition geometry, pixel and text-stream tests independently pin card bounds,
  miniature source-frame pixels, scale/crop/alignment, palette output, selected/unselected borders,
  animation frame ownership, top-right price, bottom-left `N/N` and absence of squad-name text. A
  deliberate one-pixel mutation at the production miniature draw call must make the geometry
  witness red and is then reverted byte-for-byte.
- Production input tests enumerate initial selection, every candidate producer, disabled and empty
  cards, transitions, same-cell and cross-cell click/double-click behavior, Talk/Hire dispatch,
  hiring, returning, producer removal, re-entry and reset.
- Candidate-detail tests compare projected statistics, composed doll slots and tooltip text with
  the same production party/item sources. Hover over every occupied slot is covered. Click and drag
  across every occupied slot, every empty slot and pane margins prove no item, equipment, stats,
  party or command state changes.
- Installed release tests enumerate all thirteen valid candidate paths in both EN and RU, require
  nonempty localized authored text and a production modal portrait, and prove malformed or missing
  external data is the only diagnostic route. They render with real shipped frames and palettes and
  exercise both selected and unselected miniature states.
- Headless screenshot modes expose the unselected tavern, selected inspection, next selected
  animation frame and selected Talk modal through the production composer. No GUI or game window is
  launched or controlled.
- The repository Go chain, all repository check scripts, both-root release and scenario gates, and
  the mission-10/mission-20 unsupported census run on the frozen candidate. The milestone counts are
  expected to remain at their inherited baseline; any change is a finding, not normalized away.

## Twelve-aspect closure

Data, runtime state, input, UI/HUD, inventory/equipment, shipped content and interactions with
existing mechanics are in scope. Campaign/session is limited to room entry, exit, reset and the
existing candidate producers. Simulation, AI, triggers and persistence are N-A because inspection
state and animation are client presentation only and hiring continues through the unchanged game
command seam. Any evidence that this story changes one of those four classifications is a GAP and
returns the work to specification before landing.

## Bounds and atomicity

Do not redesign mercenary generation, economy, mission offers, party equipment, item tooltips,
dialogue composition or portrait resolution. Do not add parallel art, localization, figure,
tooltip or item-data paths. Do not change squad names, `N/N`, byte forms, simulation hashes, saves
or hiring semantics. Do not launch the game GUI.

This contract names more than five numbered behaviours because the owner explicitly bound the card
selection, inspection, original-like miniature presentation and authored Talk modal to the same
surface and forbade a competing hotfix. They share one selected-candidate identity and the same
card compositor/action consumers. Splitting them would duplicate state and ownership; the slice
touches four domains and no hashed state.

## Review ceiling and stopping condition

At most three fresh adversarial passes. Only P returns the story. Stop at the first fresh no-P pass
with the remaining surface empty; apply W and D without another pass. The initial remaining surface
is: candidate producers and unavailable cases; selection, transition and reset lifecycle; card
frame, animation, pixel, palette, geometry and text composition; statistics, doll, every equipment
slot and full tooltip; pointer, click and drag refusal; buttons, hire/return/removal; candidate Talk
source, modal text, portrait and diagnostic; and both EN/RU installed populations.
