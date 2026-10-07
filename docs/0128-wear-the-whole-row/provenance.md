# provenance — 0128 a person wears his whole row

Evidence ledger. The contract is `spec.md`'s and is self-contained; nothing here is a requirement.

## Research claims this story rests on

| Claim | Confidence | What it settles here |
|---|---|---|
| `ITEM-HUMEQ-030` | **High** for the dispatch and the three destinations; **Medium** for the corpus figures | The ten cells of a `Humans` row are equipment cells; the CELL POSITION picks the class — cell 0 a weapon, cell 1 a shield, cells 2..9 armour — and no arm reads the string before it allocates. All ten are *equipped*; the carried container receives only what equipping refuses or displaces. Names the three refusal arms and the engine's own strings for them, and the one cross-cell arm: a shield equipped over a two-handed weapon takes that weapon off into the container. |
| `ITEM-ARMSLOT-031` | **High** for the column and the index arithmetic; **Medium** for "parts 1, 2, 3 and 11 are never used" | Which of the twelve worn fields an armour piece takes is param 4 of its own `Armors` row — title `Slot` — and the array is indexed by it directly. A `Slot` above twelve is discarded by the constructor with `"Invalid armor part "`; a `Slot` of zero is refused by the equip arm with `"Illegal armor"`. |
| `ITEM-EQUIP-006` | High | The worn array's shape: slot 1 is `actor+0x74`, slot 2 is `actor+0x78`, the other ten the humanoid array at `actor+0x198`. Already carried by `pkg/sim` since 0105/0110/0124. |
| `ITEM-DEATH-012` | High | A corpse gives up slots 1 and 2 alone, and an `NPC`-templated template's whole container is deleted before anything is dropped. This is why FR-8's gate reaches those two slots and no others. |
| `ITEM-CODE-029` | High | A code of zero is never an item. |
| `HERO-APPEAR-049` | High | The item code's four fields, and that field B is both the class and the slot — which is why a resolved piece's slot and its code agree by construction rather than by two statements. |

The name parse itself is `ITEM-HUMEQ-030`'s own §6 evidence: five helpers in the order the armour
constructor calls them, the descending walk with its `Left(found) + Mid(len(entry)+1)` rebuild, the
`item+0x46 > 0xf` gate that fixes which collection sits at which runtime offset, and the
material-implied shape word that the weapon constructor alone does not call.

## What is ours by choice, and disclosed

* **A name that resolves to no row is not an item and is dropped.** The original keeps a nameless
  object and puts it in the backpack; this build holds items as codes, and a code whose row field is
  zero names nothing it could later drop, re-equip or draw. It is the collapse `firstWeapon` already
  made for the weapon cell, restated for the other nine. Measured cost over the shipped corpus:
  spec.md AC-6 states the number this story measured.
* **An armour whose `Slot` column is above twelve is refused into the container.** The original
  leaves the value standing and would index past its own array; that is out-of-bounds behaviour, not
  a rule, and no shipped row reaches it.
* **The panel states the worn set but the running window does not yet fill it** (plan DD-3). The per-frame
  entity seam carries no collection with which to name a code, and giving it one is a per-frame
  allocation for every entity on screen. `cmd/paneldump`, this story's own instrument, fills it.

## Open — deliberately not closed here

* **What a worn piece does to any derived number.** `pkg/data`'s `EquipMod` says in its own doc that
  the armour and shield arm is a seam and its zero is structural. `EXP-0132` is decoding it. Nothing
  in this story fits a column to it, and the block ships zero with its disclosure intact.
* A placed person's health maximum (`EXP-0133`) and what a corpse drops (`EXP-0134`).
