# 0157 round 3 — the shop screen finished

Pipeline v2 contract, written before the implementation. The behavioural
contract in full is `spec.md`, canonicalized to as-built at this landing. The
closure evidence is `closure.md`. `plan.md`, `provenance.md` and
`verification.md` are round 2's records and are not rewritten.

## What works after this round

The town shop screen the owner plays shows no black boxes, carries the shopping
character's dressed doll with a party picker in its bottom-right corner, turns
whichever region the mouse wheel is over, and draws every string legibly on both
installs.

## The observable result

`cmd/shopdump` composes the screen against a lawful install and writes it as a
PNG. Pure-black coverage of the composed 640x480 frame falls from 21% to 9% on
the English root, and the bottom-right corner from 88% to 0%. The screen also
gains the coin, the doll, the picker and a legible message line, which the PNG
shows and the number does not.

## Owner directives this round answers

Reviewed on 2026-08-15, with screenshots:

1. No black regions. Price digits sat on black boxes, empty slots were flat
   black, the money panel showed black squares.
2. The character block in the bottom-right corner: the shopping character's
   dressed doll, a party picker, and the picked member's inventory, switching
   together.
3. The wheel scrolls the region under the cursor - over the shelf line it pages
   the rack, over the inventory it scrolls the inventory.
4. The garbled glyphs on the hover line.

## Claims

Read from the research pin `6a927ee` with `go run ./tools/claim <ID>`.

| Claim | Confidence | What is taken from it |
|---|---|---|
| `SHOP-SCREEN-030` | High | The five child regions and their rectangles |
| `SHOP-SCREEN-031` | High | The three cell formulas |
| `SHOP-SCREEN-032` | High | Which resource paints each region |
| `SHOP-SCREEN-033` | High / Medium | The two arrow rectangles and the doubled step |
| `SHOP-SCREEN-034` | Active, partly refuted | Superseded here by `SHOP-SHELF-047`; only `view+0x132`'s no-shelf value is still used |
| `SHOP-SCREEN-035` | High / Medium | The four button rectangles, their four numbers and their four commands |
| `SHOP-SCREEN-036` | High (amended) | The five cell-background arms, including the money arm and the quantity-zero arm |
| `SHOP-SCREEN-037` | High | The plaque family by side, the plaque by digit count, the right-alignment |
| `SHOP-SCREEN-038` | High | No shop-screen graphic is localised, so one address set serves both roots |
| `SHOP-SCREEN-039` | Medium | The grids print exactly two text runs a cell |
| `SHOP-USABLE-040` | High / Medium | The class usability test and that it reaches the background only |
| `SHOP-FIGURE-041` | High | The character panel's rectangle and its re-parenting |
| `SHOP-FIGURE-042` | High / Medium | The 160x240 canvas, the blit at `+0`, `+2`, the composition cache |
| `SHOP-PICKER-043` | High / Medium | The two picker rectangles, the wrap, and the strip rebinding in the same call |
| `SHOP-VIEW-044` | High / Medium | The 640x480 frame and the flat fill under the backpack strip |
| `SHOP-MERCHANT-046` | High | Where the merchant stands, 76x176 at (277,112) |
| `SHOP-SHELF-047` | High | The four shelf HIT rectangles, index for index against the draw rectangles |
| `SHOP-MONEY-048` | High | The coin, one 80x80 frame at `graphics\interface\money\money.16a` |
| `SHOP-LIMIT-049` | High / Medium | The engine limits, and the borrowed-panel seam |
| `DLG-FIGURE-020` | High | The world compositor's 160x240 output |
| `DLG-FIGURE-021` | High | No layer is excluded at a dialogue-site call |

## What is UNKNOWN

- Whether the original's blitter treats pure black as transparent. The blit
  routine `SHOP-SCREEN-036` reaches (`vt+0x38`) is not decoded. What the shipped
  art establishes is that the plaques and the buttons carry their shape as a
  black surround, which no opaque blit can draw. Ledger row DIV-013.
- Where the money element sits in a container's list. `SHOP-SCREEN-036` names it
  by `element+0x6 == 0xffff` and does not give its index. This build puts it
  first. Ledger row DIV-014.
- The picker's own art. `SHOP-PICKER-043` grades that clause Medium: no art was
  tied to either rectangle. Ledger row DIV-015.
- Which room rectangle holds which kind of stock. Not decoded; read off the
  shipped room picture. Already carried as spec D-5.

## The twelve aspects

| Aspect | Applies | Why |
|---|---|---|
| Data | Yes | Four new install resources reach the screen: the coin, and the black key over the existing bitmaps |
| Runtime state | Yes | The shown member index and the per-member figure cache |
| Simulation | No | The shop is front-end state; no `pkg/sim` field moves |
| Player input | Yes | Two new controls, one new hit region, and the wheel's region rule |
| AI | No | Nothing acts |
| UI / HUD | Yes | The whole round |
| Triggers / scripts | No | Nothing scripted reaches the shop |
| Inventory / equipment | Yes | The strip is the shown member's container; the doll is his equipment |
| Persistence / save-load | No | No serialized field changes; `formatVersion` stays 50 |
| Campaign / session | Yes | The panel and the strip read the carried party, which is session state |
| Shipped content | Yes | Both roots, and the campaign's own town chapter |
| Interactions with existing mechanics | Yes | The wear rule, the town dialogue's speaker figures, and every drawn string in the build |

## Domains touched

`docs/DOMAINS.md`: **Client** (the screen, the doll, the picker, the wheel, the
drawn-text rule), **Party, Items & Heroes** (whose container the strip is, whose
class the wear rule asks), **Town & Economy** (the purse on the money element and
on the buttons; no pricing rule changes).

## Out of scope

The shelf animations. The two shipped tip texts. The consumables shelf. The
merchant sprite: both his static picture and his Yes and No poses, so his region
stays empty (DIV-020). Re-parenting one panel object between two screens.
Anything in `pkg/sim`.
