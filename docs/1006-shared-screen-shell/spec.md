# 1006 — shared screen shell — behavioural specification

This document is canonicalized to the implementation at `1006-shared-screen-shell`. Section 5
includes the interactive-doll behaviour merged from story 1005 and verified after the merge.

## 1. Screen shell

The shell composes a 640x480 town or detailed-generation surface from three owned regions:

1. the left content region, `(0,0)-(480,480)`;
2. the upper-right extension point, normally inside `(480,0)-(640,238)` and permitted to extend left
   to x=464;
3. the lower character region, `(480,238)-(640,480)`.

The shell owns composition order, input routing, modal layering and the lower character region. A
screen owns the data and controls it installs into the left region and upper-right extension point.
The shell does not define a common upper widget. Tavern, school, shop and detailed character
generation may use different control types and geometry.

Every town-room upper widget contains a visible EXIT action. It is enabled whenever the room is
active. Tavern EXIT and school EXIT leave without changing the selected cell or skill. Shop EXIT uses
the established clear-table-and-leave command. Escape invokes the same room-leave result. An open
dialogue covers and disables EXIT until the dialogue is advanced or dismissed.

The shop and school may use `(464,0)-(640,238)`. In the overlap `(464,0)-(480,238)`, an upper-right
control is drawn above left content and receives input first. A point not claimed by that control may
reach left content. The tavern uses `(480,0)-(640,238)` unless its own shipped geometry requires less.

Every child reports its hit regions in frame coordinates. Input is resolved in this order:

1. an open modal dialogue;
2. the upper-right extension point and lower character region;
3. left content;
4. the room's Back or Escape action.

A press is owned by the region where it began until release. A drag that begins on the character
panel or an inventory control cannot become a left-content click. The wheel moves only the region
under the pointer when that region defines wheel behaviour. Controls covered by a modal dialogue
receive no hover, press, release, wheel or keyboard activation.

Missing optional art leaves only that art absent. The shell, text, hit regions and available fallback
furniture still compose. Missing data needed for a mutation makes the corresponding control visibly
unavailable and explains the refusal. No visible enabled control may be inert.

## 2. Shared character region

In town rooms, the lower character region uses the existing `PanelSubject`, `RenderCharacterPanel`,
`composeUnitFigure` and `memberFigure` seams. One resolved subject supplies the doll, statistics,
equipment and item information. No screen implements a second stat derivation, equipment-layer
order, figure selection or party-member identity rule.

The region owns one selected-member index. Its persistent control set contains previous member, next
member and a visibly stateful doll/statistics mode control. Previous and next wrap when the party has
more than one member and are disabled for a party of one. Doll mode draws the composed figure and
interactive equipment layers. Statistics mode draws the same `PanelSubject` through
`RenderCharacterPanel`; it does not recompute any value. A member change atomically changes:

- the doll and statistics;
- the hero named by school training;
- the shop container, learned-spell book, usability shading and item target;
- all hover and interactive-doll state associated with the shown member.

Previous and next are available in every town room. Each room resolves a valid selected member. A
room may initialize the view to doll mode. Preserving doll/statistics mode across room transitions is
optional. Detailed generation reuses the subject and figure seams but not these town-room controls.

The tavern lower character region shows the selected player hero. An immediate hire extends that
party and makes the hired members reachable through the same picker. A member removed by another
established game path cannot leave a stale selected index; the index clamps to a surviving member or
to no subject.

A missing figure produces an empty doll area while the member name, controls and resolved statistics
remain usable. An empty party produces no subject and disables previous, next and doll/statistics.
Disabled controls are visibly disabled and capture no action.

## 3. Tavern

The tavern is a shell surface, not a generic picker list. Its bottom cells contain both dialogue NPCs
and the live mercenary roster. Neither replaces the other. A single click selects one cell.

On a complete install the left child is assembled from `leftpicture.bmp` and `leftstats.bmp`, the
center child from `centerarea.bmp`, and a mercenary cell uses the first frame of its type-keyed
`unit<n>/sprites.16a`. These are resolved once at startup. A hand-built front end or an unreadable
cosmetic node uses the shell's text furniture and the same controls.

The hire roster contains only a type that is:

- named by the current mission's mercenary list;
- present in the permanent unlock list;
- backed by a non-empty surviving pool.

Being named before unlock does not make a type visible. A mercenary cell states the mercenary type,
squad size, whole-squad price, affordability and hired state. A cell represents the whole surviving
squad, not one member. Every offered squad may be hired; the tavern adds no count limit.

The normal price is `(PriceA + n * PriceB) * unitPrice(mission)`, using the mercenary type's values
and the mission price factor. The unresolved alternate flat-price mode is isolated behind the price
provider and reconciled in `docs/DIVERGENCES.md`; it is not spread through the screen.

