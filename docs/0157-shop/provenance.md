# 0157 — provenance

## Claims this build reads

| Claim | Confidence | What is taken |
|---|---|---|
| `SHOP-GEN-005` | High on the counts, partially retracted | The generator clears all shelves and refills. Shelf counts 100 weapons, 100 armour+shields, 20 magic items. The laziness clause is retracted and is not used. |
| `SHOP-POOL-006` | High on the window, partially retracted | The candidate is a `(tier, material bit, shape row)` triple. The mask is a `u16` at `entry+0x1c + 2·tier`, five of them per row. Kind 1 is Shields + Armors, kind 2 is Weapons. `tier == 5` means all five tiers. The admission value is superseded by `SHOP-POOL-021`. |
| `SHOP-POOL-021` | High | The admitted value is `ftol(price × Materials[m].+0x30 × Shapes[t].+0x30)`. `+0x30` is double 2 of the nine-double record. |
| `SHOP-PRICE-011` | High | `item+0x1c = ftol(base × material factor × tier factor + 0.5)`. The price is a property of the item and reads no customer. |
| `SHOP-ROUND-017` | High per site, Medium on completeness | `__ftol` truncates toward zero. The buy path executes no floating point. |
| `SHOP-MISSION-018` | High | The campaign ceiling is `[Mission N] ShopMaxPrice` in coins with no multiplier. |
| `SHOP-MISSION-019` | High | The window floor is the literal 0 at all four pool constructions. `ShopMinPrice` is read and never applied. |
| `SHOP-MISSION-020` | High on "the ceiling is the only per-mission input" | The ceiling is the generator's only per-mission input. |
| `SHOP-TOWN-022` | High | `SetCap` and `Generate` run together on the way into the town, and the ceiling sent is the next mission's. |
| `SHOP-TRAY-024` | High | One strip of five 80×80 places, shared by both sides. |
| `SHOP-TRAY-025` | High | A take is a split from the source container into the tray. `elem+0x14` is the ownership stamp: set on the customer's side, clear on the shop's. Clearing the table sends each element home by that test. |
| `SHOP-TRAY-026` | High | Two commit buttons. Buy sends only when the buy total is non-zero and `purse − total >= 0`; sell sends only when the sell total is non-zero. |
| `SHOP-TRAY-027` | High on the instructions, Medium on the unit price | The screen's sell total is `ceil(price/2) × qty` per element; the payout is `ceil(qty × price / 2)`. |
| `SHOP-BUY-009` | High | `gold -= qty × unitPrice` per element, walked in order. The abort is unreachable behind `SHOP-TRAY-026`'s guard. |
| `SHOP-SELL-010` | High | `gold += ftol(0.5 × (qty × price) + 0.5)`, i.e. `ceil(qty × price / 2)`, applied to the whole stack. A sold item returns to a shelf. |
| `SHOP-SAVE-015` | High | No part of a shop's stock is written to a save. |
| `SHOP-RNG-008` | High | The original seeds the CRT `rand()` from a clock. No assortment is reproducible. |
| `SHOP-DUP-028` | High | Nothing dedups generated stock. Two draws of one triple are two separate elements. |
| `SHOP-LIFE-013` | High | Generation clears every shelf before it fills. |
| `SHOP-MAGIC-007` | High on the two stages, Unknown on what an effect is | Mode 6 is the union of the armour and weapon pools, then an enchantment stage. |
| `SHOP-ENTRY-003`, `SHOP-ENTRY-016` | High | Single player uses the static town shop; a map-placed shop is unreachable there. |
| `SHOP-SCREEN-030` | High | Five children of one view, their control ids and their rectangles: shelf grid (0,0,164,303), backpack (0,390,480,480), table (0,303,480,390), merchant panel (164,0,480,303), button panel (464,0,640,238). |
| `SHOP-SCREEN-031` | High | The three cell formulas: shelf `(1+80c, 31+80r)` two columns of three; table `(32+80c, 303)` five; backpack `(32+80c, 395)` five. Every cell is 80x80. |
| `SHOP-SCREEN-032` | High on the paths and the sizes, Medium on the table overrun | Which resource paints each region, by path, and its measured pixel size. The table's blit asks for 480 px from a 472 px picture; whether the overrun is clipped was not read, so this build does not reproduce it (spec D-5). |
| `SHOP-SCREEN-033` | High on the rectangles, Medium on the step | Two arrow rectangles, (46,0,118,32) and (46,271,118,303); the shelf grid is the only one of the three that scrolls, by a whole row of two cells. |
| `SHOP-SCREEN-034` | High | Four shelf rectangles on the room picture, (201,20,313,108), (313,20,445,108), (197,108,277,220), (353,108,433,220); selecting one returns the shelf grid to its first row; no shelf is selected on entry. |
| `SHOP-SCREEN-035` | High, Medium on the button-to-picture pairing | The four button rectangles, the four numbers printed on them — purse, buy total, sell total, sum — and the four commands: clear, buy, sell, clear and leave. |
| `SHOP-SCREEN-036` | High on the arms, Medium on the usability test | What one cell draws, in order, and the background selection. The per-hero usability test is not reproduced: the routine behind it was not read, so this build selects on affordability alone (spec FR-14). |
| `SHOP-SCREEN-037` | High | The plaque is chosen by the price's digit count and the side of the deal, right-aligned to the cell's right edge at `cell.top + 1`; the figure is `ceil(price/2)` on the player's side and the price on the merchant's; the plaque is drawn on every priced cell. |
| `SHOP-SCREEN-038` | High | No shop-screen graphic is localised: 132 of 134 resources are byte-identical on the English and Russian installs. This build reads the same addresses on both. |
| `SHOP-SCREEN-039` | Medium | The original's item grids print the quantity and the price and nothing else — no item name and no characteristics panel. This is the premise A-4's hover is disclosed against. |
| `PARTY-MONEY-024` | — | The starting purse of 100, already in `Town`. |
| `REG-SCN-064` | — | `[Mission N] ShopMaxPrice`, already read by `Campaign`. |
| `ITEM-DISPNAME-036` | — | The shipped name table, already read by `Table.Names`. |

