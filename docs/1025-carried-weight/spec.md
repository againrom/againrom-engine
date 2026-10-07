# 1025 — carried weight: specification

Canonical to as-built at the landing of 2026-08-22. It states the behaviour that shipped, not the
behaviour that was intended. `contract.md` holds the claim provenance and the scope decision;
`closure.md` holds the evidence.

Before this story no value anywhere in this build carried an item's weight, an actor's carried load,
or a carrying capacity. Six requirements are built. They are one chain: FR-1 reads the number out of
the shipped tables, FR-2 makes it a property of an item code and of a world, FR-3 and FR-4 make the
actor's two fields, FR-5 is the only rule that reads them, and FR-6 is the row the player sees.

## FR-1 — the weight of one item definition

`Data.bin`'s `Armors`, `Shields` and `Weapons` collections share one runtime title array, and
**column 3 of it is the weight** (`ITEM-WEAPCOL-021`). The `Shapes` and `Materials` ladders carry a
scale factor for it at **slot 3** (`ITEM-LADDER-019`).

One item definition's per-unit weight is

```
weight = ftol(column3 * shapeFactor * materialFactor + 0.5)
```

`ftol` truncates toward zero, so the `+ 0.5` rounds a positive product to nearest and a negative one
toward zero. This is `itemWeight` in `pkg/data/itemweight.go` and it is the only place the
arithmetic exists.

A row too short to carry column 3 weighs its scaled zero, which is `armorColumn`'s existing
convention for a short row and not a new rule.

The result is signed and is not clamped. The shipped `Weapons` collection carries rows whose weight
column is negative, so a negative per-unit weight is a state the shipped tables produce
(`DIV-225`).

`data.Weapon`, `data.Armor` and `data.Shield` each carry the resolved value in a new `Weight` field,
filled by the same expression at the same point their defence and absorption are filled.

## FR-2 — the weight of an item code, and a world's weight table

`data.ItemCodeWeight(code, shapes, materials, armors, shields, weapons) (int32, bool, error)`
answers for any item code. It dispatches on the code's own class field `B`:

| `B` | Collection | Answer |
|---|---|---|
| the weapon class | `Weapons` | the row's scaled weight |
| the shield class | `Shields` | the row's scaled weight |
| any valid equipment slot | `Armors` | the row's scaled weight |
| anything else | none | `(0, false, nil)` |

The last line is the carried document, the key and the money bag: a code whose class names no
equipment slot resolves to no item definition and weighs nothing. That is not an error.

A collection or ladder the caller does not have answers `(0, false, nil)` as well. A partial install
therefore produces a world whose loads are understated rather than a world that will not build.

A code that **does** name an equipment class but whose row is not in the collection is an error: the
caller is reading a table it cannot read.

`pkg/sim` may not call any of this. `pkg/data` is below it in the tier order and the scaling above
is float arithmetic the determinism wall forbids. So a world carries a **weight table**: a sorted,
deduplicated list of `sim.ItemWeight{Code, Weight}` declared into it by whoever built it.

- `World.DeclareItemWeights(ws)` merges a list in and recomputes every actor's load. It is additive
  and idempotent. Declaring a code twice at the same weight changes nothing. Declaring one at two
  different weights is refused with the table left exactly as it was: the weight is a function of
  the code, so two answers is a fault in what was handed over, not a shape to fold.
- `World.ItemWeights()` returns a copy.
- A code the table does not name weighs **zero** (`DIV-223`). A world that was never handed a table
  carries none, every code weighs nothing, and no actor is ever overloaded, which is the behaviour
  every world built before this story had.

**Every producer of an item code declares its weights.** Four doors exist at construction, all in
`declareItemWeights` (`pkg/mapload/itemweight.go`): the entities' equipment slots, their containers,
the ground's sacks, and the compiled mission script's own item literals. Outside a mission,
`mapload.DeclareCodeWeights(world, table, codes)` is the door for a caller that replaces an actor's
holdings from an original save, and `sim.ReplaceStock`'s two callers in `pkg/game` use it.

## FR-3 — carrying capacity

`capacity = body * 10 + 1` (`HERO-SIGHT-007`, `L03918 IMUL EAX,EAX,0xa`). It is `data.Derived`'s
new `Capacity` field, filled by the same recompute that already answered the actor's speed, sight
and defence, and copied onto `sim.Entity.Capacity`.

It is filled where that recompute runs, which is the person arm and the party arm of the map load.
A units-band creature and an entity whose definition row did not resolve keep a capacity of zero,
and FR-5 never penalises a capacity at or below zero (`DIV-224`).

## FR-4 — the carried load

```
worn      = sum of the per-unit weights of the codes in the equipment slots
container = sum of weight * count over the container's elements
load      = worn
            if container >= 0xfa00 : load  = 0x7d00
            else                   : load += container / 2      (truncated toward zero)
```

