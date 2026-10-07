# Analysis — where the container's facts come from

Every claim id below is read from the submodule at this story's pin. Nothing
here is a fact this project holds for any other reason.

## The class

`ITEM-CONT-004` is the whole of what a container is: a `CObList` plus two
dwords, `0x24` bytes, fifteen construction sites. The two dwords are `+0x1c`,
the index the next `Add` inserts at, and `+0x20`, the load. A UNIT AND A SACK
HOLD THE SAME CLASS — `actor+0x7c`, `sack+0x40` — which is why `sim.Sack`'s
`Items` and an actor's carried codes are one shape in this tree too, and why the
transfer is a move between two of the same thing.

Two of its clauses are graded differently and the difference decides the spec:

- **High** for the shape and the arithmetic. The constructor is six
  instructions; the insert, the take and the move-all were read whole.
- **Medium** for *nothing refuses*. The instrument is `EnumRefs callto:` on the
  two inserts, of which the item-module and command-dispatcher owners were read
  and the rest classified by owner. Its blind spot is named: a caller that tests
  the load before calling.

So FR-3's unboundedness is a DISCLOSURE. `+0x1c` is explicitly NOT a capacity —
the row says so — which removes the one number a reader would otherwise reach
for as a slot count. There is no slot count anywhere, which is what makes the
window's eight cells a drawing limit (D-3) rather than the container's.

## What weight would have cost

`ITEM-LOAD-005` gives the load one consumer: `actor+0x90` takes the actor's own
weight plus HALF the container's `+0x20`, saturating to a flat `0x7d00` above
`0xfa00`, and `HERO-SPEED-008` subtracts `load/capacity` from speed only when
the load reaches capacity. `ITEM-STACK-003` gives the sum: `S (s16)+0x4a x
(u16)+0x42`, per-unit weight times count, four update sites read whole.

Both are High and both are arithmetic over a number this tree cannot obtain. A
weapon's weight is the Weapons row's slot 3 (`HERO-EQUIP-017`); no claim in the
pin establishes the column a non-weapon's weight is read from. `grep -rn Weight
pkg/ --include=*.go` returns nothing outside tests: the value has never entered
this tree. Inventing the column is the one thing the prime directive forbids, so
D-1 declines the field rather than storing a number that is always zero — a
permanently-zero field in a hashed form is a promise, not state.

`pkg/data/hero.go`'s `Speed()` documents the omitted overload term with the
reason "this tree has no inventory, so nothing carries anything and the gate is
never satisfied". That sentence is FALSE after this story. It is not repaired
here — the term needs the weight column, which is what D-1 declines — and
verification.md names it as an expired premise.

## Zero is not an item

`ITEM-CODE-029` cuts an authored `u16` into four fields and allocates by bits
8..11: 1 -> Weapon, 2 -> Armor, 3..13 -> the `0x68` siblings, 14 -> Item, and
ANYTHING ELSE -> a null the map-load caller skips at `L04694`. So a code of
zero has class zero, resolves to nothing, and the original's own loader drops
it. `data.Equipment`'s doc block reaches the same value from the other side:
zero is the sender's own encoding for an unoccupied slot.

FR-4 takes only the zero from this and reads no field of the code. `pkg/sim`
cannot import `pkg/formats/alm` (the determinism wall), and `sim.Sack`'s doc
block states that class and index are the format leaf's business. Testing a
code against zero keeps that true; a class nibble in `pkg/sim` would not.

## The map's own stock, and the one join

`ITEM-OWNED-028` is the arm: `R0461` fills a map keyed by each actor's
`+0x08`, looks a record's `+0x04` up in it, and the sack arm is guarded on that
lookup — so a stock record's coordinates and gold are never read, and the
elements go to `[actor+0x7c]`, the container `ITEM-CONT-004` names. High for the
arm and both destinations.

Its **Medium** is exactly the join this story needs: "nothing here reads what
writes `actor+0x8` or what the shipped values 1..N correspond to in the `.alm`'s
own record sets". So the id space is not established from the image, and the
spec may not assume it. What is available is the corpus, measured here over both
lawful roots through `pkg/vfs` and `pkg/formats/alm`, 28 campaign maps each:

    en: 176 type-8 records — 133 ground (152 elements),
        43 stock (25 elements, 5 with element +0x04 non-zero),
        largest 20 on scenario/120.alm
    ru: identical in every figure

which reproduces `ITEM-OWNED-028`'s own corpus line exactly. Against
`alm.Unit.UnitID` — "+0x40, the id a script's `Target_Unit` names" — all 43
stock records on each root match EXACTLY ONE unit, with zero unmatched and zero
ambiguous. That is a bijection over 43 cases and not a proof of the instruction
that writes `actor+0x08`, so FR-11 states the join and FR-12 makes a miss cost
only itself: a record matching no unit, or more than one, places nothing.

The 11 loose maps beside the campaign archive carry 4 type-8 records between
them, all ground, and no stock record at all.

## The transfer

`ITEM-PICK-009` is the routine: credit `sack+0x3c` to the owner's `Player`
money, stamp every element, pour the sack's ENTIRE container into `actor+0x7c`
with a mover that destroys the source, null `sack+0x40`, delete the sack. "There
is no capacity test, no distance test, no ownership test and no per-item
selection in that routine: the protocol has no take item *i* from the sack at
all." High for the routine.

FR-6 is that shape and FR-8 is that absence. What the row also carries — the
tick arm requiring a sack at the actor's own cell, the state dispatch, the
`+0x50`/`+0x54` relay closed by `ITEM-PICK-016` — belongs to the ORDER, which is
the later story. The one part of it FR-16 borrows is the cell test, because a
key that acts on the actor's own cell is the primitive plus a coordinate, not an
order.

`ITEM-DROP-008` puts money at `Player+0x38` and says it is never an item; that
is why FR-7 sends the gold to a per-slot purse and not into the container.

## What is deliberately not read

`ITEM-EQUIP-006` (High) puts equipped in fourteen pointer fields, a different
place from carried. It is cited here only to keep the twelve slots OUT: they are
`data.Equipment`'s already and a container holding them too would be one object
doing two decoded jobs. `ITEM-CARRY-015` and `ITEM-SPAWN-026` are read and
unused — the mission boundary and the authored spawn are other stories'.
