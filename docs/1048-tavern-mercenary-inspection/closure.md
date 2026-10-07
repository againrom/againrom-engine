# Story `1048` — closure

As-built evidence. `spec.md` owns behaviour; this file owns aspect closure, shipped witnesses,
research reconciliation and the remaining review surface.

Base `cd65f373`; contract commits `f0566043` and `ccf9bada`; name-card master `de31ac4a`, merged as
`a718380e`; production commits `66df5821` and `f081cbfd`; live-speaker master `2e744f8c`, merged as
`991928f8`; DIV-429 master `8f6e68e3`, merged before the pass-1 correction; pass-2 correction
`658e1db1`; exact master `7c4b3586`, merged after that correction as `56d193a0`; research pin
`be95a8b4`. Simulation form 62 remains current.

## Result

The tavern now starts without a candidate selection. Its mission NPCs and mercenaries share the
shipped compact portrait grid. Selecting a mercenary animates only that card and projects its
installed role/class name, computed statistics and actual equipped figure into the left column.
Every worn piece, including a completely covered real ring, exposes the ordinary full item tooltip;
the inspection owns no equipment control or mutation path.

Talk reads the selected type's installed candidate script and opens the ordinary modal with the
selected candidate's picture. Human candidates use their inspection/Hire worn figure; Catapult and
Ballista use their class flat tier portrait. The event speaker record supplies only the face-window
origin on this owner-directed path. The finite thirteen-type mapping succeeds in EN and RU. A
missing or malformed external leaf reaches a disclosed diagnostic instead of generic translated
prose.

Compact-card text uses the existing production font2 instance. The full 107-row shipped price
population composes inside the 48×64 card on EN and RU. Mission 150 type 13 shows `1000000` in full,
and the selected Hire button and hire debit retain the same value.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Campaign mission-NPC offers and thirteen mercenary types form one ordered stream. The candidate source map, `main.txt[84]`, both backgrounds, seventeen real sprite sheets, equipment slots and item lines are resolved from the active install. |
| Runtime state | PASS | One producer-stable key owns selection; a selected-type detail cache and app-only animation tick have explicit entry, removal, dialogue, exit and new-session transitions. |
| Simulation | N-A | Selection, hover, detail cache and animation remain client presentation. Hire continues through the existing game command. Simulation form and hash are unchanged. |
| Player input | PASS | Cards select on primary click; Hire/Talk require selection; disabled cards remain selectable; double-click resolves the selected identity. Empty space and every candidate-pane point refuse control and drag dispatch. |
| AI | N-A | No AI data, decision or update route changes. |
| UI/HUD | PASS | Six-column 48×64 cards use shipped art at one-to-one scale, font2, complete untruncated price/count streams at non-overlapping corners and selected-only frames. The left card/doll and existing Talk modal compose through production paths. Candidate Talk uses the selected human worn figure or siege class portrait, not the party/player speaker fallback. |
| Triggers/scripts | N-A | No trigger, mission script or register changes. The unsupported-node census is recorded unchanged in `verification.md`. |
| Inventory/equipment | PASS | Candidate generation reuses the real worn set and figure composer. Every occupied slot has normal full lines. `HoverSlotMask` adds read-only reachability for fully occluded layers without changing ordinary interactive masks or figure pixels. |
| Persistence/save-load | N-A | Candidate state and animation intentionally reset on room/session entry and are not serialized. Existing mercenary hire persistence is unchanged. |
| Campaign/session | PASS | Current mission offers and current chapter mercenaries are enumerated; accepted/removed producers cannot transfer selection. Dialogue return and still-present hire/return preserve selection; room re-entry and new game clear it. |
| Shipped content | PASS | The release witnesses enumerate every mercenary type, all 107 mission/type price rows, every candidate text/picture, every equipped slot, all campaign talk-only producers and every real frame on EN and RU. The candidate-picture expectation is derived directly from installed `Units`/`Humans` rows and low-level composers rather than the production speaker resolver. Type 14 is the male `mfighter/4` candidate on both roots. The only seven-digit price row is pinned to mission 150 type 13 at `1000000`. |
| Interactions with existing mechanics | PASS | One price producer serves presentation, affordability and hire/return. The widest shipped row is equal in the card, selected Hire value and debit. Stock, mission-NPC Talk, party panel naming, ordinary equipment controls and modal layout keep their existing seams. Ordinary dialogue retains the integrated live-speaker resolver; candidate Talk overrides only its picture with the selected candidate and retains the event record's face window. |