`ITEM-LOAD-005` gives the derive at `L03920`..`L03921`. What is worn counts in full and what is
carried counts half. The saturation arm **assigns**, so a saturated actor's worn weight does not
survive it. The halving is the compiler's signed divide-by-two and truncates toward zero, which is
what Go's `/` does for a signed operand.

The law itself is `sim.CarriedLoad(worn, container)`, the one statement of it in this tree. It is
exported because a party member between missions has a worn set and a carried list and no entity
anywhere, and his card must state the same load his card inside a mission states
(`mapload.PartyLoad`).

An equipment slot holds a single item and its code counts once. A container element counts
`weight * count`, so a stack of five arrows weighs five arrows. The container sum is walked rather
than accumulated: this tree stores the container, and a running total beside a list that already
determines it is the one thing that can drift out of step with it. The arithmetic is 32-bit and
wraps, as the original's dword does.

`sim.Entity.Load` is the field. `recomputeLoad` is its **only** writer. Every producer that moves an
item calls it: `ReplaceStock`, `MoveCarried` (both ends), `TakeSack`, `dropAll`,
`dropFromContainer`, `dropFromEquipment`, `equip`, `unequip`, the script instants in `runInstant`
that give and take items, and the death pass in `step.go` that moves a body's holdings to a sack.
`DeclareItemWeights` recomputes every actor's load, because a weight the world did not have before
changes loads that were already computed.

## FR-5 — the overload speed penalty

```
if load >= capacity : speed = max(speed - load/capacity, 6)
```

`HERO-SPEED-008`. Below capacity the arm is skipped entirely: there is no small penalty and no
scaled one. The floor of six sits inside the arm, so it can raise the speed of an overloaded actor
whose unencumbered speed was below six; that is the original's arithmetic and narrowing it would be
this tree choosing a rule the claim does not state.

Two guards are this package's own and change no value the claim covers. A base speed at or below
zero is returned untouched, because speed is this package's "moves at all" predicate. A capacity at
or below zero is returned untouched, which is FR-3's unfilled population and also removes the
division.

**This is the only consumer of the load.** Nothing refuses a pick-up, a purchase, an equip, a trade
or a sack transfer on weight. The penalty is applied to the composed mover speed, group term
included (`DIV-226`).

## FR-6 — the `WEIGHT` row on the character card

`PanelFieldWeight` is a new panel field. `PanelSubject` carries `Weight int` and
`WeightKnown bool`, and the row is drawn when the producer filled it in, rather than gated on the
combat or character blocks: a load is real for a creature carrying loot and for a party member who
is in no mission at all.

The value is drawn with **one fractional digit**, as the value's own quotient and remainder by ten
joined with a point (`panelWeightText`). A negative load is signed once and its digits are taken
from the magnitude. A load of 181 draws as `18.1`; an empty doll and an empty pack draw as `0.0`.

**The divisor is authored** (`DIV-222`). The load and its derivation are decoded; the step from that
integer to the number the original's sheet prints is not. `HERO-104` reads the sheet's row formatter
whole and rules out both of its fractional sites for weight.

The row appears in both layouts, between the last resistance row and the `XP` / `SIGHT` / `SPEED`
block:

- `CompactPanelLayout` — the 160x242 card of the character generator and the town shell.
- `AuthoredPanelLayout` — the mission viewer's own panel.

Both reach the screen through `RenderCharacterPanel`, which has three call sites: the generator's
preview, the town shell's card, and the mission viewer's panel.

Two producers fill the subject. `panelSubject` (`pkg/ui/panel.go`) reads `MapEntity.Load`, which
`entityDraws` (`pkg/game/world.go`) fills from `sim.Entity.Load` on the way out of the mission.
`partyPanelSubject` (`pkg/game/world.go`) reads `mapload.PartyLoad` (`pkg/mapload/itemweight.go`),
which is `sim.CarriedLoad` over a party member's own worn and carried codes. There is no second
copy of the load law.

## The byte form

The version moves from 55 to **56** (`pkg/sim/binary.go`, whose own constant comment is the
authority for the number). Two changes:

- The entity record grows from 267 to **275** bytes: `Load` at +267 and `Capacity` at +271, both
  `int32`.
- A new **item-weight section** sits between the spell table and the casting section: a `uint16`
  count, then that many six-byte records of code and per-unit weight.

Both are hashed. The peel chain gains `TestThePinIsTheVersion55PinPlusCarriedWeight`, which is named
for the pin it extends, as its three siblings are, so the name does not go stale when the version
next moves.

## What is not built

- No refusal on weight anywhere.
- No encumbrance category, stamina or fatigue. There is one number and one penalty.
- Money is not in the load. `ITEM-LOAD-005` states the container's sum is item weight.
- The sheet's other fractional rows are untouched; `SIGHT` keeps its own convention.
- The card's row order, fonts, rectangles and chrome are `1022`'s and `1027`'s. This story fills a
  space that already existed.
