# 0157 — the shop: stock, the table, and the two commits

This is the contract. Research provenance is in `provenance.md` (B2); nothing
here depends on reading it.

It has three rounds and this text is canonical as at the third. The first round
is the trade model and it is unchanged. The second is the screen. The third is
the screen finished: the owner reviewed it twice on 2026-08-15 and ordered no
black regions, the character block in the bottom-right corner, the wheel over the
region under the cursor, and the garbled glyphs fixed. FR-12 to FR-24 are the
screen.

## Contract

**The town shop holds generated stock, and the player trades with it across a
five-place table, on the screen the original draws.**

The merchant's shelves are filled when the party comes home. The player moves an
item from a shelf or from his own pack onto a table of five places that both
sides share. Two buttons commit the table: buy takes what the merchant owns and
debits the purse, sell hands over what the player owns and credits it. A third
clears the table.

## Scope

The stock generator and its value window, the shelves it fills, the five-place
table, the ownership stamp on a table place, the buy and sell commits with their
guards and their arithmetic, and the screen that shows all of it: its five
regions, its three grids of item cells, its shelf selection, its four buttons and
the hover the owner ruled onto every item.

## Terms

**Ceiling** — the value window's upper bound, `[Mission N] ShopMaxPrice` in
coins.

**Candidate** — one `(shape row, material, tier)` triple admitted by the window.

**Table place** — one of five slots holding an item, its unit price, its
quantity, and which side owns it.

## Functional requirements

**FR-1 — the value window.** A candidate's value is
`trunc(priceCell × material factor × shape factor)`, where `priceCell` is the
row's parameter slot 2 and each factor is slot 2 of that entry's nine-double
record. It is admitted when `0 <= value <= ceiling`. The floor is the literal 0
and no per-mission key moves it, which excludes the negative-price sentinel rows.
Every one of the five tiers is walked, and within a tier the row's own 16-bit
material mask at raw byte offset `2 × tier` decides which of the 16 materials the
row admits.

**FR-2 — an item's price.** The price a shop item carries is
`trunc(priceCell × material factor × shape factor + 0.5)` — the window's value
with the rounding addend. It is a property of the item and reads nothing about
the customer, so two customers at one shop are charged the same.

**FR-3 — three shelves, cleared and refilled.** Generating clears every shelf
before it fills. The armour shelf takes 100 draws from the Armors and Shields
rows together, the weapon shelf 100 from Weapons, the magic shelf 20 from those
two pools united. Each draw is one element of quantity 1. Nothing dedups: two
draws of one candidate are two elements.

**FR-4 — the shelves are filled on the way into the town.** The ceiling is read
from the live chapter and the shelves are generated in the same step, at every
arrival in the town and at every load of a game that is in the town. Nothing
regenerates on entering the shop room, and no part of the stock is written to a
save.

**FR-5 — the table.** Five places, shared. A place taken from a shelf is stamped
as the merchant's; a place taken from the player's pack is stamped as his. A take
removes the element from where it came from and puts the whole of it on the
table. A sixth take is refused and the table is unchanged.

**FR-6 — clearing the table.** Each place goes back to the side its stamp names:
the merchant's places to the shelf they came from, the player's places to his
pack. The table is then empty and no coin has moved.

**FR-7 — buy.** The buy total is the sum over merchant-owned places of
`quantity × unit price`. The commit runs only when that total is non-zero and the
purse is at least the total; otherwise nothing moves and the refusal is stated.
On commit each merchant-owned place is walked in order, the purse is debited
`quantity × unit price`, and the item enters the player's pack as `quantity`
units. The player's own places stay on the table.

**FR-8 — sell.** The commit runs only when the sell total is non-zero. On commit
each player-owned place is walked in order, the purse is credited
`ceil(quantity × unit price / 2)`, and the item goes back onto a shelf. The
merchant's own places stay on the table.