Double-clicking an affordable available mercenary cell immediately debits its displayed price and
adds its complete squad to the party. Double-clicking the hired cell again immediately removes that
squad from the party, restores it to stock and refunds the same price. The Hire control applies the
same toggle to the selected mercenary cell. An unaffordable cell cannot be hired and changes no purse,
party or stock state. A failed toggle changes none of those states and reports the failure.

The upper-right controls are Hire with the selected squad price, Talk and EXIT with the current purse.
Hire is disabled when the selected cell is not a mercenary or cannot perform its current toggle. Talk
opens repeatable generic speech about the selected character. EXIT only leaves and is present when
the roster is empty.

Dialogue NPC cells still open the shipped town dialogue. Dialogue pages, mission acceptance, missing
or empty dialogue fallback, and the return to the tavern remain the established town behaviour. NPC
dialogues and generic Talk may be opened repeatedly. A dialogue is modal over the composed tavern. Its
button or Enter advances it; other tavern controls cannot act through it.

After a hire or return, save and load reproduce the purse, party membership and stock result. A hired
member's identity, stats, equipment and container also reproduce after save and load. Cell selection
is presentation state and need not be serialized.

If mercenary definitions or pictures are missing, the screen states that the affected type is
unavailable. It does not synthesize a mercenary or charge the purse.

## 4. Skill school

The school is a shell surface, not a generic picker list. It resolves the complete `TrnHall.bmp` and
`ButtonsArea.bmp` surface plus two distinct five-slot class families. The selected fighter uses the
92x120 fighter mask; the selected mage uses the 100x120 mage mask. Input reads the active mask byte
through the class's own five-arm permutation, then resolves the shared skill slot. The other class
is not a purchasable control for that hero. Both class families retain separate masks, icon arrays
and enable flags.

The screen shows the selected hero's current levels. A click selects one offered skill and displays
the price of its next level in the right panel. Train shows that price and performs the purchase. It
is disabled when there is no valid selection or the purse cannot pay. The class panels offer only
their five displayed skills. General is not offered.

The visual order is Blade/Fire, Axe/Water, Bludgen/Air, Pike/Earth and Shooting/Astral. This is
detailed character generation's stored-slot order `1,2,3,4,5`, as required by the owner.
`TOWN-GENERAL-106` establishes that ROM1 school controls instead map visual positions to stored
slots `1,2,4,3,5`. `DIV-121` records the deliberate conflict.

One Train activation applies to the selected hero and performs one mutation:

- increase the chosen skill level by one;
- set that slot's experience to `S(new level) + 1`;
- add the same experience delta to the hero's total;
- rebuild the hero's derived values;
- debit `trunc(1.1^baseLevel * 200)` gold from the purse.

Fixed price witnesses include 200 at base level 0, 518 at 10, 3489 at 30 and 23478 at 50. The base
copy, not a transient live modifier, selects the price. The underlying purchase command accepts slots
0 through 5, but this screen does not expose General. No authored ceiling is added where the published
purchase rule has none.

Insufficient gold, an absent hero, an invalid slot or a disabled slot changes no level, experience,
derived value or purse. The refusal is shown on the school surface. A visible enabled skill control
must either purchase or report a concrete server-side refusal; it may not stop at the published
party-list lookup.

`TOWN-GENERAL-107` establishes ROM1's selection and price-query producer. `TOWN-GENERAL-108`
establishes the separate Train producer. The implementation uses the equivalent
client-to-town-economy seam without reproducing packet opcodes. The shared slot rectangles use the
detailed character-generation mapping. Input, displayed icon and purchased slot all read that same
mapping.

The school's existing shipped mission dialogue and mission offer remain reachable. An open dialogue
is modal over the composed school and blocks purchases. Missing art leaves a labelled fallback control;
missing skill data disables that control. English and Russian roots resolve their own art and text.

The upper-right controls are Train with the selected price and EXIT with the current purse. EXIT is
always visible while the school room is active. It leaves without purchasing the selected skill.
Escape has the same result.

A successful purchase survives save and load with the same hero level, per-slot experience, total
experience, derived values and purse. No purchase changes Sim Core state or the serialized sim byte
form.

## 5. Shop migration

The shop moves onto the shell without changing the 0157 trade model or controls.

- Stock generation keeps its value window, 100/100/20 shelf counts, deterministic seed, clear-and-refill
  lifecycle and non-persistent stock.
- The five-place table keeps merchant and player ownership, guarded buy and sell commits, exact purse
  arithmetic, clear and leave behaviour, quantity handling and the shown-member container binding.
