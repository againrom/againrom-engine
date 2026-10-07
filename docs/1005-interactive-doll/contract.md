# 1005 — the interactive doll

## The result

The equipment doll answers which item a pixel belongs to, on both screens that draw it.

- Hovering a worn item on the doll shows that item's popup — the same name and decoded stats the
  worn box and the pack bar already show.
- Pressing a worn item on the doll and releasing without moving takes it off, into the pack.
- An item can be picked up from the pack bar or a shop grid by pressing and moving, carried on the
  cursor, and released on the doll, where it is processed exactly as the screen's own equip path
  processes it, wear rule included.
- An item can be picked up from the doll the same way. The doll redraws without that layer for the
  duration of the drag, and the item can be released into the pack, onto a shop grid, or back onto
  the doll.
- During a mission, an item released outside every inventory box falls to the ground in a sack at
  the cell under the cursor.
- All of it works during a mission and in the shop.

Before this story the doll is one undifferentiated picture on both screens: `pkg/ui/invequip_test.go`
asserts that a press at the centre of the drawn figure names no cell, `pkg/ui/itempopup.go` does not
ask the doll box at all, and `docs/0151-layers-and-names/spec.md` records drag-to-unequip, clicks on
the doll picture and dropping an item on the ground as deliberate limits rather than oversights.
Nothing in the tree carries an item on the cursor between a press and a release, and no player action
can put a sack on the ground: `World.dropAll` is reachable only from a map trigger script, and the
only other sack sources are map placement and death.

Owner directive, 2026-08-16: «Кукла становится интерактивной: мы знаем что мы что-то налепили на
куклу, послойно. При наведении на слой — показываем Ховер предмета. При наведении на предмет на
кукле мы можем на нее нажать — это снимает предмет. Работать должно и в игре и в магазине. В магазине
и инвентаре можно взять любой предмет потянув его кликом drag и бросить на куклу, тогда предмет
процессится как будто его собирались надеть. Обратный момент тоже должен работать — можно драгнуть
предмет с куклы, тогда кукла сразу перерисовывается и предмет можно бросить в инвентарь или обратно
на куклу. В игре если предмет бросили за экран инвентаря, то он падает на землю в мешок.»

The observable result is in `builds/current/`: open a mission, hover and press the doll, drag between
the pack bar and the doll, drag off the window; then enter the town shop and do the same. It moves no
number in `pipeline/check-milestone.sh`'s script-gap census.

## Claims

| Claim | What it settles here |
|---|---|
| `ITEM-CMD-007` | Moving an item is one command with a source code and a destination code. Source 1 = an equipment slot, 2 = the actor's container, 3 = the ground, 4..8 = a shop shelf; destination 1 = equip, 2 = the container, 3 = the ground, 4..8 = a shop shelf. Every interaction in this story is one pair of that vocabulary, and none of them needs a new one. Round 2 (adversarial review, 2026-08-16) reads this range more precisely against `SHOP-TRAY-025` and `SHOP-SCREEN-031` — `spec.md`'s own "The five-place grid" section carries the correction; this row is left as originally written. |
| `ITEM-DROP-008` | A drop lands where it was asked only within a Chebyshev distance of 2, tested as two independent `abs(...) > 2` comparisons, so the window is a 5×5 square and not a radius. Outside it the drop goes to the dropper's own cell. It is never refused. Gold is a field of the sack rather than an item. |
| `ITEM-SACK-010`, `ITEM-SACK-011`, `ITEM-PICK-009`, `ITEM-PICK-016` | What a sack is, and how it is taken back up. The pickup path already exists in this build. |
| `ITEM-EQUIP-006`, `ITEM-ARMSLOT-031`, `ITEM-WEAR-055`, `ITEM-WEAR-056`, `ITEM-WEAR-057`, `ITEM-WEAR-058` | Which item may occupy which slot, and the class bits that gate it. The release-on-doll path applies the rule this build already enforces at `mapWorld.enqueueEquip`, not a second copy of it. |
| `HERO-BARE-037` | Taking a weapon off subtracts the three modifier fields its inverse adds. Unequip is a real state change and not a picture change. |
| `SHOP-FIGURE-041`, `SHOP-FIGURE-042` | The shop's character region is the mission screen's character panel, borrowed, and what it draws is the shown member composed in equipment through the same compositor the world view and the dialogue panel use. The two screens showing one doll is the original's own arrangement, not a convenience. |
| `DLG-FIGURE-021` | The compositor takes no layer selector, so no caller can exclude a layer. Suppressing a dragged layer is therefore done by composing a different equipment set, not by asking the compositor to skip one. |
| `SHOP-PICKER-043` | The party picker rebinds the figure and the item strip together. Which member the shop doll shows is which member's pack is shown. |

## UNKNOWN

Searched every claim ledger for `drag`, `hover`, `tooltip`, `unequip` and `mouse`. The four rows
below are what the search did not find; each is a divergence row, not a hold, on the 2026-08-15
ruling that an owner-directed feature proceeds on his intent where research is missing.

- No claim states whether the original carries an item on the cursor between a press and a release,
  nor what gesture its inventory uses. → `DIV-084`.
- No claim states whether the original hit-tests the doll picture to name a slot, nor how. This
  build answers it with a per-pixel slot mask built beside the composed canvas, which is an authored
  mechanism. → `DIV-085`.
- No claim states whether the original shows an item popup over the doll. → `DIV-086`.
- No claim states whether `ITEM-CMD-007`'s shelf-to-equip pair charges the purse. This build treats
  a shelf item released on the doll as a purchase followed by a wear attempt, and refuses when the
  purse cannot pay. → `DIV-087`.
- No claim states what the original's inventory screen does with a release outside its own surface.
  `ITEM-DROP-008` gives the geometry of the ground destination but not the gesture that reaches it.
  → `DIV-088`.

`DIV-089` through `DIV-092` are allocated to this story and returned if unused.

## Aspects that apply

Runtime state (the carried item and its origin, which live in the view and never in the world),
simulation (one new command kind for the ground destination, which does reach hashed state),
input (the whole story), UI/HUD (the mask, the popup, the cursor picture, the suppressed layer),
inventory/equipment (the subject), persistence (a sack put down by a player survives save and load),
campaign/session (the shop's purse and table), shipped content (items across shipped missions),
interactions with existing mechanics (the existing double-click equip and unequip paths, the pack
bar's scroll, and box-select on the map, which a drag beginning inside an inventory box must not
start). Data, AI and triggers are N/A: no table is added, no unit decision changes, and no script
opcode is touched.

## Domains

**Client** (8) carries most of it: the mask, hit-testing, hover, the drag machine, and both screens.
**Party, Items & Heroes** (5) for what an equip, an unequip and a drop mean. **Sim Core** (2) for the
one new command kind and its arm. **Town & Economy** (7) for the shop's own release targets and the
purse. **Persistence** (9) for a player-placed sack across save and load. **Assets** (1) is N/A: the
mask is derived from sheets already loaded, and no new file is read.

## Out of scope

Container capacity. Gold as a drag subject — `ITEM-DROP-008` puts gold on a different opcode and
makes it a field of the sack, and no screen in this build carries a coin the player can pick up.
Dragging between two party members. Any change to the figure layer order or to the compositor's
output pixels: the mask is built beside the existing composition and must not alter a single drawn
pixel, and the existing figure tests are the witness for that.