**FR-9 — what the screen states before the commit.** The sell figure shown is
`ceil(unit price / 2) × quantity` summed over the player's places. On an odd unit
price this exceeds what FR-8 pays by `floor(quantity/2)` per place. Both figures
are the original's, and the screen names the payout beside the total.

**FR-10 — the shop room.** It shows the two commits with their totals, the five
table places with what each holds and which side owns it, the player's pack, the
merchant's shelves one at a time, and the mission offer 0142 already put there.
A control that moves something is live and one that cannot is not.

**FR-11 — what a build with no item tables shows.** Shelves are empty, the room
says so, and no trade control is live. Nothing panics and no other room changes.

**FR-12 — the six regions.** The screen is the whole 640x480 frame. Five regions
are the shop view's own children, at fixed rectangles given as (left, top, right,
bottom): the shelf grid (0, 0, 164, 303), the merchant's room (164, 0, 480, 303),
the table (0, 303, 480, 390), the party backpack (0, 390, 480, 480) and the
button panel (464, 0, 640, 238). The sixth is the character panel at
(480, 238, 640, 480), which is the mission screen's own panel placed here for the
visit. Each is painted from the install resource the original names for it, read
through the archive filesystem at run time. No resource is carried in this
repository. A resource that will not read leaves its region unpainted and changes
nothing else on the screen.

**FR-12a — pure black is transparent in the screen's bitmaps.** Every 24-bit
bitmap this screen blits is drawn with its pure-black pixels cleared to zero
alpha. The shipped art carries its shape that way: `costs3.bmp` is 60x10 and
56.7% pure black, drawing its tablet only right of column 33, and
`shopbutton1.bmp` is 120x52 with black corners around a rounded plate. Sprites
loaded as `.256` or `.16a` keep their own structural or per-pixel transparency
and are not keyed.

**FR-13 — the three grids of cells.** Every cell is 80x80. The shelf grid holds
two columns of three at (1 + 80c, 31 + 80r); the table holds five at
(32 + 80c, 303); the backpack holds five at (32 + 80c, 395). A cell that holds an
element draws its background, then the item's own picture, then its quantity when
the quantity exceeds one, then its price plaque. The shelf grid shows six of the
shelf's items and the backpack grid five of the shown member's container; both
are turned rather than truncated (FR-15, FR-22).

**FR-13a — a place with no element.** A place of the shelf grid or of the table
paints nothing at all: its region's own picture shows through, which is the shelf
rack and the table. A place of the backpack strip paints `backinvg.bmp`, because
that region has no picture of its own — the shop view fills it flat.

**FR-14 — what one cell draws.** The background is `backinvs.bmp` for a shelf
item whose unit price the purse covers and whose class the shown member may use,
`backinv.bmp` for any other usable item, and `backinvg.bmp` for an item the shown
member's class may not use. The quantity is printed at
(cell.left + 10, cell.bottom - 15). The price plaque is `costm(d+1).bmp` on the
player's side and `costs(d+1).bmp` on the merchant's, where d is the price's
base-10 digit count minus one, clamped to 6; the bitmap is right-aligned to the
cell's right edge at cell.top + 1. The figure printed on it is centred on the
PLAQUE'S OWN INK — the smallest rectangle holding its non-transparent pixels —
and not on the 60x10 canvas, because the tablet occupies only the right of that
canvas and grows leftward with the digit count. The figure is `ceil(price / 2)`
on the player's side and the price itself on the merchant's. The plaque is drawn
on every priced cell whichever grid it is in.

**FR-14a — the money element.** The first element of the backpack strip's list is
the money element. It draws `backinv.bmp` with the 80x80 coin frame of
`graphics\interface\money\money.16a` composited over it, prints the purse where a
quantity is printed, holds no item and no price, answers no hover, and moves
nothing when it is clicked. It scrolls with the rest of the list.

**FR-15 — the shelf grid scrolls by whole rows.** Two arrows, at (46, 0, 118, 32)
and (46, 271, 118, 303), move the grid up and down by one row of two cells. The
first row is never scrolled above the top and a scroll never leaves the grid
empty while the shelf holds an item.

