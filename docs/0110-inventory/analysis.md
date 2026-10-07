# Analysis — the inventory area, and where this story stops

Intensity: **spec-first / static**. Terrain: **brownfield** at the equipment slot, the party path
and the viewer's input; **greenfield** for the item code, the art naming and the window.

## What we did not know

Three questions, and the third is the one that moved the boundary.

**Where the art is.** `GRAPHICS.RES` was counted on both roots. The two trees are the same file for
file — the only differences are five `infowindow/*.bmp`, two `interface/*.bmp` and `version.txt`,
all EN-only, and the archive's own root entry name. `inventory` **416** nodes, `equipment` **987**,
`backpack` 2, `interface` 584 (RU) / 586 (EN), `infowindow` 81 / 86.

**Whether the seven-digit names join.** They do, completely. Every one of the 367 `equipment` leaf
codes — 316 primary + 48 secondary under each of `mfighter` and `ffighter`, 76 + 24 under each of
`mmage` and `fmage` — is also an `inventory` node, with **zero** misses in that direction. The 49
`inventory` codes with no `equipment` sheet are **exactly** the class-14 ones, which is the class
that is never worn. `primary` totals 784 and `secondary` 144: 928, the figure-sheet count already
published. Every icon sampled is one frame of **80x80**; `interface/backinv.bmp` is 80x80 too, so
the shipped cell and the shipped icon are the same square.

**Whether the digits are the packed code or a coincidence.** They are the code. Over all 416 names
read as A/B/C/D: field B takes only `{1,2,4,5,6,7,8,9,10,12,14}` and never 0, 3, 11, 13 or 15;
outside class 14, C never exceeds 7 and D never exceeds 31; for class 14 the first field is always
0 and the low byte never exceeds 255. Nothing was fitted — the rule came from the disassembled
formatter and the corpus was asked to break it.

Then the definition table was counted: **Shapes 5, Materials 16, MagicItems 49, Weapons 28**. The
observed A domain is 15 of 16 material rows, C is exactly the five shape rows, class 14's index
range is exactly the 49 magic-item rows, and class 1's D range is exactly rows 2..22 of Weapons —
every named weapon except the blank row, `BareHands`, `rem` and the four rows ROM1 never ships.
Materials 8..13 and 15 are the wood, leather and bone rows, and they are the seven blocks the
appearance law already sends to the *light* armour directory. Class 1's wooden materials carry
D in {Club, Staff, Shaman Staff, Short Bow, Long Bow, Crossbow} and nothing else.

## What that changed

An equipped item's art name needs four fields. Research names A (the material index) and D (the
definition row); **C is published as three bits nothing names.** The party hero's weapon does not
arrive as a code at all — it arrives as a literal, and the resolution already computes a shape
index, a material index and a row from it. So either C is composed from the shape index, or this
tree cannot name the art of the one item it can put in a hand, and the window has nothing to draw.

The measurements above are why the story takes the first branch, as a **disclosed authored term**
guarded by a criterion the corpus can fail: the five weapon literals character generation can hand
out compose to `0001003`, `0101118`, `0101109`, `0101015` and `0801120`, and all five are shipped
nodes. It is not offered as a decode. `provenance.md` files it as ours, and it is worth a research
round precisely because a claim would move it.

## What already exists here

`0105` put twelve equipment slots on a character in `pkg/data`, each holding a bare **definition
row**, and derived the hero's world body from the first one. `0103` put the map's authored ground
loot into `pkg/sim` as `Sack`, carrying `[]uint16` item codes it deliberately gives no meaning;
`pkg/formats/alm` already cuts class and index out of such a code and leaves bits 12..15 and 5..7
unnamed. `pkg/formats/spr16.DecodeA` reads the `.16a` icons and `spr256` the `.256` sheets — both
were run against the install here and both work. `pkg/ui` already composes a picture into an
`*image.RGBA`, keys it for rebuild and presents it over the world; that is the panel, and the
window follows it. The four figure directories, the face sheets and the shipped window chrome are
all in the archive and nothing reads them.

## Where the boundary is drawn, and why there

**The first story is the model *and* the window**, because either alone is invisible: a code with
no picture is a refactor, and a window with nothing behind it is a frame. What makes that fit in
one story is that eleven of the twelve slots are empty and stay empty — so the twelve-slot
compositor, its paint order, the `secondary` sheets and the two-handed layer swap are all
unobservable, and every one of them is cut. With one occupied slot the figure is a base sheet and
one layer over it, which is exactly right for that case and nothing more.

Two further cuts keep it one story. The window is drawn with the frame primitive the panel already
has rather than the shipped `interface/backinv*.bmp` chrome — the BMP decoder is unexported inside
the menu package and exporting or moving it is a second concern. And the pack area is drawn
**empty**, because an actor has no container yet; that is honest rather than incomplete, since the
player has picked nothing up.

What the owner sees: he opens a window on the hero he is playing and finds him drawn, holding the
sword he was handed, with that sword's own icon in the first of twelve slots and an empty pack
below. What he does not see: anything in the pack, any armour, any way to move an item, and any of
the window's own ornament.

## The follow-ons, in the order they buy the most

**The container.** An actor's item list — unbounded, no capacity, a running weight — the stack with
its count and per-unit weight, and the load's one consequence: a speed penalty that never refuses.
This is what fills the pack area the window already draws.

**Moving an item.** The single move command with its source and destination codes, the ground as
one of them, picking up from a sack standing on the actor's own cell, dropping, and equipping and
unequipping through the twelve slots. This is the story that makes the window a thing you use.

**The map's own inventories.** The loot section's stock records name an existing actor rather than
the ground; `0103` kept only the ground ones. That gives every placed person the items the map
authored for him, with no command and no pick-up needed.

**The rest of the figure.** The paint order over twelve slots, the `secondary` sheets at four of
them, the two-handed layer swap, and the shipped window chrome.

**The other three definition tables.** Armours, shields and magic items — their rows, their numbers
and what wearing one does. Only weapons are resolved today.

**Items across a mission boundary**, and the shipped sack on the ground, which has art nothing
draws.
