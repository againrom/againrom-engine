# Story `1048` — tavern mercenary inspection

This is the canonical as-built specification. `contract.md` records the dispatched result and
authority boundary. Base `cd65f373`; name-card master `de31ac4a`, merged as `a718380e`;
live-speaker master `2e744f8c`, merged as `991928f8`; exact master `7c4b3586`, merged after the
pass-2 correction as `56d193a0`; research pin `be95a8b4`. Simulation form 62 remains current.

## 1. Candidate identity and actions

**B1.** The tavern exposes one ordered candidate stream. Current mission-NPC offers come first and
current chapter mercenary types follow. Each NPC offer is keyed by its chapter-local offer index;
each mercenary is keyed by type. Empty, locked, exhausted and otherwise unavailable producers
create no cell. List compaction never transfers selection to a numeric neighbour.

Every tavern entry and full session reset starts with no selected candidate. Hire and Talk are
disabled until a card is clicked. A primary click selects an affordable or unaffordable card. Talk
uses the selected identity; Hire additionally requires a selected mercenary which the existing
economy path says is available and affordable. A first release selects. A later same-card double
release may call the existing whole-squad hire/return seam; it cannot act on a former selection.

Dialogue return, successful hire and successful return preserve a still-present identity and
refresh its price, count, capacity and enabled state. Removing that producer clears selection.
Leaving and re-entering the tavern clears selection, candidate detail, hover ownership and the
client animation phase.

## 2. Shared compact roster

**B2.** Mission-NPC offers and mercenaries are portraits in the same six-column stream. Cells are
48×64 pixels, contiguous from x=176 through x=464, filled left-to-right on the bottom row and then
upward. NPC offers precede mercenaries in that numbering. Offer cells use `manbacktalk.bmp` and the
shipped talk-only `Unit29`/`Unit30` frame stream. The installed campaign reaches three offers in one
chapter; the stable offer ordinal cycles the two shipped sheets. Their complete eight-frame streams
are byte-identical in EN and RU, so the cycling choice is pixel-neutral. A negative or malformed
external offer index receives no invented sheet.

Mercenary cells use `manback.bmp` and the selected type's shipped `UnitN/sprites.16a` stream. Both
background and sprite draw at the cell origin at one-to-one scale, with no crop, resampling, tint,
lighting transform or authored overlay. A selected or hired state adds no border. The shipped
background owns the border. The art-backed tavern adds no `TAVERN` title over the centre picture.

**B3.** Mercenary price is right-aligned at the card's top edge. Current/capacity is the literal
`N/N` stream, left-aligned against the bottom edge. Both use the measured card colour
`#bd9e4a` and the existing production font2 instance. The full decimal price is painted without
truncation. The shipped campaign contains 107 mission/type cost rows. Its only seven-digit value is
mission 150 type 13 at `1000000`; font2 measures it at 40 pixels inside the 48-pixel card. Price and
`N/N` occupy separate top and bottom bands. The visible price, selected Hire value and hire debit
are the same whole-squad value. No squad name, semantic label, `N for` copy or outer brown box is
painted. Talk-only cards paint neither mercenary field.

Only the selected candidate consumes the client presentation phase. Its card advances through the
shipped frame stream once per six client update ticks. Every unselected card holds frame zero.
Selection change freezes the old card and starts the new card at phase zero. The clock is absent
from simulation, hashing, saves and game time.

## 3. Candidate inspection

**B4.** Selecting a mercenary builds one member through the same `buildMercenarySquad` template,
level, equipment and recompute path used by Hire. The upper-left 160×238 pane projects that member
through `partyPanelSubject` and the ordinary compact statistics-card compositor. Generated source
keys such as `NPC03_1` are not display names. The human template's installed `TypeID` selects the
localized class name. The installed global `main.txt[84]` role sits on the row above it, producing
the screenshot's `MERCENARY` / `CLUBMAN` shape in EN and the corresponding installed RU text.
Siege candidates retain their production class identity.

The lower-left 160×242 pane receives the ordinary equipment figure from
`composeInventorySubject`. No second body, equipment, palette, item or statistics model exists for
the candidate. The generated detail is cached only for the current selected type and is discarded
at every selection-loss and room/session reset boundary.