**FR-16 — the shelves are clicked in the room picture.** Four rectangles on the
merchant's room select one shelf each. They are the HIT array —
(354, 110, 459, 295), (169, 110, 274, 295), (314, 5, 454, 105) and
(172, 5, 314, 105), index 0 to 3 — and not the four rectangles the same routine
builds for DRAWING a shelf's animation, which are
(353, 108, 433, 220), (197, 108, 277, 220), (313, 20, 445, 108) and
(201, 20, 313, 108). This build marks the chosen shelf by outlining its draw
rectangle. Selecting a shelf shows its stock in the shelf grid and returns that
grid to its first row. No shelf is selected when the room is entered, and the
grid is then empty.

**FR-17 — the button panel.** Four rectangles, each carrying one number and one
command: (494, 15, 614, 67) clears the table and prints the purse;
(483, 67, 623, 113) buys and prints the buy total; (483, 114, 623, 160) sells and
prints the sell total; (494, 160, 614, 212) clears the table and leaves the room,
and prints the sum of the two totals. A command whose guard in FR-7 or FR-8
refuses is not live and its number is drawn unlit.

**FR-18 — what a click on a cell moves.** A click on an occupied shelf cell or an
occupied backpack cell puts that item on the table under FR-5. A click on an
occupied table place sends it back to the side its stamp names under FR-6. A
click on an empty cell, or on the money element, moves nothing.

**FR-19 — the characteristics hover.** Pointing at an occupied cell of any of the
three grids states that item's name and the characteristics this build decodes
for its class — the same lines the inventory screen's own item popup prints. It
is drawn beside the pointer, it moves nothing, and it disappears when the pointer
leaves the cell.

**FR-20 — the character panel.** The panel at (480, 238, 640, 480) draws the
shown party member composed in his own equipment, as a 160x240 canvas blitted at
the panel's left edge and two rows below its top. Between the two picker
rectangles it names him, and under the name it prints his position in the roster
as `i/n`. The panel is painted whether or not a figure composed: a member whose
pictures will not read leaves the figure absent and everything else drawn.

**FR-21 — the party picker.** Two 32x32 rectangles at (481, 443, 513, 475) and
(599, 443, 631, 475) step the shown member back and forward, wrapping at both
ends. One press does three things in one call: it moves the panel to that member,
it binds the backpack strip to THAT member's container and returns the strip to
its first place, and it makes the cell backgrounds ask that member's class. A
party of one answers that nobody else is with the player and moves nothing.

**FR-21a — whose container the strip is.** The backpack strip is the shown
member's own container, not a party-wide pool. A purchase lands in the shown
member's container and a sale takes from it.

**FR-22 — the wheel turns the region under the pointer.** Over the shelf grid's
region the wheel pages the rack by whole rows; over the backpack strip's region
it moves the strip by one place; anywhere else on the screen, the table included,
it moves nothing. The table has five places and no scroll base.

**FR-23 — the merchant is a control.** The merchant stands at
(277, 112, 353, 288), a column the four shelf hit rectangles leave free. Pressing
him re-opens his conversation where he has work to offer, and states that he has
none where he has not. Pressing him is the only way back into that conversation
once it has been paged to its end. While he has work his own box is outlined in
the colour the chosen shelf is marked with, because nothing decoded says how the
original invites the press.

**FR-24 — the answer to the last press.** The one line stating what the last
press did is drawn along the bottom of the tip widget's own decoded rectangle,
centred at (169, 283)-(476, 298), over the room's floor, with a one-pixel shadow
and no box behind it. A line too long for that width is cut rather than wrapped.

## Acceptance criteria

**AC-1** Over a fixture with a known price cell and known factors, a candidate's
admitted value and the item's price differ by exactly the rounding addend, and a
row whose price cell is negative is admitted at no ceiling.

**AC-2** A row admits a material only where its own tier mask has that bit set,
and a ceiling of zero admits only candidates whose value is zero.

