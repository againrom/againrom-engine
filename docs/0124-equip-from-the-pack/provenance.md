# 0124 — equip from the pack: provenance

Claims are cited by **claim id**, never by an experiment folder, and each was read through
`research/tools/claim` against the submodule at its pin — not out of a ledger by hand.

## What research establishes

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `ITEM-CMD-007` | High for the two dispatch tables and the code map | Moving an item is **one command**, `0x22`, carrying a **source code** and a **destination code**; source `2` is the actor's own container, destination `1` is equip. That is why the equip here is a `sim.Command` drained by `Step` and not a call into the world from an input handler. Amended by `EXP-0090`; the amendment is about the **ground** arm (source 3), which this story does not use. |
| `ITEM-EQUIP-006` | High for the slot map and the humanoid gate; **Medium** for "no per-slot type restriction" | Equipped is a **different place from carried** — fourteen pointer fields, not a marker in the list. The slot numbering: `1 -> actor+0x74`, `2 -> actor+0x78`, `3..12 -> actor+0x198+4i`. And: the equip returns the **displaced** item, which the command **puts back into the container at the source index** (`L04616`) — this story's displacement rule, quoted rather than invented. |
| `HERO-EQUIP-017` | High (each routine read whole; each field pair a named `ADD`/`SUB`) | `Shield::Equip`'s two-handed test "first drops the weapon in `actor+0x74`" — which fixes **slot 1 as the weapon slot** from a second direction, independently of `ResolveWeapon` already writing class 1. Also that a weapon **assigns** cadence, reach and active skill rather than adding them, which is why `data.Loadout` keeps the weapon separate from `EquipMod` and this story hands it there. |
| `HERO-FOLD-033`, `HERO-FOLD-035` | High (both routines read end to end) | The equipment fold is **eight `ADD`s and one assignment, no multiply**. Nothing in this story multiplies an item's contribution into a stat; `pkg/data/recompute.go` already implements the graph and this story only supplies it a `Loadout`. |
| `HERO-ORDER-014`, `HERO-CAP-015` | as published | The ordering points and the destructive cap. Consumed **whole and unread here**: `Hero.Recompute` implements the graph in that order already, and this story calls it rather than restating it. |
| `ITEM-CONT-004` | High for the class shape; Medium for "nothing refuses" | The container is unbounded and holds an ordered list — so removing at an index and writing back at that index are both things it can do, and no capacity test stands anywhere on the path. |
| `ITEM-CODE-029` | as published | Class is bits 8..11 and a class of 0 resolves to a null the map-load caller skips — so `B = 0` is refused as an equip target rather than treated as slot 0. |

## What is ours by choice

- **The double-click**: nothing decoded names one, and `pkg/ui` had no notion of one. The threshold
  is authored, counted in map-screen frames, and lives in exactly one named constant.
- **The re-resolution of a weapon from its code.** Field **C** of an item code is this build's own
  choice (`0110` DD-1 — research publishes `ITEM-APPEAR-023` **Unknown**), so reading C back as a
  shape index is exact for a code this build composed and a **guess** for a code a map authored.
  It moves the scale factors, never whether an item equips.
- **The byte form's equipment section**: its position, its fixed width and its refusal rules are
  this project's, exactly as every other section of that form is.
- **Where the recompute stands in the frame** — the statement immediately after `sim.Step`, in
  ascending entity id. `SetCombat`'s own doc requires *a* deterministic point and names none.

## What is open

- **Which shipped `Data.bin` column feeds the armour and shield `EquipMod`** is not decoded
  (`pkg/data/recompute.go`'s own note). Nothing here needs it: only a weapon is folded.
- **`ITEM-EQUIP-006`'s Medium**: whether a per-slot type restriction lives inside the per-class
  `vt+0x38` bodies. This story applies none, which is what the claim says it read.
- **`actor+0x19c` / `actor+0x1a0`** — array indices 1 and 2, Unknown. This story writes no slot but
  the one an item's own code names, so it cannot reach them by accident.

## What was removed

Nothing was retracted under this story. `ITEM-CMD-007` carries an **amendment** and a resolved
contradiction (`C-6`) rather than a retraction; the reader was run with the retraction state
cross-read, and the amended clause is the ground arm this story does not use.