- The shelf grid, table, pack strip, money element, plaques, quantities, usability backgrounds, item
  hover, merchant control, message line, four buttons and wheel-under-pointer behaviour remain live.
- Shelf hit rectangles remain distinct from shelf draw rectangles. The four hit rectangles are
  `(354,110)-(459,295)`, `(169,110)-(274,295)`, `(314,5)-(454,105)` and
  `(172,5)-(314,105)`. Migration must not substitute the refuted draw-rectangle reading.
- The shop upper widget keeps its x=464 overlap and wins input inside that overlap.
- Missing archive art preserves the established total, non-panicking control and text fallback.
- Shipped shop dialogue remains modal over the composed shop and does not replace the shop background.

The lower-right shop painter is removed in favour of the shared character region. Composition is
shared as a widget and as resolved subject state, which closes the reason for `DIV-017`. Existing
rows `DIV-015`, `DIV-016`, `DIV-018` and `DIV-020` are retained, updated or closed according to the
as-built result; no duplicate rows are added.

The shop extends the shared hero controls with one visibly stateful Book toggle. Book is independent
of doll/statistics. When enabled, it replaces the trade table with the selected member's learned
spells. When disabled, it restores the unchanged trade table with the same items, ownership stamps,
purse and scroll state. There is no Inventory toggle. A member change rebinds the doll, statistics,
pack, learned-spell book, usability shading and item target. It does not change the trade table, which
allows cross-member transfers. The visible EXIT action is the existing fourth shop command: it clears
the table to the stamped owners and leaves in one action.

The merged story 1005 shop interactive-doll behaviours remain:

- hover over a worn layer shows the item's popup;
- a plain press on a worn layer unequips it through the existing unequip path;
- drag from the shown member's pack or a shop item source to the doll uses the established equip and
  purchase guards, including the wear rule and purse;
- drag from the doll redraws it without the carried layer and may return to the doll, the shown
  member's pack, or a valid shop target;
- the carried icon, suppression mask and hit mask follow the same shown member and composition;
- release cancellation changes no item, purse or table state.

The migration does not create a second drag machine, wear rule, slot mask or shop transaction path.

## 6. Character generation migration

Only detailed character generation uses the shell. This composition is owner-authored and is
recorded as a `DEVIATION`; it is not presented as researched ROM1 panel borrowing.

### 6.1 Pre-create outside the shell

Pre-create remains outside the shell and unchanged. It shows the install-backed name prompt, the four
pictorial sex/class choices, the three difficulty pictures, Back and Forward. All four hero
combinations remain selectable and map to the same class and sex values the current generator uses.

Name entry stores at most ten encoded bytes, preserves case, accepts only characters representable
by the active install encoding and supports backspace. English and Russian inputs use their own
encoding and install text. Forward requires only a chosen picture and advances without final name
validation. Back returns to the screen that armed character generation and creates no party.

### 6.2 Detailed generation

Detailed generation shows the selected identity, skill choice, four stats, remaining budget and full
character panel in a symmetric left-side composition. Its upper-right controls are Accept, Reset and
Back. The right-side region renders only the preview doll through the same subject and figure seams
used by the town screens.

Before actor creation, the character region's subject is the current draft preview. Detailed
generation has no previous-member, next-member, doll/statistics or Book controls. It cannot select an
existing party member. The full character panel remains on the left when the preview doll is absent.

The four stats start at 25. The remaining pool starts at 100, equivalent to a total cumulative-cost
budget of 140. A plus step charges `T(v+1)-T(v)` and is refused when the pool cannot pay or `v >= 45`.
A minus step refunds `T(v)-T(v-1)` and is refused when `v <= 15`. A refusal changes no stat, pool,
focus or preview. An accepted stat or skill change atomically rebuilds the preview from the complete
draft.

Reset restores all four stats to 25 and the pool to 100. It preserves name, difficulty, class, sex,
appearance and selected skill, and reconstructs the preview. Detailed Back returns to pre-create.
The next Forward rebuilds a fresh detailed draft from the pre-create identity and discards prior
detailed stat and skill edits.

Accept performs final validation before a live actor is created. The name must be non-empty, must not
case-insensitively equal `Self` or `Computer`, must not byte-exactly duplicate another participant
unless it is the same returning connection, and must not bind a returning participant that already
has a character. A refusal creates no actor or party member and leaves a usable error presentation.
The screen reached after a dismissed refusal remains Unknown and is reconciled in the divergence
ledger. A successful Accept commits the exact typed bytes to the participant and created
actor, then opens the requested campaign mission through the existing mission opener.

