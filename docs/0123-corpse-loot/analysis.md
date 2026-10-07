# 0123-corpse-loot — analysis

## What was missing

Ground sacks landed in 0111 and the per-entity container in 0112, together with the
transfer primitive `TakeSack` and a key that reaches it. So an inventory can be
**filled** — from what a map authored on the ground — and never **emptied into the
world**: nothing in this tree turns a dead unit into loot. A killed clubman leaves a
body and nothing else, which is the one link the owner named.

## What we did not know, and what we looked at

**Where a death happens here.** The brief's census sized this as "one arm on
`decayPass`". That is wrong, and it matters: `decayPass` (step.go) advances bodies that
are *already* dead and removes the ones that finish decaying, so a drop there would fire
on a rung of the decay ladder rather than on the death. The once-only death transition is
`clearFelled` (step.go:813) — the function that clears the order, the crossing, the group
term, the attack and the patrol, and then, under a `Decay == DecayNone` guard that exists
precisely so it fires once, sets the stage, the dwell and the halved defence. That guard
is the shape `ITEM-DEATH-012` describes: one routine, run once, on the tick a unit stops
being one. The drop belongs there.

**Whether the container survives a removal.** It does not. `remove` (step.go:1160)
rebuilds `entities` and `routes` together and leaves `carried` at its old length, so after
any removal the two are no longer parallel — and every reader of a container indexes
`carried` with an index found in `entities`. Measured on a scratch test against master
`67d2e57`, on a world of two entities where the second carries one code:

    before: entities=2 carried=2
    after:  entities=1 carried=2
    Carried(2) = [] true          // entity 1's empty container, answered for entity 2
    Stock()    -> panic: index out of range [1] with length 1   (carry.go:139)

`Stock()` walks `carried` and indexes `entities`, so it **panics** rather than answering
wrongly. Its two callers are `pkg/mapload/start.go:366` and `:419`, the mission-start
rebuilds — so this is reachable from the game and not only from a test. It is a latent
0112 defect that this story sits directly on top of: a corpse is removed by `decayPass`
once its walk crosses the last rung, and everything a container means after that is wrong.
Fixing it is a precondition for the drop being observable at all, not a tidy-up.

**Whether the drop needs new saved state.** It does not, and that is what keeps this story
small. The container is already serialised per entity and the sack list is already
serialised per cell; moving codes from one to the other adds no field, so the byte form
stays at version 26 and no version had to be allocated.

## What was weighed and declined

`ITEM-DEATH-012` has five clauses. Three of them need per-entity state this tree does not
carry, and each would cost a byte-form version this lane was not allocated:

- the **two equipment slots** (`+0x78` unconditionally, `+0x74` gated on the weapon's
  `Data.bin` parameter 15) — there is no equipment model in `pkg/sim` at all, and the
  Units row's trailing `EquipItem` strings are `wt-0120`'s work;
- the **`"NPC"` suppression** — a template *name* test, and no entity here carries a name
  or any mark standing in for one;
- the **gold roll** — three treasure columns (`0x26`/`0x27`/`0x28`) per template, gated on
  `typeID > 0x40`; `pkg/data`'s `UnitDef` stops at slot 37 and `pkg/sim`'s `Entity` carries
  no treasure at all.

What is left is the clause the other four hang off — the container **becomes** the sack —
and it is the one the owner asked for. Built alone it drops whatever a unit carries, so it
needs no change when `wt-0120` starts putting a club and boots there.

## The precondition, stated

Nothing in this tree gives an ordinary unit a starting weapon yet, so on master a clubman's
container is empty and this story drops nothing from it. The two halves meet only after
`wt-0120` lands the `EquipItem` fold. This story's tests therefore give a synthetic entity
a container directly and prove the drop against that.