**AC-3** Generating twice from one seed gives identical shelves; from two seeds
it gives different ones; and generating a second time over a filled shop leaves
the shelf counts at 100, 100 and 20, not doubled.

**AC-4** Taking from a shelf removes the element from that shelf and puts it on
the table stamped as the merchant's. Taking from the pack removes it from the
pack and stamps it as the player's. A sixth take leaves the table at five and
both sources unchanged.

**AC-5** Clearing a table holding one place of each side restores the shelf count
and the pack count to what they were before the two takes, and the purse is
unchanged.

**AC-6** Buy with a purse below the total moves nothing: purse, table, shelves
and pack are all unchanged, and the refusal is stated. Buy with a sufficient
purse debits exactly the sum of `quantity × unit price`, empties the merchant's
places off the table, leaves the player's places on it, and puts `quantity` units
of each bought code in the shown member's container.

**AC-7** Selling a place of quantity 3 at unit price 5 credits 8, and the figure
FR-9 shows for that place is 9.

**AC-8** A sold item is on a shelf afterwards and is not in the pack.

**AC-9** Arriving in the town fills the shelves with the live chapter's ceiling.
A saved and reloaded town game has shelves again, and the save bytes carry no
shelf.

**AC-10** With a nil item table the shop room opens, states that the shelves are
empty, offers no live trade control, and can be left.

**AC-11** Every rectangle FR-12 and FR-15 to FR-17 name lies inside the 640x480
frame. Two pairs of them overlap and a fixed test order resolves both: the
button panel reaches into the merchant's room, and the up arrow's lower edge
shares one pixel row with the first row of shelf cells. Every other pair is
disjoint. A point inside a cell resolves to that cell's own grid and index, and
a point in the gap between two cells resolves to no control.

**AC-12** A shelf holding seven items shows six cells; one press of the down
arrow shows items three to eight of the shelf, and a second press does not scroll
past the last row. The up arrow at the first row does nothing.

**AC-13** Clicking each of the four room rectangles selects a different shelf, and
a click after the grid has been scrolled leaves the grid at its first row. Before
any click the grid holds no cell. A point in the room's lower right resolves to
shelf 0, its lower left to shelf 1, its upper right to shelf 2 and its upper left
to shelf 3.

**AC-14** A cell of an item priced 5 on the merchant's side selects plaque
`costs1.bmp` and prints 5; the same item on the player's side selects
`costm1.bmp` and prints 3. A price of 1000 selects plaque 4 and a price of
9999999 selects plaque 7.

**AC-15** Over a purse of 4 and a shelf holding an item priced 5, that cell's
background is `backinv.bmp`; with a purse of 5 it is `backinvs.bmp`; an item the
shown member's class may not use is `backinvg.bmp`.

**AC-16** Clicking an occupied shelf cell moves the item onto the table, clicking
the table place it landed on returns it to that shelf, and both leave the purse
unchanged. Clicking an empty cell of any grid changes nothing.

**AC-17** With nothing on the table, the buy and sell buttons are not live and
clicking them moves nothing; with a purse below the buy total the buy button is
not live. The fourth button clears the table and leaves the room in one press.

**AC-18** Pointing at an occupied cell answers that item's name and its decoded
characteristic lines; pointing at an empty cell, or at the money element, answers
nothing.

**AC-19** With no archive at all the screen still resolves every control, states
the same numbers, and draws no picture. Nothing panics.

**AC-20** A place with no element in the shelf grid or the table leaves the pixel
under it unchanged from what its region painted; a place with no element in the
backpack strip paints `backinvg.bmp`.

**AC-21** The money element paints the coin over `backinv.bmp` and answers no
hover. Turning the strip one place leaves it out of the first cell.

**AC-22** With two members in the party, the panel opens on the first, one press
of the next arrow shows the second and puts the second's container in the strip,
a second press wraps to the first, and a press of the previous arrow from the
first shows the last.

**AC-23** A purchase made while the second member is shown lands in the second
member's container and not in the first's.