The generation migration does not change stat rules, costs, budget, skill choices, starting equipment,
profile derivation, mission-start recompute, cancellation destination or retired `-chargen` flag
behaviour.

## 7. Cross-screen requirements

Town square and world map behaviour are unchanged. Entering and leaving any migrated room returns to
the same town state the prior screen used. A member selected in one room cannot make another room
mutate a different member through stale cached state.

The implementation loads screen assets and strings through the existing asset root and archive
interfaces. It contains no game asset, install path or non-ASCII production string literal. The EN
and RU roots must each complete the following shipped-content route: enter tavern, inspect an NPC
dialogue, hire and return a squad, enter school, select and train one affordable level, enter shop,
select two party members, preserve the table across that change, inspect each spellbook, trade and use
the doll, traverse unchanged pre-create into detailed generation, Reset, reject one invalid Accept and
complete one valid Accept.

Save/load witnesses immediate tavern and school mutations through the production save path. Shop
presentation state, tavern cell selection and chargen drafts remain session presentation unless an
existing contract already persists them.

## 8. Acceptance criteria

**AC-1** One shell test composes four distinct upper widgets. The three town rooms reuse one lower
character fixture; detailed generation uses its preview-only variant. The shop and school widgets
receive a click at x=470 before left content; the tavern does not claim a point outside its own upper
widget.

**AC-2** A modal dialogue over each town surface consumes Enter, button presses and pointer input;
the covered hire, purchase and trade controls do not mutate.

**AC-3** Previous and next work in tavern, school and shop. They change the doll and statistics in all
three rooms, the school target in school, and the container, Book and usability state in shop. A stale
prior member is not mutated. A shop member change does not change the trade table.

**AC-3a** Doll/statistics switches between two views of the same subject without changing any derived
value. Previous and next wrap for a multi-member party and are disabled for zero or one member.

**AC-4** A mercenary type omitted from any one of mission list, unlock list or non-empty pool is absent.
One click selects a cell. Double-click or Hire immediately adds its complete squad and debits the
shown cost. The same action on the hired cell returns the squad to stock and refunds that cost. Every
available squad can be hired; an unaffordable hire changes nothing.

**AC-5** Tavern NPC dialogue still pages and accepts its mission offer after the mercenary roster is
present. NPC dialogue and generic Talk can be opened repeatedly. Tavern EXIT shows the current purse
and only leaves.

**AC-6** Each fighter and mage control resolves its displayed icon, enable flag and purchase slot
through the detailed character-generation mapping. A click selects a skill and displays its price.
Train changes only that skill and purse, with fixed price witnesses 200, 518, 3489 and 23478. General
is absent. Disabled Train and unaffordable selection change nothing.

**AC-7** A school purchase survives save/load with the same skill, per-slot experience, total
experience, derived values and purse. The school mission dialogue remains reachable. School EXIT
shows the current purse, is visible and does not buy the selected skill.

**AC-8** The shop's existing behavioural tests pass without weakened assertions. Shelf hit geometry,
table ownership, buy/sell arithmetic, member-bound pack, wheel routing, hover and missing-art fallback
produce the same results before and after migration.

**AC-9** Story 1005's shop interactive-doll behaviour passes through the shared character region.
Hover, plain unequip, both drag directions, suppression and cancellation mutate through the existing
paths.

**AC-9a** The shop's Book toggle is independent of doll/statistics and remains bound to the selected
member. Book replaces the table with learned spells and restores the unchanged table when disabled.
Member changes rebind doll, statistics, pack, Book and usability without changing the table. Shop
EXIT clears the table and leaves.

**AC-10** Unchanged pre-create retains four hero choices, three difficulty choices, ten-byte encoded
name input, Back and Forward outside the shell. Detailed generation retains point-buy, skill selection,
Accept, Reset, Back and final name validation in the shell.

**AC-11** Reset preserves identity and selected skill while restoring four 25 stats and pool 100.
Detailed Back followed by Forward discards the prior detailed stat and skill edits.

**AC-12** An invalid final name creates no actor and leaves usable error presentation. Its destination
remains Unknown. A valid name creates one actor with the exact encoded name and opens the requested
mission.

**AC-12a** Detailed character generation shows the full character panel on the left and only the
preview doll on the right. It has no member arrows, doll/statistics toggle or Book control and never
switches to a party member.

**AC-13** Missing optional art on each screen leaves a coherent shell and live text controls. Missing
mutation data disables the affected control and reports why. No fixture panics.

**AC-14** Production-root sweeps pass separately on EN and RU. Synthetic tests read no install.

**AC-15** The implementation changes no Sim Core source, sim command, digest fixture or serialized sim
format version. If implementation requires any of them, Stage 4 stops and this contract is reclassified.