## Ours by choice

**A-1 — the shop's stock is reproducible from campaign state.** `SHOP-RNG-008`
establishes the original's generator is clock-seeded, so its stock is not
reproducible. This build seeds from the live chapter, the number of missions
finished and the ceiling. Consequence: reloading a save in the town shows the
same goods, where the original showed different goods. `SHOP-SAVE-015` is still
honoured — no part of the stock is written to a save; the seed is recomputed
from state the save already carries.

**A-2 — a uniform draw over the pool.** `SHOP-GEN-005` names the counts and
`SHOP-RNG-008` names the two uniform helpers, but no claim states the
distribution `R1711` draws a candidate with. A uniform draw over the
admitted triples is used.

**A-3 — the table holds five and refuses a sixth.** `SHOP-TRAY-025` states the
tray container has no capacity test. `SHOP-TRAY-024` states the widget has five
places, bounds its hit test to `0..4` and refuses a display-list append past
`cols·rows`. The model holds five.

**A-4 — the magic shelf is unenchanted.** `SHOP-MAGIC-007` grades what an effect
is as Unknown. The shelf is filled from the union pool the claim names, with no
second stage.

**A-5 — a bought item lands in the first party member's pack.** Nothing decoded
names which container the campaign's static shop credits; the party's own pack
model has one list per member.

**A-6 — the characteristics hover.** `SHOP-SCREEN-039` establishes the original's
shop grids draw no item name and no characteristics panel. The owner ruled on
2026-08-15 that every item on this screen carry a visual and a characteristics
hover, so the hover is authored on his ruling and disclosed as spec D-1. The
lines are `itemInfoLines`, already shipped: no item vocabulary is invented for
this screen.

**A-7 — which room rectangle holds which shelf.** `SHOP-SCREEN-034` gives four
rectangles and does not say what any of them stocks. The pairing is read off the
shipped room picture by what is drawn in each rectangle: potions upper-left,
magic upper-right, weapons lower-left, armour lower-right. Disclosed as spec D-6.

**A-8 — the bottom-right corner of the frame.** The five decoded regions leave
(480, 238)-(640, 480) unpainted and nothing decoded names it. This build puts the
message line and the mission offer there. Disclosed as spec D-7.

## Open

- **The consumables shelf is not built.** `SHOP-POOL-021` states it is built
  from the Spells collection as item subtypes `0x2a` and `0x29`, and
  `SHOP-GEN-005` names six literal potions from Magic Items. This tree has no
  item-code composition for either class, and none is published. Golden rule 4:
  not guessed.
- **A magic item's own price field.** `SHOP-ROUND-017` enumerates four rounding
  sites and all three price constructors are Armor, Shield and Weapon. What
  writes `item+0x1c` for a Magic Items row is not published. Not needed here,
  because the consumables shelf is not built.
- **The restock timer** (`SHOP-LIFE-014`) is multiplayer-only and is not built.
- **The partial-stack move** (`SHOP-TRAY-027` Unknown) is not built: a take moves
  the whole element.
- **Where the merchant stands.** `SHOP-SCREEN-032` names `movies\shopanim\...` as
  the merchant panel's third picture and gives no draw origin for it. The
  archive it is in is not among the four this front end opens. Spec D-2.
- **The two tip texts.** `SHOP-SCREEN-038` names `main\text\tips\shop1.txt` and
  `shop2.txt` and where they are loaded; where they are drawn is not decoded.
  Spec D-4.
- **The money cell's second graphic.** `SHOP-SCREEN-036` grades the identity of
  `[L09760]` as Unknown. This build's shop screen has no money cell.
- **Whether the table's 8 px source overrun is visible.** `SHOP-SCREEN-032`
  grades it Medium because the blitter's clipping was not read. Spec D-5.
- **The per-hero usability test** behind `R0882` (`SHOP-SCREEN-036`,
  Medium). The `backinv`/`backinvs` selection here reads affordability alone.

## Removed

The shop's own dialogue lines. The building already speaks the shipped
`shop/npc31m%d.txt` text through 0142's conversation path, unchanged.
