# Story 1286: shop and school presentation

## Intent

Re-read the five open presentation rows of the shop, school and town speech
(DIV-1550, DIV-143, DIV-142, DIV-1263, DIV-1404) against every claim promoted
since the rows were written, implement what the claims establish where the
player sees a difference, and state the rest as open questions.

## Authority

Pin k122. Population searched: claims added to `claims/shop.md`, `town.md`,
`dialogue.md`, `text.md`, `menu.md` and `mission.md` between k100 and k122
(TOWN-485..497, SHOP-096..105, DIALOGUE-068..070, TEXT-080..092,
MENU-051..056), plus a text search of every claim file for the school clock
global, the school state arrays and the speech and teacher terms. B1 holds: no
question below carries an expected answer.

## As-built behaviour

- DIV-1550, narrowed. The shop grid's quantity and the purse in the money cell
  draw in the ramp (185k/15, 159k/15, 73k/15) over a flat 8 8 8 shadow one pixel
  right and down, at the cell's left plus 10 and bottom less 15, in the mission
  font (`drawShopQuantity`, `pkg/ui/shopscreen.go`). The cell paints one
  routine for both arms (`SHOP-SCREEN-036`), the ramp and shadow are
  `MISSION-MSGLINE-056`, the grouping `TOWN-469`. Font and alignment flag of
  that call stay undecoded; the row stays OPEN for them. DIV-1405's sentence
  on the quantity is amended to match.

## Rows verified, not changed

- DIV-142: no claim after `TOWN-147` names a producer or reader of the school
  clock global. `TOWN-491`/`TOWN-492` describe the town hub's own clock and
  gate, a different global.
- DIV-143: `TOWN-151` states that the per-slot state dwords hold bit
  combinations written by the hit test and that -1 draws nothing. No claim names
  the value they hold at room entry or what paints the icons at rest, so the
  player-visible result of the reset is not established.
- DIV-1263: the new dialogue claims (`DIALOGUE-068..070`) concern the part
  parser and trim, not voice selection or lifetime. No claim names the teacher
  response key reader (`TOWN-022`) or the speech request call sites.
- DIV-1404: no claim reads a hover brightening, the school painter's pressed
  arm, or the shop pass's second colour and offset (`SHOP-050`, `SHOP-051`).

## Proof

- `TestReleaseShopGridQuantityAndPurseAreRampedWithAShadow` (EN and RU): the
  App's shop frames on each of the four shelves, then a synthetic population of
  counts (2 to 9,999,999) in all three grids and two money cells, compared with
  an oracle that decodes font1 and draws the claimed ramp and shadow. Loss
  controls: a pixel left, up and right, no shadow, the shop's former text colour,
  a black shadow and font2 glyphs each differ from the screen. With
  `drawShopQuantity` at the previous revision (shop text colour, no shadow) the
  witness fails at 432 pixels on the first shelf.
- `TestShopGridGroupsQuantityPriceAndMoney` and
  `TestShopPaintDrawsQuantityAfterTrailAndAdvancesVisibleSlots` follow the new
  ink.

## Open debt

Falsifiable questions for research, with no expected answer:

- Which font and alignment flag does the grid painter pass for the quantity
  and money text (DIV-1550, DIV-1405)?
- Which routines write the school clock global and which paint routines of the
  school read it, and how often does the school paint (DIV-142)?
- What do the school's state dwords hold when the room is entered and after a
  hit test leaves a slot, and which routine paints the skill icons when the
  dword holds -1 (DIV-143)? Which routines read the element flag bit 3, the
  class-change marker and the timestamp the picker step writes?
- Which routine requests the teacher response speech and which key it reads,
  and which call sites request the shop and tavern speech (DIV-1263)?
- What colour and offset does the shop button panel's pressed-and-hovered pass
  use, and what does the school painter do for a pressed command (DIV-1404)?
