# 0157 — plan

## Where each requirement is built

| Requirement | File | Decision |
|---|---|---|
| FR-1, FR-2 | `pkg/data/shopstock.go` | DD-2, DD-3, DD-4 |
| FR-3 | `pkg/game/shop.go` `Generate` | DD-5 |
| FR-4 | `pkg/game/frontend.go` `arriveInTown` | DD-1, DD-6 |
| FR-5, FR-6 | `pkg/game/shop.go` table methods | DD-7 |
| FR-7, FR-8 | `pkg/game/shop.go` `Buy`, `Sell` | DD-7 |
| FR-9 | `pkg/game/shop.go` `SellTotal`, `SellPayout` | DD-8 |
| FR-10 | `pkg/game/shoproom.go`, `pkg/ui/town.go`, `pkg/ui/app.go` | DD-9, DD-11 |
| FR-11 | `maskTable`, `ShopPool`'s refusals | DD-3, R-1 |
| FR-12 | `pkg/ui/shopscreen.go` rect block, `pkg/game/shopart.go` | DD-11, DD-12 |
| FR-13, FR-14 | `pkg/ui/shopscreen.go` `drawShopCell`, `pkg/game/shopview.go` | DD-13 |
| FR-15, FR-16 | `pkg/ui/shopscreen.go` `ShopControlAt`, `pkg/game/shopview.go` `ShopClick` | DD-14 |
| FR-17, FR-18 | `pkg/game/shopview.go` `ShopClick` | DD-14 |
| FR-19 | `pkg/game/shopview.go` cell `Info`, `pkg/ui/shopscreen.go` hover draw | DD-15 |
| D-1 … D-8 | disclosed in `spec.md`; D-6, D-7 built in `pkg/game/shopview.go` | DD-16 |

## DD-1 — the shop is front-end state, not simulation state

The shop lives in `pkg/game` beside `Town`, not in `pkg/sim`.

Three reasons, in order of weight. `SHOP-SAVE-015` establishes the stock is never
serialised, so it is not part of the state a save carries. The purse it spends is
already `Town.gold` in `pkg/game`, and the pack it fills is already
`FrontEnd.Carried`, so the shop's two counterparties are both here. And `pkg/sim`
is a determinism wall with a hashed byte form: putting the shelves there would
cost a `formatVersion` bump for state the original does not even keep.

Consequence for the gate: no `pkg/sim` file changes, `formatVersion` stays at 46,
and `pkg/sim/binary_test.go`'s version test is untouched (spec P-3).

## DD-2 — the pool and the price go in `pkg/data`, the shop in `pkg/game`

`pkg/data` already owns what an item is worth: `ResolveWeapon`, `fillArmor`,
`scale` and `ftol` are all there, and the shop's value window is the same
arithmetic at a different double slot. So `pkg/data/shopstock.go` holds the
window, the price and the candidate walk; `pkg/game/shop.go` holds the shelves,
the table and the two commits.

The split is also what keeps the walk testable from a hand-built collection with
no front end anywhere near it.

## DD-3 — a fourth interface for the material mask

`data.Collection` exposes `Len`, `EntryName`, `EntryParams` and `EntryStrings`.
The mask lives in the entry's raw block, which no interface in `pkg/data`
reaches. Two changes:

- `databin.Collection` gains `EntryRaw(i int) []byte`, the sibling of
  `EntryDoubles` and the same copy discipline.
- `pkg/data` declares `MaskTable`, which is `Collection` plus `EntryRaw`.

`mapload.Table` keeps `data.Collection` for its three item collections and the
shop asserts to `MaskTable` at the point of use. Widening `Collection` itself
would break every hand-built fake in the tree for a method only this story reads.

A collection that does not satisfy the assertion contributes no candidates, which
is FR-11's own answer.

## DD-4 — one code composer

`uint16(material)<<12 | class<<8 | uint16(shape)<<5 | uint16(row)` is spelled in
three places already: `ResolveWeapon`, `ResolveShield` and `fillArmor`. The pool
walk needs a fourth. `data.ComposeItemCode` is added and all four sites go
through it, so the encoding has one writer.

The class differs per collection and that is the walk's only per-collection
branch: 1 for a weapon, 2 for a shield, and for an armour the row's own Slot
column where that is a valid equipment slot and 0 otherwise — `fillArmor`'s own
rule, reused rather than restated.

## DD-5 — the shelves are a fixed array and the draw is one loop

Three shelves in one `[3][]ShopItem`, filled by one loop over a table of
`{pool kinds, count}`. The magic shelf's pool is the union of the other two,
built by concatenation rather than by a third walk, which is what
`SHOP-MAGIC-007` says mode 6 is.

Generation clears before it fills (`SHOP-LIFE-013`), which is one assignment
rather than an append, so AC-3's "not doubled" is structural.

## DD-6 — the seed

`shopSeed(chapter, finished, ceiling)` mixes the three with FNV-1a and returns an
`int64`. `math/rand.New` over that source is the generator; no global source is
touched and no clock is read.

The three inputs are all recomputed from state a save already carries, which is
what lets FR-4 restock after a load without the save holding a shelf. It is also
what makes the stock reproducible, which is provenance A-1's disclosed
divergence.

## DD-7 — the table's ownership stamp is a bool, not a player id

`SHOP-TRAY-025` reads `elem+0x14` as a zero/non-zero test at every consumer: the
buy commit walks `== 0`, the sell commit walks `!= 0`, and clearing the table
switches on it. Single player has one customer, so the field carries no
information beyond the test. A `Mine bool` is that test and nothing else.