**AC-24** The wheel over the shelf region pages the rack by one row and leaves
the strip's base alone; over the strip region it moves the strip by one place and
leaves the rack's base alone; over the table or the character panel it moves
neither.

**AC-25** `keyBlack` clears the alpha of a pure-black pixel and of no other, a
near-black pixel included.

**AC-26** `opaqueBounds` of a 60x10 canvas whose ink is (33, 2)-(58, 8) answers
that rectangle, and a wholly transparent canvas answers the empty rectangle.

**AC-27** No production string literal under `pkg/` holds a byte above 0x7f. The
check reads parsed syntax, so the same files' prose may hold any character.

**AC-28** The merchant's box is outlined when there is work to offer and is
untouched when there is not, and the outline reaches no pixel outside that box.

**AC-29** The wheel crosses the screen seam naming the region under the pointer,
and does not cross at all over the table or the character panel.

## Properties

**P-1** No coin is created or destroyed: after any sequence of takes, clears,
buys and sells, the change in the purse equals the sum of the buy debits minus
the sum of the sell credits.

**P-2** No item is created or destroyed: the multiset of codes across the
shelves, the table and the pack is unchanged by a take, a clear or a refused
commit, and a completed commit only moves codes between the three.

**P-3** The generator writes no field of any entity, world or save. The shop is
front-end state, and the canonical byte form's version does not move.

**P-4** Every string this build draws is ASCII. The install's own strings arrive
as the code page's bytes and are drawn byte for byte; this build's own strings
add no multi-byte rune to that stream.

## Disclosed divergences

Every one of these is a typed row in `docs/DIVERGENCES.md`, which is the single
register since pipeline v2. They are listed here because this spec is
self-contained.

**D-1 — the shop shows item characteristics and the original does not.** The
original's shop screen prints two things on a cell, the quantity and the price,
and no item name and no characteristics anywhere. This build adds FR-19's hover
on the owner's own ruling. It is an addition, not a decode.

**D-2 — the shelves do not flicker.** Each of the four shelves in the room
picture carries an eleven-frame torch loop in the original, addressed as folder
`4 - i`. This build draws the room picture and plays no loop.

**D-3 — the two lines of shop advice are not shown.** The original prints two
short texts on this screen, in a widget at (164, 162)-(476, 298). They are the
only two shop-screen resources that differ between the English and Russian
installs and this build shows neither; it uses the bottom of that widget's
rectangle for FR-24's own line instead.

**D-4 — the table picture is eight pixels short of its region.** The shipped
picture is 472 pixels wide and the region is 480. This build leaves those eight
pixels unpainted at the right-hand end of the table.

**D-5 — the top-left shelf in the room is empty.** The original's four shelves
are potions, magic, weapons and armour; this build stocks three of them, because
the item class the potion shelf holds has no code composition in this tree. Which
of the four room rectangles holds which kind is not decoded either: this build
reads the room picture and pairs them by what is drawn there.

**D-6 — the character panel is this build's own copy of the widget, not the
mission screen's own object.** The original re-parents ONE panel object between
the two screens, and that object carries a composition cache. This build cannot
re-parent a widget into two screens; what it shares instead is the composition,
filled once per room entry and read by both this panel and the town dialogue's
speaker figures. The picker's own art is not decoded either, so the two chevrons
and the name plate are authored in the inventory screen's palette.

**D-7 — the shop screen is driven with the mouse.** Escape leaves the room, as it
does in every other room. There is no keyboard equivalent of a cell click, and
none of the picker.

**D-8 — the wheel turns the region under the pointer.** ROM1 predates wheel UI
and no claim exists or is expected. This is the owner's own affordance.

**D-9 — the backpack strip's margins are black.** The strip's region has no
picture of its own and the shop view fills it flat with colour zero, which is
what the original does. The five cells stand on that fill.

## Out of scope

The consumables shelf and the six literal potions: the item class they hold has
no code composition in this tree and none is published. Enchantments. The
multiplayer restock timer. A partial-stack move. A map-placed shop. Mercenary
hire and the school's training.