No in-scope aspect is `GAP`.

## Research reconciliation

- `TOWN-010`, as amended by `TOWN-065`, establishes one ordered candidate grid and shared numbering.
  Owner geometry selects the contiguous 48×64 placement where the decoded internal slot formula
  does not identify absolute screen rectangles. `TOWN-012` supplies the 1–15 and 29–30 sheet
  population; the real roots prove Unit29 and Unit30 are pixel-identical and the campaign reaches a
  three-offer chapter.
- `MERC-TYPE-001`, `MERC-SHELF-002`, `MERC-HIRE-003`, `MERC-PRICE-004`,
  `MERC-LEVEL-005` and `MERC-CMD-007` supply type, stock, generation level, price and whole-squad
  command semantics. Selection, read-only inspection and selected-only animation remain owner
  direction in `DIV-426`.
- `DLG-LANG-010`, `DLG-NPCTAG-018`, `DLG-NPCTAG-019`, `DLG-FIGURE-020`,
  `DLG-FIGURE-021`, `REG-TEXT-037`, `REG-KIND-056` and `REG-SCN-059` supply the existing
  localization, event-source, figure and portrait consumers. The candidate-to-leaf dispatch and
  selected-candidate picture mapping remain owner direction in `DIV-427`; malformed-data
  presentation remains `DIV-117`.
- `DIV-119` now records the explicit stable-selection precondition on the existing double-click
  command. No new research id is allocated and no unreviewed claim is consumed.

## Remaining surface

Pass 2 returned one P finding: candidate Talk resolved every installed `NPC=` tag through a town
cast containing only the party/player, so all thirteen paths substituted the player picture. The
correction supplies the selected candidate's production picture and adds an independent all-type,
both-root expected-picture witness. The final pass-3 surface is the corrected candidate-picture
delta: thirteen text-to-picture paths, human worn figures, both siege flat portraits, face-window
origin, dialogue paging/return and the separation from ordinary mission/NPC speaker resolution.
This story makes no independent Brian/Naira change. The exact post-correction master merge brings
the separate tavern performance and authored-weapon hotfix while leaving this selected-picture
delta distinct. Original-save work and all ordinary equipment mutation remain outside this story.

Pass 3 found no P finding and closed the remaining surface. It recorded one D, corrected in place in
`DIV-427`, and two W. Both W are blind witnesses over behaviour pass 3 measured directly and found
correct; neither returns the story.

W-1: the production capture reaches one of the thirteen paths. `cmd/screenshot`'s
`captureSelectedTavern` selects the first offer cell with `Portrait` and not `TalkOnly`, and the
chapter that opens the town finishes into two cells, of which cell 1 is type 14. The sanctioned
production witness therefore composes candidate type 14 and no other, on either root. The remaining
twelve paths are witnessed by `TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt`
alone. `captureSelectedTavern` is unchanged by this story and the selection predates it.

W-2: no committed test pages a real mercenary dialogue past part 1.
`TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt` compares part 1 only, and the
one paging and return test drives the diagnostic arm on a fixture with no archive, so it never
reaches `talk()`. Five of the thirteen leaves carry more than one part: types 1, 8 and 14 carry two,
types 9 and 13 carry three. Pass 3 measured the paging path through `TownDialogue()` and
`AdvanceTownDialogue()` on both roots and found the face pane pixel-identical on every part, the
room returning to `roomTavern` and both dialogue fields cleared, 13 of 13 on each root. Production is
correct and nothing committed can see it.