Each place also records which shelf it came from, so FR-6 can put it back where
it was rather than re-deriving a shelf from the item's class. The sell commit
does re-derive, because `SHOP-SELL-010`'s return path is by class and a sold item
may never have been on a shelf.

## DD-8 — the two totals are two functions, and both are shipped

`SHOP-TRAY-027` establishes the screen and the payout disagree on an odd unit
price. `SellTotal` is the screen's `ceil(price/2) × qty` and `Sell` pays
`ceil(qty × price / 2)`. Writing one and deriving the other would erase a decoded
behaviour the owner has himself reported from play; the room states the payout
next to the total so the difference is visible rather than surprising.

## DD-11 — the shop room is a drawn screen, and the row list is gone

Round two replaces the row list in the shop room with the decoded screen. The
room's `Rows()` answers nothing while the shop is open; the shelf paging of DD-10
goes with it, because the shelf grid scrolls by the original's own arrows.

The seam stays one optional interface on the town screen, now three methods:
`AtTownShop`, `ShopScreen() ShopScreenView` and `ShopClick(ShopControl)
TownAction`. `ShopClick` returns a `TownAction` and is applied through the same
`applyTownAction` every other town mutation goes through, so DD-9's invariant —
one mutation door, the list rebuilt after it — survives the list's removal.

`pkg/ui` receives pixels and numbers and nothing else, exactly as
`ChargenPresentation` already does. It holds the rectangles, because they are
screen geometry; it holds no price rule, no affordability test and no shelf.

## DD-12 — the art is loaded once, on the first entry into the shop

Thirty-two archive entries paint this screen. They are read on the first
`ShopScreen()` call and cached on the front end, not at construction: a player
who never enters the shop pays nothing, and a player who enters it pays once.
The load is attempted exactly once, success or failure, which is `sheetCache`'s
own "tried" convention.

Every entry is optional. A nil picture draws nothing and the region under it
stays as it was. That is what keeps golden rule 2 true — the whole screen is
reachable in tests with no install at all — and it is also the honest behaviour
for an install missing a file.

## DD-13 — the cell is one function and the view carries a class, not a picture

`pkg/game` decides which of the three backgrounds and which of the fourteen
plaques a cell takes, because both decisions read the purse, the price and which
side owns the item. It reports them as small enumerations. `pkg/ui` maps an
enumeration to the picture it loaded. So the affordability rule lives with the
purse and the picture lives with the pictures.

The plaque index is the price's decimal digit count minus one, clamped to 6,
computed by repeated division rather than by a logarithm: `pkg/ui` may not use
floats where an integer answers, and the original's `floor(log10)` over a value
clamped to seven digits is exactly a digit count.

## DD-14 — one hit test, one control type, checked in a fixed order

`ShopControlAt(point)` answers one `ShopControl{Kind, Index}` or nothing. The
order is the button panel, the shelf arrows, the three grids, the four room
rectangles, then the authored corner. The order matters only where two
rectangles could overlap, and two pairs do. The button panel reaches into the
merchant's room in x 464..480, and the panel wins there because its own buttons
are drawn over it. The up arrow ends at y 32 and the first row of shelf cells
starts at y 31, so they share one pixel row; the arrow wins it. Both are the
original's own numbers and both are asserted, so a later change to the order is
seen rather than discovered.

`ShopClick` on the far side is a switch over the same type. A control the model
refuses returns an unchanged screen and a stated reason, never a panic: an index
out of range is a miss.

## DD-15 — the hover reuses the inventory's own item lines

FR-19 is an addition to the original and it introduces no new item vocabulary.
The cell's `Info` is `itemInfoLines`, the same function the inventory popup
prints, so an item states the same characteristics wherever it is seen and a
later decode of a class reaches both screens at once.

## DD-16 — three shelves in four rectangles

The model has three shelves and the room picture has four. The room's rectangles
are paired with what is drawn in them: the upper-left holds potions and this
build stocks none, the upper-right magic, the lower-left weapons, the lower-right
armour. Clicking the potion rectangle selects a shelf that is empty and says so,
rather than being dead.

## DD-9 — the room is rows, the strip is the table (round one, superseded by DD-11)

0142's town screen is a row list with one `Choose(i)`, and `pkg/ui` rebuilds the
list after `Choose` and `Back` and at no other time. Every mutation this story
adds therefore goes through `Choose`, which keeps that invariant intact and adds
no second mutation path.

The existing five-place 80×80 strip already drawn in the shop room becomes the
table (`SHOP-TRAY-024`'s own geometry). It gains a per-cell side mark and a
price. One method is added to the optional `TownShopScreen` seam so a click on a
cell sends that place home; it returns a `TownAction` and is applied through the
same `applyTownAction` that refreshes the list.

The row list is ordered: the two commits, clear, the table places, the pack, the
shelf selector, the selected shelf, then 0142's mission offer. `Choose` maps an
index by rebuilding the same list rather than caching one, which is the file's
existing rule.

## DD-10 — the shelf list is windowed (round one, superseded by DD-11)

A shelf holds 100 elements and the picker scrolls, but 100 rows behind two
commits is a list nobody can use. The room shows one shelf at a time and a
fixed-size page of it with a next-page row. The page size is a constant here, not
a decoded number.

## R-1 — risk: the assertion to `MaskTable`

If the shipped `databin.Collection` ever stops satisfying it, every shelf goes
empty and the room says so — a visible, stated degradation rather than a panic.
A test builds a collection that does not satisfy it and asserts exactly that.

## R-2 — risk: the pack is a flat code list

`mapload.Carry.Items` is `[]uint16` with one entry per unit, not a stack list.
A take of a quantity-3 element removes three entries and a buy appends
`quantity` entries. The stacking the screen shows is `townItemStacks`' own fold,
which already exists.
