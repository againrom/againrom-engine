# 0124 — equip from the pack: analysis

## What the owner asked for

A double-click on a weapon lying in the open inventory window's pack area puts it in the
character's hand, with the numbers following. Two things were reported together; the other one —
the window not swallowing its own clicks — is a **hotfix**, `docs/hotfix/LEDGER.md`, because it
touches nothing the digest covers. This story is the half that does.

## What we did not know, and what we looked at

**Where an equip may legally happen.** `0113` built `sim.World.SetCombat` as the door for new
combat numbers and left it **called from nowhere**; its own doc block says why that is dangerous —
two peers calling it at different points in a frame diverge silently. `ITEM-CMD-007` settles it:
moving an item is **one command** with a source code and a destination code, drained by the
dispatcher, not a direct mutation. So the equip is a `sim.Command` and the recompute is pinned to
the one statement immediately after the `Step` that applied it. This story is `SetCombat`'s first
and only call site.

**Whether "which slot" and "is it equippable" have to be authored.** They do not.
`data.ItemCode`'s own field **B is the equipment slot the item occupies and also its class**
(`0110` FR-1), and `ITEM-EQUIP-006` gives the same numbering from the other side: the take-off arm
computes a slot and dispatches `1 -> actor+0x74`, `2 -> actor+0x78`, `3..12 -> the armour array`.
`ResolveWeapon` already writes `B = 1` for every weapon it composes, and `HERO-EQUIP-017` shows
`Shield::Equip` dropping "the weapon in `actor+0x74`" — so slot 1 is the weapon slot, read off the
item rather than chosen here. `B = 14` is `ItemClassCarried`, carried and never worn; `B = 0`
resolves to a null a map-load caller skips (`ITEM-CODE-029`). That is the whole equippability
test, and it is data.

**How to get from a code back to a weapon.** The pack holds `uint16` codes; the recompute wants a
`data.Weapon`, and `ResolveWeapon` takes a **name**. A code carries the three indices that name is
built from — A the material, C the shape, D the row — so the inverse is: read the three collection
entry names back out and re-resolve. Field C is **ours by choice** (`0110` DD-1; research publishes
`ITEM-APPEAR-023` Unknown), so this inverse is exact for a code this build composed and a guess at
the shape scale for a code a map authored. It changes the magnitude of the numbers, never whether
the equip happens.

**What the displaced weapon does.** Decoded, and cheap enough to keep: `ITEM-EQUIP-006` — the equip
call returns the **displaced** item and the command "puts it back into the container at the source
index" (`L04616`). So the pack cell the weapon came out of is where the old one lands.

**Whether a double-click exists in `pkg/ui`.** It does not — checked; the package has a tap, a box,
a right press and an arming key, and no notion of two clicks at all. So one is authored here.
It counts **frames**, not wall-clock: `command` is called exactly once per map-screen frame, which
is the counter already available, and `pkg/ui` reaching for a clock would put a second timing
source beside the one the pacer owns.

## What is not asked

Armour, shields and the other eleven slots; drag-and-drop; taking a worn item off. `pkg/sim`'s arm
is written over the slot the code names rather than over slot 1, so none of those needs it reopened
— what they need is a fold, and this story folds a weapon only.