**B5.** Every nonzero worn slot gets the normal full `itemInstanceInfoLines` stream. The inspection
mask begins with the ordinary compositor's topmost slot mask and then retains one distinct pixel
from each authored worn layer which would otherwise be completely covered by later equipment.
This extra mask is returned only as `HoverSlotMask`; interactive inventory, shop and mission dolls
retain the ordinary topmost mask unchanged. The EN shipped type-10 Mithrill Ring is the discriminator:
its 59 authored pixels are all covered in the final doll, yet its full tooltip remains reachable.

The candidate panes are absent from `TownSurfaceControlAt`, shop controls, inventory controls and
every drag producer. Motion may change only the hover tooltip. Press, release, repeated click and
drag on an occupied pixel, an empty slot or pane margin cannot arm a held item, unequip, transfer,
create a sack, change party state or alter any statistic. Cards and the three tavern buttons remain
the only input controls on this surface.

## 4. Candidate-authored Talk

**B6.** `TownMercenaryTextPath` is a finite mapping for types 1 through 10 and 12 through 14. Type
11, type 15 and unrelated `npc35.txt` are excluded. A selected type reads
`main/text/inn/mercenary/npcNN.txt` from the active install through the existing archive and event
text readers. Part selection, hero audience and `NPC=` tag parsing keep their existing semantics.
The selected candidate owns the picture on this owner-directed path.

Talk opens the existing town notice/modal composition. The selected payload supplies the text and
its `NPC=` tag supplies the ordinary face-window origin. The picture is the selected candidate's
own production picture rather than the town resolver's party/player fallback. Human candidates use
the same worn-equipment figure as inspection and Hire. Catapult and Ballista use their unit class's
flat tier portrait. Closing or exhausting the modal returns to the tavern with the same selected
identity. Every mapped path has a nonempty part 1 and candidate picture in both lawful roots.

**B7.** A missing leaf or a payload with no valid first part is external-data failure, not another
candidate script. It opens the same modal with `Mercenary dialogue unavailable: npcNN`. No lawful
EN/RU candidate reaches this diagnostic, and no translated fallback prose is embedded.

Mission-NPC Talk is unchanged apart from its card joining the compact portrait stream. It continues
through the existing NPC/mission payload resolver and never becomes hireable.

## 5. Headless production evidence

**B8.** The screenshot command can reach `town-tavern`, `town-tavern-selected`,
`town-tavern-next-frame` and `town-tavern-talk` through `FinishMission`, ordinary room entry,
`TownSurfaceClick`, the phase-aware production composer and the existing Talk button. It selects a
real stocked mercenary rather than the preceding talk-only card. The command opens no GUI and
writes only requested PNGs under its output directory.

## Design decisions

**DD1.** A producer-stable `(kind,id)` key owns selection. A display index is derived each frame and
never survives a candidate-population change.

**DD2.** The card compositor consumes decoded BMP and sprite-frame images. It owns placement only;
it does not decode, recolour, resize or manufacture art.

**DD3.** Candidate generation reuses Hire's one-member production builder and the ordinary party,
equipment, item-tooltip and panel projections. The read-only surface holds no mutable party member
or inventory command handle.

**DD4.** `HoverSlotMask` is a presentation-only companion to the existing interactive mask. It is
computed during the same layer walk and cannot change the composed RGBA or ordinary doll hit rules.

**DD5.** The installed role is another row in the normal panel model. It does not restore the
removed separate DOLL member-name overlay and does not replace `PanelSubject.Name`.

**DD6.** Candidate Talk maps only the finite installed population and reuses the modal and event
parser. Its candidate-picture seam reuses the inspection/Hire figure for human candidates and the
normal class-portrait loader for siege candidates. The event speaker record contributes only its
face-window origin on this owner-directed path. Diagnostic text names data failure and is never
treated as localization.

**DD7.** Animation is one app presentation counter, reset on screen/room boundaries and divided by
six only when composing. It never reaches `FrontEnd`, `sim.World`, canonical state or persistence.

## Bounds

This story does not change mercenary generation, stock, price, whole-squad hire/return semantics,
mission offers, item effects, ordinary equipment interaction, dialogue grammar, ordinary
mission/NPC portrait lookup,
simulation, AI, triggers, canonical hashes, byte forms or save envelopes. It does not address the
Brian/Naira mission regression independently: it inherits the shared speaker path from master.
Exact master `7c4b3586` also brings the separate tavern performance and authored-weapon hotfix;
those rules remain outside this story's candidate-picture behaviour. `DIV-117`, `DIV-119`,
`DIV-426` and `DIV-427` carry the remaining research and owner-directed boundaries.
